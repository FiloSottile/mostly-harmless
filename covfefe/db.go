package covfefe

import (
	"encoding/base64"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqliteutil"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/blake2b"
)

func (c *Covfefe) execSQL(query string, args ...interface{}) error {
	return c.withConn(func(conn *sqlite.Conn) error {
		return sqliteutil.Exec(conn, query, nil, args...)
	})
}

func (c *Covfefe) initDB() error {
	return errors.Wrap(c.withConn(func(conn *sqlite.Conn) error {
		return sqliteutil.ExecScript(conn, `
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
	}), "failed to initialize database")
}

func (c *Covfefe) insertMessage(m *Message) error {
	if m.id != 0 {
		return nil
	}

	h := blake2b.Sum256(m.msg)

	if id, ok := c.msgIDs.Get(h); ok {
		log.WithFields(log.Fields{
			"id": id, "account": m.account.ScreenName, "hash": base64.RawURLEncoding.EncodeToString(h[:]),
		}).Debug("Duplicate message")

		err := c.execSQL(`UPDATE Messages SET account = json_insert(
			account, '$[' || json_array_length(account) || ']', ?) WHERE id = ?;`, m.account.ID, id)
		if err != nil {
			return errors.Wrap(err, "failed update query")
		}
		m.id = id.(int64)
		return nil
	}

	err := c.withConn(func(conn *sqlite.Conn) error {
		err := sqliteutil.Exec(conn,
			`INSERT INTO Messages (json, account) VALUES (?, json_array(?))`,
			nil, m.msg, m.account.ID)
		if err != nil {
			return err
		}
		m.id = conn.LastInsertRowID()
		return nil
	})
	if err != nil { // TODO: retry
		return errors.Wrap(err, "failed insert query")
	}

	log.WithFields(log.Fields{
		"id": m.id, "account": m.account.ScreenName, "hash": base64.RawURLEncoding.EncodeToString(h[:]),
	}).Debug("New message")

	c.msgIDs.Add(h, m.id)
	return nil
}

func (c *Covfefe) insertTweet(tweet *twitter.Tweet, message int64) (new bool, err error) {
	err = c.execSQL(
		`INSERT INTO Tweets (id, created, user, message) VALUES (?, ?, ?, ?)`,
		tweet.ID, mustParseTime(tweet.CreatedAt), tweet.User.ID, message)
	if sqlite.ErrCode(err) == sqlite.SQLITE_CONSTRAINT_PRIMARYKEY {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "failed insert query")
	}
	return true, nil
}

func (c *Covfefe) insertUser(user *twitter.User, message int64) error {
	return errors.Wrap(c.execSQL(
		`INSERT INTO Users (id, handle, name, bio, first_seen) VALUES (?, ?, ?, ?, ?);`,
		user.ID, user.ScreenName, user.Name, user.Description, message), "failed insert query")
}

func (c *Covfefe) insertFollow(follower, target, message int64) error {
	return errors.Wrap(c.execSQL(
		`INSERT INTO Follows (follower, target, first_seen) VALUES (?, ?, ?);`,
		follower, target, message), "failed insert query")
}

func (c *Covfefe) insertMedia(data []byte, id, tweet int64) {
	err := c.execSQL(`INSERT INTO Media (id, media, tweet) VALUES (?, ?, ?)`, id, data, tweet)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err, "media": id, "tweet": tweet,
		}).Error("Failed to insert media")
	}
}

func (c *Covfefe) deletedTweet(tweet, message int64) {
	err := c.execSQL(`UPDATE Tweets SET deleted = ? WHERE id = ?`, message, tweet)
	if err != nil {
		log.WithError(err).WithField("tweet", tweet).Error("Failed to delete tweet")
	}
}

func mustParseTime(CreatedAt string) time.Time {
	t, err := time.Parse(time.RubyDate, CreatedAt)
	if err != nil {
		log.WithError(err).WithField("string", CreatedAt).Fatal("Failed to parse created time")
	}
	return t
}
