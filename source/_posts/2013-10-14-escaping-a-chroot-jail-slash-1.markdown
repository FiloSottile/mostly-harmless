---
layout: post
title: "Escaping a chroot jail/1"
date: 2013-10-14 10:00
comments: true
categories: 
---

Everybody will tell you that a [chroot jail](https://en.wikipedia.org/wiki/Chroot#Uses) (that is, making a process think that a directory is instead the root folder, and not letting it access or modify anything outside of that) is ineffective against a process with root privileges[^1] (UID 0). Let's see why.

<!-- more -->

The escape basically works like this:

* We create a temporary folder (I named mine `.42`, hidden not to draw too much attention) and we `chroot` to that, this way we make sure our current working directory is outside the fake root, and we can do so because we're <del>CEO</del>root, Bitch[^2];

* then we `chroot` to parent folders all the way up to the root (we don't need to worry about going too up, `/../../.. == /`);

* finally we spawn something, a shell, `rm -rf`, whatever.

Q: Why couldn't we just do `chroot("../../../../../../..")` and call it a day?<br>
A: Because even if the kernel does not want to keep us from doing what we want (we're root, after all) it will keep faith to the chroot also with us and if from inside the chroot jail we ask to `chroot("..")` the kernel will regularly expand `/..` to `/`. It has to do so, some programs might rely on that. So we have to move our working directory outside of the root before proceeding.

{% gist 6976188 unchroot.txt %}

## Other pitfalls

If `chroot()` changes also the working directory to be inside the jail this will make it impossible to pop outside by just chrooting to a sub-directory, but this will not stop us.

We can simply grab the file descriptor of the current directory before the first chroot call and then [`fchdir()`](http://linux.die.net/man/2/fchdir) to that. `chroot()` [does not close file descriptors](http://linux.die.net/man/2/chroot).

Also, if the root privileges were incorrectly dropped, for example by calling [`seteuid()`](http://linux.die.net/man/2/seteuid), a call to `setuid(0)` might be useful in restoring them.

## So, how does a correct chroot look like?

```c
assert(UID > 0);
chdir("jail");
chroot(".");
setuid(UID);
```

And make sure that there are no [`setuid`](https://en.wikipedia.org/wiki/Setuid) binaries inside the jail[^3]!

## A catch-all compile-everywhere C `unchroot`

{% gist 6976188 unchroot.c %}

---

[^1]: Or even just the CAP_SYS_CHROOT privilege (that self-chroot jailing processes often forget to drop), most of the cases we just need to be able to run `chroot()`.
[^2]: [Ahem](http://galeri4.uludagsozluk.com/105/im-ceo-bitch_182484.jpg).
[^3]: `find / -type f \( -perm -4000 -o -perm -2000 \)`
