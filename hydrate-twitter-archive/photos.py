# Inputs:
#  - tweets.jsonl (from tweets.py)
#
# Outputs to media/ (photos and previews, no videos)

import shutil
import requests
import json
import sys
import os
import tqdm


for name in sys.argv[1:]:
    media = []
    with open(name) as f:
        data = f.read()
        for i, l in enumerate(data.splitlines()):
            t = json.loads(l)
            if not "extended_entities" in t:
                continue
            for m in t["extended_entities"]["media"]:
                media.append(m)
    for m in tqdm.tqdm(media):
        url = m["media_url_https"]
        ext = url.split(".")[-1]
        if not "ext_tw_video_thumb" in url:
            url = url + ":orig"
        name = os.path.join("media", m["id_str"] + "." + ext)
        if os.path.exists(name):
            continue
        with open(name + ".part", "wb") as f:
            with requests.get(url, stream=True) as r:
                if r.status_code == 200:
                    shutil.copyfileobj(r.raw, f)
                    success = True
                else:
                    print(r.status_code, r.reason, m["expanded_url"])
                    success = False
        if success:
            os.rename(name + ".part", name)
        else:
            os.remove(name + ".part")
