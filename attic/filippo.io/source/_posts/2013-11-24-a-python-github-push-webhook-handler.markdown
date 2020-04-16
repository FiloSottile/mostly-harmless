---
layout: post
title: "A Python GitHub Push WebHook Handler"
date: 2013-11-24 19:23
comments: true
categories: python
---

GitHub offers a number of **Service Hooks** that trigger actions when someone pushes to your repository. The generic hook is a simple WebHook that you can easily handle on your server.

There is a official Rack handler somewhere, and maybe a Django one, but nothing in pure Python. So here is it.

It's pretty simple and self-contained, start it with the IP address and port to listen on as arguments, and it will pass a function - `handle_hook()` - the payload received on each push as a Python dictionary. It also checks that the originating IP is actually GH.

Then simply enter the address of your server on the GH Service Hooks repo Admin page, and you're all set.

![The Webhooks admin page](/images/Service Hooks 2013-11-24 00-54-05.png)

For reference on what's inside the payload, [RTFM](https://help.github.com/articles/post-receive-hooks).

{% gist 7634541 HookHandler.py %}
