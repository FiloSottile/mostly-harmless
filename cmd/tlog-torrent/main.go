package main

import (
	"bytes"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"filippo.io/sunlight"
	"filippo.io/sunlight/internal/torrent"
	"filippo.io/torchwood"
	"golang.org/x/mod/sumdb/tlog"
)

func main() {
	if len(os.Args) < 3 {
		slog.Error("usage: tlog-torrent <path> <monitoring prefix>")
		return
	}

	monitoringPrefix := strings.TrimSuffix(os.Args[2], "/")

	root, err := os.OpenRoot(os.Args[1])
	if err != nil {
		slog.Error("failed to open root", "error", err)
		return
	}

	checkpointBytes, err := fs.ReadFile(root.FS(), "checkpoint")
	if err != nil {
		slog.Error("failed to read checkpoint", "error", err)
		return
	}
	checkpointBytes = checkpointBytes[:bytes.Index(checkpointBytes, []byte("\n\n"))+1]
	checkpoint, err := torchwood.ParseCheckpoint(string(checkpointBytes))
	if err != nil {
		slog.Error("failed to parse checkpoint", "error", err, "checkpoint", string(checkpointBytes))
		return
	}

	domain, path, _ := strings.Cut(checkpoint.Origin, "/")
	parts := strings.Split(domain, ".")
	slices.Reverse(parts)
	parts = append(parts, strings.Split(path, "/")...)
	reverse := strings.Join(parts, ".")

	h := torrent.NewPieceHash(524288)
	t := torrent.NewWriter(os.Stdout)
	t.WriteDict(func(t *torrent.Writer) {
		t.WriteString("comment")
		t.WriteBytes(checkpointBytes)
		t.WriteString("created by")
		t.WriteString("tlog-torrent")
		t.WriteString("creation date")
		t.WriteInt64(time.Now().Unix())
		t.WriteString("info")
		t.WriteDict(func(t *torrent.Writer) {
			t.WriteString("collections")
			t.WriteList(func(t *torrent.Writer) {
				t.WriteString(reverse)
			})
			t.WriteString("files")
			t.WriteList(func(t *torrent.Writer) {
				for n := range checkpoint.N / sunlight.TileWidth {
					tile := sunlight.TilePath(tlog.Tile{
						H: sunlight.TileHeight,
						L: -1, N: n,
						W: sunlight.TileWidth,
					})
					f, err := root.Open(tile)
					if err != nil {
						slog.Error("failed to open tile", "tile", tile, "error", err)
						os.Exit(1)
					}
					length, err := io.Copy(h, f)
					if err != nil {
						slog.Error("failed to hash tile", "tile", tile, "error", err)
						os.Exit(1)
					}
					if err := f.Close(); err != nil {
						slog.Error("failed to close tile", "tile", tile, "error", err)
						os.Exit(1)
					}
					t.WriteDict(func(t *torrent.Writer) {
						t.WriteString("length")
						t.WriteInt64(length)
						t.WriteString("path")
						t.WriteList(func(t *torrent.Writer) {
							p := strings.TrimPrefix(tile, "tile/data/")
							for _, part := range strings.Split(p, "/") {
								t.WriteString(part)
							}
						})
					})
				}
			})
			t.WriteString("name")
			t.WriteString("data")
			t.WriteString("piece length")
			t.WriteInt(524288)
			t.WriteString("pieces")
			t.WriteBytes(h.Pieces())
			t.WriteString("update-url")
			t.WriteString(monitoringPrefix + "/tile-data.torrent")
		})
		t.WriteString("title")
		t.WriteString(checkpoint.Origin)
		t.WriteString("url-list")
		t.WriteList(func(t *torrent.Writer) {
			t.WriteString(monitoringPrefix + "/tile/")
		})
	})
}
