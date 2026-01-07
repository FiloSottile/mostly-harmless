package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	indigoutil "github.com/bluesky-social/indigo/util"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"golang.org/x/sync/errgroup"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func main() {
	dbFlag := flag.String("db", "atsites.sqlite3", "path to the SQLite database file")
	tapFlag := flag.String("tap", "ws://localhost:2480/channel", "Tap WebSocket URL")
	debugFlag := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	if *debugFlag {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	group, ctx := errgroup.WithContext(ctx)

	db, err := sqlitex.NewPool(*dbFlag, sqlitex.PoolOptions{
		PrepareConn: func(db *sqlite.Conn) error {
			db.SetInterrupt(ctx.Done())
			return sqlitex.ExecuteTransient(db, `PRAGMA foreign_keys = ON;`, nil)
		},
	})
	if err != nil {
		slog.Error("failed to open SQLite database", "error", err)
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close SQLite database", "error", err)
		}
	}()
	if err := initDatabase(ctx, db); err != nil {
		slog.Error("failed to initialize database", "error", err)
		return
	}

	s := &Server{db: db}
	group.Go(func() error {
		c, _, err := websocket.Dial(ctx, *tapFlag, nil)
		if err != nil {
			return err
		}
		defer c.CloseNow()
		c.SetReadLimit(-1) // no limit

		slog.Info("connected to tap", "url", *tapFlag)
		return s.handleTap(ctx, c)
	})

	slog.Error("stopping", "error", group.Wait())
}

func initDatabase(ctx context.Context, pool *sqlitex.Pool) error {
	db, err := pool.Take(ctx)
	if err != nil {
		return err
	}
	defer pool.Put(db)

	return sqlitex.ExecScript(db, `
		CREATE TABLE IF NOT EXISTS publications (
			repo TEXT NOT NULL, -- did
			rkey TEXT NOT NULL,
			record_json BLOB NOT NULL, -- JSONB
			PRIMARY KEY (repo, rkey)
		) STRICT;
		CREATE TABLE IF NOT EXISTS documents (
			repo TEXT NOT NULL, -- did
			rkey TEXT NOT NULL,
			publication_repo TEXT NOT NULL,
			publication_rkey TEXT NOT NULL,
			document_json BLOB NOT NULL, -- JSONB
			PRIMARY KEY (repo, rkey)
			-- No foreign key because we might observe documents before their publication
		) STRICT;
		CREATE INDEX IF NOT EXISTS idx_documents_publication
			ON documents (publication_repo, publication_rkey);
	`)
}

type Server struct {
	db *sqlitex.Pool
}

func (s *Server) handleTap(ctx context.Context, c *websocket.Conn) error {
	for {
		var msg struct {
			ID     int             `json:"id"`
			Type   string          `json:"type"` // record, identity
			Record json.RawMessage `json:"record"`
		}
		if err := wsjson.Read(ctx, c, &msg); err != nil {
			return err
		}

		switch msg.Type {
		case "identity":
			slog.DebugContext(ctx, "received identity event")
			continue
		case "record":
			var rec recordEvent
			if err := json.Unmarshal(msg.Record, &rec); err != nil {
				return fmt.Errorf("unmarshal record: %w", err)
			}
			slog.DebugContext(ctx, "received record event", "action", rec.Action,
				"uri", fmt.Sprintf("at://%s/%s/%s", rec.Repo, rec.Collection, rec.Rkey))
			if err := s.handleRecordEvent(ctx, &rec); err != nil {
				return fmt.Errorf("handle record event: %w", err)
			}
		default:
			slog.WarnContext(ctx, "unknown event type", "type", msg.Type)
		}

		if err := wsjson.Write(ctx, c, struct {
			Type string `json:"type"`
			ID   int    `json:"id"`
		}{
			Type: "ack",
			ID:   msg.ID,
		}); err != nil {
			return fmt.Errorf("send ack: %w", err)
		}
	}
}

type recordEvent struct {
	Repo       string          `json:"did"`
	Rkey       string          `json:"rkey"`
	Collection string          `json:"collection"`
	Action     string          `json:"action"` // create, update, delete
	Record     json.RawMessage `json:"record"`
}

func (s *Server) handleRecordEvent(ctx context.Context, rec *recordEvent) error {
	db, err := s.db.Take(ctx)
	if err != nil {
		return err
	}
	defer s.db.Put(db)

	switch rec.Collection {
	case "site.standard.publication":
		if rec.Action == "delete" {
			return s.deletePublication(ctx, db, rec.Repo, rec.Rkey)
		}
		return s.storePublication(ctx, db, rec.Repo, rec.Rkey, rec.Record)
	case "site.standard.document":
		if rec.Action == "delete" {
			return s.deleteDocument(ctx, db, rec.Repo, rec.Rkey)
		}
		var r struct {
			Site string `json:"site"`
		}
		if err := json.Unmarshal(rec.Record, &r); err != nil {
			slog.WarnContext(ctx, "unmarshal document record", "error", err,
				"uri", fmt.Sprintf("at://%s/%s/%s", rec.Repo, rec.Collection, rec.Rkey))
			return nil
		}
		// The site can be an https:// URL for loose documents, ignore those.
		if !strings.HasPrefix(r.Site, "at://") {
			slog.DebugContext(ctx, "document site is not an at:// URI", "site", r.Site,
				"uri", fmt.Sprintf("at://%s/%s/%s", rec.Repo, rec.Collection, rec.Rkey))
			return nil
		}
		u, err := indigoutil.ParseAtUri(r.Site)
		if err != nil {
			slog.WarnContext(ctx, "parse at:// URI in document site", "error", err, "site", r.Site,
				"uri", fmt.Sprintf("at://%s/%s/%s", rec.Repo, rec.Collection, rec.Rkey))
			return nil
		}
		if u.Collection != "site.standard.publication" {
			slog.WarnContext(ctx, "document site does not point to a publication", "site", r.Site,
				"uri", fmt.Sprintf("at://%s/%s/%s", rec.Repo, rec.Collection, rec.Rkey))
			return nil
		}
		return s.storeDocument(ctx, db, rec.Repo, rec.Rkey, u.Did, u.Rkey, rec.Record)
	default:
		slog.DebugContext(ctx, "ignoring record from unknown collection",
			"uri", fmt.Sprintf("at://%s/%s/%s", rec.Repo, rec.Collection, rec.Rkey))
	}

	return nil
}

func (s *Server) storePublication(ctx context.Context, db *sqlite.Conn, repo, rkey string, record json.RawMessage) error {
	slog.DebugContext(ctx, "storing publication", "repo", repo, "rkey", rkey)
	return sqlitex.Execute(db, `
		INSERT INTO publications (repo, rkey, record_json)
		VALUES (?, ?, ?)
		ON CONFLICT(repo, rkey) DO UPDATE SET record_json=excluded.record_json
	`, &sqlitex.ExecOptions{
		Args: []any{repo, rkey, record},
	})
}

func (s *Server) deletePublication(ctx context.Context, db *sqlite.Conn, repo, rkey string) error {
	slog.DebugContext(ctx, "deleting publication", "repo", repo, "rkey", rkey)
	return sqlitex.Execute(db, `
		DELETE FROM publications
		WHERE repo = ? AND rkey = ?
	`, &sqlitex.ExecOptions{
		Args: []any{repo, rkey},
	})
}

func (s *Server) storeDocument(ctx context.Context, db *sqlite.Conn, repo, rkey, pubRepo, pubRkey string, record json.RawMessage) error {
	slog.DebugContext(ctx, "storing document", "repo", repo, "rkey", rkey,
		"publication_repo", pubRepo, "publication_rkey", pubRkey)
	return sqlitex.Execute(db, `
		INSERT INTO documents (repo, rkey, publication_repo, publication_rkey, document_json)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(repo, rkey) DO UPDATE SET document_json=excluded.document_json, publication_repo=excluded.publication_repo, publication_rkey=excluded.publication_rkey
	`, &sqlitex.ExecOptions{
		Args: []any{repo, rkey, pubRepo, pubRkey, record},
	})
}

func (s *Server) deleteDocument(ctx context.Context, db *sqlite.Conn, repo, rkey string) error {
	slog.DebugContext(ctx, "deleting document", "repo", repo, "rkey", rkey)
	return sqlitex.Execute(db, `
		DELETE FROM documents
		WHERE repo = ? AND rkey = ?
	`, &sqlitex.ExecOptions{
		Args: []any{repo, rkey},
	})
}
