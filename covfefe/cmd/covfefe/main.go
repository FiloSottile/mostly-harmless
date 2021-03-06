package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log/syslog"

	"filippo.io/mostly-harmless/covfefe"
	log "github.com/sirupsen/logrus"
	lsyslog "github.com/sirupsen/logrus/hooks/syslog"
)

func main() {
	dbFile := flag.String("db", "twitter.db", "The path of the SQLite DB")
	mediaPath := flag.String("media", "twitter-media", "The folder to store media files in")
	credsFile := flag.String("creds", "creds.json", "The path of the credentials JSON")
	syslogFlag := flag.Bool("syslog", false, "Also log to syslog")
	debugFlag := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	if *debugFlag {
		log.SetLevel(log.DebugLevel)
	}
	if *syslogFlag {
		hook, err := lsyslog.NewSyslogHook("", "", syslog.LOG_INFO, "")
		if err != nil {
			log.WithError(err).Fatal("Failed to dial syslog")
		}
		log.AddHook(hook)
	}

	credsJSON, err := ioutil.ReadFile(*credsFile)
	if err != nil {
		log.WithError(err).Fatal("Failed to read credentials file")
	}
	creds := &covfefe.Credentials{}
	if err := json.Unmarshal(credsJSON, creds); err != nil {
		log.WithError(err).Fatal("Failed to parse credentials file")
	}

	if err := covfefe.Run(*dbFile, *mediaPath, creds); err != nil {
		log.WithError(err).Fatal("Failed to run fetcher")
	}
}
