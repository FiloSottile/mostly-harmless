---
layout: post
title: "Making system calls from Assembly in Mac OS X"
date: 2013-11-04 02:05
comments: true
categories: 
---

The next step in my [playing with chroot escapes](/escaping-a-chroot-jail-slash-1/) is crafting some shellcode. Recently my main dev machine is a MacBook running OS X, so it felt reasonable to fiddle with making system calls of that platform.

By the way, a system call is a function of the kernel invoked by a userspace program and it can be something like writing to a file descriptor, or even exiting. Usually, these are wrapped by C functions in the standard library.

### The system calls

First, we need to know what system call we want to make, and what arguments it pretends.

A full list is hosted by Apple [here](http://www.opensource.apple.com/source/xnu/xnu-1504.3.12/bsd/kern/syscalls.master). The header also hints at the fact that they are inherited from BSD. Yeah, [that makes sense](https://en.wikipedia.org/wiki/OS_X).

So, to write our proverbial *Hello world* we will pick the syscall 4

    4   AUE_NULL    ALL { user_ssize_t write(int fd, user_addr_t cbuf, user_size_t nbyte); }

<!-- more -->

### 32-bit

Let's start easy. A cute 32-bit program, written in [NASM assembler language](http://alien.dowling.edu/~rohit/nasmdoc3.html). Compile with `nasm` or `yasm`, output format `MachO`, and link with `ld`.

I'm on a Intel machine, so what we are looking for is the x86 syscall calling conventions for the OS X or BSD platform. They are pretty simple:

* arguments passed on the stack, pushed right-to-left
* stack 16-bytes aligned
* syscall number in the `eax` register
* call by interrupt `0x80`

So what we have to do to print a "Hello world" is:

* push the length of the string (`int`) to the stack
* push a pointer to the string to the stack
* push the stdout file descriptor (1) to the stack
* align the stack by moving the stack pointer 4 more bytes (16 - 4 * 3)
* set the `eax` register to the `write` syscall number (4)
* interrupt `0x80`

{% gist 7125822 32.asm %}

### 64-bit

64-bit is a bit cleaner, but completely different: OS X (and GNU/Linux and everyone except Windows) on 64 architectures adopt the [System V AMD64 ABI reference](http://x86-64.org/documentation/abi.pdf). Jump to section **A.2.1** for the syscall calling convention.

* arguments are passed on the registers `rdi`, `rsi`, `rdx`, `r10`, `r8` and `r9`
* syscall number in the `rax` register
* the call is done via the `syscall` instruction
* what OS X contributes to the mix is that you have to add `0x20000000` to the syscall number (still have to figure out why)

So, here is the (IMHO) much more clean 64-bit "Hello world". Ah, if you want to do this at home and have it actually run, generate a `macho64` object with **a new version of** NASM or with YASM, and link with `ld` as usual.

{% gist 7125822 64.asm %}
