module filippo.io/mostly-harmless/quic-harness

go 1.25.4

require github.com/quic-go/quic-go v0.57.1

require (
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
)

replace github.com/quic-go/quic-go => ./quic-go-patched
