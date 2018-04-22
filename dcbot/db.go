package main

import (
	"context"
	"io"
	"os"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqliteutil"
	"github.com/pkg/errors"
)

func (dcb *DocumentCloudBot) initDB(ctx context.Context) error {
	return dcb.withConn(ctx, func(conn *sqlite.Conn) error {
		// https://github.com/crawshaw/sqlite/issues/7
		err := sqliteutil.ExecTransient(conn, `PRAGMA journal_mode=WAL`, nil)
		if err != nil {
			return err
		}
		return sqliteutil.ExecScript(conn, `
			CREATE TABLE IF NOT EXISTS Documents (
				id TEXT PRIMARY KEY,
				json TEXT NOT NULL,
				retrieved DATETIME
			);
			CREATE TABLE IF NOT EXISTS Files (
				document TEXT NOT NULL REFERENCES Documents(id),
				type TEXT NOT NULL,
				content BLOB NOT NULL,
				UNIQUE (document, type) 
			);
		`)
	})
}

func (dcb *DocumentCloudBot) insertDocument(ctx context.Context, id string, body []byte) (new bool, err error) {
	err = dcb.withConn(ctx, func(conn *sqlite.Conn) error {
		return sqliteutil.Exec(conn, `INSERT INTO Documents (id, json) VALUES (?, ?)`, nil, id, body)
	})
	if sqlite.ErrCode(err) == sqlite.SQLITE_CONSTRAINT || // https://github.com/crawshaw/sqlite/issues/5
		sqlite.ErrCode(err) == sqlite.SQLITE_CONSTRAINT_UNIQUE {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "failed to insert document")
	}
	return true, nil
}

func (dcb *DocumentCloudBot) getPendingDocument(ctx context.Context) ([]byte, error) {
	var result []byte
	err := dcb.withConn(ctx, func(conn *sqlite.Conn) error {
		return sqliteutil.Exec(conn, `SELECT json FROM Documents WHERE retrieved IS NULL LIMIT 1`,
			func(stmt *sqlite.Stmt) error {
				if result != nil {
					return errors.New("unexpected multiple results")
				}
				result = []byte(stmt.ColumnText(0))
				return nil
			})
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to select pending document")
	}
	return result, nil
}

func (dcb *DocumentCloudBot) insertFiles(ctx context.Context, document string, files map[string]*os.File) error {
	return dcb.withConn(ctx, func(conn *sqlite.Conn) (err error) {
		defer sqliteutil.Save(conn)(&err)
		for typ := range files {
			fi, err := files[typ].Stat()
			if err != nil {
				return errors.Wrap(err, "failed to stat file")
			}
			if fi.Size() == 0 {
				continue
			}
			query := `INSERT INTO Files (document, type, content) VALUES (?, ?, ?)`
			stmt, err := conn.Prepare(query)
			if err != nil {
				return errors.Wrapf(err, "failed to prepare query (%s)", query)
			}
			stmt.BindText(1, document)
			stmt.BindText(2, typ)
			stmt.BindZeroBlob(3, fi.Size())
			if _, err := stmt.Step(); err != nil {
				return errors.Wrapf(err, "failed to insert file of size %d and type %s for document %s",
					fi.Size(), typ, document)
			}
			b, err := conn.OpenBlob("", "Files", "content", conn.LastInsertRowID(), true)
			if err != nil {
				return errors.Wrapf(err, "failed to open file of size %d and type %s for document %s",
					fi.Size(), typ, document)
			}
			if _, err := io.Copy(NewAtWriter(b), files[typ]); err != nil {
				return errors.Wrapf(err, "failed to copy file of size %d and type %s for document %s",
					fi.Size(), typ, document)
			}
			if err := b.Close(); err != nil {
				return errors.Wrapf(err, "failed to close file of size %d and type %s for document %s",
					fi.Size(), typ, document)
			}

		}
		return errors.Wrap(sqliteutil.Exec(conn,
			`UPDATE Documents SET retrieved = DATETIME('now') WHERE id = ?`,
			nil, document), "failed to update document")
	})
}

type AtWriter struct {
	io.WriterAt
	off int64
}

func NewAtWriter(w io.WriterAt) *AtWriter {
	return &AtWriter{WriterAt: w}
}

func (at *AtWriter) Write(p []byte) (n int, err error) {
	n, err = at.WriteAt(p, at.off)
	at.off += int64(n)
	return n, err
}
