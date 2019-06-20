// Copyright 2019 Filippo Valsorda
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

// +build go1.12 !go1.13

package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
	"unsafe"
)

func main() {
	var target = flag.String("target", "filippo.io:443", "target HOSTNAME")
	var replay = flag.Bool("replay", false, "replay ticket from stdin")
	flag.Parse()

	if *replay {
		t := readTicket()
		sendTicket(*target, t)
	} else {
		getTicket(*target)
	}
}

func getTicket(target string) {
	f := func(cs *ClientSessionState) {
		b, err := json.MarshalIndent(cs, "", "\t")
		if err != nil {
			panic(err)
		}
		os.Stdout.Write(b)
	}
	config := &tls.Config{
		ClientSessionCache: sessionLogger(f),
		InsecureSkipVerify: true,
		MaxVersion:         tls.VersionTLS12,
		CipherSuites:       []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256},
	}
	conn, err := tls.Dial("tcp", target, config)
	if err != nil {
		panic(err)
	}
	conn.Close()
}

type sessionLogger func(cs *ClientSessionState)

func (f sessionLogger) Get(_ string) (*tls.ClientSessionState, bool) {
	return nil, false
}

func (f sessionLogger) Put(_ string, cs *tls.ClientSessionState) {
	f((*ClientSessionState)(unsafe.Pointer(cs)))
}

func readTicket() *ClientSessionState {
	var cs *ClientSessionState
	if err := json.NewDecoder(os.Stdin).Decode(&cs); err != nil {
		panic(err)
	}
	return cs
}

func sendTicket(target string, cs *ClientSessionState) {
	r := &sessionReplayer{cs: cs}
	config := &tls.Config{
		ClientSessionCache: r,
		InsecureSkipVerify: true,
		MaxVersion:         tls.VersionTLS12,
		CipherSuites:       []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256},
	}
	conn, err := tls.Dial("tcp", target, config)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Did resume: %v\n", conn.ConnectionState().DidResume)
	if !r.servedAt.IsZero() {
		fmt.Printf("Delta: %v\n", time.Since(r.servedAt))
	}
	conn.Close()
}

type sessionReplayer struct {
	cs       *ClientSessionState
	servedAt time.Time
}

func (r *sessionReplayer) Get(sk string) (*tls.ClientSessionState, bool) {
	if !r.servedAt.IsZero() {
		panic("Get called multiple times")
	}
	r.servedAt = time.Now()
	return (*tls.ClientSessionState)(unsafe.Pointer(r.cs)), true
}

func (r *sessionReplayer) Put(_ string, cs *tls.ClientSessionState) {}

// ClientSessionState is lifted from Go 1.12.6.
type ClientSessionState struct {
	SessionTicket      []uint8
	Vers               uint16
	CipherSuite        uint16
	MasterSecret       []byte
	ServerCertificates []*Certificate
	VerifiedChains     [][]*Certificate
	ReceivedAt         time.Time
	Nonce              []byte
	UseBy              time.Time
	AgeAdd             uint32
}

type Certificate x509.Certificate

func (c *Certificate) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Raw)
}

func (c *Certificate) UnmarshalJSON(b []byte) error {
	var raw []byte
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	cert, err := x509.ParseCertificate(raw)
	if err != nil {
		return err
	}
	*c = Certificate(*cert)
	return nil
}
