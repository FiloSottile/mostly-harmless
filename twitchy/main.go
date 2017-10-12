package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
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

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	return "token_cache.json", nil
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

var amountRegexp = regexp.MustCompile(`\$\s*\d+`)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	ctx := context.Background()

	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved credentials
	config, err := google.ConfigFromJSON(b, gmail.GmailModifyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(ctx, config)
	client.Timeout = 60 * time.Second

	srv, err := gmail.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve gmail Client %v", err)
	}

	for range time.NewTicker(30 * time.Second).C {
		msgs, err := getUnseenEmails(srv)
		if err != nil {
			log.Fatal(err)
		}
		for _, msg := range msgs {
			val := string(amountRegexp.Find(msg.TextBody))
			if val == "" {
				val = "???"
			}
			name := strings.TrimPrefix(msg.Subject, "Fwd: ")
			if len(name) > 60 {
				name = name[:60]
			}
			line := fmt.Sprintf(`Latest donation: %s by "%s". Thank you!`, val, name)
			log.Print(line)
			if err := setBanner(line); err != nil {
				log.Fatal(err)
			}
			if err := markMessageSeen(srv, msg.ID); err != nil {
				log.Fatal(err)
			}
			time.Sleep(30 * time.Second)
		}
	}
}

type Message struct {
	ID       string
	Subject  string
	TextBody []byte
}

var query = `"Internet Archive" NOT label:SEEN`

func getUnseenEmails(srv *gmail.Service) ([]*Message, error) {
	r, err := srv.Users.Messages.List("me").Q(query).Do()
	if err != nil {
		return nil, err
	}
	var res []*Message
	for _, ml := range r.Messages {
		msg := &Message{}
		m, err := srv.Users.Messages.Get("me", ml.Id).Do()
		if err != nil {
			return nil, err
		}
		msg.ID = ml.Id
		for _, hdr := range m.Payload.Headers {
			if hdr.Name != "Subject" {
				continue
			}
			msg.Subject = hdr.Value
		}
		for _, part := range m.Payload.Parts {
			if part.MimeType != "text/plain" {
				continue
			}
			data, err := base64.RawURLEncoding.DecodeString(part.Body.Data)
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
