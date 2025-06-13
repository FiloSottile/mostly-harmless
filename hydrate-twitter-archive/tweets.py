# Inputs:
#  - tweets.js
#  - like.js
#  - direct-messages.js
#  - direct-messages-group.js
#  - tweets.jsonl (for already fetched tweets)

import tweepy
import json
import sys
import os

want, have = set(), set()


def want_tweet_id(id):
    if id is None:
        return
    if type(id) != str:
        raise TypeError
    want.add(id)


def want_tweet(tweet):
    for attr in ("id_str", "in_reply_to_status_id_str", "quoted_status_id_str"):
        if attr in tweet:
            want_tweet_id(tweet[attr])
    for attr in ("retweeted_status", "quoted_status"):
        if attr in tweet:
            want_tweet(tweet[attr])


for name in sys.argv[1:]:
    with open(name) as f:
        data = f.read()
        if name.endswith(".jsonl"):
            for i, l in enumerate(data.splitlines()):
                t = json.loads(l)
                if "id_str" in t:
                    have.add(t["id_str"])
                    want_tweet(t)
                else:
                    have.add(str(t["id"]))
                    want.add(str(t["id"]))
        if name.endswith(".js"):
            data = data.split(" = ", maxsplit=1)[1]
            data = json.loads(data)
            for entry in data:
                if "like" in entry:
                    want_tweet_id(entry["like"]["tweetId"])
                if "tweet" in entry:
                    want_tweet(entry["tweet"])
                if "dmConversation" in entry:
                    for m in entry["dmConversation"]["messages"]:
                        if "messageCreate" in m:
                            for u in m["messageCreate"].get("urls", []):
                                if "/status/" in u["expanded"]:
                                    want_tweet_id(u["expanded"].split("/status/")[1])

CONSUMER_KEY = os.getenv("TWITTER_CONSUMER_KEY")
CONSUMER_SECRET = os.getenv("TWITTER_CONSUMER_SECRET")

oauth = tweepy.OAuth1UserHandler(CONSUMER_KEY, CONSUMER_SECRET, callback="oob")

print(oauth.get_authorization_url(access_type="read"), file=sys.stderr)
print("Input PIN: ", end="", file=sys.stderr)
oauth.get_access_token(input())

api = tweepy.API(oauth, wait_on_rate_limit=True)

while len(want) > len(have):
    print(len(have), "/", len(want), file=sys.stderr)
    ids = list(want - have)
    if len(ids) > 100:
        ids = ids[:100]
    try:
        tweets = api.lookup_statuses(
            ids, map=True, include_ext_alt_text=True, tweet_mode="extended"
        )
    except tweepy.errors.TwitterServerError as e:
        print(e, file=sys.stderr)
        continue
    have.update(ids)
    for t in tweets:
        want_tweet(t._json)
        json.dump(t._json, sys.stdout)
        print()
