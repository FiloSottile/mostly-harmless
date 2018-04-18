package main

import (
	"context"
	"flag"
	"log/syslog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"crawshaw.io/sqlite"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	lsyslog "github.com/sirupsen/logrus/hooks/syslog"
	"golang.org/x/sync/errgroup"
	// TODO: prometheus
)

type DocumentCloudBot struct {
	withConn   func(ctx context.Context, f func(conn *sqlite.Conn) error) error
	httpClient *http.Client
	rateLimit  *time.Ticker
}

func main() {
	dbFile := flag.String("db", "dc.db", "The path of the SQLite DB")
	syslogFlag := flag.Bool("syslog", false, "Also log to syslog")
	debugFlag := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	if *debugFlag {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if *syslogFlag {
		hook, err := lsyslog.NewSyslogHook("", "", syslog.LOG_INFO, "")
		if err != nil {
			logrus.WithError(err).Fatal("Failed to dial syslog")
		}
		logrus.AddHook(hook)
	}

	db, err := sqlite.Open("file:"+*dbFile, 0, 5)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to open database")
	}
	defer db.Close()

	dcb := &DocumentCloudBot{
		withConn: func(ctx context.Context, f func(conn *sqlite.Conn) error) error {
			conn := db.Get(ctx.Done())
			defer db.Put(conn)
			return f(conn)
		},
		httpClient: &http.Client{
			Timeout: 1 * time.Minute,
		},
		rateLimit: time.NewTicker(10 * time.Second),
	}

	if err := dcb.initDB(context.Background()); err != nil {
		logrus.WithError(err).Fatal("Failed to init database")
	}

	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error { return dcb.Latest(ctx) })
	g.Go(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		return errors.Errorf("received signal: %v", <-c)
	})
	logrus.WithError(g.Wait()).Error("Exiting")
}
