//+build ignore

package stages

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
)

func main() {
	l, err := net.Listen("tcp", "localhost:4242")
	fatalIfErr(err)

	for {
		conn, err := l.Accept()
		fatalIfErr(err) // TODO: check for temporary errors
		go serviceConnExtraGoroutine(conn)
	}
}

// Issues:
//  * no error handling
//  * no way to apply timeouts
//  * allocating buffers every time
//  * lost net.Conn interface
//  * complex shutdown (need to flush channel)
//  * more work to make it bidirectional

func unframeMessages(conn net.Conn) chan []byte {
	c := make(chan []byte)
	go func() {
		for {
			var length uint32
			err := binary.Read(conn, binary.BigEndian, &length)
			if err != nil {
				close(c)
				return // No way to report this error!
			}
			buf := make([]byte, length)
			_, err = io.ReadFull(conn, buf)
			if err != nil {
				close(c)
				return // No way to report this error!
			}
			c <- buf
		}
	}()
	return c
}

func serviceConnExtraGoroutine(conn net.Conn) {
	c := unframeMessages(conn)
	var n int
	for buf := range c {
		n += len(buf)
		os.Stderr.Write(buf)
	}
	log.Printf("Copied %d bytes.", n)
	conn.Close()
}
