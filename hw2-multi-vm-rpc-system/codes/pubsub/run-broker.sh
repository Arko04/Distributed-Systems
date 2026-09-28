#!/bin/bash
# run-broker.sh – Starts the PubSub broker with default settings.

set -e

PORT="${PORT:-9090}"
echo "Starting PubSub broker on port $PORT"
go run broker.go -addr ":$PORT" -verbose true