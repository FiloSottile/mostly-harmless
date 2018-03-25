package covfefe

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dghubble/go-twitter/twitter"
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
	db         *sql.DB
	wg         sync.WaitGroup
	httpClient *http.Client
	msgIDs     *lru.Cache
}

func Run(db string, creds *Credentials) error {
	var err error

	c := &Covfefe{
		httpClient: &http.Client{
			Timeout: 1 * time.Minute,
		},
		msgIDs: lru.New(1 << 16),
	}

	c.db, err = sql.Open("sqlite3", "file:"+db+"?_foreign_keys=1")
	if err != nil {
		return errors.Wrap(err, "failed to open database")
	}
	defer c.db.Close()

	c.initDB()

	log.Info("Starting...")

	messages := make(chan Message)

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
		client := twitter.NewClient(httpClient)

		user, _, err := client.Accounts.VerifyCredentials(nil)
		if err != nil {
			return errors.Wrapf(err, "invalid credetials at position %d", i)
		}

		params := &twitter.StreamUserParams{
			With:          "followings",
			StallWarnings: twitter.Bool(true),
		}
		stream, err := client.Streams.User(params)
		if err != nil {
			log.WithError(err).WithField("account", user.ScreenName).Error("Failed to open twitter stream")
		}

		streamsWG.Add(1)
		go func() {
			log.WithField("account", user.ScreenName).Info("Starting streaming")
			for m := range StreamWithContext(ctx, stream) {
				messages <- Message{account: user.ScreenName, msg: m}
			}
			if ctx.Err() == nil {
				log.WithField("account", user.ScreenName).Error("Stream terminated")
				cancel() // TODO: retry and reopen
			}
			streamsWG.Done()
		}()
	}
	streamsWG.Wait()

	close(messages)
	c.wg.Wait()
	return nil
}

type Message struct {
	account string
	msg     interface{}
}

func StreamWithContext(ctx context.Context, stream *twitter.Stream) chan interface{} {
	c := make(chan interface{})
	go func() {
	Loop:
		for {
			select {
			case m, ok := <-stream.Messages:
				if !ok {
					break Loop
				}
				select {
				case c <- m:
				case <-ctx.Done():
					break Loop
				}
			case <-ctx.Done():
				break Loop
			}
		}
		stream.Stop()
		close(c)
	}()
	return c
}
