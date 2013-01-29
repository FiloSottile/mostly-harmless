---
layout: post
title: "Decompressing Lempel-Ziv-Stac in Python"
date: 2013-01-29 12:06
comments: true
categories: 
---
Lempel-Ziv-Stac is a simple (and a bit exotic) compression algorithm,
used on embedded devices, for example for config files, for example on routers,
for example on those that expose the config file on the public internet. Just sayin'...

There is not a Python implementation of it, so here is my Lempel-Ziv-Stac decompression routine.
<!-- more -->
{% gist 4663892 %}
