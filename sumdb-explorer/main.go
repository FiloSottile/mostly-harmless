package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/torchwood"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/mod/module"
	"golang.org/x/mod/sumdb/note"
	"golang.org/x/mod/sumdb/tlog"
	"golang.org/x/net/publicsuffix"
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
	flag.Parse()

	if err := os.MkdirAll(*cacheFlag, 0o755); err != nil {
		slog.Error("failed to create cache directory", "error", err)
		return
	}

	db, err := sqlite.OpenConn(*dbFlag)
	if err != nil {
		slog.Error("failed to open SQLite database", "error", err)
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close SQLite database", "error", err)
		}
	}()
	if err := sqlitex.ExecScript(db, `
		CREATE TABLE IF NOT EXISTS versions (
			idx INTEGER PRIMARY KEY,
			path TEXT NOT NULL,
			version TEXT NOT NULL,
			error TEXT DEFAULT NULL
		) WITHOUT ROWID;
		CREATE UNIQUE INDEX IF NOT EXISTS idx_versions_path_version ON versions (path, version);
		CREATE TABLE IF NOT EXISTS checkpoint (
			checkpoint TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS hostnames (
			hostname TEXT NOT NULL PRIMARY KEY,
			-- etldp1 is the effective TLD+1 of the hostname
			etldp1 TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_hostnames_etldp1 ON hostnames (etldp1);
	`); err != nil {
		slog.Error("failed to create database schema", "error", err)
		return
	}
	sqlitex.ExecuteTransient(db, `PRAGMA foreign_keys = ON;`, nil)
	// Optimize for initial import speed.
	sqlitex.ExecuteTransient(db, `PRAGMA journal_mode = TRUNCATE;`, nil)
	sqlitex.ExecuteTransient(db, `PRAGMA synchronous = OFF;`, nil)
	sqlitex.ExecuteTransient(db, `PRAGMA cache_size = -1000000;`, nil)
	sqlitex.ExecuteTransient(db, `PRAGMA temp_store = MEMORY;`, nil)

	fetcher, err := torchwood.NewTileFetcher("https://sum.golang.org/",
		torchwood.WithTilePath(tlog.Tile.Path))
	if err != nil {
		slog.Error("failed to create tile fetcher", "error", err)
		return
	}
	dirCache, err := torchwood.NewPermanentCache(fetcher, *cacheFlag,
		torchwood.WithPermanentCacheTilePath(tlog.Tile.Path))
	if err != nil {
		slog.Error("failed to create permanent cache", "error", err)
		return
	}
	client, err := torchwood.NewClient(dirCache, torchwood.WithSumDBEntries())
	if err != nil {
		slog.Error("failed to create client", "error", err)
		return
	}

	ctx := context.Background()
	checkpoint, err := fetchCheckpoint(ctx, fetcher)
	if err != nil {
		slog.Error("failed to fetch checkpoint", "error", err)
		return
	}
	hr := torchwood.TileHashReaderWithContext(ctx, checkpoint.Tree, fetcher)
	if err := updateCheckpoint(db, checkpoint, hr); err != nil {
		slog.Error("failed to update checkpoint", "error", err)
		return
	}

	insertVersion := db.Prep(`INSERT INTO versions (idx, path, version, error) VALUES (:idx, :path, :version, :error)`)
	insertHostname := db.Prep(`INSERT OR IGNORE INTO hostnames (hostname, etldp1) VALUES (:hostname, :etldp1)`)

	start, err := dbSize(db)
	if err != nil {
		slog.Error("failed to get database size", "error", err)
		return
	}
	pb := progressbar.Default(checkpoint.N)
	pb.Set64(start)
	release := sqlitex.Save(db)
	defer func() {
		var err error
		release(&err)
	}()
	for i, entry := range client.Entries(ctx, checkpoint.Tree, start) {
		if i%10000 == 0 {
			var err error
			release(&err)
			release = sqlitex.Save(db)
		}
		v, err := parseEntry(string(entry))
		if err != nil {
			slog.Error("failed to parse entry", "entry", entry, "index", i, "error", err)
			return
		}
		checkErr := module.Check(v.Path, v.Version)

		hostname, _, _ := strings.Cut(v.Path, "/")
		etldp1, err := publicsuffix.EffectiveTLDPlusOne(hostname)
		if err != nil {
			suffix, _ := publicsuffix.PublicSuffix(hostname)
			if suffix == hostname {
				etldp1 = hostname
			} else {
				slog.Error("failed to get effective TLD+1", "hostname", hostname, "error", err)
				return
			}
		}

		if err := insertHostname.Reset(); err != nil {
			slog.Error("failed to reset prepared statement", "error", err)
			return
		}
		insertHostname.SetText(":hostname", hostname)
		insertHostname.SetText(":etldp1", etldp1)
		if _, err := insertHostname.Step(); err != nil {
			slog.Error("failed to insert hostname into database", "hostname", hostname, "error", err)
			return
		}

		if err := insertVersion.Reset(); err != nil {
			slog.Error("failed to reset prepared statement", "error", err)
			return
		}
		insertVersion.SetInt64(":idx", i)
		insertVersion.SetText(":path", v.Path)
		insertVersion.SetText(":version", v.Version)
		if checkErr != nil {
			slog.Warn("invalid module version", "module", v, "error", checkErr, "index", i)
			insertVersion.SetText(":error", checkErr.Error())
		} else {
			insertVersion.SetNull(":error")
		}
		if _, err := insertVersion.Step(); err != nil {
			slog.Error("failed to insert entry into database", "index", i, "error", err, "module", v)
			return
		}

		pb.Add(1)
	}
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

func updateCheckpoint(db *sqlite.Conn, checkpoint torchwood.Checkpoint, hr tlog.HashReader) error {
	var old string
	if err := sqlitex.ExecuteTransient(db, `SELECT checkpoint FROM checkpoint`, &sqlitex.ExecOptions{
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
		// We simply fetch the expected hash instead of doing ProveTree and
		// CheckTree because the TileHashReader only returns hashes that are
		// already proven to be part of the new tree.
		expectedHash, err := tlog.TreeHash(oc.N, hr)
		if err != nil {
			return err
		}
		if expectedHash != oc.Hash {
			return errors.New("checkpoint hash mismatch")
		}
	} else {
		slog.Warn("no previous checkpoint found, skipping verification")
	}
	stmt := db.Prep(`INSERT OR REPLACE INTO checkpoint (checkpoint) VALUES (:checkpoint)`)
	if err := stmt.Reset(); err != nil {
		return err
	}
	stmt.SetText(":checkpoint", checkpoint.String())
	if _, err := stmt.Step(); err != nil {
		return err
	}
	return nil
}
