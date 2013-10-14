---
layout: post
title: "Escaping a chroot jail/1"
date: 2013-10-14 10:00
comments: true
categories: 
---

Everybody will tell you that a [chroot jail](https://en.wikipedia.org/wiki/Chroot#Uses) (that is, making a process think that a directory is instead the root folder, and not letting it access or modify anything outside of that) is ineffective against a process with root privileges (UID 0). Let's see why.

<!-- more -->

{% gist 6976188 unchroot.txt %}
