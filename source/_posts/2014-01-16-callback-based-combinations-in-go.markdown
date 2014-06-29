---
layout: post
title: "Callback-based combinations (in Go)"
date: 2014-01-16 15:20
comments: true
categories: golang
---

Let's have a look at this task: generate all the **combinations of *k* elements out of *N***. That is, all the *unordered k-tuples* made of elements from a pool of length N. [See Wikipedia](https://en.wikipedia.org/wiki/Combination) for more.

An example of an application: if you are bruteforcing two misspellings in a password, the optimal set of couples of characters to bruteforce at the same time is the set of all the combinations of 2 characters out of the password.

There are a number of ways you can do this in code. Algorithm-wise you have to choose between a recursive approach and an iterative one. The recursive one might be more immediate for some people, but it does. Not. Scale. (*Recursion limit reached* anyone?) Also in some languages **function calls are really expensive**.

However what this article is about is how to grab the output. First you have to decide whether to return *k*-tuples of indices in 0 -- N-1 or of actual pool elements.

For example with a pool of elements of `qwerty` and a *k* of 2, you can decide to return values like `(q, t)` and `(t, y)` or `(0, 4)` and `(4, 5)`.

<!-- more -->

My opinion is that you should always prefer the indices:

* to return actual elements you have to pass the pool to the algorithm;
* you can reuse a set of indices over two same-size pools;
* for some tasks you can avoid the work of extracting the values from the pool by index all the times (e.g. if you filter them);
* by returning elements you lose information about their index that might be irrecoverable (if there are duplicates in the pool) or expensive to recover (`O(N)`);
* sometimes, well, you just need the indices.

Then, you can think to a number of approaches here:

1. just return an array or a set of all the combinations
2. yield them (if you have support for generators)
3. call a callback on each one
4. plainly process them where you generate them

I prefer by far the callback approach:

* it is supported in much more languages than generators;
* can be used elegantly and succinctly with anonymous functions;
* **doesn't require `k*N` memory**, you can just filter or process them on the fly;
* you can build any other approach over it, e.g. by passing a `append` function as the callback;
* by using closures you can share the callee scope;
* keeps your code [DRY](https://en.wikipedia.org/wiki/Don%27t_Repeat_Yourself).

So, code! Here are the Go snippets for combinations with and without repetitions. Most of it was translated to Go from [the Python documentation](http://docs.python.org/2/library/itertools.html#itertools.combinations) and adapted according to the contents of this article.

{% gist 8463644 combinations.go %}

{% gist 8463644 combinations_with_replacement.go %}