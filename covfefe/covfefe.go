package covfefe

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqliteutil"
	"github.com/dghubble/oauth1"
	"github.com/golang/groupcache/lru"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Credentials struct {
	APIKey    string `json:"API_KEY"`
	APISecret string `json:"API_SECRET"`
	Accounts  []Account
}

type Account struct {
	Token       string `json:"TOKEN"`
	TokenSecret string `json:"TOKEN_SECRET"`
}

type Covfefe struct {
	withConn   func(f func(conn *sqlite.Conn) error) error
	wg         sync.WaitGroup
	httpClient *http.Client
	msgIDs     *lru.Cache
	rescan     bool // TODO: get rid of this field
}

func Run(dbPath string, creds *Credentials) error {
	db, err := sqlite.Open("file:"+dbPath, 0, 5)
	if err != nil {
		return errors.Wrap(err, "failed to open database")
	}
	defer db.Close()

	c := &Covfefe{
		withConn: func(f func(conn *sqlite.Conn) error) error {
			conn := db.Get(nil)
			defer db.Put(conn)
			if err := sqliteutil.Exec(conn, "PRAGMA foreign_keys = ON;", nil); err != nil {
				return err
			}
			return f(conn)
		},
		httpClient: &http.Client{
			Timeout: 1 * time.Minute,
		},
		msgIDs: lru.New(1 << 16),
	}

	if err := c.initDB(); err != nil {
		return err
	}

	log.Info("Starting...")

	messages := make(chan *Message)

	c.wg.Add(1)
	go func() {
		c.HandleChan(messages)
		c.wg.Done()
	}()

	ctx, cancel := context.WithCancel(context.Background())

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		log.WithField("signal", <-ch).Info("Received signal, stopping...")
		cancel()
	}()

	var streamsWG sync.WaitGroup
	config := oauth1.NewConfig(creds.APIKey, creds.APISecret)
	for i, account := range creds.Accounts {
		token := oauth1.NewToken(account.Token, account.TokenSecret)
		httpClient := config.Client(oauth1.NoContext, token)
		httpClient.Timeout = 10 * time.Second

		user, err := verifyCredentials(ctx, httpClient)
		if err != nil {
			return errors.Wrapf(err, "invalid credetials at position %d", i)
		}

		streamsWG.Add(1)
		go func() {
			log.WithField("account", user.ScreenName).WithField("id", user.ID).Info(
				"Starting to monitor timeline")
			err := followTimeline(ctx, httpClient, user, messages)
			log.WithField("account", user.ScreenName).WithError(err).Error(
				"Stopped following timeline") // TODO: retry
			cancel()
			streamsWG.Done()
		}()
	}
	streamsWG.Wait()

	close(messages)
	c.wg.Wait()
	return nil
}
