package stages

import (
	"log"
	"net"
	"os"
	"time"
)

func copyToStderr(conn net.Conn) {
	var total int
	for {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		var buf [128]byte
		n, err := conn.Read(buf[:])
		os.Stderr.Write(buf[:n])
		total += n
		if err != nil {
			log.Printf("Copied %d bytes and ended with err = %v.", total, err)
			return // or we could recover if Timeout()
		}
	}
}
