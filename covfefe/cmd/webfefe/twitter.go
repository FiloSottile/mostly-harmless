package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/FiloSottile/mostly-harmless/covfefe"
	"github.com/dghubble/go-twitter/twitter"
	oauth1Login "github.com/dghubble/gologin/oauth1"
	twitterLogin "github.com/dghubble/gologin/twitter"
	"github.com/dghubble/oauth1"
	"github.com/sirupsen/logrus"
)

func loadKnownUsers(oauth1Config *oauth1.Config, accounts []covfefe.Account) (map[int64]bool, error) {
	knownUsers := make(map[int64]bool)
	for _, account := range accounts {
		httpClient := oauth1Config.Client(context.TODO(),
			oauth1.NewToken(account.Token, account.TokenSecret))
		twitterClient := twitter.NewClient(httpClient)
		accountVerifyParams := &twitter.AccountVerifyParams{
			IncludeEntities: twitter.Bool(false),
			SkipStatus:      twitter.Bool(true),
			IncludeEmail:    twitter.Bool(false),
		}
		user, resp, err := twitterClient.Accounts.VerifyCredentials(accountVerifyParams)
		if err != nil {
			return nil, fmt.Errorf("failed to verify token %s: %v", account.Token, err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to verify token %s: %v", account.Token, resp.Status)
		}
		if user == nil || user.ID == 0 || user.IDStr == "" {
			return nil, fmt.Errorf("failed to verify token %s: empty response", account.Token)
		}
		knownUsers[user.ID] = true
	}
	return knownUsers, nil
}

func (s *Server) loggedIn(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := s.store.Get(r, "webfefe")
		if loggedIn, ok := session.Values["logged-in"].(bool); ok && loggedIn {
			fn(w, r)
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	twitterUser, err := twitterLogin.UserFromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	accessToken, accessSecret, err := oauth1Login.AccessTokenFromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logrus.WithFields(logrus.Fields{
		"name": twitterUser.ScreenName, "ID": twitterUser.ID,
	}).Info("User logged in")

	session, _ := s.store.Get(r, "webfefe")
	session.Values["logged-in"] = true
	session.Values["twitter-user"] = twitterUser.ID
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)

	s.credsMu.Lock()
	defer s.credsMu.Unlock()
	if s.knownUsers[twitterUser.ID] {
		return
	}
	logrus.WithFields(logrus.Fields{
		"name": twitterUser.ScreenName, "token": accessToken,
	}).Info("New user")

	credsJSON, err := ioutil.ReadFile(s.credsPath)
	if err != nil {
		logrus.WithError(err).Error("Failed to read credentials file")
		return
	}
	creds := &covfefe.Credentials{}
	if err := json.Unmarshal(credsJSON, creds); err != nil {
		logrus.WithError(err).Error("Failed to parse credentials file")
		return
	}
	creds.Accounts = append(creds.Accounts, covfefe.Account{
		Token: accessToken, TokenSecret: accessSecret,
	})
	credsJSON, err = json.MarshalIndent(creds, "", "    ")
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal credentials file")
		return
	}
	if ioutil.WriteFile(s.credsPath, credsJSON, 0644); err != nil {
		logrus.WithError(err).Error("Failed to write credentials file")
		return
	}

	s.knownUsers[twitterUser.ID] = true
}
