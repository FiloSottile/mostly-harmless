---
layout: post
title: "Salt &amp; Pepper, please: a note on password storage"
date: 2014-05-28 19:50
comments: true
categories: 
---

Everyone will tell you that the best practice for password storage is [sb]crypt with random salt. Ok, we got that and even maybe got everyone to agree. But let me bump that up a notch: do you know what pepper is?

The concept of peppering is simple: **add a extra fixed, hardcoded salt**.

That is, do something like:

```python
salt = urandom(16)
pepper = "oFMLjbFr2Bb3XR)aKKst@kBF}tHD9q"  # or, getenv('PEPPER')
hashed_password = scrypt(password, salt + pepper)
store(hashed_password, salt)
```

Does this seem useless? Well if you think about it, most password leaks happen because of database leaks (SQL injection, DB credential compromise, DB auth bypass...) and attackers might not necessarily get access to the webserver. In that case, the hashes would be *completely useless*.

Yes, this is not sureproof, attackers might also get access to your webserver, but security is all about layers and raising cost, no? Who knows, maybe the eBay leaked hashes would have been useless to the attackers were they peppered.