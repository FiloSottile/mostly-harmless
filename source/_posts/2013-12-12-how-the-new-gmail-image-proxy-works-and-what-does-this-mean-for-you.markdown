---
layout: post
title: "How the new GMail image proxy works and what does this mean for you"
date: 2013-12-12 17:52
comments: true
categories: 
---

Google [recently announced](http://gmailblog.blogspot.com/2013/12/images-now-showing.html) that images in emails will be displayed automatically by default to GMail users, thanks to an anonymizing proxy operated by them.

This, they say, will actually *benefit* users privacy.

This might very well be true if images are prefetched when an email is received. The [help page](https://support.google.com/mail/answer/145919?p=display_images&rd=1) however does not make it seem like so (and states that images are transcoded, interesting).

Since this feature has already been rolled out to me, I thought to check out how it actually works.

<!-- more -->

So, I set up a slightly modified SimpleHTTPServer to also log request headers (just added the line below)

```python
print json.dumps(self.headers.dict, indent=4, separators=(',', ': '))
```

Downloaded this image and exposed it at `http://filosottile.info/test.png`

![the test image](/images/test.png)

Here how a request from my browser looks like

{% gist 7937352 browser_request %}

Then, I sent the following HTML message to myself at 17:21:29 EST ([here](https://gist.github.com/FiloSottile/7937352#file-full_body) the full email body when received)

{% gist 7937352 message.html %}

It immediately showed up on my phone. No requests. I waited a bit and opened my desktop inbox. No request.

**Then, I opened the email, the image automatically loaded and immediately a request got logged on my server**

{% gist 7937352 on_open %}

The image is indeed transcoded: exact same metadata (format, size...) but different body. Here is it, as got from the URL `https://ci6.googleusercontent.com/proxy/5YvKA8rt5kSAfWUwLZ1LfA_3fBdc2Qr5pHI-aWBr8fg0I27pvkXn5vljroVhYVWBHb5iCIIs=s0-d-e1-ft#http://filosottile.info/test.png`

![the test image](/images/unnamed.png)

And here are the `md5sum` and `identify` outputs

{% gist 7937352 image_files %}

Also, no caching is performed server-side, every time I downloaded that URL, [a request showed up on my server](https://gist.github.com/FiloSottile/7937352#file-other_hits).

## So, what's the issue?

The issue is that the single most useful piece of information a sender gets from you (or the Google proxy) loading the image is **that/when you read the email**. And this is not mitigated at all by this system, as it is only really a proxy and when you open an email the server will see a request. Mix that with the ubiquitous uniquely-named images (images with a name that is unique to an email) and you get read notifications.

Ok, they won't know my IP and this is really good, they won't set tracking cookies to link my different email accounts and they won't know what browser I'm running, they might even fail to exploit my machine thanks to transcoding (if they wanted to waste such a 0-day) but the default setting -- what most users settle on, let's face it -- just got weaker on privacy.

Now, GMail has "✓ Seen".

Note: you can [turn automatic loading off](https://support.google.com/mail/answer/145919?p=display_images&rd=1) and gain the privacy benefits of the proxy anyway.

And you can [follow me on Twitter](https://twitter.com/FiloSottile), too.

## Bonus: the ArsTechnica article

ArsTechnica put out [a terribly un-informed and un-researched article](http://arstechnica.com/information-technology/2013/12/gmail-blows-up-e-mail-marketing-by-caching-all-images-on-google-servers/) that is so full of errors that I'm going to dissect it in reading order.

Starting from the title, *"Gmail blows up e-mail marketing by caching all images on Google servers"*. As you can see, this might even benefit email marketing, for sure not blow it up.

> [...] it will cache all images for Gmail users. Embedded images will now be saved by Google, and the e-mail content will be modified to display those images from Google's cache, instead of from a third-party server.

Simply wrong.

> E-mail marketers will no longer be able to get any information from images—they will see a single request from Google, which will then be used to send the image out to all Gmail users. Unless you click on a link, marketers will have no idea the e-mail has been seen.

We verified that instead this data is alive and kickin', and there is NOT a single request.

> While this means improved privacy from e-mail marketers, Google will now be digging deeper than ever into your e-mails and literally modifying the contents. If you were worried about e-mail scanning, this may take things a step further.

Google always modified the email contents to sanitize HTML and, guess what, to disable images. Also, nothing barred Google from fetching the images in your emails anyway.

> Google servers should also be faster than the usual third-party image host.

All the opposite, as it is a proxy server and NOT a caching server it adds roundtrips to image loading.
