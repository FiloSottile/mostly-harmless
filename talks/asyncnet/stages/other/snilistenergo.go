//+build ignore

package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"time"
)

func main() {
	l, err := net.Listen("tcp", "localhost:4242")
	fatalIfErr(err)

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		io.WriteString(rw, "Hello, GopherCon!")
	})
	log.Println(http.ServeTLS(NewSNIListener(l), nil,
		"localhost.pem", "localhost-key.pem"))
}

type sniListener struct {
	net.Listener
	c chan net.Conn
}

func NewSNIListener(l net.Listener) net.Listener {
	ll := sniListener{
		Listener: l,
		c:        make(chan net.Conn),
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				// TODO: handle temporary errors
				break
			}
			go ll.serviceConn(conn)
		}
	}()
	return ll
}

func (l sniListener) serviceConn(conn net.Conn) {
	conn.SetDeadline(time.Now().Add(30 * time.Second))
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, conn, 1+2+2); err != nil {
		conn.Close()
		log.Println(err)
		return
	}
	length := binary.BigEndian.Uint16(buf.Bytes()[3:5])
	if _, err := io.CopyN(&buf, conn, int64(length)); err != nil {
		conn.Close()
		log.Println(err)
		return
	}

	ch, ok := ParseClientHello(buf.Bytes())
	if !ok {
		log.Println("Failed to parse Client Hello.")
	} else {
		log.Printf("Received connection for %q!", ch.SNI)
	}

	conn.SetDeadline(time.Time{}) // reset deadline
	conn.(*net.TCPConn).SetKeepAlive(true)
	conn.(*net.TCPConn).SetKeepAlivePeriod(3 * time.Minute)
	l.c <- prefixConn{
		Reader: io.MultiReader(&buf, conn),
		Conn:   conn,
	}
}

func (l sniListener) Accept() (net.Conn, error) {
	conn, ok := <-l.c
	if !ok {
		return nil, errors.New("Listener closed")
	}
	return conn, nil
}

func (l sniListener) Close() error {
	// TODO: cancel the inflight serviceConn
	return l.Listener.Close()
}

type prefixConn struct {
	io.Reader
	net.Conn
}

func (c prefixConn) Read(b []byte) (int, error) {
	return c.Reader.Read(b)
}

func fatalIfErr(err error) {
	if err != nil {
		debug.PrintStack()
		log.Fatal(err)
	}
}
