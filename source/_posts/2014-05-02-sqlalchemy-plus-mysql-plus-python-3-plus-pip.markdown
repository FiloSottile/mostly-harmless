---
layout: post
title: "SQLAlchemy + MySQL + Python 3 + pip"
date: 2014-05-02 00:40
comments: true
categories: python
---

A short PSA/reference blog post: how to make SQLAlchemy work with MySQL on Python 3, with requirements installable via pip.

**tl;dr**: pip `https://launchpad.net/oursql/py3k/py3k-0.9.4/+download/oursql-0.9.4.zip`, connection string `mysql+oursql://` and Unicode everywhere.

This is a common pain of the Python 3 adopter: even if a project supports it, add-ons and testing are falling behind.

SQLAlchemy supports [a number of MySQL interfaces](http://docs.sqlalchemy.org/en/rel_0_9/dialects/mysql.html). Most of them don't work on Python 3 (the common MySQLdb/mysql-python will error out with `ImportError: No module named 'ConfigParser'`) or only come as a DMG (mysql-connector -- ugh).

<!-- more -->

[OurSQL](https://launchpad.net/oursql/) advertises Python 3 support, and seems an elegant/modern implementation, but installing it with pip will fail with an error that smells of Python 3 incompatibility:

```
    print "cython not found, using previously-cython'd .c file."
                                                               ^
SyntaxError: invalid syntax
```

Turns out they maintain [a branch](https://launchpad.net/oursql/py3k/) for Python 3 support, so pointing pip at the zip below will work.

```
https://launchpad.net/oursql/py3k/py3k-0.9.4/+download/oursql-0.9.4.zip
```

A note on Unicode: disabling Unicode or setting a charset simply won't work (they do things as `'foo' in returned_value` that breaks with Python 3 strings). Just use Unicode, it's what you should be doing on Python 3 anyway.