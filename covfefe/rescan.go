package covfefe

import (
	"database/sql"
	"encoding/json"

	"github.com/FiloSottile/mostly-harmless/covfefe/internal/twitter"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func Rescan(dbPath string) error {
	db, err := sql.Open("sqlite3", "file:"+dbPath)
	if err != nil {
		return errors.Wrap(err, "failed to open database")
	}
	defer db.Close()

	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		return errors.Wrap(err, "failed to convert to WAL")
	}
	db.SetMaxOpenConns(1)

	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}

	c := &Covfefe{rescan: true, db: tx}

	if err := c.initDB(); err != nil {
		return err
	}

	log.Info("Dropping tables...")

	// Need to have foreign keys OFF for TRUNCATE
	_, err = tx.Exec(`
		DELETE FROM Tweets;
		DELETE FROM Users;
		DELETE FROM Follows;
	`)
	if err != nil {
		return errors.Wrap(err, "failed to truncate tables")
	}

	log.Info("Starting rescan...")

	rows, err := tx.Query("SELECT id, json FROM Messages")
	if err != nil {
		log.WithError(err).Fatal("Query failed")
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var json []byte
		if err := rows.Scan(&id, &json); err != nil {
			log.WithError(err).Fatal("Scan failed")
		}
		c.Handle(&Message{id: id, msg: getMessage(json)})
	}
	if err := rows.Err(); err != nil {
		log.WithError(err).Fatal("Query returned an error")
	}

	log.Info("Finishing up...")

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}
	_, err = db.Exec("PRAGMA foreign_keys = ON; PRAGMA foreign_key_check;")
	if err != nil {
		return errors.Wrap(err, "failed to check foreign keys")
	}

	return nil
}

func getMessage(token []byte) interface{} {
	var data map[string]json.RawMessage
	err := json.Unmarshal(token, &data)
	if err != nil {
		panic(err)
	}

	var res interface{}
	switch {
	case hasPath(data, "retweet_count"):
		res = new(twitter.Tweet)
	case hasPath(data, "event"):
		res = new(twitter.Event)
	case hasPath(data, "withheld_in_countries") && hasPath(data, "user_id"):
		res = new(twitter.StatusWithheld)
	case hasPath(data, "withheld_in_countries"):
		res = new(twitter.UserWithheld)
	case hasPath(data, "synthetic"):
		fallthrough // migrated deletion events
	case hasPath(data, "user_id_str"):
		res = new(twitter.StatusDeletion)
	default:
		panic("unknown Twitter message type")
	}
	json.Unmarshal(token, res)
	return res
}

func hasPath(data map[string]json.RawMessage, key string) bool {
	_, ok := data[key]
	return ok
}
