#!/bin/bash
set -e

PIPE=$(mktemp -u)
mkfifo "$PIPE"
trap "rm -f $PIPE" EXIT

timeout 20 sh -c "QUIC_GO_LOG_LEVEL=debug go run . < $PIPE | tee $PIPE"
