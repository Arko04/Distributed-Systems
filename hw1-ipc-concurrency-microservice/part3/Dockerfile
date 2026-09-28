FROM golang:1.21-alpine AS builder
WORKDIR /app

# Ensure modules are ready (if you later add go.mod)
# COPY go.mod go.sum ./
# RUN go mod download

COPY main.go .
# Build a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
