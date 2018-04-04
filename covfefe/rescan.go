package covfefe

import (
	"database/sql"

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
		c.Handle(&Message{id: id, msg: json})
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
