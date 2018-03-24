package covfefe

import (
	"encoding/json"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (c *Covfefe) initDB() error {
	_, err := c.db.Exec(`
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
	);`)

	return errors.Wrap(err, "failed to initialize database")
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
