---
layout: post
title: "My remote shell session setup"
date: 2014-03-18 04:08
comments: true
categories: 
---

It's 2014 and I feel entitled to a good experience connecting to a remote server, instead the default still feels like `telnet`.

After searching for quite a long time, I finally built my dream setup. These were the requirements:

* I want a single window/tab/panel of the terminal I'm using to be dedicated to the remote shell (without any new window, etc.)
* I want the shell to survive unaffected with no context loss the following events
	* connection failure
	* route change (like, toggling the VPN or changing Wi-fi)
	* laptop sleep (like, me closing the lid)
	* local terminal restart or laptop reboot
* I want to be able to scroll back with my touchpad
* I want to be able to copy-paste
* I want colors
* I want to launch it with a single command

And a unicorn.

<!-- more -->

(Some fellow travelers in search for the same utopia are [here]())

## The setup

I managed to get this with the following combination: iTerm2 + mosh + tmux.

### iTerm2

The terminal.

I'm on the nightly, but stable should work the same. Just make sure to *Enable xterm mouse reporting* in the *Terminal* Profile settings, and set *Terminal Type* to `xterm-256color`.

### tmux

The session manager.

`tmux` is the new `screen`. It has a ton of features, but I'm using it here just to keep track of my session server side. On 1.8 right now, the one that comes in packages.

`~/.tmux.conf`:

```
new-session
set-window-option -g mode-mouse on
set -g history-limit 30000
```

The first line makes sure that if I try to attach and no sessions are alive, one is created. This means that I can invoke it like this `tmux a` all the time.

The second enables mouse interactions. This will allow us to scroll with the touchpad! (See below)

NOTE: the key combination to detach is `C-b d`.

### mosh

The bridge.

`mosh` is an awesome piece of software. All network-interacting software should behave like it. It will withstand whatever you throw at it from the network. It will even tell you when and since when your connection went down.

Sadly the latest release is ooooold, and doesn't support mouse reporting. So no scrolling. Sigh.

So, you have to build from git.

On OS X: `brew install --HEAD mobile-shell`

On the server:

```
git clone https://github.com/keithw/mosh.git
cd mosh/
sudo apt-get build-dep mosh
./autogen.sh && ./configure && make
sudo make install
```

## Result

The result is that I can type

```
mosh HOST -- tmux a
```

and get my motherfucking shell. Period.

iTerm2 will show me things, `mosh` will make sure that my connection stays up in all the aforementioned cases and `tmux` will keep my scrollback and allow me to detach and reattach. `mosh` and `tmux` collaborating, finally, will allow me to use my dear touchpad. Done.

NOTE: to select text "on the client side", in order to copy/paste, you'll have to hold the Option key.

## Future work

* Scrolling is way less fluid than native. I have no idea how to fix this.
* I'd like click+drag not to be relayed so that I don't have to hold Option to select.