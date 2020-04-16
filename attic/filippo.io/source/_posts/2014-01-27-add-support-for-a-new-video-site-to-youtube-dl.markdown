---
layout: post
title: "Add support for a new video site to youtube-dl"
date: 2014-01-27 02:12
comments: true
categories:
 - "python"
 - "youtube-dl"
---

[youtube-dl](https://github.com/rg3/youtube-dl) is a very feature packed command line video downloader. Contrary to what the name might make you think, it supports way more sites than YouTube. **240** as of [`5700e77`](https://github.com/rg3/youtube-dl/tree/5700e7792aed45d6504ae957610d8254d5bb073f).

What makes this possible is the structure of ytdl and its awesome community: all the common stuff (CLI, Downloading, Postprocessing) is in the core, and websites support is added in a plugin fashion (with a lot of helper functions available). So anyone can add support for its favorite video site by using another plugin as a template, with no need to understand the whole codebase. And a lot of people indeed did: [we're nearing **500 Pull Requests**](https://github.com/rg3/youtube-dl/pulls)!

So, what I'm going to show you today is how to add support to ytdl for a simple site (I picked [Vine](https://vine.co/) for the tutorial) and how to contribute to ytdl in general.

<!-- more -->

## How ytdl is organized

The website plugins are called Information Extractors -- IE -- and their role is clear and simple:

1. they describe what URLs they are able to interpret (with a regex)
2. they get a input URL, usually interact with the site and return a dictionary of information about the video, including its video file URL and its title *(over-simplified)*

You can find IEs in `youtube_dl/extractor`.

The rest of ytdl deals with parsing the input arguments (`youtube_dl/__init__.py`), downloading the file (`youtube_dl.downloader`) and post-processing (`youtube_dl.postprocessor`)

## Let's get started

Of course, if you didn't already `git clone` ytdl GitHub repository and make sure it's up-to-date.

Remove the existing Vine IE if you want to follow along the tutorial step by step

```bash
rm youtube_dl/extractor/vine.py
sed -i '/VineIE/d' youtube_dl/extractor/__init__.py
```

## Anatomy of a IE

We already know that a IE is found in `youtube_dl/extractor`, but how does it look like?

Each site has its own file, named `lowercase_site.py`. Inside it, a subclass of `youtube_dl.extractor.common.InfoExtractor` named `CameCaseSiteIE` is defined.

That subclass has a property, `_VALID_URL`, a regex that defines what URLs will be handled by the IE (a `re.match` is performed) and is usually reused to extract for example the video id.

The only other thing needed is the `_real_extract` method. It takes a URL as its only argument and return a list of dicts, one for each video (usually just one), with *at least* the following fields:

* `id`: a short video id, should be unique for the site, usually it is site-internal
* `url`: the URL of the actual downloadable video file
* `ext`: the extension of the video file
* `title`: the human-readable full title of the video, all characters allowed, Unicode possibly

So, this is how our bare VineIE should start looking like:

```python
from .common import InfoExtractor

class VineIE(InfoExtractor):
    _VALID_URL = r'(?:https?://)?(?:www\.)?vine\.co/.*'

    def _real_extract(self, url):
        return []
```

Finally, each IE is imported inside `youtube_dl/extractor/__init__.py` to be exposed. So, you'll want to add a line like this to that file (please note that the IEs are alphabetically sorted)

```python
from .vine import VineIE
```

Just this line will be enough.

**A note about syntax**: ytdl is a Python2/3 double codebase -- that means, it runs both on Python 2 and Python 3, so be careful to use features and statements that are cross-compatible. You'll find all the compatibility imports already done for you in `youtube_dl.utils`.

## How to run it

Before digging deeper, let's see how to test-run our development ytdl.

Since youtube_dl is a executable Python package, you can run it from inside your working directory like this

```
python -m youtube_dl URL
```

So to run our Vine IE we would use something like

```
python -m youtube_dl vine.co/foo
```

That indeed does not generate any output or error, great.

## Now let's look at Vine

The first thing you want to do is get a bunch of different videos from your target site, and try to spot the differences. In particular, start with the URL pattern and test assumptions about what parts of it are required or optional.

Here is a Vine for you: [`https://vine.co/v/b9KOOWX7HUx`](https://vine.co/v/b9KOOWX7HUx)

The Vine URL pattern is really simple "`https://vine.co/v/VIDEO_ID`" so we can rewrite `_VALID_URL` as:

```python
_VALID_URL = r'(?:https?://)?(?:www\.)?vine\.co/v/(?P<id>\w+)'
```

So we can start doing some useful stuff in `_real_extract`:

```python
mobj = re.match(self._VALID_URL, url)

video_id = mobj.group('id')
webpage_url = 'https://vine.co/v/' + video_id
webpage = self._download_webpage(webpage_url, video_id)
```

`InfoExtractor._download_webpage` downloads a webpage logging progress (this is what `video_id` is used for) and handles errors.

Feel free to add a `print webpage` at the bottom of the function and run with `python -m youtube_dl https://vine.co/v/b9KOOWX7HUx` to check that everything is working.

## The fun part: reversing

Ok, so we have the page HTML and we know what we want to extract, now let's dissect the page to get our file out.

For this I usually turn to Chrome and its Developer Tools. The Network tab is invaluable in identifying what your final goal is, and so what you should be looking for.

However Vine is really friendly, and a simple right-click > Inspect Element on the playing video will be enough

![The video tag](/images/Jack Dorsey's post on Vine 2014-01-27 04-25-35.png)

So, we just have to get the mp4 URL out of the `source` tag. *Tip*: use the Developer Tools to spot what you're looking for, but then build your regex based on the actual page source, as pretty printing WILL get in your way and the live DOM might be substantially different from the source.

A regex like this should fit: `<source src="([^"]+)" type="video/mp4">`

Here comes the next step in our IE:

```python
# Log that we are starting to parse the page
self.report_extraction(video_id)

video_url = self._html_search_regex(r'<meta property="twitter:player:stream" content="(.+?)"', webpage, u'video URL')
```

`InfoExtractor._html_search_regex`, as above, is a helper function that does the boilerplate searching, logging and error handling for you.

Only the title to go. Again, modern pages help: we can piggyback on Facebook-targeted OpenGraph metadata to reliably extract the title

![The OpenGraph tag](/images/Jack Dorsey's post on Vine 2014-01-27 04-37-45.png)

Aaaand, there's a helper for that! The whole `InfoExtractor._og_search_*` suite.

Let's put this last piece in place and return our data

```python
return [{
    'id':        video_id,
    'url':       video_url,
    'ext':       'mp4',
    'title':     self._og_search_title(webpage),
}]
```

**Note**: there are better ways to parse HTML than regexes, but ytdl is Public Domain and self-contained, so using external libraries is not an option.

## Finish

Putting it all together, this should be more or less your final result

```python
import re

from .common import InfoExtractor


class VineIE(InfoExtractor):
    _VALID_URL = r'(?:https?://)?(?:www\.)?vine\.co/v/(?P<id>\w+)'

    def _real_extract(self, url):
        mobj = re.match(self._VALID_URL, url)

        video_id = mobj.group('id')
        webpage_url = 'https://vine.co/v/' + video_id
        webpage = self._download_webpage(webpage_url, video_id)

        # Log that we are starting to parse the page
        self.report_extraction(video_id)

        video_url = self._html_search_regex(r'<meta property="twitter:player:stream" content="(.+?)"', webpage, u'video URL')

        return [{
            'id':        video_id,
            'url':       video_url,
            'ext':       'mp4',
            'title':     self._og_search_title(webpage),
        }]
```

With this few lines of code, you get all the power and the features of ytdl, for a new site!

Now just run it, sit back and enjoy (and test a bunch of videos to be sure!)

```
$ python -m youtube_dl https://vine.co/v/b9KOOWX7HUx
[Vine] b9KOOWX7HUx: Downloading webpage
[Vine] b9KOOWX7HUx: Extracting information
[download] Destination: Chicken.-b9KOOWX7HUx.mp4
[download] 100% of 884.30KiB in 00:00
```

Finally, please [submit a PR](https://github.com/rg3/youtube-dl/pulls) to get your IE included in ytdl. Don't worry, if it downloads, we will be happy to merge it, and if it doesn't, we will be happy to help!

## Ah, add a test

Forgot to mention, ytdl has a complete testing system built in. It is really important that you add a test to your IE before submitting it, as otherwise it would not be possible to do maintenance of so many IEs that break all the time when sites change layout.

Try to write one for each video or URL type.

You just need to add a `_TEST` dict property (or a `_TESTS` list of dicts) looking like this:

```python
_TEST = {
    u'url': u'https://vine.co/v/b9KOOWX7HUx',
    u'file': u'b9KOOWX7HUx.mp4',
    u'md5': u'2f36fed6235b16da96ce9b4dc890940d',
    u'info_dict': {
        u"id": u"b9KOOWX7HUx",
        u"ext": u"mp4",
        u"title": u"Chicken."
    }
}
```

The properties are as follows:

* `url` is the input URL
* `md5` is the md5 hash **of the first 10KB** of the file, to get it download the video with the `--test` flag and run `md5sum` on it
* `info_dict` is just a dict of fields that will be checked against the `_real_extract` return value (missing fields will be ignored)
* <strike>`file` is the filename of the resulting video, with this format "`{id}.{ext}`"</strike> `file` is deprecated, simply add `info_dict.id` and `info_dict.ext`

You can run a single IE test on all the supported Python environments using [tox](https://testrun.org/tox/latest/)

```
$ tox test.test_download:TestDownload.test_Vine
[...]
__________ summary __________
  py26: commands succeeded
  py27: commands succeeded
  py33: commands succeeded
  congratulations :)
```

---

In the next article we will have a look at how to write a IE for a more picky/obfuscated video site.
