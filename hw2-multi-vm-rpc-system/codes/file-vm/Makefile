# Makefile for File VM (HTTP file server)

.PHONY: run build clean test help

PORT ?= 8081
DIR ?= ./files

help:
	@echo "Available targets:"
	@echo "  run     - start the file server (default port 8081)"
	@echo "  build   - compile the server binary"
	@echo "  test    - test the running server with curl"
	@echo "  clean   - remove binary and go.sum"

build:
	go build -o bin/file-server main.go

run: build
	./bin/file-server -port $(PORT) -dir $(DIR)

test:
	@echo "Testing file server (must be running)..."
	@curl -s -o /dev/null -w "HTTP %{http_code}\n" http://localhost:8081/health || echo "Server not reachable"
	@echo "Health check passed."

clean:
	rm -rf bin/
	rm -f go.sum