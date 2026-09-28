#!/bin/bash
# install.sh - Sets up the authentication service environment

set -e

echo "🔧 Installing dependencies for Auth VM"

# Set Go proxy (use Iranian mirror if needed)
go env -w GOPROXY=https://go.devneeds.ir,direct
go env -w GOSUMDB=off

# Download Go modules
go mod tidy

# Install protoc plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate gRPC stubs
protoc --go_out=. --go-grpc_out=. auth.proto

echo "✅ Setup complete. Run './run.sh' to start the server."