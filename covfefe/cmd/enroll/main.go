package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/FiloSottile/mostly-harmless/covfefe"
	"github.com/dghubble/oauth1"
	"github.com/dghubble/oauth1/twitter"
	log "github.com/sirupsen/logrus"
)

func main() {
	credsFile := flag.String("creds", "creds.json", "The path of the credentials JSON")
	flag.Parse()

	credsJSON, err := ioutil.ReadFile(*credsFile)
	if err != nil {
		log.WithError(err).Fatal("Failed to read credentials file")
	}
	creds := &covfefe.Credentials{}
	if err := json.Unmarshal(credsJSON, creds); err != nil {
		log.WithError(err).Fatal("Failed to parse credentials file")
	}

	config := oauth1.Config{
		ConsumerKey:    creds.APIKey,
		ConsumerSecret: creds.APISecret,
		CallbackURL:    "oob",
		Endpoint:       twitter.AuthorizeEndpoint,
	}

	requestToken, _, err := config.RequestToken()
	if err != nil {
		log.WithError(err).Fatal("RequestToken failed")
	}
	authorizationURL, err := config.AuthorizationURL(requestToken)
	if err != nil {
		log.WithError(err).Fatal("AuthorizationURL failed")
	}
	fmt.Fprintf(os.Stderr, "URL: %s\n", authorizationURL.String())

	fmt.Fprintf(os.Stderr, "Paste your PIN here: ")
	var verifier string
	if _, err := fmt.Scanf("%s", &verifier); err != nil {
		log.WithError(err).Fatal("Scan failed")
	}

	accessToken, accessSecret, err := config.AccessToken(requestToken, "secret does not matter", verifier)
	if err != nil {
		log.WithError(err).Fatal("AccessToken failed")
	}

	creds.Accounts = append(creds.Accounts, covfefe.Account{
		Token: accessToken, TokenSecret: accessSecret,
	})

	res, err := json.MarshalIndent(creds, "", "\t")
	if err != nil {
		log.WithError(err).Fatal("MarshalIndent failed")
	}
	os.Stdout.Write(res)
	os.Stdout.WriteString("\n")
}
