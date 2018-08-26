package stages

import (
	"io"
	"log"
	"net"
	"os"
	"runtime/debug"
)

func AcceptLoop() {
	// go build . && ./asyncnet
	// nc localhost 4242

	l, err := net.Listen("tcp", "localhost:4242")
	fatalIfErr(err)

	for {
		conn, err := l.Accept()
		fatalIfErr(err) // TODO: check for temporary errors
		go serviceConn(conn)
	}
}

func serviceConn(conn net.Conn) {
	n, err := io.Copy(os.Stderr, conn)
	log.Printf("Copied %d bytes and ended with err = %v.", n, err)
	conn.Close()
}

func fatalIfErr(err error) {
	if err != nil {
		debug.PrintStack()
		log.Fatal(err)
	}
}
