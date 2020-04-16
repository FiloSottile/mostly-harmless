---
layout: post
title: "GitHub: checkout a pull request as a branch"
date: 2013-01-01 06:36
comments: true
categories: 
external-url: https://coderwall.com/p/z5rkga
---

Today looking at the Travis log of a Pull request build I saw this interesting command:

```
git fetch origin +refs/pull/611/merge:
```

Turns out that GitHub makes available from your main remote the PR branches as remote refs.

Also (discovered by blind guessing), if you change `/merge` with `/head` you get a ref to the clean PR head, unmerged with its target branch. What can be the most useful is up to you, I guess.

This is probably easy because GH on its side stores all the forks of a repo as the same Git repository.

An example in the Coderwall ProTip linked at the title.
