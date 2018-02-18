package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/mattn/go-sqlite3"
)

func main() {
	dbFile := flag.String("db", "twitter.db", "The path of the SQLite DB")
	credsFile := flag.String("creds", "creds.json", "The path of the credentials JSON")
	flag.Parse()

	credsJSON, err := ioutil.ReadFile(*credsFile)
	if err != nil {
		log.Fatal(err)
	}
	var creds map[string]string
	if err := json.Unmarshal(credsJSON, &creds); err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite3", "file:"+*dbFile+"?_foreign_keys=1")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	mustExec := func(query string, args ...interface{}) {
		_, err = db.Exec(query, args...)
		if err != nil {
			debug.PrintStack()
			log.Fatalln("Failed to execute query:", err)
		}
	}
	mustInsert := func(query string, args ...interface{}) (id int64) {
		res, err := db.Exec(query, args...)
		if err != nil {
			debug.PrintStack()
			log.Fatalln("Failed to execute insert:", err)
		}
		id, err = res.LastInsertId()
		if err != nil {
			debug.PrintStack()
			log.Fatalln("Failed to get id:", err)
		}
		return id
	}

	mustExec(`CREATE TABLE IF NOT EXISTS Messages (
		id INTEGER PRIMARY KEY,
		created TEXT NOT NULL,
		json TEXT NOT NULL
	)`)
	insertMessage := func(object interface{}, created time.Time) (id int64) {
		return mustInsert(
			`INSERT INTO Messages (created, json) VALUES (?, ?)`,
			created, mustMarshal(object))
	}
	mustExec(`CREATE TABLE IF NOT EXISTS Tweets (
		id INTEGER PRIMARY KEY,
		created TEXT NOT NULL,
		user TEXT NOT NULL,
		message INTEGER NOT NULL REFERENCES Messages(id),
		deleted TEXT
	)`)
	insertTweet := func(tweet *twitter.Tweet, message int64) (new bool) {
		_, err := db.Exec(
			`INSERT INTO Tweets (id, created, user, message) VALUES (?, ?, ?, ?)`,
			tweet.ID, mustTimeParse(tweet.CreatedAt), tweet.User.ScreenName, message)
		if err, ok := err.(*sqlite3.Error); ok && err.ExtendedCode != sqlite3.ErrConstraintPrimaryKey {
			return false
		} else if err != nil {
			debug.PrintStack()
			log.Fatalln("Failed to execute insert:", err)
		}
		return true
	}
	deletedTweet := func(id int64) {
		mustExec(`UPDATE Tweets SET deleted = datetime('now') WHERE id = ?`, id)
	}
	mustExec(`CREATE TABLE IF NOT EXISTS Media (
		id INTEGER PRIMARY KEY,
		media BLOB NOT NULL,
		tweet INTEGER NOT NULL REFERENCES Tweets(id)
	)`)
	insertMedia := func(data []byte, id, tweet int64) {
		mustInsert(`INSERT INTO Media (id, media, tweet) VALUES (?, ?, ?)`, id, data, tweet)
	}

	config := oauth1.NewConfig(creds["API_KEY"], creds["API_SECRET"])
	// TODO: multiple tokens
	token := oauth1.NewToken(creds["TOKEN"], creds["TOKEN_SECRET"])
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)

	params := &twitter.StreamUserParams{
		With:          "followings",
		StallWarnings: twitter.Bool(true),
	}
	stream, err := client.Streams.User(params)
	if err != nil {
		log.Fatal(err)
	}
	demux := twitter.NewSwitchDemux()

	demux.Tweet = func(tweet *twitter.Tweet) {
		messageID := insertMessage(tweet, mustTimeParse(tweet.CreatedAt))

		var processTweet func(*twitter.Tweet)
		processTweet = func(tweet *twitter.Tweet) {
			if insertTweet(tweet, messageID) {
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
					go func() {
						for _, m := range media {
							if m.SourceStatusID != 0 {
								// We'll find this media attached to the retweet.
								continue
							}
							insertMedia(mustGet(m.MediaURLHttps), m.ID, tweet.ID)
							// TODO: archive videos?
						}
					}()
				}
				if tweet.RetweetedStatus != nil {
					processTweet(tweet.RetweetedStatus)
				}
				if tweet.QuotedStatus != nil {
					processTweet(tweet.QuotedStatus)
				}
				// TODO: crawl thread, non-embedded linked tweets
			}
		}

		processTweet(tweet)
	}
	demux.StatusDeletion = func(deletion *twitter.StatusDeletion) {
		deletedTweet(deletion.ID)
	}
	demux.Event = func(event *twitter.Event) {
		insertMessage(event, mustTimeParse(event.CreatedAt))
	}

	demux.StreamLimit = func(limit *twitter.StreamLimit) {
		log.Println("Stream limit:", limit.Track)
	}
	demux.StreamDisconnect = func(disconnect *twitter.StreamDisconnect) {
		log.Println("Stream disconnect:", disconnect.Reason)
	}
	demux.Warning = func(warning *twitter.StallWarning) {
		log.Println("Warning:", warning.Message)
	}

	demux.HandleChan(stream.Messages)
}

func mustMarshal(v interface{}) []byte {
	j, err := json.Marshal(v)
	if err != nil {
		log.Fatalln("Failed to marshal JSON:", err)
	}
	return j
}

func mustTimeParse(CreatedAt string) time.Time {
	t, err := time.Parse(time.RubyDate, CreatedAt)
	if err != nil {
		log.Fatalln("Failed to get created time:", err)
	}
	return t
}

var httpClient = &http.Client{
	Timeout: 1 * time.Minute,
}

func mustGet(url string) []byte {
	res, err := httpClient.Get(url)
	if err != nil {
		log.Fatalln("Failed HTTP request:", err)
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln("Failed HTTP read:", err)
	}
	return data
}
