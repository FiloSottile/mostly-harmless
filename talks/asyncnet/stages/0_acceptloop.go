package stages

import (
	"io"
	"log"
	"net"
	"os"
	"time"
)

func AcceptLoop() {
	l, err := net.Listen("tcp", "localhost:4242")
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := l.Accept()
		if err, ok := err.(net.Error); ok && err.Temporary() {
			log.Printf("Temporary Accept error: %v; sleeping 1s...", err)
			time.Sleep(1 * time.Second)
		} else if err != nil {
			log.Fatal(err)
		}
		go serviceConn(conn)
	}
}

func serviceConn(conn net.Conn) {
	defer conn.Close()
	n, err := io.Copy(os.Stderr, conn)
	log.Printf("Copied %d bytes and ended with err = %v.", n, err)
}
