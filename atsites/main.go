package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"filippo.io/mostly-harmless/atsites/internal/db"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	indigoutil "github.com/bluesky-social/indigo/util"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/gorilla/feeds"
	"golang.org/x/sync/errgroup"
	_ "modernc.org/sqlite"
)

//go:embed sql/schema.sql
var schemaSQL string

var lastDocumentSeen atomic.Pointer[time.Time]

func main() {
	dbFlag := flag.String("db", "atsites.sqlite3", "path to the SQLite database file")
	tapFlag := flag.String("tap", "ws://localhost:2480/channel", "Tap WebSocket URL")
	listenFlag := flag.String("listen", ":8000", "address to listen on for HTTP server")
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

	sqlDB, err := sql.Open("sqlite", *dbFlag+"?_pragma=foreign_keys(1)")
	if err != nil {
		slog.Error("failed to open SQLite database", "error", err)
		return
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			slog.Error("failed to close SQLite database", "error", err)
		}
	}()
	if _, err := sqlDB.ExecContext(ctx, schemaSQL); err != nil {
		slog.Error("failed to initialize database", "error", err)
		return
	}

	s := &Server{db: sqlDB, queries: db.New(sqlDB)}
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
	hs := &http.Server{
		Addr:        *listenFlag,
		Handler:     s.httpHandler(),
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}
	group.Go(func() error {
		slog.Info("starting HTTP server", "addr", *listenFlag)
		return hs.ListenAndServe()
	})
	group.Go(func() error {
		<-ctx.Done()
		slog.Info("shutting down")
		hs.Shutdown(context.Background())
		return nil
	})

	slog.Error("stopping", "error", group.Wait())
}

type Server struct {
	db      *sql.DB
	queries *db.Queries
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
	switch rec.Collection {
	case "site.standard.publication":
		if rec.Action == "delete" {
			slog.DebugContext(ctx, "deleting publication", "repo", rec.Repo, "rkey", rec.Rkey)
			return s.queries.DeletePublication(ctx, db.DeletePublicationParams{
				Repo: rec.Repo,
				Rkey: rec.Rkey,
			})
		}
		slog.DebugContext(ctx, "storing publication", "repo", rec.Repo, "rkey", rec.Rkey)
		return s.queries.StorePublication(ctx, db.StorePublicationParams{
			Repo:       rec.Repo,
			Rkey:       rec.Rkey,
			RecordJson: rec.Record,
		})
	case "site.standard.document":
		if rec.Action == "delete" {
			slog.DebugContext(ctx, "deleting document", "repo", rec.Repo, "rkey", rec.Rkey)
			return s.queries.DeleteDocument(ctx, db.DeleteDocumentParams{
				Repo: rec.Repo,
				Rkey: rec.Rkey,
			})
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
		lastDocumentSeen.Store(new(time.Now()))
		slog.DebugContext(ctx, "storing document", "repo", rec.Repo, "rkey", rec.Rkey,
			"publication_repo", u.Did, "publication_rkey", u.Rkey)
		return s.queries.StoreDocument(ctx, db.StoreDocumentParams{
			Repo:            rec.Repo,
			Rkey:            rec.Rkey,
			PublicationRepo: u.Did,
			PublicationRkey: u.Rkey,
			DocumentJson:    rec.Record,
		})
	default:
		slog.DebugContext(ctx, "ignoring record from unknown collection",
			"uri", fmt.Sprintf("at://%s/%s/%s", rec.Repo, rec.Collection, rec.Rkey))
	}

	return nil
}

func (s *Server) httpHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if t := lastDocumentSeen.Load(); time.Since(*t) > 24*time.Hour {
			http.Error(w, "no documents seen in the last 24 hours", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /profile/{handle}", s.handleProfile)
	mux.HandleFunc("GET /profile/{did}/publication/{rkey}", s.handlePublication)
	mux.HandleFunc("GET /profile/{did}/publication/{rkey}/atom.xml", s.handleFeed)
	return mux
}

//go:embed templates
var templates embed.FS

var profileTemplate = template.Must(template.New("profile.html").ParseFS(templates, "templates/profile.html"))

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	id, err := syntax.ParseAtIdentifier(r.PathValue("handle"))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid AT identifier: %v", err), http.StatusBadRequest)
		return
	}
	i, err := (&identity.BaseDirectory{}).Lookup(r.Context(), *id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resolve handle: %v", err), http.StatusInternalServerError)
		return
	}

	publications, err := s.getPublications(r.Context(), i.DID.String())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch publications: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := profileTemplate.Execute(w, struct {
		Handle       string
		DID          string
		Publications []*Publication
	}{
		Handle:       i.Handle.String(),
		DID:          i.DID.String(),
		Publications: publications,
	}); err != nil {
		slog.ErrorContext(r.Context(), "execute profile template", "error", err)
	}
}

func (s *Server) getPublications(ctx context.Context, did string) ([]*Publication, error) {
	rows, err := s.queries.GetPublications(ctx, did)
	if err != nil {
		return nil, err
	}

	publications := make([]*Publication, 0, len(rows))
	for _, row := range rows {
		publications = append(publications, parsePublication(did, row.Rkey, row.RecordJson))
	}
	return publications, nil
}

type Publication struct {
	Repo        string `json:"-"`
	Rkey        string `json:"-"`
	Invalid     bool   `json:"-"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func parsePublication(repo, rkey string, record json.RawMessage) *Publication {
	pub := &Publication{Repo: repo, Rkey: rkey}
	if err := json.Unmarshal(record, &pub); err != nil {
		pub.Invalid = true
	}
	return pub
}

var publicationTemplate = template.Must(template.New("publication.html").ParseFS(templates, "templates/publication.html"))

func (s *Server) handlePublication(w http.ResponseWriter, r *http.Request) {
	repo := r.PathValue("did")
	rkey := r.PathValue("rkey")

	did, err := syntax.ParseDID(repo)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid DID: %v", err), http.StatusBadRequest)
		return
	}
	i, err := (&identity.BaseDirectory{}).LookupDID(r.Context(), did)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resolve DID: %v", err), http.StatusInternalServerError)
		return
	}

	publication, err := s.getPublication(r.Context(), repo, rkey)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch publication: %v", err), http.StatusInternalServerError)
		return
	}
	if publication == nil {
		http.Error(w, "publication not found", http.StatusNotFound)
		return
	}

	documents, err := s.getDocumentsForPublication(r.Context(), repo, rkey)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch documents: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := publicationTemplate.Execute(w, struct {
		Handle      string
		DID         string
		Publication *Publication
		Documents   []*Document
	}{
		Handle:      i.Handle.String(),
		Publication: publication,
		Documents:   documents,
	}); err != nil {
		slog.ErrorContext(r.Context(), "execute publication template", "error", err)
	}
}

type Document struct {
	Repo        string `json:"-"`
	Rkey        string `json:"-"`
	Invalid     bool   `json:"-"`
	Path        string `json:"path"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PublishedAt string `json:"publishedAt"`
	UpdatedAt   string `json:"updatedAt"`
}

func (s *Server) getPublication(ctx context.Context, repo, rkey string) (*Publication, error) {
	record, err := s.queries.GetPublication(ctx, db.GetPublicationParams{
		Repo: repo,
		Rkey: rkey,
	})
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parsePublication(repo, rkey, record), nil
}

func (s *Server) getDocumentsForPublication(ctx context.Context, pubRepo, pubRkey string) ([]*Document, error) {
	rows, err := s.queries.GetDocumentsForPublication(ctx, db.GetDocumentsForPublicationParams{
		PublicationRepo: pubRepo,
		PublicationRkey: pubRkey,
	})
	if err != nil {
		return nil, err
	}

	documents := make([]*Document, 0, len(rows))
	for _, row := range rows {
		documents = append(documents, parseDocument(pubRepo, row.Rkey, row.DocumentJson))
	}

	slices.SortStableFunc(documents, func(a, b *Document) int {
		if a.PublishedAt == "" && b.PublishedAt == "" {
			return 0
		}
		if a.PublishedAt == "" {
			return 1
		}
		if b.PublishedAt == "" {
			return -1
		}
		return strings.Compare(b.PublishedAt, a.PublishedAt)
	})

	return documents, nil
}

func parseDocument(repo, rkey string, record json.RawMessage) *Document {
	doc := &Document{Repo: repo, Rkey: rkey}
	if err := json.Unmarshal(record, &doc); err != nil {
		doc.Invalid = true
	}
	return doc
}

func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	repo := r.PathValue("did")
	rkey := r.PathValue("rkey")

	did, err := syntax.ParseDID(repo)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid DID: %v", err), http.StatusBadRequest)
		return
	}
	i, err := (&identity.BaseDirectory{}).LookupDID(r.Context(), did)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resolve DID: %v", err), http.StatusInternalServerError)
		return
	}

	publication, err := s.getPublication(r.Context(), repo, rkey)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch publication: %v", err), http.StatusInternalServerError)
		return
	}
	if publication == nil {
		http.Error(w, "publication not found", http.StatusNotFound)
		return
	}

	documents, err := s.getDocumentsForPublication(r.Context(), repo, rkey)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch documents: %v", err), http.StatusInternalServerError)
		return
	}

	f := &feeds.Feed{
		Title:       publication.Name + " by " + i.Handle.String(),
		Link:        &feeds.Link{Href: publication.URL},
		Description: publication.Description,
		Author:      &feeds.Author{Name: i.Handle.String()},
	}
	for _, doc := range documents {
		item := &feeds.Item{
			Id:          doc.Rkey,
			IsPermaLink: "false",
			Title:       doc.Title,
			Content:     doc.Description,
			Link:        &feeds.Link{Href: fmt.Sprintf("%s/%s", publication.URL, doc.Path)},
			Author:      &feeds.Author{Name: i.Handle.String()},
		}
		t, err := syntax.ParseDatetimeLenient(doc.PublishedAt)
		if err == nil {
			item.Created = t.Time()
		}
		u, err := syntax.ParseDatetimeLenient(doc.UpdatedAt)
		if err == nil {
			item.Updated = u.Time()
		}
		f.Add(item)
	}
	atom, err := f.ToAtom()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate Atom feed: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	w.Write([]byte(atom))

}
