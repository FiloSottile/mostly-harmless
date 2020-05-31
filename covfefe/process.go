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
	"strconv"
	"strings"

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
	log := log.WithFields(log.Fields{"message": id, "tweet": tweet.ID})
	// Just in case we forget the magic tweet_mode=extended and end up archiving
	// truncated tweets without full_text again, ugh. (This also works where not
	// documented like followers/list.json. Sigh.)
	// https://developer.twitter.com/en/docs/tweets/tweet-updates
	if tweet.Truncated && !c.rescan {
		log.Warn("Truncated tweet")
	}

	// User objects drop the user field not only of the top-level tweet, for
	// which we know and set the user, but also for nested tweets not by the
	// same user :(
	if tweet.User == nil {
		// TODO: hydrate this tweet instead.
		log.Debug("User-less tweet :(")
		return
	}

	if new, err := c.insertTweet(tweet, id); err != nil {
		log.WithError(err).Error("Failed to insert tweet")
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
	if user.Status != nil {
		user.Status.User = user
		c.processTweet(id, user.Status)
		user.Status.User = nil
	}
}

func (c *Covfefe) HandleChan(messages <-chan *Message) {
	for m := range messages {
		c.Handle(m)
	}
}

func (c *Covfefe) Handle(m *Message) {
	switch m.kind {
	case "tweet":
		tweet := new(twitter.Tweet)
		if err := json.Unmarshal(m.msg, tweet); err != nil {
			log.WithError(err).Warning("Failed to unmarshal message")
			return
		}

		if tweet.User.Protected {
			log.Debug("Dropped protected message")
			return
		}

		if err := c.insertMessage(m); err != nil {
			log.WithError(err).Error("Failed to insert message")
			return
		}

		c.processTweet(m.id, tweet)

	case "event":
		event := new(twitter.Event)
		if err := json.Unmarshal(m.msg, event); err != nil {
			log.WithError(err).Warning("Failed to unmarshal message")
			return
		}

		if err := c.insertMessage(m); err != nil {
			log.WithError(err).Error("Failed to insert message")
			return
		}

		if event.Source != nil {
			c.processUser(m.id, event.Source)
		}
		if event.Target != nil {
			c.processUser(m.id, event.Target)
		}
		if event.TargetObject != nil {
			c.processTweet(m.id, event.TargetObject)
		}
		if event.Event == "follow" {
			if err := c.insertFollow(event.Source.ID, event.Target.ID, m.id); err != nil {
				log.WithError(err).WithField("message", m.id).Error("Failed to insert follow")
			}
		}

	case "del":
		del := new(twitter.StatusDeletion)
		if err := json.Unmarshal(m.msg, del); err != nil {
			log.WithError(err).Warning("Failed to unmarshal message")
			return
		}

		if err := c.insertMessage(m); err != nil {
			log.WithError(err).Error("Failed to insert message")
			return
		}

		c.deletedTweet(del.ID, m.id)

	case "deletion":
		del := new(struct {
			Delete struct {
				Status *twitter.StatusDeletion
			}
		})
		if err := json.Unmarshal(m.msg, del); err != nil {
			log.WithError(err).Warning("Failed to unmarshal message")
			return
		}

		if err := c.insertMessage(m); err != nil {
			log.WithError(err).Error("Failed to insert message")
			return
		}

		c.deletedTweet(del.Delete.Status.ID, m.id)

	case "follower":
		user := new(twitter.User)
		if err := json.Unmarshal(m.msg, user); err != nil {
			log.WithError(err).Warning("Failed to unmarshal message")
			return
		}

		if err := c.insertMessage(m); err != nil {
			log.WithError(err).Error("Failed to insert message")
			return
		}

		c.processUser(m.id, user)
		target, err := strconv.ParseInt(strings.TrimPrefix(m.source, "fl:"), 10, 64)
		if err != nil {
			log.WithError(err).WithField("source", m.source).Error("Could not reconstruct target")
		} else {
			if err := c.insertFollow(user.ID, target, m.id); err != nil {
				log.WithError(err).WithField("message", m.id).Error("Failed to insert follow")
			}
		}

	default:
		log.Warning("Dropped unknown message")
		return
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
