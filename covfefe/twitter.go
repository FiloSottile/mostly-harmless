package covfefe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/FiloSottile/mostly-harmless/covfefe/internal/twitter"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func getJSON(ctx context.Context, c *http.Client, url string, v interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	r, err := c.Do(req.WithContext(ctx))
	if err != nil {
		return errors.Wrapf(err, "error getting %s", url)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		var errs struct {
			Errors []struct {
				Message string
			}
		}
		json.NewDecoder(r.Body).Decode(&errs)
		for _, e := range errs.Errors {
			return errors.Errorf("error getting %s: %s", url, e.Message)
		}
		return errors.Errorf("error getting %s: %s", url, r.Status)
	}

	return errors.Wrapf(json.NewDecoder(r.Body).Decode(v),
		"error reading and decoding %q", url)
}

func verifyCredentials(ctx context.Context, c *http.Client) (*twitter.User, error) {
	url := "https://api.twitter.com/1.1/account/verify_credentials.json?skip_status=true"
	var u *twitter.User
	if err := getJSON(ctx, c, url, &u); err != nil {
		return nil, err
	}
	return u, nil
}

type Message struct {
	account *twitter.User
	msg     []byte
	id      int64
}

func followTimeline(ctx context.Context, c *http.Client, u *twitter.User, m chan *Message) error {
	tick := time.NewTicker(1 * time.Minute)
	defer tick.Stop()

	var sinceID uint64
	for {
		url := "https://api.twitter.com/1.1/statuses/home_timeline.json?count=200"
		if sinceID != 0 { // Twitter hates devs.
			url = fmt.Sprintf("%s&since_id=%d", url, sinceID)
		}
		var tweets []json.RawMessage
		if err := getJSON(ctx, c, url, &tweets); err != nil {
			return err
		}

		log.WithField("account", u.ScreenName).WithField("tweets", len(tweets)).Debug(
			"Fetched home timeline")

		for _, t := range tweets {
			m <- &Message{account: u, msg: t}
		}

		if len(tweets) > 0 {
			var lastTweet struct {
				ID uint64
			}
			if err := json.Unmarshal(tweets[0], &lastTweet); err != nil {
				return errors.Wrap(err, "couldn't decode tweet ID")
			}
			sinceID = lastTweet.ID
		}

		// There's little point in trying to paginate: with a rate limit of 15
		// requests per 15 minutes, if we are falling behind we will not recover
		// anyway. Also, to be sure there's a need for pagination we'd have to
		// fetch overlapping ranges, unlike suggested at
		// https://developer.twitter.com/en/docs/tweets/timelines/guides/working-with-timelines
		// because "count" is actually a limit, and getting less than 200 tweets
		// does not mean we reached the "max_id". Twitter hates devs.

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}

func getMessage(token []byte) interface{} {
	var data map[string]json.RawMessage
	err := json.Unmarshal(token, &data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", token)
		panic(err)
	}

	var res interface{}
	switch {
	case hasPath(data, "retweet_count"):
		res = new(twitter.Tweet)

	case hasPath(data, "event"):
		res = new(twitter.Event)
	case hasPath(data, "withheld_in_countries") && hasPath(data, "user_id"):
		res = new(twitter.StatusWithheld)
	case hasPath(data, "withheld_in_countries"):
		res = new(twitter.UserWithheld)
	case hasPath(data, "synthetic"):
		fallthrough // migrated deletion events
	case hasPath(data, "user_id_str"):
		res = new(twitter.StatusDeletion)

	case hasPath(data, "direct_message"):
		res = new(twitter.DirectMessage)
		token = data["direct_message"]
	case hasPath(data, "delete"):
		res = new(twitter.StatusDeletion)
		notice := &struct {
			StatusDeletion interface{} `json:"status"`
		}{StatusDeletion: res}
		json.Unmarshal(data["delete"], notice)
		return res
	case hasPath(data, "scrub_geo"):
		res = new(twitter.LocationDeletion)
		token = data["scrub_geo"]
	case hasPath(data, "limit"):
		res = new(twitter.StreamLimit)
		token = data["limit"]
	case hasPath(data, "status_withheld"):
		res = new(twitter.StatusWithheld)
		token = data["status_withheld"]
	case hasPath(data, "user_withheld"):
		res = new(twitter.UserWithheld)
		token = data["user_withheld"]
	case hasPath(data, "disconnect"):
		res = new(twitter.StreamDisconnect)
		token = data["disconnect"]
	case hasPath(data, "warning"):
		res = new(twitter.StallWarning)
		token = data["warning"]
	case hasPath(data, "friends"):
		res = new(twitter.FriendsList)
	case hasPath(data, "event"):
		res = new(twitter.Event)
	default:
		res = make(map[string]interface{})
		json.Unmarshal(token, &res)
		return res
	}

	json.Unmarshal(token, res)
	return res
}

func hasPath(data map[string]json.RawMessage, key string) bool {
	_, ok := data[key]
	return ok
}
