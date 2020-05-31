// Package covfefe is a mystery.
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
	"golang.org/x/sync/errgroup"
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
	rescan     bool
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

	ctx, _ := contextWithSignal(context.Background(), func(s os.Signal) {
		log.WithField("signal", s).Info("Received signal, stopping...")
	}, syscall.SIGINT, syscall.SIGTERM)

	g, ctx := errgroup.WithContext(ctx)
	config := oauth1.NewConfig(creds.APIKey, creds.APISecret)
	for i, account := range creds.Accounts {
		token := oauth1.NewToken(account.Token, account.TokenSecret)
		httpClient := config.Client(oauth1.NoContext, token)
		httpClient.Timeout = 10 * time.Second

		user, err := verifyCredentials(ctx, httpClient)
		if err != nil {
			return errors.Wrapf(err, "invalid credentials at position %d", i)
		}

		log := log.WithFields(log.Fields{
			"account": user.ScreenName, "id": user.ID,
		})

		for _, timeline := range []string{"home", "mentions", "user", "likes"} {
			timeline := timeline
			g.Go(func() error {
				log.WithField("timeline", timeline).Info("Starting to monitor timeline")
				m := &twitterClient{c: httpClient, u: user, m: messages}
				return errors.Wrapf(m.followTimeline(ctx, timeline),
					"%s of %d", timeline, user.ID)
			})
		}

		g.Go(func() error {
			log.Info("Starting to fetch followers")
			m := &twitterClient{c: httpClient, u: user, m: messages}
			for {
				if err := m.fetchFollowers(ctx, user.ID); err != nil {
					return errors.Wrapf(err, "followers of %d", user.ID)
				}
				log.Debug("Starting over fetching followers")
				time.Sleep(24 * time.Hour)
			}
		})
	}
	log.WithError(g.Wait()).Error("Stopped following timelines")

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
