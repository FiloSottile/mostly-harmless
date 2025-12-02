package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/qlogwriter"
	"github.com/quic-go/quic-go/qlogwriter/jsontext"
)

// Packet represents a JSON-encoded packet for the wire format.
type Packet struct {
	Type   string `json:"type"`             // "client->server" or "server->client"
	Packet string `json:"packet,omitempty"` // base64-encoded packet data
}

// QlogEvent represents a qlog event output line.
type QlogEvent struct {
	Type  string          `json:"type"`  // "client qlog" or "server qlog"
	Time  string          `json:"time"`  // relative time since connection start
	Event string          `json:"event"` // event name
	Data  json.RawMessage `json:"data"`  // event-specific data as raw JSON
}

// DebugLog represents a debug log message output line.
type DebugLog struct {
	Type    string `json:"type"`    // "client log", "server log", or "harness log"
	Message string `json:"message"` // log message
}

// jsonLogWriter captures log output and writes it as JSON lines.
type jsonLogWriter struct {
	mu     sync.Mutex
	outEnc *json.Encoder
	buf    []byte
}

func newJsonLogWriter(outEnc *json.Encoder) *jsonLogWriter {
	return &jsonLogWriter{
		outEnc: outEnc,
	}
}

func (w *jsonLogWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, p...)

	// Process complete lines
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}
		line := string(w.buf[:idx])
		w.buf = w.buf[idx+1:]

		if line == "" {
			continue
		}

		// Determine log type from prefix and strip it
		var logType, message string
		if strings.HasPrefix(line, "harness: ") {
			logType = "harness log"
			message = strings.TrimPrefix(line, "harness: ")
		} else if strings.HasPrefix(line, "client ") {
			logType = "client log"
			message = strings.TrimPrefix(line, "client ")
		} else if strings.HasPrefix(line, "server ") {
			logType = "server log"
			message = strings.TrimPrefix(line, "server ")
		} else {
			// Unknown prefix, use generic type
			logType = "quic-go log"
			message = line
		}
		w.outEnc.Encode(DebugLog{
			Type:    logType,
			Message: message,
		})
	}
	return len(p), nil
}

// fakeAddr implements net.Addr for our fake connections.
type fakeAddr struct {
	name string
}

func (a fakeAddr) Network() string { return "fake" }
func (a fakeAddr) String() string  { return a.name }

// interceptConn is a fake net.PacketConn that intercepts packets.
type interceptConn struct {
	localAddr  net.Addr
	remoteAddr net.Addr
	direction  string // "client->server" or "server->client"

	mu       sync.Mutex
	outEnc   *json.Encoder
	incoming chan packet

	closed   bool
	closedCh chan struct{}
}

type packet struct {
	data []byte
	addr net.Addr
}

func newInterceptConn(local, remote net.Addr, direction string, outEnc *json.Encoder) *interceptConn {
	return &interceptConn{
		localAddr:  local,
		remoteAddr: remote,
		direction:  direction,
		outEnc:     outEnc,
		incoming:   make(chan packet, 100),
		closedCh:   make(chan struct{}),
	}
}

// ReadFrom receives packets that have been injected via Deliver.
func (c *interceptConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	select {
	case pkt := <-c.incoming:
		n = copy(p, pkt.data)
		return n, pkt.addr, nil
	case <-c.closedCh:
		return 0, nil, net.ErrClosed
	}
}

// WriteTo outputs packets as JSON but does NOT deliver them.
func (c *interceptConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, net.ErrClosed
	}

	pkt := Packet{
		Type:   c.direction,
		Packet: base64.StdEncoding.EncodeToString(p),
	}
	if err := c.outEnc.Encode(pkt); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Deliver injects a packet to be received by this connection.
func (c *interceptConn) Deliver(data []byte, from net.Addr) {
	select {
	case c.incoming <- packet{data: data, addr: from}:
	case <-c.closedCh:
	}
}

func (c *interceptConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.closedCh)
	}
	return nil
}

func (c *interceptConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *interceptConn) SetDeadline(t time.Time) error      { return nil }
func (c *interceptConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *interceptConn) SetWriteDeadline(t time.Time) error { return nil }

// jsonQlogTrace implements qlogwriter.Trace to output events as JSON lines.
type jsonQlogTrace struct {
	mu        sync.Mutex
	outEnc    *json.Encoder
	typ       string // "client qlog" or "server qlog"
	startTime time.Time
	producers int
	closed    bool
}

func newJsonQlogTrace(outEnc *json.Encoder, typ string) *jsonQlogTrace {
	return &jsonQlogTrace{
		outEnc:    outEnc,
		typ:       typ,
		startTime: time.Now(),
	}
}

// AddProducer returns a new Recorder for this trace.
func (t *jsonQlogTrace) AddProducer() qlogwriter.Recorder {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.producers++
	return &jsonQlogRecorder{trace: t}
}

// SupportsSchemas returns true for the QUIC event schema.
func (t *jsonQlogTrace) SupportsSchemas(schema string) bool {
	return true // Accept all schemas
}

// emit writes a qlog event as JSON.
func (t *jsonQlogTrace) emit(event string, data json.RawMessage) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	t.outEnc.Encode(QlogEvent{
		Type:  t.typ,
		Time:  fmt.Sprintf("%.3fms", float64(time.Since(t.startTime).Microseconds())/1000),
		Event: event,
		Data:  data,
	})
}

// jsonQlogRecorder implements qlogwriter.Recorder.
type jsonQlogRecorder struct {
	trace *jsonQlogTrace
}

// RecordEvent records a qlog event.
func (r *jsonQlogRecorder) RecordEvent(ev qlogwriter.Event) {
	// Capture the event name
	name := ev.Name()

	// Encode the event data using jsontext.Encoder
	var buf bytes.Buffer
	enc := jsontext.NewEncoder(&buf)
	ev.Encode(enc, time.Now())

	// Use the encoded data, or empty object if encoding failed
	data := buf.Bytes()
	if len(data) == 0 {
		data = []byte("{}")
	}
	r.trace.emit(name, json.RawMessage(data))
}

// Close closes this recorder.
func (r *jsonQlogRecorder) Close() error {
	r.trace.mu.Lock()
	defer r.trace.mu.Unlock()
	r.trace.producers--
	return nil
}

// generateSelfSignedCert creates a self-signed certificate for testing.
func generateSelfSignedCert() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
	}, nil
}

func main() {
	// Create JSON encoder for stdout
	outEnc := json.NewEncoder(os.Stdout)

	// Redirect log output to JSON - all logs become JSON lines
	// "harness: " prefix -> "harness log", "client " -> "client log", "server " -> "server log"
	logWriter := newJsonLogWriter(outEnc)
	log.SetOutput(logWriter)
	log.SetFlags(0)

	// Create fake addresses
	clientAddr := fakeAddr{"client:1234"}
	serverAddr := fakeAddr{"server:443"}

	// Create intercepting connections
	clientConn := newInterceptConn(clientAddr, serverAddr, "client->server", outEnc)
	serverConn := newInterceptConn(serverAddr, clientAddr, "server->client", outEnc)

	// Generate self-signed certificate
	cert, err := generateSelfSignedCert()
	if err != nil {
		log.Fatalf("harness: failed to generate certificate: %v", err)
	}

	// Server TLS config
	serverTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"quic-harness"},
	}

	// Client TLS config (skip verification for self-signed cert)
	clientTLSConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-harness"},
	}

	// Create qlog tracers
	clientQlog := newJsonQlogTrace(outEnc, "client qlog")
	serverQlog := newJsonQlogTrace(outEnc, "server qlog")

	// Create server transport with Retry enabled
	serverTransport := &quic.Transport{
		Conn: serverConn,
		// Enable Retry packets by always requiring source address verification
		VerifySourceAddress: func(addr net.Addr) bool {
			return true
		},
	}

	// Create client transport
	clientTransport := &quic.Transport{
		Conn: clientConn,
	}

	// QUIC config with qlog tracers and very long timeouts (1 year)
	// to avoid any timeout-based disconnections
	const oneYear = 365 * 24 * time.Hour
	serverQuicConfig := &quic.Config{
		HandshakeIdleTimeout: oneYear,
		MaxIdleTimeout:       oneYear,
		Tracer: func(ctx context.Context, isClient bool, connID quic.ConnectionID) qlogwriter.Trace {
			return serverQlog
		},
	}

	clientQuicConfig := &quic.Config{
		HandshakeIdleTimeout: oneYear,
		MaxIdleTimeout:       oneYear,
		Tracer: func(ctx context.Context, isClient bool, connID quic.ConnectionID) qlogwriter.Trace {
			return clientQlog
		},
	}

	// Start server listener
	listener, err := serverTransport.Listen(serverTLSConfig, serverQuicConfig)
	if err != nil {
		log.Fatalf("harness: failed to create listener: %v", err)
	}

	// Channels for signaling connection completion
	serverDone := make(chan struct{})
	clientDone := make(chan struct{})

	// Server goroutine: accept connection and handle it
	go func() {
		defer close(serverDone)

		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Printf("harness: server accept error: %v", err)
			return
		}

		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			log.Printf("harness: server accept stream error: %v", err)
			return
		}

		data, err := io.ReadAll(stream)
		if err != nil {
			log.Printf("harness: server read error: %v", err)
			return
		}

		log.Printf("harness: server received: %s", string(data))

		_, err = stream.Write([]byte("pong"))
		if err != nil {
			log.Printf("harness: server write error: %v", err)
			return
		}
		stream.Close()
		log.Printf("harness: server done")
	}()

	// Client goroutine: dial and send data
	go func() {
		defer close(clientDone)

		conn, err := clientTransport.Dial(context.Background(), serverAddr, clientTLSConfig, clientQuicConfig)
		if err != nil {
			log.Printf("harness: client dial error: %v", err)
			return
		}

		stream, err := conn.OpenStreamSync(context.Background())
		if err != nil {
			log.Printf("harness: client open stream error: %v", err)
			return
		}

		_, err = stream.Write([]byte("ping"))
		if err != nil {
			log.Printf("harness: client write error: %v", err)
			return
		}
		stream.Close()

		data, err := io.ReadAll(stream)
		if err != nil {
			log.Printf("harness: client read error: %v", err)
			return
		}

		log.Printf("harness: client received: %s", string(data))
		log.Printf("harness: client done")
	}()

	// Main goroutine: read packets from stdin and deliver them
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			var pkt Packet
			if err := json.Unmarshal(scanner.Bytes(), &pkt); err != nil {
				log.Printf("harness: invalid JSON: %v", err)
				continue
			}

			// Ignore qlog events and other non-packet types
			switch pkt.Type {
			case "client->server":
				data, err := base64.StdEncoding.DecodeString(pkt.Packet)
				if err != nil {
					log.Printf("harness: invalid base64: %v", err)
					continue
				}
				// Deliver to server (packet came from client)
				serverConn.Deliver(data, clientAddr)
			case "server->client":
				data, err := base64.StdEncoding.DecodeString(pkt.Packet)
				if err != nil {
					log.Printf("harness: invalid base64: %v", err)
					continue
				}
				// Deliver to client (packet came from server)
				clientConn.Deliver(data, serverAddr)
			default:
				// Ignore qlog events and other types
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("harness: stdin error: %v", err)
		}
	}()

	// Wait for server to complete (no timeout - use Ctrl+C to exit)
	<-serverDone
	log.Printf("harness: connection completed successfully")

	// Clean up
	clientConn.Close()
	serverConn.Close()
	listener.Close()
}
