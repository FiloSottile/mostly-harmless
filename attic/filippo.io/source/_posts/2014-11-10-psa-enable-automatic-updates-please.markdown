---
layout: post
title: "PSA: enable automatic updates. Please."
date: 2014-11-10 15:32
comments: true
categories: 
---

I want you to do a quick inventory of all the boxes, VPS, servers etc. you have root on.

Ok, now tell me, when is the last time you updated the one you almost forgot about? Is it vulnerable to ShellShock? Is it vulnerable to Heartbleed?

Go patch it now, I'll wait.

Now, **turn on automatic security updates on all the boxes you don't log into at least every few days**. (If I convinced you already, just skip at the bottom of this post to read how.)

It does not matter if you don't care about those boxes. They WILL get owned and [turned into a botnet](http://status.ovh.net/?do=details&id=8120) that will make all of us on the Internet less secure. It's a **responsibility** you have for managing a server on our Internet, together with making sure your mail server is not an open remailer and your DNS server can't be used for DDoS reflection.

*"But Filippo, automatic updates are going to break my box!"*

No. Distribution security updates are MEANT not to break things. And trust me, not patching security vulnerabilities is going to disrupt your service way sooner than a breaking update (if that ever happens).

*"But my box can't reboot cleanly and resume service"*

This is bad, there are countless things that can reboot your box, host mainteinance being the most likely, followed by kernel panics, out-of-memory... It's just part of the mindless server setup having things start on boot.

Anyway, you can turn off automatic reboots and still get 70% of the benefits (maybe).

*"Ok you sold it, how do I do it?"*

Easy-peasy. Here are the instructions if you use Ubuntu (and I think it works also on Debian). If you know how to do it on other systems please email me!

**`/etc/apt/apt.conf.d/20auto-upgrades`**

```
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
```

**`/etc/apt/apt.conf.d/50unattended-upgrades`**

```
// Automatically upgrade packages from these (origin:archive) pairs
Unattended-Upgrade::Allowed-Origins {
        "${distro_id}:${distro_codename}-security";
};

// List of packages to not update (regexp are supported)
Unattended-Upgrade::Package-Blacklist {};

// Send email to this address for problems or packages upgrades
// If empty or unset then no email is sent, make sure that you
// have a working mail setup on your system. A package that provides
// 'mailx' must be installed. E.g. "user@example.com"
Unattended-Upgrade::Mail "TODO_YOUR_EMAIL_HERE_TODO";

// Set this value to "true" to get emails only on errors. Default
// is to always send a mail if Unattended-Upgrade::Mail is set
Unattended-Upgrade::MailOnlyOnError "true";

// Automatically reboot *WITHOUT CONFIRMATION*
//  if the file /var/run/reboot-required is found after the upgrade
Unattended-Upgrade::Automatic-Reboot "true";
```

```
# apt-get install unattended-upgrades
# service unattended-upgrades restart
```
