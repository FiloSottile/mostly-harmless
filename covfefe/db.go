// The SQL in this file requires SQLite to be built with the json1 extension.
// +build json1

package covfefe

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/blake2b"
)

func (c *Covfefe) initDB() error {
	_, err := c.db.Exec(`
	CREATE TABLE IF NOT EXISTS Messages (
		id INTEGER PRIMARY KEY,
		received DATETIME DEFAULT (DATETIME('now')),
		json TEXT NOT NULL,
		account TEXT NOT NULL -- JSON array of IDs
	);
	CREATE TABLE IF NOT EXISTS Tweets (
		id INTEGER PRIMARY KEY,
		created DATETIME NOT NULL,
		user INTEGER NOT NULL,
		message INTEGER NOT NULL REFERENCES Messages(id),
		deleted INTEGER REFERENCES Messages(id)
	);
	CREATE TABLE IF NOT EXISTS Media (
		id INTEGER PRIMARY KEY,
		media BLOB NOT NULL,
		tweet INTEGER NOT NULL REFERENCES Tweets(id)
	);
	CREATE TABLE IF NOT EXISTS Users (
		id INTEGER NOT NULL,
		handle TEXT NOT NULL,
		name TEXT NOT NULL,
		bio TEXT NOT NULL,
		first_seen INTEGER NOT NULL REFERENCES Messages(id),
		UNIQUE (id, handle, name, bio) ON CONFLICT IGNORE
	);
	CREATE TABLE IF NOT EXISTS Follows (
		follower INTEGER NOT NULL,
		target INTEGER NOT NULL,
		first_seen INTEGER NOT NULL REFERENCES Messages(id),
		UNIQUE (target, follower) ON CONFLICT IGNORE
	);`)

	return errors.Wrap(err, "failed to initialize database")
}

func (c *Covfefe) insertMessage(m *Message) error {
	if m.id != 0 {
		return nil
	}

	msg := mustMarshal(m.msg)
	h := blake2b.Sum256(msg)

	if id, ok := c.msgIDs.Get(h); ok {
		log.WithFields(log.Fields{
			"id": id, "account": m.account.ScreenName, "hash": base64.RawURLEncoding.EncodeToString(h[:]),
		}).Debug("Duplicate message")

		_, err := c.db.Exec(`UPDATE Messages SET account = (
				SELECT json_group_array(value) FROM (
					SELECT json_each.value
					FROM Messages, json_each(Messages.account)
					WHERE Messages.id = ?
					UNION ALL SELECT ?
				) GROUP BY ''
			) WHERE id = ?;`, id, m.account.ID, id)
		if err != nil {
			return errors.Wrap(err, "failed update query")
		}
		m.id = id.(int64)
		return nil
	}

	res, err := c.db.Exec(`INSERT INTO Messages (json, account) VALUES (?, json_array(?))`, msg, m.account.ID)
	if err != nil { // TODO: retry
		return errors.Wrap(err, "failed insert query")
	}
	m.id, err = res.LastInsertId()
	if err != nil {
		return errors.Wrap(err, "failed to get message id")
	}

	log.WithFields(log.Fields{
		"id": m.id, "account": m.account.ScreenName, "hash": base64.RawURLEncoding.EncodeToString(h[:]),
	}).Debug("New message")

	c.msgIDs.Add(h, m.id)
	return nil
}

func (c *Covfefe) insertTweet(tweet *twitter.Tweet, message int64) (new bool, err error) {
	_, err = c.db.Exec(
		`INSERT INTO Tweets (id, created, user, message) VALUES (?, ?, ?, ?)`,
		tweet.ID, mustParseTime(tweet.CreatedAt), tweet.User.ID, message)
	if err, ok := err.(sqlite3.Error); ok && err.ExtendedCode != sqlite3.ErrConstraintUnique {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "failed insert query")
	}
	return true, nil
}

func (c *Covfefe) insertUser(user *twitter.User, message int64) error {
	_, err := c.db.Exec(`INSERT INTO Users (id, handle, name, bio, first_seen)
		VALUES (?, ?, ?, ?, ?)`, user.ID, user.ScreenName, user.Name, user.Description, message)
	return errors.Wrap(err, "failed insert query")
}

func (c *Covfefe) insertFollow(follower, target, message int64) error {
	_, err := c.db.Exec(`INSERT INTO Follows (follower, target, first_seen)
		VALUES (?, ?, ?)`, follower, target, message)
	return errors.Wrap(err, "failed insert query")
}

func (c *Covfefe) insertMedia(data []byte, id, tweet int64) {
	_, err := c.db.Exec(`INSERT INTO Media (id, media, tweet) VALUES (?, ?, ?)`, id, data, tweet)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err, "media": id, "tweet": tweet,
		}).Error("Failed to insert media")
	}
}

func (c *Covfefe) deletedTweet(tweet, message int64) {
	_, err := c.db.Exec(`UPDATE Tweets SET deleted = ? WHERE id = ?`, message, tweet)
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
