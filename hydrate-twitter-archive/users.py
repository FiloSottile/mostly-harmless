# Inputs:
#  - direct-messages-group.js
#  - direct-messages.js
#  - block.js
#  - follower.js
#  - following.js
#  - mute.js
#  - tweets.jsonl (from tweets.py)

import tweepy
import json
import sys
import os

want, done = set(), set()

for name in sys.argv[1:]:
    with open(name) as f:
        data = f.read()
        if name.endswith(".jsonl"):
            for i, l in enumerate(data.splitlines()):
                t = json.loads(l)
                if "in_reply_to_user_id_str" in t:
                    want.add(t["in_reply_to_user_id_str"])
                if "user" in t:
                    want.add(t["user"]["id_str"])
                for m in t.get("entities", {}).get("user_mentions", []):
                    want.add(m["id_str"])
        if name.endswith(".js"):
            data = data.split(" = ", maxsplit=1)[1]
            data = json.loads(data)
            for entry in data:
                for attr in ("follower", "following", "muting", "blocking"):
                    if attr in entry:
                        want.add(entry[attr]["accountId"])
                if "dmConversation" in entry:
                    for m in entry["dmConversation"]["messages"]:
                        if "messageCreate" in m:
                            want.add(m["messageCreate"]["senderId"])
                            if "recipientId" in m["messageCreate"]:
                                want.add(m["messageCreate"]["recipientId"])
                        if "joinConversation" in m:
                            want.update(m["joinConversation"]["participantsSnapshot"])
                        if "participantsJoin" in m:
                            want.update(m["participantsJoin"]["userIds"])

want.remove(None)
for id in want:
    if type(id) != str:
        raise TypeError

CONSUMER_KEY = os.getenv("TWITTER_CONSUMER_KEY")
CONSUMER_SECRET = os.getenv("TWITTER_CONSUMER_SECRET")

oauth = tweepy.OAuth1UserHandler(CONSUMER_KEY, CONSUMER_SECRET, callback="oob")

print(oauth.get_authorization_url(access_type="read"), file=sys.stderr)
print("Input PIN: ", end="", file=sys.stderr)
oauth.get_access_token(input())

api = tweepy.API(oauth, wait_on_rate_limit=True)

while len(want) > len(done):
    print(len(done), "/", len(want), file=sys.stderr)
    ids = list(want - done)
    if len(ids) > 100:
        ids = ids[:100]
    try:
        users = api.lookup_users(user_id=ids, tweet_mode="extended")
    except tweepy.errors.TwitterServerError as e:
        print(e, file=sys.stderr)
        continue
    done.update(ids)
    for u in users:
        json.dump(u._json, sys.stdout)
        print()
