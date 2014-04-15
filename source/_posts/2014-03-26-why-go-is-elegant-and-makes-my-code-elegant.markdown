---
layout: post
title: "Why Go is elegant and makes my code elegant"
date: 2014-03-26 18:16
comments: true
categories: Golang
---

This is a enthusiast blog post. I'm not even gonna speak about how concurrency comes easy with Go. Honestly, I'm not good enough to speak about it. I'll just speak about how using Go in my everyday programming makes me happy.

Go has elegance, good tools and has had the possibility to start designing from scratch.

The following should not be a list but a tree, since good design decisions enable other good design decision. But I'll go over them as they come to my mind.

### Every package has a unique name in the universe
If you write a package, it will have a name. A unique one. In all the universe. Like, `github.com/FiloSottile/TripleSec` or `filippo.io/foobar`.

This is **awesome**. First and most obviously, `import` statements are sure, clear things. You import packages by their full name and you know what you are importing. No need for `bundler` or `requirements.txt`.

Second, you can keep a tree of the packages you use or develop on your disk, and it remains tidy by itself. You just put them in `$GOPATH/THEIR/FULL/NAME` and both you and the toolkit always know where they are.

Finally, most of the times the name will tell you *and your toolkit* where to get that package. If the package name starts with *bitbucket.com*, *github.com* or *code.google.com* then you just need to run `go get NAME` and it will get downloaded for you. Automatically.

<!-- more -->

### GoDoc

Go has a [clear, easy, multi-output documentation style][doc_blog]. You just write comments and there's a tool that will write txt, man pages and HTML for you.

Mix this and the above and you get [godoc.org](http://godoc.org). The site will index all the packages it can crawl, generate docs for them and provide them to you at *http://godoc.org/PACKAGE_NAME*. All of the docs, in the same place, automatically. **You literally just have to write comments and push and it's like you registered the package on PyPi, generated the docs and uploaded them**.

[doc_blog]: http://blog.golang.org/godoc-documenting-go-code

### Compiling and static binaries (+ with easy cross-compiling)

Go is compiled. This has a number of advantages, first being all the errors that can be detected by the compiler, and speed. But usually there's a tradeoff: Makefiles get messy, dynamic libraries requirements on the target machines.

Go solves all these: there are no Makefiles at all (or requirements.txt or metadata whatsoever), just imports and at most build inclusion/exclusion statements at top of the file; the binaries are static.

You just compile a binary with the libraries you have on your system (see point 1) and ship it to the machine that runs it. No dependencies. No interpreter versions.

Or if you just want to solve that Project Euler problem, `go run solution.go`. I don't miss Python anymore.

And by mixing good design, good tools and good community, you get [cross-compilation for free][cross_tool].

[cross_tool]: https://github.com/laher/goxc

### Static types, that I have to type only once

Static types are good, if they don't slow me down. Go simply figured out that you just have to define them for functions parameters and return values. Think about it, (almost) all variables are parameters, receive a return value from a function or are defined as an explicit value.

When I happen to explicitly define a type, I'm probably doing something complex enough that static types are only gonna be helpful anyway.

### A unique, documented style

Go code has an official style. It's part of the language. It's documented. And patterns are made explicit in the [blog posts][blog_posts], that are committed to the same repositories as the Go tools.

And also here there's an awesome tool. `go fmt` will take your source code and format it according to the style. No more "personal preferences" and discussions.

This makes reading Go code easier, whoever wrote it and whatever the project.

<!-- As Linus Torvalds put it in the kernel [`CodingStyle`][codingstyle]: -->

[blog_posts]: http://blog.golang.org/
[codingstyle]: https://www.kernel.org/doc/Documentation/CodingStyle

### Pleasant conventions

Go has some conventions that affect the behavior of your programs. These are easy and natural, and simplify the language without adding confusion.

* Anything that starts with an upper case letter is exported. Everything else is not.
* If you have tests and benchmarks (you better have them, Go makes it so easy to write them), you put them in `*_test.go`.
* You write some Assembly, just put it in a file named like `*_amd64.s`

### Control flow is explicit and in your hands

There are no exceptions that can traverse your code up to who knows where. Functions return a error, you handle it, or just return it.

Tests follow the same philosophy: you explicitly check for errors or expected states and `.Fail` the test if you want.

### Even if it's young it has a decent library

I wrote [Golang TripleSec][triplesec] using only standard and `go.crypto` libraries. And they are fast and pretty. (Ok, I'm cheating, there's Adam writing Go crypto libs, but also the rest is good!)

Also, thanks to point 1, it's seamless to use external libraries, obviously.

[triplesec]: https://github.com/FiloSottile/TripleSec/

### There's no pointer arithmetics but no flexibility is lost

Go has no low-level memory arithmetics, but pointers, `&` and `*` are still there for your usual pass-by-pointer needs. This alone kills a lot of complexity and potential bugs.

However flexible array pointers are still there, on disguise: they are called **slices**, actually just a struct of a pointer, a own length and a length of the underlying allocated memory (*capacity*). The built-in length means that APIs don't need to explicitly pass it around and built-in capacity means no more silent overflows and segfaults. A bunch of built-ins come bundled to seamlessly handle length, capacity, copying and reallocation.

It's how C arrays should have always worked if performance wasn't prioritized over simplicity, security... everything, actually.

### It made the Heartbleed checker possible

A new one: thanks to the crypto/tls library I wrote [the Heartbleed checker](http://filippo.io/Heartbleed/) in one hour, plus a couple for the web side (but I'm just a bad web developer).

Then rewriting the backed entirely in Go allowed me to scale to 12,000 requests a minute over 40 machines, each of them requiring me to open a HTTPS connection and potentially wait some seconds.

---

So, this is a quick recap of what I loved about programming in Go. I probably missed something and will be adding things over time.

If you want to share your opinion on Go, I'm [@FiloSottile](https://twitter.com/FiloSottile) on Twitter. And if you've never tried Go, give it a shot! It's not only about concurrency!