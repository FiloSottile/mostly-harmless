package covfefe

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/dghubble/go-twitter/twitter"
	log "github.com/sirupsen/logrus"
)

func (c *Covfefe) processTweet(messageID int64, tweet *twitter.Tweet) {
	new, err := c.insertTweet(tweet, messageID)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err, "message": messageID, "tweet": tweet.ID,
		}).Error("Failed to insert tweet")
		return
	}
	if !new {
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
	if len(media) != 0 {
		c.wg.Add(1)
		go func() {
			for _, m := range media {
				if m.SourceStatusID != 0 {
					// We'll find this media attached to the retweet.
					continue
				}
				body, err := c.httpGet(m.MediaURLHttps)
				if err != nil {
					log.WithFields(log.Fields{
						"err": err, "url": m.MediaURLHttps,
						"media": m.ID, "tweet": tweet.ID,
					}).Error("Failed to download media")
					continue
				}
				c.insertMedia(body, m.ID, tweet.ID)
				// TODO: archive videos?
			}
			c.wg.Done()
		}()
	}

	if tweet.RetweetedStatus != nil {
		c.processTweet(messageID, tweet.RetweetedStatus)
	}
	if tweet.QuotedStatus != nil {
		c.processTweet(messageID, tweet.QuotedStatus)
	}
	// TODO: crawl thread, non-embedded linked tweets
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
			log.WithFields(log.Fields{
				"event": m.Event, "json": mustMarshal(m),
			}).Warning("Unknown event type")
			return true // when in doubt...
		}
	}
	return false
}

func (c *Covfefe) HandleChan(messages <-chan Message) {
	for m := range messages {
		log.WithFields(log.Fields{
			"type":    fmt.Sprintf("%T", m.msg),
			"account": m.account,
		}).Debug("Received message")

		if isProtected(m.msg) {
			log.Debug("Dropped protected message")
			continue
		}

		switch msg := m.msg.(type) {
		case *twitter.Tweet:
			messageID, err := c.insertMessage(m.account, msg)
			if err != nil {
				log.WithError(err).WithField("tweet", msg.ID).Error("Failed to insert message")
				continue
			}
			c.processTweet(messageID, msg)
		case *twitter.StatusDeletion:
			messageID, err := c.insertMessage(m.account, msg)
			if err != nil {
				log.WithError(err).WithField("deletion", msg.ID).Error("Failed to insert message")
				continue
			}
			log.WithField("id", msg.ID).Debug("Deleted Tweet")
			c.deletedTweet(messageID, msg.ID)
		case *twitter.Event:
			messageID, err := c.insertMessage(m.account, msg)
			if err != nil {
				log.WithError(err).WithField("event", msg.Event).Error("Failed to insert message")
				continue
			}
			if msg.TargetObject != nil {
				c.processTweet(messageID, msg.TargetObject)
			}

		case *twitter.StatusWithheld:
			_, err := c.insertMessage(m.account, msg)
			if err != nil {
				log.WithError(err).Error("Failed to insert message")
			}
			log.WithFields(log.Fields{
				"id": strconv.FormatInt(msg.ID, 10), "user": strconv.FormatInt(msg.UserID, 10),
				"countries": strings.Join(msg.WithheldInCountries, ","),
			}).Info("Status withheld")
		case *twitter.UserWithheld:
			_, err := c.insertMessage(m.account, msg)
			if err != nil {
				log.WithError(err).Error("Failed to insert message")
			}
			log.WithFields(log.Fields{
				"user":      strconv.FormatInt(msg.ID, 10),
				"countries": strings.Join(msg.WithheldInCountries, ","),
			}).Info("User withheld")

		case *twitter.StreamLimit:
			log.WithFields(log.Fields{
				"type": "StreamLimit", "track": msg.Track,
			}).Warn("Warning message")
		case *twitter.StreamDisconnect:
			log.WithFields(log.Fields{
				"type": "StreamDisconnect", "reason": msg.Reason,
				"name": msg.StreamName, "code": msg.Code,
			}).Warn("Warning message")
		case *twitter.StallWarning:
			log.WithFields(log.Fields{
				"type": "StallWarning", "message": msg.Message,
				"code": msg.Code, "percent": msg.PercentFull,
			}).Warn("Warning message")
		}
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
