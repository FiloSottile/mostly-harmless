package main

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
	"runtime/debug"
	"time"
)

func main() {
	l, err := net.Listen("tcp", "localhost:4242")
	fatalIfErr(err)

	for {
		conn, err := l.Accept()
		fatalIfErr(err) // TODO: check for temporary errors
		go serviceConn(conn)
	}
}

func serviceConn(conn net.Conn) {
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, conn, 1+2+2); err != nil {
		log.Println("Failed to read record header:", err)
		conn.Close()
		return
	}
	length := binary.BigEndian.Uint16(buf.Bytes()[3:5])
	if _, err := io.CopyN(&buf, conn, int64(length)); err != nil {
		log.Println("Failed to read Client Hello record:", err)
		conn.Close()
		return
	}

	ch, ok := ParseClientHello(buf.Bytes())
	if !ok {
		log.Println("Failed to parse Client Hello.")
	} else {
		log.Printf("Received connection for SNI %q!", ch.SNI)
	}

	conn.SetDeadline(time.Time{}) // reset deadline
	conn.(*net.TCPConn).SetKeepAlive(true)
	conn.(*net.TCPConn).SetKeepAlivePeriod(3 * time.Minute)

	// TODO: move to main()
	cert, err := tls.LoadX509KeyPair("localhost.pem", "localhost-key.pem")
	fatalIfErr(err)
	config := &tls.Config{Certificates: []tls.Certificate{cert}}

	c := tls.Server(prefixConn{
		Reader: io.MultiReader(&buf, conn),
		Conn:   conn,
	}, config)

	// proxyConn(c, "gophercon.com:http")
	copyToStderr(c) // TODO: timeouts
}

type prefixConn struct {
	io.Reader
	net.Conn
}

func (c prefixConn) Read(b []byte) (int, error) {
	return c.Reader.Read(b)
}

func proxyConn(conn net.Conn, addr string) {
	upstream, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println(err)
		return
	}
	defer upstream.Close()
	go io.Copy(upstream, conn)
	_, err = io.Copy(conn, upstream) // splice from 1.11!
	log.Printf("Proxy connection finished with err = %v", err)
}

func copyToStderr(conn net.Conn) {
	n, err := io.Copy(os.Stderr, conn)
	log.Printf("Copied %d bytes and ended with err = %v.", n, err)
}

func fatalIfErr(err error) {
	if err != nil {
		debug.PrintStack()
		log.Fatal(err)
	}
}
