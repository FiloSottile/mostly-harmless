---
layout: post
title: "Please take this and go signing those updates"
date: 2012-12-19 20:47
comments: true
categories: crypto
---
If your program does any sort of self-updating, it is *fundamental* that you **check the update payload integrity**. And no, fetching it over HTTPS might [not](http://docs.python.org/2/library/urllib2.html) [be](http://docs.python.org/3.3/library/urllib.request.html) [enough](http://www.rubyinside.com/how-to-cure-nethttps-risky-default-https-behavior-4010.html).

Otherwise, anyone who can tamper with the traffic of your users, like anyone on their same network, or their ISP, can trivially get **code execution** by modifying the update while your program downloads it. And yes, [it is exploited in the wild and it is easy](http://www.infobytesec.com/down/isr-evilgrade-Readme.txt).

The common way to sign something is to use RSA, but you might not want to rely on *yet another external dependency*, with God knows which license...  
Then, **take this**! It's a drop-in, *zero-dependency* **RSA signature verifying function** that run on Python 2.4+ (seriously) and... it's in the Public Domain ([CC0](http://creativecommons.org/publicdomain/zero/1.0/)), it's yours.

``` python rsa_verify.py https://gist.github.com/4340076#file-rsa_verify-py
def rsa_verify(message, signature, key):
    from struct import pack
    from hashlib import sha256 # You'll need the backport for 2.4 http://code.krypto.org/python/hashlib/
    from sys import version_info
    def b(x):
        if version_info[0] == 2: return x
        else: return x.encode('latin1')
    assert(type(message) == type(b('')))
    block_size = 0
    n = key[0]
    while n:
        block_size += 1
        n >>= 8
    signature = pow(int(signature, 16), key[1], key[0])
    raw_bytes = []
    while signature:
        raw_bytes.insert(0, pack("B", signature & 0xFF))
        signature >>= 8
    signature = (block_size - len(raw_bytes)) * b('\x00') + b('').join(raw_bytes)
    if signature[0:2] != b('\x00\x01'): return False
    signature = signature[2:]
    if not b('\x00') in signature: return False
    signature = signature[signature.index(b('\x00'))+1:]
    if not signature.startswith(b('\x30\x31\x30\x0D\x06\x09\x60\x86\x48\x01\x65\x03\x04\x02\x01\x05\x00\x04\x20')): return False
    signature = signature[19:]
    if signature != sha256(message).digest(): return False
    return True
```

[Here](https://gist.github.com/4340076) are the instructions on how to generate your private and public keys and how to sign new updates. Don't worry, it's all really easy; if you happen to encounter any issues, shoot me a mail at `filippo.valsorda -> gmail.com`!

I am sufficiently proficient only in Python, so if any C, Perl, PHP or Brainfuck guru wants to show up and contribute the same function in another language, it would be awesome!

Now you don't have any excuses anymore (at least you Python devs): **go signing your updates**!  
(And maybe also [following me on Twitter](https://www.twitter.com/FiloSottile))