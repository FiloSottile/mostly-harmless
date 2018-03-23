package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log/syslog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	lsyslog "github.com/sirupsen/logrus/hooks/syslog"
	// TODO: prometheus
)

func main() {
	dbFile := flag.String("db", "twitter.db", "The path of the SQLite DB")
	credsFile := flag.String("creds", "creds.json", "The path of the credentials JSON")
	syslogFlag := flag.Bool("syslog", false, "Also log to syslog")
	debugFlag := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	if *debugFlag {
		log.SetLevel(log.DebugLevel)
	}
	if *syslogFlag {
		hook, err := lsyslog.NewSyslogHook("", "", syslog.LOG_INFO, "")
		if err != nil {
			log.WithError(err).Fatal("Failed to dial syslog")
		}
		log.AddHook(hook)
	}
	log.Info("Starting...")

	credsJSON, err := ioutil.ReadFile(*credsFile)
	if err != nil {
		log.WithError(err).Fatal("Failed to read credentials file")
	}
	var creds struct {
		APIKey    string `json:"API_KEY"`
		APISecret string `json:"API_SECRET"`
		Accounts  []struct {
			Token       string `json:"TOKEN"`
			TokenSecret string `json:"TOKEN_SECRET"`
		}
	}
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

	messages := make(chan Message)

	c.wg.Add(1)
	go func() {
		c.HandleChan(messages)
		c.wg.Done()
	}()

	ctx, cancel := context.WithCancel(context.Background())

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		log.WithField("signal", <-ch).Info("Received signal, stopping...")
		cancel()
	}()

	var streamsWG sync.WaitGroup
	config := oauth1.NewConfig(creds.APIKey, creds.APISecret)
	for i, account := range creds.Accounts {
		token := oauth1.NewToken(account.Token, account.TokenSecret)

		httpClient := config.Client(oauth1.NoContext, token)
		client := twitter.NewClient(httpClient)

		user, _, err := client.Accounts.VerifyCredentials(nil)
		if err != nil {
			log.WithError(err).WithField("i", i).Error("Invalid credentials")
			return
		}

		params := &twitter.StreamUserParams{
			With:          "followings",
			StallWarnings: twitter.Bool(true),
		}
		stream, err := client.Streams.User(params)
		if err != nil {
			log.WithError(err).WithField("account", user.ScreenName).Error("Failed to open twitter stream")
		}

		streamsWG.Add(1)
		go func() {
			log.WithField("account", user.ScreenName).Info("Starting streaming")
			for m := range StreamWithContext(ctx, stream) {
				messages <- Message{account: user.ScreenName, msg: m}
			}
			if ctx.Err() == nil {
				log.WithField("account", user.ScreenName).Error("Stream terminated")
				cancel() // TODO: retry and reopen
			}
			streamsWG.Done()
		}()
	}
	streamsWG.Wait()

	close(messages)
	c.wg.Wait()
}

type Message struct {
	account string
	msg     interface{}
}

func StreamWithContext(ctx context.Context, stream *twitter.Stream) chan interface{} {
	c := make(chan interface{})
	go func() {
	Loop:
		for {
			select {
			case m, ok := <-stream.Messages:
				if !ok {
					break Loop
				}
				select {
				case c <- m:
				case <-ctx.Done():
					break Loop
				}
			case <-ctx.Done():
				break Loop
			}
		}
		stream.Stop()
		close(c)
	}()
	return c
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
		received DATETIME DEFAULT (DATETIME('now')),
		json TEXT NOT NULL,
		account TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS Tweets (
		id INTEGER PRIMARY KEY,
		created DATETIME NOT NULL,
		user TEXT NOT NULL,
		message INTEGER NOT NULL REFERENCES Messages(id),
		deleted INTEGER REFERENCES Messages(id)
	);
	CREATE TABLE IF NOT EXISTS Media (
		id INTEGER PRIMARY KEY,
		media BLOB NOT NULL,
		tweet INTEGER NOT NULL REFERENCES Tweets(id)
	);`); err != nil {
		log.WithError(err).Fatal("Failed to initialize database")
	}
}

func (c *Covfefe) insertMessage(account string, object interface{}) (id int64, err error) {
	res, err := c.db.Exec(`INSERT INTO Messages (json, account) VALUES (?, ?)`,
		mustMarshal(object), account)
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

func (c *Covfefe) deletedTweet(msgID, tweet int64) {
	_, err := c.db.Exec(`UPDATE Tweets SET deleted = ? WHERE id = ?`, msgID, tweet)
	if err != nil {
		log.WithError(err).WithField("tweet", tweet).Error("Failed to delete tweet")
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

func isProtected(message interface{}) bool {
	switch m := message.(type) {
	case *twitter.Tweet:
		if m.User.Protected {
			return true
		}
	case *twitter.Event:
		if (m.Source != nil && m.Source.Protected) ||
			(m.Target != nil && m.Target.Protected) ||
			(m.TargetObject != nil && m.TargetObject.User.Protected) {
			return true
		}
		switch m.Event {
		case "quoted_tweet":
		case "favorite", "unfavorite":
		case "favorited_retweet":
		case "retweeted_retweet":
		case "follow", "unfollow":
		case "user_update":
		case "list_created", "list_destroyed", "list_updated", "list_member_added",
			"list_member_removed", "list_user_subscribed", "list_user_unsubscribed":
			return true // lists can be private
		case "block", "unblock":
			return true
		case "mute", "unmute":
			return true
		default:
			log.WithFields(log.Fields{
				"event": m.Event, "json": mustMarshal(m),
			}).Warning("Unknown event type")
			return true // when in doubt...
		}
	}
	return false
}

func (c *Covfefe) HandleChan(messages <-chan Message) {
	for m := range messages {
		log.WithFields(log.Fields{
			"type":    fmt.Sprintf("%T", m.msg),
			"account": m.account,
		}).Debug("Received message")

		if isProtected(m.msg) {
			log.Debug("Dropped protected message")
			continue
		}

		switch msg := m.msg.(type) {
		case *twitter.Tweet:
			messageID, err := c.insertMessage(m.account, msg)
			if err != nil {
				log.WithError(err).WithField("tweet", msg.ID).Error("Failed to insert message")
				return
			}
			c.processTweet(messageID, msg)
		case *twitter.StatusDeletion:
			messageID, err := c.insertMessage(m.account, msg)
			if err != nil {
				log.WithError(err).WithField("deletion", msg.ID).Error("Failed to insert message")
				return
			}
			log.WithField("id", msg.ID).Debug("Deleted Tweet")
			c.deletedTweet(messageID, msg.ID)
		case *twitter.Event:
			messageID, err := c.insertMessage(m.account, msg)
			if err != nil {
				log.WithError(err).WithField("event", msg.Event).Error("Failed to insert message")
				return
			}
			if msg.TargetObject != nil {
				c.processTweet(messageID, msg.TargetObject)
			}

		case *twitter.StatusWithheld:
			_, err := c.insertMessage(m.account, msg)
			if err != nil {
				log.WithError(err).Error("Failed to insert message")
			}
			log.WithFields(log.Fields{
				"id": strconv.FormatInt(msg.ID, 10), "user": strconv.FormatInt(msg.UserID, 10),
				"countries": strings.Join(msg.WithheldInCountries, ","),
			}).Info("Status withheld")
		case *twitter.UserWithheld:
			_, err := c.insertMessage(m.account, msg)
			if err != nil {
				log.WithError(err).Error("Failed to insert message")
			}
			log.WithFields(log.Fields{
				"user":      strconv.FormatInt(msg.ID, 10),
				"countries": strings.Join(msg.WithheldInCountries, ","),
			}).Info("User withheld")

		case *twitter.StreamLimit:
			log.WithFields(log.Fields{
				"type": "StreamLimit", "track": msg.Track,
			}).Warn("Warning message")
		case *twitter.StreamDisconnect:
			log.WithFields(log.Fields{
				"type": "StreamDisconnect", "reason": msg.Reason,
				"name": msg.StreamName, "code": msg.Code,
			}).Warn("Warning message")
		case *twitter.StallWarning:
			log.WithFields(log.Fields{
				"type": "StallWarning", "message": msg.Message,
				"code": msg.Code, "percent": msg.PercentFull,
			}).Warn("Warning message")
		}
	}
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
	// TODO: retry
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
