# Makefile for Web VM

.PHONY: deps proto build run clean help

help:
	@echo "Available targets:"
	@echo "  deps   - install Go modules and protoc plugins"
	@echo "  proto  - generate gRPC stubs (requires auth.proto)"
	@echo "  build  - compile the web server binary"
	@echo "  run    - run the web server (uses default flags)"
	@echo "  clean  - remove binary and generated files"

deps:
	go mod tidy
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

proto: deps
	@if [ -f auth.adding missing files proto ]; then \
		protoc --go_out=. --go-grpc_out=. auth.proto; \
		mkdir -p authpb; \
		mv auth.pb.go auth_grpc.pb.go authpb/; \
		echo "✅ Stubs generated"; \
	else \
		echo "❌ auth.proto not found. Cannot generate stubs."; \
		exit 1; \
	fi

build: proto
	go build -o bin/web-server main.go

run: build
	./bin/web-server -auth 127.0.0.1:50051 -file http://127.0.0.1:8081 -broker 127.0.0.1:9090

clean:
	rm -rf bin/
	rm -rf authpb/
	rm -f go.sum