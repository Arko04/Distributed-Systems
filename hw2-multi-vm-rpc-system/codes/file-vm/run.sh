#!/bin/bash
# run.sh - Start the file server with default parameters

set -e

PORT="${PORT:-8081}"
DIR="${DIR:-./files}"

echo "Starting File VM on port $PORT serving directory $DIR"
go run main.go -port "$PORT" -dir "$DIR"