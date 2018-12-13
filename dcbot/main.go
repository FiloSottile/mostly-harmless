package main

import (
	"context"
	"flag"
	"log"
	"log/syslog"
	"net/http"
	_ "net/http/pprof"
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
	searchRate *time.Ticker
	assetRate  *time.Ticker
	filePath   string
}

func main() {
	dbFile := flag.String("db", "dc.db", "`path` of the SQLite DB")
	syslogFlag := flag.Bool("syslog", false, "also log to syslog")
	debugFlag := flag.Bool("debug", false, "enable debug logging")
	backFlag := flag.Int("backfill", -1, "enable backfilling from `page`")
	filesFlag := flag.String("files", ".", "store files at `path`")
	drainFlag := flag.Bool("drain", false, "move files out of SQLite")
	flag.Parse()

	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()

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
	defer func() {
		logrus.WithError(db.Close()).Info("Closed database")
	}()

	dcb := &DocumentCloudBot{
		withConn: func(ctx context.Context, f func(conn *sqlite.Conn) error) error {
			conn := db.Get(ctx.Done())
			defer db.Put(conn)
			return f(conn)
		},
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
		searchRate: time.NewTicker(10 * time.Second),
		assetRate:  time.NewTicker(1 * time.Second),
		filePath:   *filesFlag,
	}

	if err := dcb.initDB(context.Background()); err != nil {
		logrus.WithError(err).Fatal("Failed to init database")
	}

	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error { return dcb.Latest(ctx) })
	g.Go(func() error { return dcb.Download(ctx) })
	if *backFlag >= 0 {
		g.Go(func() error { return dcb.Backfill(ctx, *backFlag) })
	}
	if *drainFlag {
		g.Go(func() error { return dcb.Drain(ctx) })
	}
	g.Go(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		select {
		case s := <-c:
			return errors.Errorf("received signal: %v", s)
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	logrus.WithError(g.Wait()).Error("Exiting")
}
