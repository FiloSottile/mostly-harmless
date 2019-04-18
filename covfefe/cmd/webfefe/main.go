package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dghubble/go-twitter/twitter"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/sirupsen/logrus"
)

func main() {
	dbFile := flag.String("db", "twitter.db", "The path of the SQLite DB")
	mediaPath := flag.String("media", "twitter-media", "The folder to store media files in")
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
	}

	logrus.Info("Starting...")
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
			return tmplHome.Execute(w, stmt.ColumnInt64(0))
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

	if err := s.withConn(func(conn *sqlite.Conn) error {
		sql := `SELECT Messages.json FROM Messages, Tweets
			WHERE Messages.id = Tweets.message AND Tweets.id = ?;`
		return sqlitex.Exec(conn, sql, func(stmt *sqlite.Stmt) error {
			var out bytes.Buffer
			if err := json.Indent(&out, []byte(stmt.ColumnText(0)), "", "    "); err != nil {
				return err
			}

			var tweet twitter.Tweet
			if err := json.Unmarshal([]byte(stmt.ColumnText(0)), &tweet); err != nil {
				return err
			}

			if err := tmplTweet.Execute(w, tweet); err != nil {
				return err
			}
			fmt.Fprintf(w, "<pre><code>%s</code></pre>", html.EscapeString(out.String()))
			return nil
		}, n)
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
