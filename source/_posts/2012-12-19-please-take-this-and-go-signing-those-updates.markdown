---
layout: post
title: "Please take this and go signing those updates"
date: 2012-12-19 20:47
comments: true
categories: crypto
---
If your program does any sort of self-updating, it is *fundamental* that you **check the update payload integrity**. And no, fetching it over HTTPS might [not](http://docs.python.org/2/library/urllib2.html) [be](http://docs.python.org/3.3/library/urllib.request.html) [enough](http://www.rubyinside.com/how-to-cure-nethttps-risky-default-https-behavior-4010.html).

Otherwise, anyone who can tamper with the traffic of your users, like anyone on their same network, or their ISP, can trivially get **code execution** by modifying the update while your program downloads it. And yes, [it is exploited in the wild and it is easy](http://www.infobytesec.com/down/isr-evilgrade-Readme.txt).

The common way to sign something is to use RSA, but you might not want to rely on *yet another external dependency*, with God knows which license...  
Then, **take this**! It's a drop-in, *zero-dependency* **RSA signature verifying function** that run on Python 2.4+ (seriously) and... it's in the Public Domain ([CC0](http://creativecommons.org/publicdomain/zero/1.0/)), it's yours.

{% gist 4340076 rsa_verify.py %}

[Here](https://gist.github.com/4340076) are the instructions on how to generate your private and public keys and how to sign new updates. Don't worry, it's all really easy; if you happen to encounter any issues, shoot me a mail at `filippo.valsorda -> gmail.com`!

I am sufficiently proficient only in Python, so if any C, Perl, PHP or Brainfuck guru wants to show up and contribute the same function in another language, it would be awesome!

Now you don't have any excuses anymore (at least you Python devs): **go signing your updates**!  
(And maybe also [following me on Twitter](https://www.twitter.com/FiloSottile))
