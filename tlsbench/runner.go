package main

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"

	tls "github.com/google/boringssl/ssl/test/runner"
)

// ttfbConn records the time at which the first byte is read
type ttfbConn struct {
	net.Conn

	firstReadTime *time.Time
}

func (c ttfbConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if n > 0 && c.firstReadTime.IsZero() {
		*c.firstReadTime = time.Now()
	}
	return
}

// inFlightConn records the time between each Write and the next Read
type inFlightConn struct {
	net.Conn

	inFlightTime time.Duration
	writeTime    int64 // UnixNano, access with sync/atomic
}

func (c *inFlightConn) Write(p []byte) (n int, err error) {
	n, err = c.Conn.Write(p)
	atomic.StoreInt64(&c.writeTime, time.Now().UnixNano())
	return
}

func (c *inFlightConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if n > 0 {
		wt := atomic.SwapInt64(&c.writeTime, 0)
		if wt != 0 {
			c.inFlightTime += time.Duration(time.Now().UnixNano() - wt)
		}
	}
	return
}

func runJob(j *job) {
	start := time.Now()
	var handshakeStart time.Time
	var serverHelloTime time.Time
	var inFlightTime time.Duration

	conn, err := net.DialTimeout("tcp", j.Address, j.Timeout)
	if err == nil {
		conn.SetDeadline(start.Add(j.Timeout))
		ifConn := &inFlightConn{Conn: ttfbConn{conn, &serverHelloTime}}
		tlsConn := tls.Client(ifConn, j.tlsConfig)
		handshakeStart = time.Now()
		err = tlsConn.Handshake()
		inFlightTime = ifConn.inFlightTime
	}

	outputMu.Lock()
	j.bar.Increment()
	j.bar.Update()
	if err != nil {
		fmt.Printf("\r\033[K\x1b\x5b\x31\x6d%v\x1b\x5b\x30\x6d: %v (%v)\n", j.Name, err, time.Since(start))
		fmt.Print(j.bar.String())
		outputMu.Unlock()
	} else {
		j.h.Observe(time.Since(handshakeStart))
		j.sh.Observe(serverHelloTime.Sub(handshakeStart))
		j.ih.Observe(inFlightTime)
		outputMu.Unlock()
		conn.Close()
	}
}
