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
	if sqlite.ErrCode(err) == sqlite.SQLITE_CONSTRAINT_UNIQUE {
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

func (dcb *DocumentCloudBot) markRetrieved(ctx context.Context, document string) error {
	return dcb.withConn(ctx, func(conn *sqlite.Conn) (err error) {
		return errors.Wrap(sqliteutil.Exec(conn,
			`UPDATE Documents SET retrieved = DATETIME('now') WHERE id = ?`,
			nil, document), "failed to update document")
	})
}
