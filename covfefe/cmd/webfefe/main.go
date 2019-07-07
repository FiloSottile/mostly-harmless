package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/FiloSottile/mostly-harmless/covfefe/cmd/webfefe/data"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/shurcooL/httpfs/html/vfstemplate"
	"github.com/sirupsen/logrus"
)

func main() {
	dbFile := flag.String("db", "twitter.db", "The path of the SQLite DB")
	mediaPath := flag.String("media", "twitter-media", "The folder to store media files in")
	credsFile := flag.String("creds", "creds.json", "The path of the credentials JSON")
	listenAddr := flag.String("listen", "0.0.0.0:6052", "The address to listen on for HTTP")
	flag.Parse()

	db, err := sqlitex.Open("file:"+*dbFile, 0, 5)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to open database")
	}
	defer db.Close()

	s := &Server{
		withConn: func(f func(conn *sqlite.Conn) error) error {
			conn := db.Get(context.Background())
			defer db.Put(conn)
			if err := sqlitex.Exec(conn, "PRAGMA foreign_keys = ON;", nil); err != nil {
				return err
			}
			return f(conn)
		},
		mediaPath: *mediaPath,
		credsPath: *credsFile,
		tmpl:      template.Must(vfstemplate.ParseGlob(data.Templates, nil, "*.tmpl")),
	}

	logrus.WithField("address", *listenAddr).Info("Starting...")
	logrus.WithError((&http.Server{
		Addr:         *listenAddr,
		Handler:      s.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 1 * time.Minute,
		IdleTimeout:  10 * time.Minute,
		// logrus seems to provide WriterLevel for this, but it seems to start a
		// goroutine, and set a finalizer!?
		ErrorLog: log.New(WriterFunc(func(p []byte) (n int, err error) {
			logrus.Error(string(p))
			return len(p), nil
		}), "", 0),
	}).ListenAndServe()).Error("Exiting...")
}

type WriterFunc func(p []byte) (n int, err error)

func (f WriterFunc) Write(p []byte) (n int, err error) {
	return f(p)
}

type Server struct {
	withConn  func(f func(conn *sqlite.Conn) error) error
	mediaPath string
	tmpl      *template.Template

	credsMu   sync.Mutex
	credsPath string
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.Home)
	mux.HandleFunc("/id/", s.Tweet)
	return mux
}

func (s *Server) Home(w http.ResponseWriter, r *http.Request) {
	if err := s.withConn(func(conn *sqlite.Conn) error {
		return sqlitex.Exec(conn, "SELECT COUNT(*) FROM Messages;", func(stmt *sqlite.Stmt) error {
			return s.tmpl.ExecuteTemplate(w, "home.html.tmpl", stmt.ColumnInt64(0))
		})
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) Tweet(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/id/"), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var (
		tweetJSON   []byte
		tweetSource string
	)
	if err := s.withConn(func(conn *sqlite.Conn) error {
		sql := `SELECT Messages.json, Messages.source FROM Messages, Tweets
			WHERE Messages.id = Tweets.message AND Tweets.id = ?;`
		fn := func(stmt *sqlite.Stmt) error {
			tweetJSON = []byte(stmt.ColumnText(0))
			tweetSource = stmt.ColumnText(1)
			return nil
		}
		return sqlitex.Exec(conn, sql, fn, n)
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if tweetJSON == nil {
		http.Error(w, "Tweet not found.", http.StatusNotFound)
		return
	}

	var out bytes.Buffer
	if err := json.Indent(&out, tweetJSON, "", "    "); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var tweet twitter.Tweet
	if err := json.Unmarshal(tweetJSON, &tweet); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.tmpl.ExecuteTemplate(w, "tweet_page.html.tmpl", map[string]interface{}{
		"Tweet": tweet, "Source": tweetSource, "JSON": out.String(),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
