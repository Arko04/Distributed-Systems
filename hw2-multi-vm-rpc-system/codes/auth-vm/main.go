// Package main implements the authentication service (gRPC) for the distributed system.
// It reads user credentials from a JSON file and provides a Login RPC.
// Graceful shutdown ensures the server stops cleanly on SIGINT/SIGTERM.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "auth/authpb" // adjust import path to match your module structure
)

// User represents a single user entry in the JSON file.
// For simplicity, passwords are stored in plaintext.
// (Extra credit: hashing is shown as commented code below.)
type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// PasswordHash string `json:"password_hash"` // for hashed version
}

var (
	users        []User
	usersFile    = flag.String("users", "users.json", "Path to users JSON file")
	port         = flag.String("port", ":50051", "Port to listen on (e.g., :50051 or 0.0.0.0:50051)")
)

// loadUsers reads and parses the JSON user file.
func loadUsers(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("cannot read user file: %w", err)
	}
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("invalid JSON in user file: %w", err)
	}
	if len(users) == 0 {
		log.Println("⚠️ Warning: No users loaded from file")
	}
	log.Printf("✅ Loaded %d users from %s", len(users), filename)
	return nil
}

// authServer implements the gRPC AuthServiceServer interface.
type authServer struct {
	pb.UnimplementedAuthServiceServer
}

// Login checks the username and password against the in-memory user list.
// For extra security, you could hash the password and compare with stored hash.
func (s *authServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	username := req.GetUsername()
	password := req.GetPassword()

	if username == "" || password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}

	for _, u := range users {
		if u.Username == username {
			// Plaintext comparison (simple)
			if u.Password == password {
				log.Printf("✅ Successful login for user: %s", username)
				return &pb.LoginResponse{Success: true}, nil
			}
			// Example for hashed password (extra credit):
			// if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil {
			//     return &pb.LoginResponse{Success: true}, nil
			// }
			log.Printf("❌ Failed login attempt for user %s: wrong password", username)
			return &pb.LoginResponse{Success: false}, nil
		}
	}
	log.Printf("❌ Failed login attempt: unknown user %s", username)
	return &pb.LoginResponse{Success: false}, nil
}

// startGRPCServer launches the gRPC server and waits for shutdown signal.
func startGRPCServer(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterAuthServiceServer(grpcServer, &authServer{})

	// Run server in a goroutine so we can handle shutdown
	go func() {
		log.Printf("🚀 Auth VM listening on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("❌ gRPC server error: %v", err)
		}
	}()

	// Wait for interrupt signal (SIGINT/SIGTERM)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("🛑 Shutdown signal received, stopping gracefully...")
	grpcServer.GracefulStop()
	log.Println("✅ Auth VM stopped cleanly")
	return nil
}

func main() {
	flag.Parse()

	// Load user data
	if err := loadUsers(*usersFile); err != nil {
		log.Fatalf("❌ Failed to load users: %v", err)
	}

	// Start server with graceful shutdown
	if err := startGRPCServer(*port); err != nil {
		log.Fatalf("❌ Server error: %v", err)
	}
}