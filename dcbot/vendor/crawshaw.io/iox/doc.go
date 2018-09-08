// Copyright (c) 2018 David Crawshaw <david@zentus.com>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

// Package iox contains I/O utilities.
//
// The primary concern of the package is managing and minimizing the use
// of file descriptors, an operating system resource which is often in
// short supply in high-concurrency servers.
//
// The two objects that help in this are the Filer and BufferFile.
//
// A filer manages a allotment of file descriptors, blocking on file
// creation until an old file closes and frees up a descriptor allotment.
//
// A BufferFile keeps a fraction of its contents in memory.
// If the number of bytes stored in a BufferFile is small, no file
// descriptor is ever used.
package iox // import "crawshaw.io/iox"
