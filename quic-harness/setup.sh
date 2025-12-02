#!/bin/bash
set -e

# This script creates a patched copy of quic-go with a very long initial RTT
# to prevent retransmits during debugging.

QUIC_GO_VERSION="v0.57.1"

echo "Fetching quic-go source..."
MODPATH=$(go mod download -json "github.com/quic-go/quic-go@${QUIC_GO_VERSION}" | jq -r .Dir)

echo "Creating patched copy..."
rm -rf quic-go-patched
cp -r "$MODPATH" quic-go-patched
chmod -R u+w quic-go-patched

echo "Applying RTT patch..."
sed -i.bak 's/const DefaultInitialRTT = 100 \* time.Millisecond/const DefaultInitialRTT = 365 * 24 * time.Hour  \/\/ PATCHED/' \
    quic-go-patched/internal/utils/rtt_stats.go
rm quic-go-patched/internal/utils/rtt_stats.go.bak

echo "Done! quic-go-patched/ created with 1-year initial RTT."
echo "Run 'go build' to build the harness."
