---
layout: post
title: "Offline storage on a RPi"
date: 2013-02-22 23:21
comments: true
categories: 
---

The only really secure place for data is **an [offline][airgap] computer**.

A couple of use cases: [store your primary **GPG key** offline][subkeys], without expire date, then generate a couple of subkeys with 1y expiration and use them day to day. This way you will need your primary key only to issue/revoke subkeys or to sign other people keys (and no need to waste all your signatures every time your key expire).
Or, [create a cold wallet for **Bitcoin** with Armory][coldwallet] and authorize all transactions from the offline machine while monitoring them from the online one.
And there are much more examples of data best kept offline...

But not everybody has money and space to keep a rarely used computer around only to store a couple of keys.

My solution is to use a **Raspberry Pi** with a *dedicated SD card*. Budget: **10$** for the SD (every good nerd already has a RPi, right?). Space: negligible.

If you don't know what a RPi is, [it is a credit card sized computer with Ethernet, USB, HDMI that costs 35$][RPi]. Now you either want to buy one or you stumbled here by accident.

It's simple: just download [Raspbian][raspbian], **unplug Ethernet**, install Gnupg and [Armory][armorygist] and transfer data with any USB key! *Finish!* And now you have highest grade security on the cheap.

[subkeys]: http://wiki.debian.org/subkeys
[airgap]: https://en.wikipedia.org/wiki/Air_gap_(networking)
[coldwallet]: http://bitcoinarmory.com/using-offline-wallets-in-armory/
[RPi]: http://www.raspberrypi.org/faqs
[raspbian]: http://www.raspbian.org/
[armorygist]: https://gist.github.com/FiloSottile/3646033
