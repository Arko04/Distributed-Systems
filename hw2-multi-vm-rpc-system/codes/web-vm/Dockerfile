FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o web-server main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/web-server .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/authpb ./authpb
EXPOSE 8080
CMD ["./web-server"]