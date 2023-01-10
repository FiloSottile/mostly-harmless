// Command bastion runs a reverse proxy service that allows un-addressable
// applications (for example because they are running behind a firewall or a
// NAT, or because the operator doesn't wish to take the DoS risk of being
// reachable from the Internet) to accept HTTP requests.
//
// Backends are identified by an Ed25519 public key, they authenticate with a
// self-signed TLS 1.3 certificate, and are reachable at a sub-path prefixed by
// the key hash.
//
// Read more at
// https://git.glasklar.is/sigsum/project/documentation/-/blob/main/bastion.md.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
)

var listenAddr = flag.String("listen", "localhost:8443", "host and port to listen at")
var testCertificates = flag.Bool("testcert", false, "use localhost.pem and localhost-key.pem instead of ACME")
var autocertCache = flag.String("cache", "", "directory to cache ACME certificates at")
var autocertHost = flag.String("host", "", "host to obtain ACME certificate for")
var autocertEmail = flag.String("email", "", "")
var allowedBackendsFile = flag.String("backends", "", "file listing accepted key hashes, one per line")

type keyHash [sha256.Size]byte

func main() {
	flag.BoolVar(&http2.VerboseLogs, "h2v", false, "enable HTTP/2 verbose logs")
	flag.Parse()

	var getCertificate func(hello *tls.ClientHelloInfo) (*tls.Certificate, error)
	if *testCertificates {
		cert, err := tls.LoadX509KeyPair("localhost.pem", "localhost-key.pem")
		getCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return &cert, err
		}
	} else {
		if *autocertCache == "" || *autocertHost == "" || *autocertEmail == "" {
			log.Fatal("-cache, -host, and -email or -testcert are required")
		}
		m := &autocert.Manager{
			Cache:      autocert.DirCache(*autocertCache),
			Prompt:     autocert.AcceptTOS,
			Email:      *autocertEmail,
			HostPolicy: autocert.HostWhitelist(*autocertHost),
		}
		getCertificate = m.GetCertificate
	}

	if *allowedBackendsFile == "" {
		log.Fatal("-backends is empty")
	}
	bs, err := os.ReadFile(*allowedBackendsFile)
	if err != nil {
		log.Fatalln("Failed to read backends file:", err)
	}
	allowedBackends := make(map[keyHash]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(bs)), "\n") {
		l, err := hex.DecodeString(line)
		if err != nil {
			log.Fatalf("Invalid backend: %q", line)
		}
		if len(l) != sha256.Size {
			log.Fatalf("Invalid backend: %q", line)
		}
		h := keyHash(l)
		allowedBackends[h] = true
	}

	bastionTLSConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		NextProtos: []string{"bastion/0"},
		ClientAuth: tls.RequireAnyClientCert,
		VerifyConnection: func(cs tls.ConnectionState) error {
			leaf := cs.PeerCertificates[0]
			pk, ok := leaf.PublicKey.(ed25519.PublicKey)
			if !ok {
				return errors.New("self-signed certificate key type is not Ed25519")
			}
			h := sha256.Sum256(pk)
			if !allowedBackends[h] {
				return fmt.Errorf("unrecognized backend %x", h)
			}
			return nil
		},
		GetCertificate: getCertificate,
	}

	p := &backendConnectionsPool{
		conns: make(map[keyHash]*http2.ClientConn),
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme = "https" // needed for the required :scheme header
			pr.Out.Host = pr.In.Context().Value("backend").(string)
			pr.SetXForwarded()
			// We don't interpret the query, so pass it on unmodified.
			pr.Out.URL.RawQuery = pr.In.URL.RawQuery
		},
		Transport: p,
	}

	hs := &http.Server{
		Addr: *listenAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if !strings.HasPrefix(path, "/") {
				http.Error(w, "request must start with /KEY_HASH/", http.StatusNotFound)
				return
			}
			path = path[1:]
			kh, path, ok := strings.Cut(path, "/")
			if !ok {
				http.Error(w, "request must start with /KEY_HASH/", http.StatusNotFound)
				return
			}
			ctx := context.WithValue(r.Context(), "backend", kh)
			r = r.Clone(ctx)
			r.URL.Path = "/" + path
			proxy.ServeHTTP(w, r)
		}),
		TLSConfig: &tls.Config{
			NextProtos:     []string{acme.ALPNProto},
			GetCertificate: getCertificate,
			GetConfigForClient: func(chi *tls.ClientHelloInfo) (*tls.Config, error) {
				for _, proto := range chi.SupportedProtos {
					if proto == "bastion/0" {
						// This is a bastion connection from a backend.
						return bastionTLSConfig, nil
					}
				}
				return nil, nil
			},
		},
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){
			"bastion/0": p.handleBackend,
		},
	}
	if err := http2.ConfigureServer(hs, nil); err != nil {
		log.Fatalln("Failed to configure HTTP/2:", err)
	}
	log.Printf("Serving at https://%s...", *listenAddr)
	hs.ListenAndServeTLS("", "")
}

type backendConnectionsPool struct {
	sync.RWMutex
	conns map[keyHash]*http2.ClientConn
}

func (p *backendConnectionsPool) RoundTrip(r *http.Request) (*http.Response, error) {
	kh, err := hex.DecodeString(r.Host)
	if err != nil || len(kh) != sha256.Size {
		// TODO: return this as a response instead.
		return nil, errors.New("invalid backend key hash")
	}
	p.RLock()
	cc, ok := p.conns[keyHash(kh)]
	p.RUnlock()
	if !ok {
		// TODO: return this as a response instead.
		return nil, errors.New("backend unavailable")
	}
	return cc.RoundTrip(r)
}

func (p *backendConnectionsPool) handleBackend(hs *http.Server, c *tls.Conn, h http.Handler) {
	backend := sha256.Sum256(c.ConnectionState().PeerCertificates[0].PublicKey.(ed25519.PublicKey))
	t := &http2.Transport{
		// Send a PING every 15s, with the default 15s timeout.
		ReadIdleTimeout: 15 * time.Second,
	}
	cc, err := t.NewClientConn(c)
	if err != nil {
		log.Printf("%x: failed to convert to HTTP/2 client connection: %v", backend, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cc.Ping(ctx); err != nil {
		log.Printf("%x: did not respond to PING: %v", backend, err)
		return
	}

	p.Lock()
	if oldCC, ok := p.conns[backend]; ok && !oldCC.State().Closed {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			oldCC.Shutdown(ctx)
		}()
	}
	p.conns[backend] = cc
	p.Unlock()

	log.Printf("%x: accepted new backend connection", backend)
	// We need not to return, or http.Server will close this connection. There
	// is no way to wait for the ClientConn's closing, so we poll. We could
	// switch this to a Server.ConnState callback with some plumbing.
	for !cc.State().Closed {
		time.Sleep(1 * time.Second)
	}
	log.Printf("%x: backend connection expired", backend)
}
