---
layout: post
title: "Archive your GitHub repo and data"
date: 2013-01-14 23:17
comments: true
categories: 
 - python
---
GitHub is a service we all trust, so this is not a "get your data off that cloud before it explodes!"-style post,
but sometimes you want to take an offline copy of your or somebody's work.

Here is a quick and dirty Python script that will help you clone all the repositories, the Gists and some metadata
that can be fetched over the API.
Be warned, it only fetches public repos and data and there's no error checking.

{% gist 4710058 usage.txt %}

{% gist 4710058 archive_GH.py %}

_I wrote and used this to archive Aaron Swartz GitHub account on [archive.org](https://archive.org/details/aaronswGHarchive). R.I.P._
