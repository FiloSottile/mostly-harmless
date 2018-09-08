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

package iox

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// A Filer creates files, managing load on file descriptors.
type Filer struct {
	tempdir string

	shuttingDown chan struct{} // closed on shutdown

	mu      sync.Mutex
	cond    *sync.Cond
	files   map[*File]struct{}
	fdlimit int
	seed    uint32
	seq     int
}

// NewFiler creates a Filer which will open at most fdLimit files simultaneously.
// If fdLimit is 0, a Filer is limited to 90% of the process's allowed files.
func NewFiler(fdLimit int) *Filer {
	if fdLimit == 0 {
		var lim syscall.Rlimit
		syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim)
		fdLimit = int(lim.Max - (lim.Max / 10))
	}
	if fdLimit == 0 {
		fdLimit = 90 // getrlimit failed, guess
	}
	filer := &Filer{
		tempdir:      os.TempDir(),
		shuttingDown: make(chan struct{}),
		files:        make(map[*File]struct{}),
		fdlimit:      fdLimit,
	}
	filer.cond = sync.NewCond(&filer.mu)
	return filer
}

// SetTempdir sets the default directory used to hold temporary files.
func (f *Filer) SetTempdir(tempdir string) {
	// TODO: just export tempdir field?
	f.tempdir = tempdir
}

// Open opens the named file for reading.
//
// It is similar to os.Open except it will block if Filer has exhasted
// its file descriptors until one is available.
func (f *Filer) Open(name string) (*File, error) {
	file := f.newFile()
	if file == nil {
		return nil, context.Canceled
	}
	osfile, err := os.Open(name)
	if err != nil {
		file.remove()
		return nil, err
	}
	file.File = osfile
	return file, nil
}

// OpenFile is a generalized file open method.
//
// It is similar to os.OpenFile except it will block if Filer has exhasted
// its file descriptors until one is available.
func (f *Filer) OpenFile(name string, flag int, perm os.FileMode) (*File, error) {
	file := f.newFile()
	if file == nil {
		return nil, context.Canceled
	}
	osfile, err := os.OpenFile(name, flag, perm)
	if err != nil {
		file.remove()
		return nil, err
	}
	file.File = osfile
	return file, nil
}

func (f *Filer) TempFile(dir, prefix, suffix string) (file *File, err error) {
	file = f.newFile()
	if file == nil {
		return nil, context.Canceled
	}

	if dir == "" {
		dir = f.tempdir
	}

	var osfile *os.File
	for i := 0; i < 1000; i++ {
		name := filepath.Join(dir, prefix+f.rand()+suffix)
		osfile, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			continue
		}
		break
	}
	if err != nil {
		file.remove()
		return nil, err
	}
	file.File = osfile
	file.isTemp = true
	return file, nil
}

// Shutdown gracefully shuts down the Filer.
// Any active files continue to work until the passed context is done.
// At that point they are explicitly closed and further operations return errors.
// Shutdown returns the error from ctx.
func (f *Filer) Shutdown(ctx context.Context) error {
	close(f.shuttingDown)
	f.cond.Broadcast()
	done := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			f.cond.Broadcast()
		case <-done:
		}
	}()

	f.mu.Lock()
	for {
		select {
		case <-ctx.Done():
			for file := range f.files {
				file.File.Close()
				delete(f.files, file)
			}
			// now len(f.files) == 0
		default:
		}
		if len(f.files) == 0 {
			break
		}
		f.cond.Wait()
	}
	f.mu.Unlock()

	close(done)
	return ctx.Err()
}

func (f *Filer) newFile() *File {
	file := &File{filer: f}

	f.mu.Lock()
	for {
		select {
		case <-f.shuttingDown:
			f.mu.Unlock()
			return nil
		default:
		}
		if len(f.files) < f.fdlimit {
			break
		}
		f.cond.Wait()
	}
	f.files[file] = struct{}{}
	f.mu.Unlock()

	return file
}

func (f *Filer) rand() string {
	const mod = 0x7fffffff

	f.mu.Lock()
	for f.seed == 0 || f.seed >= mod || f.seq > 100 {
		f.seed = uint32(time.Now().UnixNano() + int64(os.Getpid()))
	}
	// Park-Miller RNG, constants from wikipedia.
	v := uint32(uint64(f.seed) * 48271 % mod)
	f.seed = v
	f.mu.Unlock()

	return strconv.FormatUint(uint64(v), 16)
}

// File is an *os.File managed by a Filer.
//
// The Close method must be called on a File.
type File struct {
	*os.File

	filer  *Filer
	isTemp bool
}

func (file *File) remove() {
	file.filer.mu.Lock()
	delete(file.filer.files, file)
	file.filer.cond.Signal()
	file.filer.mu.Unlock()
}

// Close closes the underlying file descriptor and informs the Filer.
func (file *File) Close() error {
	err := file.File.Close()
	file.remove()

	if file.isTemp {
		rmErr := os.Remove(file.File.Name())
		if err == nil {
			err = rmErr
		}
	}
	return err
}
