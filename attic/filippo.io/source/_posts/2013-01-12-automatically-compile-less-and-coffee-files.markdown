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

{% gist 4710041 watch_and_build.py %}

It requires `coffee` (`npm install coffee-script`) and `lessc` (`npm install less`).

Should be compatible with Mac OS X and Linux at least, BSD and Win... maybe.
