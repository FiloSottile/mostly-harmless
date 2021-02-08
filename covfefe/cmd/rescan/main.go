package main

import (
	"flag"
	"os"
	"runtime/pprof"

	"filippo.io/mostly-harmless/covfefe"
	log "github.com/sirupsen/logrus"
)

func main() {
	dbFile := flag.String("db", "twitter.db", "The path of the SQLite DB")
	debugFlag := flag.Bool("debug", false, "Enable debug logging")
	pprofFlag := flag.Bool("pprof", false, "Write a CPU profile")
	flag.Parse()

	if *debugFlag {
		log.SetLevel(log.DebugLevel)
	}

	if *pprofFlag {
		f, err := os.Create("rescan.pprof")
		if err != nil {
			log.WithError(err).Fatal("Failed to create pprof")
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.WithError(err).Fatal("Failed to start pprof")
		}
		defer pprof.StopCPUProfile()
	}

	if err := covfefe.Rescan(*dbFile); err != nil {
		log.WithError(err).Fatal("Failed to run rescan")
	}
}
