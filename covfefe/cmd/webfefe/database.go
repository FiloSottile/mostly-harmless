package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/dghubble/go-twitter/twitter"
)

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
