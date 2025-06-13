import tweepy
import json
import sys
import os

users = {}

with open("users.jsonl") as f:
    data = f.read()
    for i, l in enumerate(data.splitlines()):
        u = json.loads(l)
        users[u["id_str"]] = u

total = 0

with open("data/following.js") as f:
    data = f.read()
    data = data.split(" = ", maxsplit=1)[1]
    data = json.loads(data)
    for entry in data:
        u = users.get(entry["following"]["accountId"])
        if not u:
            print(0, entry["following"]["accountId"])
            continue
        if (
            u["screen_name"] == "ReciteSocial"
            or "everyh" in u["screen_name"].lower()
            or "hourly" in u["screen_name"].lower()
        ):
            continue
        print(u["statuses_count"], "@" + u["screen_name"])
        total += u["statuses_count"]

print(total)
