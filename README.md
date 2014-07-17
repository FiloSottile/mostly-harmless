# ar.chiv.io
**A work in progress.** The design is being drafted right now. Watch the repository and [come shape the project](https://github.com/FiloSottile/ar.chiv.io/issues/2)! We'll use [the Issues section](https://github.com/FiloSottile/ar.chiv.io/issues/) as a mailing list.

**I'm looking for developers to join me!**

[Here](https://draftin.com/documents/385453?mode=presentation&token=vJDa78r0Ku2JsGeFHbw2LF-WEnkM1CntBHYa7QXnxxeA6joZ6KuUnzV7uKyls3s9paSgntlisg9ItFStbFTEST0) is a quick presentation on the project.

## Goals

Produce a **permanent**, *future-proof*, **faithful** archive of all the web pages you visit.

* Archived pages should be as static as possible, but they should include dynamic content as loaded in a browser. À la [archive.today](http://archive.today).
* There should be broad support for plug-ins, both JS injected in pages and for the backend.
* Storage should be permanent, deduplicated and indexed.

## How

Here are just some ideas on how to build it. Have a better idea? [Share it](https://github.com/FiloSottile/ar.chiv.io/issues/2)!

* A local browser extension should store the URLs
* The browser-loading part should probably happen in PhantomJS or Selenium.
* Then JS plugins should be injected.
* Then CSS should be embedded, forms neutralized, JS removed, links edited (?), images fetched and DOM snapshotted.
* Then the result should be stored encrypted and deduplicated Tarsnap style.

## Plug-in ideas

* Auto scroll to load more content
* Expand comments (Reddit, ...)
* Dismiss pop-ups (Quora, ...)
* youtube-dl for supported sites
* Blacklist sites - regexes
* Fetch whole site (Readthedocs, ...)
* Repo cloning (GitHub, ...)