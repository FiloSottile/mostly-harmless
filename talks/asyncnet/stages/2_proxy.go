package stages

import (
	"io"
	"log"
	"net"
)

func serviceConnProxy(conn net.Conn) {
	defer conn.Close()
	log.Printf("Received a connection from %v.", conn.RemoteAddr())
	upstream, err := net.Dial("tcp", "gophercon.com:https")
	if err != nil {
		log.Println(err)
		return
	}
	defer upstream.Close()
	go io.Copy(upstream, conn)       // cancelled by Close
	_, err = io.Copy(conn, upstream) // splice from 1.11!
	log.Printf("Connection finished with err = %v", err)
}
