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

type twitterClient struct {
	c *http.Client
	m chan *Message
	u *twitter.User
}

func (t *twitterClient) fetchFollowers(ctx context.Context, followed int64) error {
	log := logrus.WithFields(logrus.Fields{
		"account": t.u.ScreenName, "followed": followed,
	})

	source := fmt.Sprintf("fl:%d", followed)

	interval := 1*time.Minute + 5*time.Second
	tick := time.NewTicker(interval)
	defer tick.Stop()

	var cursor int64 = -1
	for cursor != 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}

		url := "https://api.twitter.com/1.1/followers/list.json?cursor=%d&user_id=%d&count=200&tweet_mode=extended"
		url = fmt.Sprintf(url, cursor, followed)

		var result struct {
			Users      []json.RawMessage
			NextCursor int64 `json:"next_cursor"`
		}
		const maxRetry = 4
		for retry := 0; retry <= maxRetry; retry++ {
			if err := getJSON(ctx, t.c, url, &result); err != nil {
				if retry == maxRetry {
					return err
				} else {
					log.WithField("retry", retry).WithError(err).Error("Failed to fetch timeline")
					time.Sleep(interval)
					continue
				}
			}
			break
		}

		log.WithFields(logrus.Fields{
			"users": len(result.Users), "next": result.NextCursor,
		}).Debug("Fetched followers")

		for _, user := range result.Users {
			t.m <- &Message{source: source, kind: "follower", msg: user}
		}
		cursor = result.NextCursor
	}
	return nil
}

func (t *twitterClient) followTimeline(ctx context.Context, timeline string) error {
	log := logrus.WithFields(logrus.Fields{
		"account": t.u.ScreenName, "timeline": timeline,
	})

	var (
		interval time.Duration
		source   string
		endpoint string
	)
	switch timeline {
	case "home":
		interval = 1*time.Minute + 5*time.Second
		source = fmt.Sprintf("tl:%d", t.u.ID)
		endpoint = "statuses/home_timeline"
	case "mentions":
		interval = 15 * time.Second
		source = fmt.Sprintf("at:%d", t.u.ID)
		endpoint = "statuses/mentions_timeline"
	case "user":
		interval = 10 * time.Second
		source = fmt.Sprintf("us:%d", t.u.ID)
		endpoint = "statuses/user_timeline"
	case "likes":
		interval = 15 * time.Second
		source = fmt.Sprintf("lk:%d", t.u.ID)
		endpoint = "favorites/list"
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

		url := "https://api.twitter.com/1.1/%s.json?count=200&tweet_mode=extended"
		url = fmt.Sprintf(url, endpoint)
		if sinceID != 0 { // Twitter hates devs.
			url = fmt.Sprintf("%s&since_id=%d", url, sinceID)
		}
		var tweets []json.RawMessage
		const maxRetry = 4
		for retry := 0; retry <= maxRetry; retry++ {
			if err := getJSON(ctx, t.c, url, &tweets); err != nil {
				if retry == maxRetry {
					return err
				} else {
					log.WithField("retry", retry).WithError(err).Error("Failed to fetch timeline")
					time.Sleep(60 * time.Second)
					continue
				}
			}
			break
		}

		log.WithField("tweets", len(tweets)).Debug("Fetched timeline")

		for _, tweet := range tweets {
			t.m <- &Message{source: source, kind: "tweet", msg: tweet}
		}

		if len(tweets) > 0 {
			var lastTweet struct {
				ID uint64
			}
			if err := json.Unmarshal(tweets[0], &lastTweet); err != nil {
				return errors.Wrap(err, "couldn't decode tweet ID")
			} else if lastTweet.ID == 0 {
				return errors.New("couldn't decode tweet ID")
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
	}
}

func (c *Covfefe) hydrateTweet(ctx context.Context, id int64) ([]byte, error) {
	url := "https://api.twitter.com/1.1/statuses/show.json?id=%d" +
		"&include_ext_alt_text=true&tweet_mode=extended"
	url = fmt.Sprintf(url, id)
	var tweet json.RawMessage
	if err := getJSON(ctx, c.httpClient, url, &tweet); err != nil {
		return nil, err
	}
	return tweet, nil
}
