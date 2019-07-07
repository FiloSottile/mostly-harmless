package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/FiloSottile/mostly-harmless/covfefe"
	"github.com/FiloSottile/mostly-harmless/covfefe/cmd/webfefe/data"
	"github.com/dghubble/go-twitter/twitter"
	oauth1Login "github.com/dghubble/gologin/oauth1"
	twitterLogin "github.com/dghubble/gologin/twitter"
	"github.com/dghubble/oauth1"
	twitterOAuth1 "github.com/dghubble/oauth1/twitter"
	"github.com/gorilla/sessions"
	"github.com/shurcooL/httpfs/html/vfstemplate"
	"github.com/sirupsen/logrus"
)

func main() {
	dbFile := flag.String("db", "twitter.db", "The path of the SQLite DB")
	mediaPath := flag.String("media", "twitter-media", "The folder to store media files in")
	credsFile := flag.String("creds", "creds.json", "The path of the credentials JSON")
	listenAddr := flag.String("listen", "127.0.0.1:6052", "The address to listen on for HTTP")
	baseAddr := flag.String("base", "http://localhost:6052", "The base address at which we run")
	flag.Parse()

	credsJSON, err := ioutil.ReadFile(*credsFile)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to read credentials file")
	}
	creds := &covfefe.Credentials{}
	if err := json.Unmarshal(credsJSON, creds); err != nil {
		logrus.WithError(err).Fatal("Failed to parse credentials file")
	}

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
		store:     sessions.NewCookieStore([]byte(creds.APISecret)),
		oauth1Config: &oauth1.Config{
			ConsumerKey:    creds.APIKey,
			ConsumerSecret: creds.APISecret,
			CallbackURL:    *baseAddr + "/callback",
			Endpoint:       twitterOAuth1.AuthorizeEndpoint,
		},
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

	store        sessions.Store
	oauth1Config *oauth1.Config
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.loggedIn(s.Home))
	mux.HandleFunc("/id/", s.loggedIn(s.Tweet))
	mux.Handle("/login", twitterLogin.LoginHandler(s.oauth1Config, nil))
	mux.Handle("/callback", twitterLogin.CallbackHandler(s.oauth1Config, http.HandlerFunc(s.Login), nil))
	return mux
}

func (s *Server) loggedIn(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := s.store.Get(r, "webfefe")
		if loggedIn, ok := session.Values["logged-in"].(bool); ok && loggedIn {
			fn(w, r)
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	twitterUser, err := twitterLogin.UserFromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	accessToken, accessSecret, err := oauth1Login.AccessTokenFromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = accessToken, accessSecret
	logrus.WithFields(logrus.Fields{
		"name": twitterUser.ScreenName, "ID": twitterUser.ID,
	}).Info("User logged in")
	session, _ := s.store.Get(r, "webfefe")
	session.Values["logged-in"] = true
	session.Values["twitter-user"] = twitterUser.ID
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
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
