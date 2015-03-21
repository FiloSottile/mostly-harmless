package main

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
)

// TODO:
// * gracefully handle SIGTERM letting connections finish
// * reload on SIGHUP without killing connections

func serve(port string, filename string) {
	l, err := net.Listen("tcp", net.JoinHostPort("", port))
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	log.Printf("serving on %s", l.Addr())

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go func(c net.Conn, name string) {
			log.Printf("%s -> %s: connected", c.RemoteAddr(), c.LocalAddr())

			file, err := os.Open(name)
			if err != nil {
				log.Fatal(err)
			}

			n, err := io.Copy(c, file)
			log.Printf("%s -> %s: transferred %d bytes", c.RemoteAddr(), c.LocalAddr(), n)
			if err != nil {
				log.Printf("%s -> %s: %v", c.RemoteAddr(), c.LocalAddr(), err)
			}

			c.Close()
		}(conn, filename)
	}
}

var digitRe = regexp.MustCompile(`\d+`)

const folder = "."

func main() {
	files, err := ioutil.ReadDir(folder)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if f.IsDir() || !digitRe.MatchString(f.Name()) {
			continue
		}

		go serve(f.Name(), filepath.Join(folder, f.Name()))
	}

	select {} // Wait forever
}
