---
layout: post
title: "Archive your GitHub repo and data"
date: 2013-01-14 23:17
comments: true
categories: 
---
GitHub is a service we all trust, so this is not a "get your data off that cloud before it explodes!"-style post,
but sometimes you want to take an offline copy of your or somebody's work.

Here is a quick and dirty Python script that will help you clone all the repositories, the Gists and some metadata
that can be fetched over the API.
Be warned, it only fetches public repos and data and there's no error checking.

```
usage: gh_dump.py [-h] [--forks] [--no-gist] [--no-metadata] username

Dump an user's public GitHub data into current directory.

positional arguments:
  username       the GH username

optional arguments:
  -h, --help     show this help message and exit
  --forks        git clone also forks (default is don't)
  --no-gist      don't download user gists (default is do)
  --no-metadata  don't download user metadata (default is do)
```

```python
#!/usr/bin/env python3

# This is free and unencumbered software released into the public domain.

# Anyone is free to copy, modify, publish, use, compile, sell, or
# distribute this software, either in source code form or as a compiled
# binary, for any purpose, commercial or non-commercial, and by any
# means.

# In jurisdictions that recognize copyright laws, the author or authors
# of this software dedicate any and all copyright interest in the
# software to the public domain. We make this dedication for the benefit
# of the public at large and to the detriment of our heirs and
# successors. We intend this dedication to be an overt act of
# relinquishment in perpetuity of all present and future rights to this
# software under copyright law.

# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
# EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
# MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
# IN NO EVENT SHALL THE AUTHORS BE LIABLE FOR ANY CLAIM, DAMAGES OR
# OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
# ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
# OTHER DEALINGS IN THE SOFTWARE.

# For more information, please refer to <http://unlicense.org/>

import argparse
from urllib.request import urlopen
from subprocess import call
import json
import re
import os.path

parser = argparse.ArgumentParser(description='Dump an user\'s public GitHub data into current directory.')
parser.add_argument('user', metavar='username',
                   help='the GH username')
parser.add_argument('--forks', dest='forks', action='store_true',
                   help='git clone also forks (default is don\'t)')
parser.add_argument('--no-gist', dest='gists', action='store_false',
                   help='don\'t download user gists (default is do)')
parser.add_argument('--no-metadata', dest='metadata', action='store_false',
                   help='don\'t download user metadata (default is do)')

args = parser.parse_args()

def clear_url(url):
    return re.sub(r'\{[^\}]*\}', '', url)

data = urlopen('https://api.github.com/users/' + args.user).read()
user = json.loads(data.decode('utf-8'))
if args.metadata:
    with open('user.json', 'wb') as f:
        f.write(data)

data = urlopen(clear_url(user['repos_url'])).read()
repos = json.loads(data.decode('utf-8'))
if args.metadata:
    with open('repos.json', 'wb') as f:
        f.write(data)
for repo in repos:
    if not repo['fork']:
        call(['git', 'clone', repo['clone_url']])
    elif args.forks:
        if not os.path.exists('forks'):
            os.makedirs('forks')
        call(['git', 'clone', repo['clone_url'], os.path.join('forks', repo['name'])])

data = urlopen(clear_url(user['gists_url'])).read()
gists = json.loads(data.decode('utf-8'))
if args.metadata:
    with open('gists.json', 'wb') as f:
        f.write(data)
if args.gists:
    if not os.path.exists('gists'):
        os.makedirs('gists')
    for gist in gists:
        call(['git', 'clone', gist['git_pull_url'], os.path.join('gists', gist['id'])])

if args.metadata:
    for name in ['received_events', 'events', 'organizations', 'followers', 'starred', 'following', 'subscriptions']:
        data = urlopen(clear_url(user[name + '_url'])).read()
        with open(name + '.json', 'wb') as f:
            f.write(data)

```

_I wrote and used this to archive Aaron Swartz GitHub account on [archive.org](https://archive.org/details/aaronswGHarchive). R.I.P._
