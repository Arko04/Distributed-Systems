#!/bin/bash
# run-subscriber.sh – Connects to the broker and listens for alerts.

set -e

BROKER="${BROKER:-localhost:9090}"
echo "Starting subscriber, connecting to $BROKER"
go run subscriber.go -broker "$BROKER" -verbose true