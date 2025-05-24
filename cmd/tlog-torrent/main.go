package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"strings"
	"text/template"
	"time"

	"filippo.io/sunlight"
	"filippo.io/sunlight/internal/torrent"
	"filippo.io/torchwood"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/mod/sumdb/tlog"
)

const sliceSubtreeHeight = 26
const sliceEntries = 1 << sliceSubtreeHeight // ~67M

const torrentPieceSize = 1 << 19 // 512 KiB

func main() {
	if len(os.Args) != 2 {
		slog.Error("usage: tlog-torrent <path>")
		return
	}

	root, err := os.OpenRoot(os.Args[1])
	if err != nil {
		slog.Error("failed to open root", "error", err)
		return
	}
	root.Mkdir("torrent", 0755)

	logJSON, err := fs.ReadFile(root.FS(), "log.v3.json")
	if err != nil {
		slog.Error("failed to read log.v3.json", "error", err)
		return
	}
	var log struct {
		MonitoringPrefix string `json:"monitoring_url"`
	}
	if err := json.Unmarshal(logJSON, &log); err != nil {
		slog.Error("failed to parse log.v3.json", "error", err, "json", string(logJSON))
		return
	}

	signedCheckpoint, err := fs.ReadFile(root.FS(), "checkpoint")
	if err != nil {
		slog.Error("failed to read checkpoint", "error", err)
		return
	}
	checkpointBytes := signedCheckpoint[:bytes.Index(signedCheckpoint, []byte("\n\n"))+1]
	checkpoint, err := torchwood.ParseCheckpoint(string(checkpointBytes))
	if err != nil {
		slog.Error("failed to parse checkpoint", "error", err, "checkpoint", string(checkpointBytes))
		return
	}

	hr := tlog.TileHashReader(checkpoint.Tree, localTileReader{r: root})

	type Torrent struct {
		Title string
		GUID  string
		URL   string
	}
	var torrents []Torrent
	for slice := range checkpoint.N / sliceEntries {
		subtreeHeadIndex := tlog.StoredHashIndex(sliceSubtreeHeight, slice)
		subtreeHead, err := hr.ReadHashes([]int64{subtreeHeadIndex})
		if err != nil {
			slog.Error("failed to read subtree head", "error", err, "slice", slice)
			return
		}
		proof, err := torchwood.ProveHash(checkpoint.N, subtreeHeadIndex, hr)
		if err != nil {
			slog.Error("failed to prove hash", "error", err, "slice", slice)
			return
		}

		lo, hi := slice*sliceEntries, (slice+1)*sliceEntries
		title := fmt.Sprintf("%s entries %d to %d", checkpoint.Origin, lo, hi-1)

		name := fmt.Sprintf("torrent/%03d.torrent", slice)
		torrents = append(torrents, Torrent{
			Title: title,
			GUID:  subtreeHead[0].String(),
			URL:   log.MonitoringPrefix + name,
		})

		if _, err := root.Stat(name); err == nil {
			slog.Info("skipping existing slice", "slice", slice, "name", name)
			continue
		} else if !os.IsNotExist(err) {
			slog.Error("failed to stat slice", "error", err, "slice", slice, "name", name)
			return
		}

		slog.Info("generating slice", "slice", slice, "of", checkpoint.N/sliceEntries)

		comment := string(signedCheckpoint)
		comment += "\n"
		comment += fmt.Sprintf("%d %d %s\n", sliceSubtreeHeight, slice, subtreeHead[0])
		for _, p := range proof {
			comment += fmt.Sprintf("%s\n", p)
		}

		f, err := root.Create(name)
		if err != nil {
			slog.Error("failed to create torrent file", "error", err, "name", name)
			return
		}
		if err := makeTorrent(f, lo, hi, root, log.MonitoringPrefix, title, comment); err != nil {
			slog.Error("failed to write torrent file", "error", err, "name", name,
				"close", f.Close(), "remove", root.Remove(name))
			return
		}
		if err := f.Close(); err != nil {
			slog.Error("failed to close torrent file", "error", err, "name", name,
				"remove", root.Remove(name))
			return
		}
	}

	feed := struct {
		Name     string
		Torrents []Torrent
	}{
		Name:     checkpoint.Origin,
		Torrents: torrents,
	}
	feedFile, err := root.Create("torrent/feed.xml")
	if err != nil {
		slog.Error("failed to create feed file", "error", err)
		return
	}
	if err := feedTemplate.Execute(feedFile, feed); err != nil {
		slog.Error("failed to write feed file", "error", err,
			"close", feedFile.Close(), "remove", root.Remove("torrent/feed.xml"))
		return
	}
	if err := feedFile.Close(); err != nil {
		slog.Error("failed to close feed file", "error", err,
			"remove", root.Remove("torrent/feed.xml"))
		return
	}
}

func makeTorrent(out io.Writer, lo, hi int64, root *os.Root, monitoringPrefix, title, comment string) error {
	bar := progressbar.Default(sliceEntries)
	h := torrent.NewPieceHash(torrentPieceSize)
	t := torrent.NewWriter(out)
	t.WriteDict(func(t *torrent.Writer) {
		t.WriteString("comment")
		t.WriteString(comment)
		t.WriteString("created by")
		t.WriteString("tlog-torrent")
		t.WriteString("creation date")
		t.WriteInt64(time.Now().Unix())
		t.WriteString("info")
		t.WriteDict(func(t *torrent.Writer) {
			t.WriteString("files")
			t.WriteList(func(t *torrent.Writer) {
				for n := lo / sunlight.TileWidth; n < hi/sunlight.TileWidth; n++ {
					tile := sunlight.TilePath(tlog.Tile{
						H: sunlight.TileHeight,
						L: -1, N: n,
						W: sunlight.TileWidth,
					})
					f, err := root.Open(tile)
					if err != nil {
						t.SetError(fmt.Errorf("failed to open tile %s: %v", tile, err))
						return
					}
					length, err := io.Copy(h, f)
					if err != nil {
						t.SetError(fmt.Errorf("failed to read tile %s: %v", tile, err))
						return
					}
					if err := f.Close(); err != nil {
						t.SetError(fmt.Errorf("failed to close tile %s: %v", tile, err))
						return
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
					bar.Add(sunlight.TileWidth)
				}
			})
			t.WriteString("name")
			// We have to call the torrent "data" because the name gets concatenated
			// to the webseed prefix, so it has to appear in the URL.
			t.WriteString("data")
			t.WriteString("piece length")
			t.WriteInt(torrentPieceSize)
			t.WriteString("pieces")
			t.WriteBytes(h.Pieces())
		})
		t.WriteString("title")
		t.WriteString(title)
		t.WriteString("url-list")
		t.WriteList(func(t *torrent.Writer) {
			t.WriteString(monitoringPrefix + "tile/")
		})
	})
	bar.Finish()
	return t.Err()
}

var feedTemplate = template.Must(template.New("feed").Parse(`<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
    <channel>
        <title>{{ .Name }}</title>
        {{ range .Torrents }}
        <item>
            <title>{{ .Title }}</title>
            <guid>{{ .GUID }}</guid>
            <enclosure type="application/x-bittorrent" url="{{ .URL }}"/>
        </item>
        {{ end }}
    </channel>
</rss>
`))

type localTileReader struct {
	r *os.Root
}

func (r localTileReader) Height() int {
	return sunlight.TileHeight
}

func (r localTileReader) ReadTiles(tiles []tlog.Tile) (data [][]byte, err error) {
	data = make([][]byte, len(tiles))
	for i, tile := range tiles {
		path := sunlight.TilePath(tile)
		b, err := fs.ReadFile(r.r.FS(), path)
		if os.IsNotExist(err) && tile.W != sunlight.TileWidth && tile.L != -1 {
			// Retry the full tile.
			full := tile
			full.W = sunlight.TileWidth
			b, err = fs.ReadFile(r.r.FS(), sunlight.TilePath(full))
			if err == nil {
				b = b[:tlog.HashSize*tile.W]
			}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tile %s: %v", path, err)
		}
		if tile.L != -1 && len(b) != tlog.HashSize*tile.W {
			return nil, fmt.Errorf("tile %s has wrong length: got %d, want %d",
				path, len(b), tlog.HashSize*tile.W)
		}
		data[i] = b
	}
	return data, nil
}

func (r localTileReader) SaveTiles(tiles []tlog.Tile, data [][]byte) {}
