package stages

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"
)

func serviceConnLogSNI(conn net.Conn) {
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, conn, 1+2+2); err != nil {
		log.Println("Failed to read record header:", err)
		return
	}
	length := binary.BigEndian.Uint16(buf.Bytes()[3:5])
	if _, err := io.CopyN(&buf, conn, int64(length)); err != nil {
		log.Println("Failed to read Client Hello record:", err)
		return
	}

	ch, ok := ParseClientHello(buf.Bytes())
	if !ok {
		log.Println("Failed to parse Client Hello.")
	} else {
		log.Printf("Received connection for %q!", ch.SNI)
	}
}
