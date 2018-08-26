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
	go io.Copy(upstream, conn)
	_, err = io.Copy(conn, upstream)
	log.Printf("Connection finished with err = %v", err)
}
