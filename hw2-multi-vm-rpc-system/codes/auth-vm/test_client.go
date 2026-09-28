// Simple gRPC client to test the auth service.
// Build with: go build -o test-client test_client.go
// Run: ./test-client -server 192.168.56.11:50051 -user alice -pass alice123
package main

import (
    "context"
    "flag"
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    pb "auth/authpb"
)

func main() {
    serverAddr := flag.String("server", "localhost:50051", "auth server address")
    username := flag.String("user", "alice", "username")
    password := flag.String("pass", "alice123", "password")
    flag.Parse()

    conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()
    client := pb.NewAuthServiceClient(conn)

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    resp, err := client.Login(ctx, &pb.LoginRequest{Username: *username, Password: *password})
    if err != nil {
        log.Fatalf("RPC error: %v", err)
    }
    if resp.Success {
        log.Printf("✅ Login successful for %s", *username)
    } else {
        log.Printf("❌ Login failed for %s", *username)
    }
}