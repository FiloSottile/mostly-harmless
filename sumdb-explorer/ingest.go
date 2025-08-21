package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"filippo.io/torchwood"
	"golang.org/x/mod/module"
	"golang.org/x/mod/sumdb/note"
	"golang.org/x/mod/sumdb/tlog"
	"golang.org/x/net/publicsuffix"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func ingest(ctx context.Context, pool *sqlitex.Pool, cachePath string) error {
	db, err := pool.Take(ctx)
	if err != nil {
		return fmt.Errorf("failed to take database connection: %w", err)
	}
	defer pool.Put(db)

	fetcher, err := torchwood.NewTileFetcher("https://sum.golang.org/",
		torchwood.WithTilePath(tlog.Tile.Path))
	if err != nil {
		return fmt.Errorf("failed to create tile fetcher: %w", err)
	}
	dirCache, err := torchwood.NewPermanentCache(fetcher, cachePath,
		torchwood.WithPermanentCacheTilePath(tlog.Tile.Path))
	if err != nil {
		return fmt.Errorf("failed to create permanent cache: %w", err)
	}
	client, err := torchwood.NewClient(dirCache, torchwood.WithSumDBEntries())
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	start, err := dbSize(db)
	if err != nil {
		return fmt.Errorf("failed to get database size: %w", err)
	}

	ticker := time.NewTicker(1 * time.Minute)
	for {
		checkpoint, err := fetchCheckpoint(ctx, fetcher)
		if err != nil {
			return fmt.Errorf("failed to fetch checkpoint: %w", err)
		}
		if err := updateCheckpoint(ctx, db, checkpoint, fetcher); err != nil {
			return fmt.Errorf("failed to update checkpoint to %v: %w", checkpoint, err)
		}

		if checkpoint.N <= start {
			slog.Debug("no new entries to ingest", "start", start, "checkpoint", checkpoint.N)
		} else {
			if err := processEntries(ctx, client, db, checkpoint, start); err != nil {
				return fmt.Errorf("failed to ingest entries: %w", err)
			}
			slog.Debug("ingested entries", "start", start, "end", checkpoint.N)
			start = checkpoint.N
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func processEntries(ctx context.Context, client *torchwood.Client, db *sqlite.Conn,
	checkpoint torchwood.Checkpoint, start int64) (err error) {
	release, err := sqlitex.ImmediateTransaction(db)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer release(&err)

	for i, entry := range client.Entries(ctx, checkpoint.Tree, start) {
		v, err := parseEntry(string(entry))
		if err != nil {
			return fmt.Errorf("failed to parse entry %d: %w", i, err)
		}
		checkErr := module.Check(v.Path, v.Version)

		hostname, _, _ := strings.Cut(v.Path, "/")
		etldp1, err := publicsuffix.EffectiveTLDPlusOne(hostname)
		if err != nil {
			suffix, _ := publicsuffix.PublicSuffix(hostname)
			if suffix == hostname {
				etldp1 = hostname
			} else {
				return fmt.Errorf("failed to derive eTLD+1 for %q at index %d: %w", hostname, i, err)
			}
		}

		if err := sqlitex.Execute(db, `
			INSERT OR IGNORE INTO hostnames (hostname, etldp1)
			VALUES (:hostname, :etldp1)
		`, &sqlitex.ExecOptions{
			Named: map[string]any{
				":hostname": hostname,
				":etldp1":   etldp1,
			},
		}); err != nil {
			return fmt.Errorf("failed to insert hostname %q into database: %w", hostname, err)
		}

		if checkErr != nil {
			return sqlitex.Execute(db, `
				INSERT INTO versions (idx, path, version, error)
				VALUES (:idx, :path, :version, :error)
			`, &sqlitex.ExecOptions{
				Named: map[string]any{
					":idx":     i,
					":path":    v.Path,
					":version": v.Version,
					":error":   checkErr.Error(),
				},
			})
		}
		return sqlitex.Execute(db, `
			INSERT INTO versions (idx, path, version)
			VALUES (:idx, :path, :version)
		`, &sqlitex.ExecOptions{
			Named: map[string]any{
				":idx":     i,
				":path":    v.Path,
				":version": v.Version,
			},
		})
	}
	return nil
}

func parseEntry(entry string) (module.Version, error) {
	name, rest, ok := strings.Cut(string(entry), " ")
	if !ok {
		return module.Version{}, errors.New("invalid entry format")
	}
	version, rest, ok := strings.Cut(rest, " ")
	if !ok {
		return module.Version{}, errors.New("invalid entry format")
	}
	v := module.Version{Path: name, Version: version}
	if module.CanonicalVersion(version) != version {
		return v, module.VersionError(v, errors.New("version is not canonical"))
	}
	_, rest, ok = strings.Cut(rest, "\n")
	if !ok {
		return v, module.VersionError(v, errors.New("invalid entry format"))
	}
	name1, rest, ok := strings.Cut(rest, " ")
	if !ok || name1 != name {
		return v, module.VersionError(v, errors.New("invalid entry format"))
	}
	version1, rest, ok := strings.Cut(rest, " ")
	if !ok || version1 != version+"/go.mod" {
		return v, module.VersionError(v, errors.New("go.mod version mismatch"))
	}
	_, rest, ok = strings.Cut(rest, "\n")
	if !ok || rest != "" {
		return v, module.VersionError(v, errors.New("invalid entry format"))
	}
	return v, nil
}

func dbSize(db *sqlite.Conn) (int64, error) {
	var index int64 = -1
	if err := sqlitex.ExecuteTransient(db, `SELECT MAX(idx) FROM versions`, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			index = stmt.ColumnInt64(0)
			return nil
		},
	}); err != nil {
		return 0, err
	}
	if index == -1 {
		slog.Warn("no entries found in the database, starting from index 0")
	}
	return index + 1, nil
}

func fetchCheckpoint(ctx context.Context, fetcher *torchwood.TileFetcher) (torchwood.Checkpoint, error) {
	signed, err := fetcher.ReadEndpoint(ctx, "latest")
	if err != nil {
		return torchwood.Checkpoint{}, err
	}
	v, err := note.NewVerifier("sum.golang.org+033de0ae+Ac4zctda0e5eza+HJyk9SxEdh+s3Ux18htTTAD8OuAn8")
	if err != nil {
		return torchwood.Checkpoint{}, err
	}
	n, err := note.Open(signed, note.VerifierList(v))
	if err != nil {
		return torchwood.Checkpoint{}, err
	}
	return torchwood.ParseCheckpoint(n.Text)
}

func updateCheckpoint(ctx context.Context, db *sqlite.Conn, checkpoint torchwood.Checkpoint, fetcher torchwood.TileReaderWithContext) (err error) {
	release, err := sqlitex.ImmediateTransaction(db)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer release(&err)
	var old string
	if err := sqlitex.Execute(db, `
		SELECT checkpoint FROM checkpoint
	`, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			old = stmt.ColumnText(0)
			return nil
		},
	}); err != nil {
		return err
	}
	if old != "" {
		oc, err := torchwood.ParseCheckpoint(old)
		if err != nil {
			return err
		}
		if oc.N > checkpoint.N {
			slog.Warn("new checkpoint is older than the current one", "current", oc.N, "new", checkpoint.N)
			return checkConsistency(ctx, checkpoint.Tree, oc.Tree, fetcher)
		}
		if err := checkConsistency(ctx, oc.Tree, checkpoint.Tree, fetcher); err != nil {
			return err
		}
	} else {
		slog.Warn("no previous checkpoint found, skipping verification")
	}
	return sqlitex.Execute(db, `
		INSERT OR REPLACE INTO checkpoint (checkpoint) VALUES (:checkpoint)
	`, &sqlitex.ExecOptions{
		Named: map[string]any{
			":checkpoint": checkpoint.String(),
		},
	})
}

func checkConsistency(ctx context.Context, old, new tlog.Tree, fetcher torchwood.TileReaderWithContext) error {
	// We simply fetch the expected hash instead of doing ProveTree and
	// CheckTree because the TileHashReader only returns hashes that are
	// already proven to be part of the new tree.
	hr := torchwood.TileHashReaderWithContext(ctx, new, fetcher)
	expectedHash, err := tlog.TreeHash(old.N, hr)
	if err != nil {
		return fmt.Errorf("failed to compute old tree hash at size %d: %w", old.N, err)
	}
	if expectedHash != old.Hash {
		return fmt.Errorf("old tree hash mismatch: expected %s, got %s", old.Hash, expectedHash)
	}
	return nil
}
