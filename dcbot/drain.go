package main

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqliteutil"
	"github.com/sirupsen/logrus"
)

func (dcb *DocumentCloudBot) Drain(ctx context.Context) error {
	log := logrus.WithField("module", "drain")

	return dcb.withConn(ctx, func(conn *sqlite.Conn) error {
		return sqliteutil.Exec(conn, `SELECT rowid, document, type, content FROM Files`,
			func(stmt *sqlite.Stmt) error {
				if ctx.Err() != nil {
					log.WithError(ctx.Err()).Debug("Shutting down")
					return ctx.Err()
				}
				name := stmt.GetText("document") + "." + stmt.GetText("type")
				f, err := os.Create(filepath.Join(dcb.filePath, name))
				if err != nil {
					return err
				}
				if _, err := io.Copy(f, stmt.GetReader("content")); err != nil {
					f.Close()
					os.Remove(f.Name())
					return err
				}
				if err := f.Close(); err != nil {
					os.Remove(f.Name())
					return err
				}
				log.WithField("file", name).Debug("Copied file")
				return dcb.withConn(ctx, func(conn *sqlite.Conn) error {
					return sqliteutil.Exec(conn, `DELETE FROM Files WHERE rowid = ?`, nil,
						stmt.GetInt64("rowid"))
				})
			})
	})
}
