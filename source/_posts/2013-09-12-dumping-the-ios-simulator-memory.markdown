---
layout: post
title: "Dumping the iOS simulator memory"
date: 2013-09-12 18:26
comments: true
categories: iOS
---

To audit memory or to debug with external tools it can be useful to get a **dump of the running memory of an app**.

To do so on a device you'll need a Jailbreak, SSH access, and `gdb`. See [this](https://www.soldierx.com/tutorials/iPhone-Dumping-Game-Memory-and-Injecting-Custom-Code-into-Games) or [this](http://rce64.wordpress.com/2013/01/26/decrypting-apps-on-ios-6-single-architecture-no-pieaslr/).

If instead you're up to a simulated app, things are easier: apps running in the simulator are actually just *native processes* on your Mac OS X.

So, how to get a core dump of a Mac OS X process? Sadly gdb [can't do so](http://sourceware.org/gdb/onlinedocs/gdb/Core-File-Generation.html). *Mac OS X Internals* comes to the rescue with [this](http://osxbook.com/book/bonus/chapter8/core/) article.

It is actually an interesting read, but if you are in a hurry, skip to downloading [the code](http://osxbook.com/book/bonus/chapter8/core/download/gcore.c) and compile it like this (screw the Makefile, it compiles also for PowerPC)

    gcc -O2 -arch i386 -Wall -o gcore gcore.c
    gcc -O2 -arch x86_64 -Wall -o gcore64 gcore.c

Then simply run your app, find the process id `grep`-ping `ps -hax` and run

    sudo gcore 1234

And enjoy your core dump. (Bonus: you can load it up in `gdb`)

If you happen to want the dump happen at a particular moment, place a regular breakpoint in XCode, then dump the memory when the process is paused.
