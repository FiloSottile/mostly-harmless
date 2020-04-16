---
layout: post
title: "Native scrolling and iTerm2"
date: 2014-03-13 03:15
comments: true
categories: 
---

**tl;dr** See the bullet points for the supported programs and the last paragraph for installation.

Something I always wanted is native touchpad/mousewheel scrolling in all my terminal programs.

[MouseTerm](https://bitheap.org/mouseterm/) hacks that into the OS X Terminal, but I am a iTerm2 user.

I tried and gave up researching this a while ago, but today I got a notification from a Google Code bug I starred linking to [this](https://code.google.com/p/iterm2/issues/detail?id=974). Someone actually patched support for this a while ago, and someone else now updated the patch for current git!

<!-- more -->

The patch worked like a charm. It basically send arrow keystrokes on mousewheel when the terminal is in alternate mode. The actual logic amounts to this, reworked by me:

```obj-c
case MOUSE_REPORTING_NONE:
    if ([[PreferencePanel sharedInstance] alternateMouseScroll] &&
        [_dataSource isAlternate]) {
        CGFloat deltaY = [event deltaY];
        NSData* keyMove;
        if (deltaY > 0) {
            keyMove = [terminal.output keyArrowUp:[event modifierFlags]];
        } else if (deltaY < 0) {
            keyMove = [terminal.output keyArrowDown:[event modifierFlags]];
        }
        for (int i = 0; i < ceil(fabs(deltaY)); i++) {
            [_delegate writeTask:keyMove];
        }
        return;
    }
```

I tested and confirmed compatibility with:

* `less`
* `vim`
* `screen` (after the `C-a ESC` escape - `ESC` to exit)
* `tmux` (after the `C-b [` escape - `q` to exit)
* all of the above over `ssh` and `mosh`

In particular the point about `mosh` and `screen` makes me happy, since this allows me to use them together to get session resuming and native scrollback - fixing [what annoyed me (and others) most of mosh](https://github.com/keithw/mosh/issues/122).

I took the patch, wrapped it in a hidden (not exposed) setting, and submitted as a [Pull Request](https://github.com/gnachman/iTerm2/pull/164). iTerm2 author was quick to suggest changes to the code and then to merge.

By the way, iTerm2 builds so pleasantly with a simple run of `xcodebuild`!

This means that it should be in the Nightly builds from tomorrow. To activate it just run

```bash
$ defaults write com.googlecode.iterm2 AlternateMouseScroll -bool true
```

