# Inputs:
#  - users.jsonl (from users.py)
#
# Outputs to profile_images/ (approx. 50KB each)

import shutil
import requests
import json
import sys
import os
import tqdm


def fetch(url):
    name = url.replace("https://pbs.twimg.com/profile_images/", "")
    name = name.replace("/", "_")
    name = os.path.join("profile_images", name)
    if os.path.exists(name):
        return True
    with open(name + ".part", "wb") as f:
        with requests.get(url, stream=True) as r:
            if r.status_code == 200:
                shutil.copyfileobj(r.raw, f)
                success = True
            else:
                print(r.status_code, r.reason, url)
                success = False
    if success:
        os.rename(name + ".part", name)
    else:
        os.remove(name + ".part")
    return success


for name in sys.argv[1:]:
    with open(name) as f:
        data = f.read()
        if name.endswith(".jsonl"):
            for i, l in enumerate(tqdm.tqdm(data.splitlines())):
                u = json.loads(l)
                if u["default_profile_image"]:
                    continue
                if not u["profile_image_url_https"]:
                    continue  # Twitter Media Policy violations
                if not fetch(u["profile_image_url_https"].replace("_normal", "")):
                    fetch(u["profile_image_url_https"])
