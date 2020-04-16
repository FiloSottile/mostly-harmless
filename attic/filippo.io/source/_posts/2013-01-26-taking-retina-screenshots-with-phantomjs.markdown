---
layout: post
title: "Taking Retina screenshots with PhantomJS"
date: 2012-05-12 16:44
comments: true
categories: 
 - imported
 - javascript
---

With [PhantomJS](http://phantomjs.org), a headless WebKit browser with Javascript API, you can automatically render a webpage like you see it on your screen in an image or PDF. This is an awesome feature, useful for testing or - that's what I use it for - rendering some elements of the page as images for later use.

Here I will explain how to take Retina-like screenshots. These are screenshots with double width and height for the same element where the details are rendered with double the precision. There are different reasons to want that: you might not own a new iPad or an iPhone4* and want to see how your website would look on these devices or you might want to add a Retina unit test to your awesome test stack. I want to render text to images so that they will still look sharp on Retina screens when used as replacements.

The key is the CSS3 [`transform`][transform] property and its `scale(2)` value, plus a couple of tweaks.
<!--more-->
Here is a modified version of the rasterize.js example to output Retina screenshots.
{% gist 2667199 rasterize.js %}

### Bonus
You might want to render only a single element, for example your content div or your always-buggy sidebar, to an image.  
Well, have a look at [`element.getBoundingClientRect`][getBoundingClientRect] ([getBoundingClientRect is Awesome][awesome]) and PhantomJS [`page.clipRect`][clipRect].

Here is a spoiler ;)
{% gist 2667279 gistfile1.js %}

### References
* [Use PhantomJS to take screenshots of your webapp for you](http://fcargoet.evolix.net/2012/01/use-phantomjs-to-take-screenshots-of-you-webapp-for-you/) - /home/florian
* [Rendering QuickStart example](https://github.com/ariya/phantomjs/wiki/Screen-Capture) - PhantomJs Wiki
* [`render()` API reference](https://github.com/ariya/phantomjs/wiki/API-Reference#wiki-webpage-render) 

[transform]: http://www.w3schools.com/css3/css3_2dtransforms.asp
[getBoundingClientRect]: https://developer.mozilla.org/en/DOM/element.getBoundingClientRect
[awesome]: http://ejohn.org/blog/getboundingclientrect-is-awesome/
[clipRect]: https://github.com/ariya/phantomjs/wiki/API-Reference#wiki-webpage-clipRect
