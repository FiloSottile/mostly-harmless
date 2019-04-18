package covfefe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fastjson"
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

type timelineMonitor struct {
	c *http.Client
	m chan *Message
	u *twitter.User
}

func (t *timelineMonitor) followTimeline(ctx context.Context, timeline string) error {
	log := logrus.WithFields(logrus.Fields{
		"account": t.u.ScreenName, "timeline": timeline,
	})

	var (
		interval time.Duration
		source   string
	)
	switch timeline {
	case "home":
		interval = 1*time.Minute + 5*time.Second
		source = fmt.Sprintf("tl:%d", t.u.ID)
	case "mentions":
		interval = 15 * time.Second
		source = fmt.Sprintf("at:%d", t.u.ID)
	default:
		return errors.Errorf("unknown timeline %q", timeline)
	}

	tick := time.NewTicker(interval)
	defer tick.Stop()

	var sinceID uint64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}

		url := "https://api.twitter.com/1.1/statuses/%s_timeline.json?count=200"
		url = fmt.Sprintf(url, timeline)
		if sinceID != 0 { // Twitter hates devs.
			url = fmt.Sprintf("%s&since_id=%d", url, sinceID)
		}
		var tweets []json.RawMessage
		if err := getJSON(ctx, t.c, url, &tweets); err != nil {
			return err
		}

		log.WithField("tweets", len(tweets)).Debug("Fetched timeline")

		for _, tweet := range tweets {
			t.m <- &Message{source: source, msg: tweet}
		}

		if len(tweets) > 0 {
			sinceID = fastjson.MustParseBytes(tweets[0]).GetUint64("id")
			if sinceID == 0 {
				return errors.New("couldn't decode tweet ID")
			}
		}

		// There's little point in trying to paginate: with a rate limit of 15
		// requests per 15 minutes, if we are falling behind we will not recover
		// anyway. Also, to be sure there's a need for pagination we'd have to
		// fetch overlapping ranges, unlike suggested at
		// https://developer.twitter.com/en/docs/tweets/timelines/guides/working-with-timelines
		// because "count" is actually a limit, and getting less than 200 tweets
		// does not mean we reached the "max_id". Twitter hates devs.
	}
}

func (c *Covfefe) hydrateTweet(ctx context.Context, id int64) ([]byte, error) {
	url := "https://api.twitter.com/1.1/statuses/show.json?id=%d&include_ext_alt_text=true"
	url = fmt.Sprintf(url, id)
	var tweet json.RawMessage
	if err := getJSON(ctx, c.httpClient, url, &tweet); err != nil {
		return nil, err
	}
	return tweet, nil
}

func getMessage(message []byte) interface{} {
	v := fastjson.MustParseBytes(message)

	var res interface{}
	switch {
	case v.Exists("retweet_count"):
		res = new(twitter.Tweet)
	case v.Exists("event"):
		res = new(twitter.Event)
	case v.Exists("withheld_in_countries") && v.Exists("user_id"):
		res = new(twitter.StatusWithheld)
	case v.Exists("withheld_in_countries"):
		res = new(twitter.UserWithheld)
	case v.Exists("synthetic"):
		fallthrough // migrated deletion events
	case v.Exists("user_id_str"):
		res = new(twitter.StatusDeletion)
	case v.Exists("delete"):
		res = new(twitter.StatusDeletion)
		notice := &struct {
			StatusDeletion interface{} `json:"status"`
		}{StatusDeletion: res}
		json.Unmarshal(v.Get("delete").MarshalTo(nil), notice)
		return res
	case v.Exists("status_withheld"):
		res = new(twitter.StatusWithheld)
		message = v.Get("status_withheld").MarshalTo(nil)
	case v.Exists("user_withheld"):
		res = new(twitter.UserWithheld)
		message = v.Get("user_withheld").MarshalTo(nil)
	case v.Exists("event"):
		res = new(twitter.Event)
	default:
		panic("unrecognized message")
	}

	json.Unmarshal(message, res)
	return res
}
