---
layout: post
title: "Automatically compile .less and .coffee files"
date: 2013-01-12 00:25
comments: true
categories: 
---

This small python script makes use of [`watchdog`][w] (and [`sh`][s]) to monitor your code directory (recursively) and build [less][l] and [CoffeeScript][c] files upon edit.

Simply launch it from the relevant folder and it will work in the background.

It should be trivial to add minification (and linting, but I suggest linting in the editor) to the process.

[w]: http://packages.python.org/watchdog/
[s]: http://amoffat.github.com/sh/
[l]: http://lesscss.org/
[c]: http://coffeescript.org

```python
#!/usr/bin/env python2

import watchdog.events
import watchdog.observers
import sh
import time
import os

# Detach
if os.fork(): os._exit(0)

coffee = sh.coffee.bake('-c')
less = sh.lessc

class Handler(watchdog.events.PatternMatchingEventHandler):
    def __init__(self):
        watchdog.events.PatternMatchingEventHandler.__init__(self, patterns=['*.less', '*.coffee'],
            ignore_directories=True, case_sensitive=False)

    def on_modified(self, event):
        if event.src_path.lower().endswith('.less'):
            less(event.src_path, event.src_path[:-5] + '.css')
        if event.src_path.lower().endswith('.coffee'):
            coffee(event.src_path)

    on_created = on_modified

if __name__ == "__main__":
    event_handler = Handler()
    observer = watchdog.observers.Observer()
    observer.schedule(event_handler, path='.', recursive=True)
    observer.start()
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        observer.stop()
    observer.join()
```

It requires `coffee` (`npm install coffee-script`) and `lessc` (`npm install less`).

Should be compatible with Mac OS X and Linux at least, BSD and Win... maybe.
