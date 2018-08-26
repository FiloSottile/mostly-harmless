//+build ignore

package stages

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

func mainSNIListener() {
	l, err := net.Listen("tcp", "localhost:4242")
	fatalIfErr(err)

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		io.WriteString(rw, "Hello, GopherCon!")
	})
	log.Println(http.ServeTLS(sniListener{l}, nil,
		"localhost.pem", "localhost-key.pem"))
}

type sniListener struct {
	net.Listener
}

func (l sniListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	conn.SetDeadline(time.Now().Add(30 * time.Second))
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, conn, 1+2+2); err != nil {
		conn.Close()
		return nil, err
	}
	length := binary.BigEndian.Uint16(buf.Bytes()[3:5])
	if _, err := io.CopyN(&buf, conn, int64(length)); err != nil {
		conn.Close()
		return nil, err
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
	return prefixConn{
		Reader: io.MultiReader(&buf, conn),
		Conn:   conn,
	}, nil
}

type prefixConn struct {
	io.Reader
	net.Conn
}

func (c prefixConn) Read(b []byte) (int, error) {
	return c.Reader.Read(b)
}
