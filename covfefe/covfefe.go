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
	"crawshaw.io/sqlite/sqlitex"
	"github.com/dghubble/oauth1"
	"github.com/golang/groupcache/lru"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2/clientcredentials"
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
	mediaPath  string
	rescan     bool // TODO: get rid of this field
}

func Run(dbPath, mediaPath string, creds *Credentials) error {
	db, err := sqlitex.Open("file:"+dbPath, 0, 5)
	if err != nil {
		return errors.Wrap(err, "failed to open database")
	}
	defer db.Close()

	c := &Covfefe{
		withConn: func(f func(conn *sqlite.Conn) error) error {
			conn := db.Get(context.Background())
			defer db.Put(conn)
			if err := sqlitex.Exec(conn, "PRAGMA foreign_keys = ON;", nil); err != nil {
				return err
			}
			return f(conn)
		},
		msgIDs:    lru.New(1 << 16),
		mediaPath: mediaPath,
	}

	c.httpClient = (&clientcredentials.Config{
		ClientID:     creds.APIKey,
		ClientSecret: creds.APISecret,
		TokenURL:     "https://api.twitter.com/oauth2/token",
	}).Client(context.Background())
	c.httpClient.Timeout = 1 * time.Minute

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

	ctx, cancel := contextWithSignal(context.Background(), func(s os.Signal) {
		log.WithField("signal", s).Info("Received signal, stopping...")
	}, syscall.SIGINT, syscall.SIGTERM)

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
			m := &timelineMonitor{
				ctx: ctx, c: httpClient, u: user, m: messages,
			}
			err := m.followTimeline()
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

// contextWithSignal acts like context.WithCancel, but the returned Context is
// also cancelled when one of the passed signals is received. A function f, if
// not nil, is called before cancelling the Context.
func contextWithSignal(parent context.Context, f func(os.Signal), sig ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sig...)
	go func() {
		select {
		case <-ctx.Done():
		case s := <-ch:
			if f != nil {
				f(s)
			}
			cancel()
		}
		signal.Stop(ch)
	}()

	return ctx, cancel
}
