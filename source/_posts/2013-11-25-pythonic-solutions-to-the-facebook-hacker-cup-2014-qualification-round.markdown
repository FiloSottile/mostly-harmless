---
layout: post
title: "Pythonic solutions to the Facebook Hacker Cup 2014 Qualification Round"
date: 2013-11-25 10:52
comments: true
categories: python
---

Facebook organizes this cool competition called the [Hacker Cup][hackercup]. Yesterday the Qualification Round finished, and the user solutions got published. So, since the problems text is under a [CC license][cc] (thanks FB!) I'm publishing here the problems and my answers.

This code pretty much embodies why I love Python: it's clear, fast to write and reads almost like English. When I (thought I) needed speed, I just turned at [Cython][cython] with a few edits to the code.

*NOTE: if for some reason I misunderstood and I wasn't allowed to do this, please get in contact with me ASAP and I'll take this down.*

<!-- more -->

## Square Detector

Read the [problem](https://gist.github.com/FiloSottile/7643628#file-square-detector-md) and check out [the test cases](https://gist.github.com/FiloSottile/7643628#file-square_detector-txt) and [the answer](https://gist.github.com/FiloSottile/7643628#file-square_detector_answer-txt).

This was an easy one, I just scanned the grid until I found a `#`, assumed it was the upper-left corner and counted the following `#` to learn the edge length. At this point I had all the info to build a model of how a correct grid should look like, so I just checked the real one against it.

{% gist 7643628 Square%20Detector.py %}

## Basketball Game

Read the [problem](https://gist.github.com/FiloSottile/7643628#file-basketball-game-md) and check out [the test cases](https://gist.github.com/FiloSottile/7643628#file-basketball_game-txt) and [the answer](https://gist.github.com/FiloSottile/7643628#file-basketball_game_answer-txt).

This is actually my favorite. The problem was fun and the Python code reads as if it was English. It makes hard use of mutable objects and their properties.

{% gist 7643628 Basketball%20Game.py %}

## Tennison

Read the [problem](https://gist.github.com/FiloSottile/7643628#file-tennison-md) and check out [the test cases](https://gist.github.com/FiloSottile/7643628#file-tennison-txt) and [the answer](https://gist.github.com/FiloSottile/7643628#file-tennison_answer-txt).

Finally the hardest one. This was a nice recursive problem. The constrains allowed for a lot of big test cases, so I went a bit overkill with speed, wrote some custom caching, ported my actual recursive function to Cython (it's awesome! Just check out the `-a` HTML output to figure out what you have to optimize and you're done) and made the program parallelizable.

Turns out, memoization would have been enough. Still, it has been really fun!

{% gist 7643628 Tennison.py %}
{% gist 7643628 fast_Tennison.pyx %}


That's all! I got admitted to the next round, so maybe [follow me on Twitter](https://twitter.com/FiloSottile) if you want to read the next batch of problems and solutions!

[hackercup]: https://www.facebook.com/hackercup
[cc]: https://creativecommons.org/
[cython]: http://cython.org/
