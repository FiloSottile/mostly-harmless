package covfefe

import (
	"fmt"
	"os"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/pkg/errors"
	"github.com/schollz/progressbar/v2"
	log "github.com/sirupsen/logrus"
	"github.com/v2pro/plz/gls"
)

func Rescan(dbPath string) (err error) {
	// We use a single connection for performance and rollback.
	conn, err := sqlite.OpenConn("file:"+dbPath, 0)
	if err != nil {
		return errors.Wrap(err, "failed to open database")
	}
	defer conn.Close()
	defer sqlitex.Save(conn)(&err)

	mainID := gls.GoID()
	c := &Covfefe{
		withConn: func(f func(conn *sqlite.Conn) error) error {
			if gls.GoID() != mainID {
				// This goroutine owns the conn, as it has an open
				// statement until the end. Also, no locking!
				panic("rescan should not use multiple goroutines")
			}
			return f(conn)
		},
		rescan: true,
	}

	if err := c.initDB(); err != nil {
		return err
	}

	log.Info("Dropping tables...")

	// Need to have foreign keys OFF for TRUNCATE
	if err := sqlitex.ExecScript(conn, `
		DELETE FROM Tweets;
		DELETE FROM Users;
		DELETE FROM Follows;
	`); err != nil {
		return errors.Wrap(err, "failed to truncate tables")
	}

	log.Info("Starting rescan...")

	count, err := sqlitex.ResultInt64(conn.Prep("SELECT COUNT(*) FROM Messages;"))
	if err != nil {
		return errors.Wrap(err, "failed to count messages")
	}
	pb := progressbar.NewOptions64(count, progressbar.OptionShowCount())

	if err := sqlitex.Exec(conn, "SELECT id, json, kind FROM Messages;",
		func(stmt *sqlite.Stmt) error {
			c.Handle(&Message{
				id:   stmt.GetInt64("id"),
				kind: stmt.GetText("kind"),
				msg:  []byte(stmt.GetText("json")),
			})
			pb.Add(1)
			return nil
		}); err != nil {
		return errors.Wrap(err, "listing Messages failed")
	}

	fmt.Fprintf(os.Stderr, "\n")
	log.Info("Finishing up...")
	return nil
}
