---
layout: post
title: "Send a HEAD request in Python"
date: 2012-03-18 17:53
comments: true
categories: 
 - imported
---

There are a lot of questions on this topic around the web and common answers are to use `httplib`, that however is a really-low level library, or to use `urllib2`, but a lot of people complains about it returning to `GET` if following a redirect.

Here is my `urllib2` solution, written looking at the code of `urllib2.HTTPRedirectHandler` and subclassing it in order to make it keep using the `HeadRequest`.

{% gist 2077204 HEAD-request.py %}

For example, here is a fast URL un-shortener (redirect follower) realized with the method above (and a fallback).

{% gist 2077115 redirect-follower.py %}
