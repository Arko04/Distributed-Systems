#!/bin/bash
# run.sh - Start the Web VM with default parameters.
# Override any flag by setting environment variables or passing arguments.

set -e

AUTH="${AUTH:-127.0.0.1:50051}"
FILE="${FILE:-http://127.0.0.1:8081}"
BROKER="${BROKER:-127.0.0.1:9090}"
THRESHOLD="${THRESHOLD:-300}"
LISTEN="${LISTEN:-:8080}"
INTERVAL="${INTERVAL:-10s}"
VERBOSE="${VERBOSE:-true}"

echo "Starting Web VM with:"
echo "  Auth VM: $AUTH"
echo "  File VM: $FILE"
echo "  Broker:  $BROKER"
echo "  Threshold: $THRESHOLD MB"
echo "  Listen:  $LISTEN"
echo "  Interval: $INTERVAL"

go run main.go \
    -auth "$AUTH" \
    -file "$FILE" \
    -broker "$BROKER" \
    -threshold "$THRESHOLD" \
    -listen "$LISTEN" \
    -interval "$INTERVAL" \
    -verbose "$VERBOSE"