package torrent

import (
	"fmt"
	"io"
)

type Writer struct {
	w      io.Writer
	isDict bool
	isKey  bool
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

func (w *Writer) WriteString(s string) {
	fmt.Fprintf(w.w, "%d:%s", len(s), s)
	if w.isKey {
		w.isKey = false
	} else if w.isDict {
		w.isKey = true
	}
}

func (w *Writer) WriteBytes(b []byte) {
	fmt.Fprintf(w.w, "%d:", len(b))
	w.w.Write(b)
	if w.isKey {
		w.isKey = false
	} else if w.isDict {
		w.isKey = true
	}
}

func (w *Writer) WriteInt(i int) {
	if w.isKey {
		panic("int can't be a key")
	}
	fmt.Fprintf(w.w, "i%de", i)
	if w.isDict {
		w.isKey = true
	}
}

func (w *Writer) WriteInt64(i int64) {
	if w.isKey {
		panic("int can't be a key")
	}
	fmt.Fprintf(w.w, "i%de", i)
	if w.isDict {
		w.isKey = true
	}
}

func (w *Writer) WriteList(f func(*Writer)) {
	if w.isKey {
		panic("list can't be a key")
	}
	fmt.Fprintf(w.w, "l")
	f(&Writer{w: w.w})
	fmt.Fprintf(w.w, "e")
	if w.isDict {
		w.isKey = true
	}
}

func (w *Writer) WriteDict(f func(*Writer)) {
	if w.isKey {
		panic("dict can't be a key")
	}
	fmt.Fprintf(w.w, "d")
	ww := &Writer{}
	ww.isDict = true
	ww.isKey = true
	ww.w = w.w
	f(ww)
	if !ww.isKey {
		panic("missing value for key")
	}
	fmt.Fprintf(w.w, "e")
	if w.isDict {
		w.isKey = true
	}
}
