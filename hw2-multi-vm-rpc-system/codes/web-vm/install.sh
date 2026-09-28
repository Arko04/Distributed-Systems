#!/bin/bash
# install.sh - Setup script for Web VM
# It ensures the required Go modules are installed and attempts to generate
# gRPC stubs if auth.proto is present.

set -e

echo "🔧 Setting up Web VM dependencies"

# Optional: set Go proxy for Iranian networks
go env -w GOPROXY=https://go.devneeds.ir,direct
go env -w GOSUMDB=off

# Download Go modules
go mod tidy

# Check if auth.proto exists (copy from auth-vm if needed)
if [ -f auth.proto ]; then
    echo "📄 Generating gRPC stubs from auth.proto..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    protoc --go_out=. --go-grpc_out=. auth.proto
    mkdir -p authpb
    mv auth.pb.go auth_grpc.pb.go authpb/ 2>/dev/null || true
    echo "✅ Stubs generated"
else
    echo "⚠️ auth.proto not found. Please copy authpb/ folder from auth-vm or provide auth.proto."
    echo "   You can manually copy the generated 'authpb' directory from the auth-vm machine."
fi

# Ensure templates directory exists
mkdir -p templates

echo "✅ Setup complete. Run './run.sh' to start the web server."