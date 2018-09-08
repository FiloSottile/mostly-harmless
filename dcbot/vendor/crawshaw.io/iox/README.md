# iox: I/O tools for Go programs

Package iox contains two Go objects of note: _Filer_ and _BufferFile_.

https://godoc.org/crawshaw.io/iox

## Filer

Managing file resources in highly-concurrent programs gets tricky.
A process easily, even typically, has more in-flight goroutines
than allowed file descriptors from the operating system.
This requires programmers limit the number of open descriptors
with some kind of throttle object.

An iox.Filer wraps the functions used to open file descriptors and
makes sure it never opens more than some maximum (typically derived
from the processes rlimit).

It wraps *os.File pointers in a new object which returns the file
descriptor allotment to the Filer pool when Close is called.

## BufferFile

A BufferFile is a file-like object that stores its first N bytes in
memory, and the rest in a temporary file on disk.

It is designed for loads where the **typical** case fits in some
small amount of memory, but the **worst** case requires more space
than can be provisioned in RAM.
(This usually means a server is handling tens to hundreds of thousands
of simultaneous requests.)

BufferFile does not create its temporary backing file until its
contents exceed the memory buffer, so the typical case does not require
any file descriptors.
Programs can begin (and usually complete) processing a request without
ever blocking on file descriptors, meaning a server never runs into
file descriptors as a bottleneck when processing a typical workload.

# Installation

Install with:

```
go get crawshaw.io/iox
```

There are no version numbers yet, this package needs some time to bake.