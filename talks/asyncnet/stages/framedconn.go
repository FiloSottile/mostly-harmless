package stages

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

type framedConn struct {
	net.Conn
	readMu, writeMu   sync.Mutex
	readBuf           bytes.Buffer
	readErr, writeErr error
}

// func (c *framedConn) Read(b []byte) (n int, err error) {
// 	c.readMu.Lock()
// 	defer c.readMu.Unlock()
// 	if c.readRemainder == 0 {
// 		err = binary.Read(c.Conn, binary.BigEndian, &c.readRemainder)
// 		if err != nil {
// 			return 0, err
// 		}
// 	}
// 	if uint64(len(b)) > c.readRemainder {
// 		b = b[:c.readRemainder]
// 	}
// 	// Note that we return from Read as soon as possible.
// 	n, err = c.Conn.Read(b)
// 	c.readRemainder -= uint64(n)
// 	return
// }

func (c *framedConn) Read(b []byte) (n int, err error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	if c.readErr != nil {
		return 0, c.readErr
	}
	defer func() { c.readErr = err }()
	if c.readBuf.Len() == 0 {
		var length uint64
		err = binary.Read(c.Conn, binary.BigEndian, &length)
		if err != nil {
			return 0, err
		}
		_, err = io.CopyN(&c.readBuf, c.Conn, int64(length))
		if err != nil {
			return 0, err
		}
	}
	return c.readBuf.Read(b)
}

func (c *framedConn) Write(b []byte) (n int, err error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.writeErr != nil {
		return 0, c.writeErr
	}
	defer func() { c.writeErr = err }()
	err = binary.Write(c.Conn, binary.BigEndian, uint64(len(b)))
	if err != nil {
		return 0, err
	}
	return c.Conn.Write(b)
}

func serviceConnFramed(conn net.Conn) {
	n, err := io.Copy(os.Stderr, &framedConn{Conn: conn})
	log.Printf("Copied %d bytes and ended with err = %v.", n, err)
	conn.Close()
}
