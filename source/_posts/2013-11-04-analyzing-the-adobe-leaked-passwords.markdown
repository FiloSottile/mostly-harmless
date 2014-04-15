---
layout: post
title: "Analyzing the Adobe leaked passwords"
date: 2013-11-04 11:15
comments: true
categories: security
---

![XKCD is on it](https://imgs.xkcd.com/comics/encryptic.png){:.center}

On October Adobe reported that some user data, including credit cards and password dumps, got stolen from their servers. Now the passwords dump has leaked, and it's hilarious.

We (Jari Takkala and I) got hold of the files and are starting to analyze them.

<!-- more -->

## The files

**users.tar.gz** (compressed) - 3.8 GB - e3eda0284c82aaf7a043a579a23a09ce<br>
**cred** (uncompressed) - 9.3GB - 020aaacc56de7a654be224870fb2b516

The 152,982,479 entries are formatted like this

`UID-|--|-EMAIL-|-BASE64 PASSWORD-|-HINT|--`

## The algorithm: four errors

The passwords seem to be encrypted with a 8-bytes block cipher, allegedly 3DES, in ECB mode. This is bad for four main reasons:

* **It is fast**: you don't want a fast algorithm for storing your passwords, you want to make it slow, so that bruteforce is infeasible.

* **It is a block cipher**: this is a complete misuse. Hashing, password strengthening and encryption are different things. Namely, the problem with this are that (A) you need to have access to the cipher password for all the time the system is online, and if that is compromised, **all the passwords can be retrieved at once** (B) you leak passwords lengths

* **It is used in ECB mode**: ECB is evil, as every block of 8 bytes is encrypted separately and you can spot duplicates between 8-character blocks. The XKCD comic refers to this.

* **It is not salted**: this means that duplicate passwords will stand out, but hey, they even went a step further with the point above.

## Cracking

However, the use of a keyed cipher makes cracking the passwords with only a DB dump like this infeasible, even if we can get some nice stats out of it.

But again: it's not secure because it's a keyed cipher. The hacker might have the key for that, something that would allow him to read ALL the passwords, even the strong ones (this can't happen with any proper hashing algorithm) and anyone with a 8-characters block in common with you will now (all or a portion of) your password.

Also, I'm eager to check if they used a strong master password...

## The XKCD

By the way, the comic is not using real data, the first hex block, Base64 encoded is `ThiswasnotY=` :)

## A first manual effort

Jeremi Gosney ([@jmgosney](https://twitter.com/jmgosney)) counted the password repetitions, took the most common ones and then guessed the plaintext either by getting it from one of the users or from the hints. Again: anyone that shares a 8-characters block with your key can recover it.

[http://stricture-group.com/files/adobe-top100.txt](http://stricture-group.com/files/adobe-top100.txt)

We should crowdsource this.

---

This is a rolling blog post, [follow me](https://twitter.com/FiloSottile) on Twitter for updates
