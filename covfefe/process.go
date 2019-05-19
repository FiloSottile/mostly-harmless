package covfefe

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/h2non/filetype"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Message struct {
	source string
	kind   string
	msg    []byte
	id     int64
}

func (c *Covfefe) processTweet(id int64, tweet *twitter.Tweet) {
	// Just in case we forget the magic tweet_mode=extended and end up archiving
	// truncated tweets without full_text again, ugh.
	// https://developer.twitter.com/en/docs/tweets/tweet-updates
	if tweet.Truncated && !c.rescan {
		log.WithFields(log.Fields{
			"message": id, "tweet": tweet.ID,
		}).Warn("Truncated tweet")
	}

	if new, err := c.insertTweet(tweet, id); err != nil {
		log.WithFields(log.Fields{
			"err": err, "message": id, "tweet": tweet.ID,
		}).Error("Failed to insert tweet")
		return
	} else if !new {
		return
	}

	c.processUser(id, tweet.User)

	if tweet.RetweetedStatus != nil {
		c.processTweet(id, tweet.RetweetedStatus)
	}
	if tweet.QuotedStatus != nil {
		c.processTweet(id, tweet.QuotedStatus)
	}

	c.fetchMedia(tweet)
	c.fetchParent(tweet)
}

func (c *Covfefe) fetchMedia(tweet *twitter.Tweet) {
	if c.rescan {
		return
	}

	var media []twitter.MediaEntity
	if tweet.Entities != nil {
		media = tweet.Entities.Media
	}
	if tweet.ExtendedEntities != nil {
		media = tweet.ExtendedEntities.Media
	}
	if tweet.ExtendedTweet != nil {
		if tweet.ExtendedTweet.Entities != nil {
			media = tweet.ExtendedTweet.Entities.Media
		}
		if tweet.ExtendedTweet.ExtendedEntities != nil {
			media = tweet.ExtendedTweet.ExtendedEntities.Media
		}
	}
	for _, m := range media {
		if m.SourceStatusID != 0 {
			// We'll find this media attached to the retweet.
			continue
		}
		for retry := 0; retry < 3; retry++ {
			log := log.WithFields(log.Fields{
				"retry": retry,
				"url":   m.MediaURLHttps,
				"media": m.ID, "tweet": tweet.ID,
			})
			body, err := c.httpGet(m.MediaURLHttps)
			if err != nil {
				log.WithError(err).Error("Failed to download media")
				continue
			}
			if err := c.saveMedia(body, m.ID); err != nil {
				log.WithError(err).Error("Failed to save media")
				continue
			}
			break
		}
	}
}

func (c *Covfefe) fetchParent(tweet *twitter.Tweet) {
	if c.rescan || tweet.InReplyToStatusID == 0 {
		return
	}

	log := log.WithFields(log.Fields{
		"tweet": tweet.ID, "parent": tweet.InReplyToStatusID,
	})

	if seen, err := c.seenTweet(tweet.InReplyToStatusID); err != nil {
		log.WithError(err).Error("Failed to check if parent tweet was already seen")
		return
	} else if seen {
		log.Debug("Parent already in the database")
		return
	}

	log.Debug("Fetching parent tweet")
	parent, err := c.hydrateTweet(context.TODO(), tweet.InReplyToStatusID)
	if err != nil {
		log.WithError(err).Error("Failed to hydrate parent tweet")
		return
	}
	c.Handle(&Message{
		source: fmt.Sprintf("parent:%d", tweet.ID),
		kind:   "tweet",
		msg:    parent,
	})
}

func (c *Covfefe) saveMedia(data []byte, id int64) error {
	t, err := filetype.Match(data)
	if err != nil {
		return errors.WithStack(err)
	}

	name := filepath.Join(c.mediaPath, fmt.Sprintf("%d.%s", id, t.Extension))
	base := filepath.Join(c.mediaPath, fmt.Sprintf("%d", id))

	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return errors.WithStack(err)
	}
	if _, err := f.Write(data); err != nil {
		return errors.WithStack(err)
	}
	if err := f.Close(); err != nil {
		return errors.WithStack(err)
	}

	return exec.Command("tesseract", name, base).Run()
}

func (c *Covfefe) processUser(id int64, user *twitter.User) {
	if err := c.insertUser(user, id); err != nil {
		log.WithError(err).WithField("message", id).Error("Failed to insert user")
	}
}

func isProtected(message interface{}) bool {
	switch m := message.(type) {
	case *twitter.Tweet:
		if m.User.Protected {
			return true
		}
	case *twitter.Event:
		if (m.Source != nil && m.Source.Protected) ||
			(m.Target != nil && m.Target.Protected) ||
			(m.TargetObject != nil && m.TargetObject.User.Protected) {
			return true
		}
		switch m.Event {
		case "quoted_tweet":
		case "favorite", "unfavorite":
		case "favorited_retweet":
		case "retweeted_retweet":
		case "follow", "unfollow":
		case "user_update":
		case "list_created", "list_destroyed", "list_updated", "list_member_added",
			"list_member_removed", "list_user_subscribed", "list_user_unsubscribed":
			return true // lists can be private
		case "block", "unblock":
			return true
		case "mute", "unmute":
			return true
		default:
			log.WithField("event", m.Event).Warning("Unknown event type")
			return true // when in doubt...
		}
	}
	return false
}

func (c *Covfefe) HandleChan(messages <-chan *Message) {
	for m := range messages {
		c.Handle(m)
	}
}

func (c *Covfefe) Handle(m *Message) {
	msg := unmarshalMessage(m)
	if msg == nil {
		log.Debug("Dropped unknown message")
		return
	}

	if isProtected(msg) {
		log.Debug("Dropped protected message")
		return
	}

	switch obj := msg.(type) {
	case *twitter.Tweet:
		if err := c.insertMessage(m); err != nil {
			log.WithError(err).WithField("tweet", obj.ID).Error("Failed to insert message")
			return
		}
		c.processTweet(m.id, obj)
	case *twitter.StatusDeletion:
		if err := c.insertMessage(m); err != nil {
			log.WithError(err).WithField("deletion", obj.ID).Error("Failed to insert message")
			return
		}
		log.WithField("id", obj.ID).Debug("Deleted Tweet")
		c.deletedTweet(obj.ID, m.id)
	case *twitter.Event:
		if err := c.insertMessage(m); err != nil {
			log.WithError(err).WithField("event", obj.Event).Error("Failed to insert message")
			return
		}
		if obj.Source != nil {
			c.processUser(m.id, obj.Source)
		}
		if obj.Target != nil {
			c.processUser(m.id, obj.Target)
		}
		if obj.TargetObject != nil {
			c.processTweet(m.id, obj.TargetObject)
		}
		if obj.Event == "follow" {
			if err := c.insertFollow(obj.Source.ID, obj.Target.ID, m.id); err != nil {
				log.WithError(err).WithField("message", m.id).Error("Failed to insert follow")
			}
		}
	}
}

func unmarshalMessage(m *Message) interface{} {
	switch m.kind {
	case "tweet":
		res := new(twitter.Tweet)
		json.Unmarshal(m.msg, res)
		return res
	case "event":
		res := new(twitter.Event)
		json.Unmarshal(m.msg, res)
		return res
	case "del":
		res := new(twitter.StatusDeletion)
		json.Unmarshal(m.msg, res)
		return res
	case "deletion":
		res := new(struct {
			Delete struct {
				Status *twitter.StatusDeletion
			}
		})
		json.Unmarshal(m.msg, res)
		return res.Delete.Status
	default:
		return nil
	}
}

func (c *Covfefe) httpGet(url string) ([]byte, error) {
	res, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("fetching %q returned status %q", url, res.Status)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}
