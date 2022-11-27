import random
import sys
import openai
import mastodon
import os
import requests

hc_uuid = os.environ["HNTITLES_HC_UUID"]

if random.SystemRandom().random() > 0.2:
    requests.post("https://hc-ping.com/" + hc_uuid, timeout=10)
    print("Rolled a skip.")
    sys.exit(0)

openai.api_key = os.getenv("OPENAI_API_KEY")

c = openai.Completion.create(
    model="ada:ft-personal:hntitles-2022-11-26-17-58-24",
    prompt="A plausible Hacker News title:",
    max_tokens=50,
    temperature=0.9,
    stop="END",
)

if c["choices"][0]["finish_reason"] != "stop":
    print(c)
    sys.exit(2)

mastodon = mastodon.Mastodon(
    access_token=os.getenv("MASTODON_ACCESS_TOKEN"),
    api_base_url="https://botsin.space/",
)

post = mastodon.status_post(c["choices"][0]["text"].strip())
requests.post("https://hc-ping.com/" + hc_uuid, data=str(post["id"]), timeout=10)
print(post)
