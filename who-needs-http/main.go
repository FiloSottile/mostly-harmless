// Copyright 2015 Filippo Valsorda
//
// Permission to use, copy, modify, and/or distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

// Command who-needs-http serves files over raw TCP.
//
// Who needs HTTP when you have TCP? Simply serve one file per port. Meant for a
// PoC||GTFO mirror. https://twitter.com/FiloSottile/status/579650094074564608
//
// Just run it in a folder containing files named after the port they should be
// served from.
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
