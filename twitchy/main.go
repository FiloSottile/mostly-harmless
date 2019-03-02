// Copyright 2017 Filippo Valsorda
//
// Permission to use, copy, modify, and/or distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

// Command twitchy provides a Twitch bot that notifies a stream about new
// donation receipts received via Gmail.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config, cacheFile string) *http.Client {
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

var amountRegexp = regexp.MustCompile(`\$\s*(\d+)`)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	ctx := context.Background()

	var nightconfig struct {
		ID, Secret, Redirect string
	}

	nb, err := ioutil.ReadFile("client_secret_nightbot.json")
	if err != nil {
		log.Fatalf("Unable to read nightbot secret file: %v", err)
	}
	if err := json.Unmarshal(nb, &nightconfig); err != nil {
		log.Fatalf("Unable to read nightbot secret file: %v", err)
	}

	nightbot := getClient(ctx, &oauth2.Config{
		ClientID:     nightconfig.ID,
		ClientSecret: nightconfig.Secret,
		RedirectURL:  nightconfig.Redirect,
		Scopes:       []string{"channel", "channel_send"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://api.nightbot.tv/oauth2/authorize",
			TokenURL: "https://api.nightbot.tv/oauth2/token",
		},
	}, "token_cache_nightbot.json")
	nightbot.Timeout = 60 * time.Second

	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved credentials
	config, err := google.ConfigFromJSON(b, gmail.GmailModifyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(ctx, config, "token_cache.json")
	client.Timeout = 60 * time.Second

	srv, err := gmail.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve gmail Client %v", err)
	}

	var total int

	msgs, err := getEmails(srv, false)
	if err != nil {
		log.Fatal(err)
	}
	for _, msg := range msgs {
		match := amountRegexp.FindSubmatch(msg.TextBody)
		val, err := strconv.Atoi(string(match[1]))
		if err != nil {
			log.Fatal(err)
		}
		total += val
	}
	log.Printf("Starting with total: $%d", total)

	for range time.NewTicker(30 * time.Second).C {
		msgs, err := getEmails(srv, true)
		if err != nil {
			log.Fatal(err)
		}
		for _, msg := range msgs {
			match := amountRegexp.FindSubmatch(msg.TextBody)
			if match == nil {
				log.Println("Got an email with no regex match! :(")
				continue
			}
			val, err := strconv.Atoi(string(match[1]))
			if err != nil {
				log.Fatal(err)
			}
			total += val
			name := strings.Split(msg.From, " ")[0]
			line := fmt.Sprintf(`Latest donation: $%d by %s. Thank you! Total: $%d`, val, name, total)
			log.Print(line)
			if err := setBanner(line); err != nil {
				log.Fatal(err)
			}
			nightbot.PostForm("https://api.nightbot.tv/1/channel/send", url.Values{
				"message": []string{fmt.Sprintf("Thanks to %s for a %d$ donation to the Internet Archive!", name, val)},
			})
			time.Sleep(5 * time.Second)
			nightbot.PostForm("https://api.nightbot.tv/1/channel/send", url.Values{
				"message": []string{fmt.Sprintf("We raised %d$ so far! Donate at archive.org/donate and forward your receipt to filippo.donations@gmail.com :)", total)},
			})
			if err := markMessageSeen(srv, msg.ID); err != nil {
				log.Fatal(err)
			}
			time.Sleep(5 * time.Minute)
		}
	}
}

type Message struct {
	ID       string
	Subject  string
	From     string
	TextBody []byte
}

func getEmails(srv *gmail.Service, unseen bool) ([]*Message, error) {
	query := `"Internet Archive" label:SEEN`
	if unseen {
		query = `"Internet Archive" NOT label:SEEN`
	}
	r, err := srv.Users.Messages.List("me").Q(query).Do()
	if err != nil {
		return nil, err
	}
	var res []*Message
	for i := range r.Messages {
		ml := r.Messages[len(r.Messages)-1-i]
		msg := &Message{}
		m, err := srv.Users.Messages.Get("me", ml.Id).Do()
		if err != nil {
			return nil, err
		}
		msg.ID = ml.Id
		for _, hdr := range m.Payload.Headers {
			if hdr.Name == "Subject" {
				msg.Subject = hdr.Value
			}
			if hdr.Name == "From" {
				msg.From = hdr.Value
			}
		}
		for _, part := range m.Payload.Parts {
			if part.MimeType != "text/plain" {
				continue
			}
			data, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err != nil {
				return nil, err
			}
			msg.TextBody = data
		}
		res = append(res, msg)
	}
	return res, nil
}

var seenLabelID = "Label_1"

func markMessageSeen(srv *gmail.Service, ID string) error {
	_, err := srv.Users.Messages.Modify("me", ID, &gmail.ModifyMessageRequest{
		AddLabelIds: []string{seenLabelID},
	}).Do()
	return err
}

var bannerFile = "/Users/filippo/banner.txt"

func setBanner(msg string) error {
	data, err := ioutil.ReadFile(bannerFile)
	if err != nil {
		return err
	}
	content := &bytes.Buffer{}
	content.Write(bytes.Split(data, []byte("\n"))[0])
	content.Write([]byte("\n"))
	content.WriteString(msg)
	return ioutil.WriteFile(bannerFile, content.Bytes(), 0664)
}
