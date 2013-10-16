---
layout: post
title: "Customizing a Linode kernel"
date: 2013-10-15 23:15
comments: true
categories: 
---

I'm trying to compartmentalize my Linode server with [Docker](http://docker.io), and so I'll need a **3.8+ 64-bit kernel with AUFS** support[^1]. Ok.

My old Linode was 32-bit, but using the Dashboard and the doubled storage Linode just upgraded me to I was able to add a **Ubuntu 12.04 64-bit Configuration Profile and Disk Image**, reboot to that and mount the old disk image to copy files over. So far so good.

The current Linode kernel is a custom **3.9.3**. Nice. But without **AUFS support**, ouch. Ok then, I'll need to recompile this thing.

Also, `lxc-checkconfig` tells me that I miss support for a lot of things, so...

NOTE: There are easy tutorials[^2] telling you to use the vendor provided kernels, but I feel like there is a reason if Linode ships his own custom kernel, so I really want to just customize theirs.

<!-- more -->

## Getting the source and putting the config in place

The Linode feature that allows us to load our own module is [PV-GRUB](http://wiki.xen.org/wiki/PvGrub) and [here](https://library.linode.com/custom-instances/pv-grub-custom-compiled-kernel) is the Linode Library article about that, keep it open for reference.

First, have a look at what kernel branch your box is currently running and download the tarball of its source from [kernel.org](https://www.kernel.org):

```
$ uname -a
Linux li593-45 3.9.3-x86_64-linode33 #1 SMP Mon May 20 10:22:57 EDT 2013 x86_64 x86_64 x86_64 GNU/Linux
$ aria2c https://www.kernel.org/pub/linux/kernel/v3.x/linux-3.9.11.tar.xz
[...]
$ tar xvf linux-3.9.11.tar.xz
[...]
$ cd linux-3.9.11
```

Now we will extract the config from the running Linode kernel and update it in case there's need.

```
$ zcat /proc/config.gz > .config
$ make oldconfig
```

## Mixing AUFS in[^3]

I'll go fast over this, as it's almost off-topic. You can skip to the next heading if you are not interested.

```
$ git clone git://git.code.sf.net/p/aufs/aufs3-standalone aufs3-standalone.git
$ cd aufs3-standalone.git
$ git checkout origin/aufs3.9
$ cd ../linux-3.9.11/
$ patch -p1 < ../aufs3-standalone.git/aufs3-kbuild.patch
$ patch -p1 < ../aufs3-standalone.git/aufs3-base.patch
$ patch -p1 < ../aufs3-standalone.git/aufs3-proc_map.patch
$ patch -p1 < ../aufs3-standalone.git/aufs3-standalone.patch
$ cp -a ../aufs3-standalone.git/{Documentation,fs} .
$ cp -a ../aufs3-standalone.git/include/uapi/linux/aufs_type.h include/uapi/linux/
$ cp -a ../aufs3-standalone.git/include/linux/aufs_type.h include/linux/
```

## Compiling

Great, finally we do our customizations to the config with `make menuconfig` (you'll need `libncurses5-dev`) and compile. (I enabled AUFS in Misc filesystems and the things listed in the `lxc-checkconfig` source code)

Ah, you might want to change the name of the kernel to something like `3.9.11-custom`. You can do that by editing the following `Makefile` line like this:

    EXTRAVERSION = -custom

```
$ make
# make modules_install
# make install
```

## Installing

```
# apt-get install grub-legacy-ec2
# sed -i 's/indomU=true/indomU=false/' /boot/grub/menu.lst
# update-grub-legacy-ec2
```

And that's it! Now go to the **Linode Manager**, edit your Configuration Profile to use *pv-grub-x86_64* as the "Kernel" and reboot.

You should be able to verify what you are running with `uname -a`, and if you need to see/interact with the boot process, the **Lish console** is like being in front of a screen. Have fun! (And why did we start in the first place...? Ah, Docker!)

```
filosottile@li593-45:~$ uname -a
Linux li593-45 3.9.11-custom #3 SMP Tue Oct 15 19:57:48 UTC 2013 x86_64 x86_64 x86_64 GNU/Linux
```

NOTE: make sure that the first kernel listed in `/boot/grub/menu.lst` is your new one, as PV-GRUB boots the first kernel of the list and `make install` backups existing kernels to `*.old` copies, and these get positioned first by `update-grub`. I had a Linode blow up all over my face because of this.

[^1]: [Kernel Requirements - Docker Documentation](http://docs.docker.io/en/latest/installation/kernel/)
[^2]: [Install Docker on Linode (Ubuntu 12.04)](http://coder1.com/node/87)
[^3]: [http://aufs.sourceforge.net/](http://aufs.sourceforge.net/)
