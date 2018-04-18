package main

import (
	"context"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqliteutil"
	"github.com/pkg/errors"
)

func (dcb *DocumentCloudBot) initDB(ctx context.Context) error {
	return dcb.withConn(ctx, func(conn *sqlite.Conn) error {
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

func (dcb *DocumentCloudBot) insertFiles(ctx context.Context, document string, files map[string][]byte) error {
	return dcb.withConn(ctx, func(conn *sqlite.Conn) (err error) {
		defer sqliteutil.Save(conn)(&err)
		for typ, content := range files {
			err = sqliteutil.Exec(conn, `INSERT INTO Files VALUES (?, ?, ?)`, nil, document, typ, content)
			if err != nil {
				return
			}
		}
		return sqliteutil.Exec(conn,
			`UPDATE Documents SET retrieved = DATETIME('now') WHERE document = ?`,
			nil, document)
	})
}
