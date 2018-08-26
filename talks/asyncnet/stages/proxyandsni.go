package stages

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"
)

func serviceConnProxiAndSNI(conn net.Conn) {
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
	proxyConn(prefixConn{
		Reader: io.MultiReader(&buf, conn),
		Conn:   conn,
	})
}

type prefixConn struct {
	io.Reader
	net.Conn
}

func (c prefixConn) Read(b []byte) (int, error) {
	return c.Reader.Read(b)
}

func proxyConn(conn net.Conn) {
	upstream, err := net.Dial("tcp", "gophercon.com:https")
	if err != nil {
		log.Println(err)
		return
	}
	defer upstream.Close()
	go io.Copy(upstream, conn)
	_, err = io.Copy(conn, upstream)
	log.Printf("Proxy connection finished with err = %v", err)
}
