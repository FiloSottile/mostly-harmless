package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"

	"golang.org/x/sync/errgroup"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func main() {
	cache, err := os.UserCacheDir()
	if err != nil {
		slog.Error("failed to get user cache directory", "error", err)
		return
	}
	cache = filepath.Join(cache, "sumdb")

	dbFlag := flag.String("db", "sumdb.sqlite3", "path to the SQLite database file")
	cacheFlag := flag.String("cache", cache, "path to the cache directory")
	yoloFlag := flag.Bool("yolo", false, "speed up import by reducing safety")
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

	if err := os.MkdirAll(*cacheFlag, 0o755); err != nil {
		slog.Error("failed to create cache directory", "error", err)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	group, ctx := errgroup.WithContext(ctx)

	db, err := sqlitex.NewPool(*dbFlag, sqlitex.PoolOptions{
		PrepareConn: func(db *sqlite.Conn) error {
			db.SetInterrupt(ctx.Done())
			if *yoloFlag {
				slog.Warn("yolo mode enabled, reducing database safety")
				// Optimize for initial import speed.
				sqlitex.ExecuteTransient(db, `PRAGMA journal_mode = TRUNCATE;`, nil)
				sqlitex.ExecuteTransient(db, `PRAGMA synchronous = OFF;`, nil)
				sqlitex.ExecuteTransient(db, `PRAGMA cache_size = -1000000;`, nil)
				sqlitex.ExecuteTransient(db, `PRAGMA temp_store = MEMORY;`, nil)
			}
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

	if !*yoloFlag {
		group.Go(func() error {
			err := domainr(ctx, db)
			return fmt.Errorf("domainr processing failed: %w", err)
		})
	}
	group.Go(func() error {
		err := ingest(ctx, db, *cacheFlag)
		return fmt.Errorf("ingestion failed: %w", err)
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
		CREATE TABLE IF NOT EXISTS versions (
			idx INTEGER PRIMARY KEY,
			path TEXT NOT NULL,
			version TEXT NOT NULL,
			error TEXT DEFAULT NULL
		) STRICT, WITHOUT ROWID;
		CREATE UNIQUE INDEX IF NOT EXISTS idx_versions_path_version ON versions (path, version);
		CREATE TABLE IF NOT EXISTS checkpoint (
			checkpoint TEXT NOT NULL
		) STRICT;
		CREATE TABLE IF NOT EXISTS hostnames (
			hostname TEXT NOT NULL PRIMARY KEY,
			-- etldp1 is the effective TLD+1 of the hostname
			etldp1 TEXT NOT NULL,
			domainr_status TEXT DEFAULT NULL,
			domainr_updated TEXT DEFAULT NULL,
			bad_since INTEGER DEFAULT NULL
		) STRICT;
		CREATE INDEX IF NOT EXISTS idx_hostnames_etldp1 ON hostnames (etldp1);
	`)
}
