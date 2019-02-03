package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqliteutil"
	log "github.com/sirupsen/logrus"
)

func main() {
	dbFile := flag.String("db", "twitter.db", "The path of the SQLite DB")
	mediaPath := flag.String("media", "twitter-media", "The folder to store media in")
	flag.Parse()

	log.SetLevel(log.DebugLevel)

	db, err := sqlite.Open("file:"+*dbFile, 0, 2)
	if err != nil {
		log.WithError(err).Fatal("Failed to open database")
	}
	defer db.Close()
	conn := db.Get(nil)

	log.WithError(sqliteutil.Exec(db.Get(nil), `SELECT id, SUBSTR(media, 1, 4), media FROM Media`,
		func(stmt *sqlite.Stmt) error {
			var ext string
			switch stmt.ColumnText(1) {
			case "\x89PNG":
				ext = "png"
			case "\xFF\xD8\xFF\xE0":
				ext = "jpg"
			default:
				log.WithError(err).Fatal("Unknown media type")
			}
			name := fmt.Sprintf("%d.%s", stmt.GetInt64("id"), ext)

			f, err := os.Create(filepath.Join(*mediaPath, name))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, stmt.GetReader("media")); err != nil {
				f.Close()
				os.Remove(f.Name())
				return err
			}
			if err := f.Close(); err != nil {
				os.Remove(f.Name())
				return err
			}
			log.WithField("file", name).Debug("Copied file")
			return sqliteutil.Exec(conn, `DELETE FROM Media WHERE id = ?`, nil,
				stmt.GetInt64("id"))
		})).Info("Done")
}
