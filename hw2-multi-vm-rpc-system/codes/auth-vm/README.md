# Auth VM – Complete Authentication Service (gRPC)

## Table of Contents

- [Auth VM – Complete Authentication Service (gRPC)](#auth-vm--complete-authentication-service-grpc)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Architecture \& Design](#architecture--design)
  - [File Structure](#file-structure)
  - [Prerequisites](#prerequisites)
  - [Quick Start](#quick-start)
  - [Detailed Setup Instructions](#detailed-setup-instructions)
    - [Step 1: Copy files to VM2](#step-1-copy-files-to-vm2)
    - [Step 2: Install Go dependencies](#step-2-install-go-dependencies)
    - [Step 3: Verify stub generation](#step-3-verify-stub-generation)
    - [Step 4: Customize user data (optional)](#step-4-customize-user-data-optional)
  - [Running the Service](#running-the-service)
    - [Basic run (defaults)](#basic-run-defaults)
    - [With custom port and user file](#with-custom-port-and-user-file)
    - [Using the run script](#using-the-run-script)
    - [Using Makefile](#using-makefile)
    - [Running as a background service (systemd)](#running-as-a-background-service-systemd)
  - [Configuration](#configuration)
  - [Testing the Service](#testing-the-service)
    - [1. Using `grpcurl` (recommended)](#1-using-grpcurl-recommended)
    - [2. Using the provided test client (`test_client.go`)](#2-using-the-provided-test-client-test_clientgo)
    - [3. Negative test (wrong password)](#3-negative-test-wrong-password)
    - [4. Missing fields test](#4-missing-fields-test)
  - [Logs \& Monitoring](#logs--monitoring)
  - [Integration with Other VMs](#integration-with-other-vms)
    - [VM1 (Web Service) expects:](#vm1-web-service-expects)
    - [Network requirements:](#network-requirements)
    - [Testing integration from VM1:](#testing-integration-from-vm1)
  - [Troubleshooting](#troubleshooting)
  - [Makefile Targets](#makefile-targets)
  - [Security Considerations](#security-considerations)
  - [Extra Credit: Password Hashing](#extra-credit-password-hashing)
    - [Steps:](#steps)
  - [Graceful Shutdown](#graceful-shutdown)
  - [Complete Example Session](#complete-example-session)
  - [References](#references)

---

## Overview

This component is **VM2 – Authentication Service** for the distributed system assignment. It provides a **gRPC-based RPC** service that validates user credentials. The service:

- Reads user data from a JSON file (`users.json`) stored locally on VM2.
- Exposes a single RPC method: `Login(LoginRequest) returns (LoginResponse)`.
- Returns `success: true` only if the username and password match an entry.
- Is completely independent of the web service (VM1) and file service (VM3).
- Implements graceful shutdown and detailed logging.

This service fulfills **Section 7‑4** of the assignment specification.

---

## Architecture & Design

```
┌─────────────────────────────────────────────────────────┐
│                     VM2 (Auth Service)                   │
│                                                           │
│  ┌─────────────┐    ┌──────────────┐    ┌────────────┐  │
│  │  gRPC Server │◄───│   AuthServer │    │  users.json│  │
│  │   :50051     │    │   (impl)     │───►│  (storage) │  │
│  └─────────────┘    └──────────────┘    └────────────┘  │
│         ▲                                                │
│         │ gRPC (TCP)                                     │
│         │                                                │
└─────────┼────────────────────────────────────────────────┘
          │
          │ network
          │
          ▼
    ┌──────────┐
    │ VM1 (Web) │  (calls Login RPC)
    └──────────┘
```

**Key design decisions**:
- **gRPC** chosen for type safety, performance, and code generation.
- **Plaintext passwords** for simplicity (extra credit version shows bcrypt).
- **Stateless** – no session storage; each login request is independent.
- **Graceful shutdown** ensures no in‑flight requests are dropped.

---

## File Structure

After running the setup, the `auth-vm/` directory will contain:

```
auth-vm/
├── auth.proto              # Protocol Buffers service definition (IDL)
├── users.json              # JSON file with user credentials
├── main.go                 # Main server implementation
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums (after `go mod tidy`)
├── authpb/                 # Generated gRPC stubs (created by protoc)
│   ├── auth.pb.go          # Protobuf message types
│   └── auth_grpc.pb.go     # gRPC client/server interfaces
├── Makefile                # Build automation (proto, build, run, test, clean)
├── run.sh                  # Simple launcher script
├── install.sh              # One‑time setup script
├── test_client.go          # Optional test client (for debugging)
└── README.md               # This file
```

---

## Prerequisites

Before running the authentication service, ensure the following are installed on **VM2**:

| Tool          | Minimum Version | Installation Command (Ubuntu/Debian)          |
|---------------|----------------|------------------------------------------------|
| Go            | 1.23+          | `sudo snap install go --classic` or from apt  |
| protoc        | 25.0+          | `sudo apt install protobuf-compiler`          |
| git           | any            | `sudo apt install git`                        |
| (optional) grpcurl | latest    | `go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest` |

**For macOS**: Use `brew install go protobuf grpcurl`.

**Network**: VM2 must have a static IP (e.g., `192.168.56.11`) and port `50051` open (no firewall blocking).

---

## Quick Start

```bash
# 1. Make scripts executable
chmod +x install.sh run.sh

# 2. Run the one‑time setup (downloads deps, generates stubs)
./install.sh

# 3. Start the server
./run.sh
```

The server will start and listen on `:50051` by default. You should see:
```
✅ Loaded 3 users from users.json
🚀 Auth VM listening on :50051
```

---

## Detailed Setup Instructions

### Step 1: Copy files to VM2
Copy all files from the assignment package into a directory named `auth-vm/` on VM2.

### Step 2: Install Go dependencies
The `install.sh` script does:
```bash
go env -w GOPROXY=https://go.devneeds.ir,direct   # Iranian mirror (if needed)
go env -w GOSUMDB=off
go mod tidy
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
protoc --go_out=. --go-grpc_out=. auth.proto
```

If you prefer manual steps, run them one by one.

### Step 3: Verify stub generation
After `protoc`, the `authpb/` directory should contain two `.go` files. If missing, check that `protoc-gen-go` and `protoc-gen-go-grpc` are in your `PATH` (usually `~/go/bin`).

### Step 4: Customize user data (optional)
Edit `users.json` to add/remove users. The format is:
```json
[
  {"username": "name1", "password": "pass1"},
  {"username": "name2", "password": "pass2"}
]
```

---

## Running the Service

### Basic run (defaults)
```bash
go run main.go
```

### With custom port and user file
```bash
go run main.go -port :9090 -users /path/to/users.json
```

### Using the run script
```bash
PORT=:50051 USERS=users.json ./run.sh
```

### Using Makefile
```bash
make run
```

### Running as a background service (systemd)
Create `/etc/systemd/system/auth.service`:
```
[Unit]
Description=Auth gRPC Service
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/auth-vm
ExecStart=/home/ubuntu/auth-vm/bin/auth-server -port :50051 -users users.json
Restart=always

[Install]
WantedBy=multi-user.target
```
Then:
```bash
sudo systemctl enable auth
sudo systemctl start auth
```

---

## Configuration

| Flag / Environment Variable | Default       | Description                                    |
|-----------------------------|---------------|------------------------------------------------|
| `-port`                     | `:50051`      | TCP address to listen on (e.g., `:50051` or `0.0.0.0:50051`) |
| `-users`                    | `users.json`  | Path to JSON user file (relative or absolute) |
| `PORT` (env)                | `:50051`      | Used by `run.sh`                              |
| `USERS` (env)               | `users.json`  | Used by `run.sh`                              |

**Note**: The service listens on all interfaces by default (e.g., `0.0.0.0:50051`). If you specify `:50051`, it binds to all available IPs.

---

## Testing the Service

### 1. Using `grpcurl` (recommended)

Install grpcurl:
```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

Then test:
```bash
grpcurl -plaintext -d '{"username":"alice","password":"alice123"}' 192.168.56.11:50051 auth.AuthService/Login
```

Expected output:
```json
{
  "success": true
}
```

### 2. Using the provided test client (`test_client.go`)

```bash
go run test_client.go -server 192.168.56.11:50051 -user alice -pass alice123
```

Output:
```
✅ Login successful for alice
```

### 3. Negative test (wrong password)
```bash
grpcurl -plaintext -d '{"username":"alice","password":"wrong"}' localhost:50051 auth.AuthService/Login
```
Output:
```json
{
  "success": false
}
```

### 4. Missing fields test
```bash
grpcurl -plaintext -d '{"username":"alice"}' localhost:50051 auth.AuthService/Login
```
Returns a gRPC error with code `INVALID_ARGUMENT` and message "username and password are required".

---

## Logs & Monitoring

The server logs the following events to stdout:

| Event                                   | Log Example                                                   |
|-----------------------------------------|---------------------------------------------------------------|
| Successful start                        | `🚀 Auth VM listening on :50051`                             |
| User file loaded                        | `✅ Loaded 3 users from users.json`                          |
| Successful login                        | `✅ Successful login for user: alice`                        |
| Failed login (wrong password)           | `❌ Failed login attempt for user alice: wrong password`     |
| Failed login (unknown user)             | `❌ Failed login attempt: unknown user xyz`                  |
| Shutdown signal received                | `🛑 Shutdown signal received, stopping gracefully...`        |
| Graceful stop complete                  | `✅ Auth VM stopped cleanly`                                 |

For persistent logging, redirect output:
```bash
./run.sh > auth.log 2>&1
```

Or use `journalctl -u auth` if running as a systemd service.

---

## Integration with Other VMs

### VM1 (Web Service) expects:
- gRPC client that calls `auth.AuthService/Login`.
- The web service must be configured with the IP address of VM2 and the gRPC port (default `50051`).

**Example web service command line**:
```bash
./web-server -auth=192.168.56.11:50051 -file=http://192.168.56.12:8081 -broker=192.168.56.10:9090
```

### Network requirements:
- VM1 and VM2 must be able to communicate over TCP port `50051` (no NAT, no firewall blocking).
- Use `ping` to verify connectivity, then `telnet <VM2_IP> 50051` to check port reachability.

### Testing integration from VM1:
If VM1 has `grpcurl` installed, run:
```bash
grpcurl -plaintext 192.168.56.11:50051 list
```
Should show `auth.AuthService`.

---

## Troubleshooting

| Problem                                     | Likely Cause                                  | Solution                                                                 |
|---------------------------------------------|-----------------------------------------------|--------------------------------------------------------------------------|
| `protoc: command not found`                 | protobuf compiler not installed               | `sudo apt install protobuf-compiler` (Ubuntu) or `brew install protobuf` (macOS) |
| `cannot find package "auth/authpb"`         | Stubs not generated or wrong import path      | Run `protoc --go_out=. --go-grpc_out=. auth.proto` and ensure `module auth` in `go.mod` |
| `connection refused` when testing           | Server not running, or wrong IP/port          | Run `lsof -i :50051`; check IP with `ip a`; use correct IP in client    |
| `rpc error: code = Unimplemented`           | Wrong service/method name                     | Use fully qualified name: `auth.AuthService/Login`                       |
| `go mod tidy` fails with checksum error     | Network proxy or corrupted cache              | Set `GOPROXY=direct` and `GOSUMDB=off`                                   |
| `missing go.sum` entry                      | Dependency not downloaded                     | Run `go mod download` then `go mod tidy`                                 |
| `bind: address already in use`              | Another process using port 50051              | Change port with `-port :50052` or kill the existing process             |
| Server starts but does not respond          | Firewall blocking port                        | `sudo ufw allow 50051` (Ubuntu) or disable firewall temporarily          |

---

## Makefile Targets

| Target       | Action                                                                          |
|--------------|---------------------------------------------------------------------------------|
| `make deps`  | Install protoc plugins and run `go mod tidy`                                   |
| `make proto` | Generate gRPC stubs from `auth.proto` (runs `deps` first)                      |
| `make build` | Compile the server binary into `bin/auth-server` (runs `proto`)                |
| `make run`   | Build and run the server (default port :50051, users.json)                     |
| `make test`  | Test using `grpcurl` (requires grpcurl in PATH)                                |
| `make clean` | Remove `bin/`, `authpb/`, and `go.sum`                                         |
| `make help`  | Show available targets                                                         |

**Example usage**:
```bash
make clean
make proto
make build
make run
```

---

## Security Considerations

- **Plaintext passwords**: The default implementation stores passwords as plaintext in `users.json`. This is acceptable for the assignment but **not for production**. The VM2 file system must be protected (e.g., `chmod 600 users.json`, owned by the service user).
- **No TLS**: gRPC uses insecure credentials (`grpc.WithTransportCredentials(insecure.NewCredentials())`). In a real deployment, use TLS with certificates.
- **Input validation**: Empty username/password return an `InvalidArgument` error, preventing empty string attacks.
- **No rate limiting**: The service does not limit login attempts. A malicious client could brute‑force passwords. For production, add rate limiting or account lockout.

---

## Extra Credit: Password Hashing

To receive extra design points, you can replace plaintext passwords with bcrypt hashes.

### Steps:
1. Add `"password_hash"` field to `users.json` (instead of `"password"`).
2. Hash passwords using `bcrypt.GenerateFromPassword`.
3. Modify `Login` method to compare using `bcrypt.CompareHashAndPassword`.

**Modified `User` struct**:
```go
type User struct {
    Username     string `json:"username"`
    PasswordHash string `json:"password_hash"`
}
```

**Modified `Login` method**:
```go
import "golang.org/x/crypto/bcrypt"

// ... in Login:
for _, u := range users {
    if u.Username == username {
        err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
        if err == nil {
            log.Printf("✅ Successful login for user: %s", username)
            return &pb.LoginResponse{Success: true}, nil
        }
        log.Printf("❌ Failed login attempt for user %s: wrong password", username)
        return &pb.LoginResponse{Success: false}, nil
    }
}
```

**Example `users.json` with bcrypt** (hash for "alice123"):
```json
[
  {"username": "alice", "password_hash": "$2a$10$N9qo8uLOickgx2ZMRZoMy.Mr/.2Z5QeUvU6R3Y7Qz3X5L6K7M8N9O"},
  ...
]
```

Don’t forget to add `golang.org/x/crypto` to `go.mod`.

---

## Graceful Shutdown

The server listens for `SIGINT` (Ctrl+C) and `SIGTERM` (systemd stop). On receiving the signal:
1. It stops accepting new connections.
2. It waits for existing RPC calls to finish (up to 5 seconds).
3. It then exits.

This ensures that a login request in progress is not abruptly terminated.

**Example shutdown log**:
```
^C
🛑 Shutdown signal received, stopping gracefully...
✅ Auth VM stopped cleanly
```

---

## Complete Example Session

Here is a full terminal session from setup to test:

```bash
$ cd auth-vm/
$ chmod +x install.sh run.sh
$ ./install.sh
🔧 Installing dependencies for Auth VM
go: downloading google.golang.org/grpc v1.68.0
...
✅ Setup complete. Run './run.sh' to start the server.

$ ./run.sh
Starting Auth VM on port :50051 with user file users.json
2026/06/03 14:00:01 ✅ Loaded 3 users from users.json
2026/06/03 14:00:01 🚀 Auth VM listening on :50051

# In another terminal on VM2 (or from VM1):
$ grpcurl -plaintext -d '{"username":"alice","password":"alice123"}' localhost:50051 auth.AuthService/Login
{
  "success": true
}

# Server log shows:
2026/06/03 14:00:05 ✅ Successful login for user: alice

# Press Ctrl+C on the server terminal:
^C
2026/06/03 14:01:00 🛑 Shutdown signal received, stopping gracefully...
2026/06/03 14:01:00 ✅ Auth VM stopped cleanly
```

---

## References

- [gRPC Go Quickstart](https://grpc.io/docs/languages/go/quickstart/)
- [Protocol Buffers documentation](https://protobuf.dev/)
- [bcrypt package](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
- Assignment specification – Sections 7‑4 (Authentication Service), 8‑8 (RPC technology choice)
- Course slides – Dr. Shourniya, University of Tehran, Spring 2026

---

**Author:** Taha Majlesi  
**Course:** Fundamentals of Distributed Computing  
**University of Tehran – Spring 1405 (2026)**

**End of README**
