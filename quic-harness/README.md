# quic-harness

A test harness for QUIC packet injection experiments. It runs a quic-go client and server with intercepted packet I/O, exposing all packets and logs as JSON Lines on stdin/stdout.

## Building

```bash
./setup.sh  # Creates patched quic-go (only needed once)
go build .
```

The setup script creates `quic-go-patched/` with a 1-year initial RTT to prevent retransmits during debugging. This directory is gitignored.

## JSON Lines Format

All output is JSON Lines (one JSON object per line) on stdout.

### Packet Types

Packets sent by client or server (but NOT automatically delivered):

```json
{"type":"client->server","packet":"<base64>"}
{"type":"server->client","packet":"<base64>"}
```

To deliver a packet, echo it back to stdin. The harness reads packets from stdin and delivers them to the appropriate endpoint.

### Log Types

```json
{"type":"client log","message":"..."}
{"type":"server log","message":"..."}
{"type":"harness log","message":"..."}
{"type":"quic-go log","message":"..."}
```

### Qlog Events

```json
{"type":"client qlog","time":"1.234ms","event":"transport:packet_sent","data":{...}}
{"type":"server qlog","time":"5.678ms","event":"transport:packet_received","data":{...}}
```

## Running

### Echo Test

The simplest way to run the harness is to pipe stdout back to stdin, which delivers all packets immediately (like a normal connection):

```bash
./echo-test.sh
```

Or manually:

```bash
PIPE=$(mktemp -u)
mkfifo "$PIPE"
QUIC_GO_LOG_LEVEL=debug ./quic-harness < "$PIPE" | tee "$PIPE"
rm "$PIPE"
```

### Interactive Debugging

For manual packet injection:

```bash
QUIC_GO_LOG_LEVEL=debug ./quic-harness
```

The harness will:
1. Start a QUIC client and server
2. Print packets as JSON Lines to stdout
3. Wait for you to inject packets via stdin
4. Wait indefinitely (Ctrl+C to exit)

Timeouts and retransmits are set to ~1 year, so you can take your time.

### Environment Variables

- `QUIC_GO_LOG_LEVEL=debug` - Enable verbose quic-go debug logging (packet contents, state changes, etc.)

## Terminal Input Limits

When pasting or typing long JSON packets (>1024 bytes) directly into the terminal, you may hit the kernel's canonical mode line buffer limit. Symptoms: terminal bell rings, input is truncated or ignored.

**Workaround**: Disable canonical mode before running:

```bash
stty -icanon
./quic-harness
# To restore: stty icanon (or open a new terminal)
```

## Server Configuration

The server is configured with:
- `VerifySourceAddress` returning `true` - forces Retry packets on every connection
- Self-signed TLS certificate for "localhost"
- ALPN: "quic-harness"

## Example: Observing a Retry

```bash
$ QUIC_GO_LOG_LEVEL=debug ./quic-harness 2>&1 | head -30
```

You'll see:
1. Client sends Initial packets
2. Server responds with a Retry packet
3. (Nothing more until you inject packets via stdin)

To complete the handshake, copy the `client->server` and `server->client` packets and paste them back to stdin (or use the echo test).
