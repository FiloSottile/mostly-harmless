package covfefe

import (
	"encoding/base64"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/blake2b"
)

func (c *Covfefe) execSQL(query string, args ...interface{}) error {
	return c.withConn(func(conn *sqlite.Conn) error {
		return sqlitex.Exec(conn, query, nil, args...)
	})
}

func (c *Covfefe) initDB() error {
	return errors.Wrap(c.withConn(func(conn *sqlite.Conn) error {
		return sqlitex.ExecScript(conn, `
			CREATE TABLE IF NOT EXISTS Messages (
				id INTEGER PRIMARY KEY,
				received DATETIME DEFAULT (DATETIME('now')),
				json TEXT NOT NULL,
				source TEXT NOT NULL, -- JSON array of source IDs
				kind TEXT -- tweet / event / del / deletion
			);
			CREATE TABLE IF NOT EXISTS Tweets (
				id INTEGER PRIMARY KEY,
				created DATETIME NOT NULL,
				user INTEGER NOT NULL,
				message INTEGER NOT NULL REFERENCES Messages(id),
				deleted INTEGER REFERENCES Messages(id)
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
	log := log.WithFields(log.Fields{"source": m.source, "kind": m.kind})

	if m.id != 0 {
		log.WithField("id", m.id).Debug("Read message")
		return nil
	}

	h, _ := blake2b.New256([]byte(m.kind))
	h.Write(m.msg)
	log = log.WithField("hash", base64.RawURLEncoding.EncodeToString(h.Sum(nil)))

	if id, ok := c.msgIDs.Get(h); ok {
		log.WithField("id", id).Debug("Duplicate message")

		err := c.execSQL(`UPDATE Messages SET source = json_insert(
			source, '$[' || json_array_length(source) || ']', ?) WHERE id = ?;`, m.source, id)
		if err != nil {
			return errors.Wrap(err, "failed update query")
		}
		m.id = id.(int64)
		return nil
	}

	err := c.withConn(func(conn *sqlite.Conn) error {
		query := `INSERT INTO Messages (json, source, kind) VALUES (?, json_array(?), ?)`
		err := sqlitex.Exec(conn, query, nil, m.msg, m.source, m.kind)
		if err != nil {
			return err
		}
		m.id = conn.LastInsertRowID()
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "failed insert query")
	}

	log.WithField("id", m.id).Debug("New message")

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

func (c *Covfefe) seenTweet(id int64) (found bool, err error) {
	err = errors.Wrap(c.withConn(func(conn *sqlite.Conn) error {
		return sqlitex.Exec(conn, "SELECT id FROM Tweets WHERE id = ?",
			func(stmt *sqlite.Stmt) error {
				found = true
				return nil
			}, id)
	}), "failed select query")
	return
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
