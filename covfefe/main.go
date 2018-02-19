package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func main() {
	dbFile := flag.String("db", "twitter.db", "The path of the SQLite DB")
	credsFile := flag.String("creds", "creds.json", "The path of the credentials JSON")
	flag.Parse()

	credsJSON, err := ioutil.ReadFile(*credsFile)
	if err != nil {
		log.WithError(err).Fatal("Failed to read credentials file")
	}
	var creds map[string]string
	if err := json.Unmarshal(credsJSON, &creds); err != nil {
		log.WithError(err).Fatal("Failed to parse credentials file")
	}

	db, err := sql.Open("sqlite3", "file:"+*dbFile+"?_foreign_keys=1")
	if err != nil {
		log.WithError(err).Fatal("Failed to open database")
	}
	defer db.Close()

	c := &Covfefe{
		db: db,
		httpClient: &http.Client{
			Timeout: 1 * time.Minute,
		},
	}
	c.initDB()

	config := oauth1.NewConfig(creds["API_KEY"], creds["API_SECRET"])
	token := oauth1.NewToken(creds["TOKEN"], creds["TOKEN_SECRET"]) // TODO: multiple tokens
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)

	params := &twitter.StreamUserParams{
		With:          "followings",
		StallWarnings: twitter.Bool(true),
	}
	stream, err := client.Streams.User(params)
	if err != nil {
		log.WithError(err).Fatal("Failed to open twitter stream")
	}

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		c.demux().HandleChan(stream.Messages)
		signal.Stop(ch)
		close(ch)
	}()

	log.WithField("signal", <-ch).Info("Gracefully stopping")

	stream.Stop()
	c.wg.Wait()
}

type Covfefe struct {
	db         *sql.DB
	wg         sync.WaitGroup
	httpClient *http.Client
}

func (c *Covfefe) initDB() {
	if _, err := c.db.Exec(`
	CREATE TABLE IF NOT EXISTS Messages (
		id INTEGER PRIMARY KEY,
		created TEXT NOT NULL,
		json TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS Tweets (
		id INTEGER PRIMARY KEY,
		created TEXT NOT NULL,
		user TEXT NOT NULL,
		message INTEGER NOT NULL REFERENCES Messages(id),
		deleted TEXT
	);
	CREATE TABLE IF NOT EXISTS Media (
		id INTEGER PRIMARY KEY,
		media BLOB NOT NULL,
		tweet INTEGER NOT NULL REFERENCES Tweets(id)
	);`); err != nil {
		log.WithError(err).Fatal("Failed to initialize database")
	}
}

func (c *Covfefe) insertMessage(object interface{}, created time.Time) (id int64, err error) {
	res, err := c.db.Exec(`INSERT INTO Messages (created, json) VALUES (?, ?)`,
		created, mustMarshal(object))
	if err != nil {
		return 0, errors.Wrap(err, "failed insert query")
	}
	id, err = res.LastInsertId()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get message id")
	}
	return id, nil
}

func (c *Covfefe) insertTweet(tweet *twitter.Tweet, message int64) (new bool, err error) {
	_, err = c.db.Exec(
		`INSERT INTO Tweets (id, created, user, message) VALUES (?, ?, ?, ?)`,
		tweet.ID, mustParseTime(tweet.CreatedAt), tweet.User.ScreenName, message)
	if err, ok := err.(sqlite3.Error); ok && err.ExtendedCode != sqlite3.ErrConstraintUnique {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "failed insert query")
	}
	return true, nil
}

func (c *Covfefe) insertMedia(data []byte, id, tweet int64) {
	_, err := c.db.Exec(`INSERT INTO Media (id, media, tweet) VALUES (?, ?, ?)`, id, data, tweet)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err, "media": id, "tweet": tweet,
		}).Error("Failed to insert media")
	}
}

func (c *Covfefe) deletedTweet(id int64) {
	_, err := c.db.Exec(`UPDATE Tweets SET deleted = datetime('now') WHERE id = ?`, id)
	if err != nil {
		log.WithError(err).WithField("tweet", id).Error("Failed to delete tweet")
	}
}

func (c *Covfefe) processTweet(messageID int64, tweet *twitter.Tweet) {
	new, err := c.insertTweet(tweet, messageID)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err, "message": messageID, "tweet": tweet.ID,
		}).Error("Failed to insert tweet")
		return
	}
	if !new {
		return
	}

	var media []twitter.MediaEntity
	if tweet.Entities != nil {
		media = tweet.Entities.Media
	}
	if tweet.ExtendedEntities != nil {
		media = tweet.ExtendedEntities.Media
	}
	if tweet.ExtendedTweet != nil {
		if tweet.ExtendedTweet.Entities != nil {
			media = tweet.ExtendedTweet.Entities.Media
		}
		if tweet.ExtendedTweet.ExtendedEntities != nil {
			media = tweet.ExtendedTweet.ExtendedEntities.Media
		}
	}
	if len(media) != 0 {
		c.wg.Add(1)
		go func() {
			for _, m := range media {
				if m.SourceStatusID != 0 {
					// We'll find this media attached to the retweet.
					continue
				}
				body, err := c.httpGet(m.MediaURLHttps)
				if err != nil {
					log.WithFields(log.Fields{
						"err": err, "url": m.MediaURLHttps,
						"media": m.ID, "tweet": tweet.ID,
					}).Error("Failed to download media")
					continue
				}
				c.insertMedia(body, m.ID, tweet.ID)
				// TODO: archive videos?
			}
			c.wg.Done()
		}()
	}

	if tweet.RetweetedStatus != nil {
		c.processTweet(messageID, tweet.RetweetedStatus)
	}
	if tweet.QuotedStatus != nil {
		c.processTweet(messageID, tweet.QuotedStatus)
	}
	// TODO: crawl thread, non-embedded linked tweets
}

func (c *Covfefe) demux() twitter.Demux {
	demux := twitter.NewSwitchDemux()

	demux.Tweet = func(tweet *twitter.Tweet) {
		if tweet.User.Protected {
			return
		}
		messageID, err := c.insertMessage(tweet, mustParseTime(tweet.CreatedAt))
		if err != nil {
			log.WithError(err).WithField("tweet", tweet.ID).Error("Failed to insert message")
			return
		}
		c.processTweet(messageID, tweet)
	}
	demux.StatusDeletion = func(deletion *twitter.StatusDeletion) {
		c.deletedTweet(deletion.ID)
	}
	demux.Event = func(event *twitter.Event) {
		if (event.Source != nil && event.Source.Protected) ||
			(event.Target != nil && event.Target.Protected) {
			return
		}
		_, err := c.insertMessage(event, mustParseTime(event.CreatedAt))
		if err != nil {
			log.WithError(err).WithField("event", event.Event).Error("Failed to insert message")
		}
	}

	demux.StreamLimit = func(limit *twitter.StreamLimit) {
		log.WithFields(log.Fields{
			"type": "StreamLimit", "track": limit.Track,
		}).Warn("Warning message")
	}
	demux.StreamDisconnect = func(disconnect *twitter.StreamDisconnect) {
		log.WithFields(log.Fields{
			"type": "StreamDisconnect", "reason": disconnect.Reason,
			"name": disconnect.StreamName, "code": disconnect.Code,
		}).Warn("Warning message")
	}
	demux.Warning = func(warning *twitter.StallWarning) {
		log.WithFields(log.Fields{
			"type": "StallWarning", "message": warning.Message,
			"code": warning.Code, "percent": warning.PercentFull,
		}).Warn("Warning message")
	}

	return demux
}

func mustMarshal(v interface{}) []byte {
	j, err := json.Marshal(v)
	if err != nil {
		log.WithError(err).WithField("object", v).Fatal("Failed to marshal JSON")
	}
	return j
}

func mustParseTime(CreatedAt string) time.Time {
	t, err := time.Parse(time.RubyDate, CreatedAt)
	if err != nil {
		log.WithError(err).WithField("string", CreatedAt).Fatal("Failed to parse created time")
	}
	return t
}

func (c *Covfefe) httpGet(url string) ([]byte, error) {
	res, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}
