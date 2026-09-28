#!/bin/bash
# run.sh - Start the authentication server with default parameters

set -e

PORT="${PORT:-:50051}"
USERS="${USERS:-users.json}"

echo "Starting Auth VM on port $PORT with user file $USERS"
go run main.go -port "$PORT" -users "$USERS"