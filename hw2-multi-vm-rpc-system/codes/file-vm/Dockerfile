FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o file-server main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/file-server .
# Copy the files directory from the builder to the final image
COPY --from=builder /app/files ./files
EXPOSE 8081
CMD ["./file-server"]