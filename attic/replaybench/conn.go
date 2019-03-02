package replaybench

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"time"
)

type Recording struct {
	LocalAddr  string
	RemoteAddr string

	Output io.Writer

	Recording []byte
}

func (r *Recording) GetConn() (net.Conn, error) {
	if r.Recording == nil {
		if r.LocalAddr != "" && r.RemoteAddr != "" {
			panic("must only specify one of LocalAddr or RemoteAddr")
		}
		if r.LocalAddr == "" && r.RemoteAddr == "" {
			panic("must specify one of LocalAddr, RemoteAddr or Recording")
		}
		o := r.Output
		if o == nil {
			o = os.Stderr
		}
		if r.LocalAddr != "" {
			fmt.Fprintf(o, "Listening on %s...\n", r.LocalAddr)
			l, err := net.Listen("tcp", r.LocalAddr)
			if err != nil {
				return nil, err
			}
			defer l.Close()
			c, err := l.Accept()
			return &recordingConn{Conn: c, o: r.Output}, err
		}
		if r.RemoteAddr != "" {
			fmt.Fprintf(o, "Connecting to %s...\n", r.RemoteAddr)
			c, err := net.Dial("tcp", r.RemoteAddr)
			return &recordingConn{Conn: c, o: r.Output}, err
		}
	}

	return &conn{r: r,
		ReadCloser: ioutil.NopCloser(bytes.NewReader(r.Recording)),
		Writer:     ioutil.Discard,
	}, nil
}

type conn struct {
	io.ReadCloser
	io.Writer

	r *Recording
}

var _ net.Conn = &conn{}

func (c *conn) LocalAddr() net.Addr {
	addr, err := net.ResolveTCPAddr("tcp", c.r.LocalAddr)
	if err != nil {
		panic(err)
	}
	return addr
}
func (c *conn) RemoteAddr() net.Addr {
	addr, err := net.ResolveTCPAddr("tcp", c.r.RemoteAddr)
	if err != nil {
		panic(err)
	}
	return addr
}

func (*conn) SetDeadline(t time.Time) error      { return nil }
func (*conn) SetReadDeadline(t time.Time) error  { return nil }
func (*conn) SetWriteDeadline(t time.Time) error { return nil }

type recordingConn struct {
	net.Conn

	buf bytes.Buffer
	o   io.Writer
}

func (c *recordingConn) Read(p []byte) (n int, err error) {
	n, err = c.Conn.Read(p)
	c.buf.Write(p[:n])
	if err == io.EOF {
		c.print()
	}
	return n, err
}

func (c *recordingConn) Close() error {
	c.print()
	return c.Conn.Close()
}

func (c *recordingConn) print() {
	o := c.o
	if o == nil {
		o = os.Stderr
	}
	n := rand.Intn(9999)
	fmt.Fprintf(o, "\nvar recording%d = []byte{", n)
	for i, b := range c.buf.Bytes() {
		if i%16 == 0 {
			fmt.Fprintf(o, "\n\t")
		} else {
			fmt.Fprintf(o, " ")
		}
		fmt.Fprintf(o, "0x%02x,", b)
	}
	fmt.Fprintf(o, "\n}\n\n")
}
