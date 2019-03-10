package covfefe

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/h2non/filetype"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Message struct {
	source string
	msg    []byte
	id     int64
}

func (c *Covfefe) processTweet(id int64, tweet *twitter.Tweet) {
	new, err := c.insertTweet(tweet, id)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err, "message": id, "tweet": tweet.ID,
		}).Error("Failed to insert tweet")
		return
	}
	if !new {
		return
	}

	c.processUser(id, tweet.User)

	if tweet.RetweetedStatus != nil {
		c.processTweet(id, tweet.RetweetedStatus)
	}
	if tweet.QuotedStatus != nil {
		c.processTweet(id, tweet.QuotedStatus)
	}

	// The rest of the function generates new Media and Message entries with
	// contemporaneous discovery or fetch requests.
	if c.rescan {
		return
	}

	c.fetchMedia(tweet)
	c.fetchParent(tweet)
}

func (c *Covfefe) fetchMedia(tweet *twitter.Tweet) {
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
		log := log.WithFields(log.Fields{
			"url": m.MediaURLHttps, "media": m.ID, "tweet": tweet.ID,
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
	}
}

func (c *Covfefe) fetchParent(tweet *twitter.Tweet) {
	if tweet.InReplyToStatusID == 0 {
		return
	}

	log := log.WithFields(log.Fields{
		"tweet": tweet.ID, "parent": tweet.InReplyToStatusID,
	})
	seen, err := c.seenTweet(tweet.InReplyToStatusID)
	if err != nil {
		log.WithError(err).Error("Failed to check if parent tweet was already seen")
		return
	}
	if seen {
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
		msg:    parent,
	})
}

func (c *Covfefe) saveMedia(data []byte, id int64) error {
	t, err := filetype.Match(data)
	if err != nil {
		return errors.WithStack(err)
	}
	name := filepath.Join(c.mediaPath, fmt.Sprintf("%d.%s", id, t.Extension))
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return errors.WithStack(err)
	}
	if _, err := f.Write(data); err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(f.Close())
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
	msg := getMessage(m.msg)

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

	// There are a couple of these events in the stream from the Streaming APIs
	// days, but no new ones are generated.
	case *twitter.StatusWithheld:
	case *twitter.UserWithheld:

	default:
		log.Warningf("Unhandled message type: %T", msg)
	}
}

func (c *Covfefe) httpGet(url string) ([]byte, error) {
	// TODO: retry
	res, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}
