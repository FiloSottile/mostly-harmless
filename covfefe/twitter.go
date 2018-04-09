package covfefe

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/FiloSottile/mostly-harmless/covfefe/internal/twitter"
)

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
