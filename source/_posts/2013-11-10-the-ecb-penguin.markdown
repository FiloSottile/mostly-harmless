---
layout: post
title: "The ECB Penguin"
date: 2013-11-10 19:54
comments: true
categories: crypto
---

![Tux ecb.jpg](https://upload.wikimedia.org/wikipedia/commons/f/f0/Tux_ecb.jpg){:.center}

This is an image that has become kind of a cultural icon in the cryptography and InfoSec community. I'm speaking about "the penguin", a picture of the [Tux Linux mascot][tux] encrypted with a block cipher in [ECB mode][ecb] that still shows clearly the outline of the original.

<div class="reset-zoom"><blockquote class="twitter-tweet" data-conversation="none" align="center" data-dnt="true"><p>.<a href="https://twitter.com/solardiz">@solardiz</a> <a href="https://twitter.com/ErrataRob">@ErrataRob</a> ECB mode strikes again, I see. It&#39;s hard to believe there&#39;s anyone left who hasn&#39;t seen the penguin.</p>&mdash; Andrea (@puellavulnerata) <a href="https://twitter.com/puellavulnerata/statuses/396863689602519041">November 3, 2013</a></blockquote>
<script async src="//platform.twitter.com/widgets.js" charset="utf-8"></script></div>

<!-- more -->

![Google suggestions](/images/ecb penguin - Google.png){:.center}

## ECB

You have a cipher, that with a key will encrypt 16 bytes of data. And you have some data, that is more than 16 bytes. So you have a problem. Well, ECB is the wrong solution to that problem: you just encrypt each 16-bytes block separately.

Why is it wrong? Because this way blocks that were equal before encryption will **remain equal** also after! And this will lead to all kinds of unwanted consequences.

One good example is the recent [Adobe passwords crossword game][adobe] but the best visualization of the concept is him, the penguin!

## The original

The [original image][file] has been created by [User:Lunkwill][user] of en.wikipedia in 2004 and added to the page "[Block cipher mode of operation][modes]" with [this edit][diff].

It has even been [proposed as a Wikipedia featured picture][fp].

Nothing more is known about the original. I wrote an email to the author, and I will update the blog post if he replies.

## My take at it

The picture is amazing, but rather low quality even for screen, let alone for printing. So, I decided to generate my own.

First thing needed was an image format where the pixels were represented sequentially as plain bytes, without any compression, and possibly with a simple header. The perfect candidate turned out to be the [PPM binary format][ppm], part of the Netpbm spec. (It is just basically a ASCII header and then a sequence of 3-bytes RGB representations of the pixels.)

Here is the process:

```bash
# First convert the Tux to PPM with Gimp
# Then take the header apart
head -n 4 Tux.ppm > header.txt
tail -n +5 Tux.ppm > body.bin
# Then encrypt with ECB (experiment with some different keys)
openssl enc -aes-128-ecb -nosalt -pass pass:"ANNA" -in body.bin -out body.ecb.bin
# And finally put the result together and convert to some better format with Gimp
cat header.txt body.ecb.bin > Tux.ecb.ppm
```

And the result! (Prints soon on sale, it makes for a great nerdy office decoration, much like "Crypto Safety Procedures")

[![Tux ecb.jpg](/images/Tux-ECB-small.png){:.center}](/images/Tux-ECB.png)

### Bonus: pop art

Also, the color combinations spawning from the different keys reminded me of the [Marilyn Monroe by Andy Warhol][marylin], so... (Canvas prints soon on sale for this, too!)

![POP Tuxes](/images/POP-xsmall.png){:.center}

[marylin]: https://www.google.com/search?q=Marilyn+Monroe+by+Andy+Warhol&tbm=isch
[ecb]: https://en.wikipedia.org/wiki/Block_cipher_mode_of_operation#Electronic_codebook_.28ECB.29
[diff]: https://en.wikipedia.org/w/index.php?title=Block_cipher_mode_of_operation&diff=prev&oldid=2191923
[adobe]: /analyzing-the-adobe-leaked-passwords/
[user]: https://en.wikipedia.org/wiki/User:Lunkwill
[modes]: https://en.wikipedia.org/wiki/Block_cipher_mode_of_operation
[file]: https://en.wikipedia.org/wiki/File:Tux_ecb.jpg
[fp]: https://en.wikipedia.org/wiki/Wikipedia:Featured_picture_candidates/April-2004#Tux_ecb.jpg
[tux]: https://commons.wikimedia.org/wiki/File:Tux.jpg
[ppm]: https://en.wikipedia.org/wiki/Netpbm_format#PPM_example
