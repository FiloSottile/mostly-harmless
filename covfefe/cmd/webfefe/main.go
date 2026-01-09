package main

import (
	"context"
	"encoding/json"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"filippo.io/mostly-harmless/covfefe"
	"filippo.io/mostly-harmless/covfefe/cmd/webfefe/data"
	twitterLogin "github.com/dghubble/gologin/v2/twitter"
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

	knownUsers, err := loadKnownUsers(s.oauth1Config, creds.Accounts)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load known users")
	}
	s.knownUsers = knownUsers

	logrus.WithField("address", *listenAddr).Info("Starting...")
	logrus.WithError((&http.Server{
		Addr:         *listenAddr,
		Handler:      s.Handler(),
		ReadTimeout:  1 * time.Minute,
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

	credsMu    sync.Mutex
	credsPath  string
	knownUsers map[int64]bool

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
