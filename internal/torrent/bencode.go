package torrent

import (
	"errors"
	"fmt"
	"io"
)

type Writer struct {
	w      io.Writer
	err    error
	isDict bool
	isKey  bool
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

func (w *Writer) SetError(err error) {
	if w.err == nil {
		w.err = err
	}
}

func (w *Writer) Err() error {
	return w.err
}

func (w *Writer) WriteString(s string) {
	if w.err != nil {
		return
	}
	_, w.err = fmt.Fprintf(w.w, "%d:%s", len(s), s)
	if w.isKey {
		w.isKey = false
	} else if w.isDict {
		w.isKey = true
	}
}

func (w *Writer) WriteBytes(b []byte) {
	if w.err != nil {
		return
	}
	_, w.err = fmt.Fprintf(w.w, "%d:", len(b))
	w.w.Write(b)
	if w.isKey {
		w.isKey = false
	} else if w.isDict {
		w.isKey = true
	}
}

func (w *Writer) WriteInt(i int) {
	if w.err != nil {
		return
	}
	if w.isKey {
		w.err = errors.New("int can't be a key")
		return
	}
	_, w.err = fmt.Fprintf(w.w, "i%de", i)
	if w.isDict {
		w.isKey = true
	}
}

func (w *Writer) WriteInt64(i int64) {
	if w.err != nil {
		return
	}
	if w.isKey {
		w.err = errors.New("int can't be a key")
		return
	}
	_, w.err = fmt.Fprintf(w.w, "i%de", i)
	if w.isDict {
		w.isKey = true
	}
}

func (w *Writer) WriteList(f func(*Writer)) {
	if w.err != nil {
		return
	}
	if w.isKey {
		w.err = errors.New("list can't be a key")
		return
	}
	if _, w.err = fmt.Fprintf(w.w, "l"); w.err != nil {
		return
	}
	f(&Writer{w: w.w})
	if w.err != nil {
		return
	}
	_, w.err = fmt.Fprintf(w.w, "e")
	if w.isDict {
		w.isKey = true
	}
}

func (w *Writer) WriteDict(f func(*Writer)) {
	if w.err != nil {
		return
	}
	if w.isKey {
		w.err = errors.New("dict can't be a key")
		return
	}
	_, w.err = fmt.Fprintf(w.w, "d")
	ww := &Writer{}
	ww.isDict = true
	ww.isKey = true
	ww.w = w.w
	f(ww)
	if ww.err != nil {
		w.err = ww.err
		return
	}
	if !ww.isKey {
		w.err = errors.New("missing value for key")
		return
	}
	_, w.err = fmt.Fprintf(w.w, "e")
	if w.isDict {
		w.isKey = true
	}
}
