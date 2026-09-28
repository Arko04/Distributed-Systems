# Web VM – Frontend Web Service

This is **VM1** in the distributed system. It provides:

- A **login page** that authenticates users by calling the **Auth VM** (VM2) via **gRPC**.
- A **dashboard** that displays an image retrieved from the **File VM** (VM3) over HTTP.
- A **memory monitoring** goroutine that checks the Go heap every 10 seconds (configurable).
- A **Pub/Sub publisher** that sends a JSON event to a broker when memory exceeds a threshold (default 300 MB).
- An endpoint `/consume-memory` to manually allocate memory for testing alerts.

All external service addresses are configurable via command-line flags, making it easy to deploy on separate VMs.

---

## 📁 Folder Structure (after setup)

```
web-vm/
├── main.go                    # Main web server implementation
├── go.mod                     # Go module definition
├── go.sum                     # Dependency checksums
├── templates/
│   ├── login.html             # Login page template
│   └── dashboard.html         # Dashboard template
├── authpb/                    # Generated gRPC stubs (copied from auth-vm)
│   ├── auth.pb.go
│   └── auth_grpc.pb.go
├── install.sh                 # One-time setup script
├── run.sh                     # Simple launcher script
├── Makefile                   # Build automation
└── README.md                  # This file
```

---

## 🛠️ Prerequisites

- **Go 1.23+** installed on VM1.
- The **Auth VM** (VM2) must be running and reachable on its gRPC port (default `50051`).
- The **File VM** (VM3) must be running and serving images via HTTP (default port `8081`).
- The **Pub/Sub broker** (part of `pubsub/`) must be running and reachable (default port `9090`).
- The generated gRPC stubs (`authpb/`) must be present inside `web-vm/`.  
  (Copy them from the `auth-vm/` directory after generating stubs there.)

---

## 📦 Installation & Setup

### 1. Copy the `web-vm/` folder to VM1.

### 2. Obtain the gRPC stubs from the Auth VM.

The web service needs the Go code generated from `auth.proto`. The easiest way:

```bash
cp -r ../auth-vm/authpb .   # if auth-vm is in the same parent directory
```

Alternatively, copy the `authpb/` folder manually from the Auth VM after running `make proto` there.

### 3. Run the setup script:

```bash
chmod +x install.sh run.sh
./install.sh
```

This will:
- Set the Go proxy (optional, for Iranian networks).
- Download Go dependencies (`go mod tidy`).
- (If `auth.proto` is present) generate stubs – but we already copied them.
- Ensure the `templates/` directory exists.

### 4. Verify that the `authpb/` directory contains `auth.pb.go` and `auth_grpc.pb.go`. If missing, repeat step 2.

---

## 🚀 Running the Web Service

### Using the run script (default flags)

```bash
./run.sh
```

### Using Makefile

```bash
make run
```

### Directly with `go run`

```bash
go run main.go -auth 192.168.56.11:50051 -file http://192.168.56.12:8081 -broker 192.168.56.10:9090
```

### Customising with environment variables (run.sh)

| Variable   | Default               | Description                        |
|------------|-----------------------|------------------------------------|
| `AUTH`     | `127.0.0.1:50051`     | Auth VM gRPC address              |
| `FILE`     | `http://127.0.0.1:8081`| File VM HTTP base URL             |
| `BROKER`   | `127.0.0.1:9090`      | Pub/Sub broker address             |
| `THRESHOLD`| `300`                 | Memory threshold in MB             |
| `LISTEN`   | `:8080`               | Web server listen address          |
| `INTERVAL` | `10s`                 | Memory check interval              |
| `VERBOSE`  | `true`                | Enable detailed logging            |

Example:

```bash
export AUTH=192.168.56.11:50051
export FILE=http://192.168.56.12:8081
export BROKER=192.168.56.10:9090
./run.sh
```

### Command-line flags (direct `go run`)

| Flag          | Default               | Description                                                |
|---------------|-----------------------|------------------------------------------------------------|
| `-auth`       | `127.0.0.1:50051`     | Auth VM gRPC address (host:port)                          |
| `-file`       | `http://127.0.0.1:8081`| File VM HTTP base URL (without trailing slash)            |
| `-broker`     | `127.0.0.1:9090`      | Pub/Sub broker address (host:port)                        |
| `-threshold`  | `300`                 | Memory threshold in MB (alerts when exceeded)             |
| `-listen`     | `:8080`               | Web server listen address (e.g., `:8080`, `0.0.0.0:8080`) |
| `-interval`   | `10s`                 | Memory check interval (e.g., `5s`, `30s`)                 |
| `-verbose`    | `true`                | Enable detailed logging                                    |

---

## 🧪 Testing the Service

### 1. Start all dependent services (Auth VM, File VM, Pub/Sub broker).

### 2. Open a browser and go to `http://<VM1_IP>:8080/login`.

Replace `<VM1_IP>` with the actual IP address of VM1 (e.g., `192.168.56.10`).

### 3. Login with credentials from `users.json` (default test users):

| Username | Password  |
|----------|-----------|
| alice    | alice123  |
| bob      | bob123    |
| admin    | admin123  |

- Wrong credentials → error message.
- Correct credentials → redirect to dashboard.

### 4. Dashboard should display an image from the File VM.  
  The image URL will be `http://<VM3_IP>:8081/files/sample.jpg`.

### 5. Trigger a memory alert:

Use the **“Consume Memory”** button on the dashboard, or send a `curl` command:

```bash
curl "http://<VM1_IP>:8080/consume-memory?mb=100"
```

Repeat until total allocated memory exceeds the threshold (300 MB).  
The memory monitor checks every `-interval` seconds (default 10s).  
If the subscriber (`pubsub/subscriber.go`) is running, it will print a formatted alert.

---

## 🧠 Design Details

- **gRPC client** – uses insecure credentials for simplicity (no TLS). In production, add TLS.
- **Memory monitoring** – uses `runtime.ReadMemStats()` to get live heap allocation (`Alloc`). This is the memory actively used by the Go program.
- **Pub/Sub publisher** – opens a new TCP connection for each alert, sends `pub\n<JSON>\n`, and closes, following the broker’s protocol.
- **Graceful shutdown** – on `SIGINT`/`SIGTERM`, the HTTP server shuts down gracefully (waits for existing requests) and the memory monitor stops.
- **Session management** – a simple cookie is used for demonstration. Real systems would use JWT or server-side sessions.
- **Templates** – standard Go `html/template` with minimal inline CSS.

---

## 🔥 Troubleshooting

| Problem | Likely Cause | Solution |
|---------|--------------|----------|
| `cannot find package "web/authpb"` | gRPC stubs missing | Copy the `authpb/` folder from `auth-vm/`. |
| `connection refused` to Auth VM | Auth VM not running or wrong IP/port | Verify Auth VM is running (`lsof -i :50051`). Check `-auth` flag. |
| Dashboard shows broken image | File VM unreachable or image missing | Ensure File VM is running and `sample.jpg` exists in `files/`. Check `-file` URL. |
| No memory alert appears | Broker not running or subscriber not subscribed | Start broker and subscriber; verify `-broker` address. |
| Memory alert never triggers | Allocated memory not reaching threshold | Allocate more MB (e.g., 4×100 MB). Ensure `-threshold` is not higher. |
| `rpc error: code = Unimplemented` | Wrong gRPC service name | The generated stub uses `auth.AuthService/Login` – this is correct. Check that the Auth VM uses the same proto package. |
| `template: login.html: no such file` | Templates missing | Ensure `templates/login.html` and `templates/dashboard.html` exist in the working directory. |
| `bind: address already in use` | Another process using port 8080 | Change `-listen` to another port (e.g., `-listen :8081`). |

---

## 📚 Makefile Targets

| Target   | Action                                                                 |
|----------|------------------------------------------------------------------------|
| `make deps` | Install protoc plugins and run `go mod tidy`.                        |
| `make proto`| Generate gRPC stubs from `auth.proto` (requires `auth.proto`).       |
| `make build`| Compile the server binary into `bin/web-server`.                     |
| `make run`  | Build and run the server (default flags).                            |
| `make clean`| Remove `bin/`, `authpb/`, and `go.sum`.                              |

> **Note:** You normally don’t need to run `make proto` if you copy `authpb/` from the Auth VM.

---

## 🔗 Integration with Other Components

| Component       | Role                                                              |
|----------------|-------------------------------------------------------------------|
| **Auth VM**     | Provides gRPC `Login` method. Called by the web service on POST `/login`. |
| **File VM**     | Serves static images over HTTP. The dashboard fetches `sample.jpg` from it. |
| **Pub/Sub broker** | Receives memory events via TCP. The web service connects as a publisher. |
| **Subscriber**  | (Optional) Receives and displays alerts. Run it on any VM to monitor memory usage. |

---

## ✅ Verification Checklist

- [ ] `go run main.go` starts without errors.
- [ ] Login page loads at `http://VM1_IP:8080/login`.
- [ ] Wrong credentials show error message.
- [ ] Correct credentials redirect to `/dashboard`.
- [ ] Dashboard shows an image from the File VM.
- [ ] `curl "http://VM1_IP:8080/consume-memory?mb=100"` increases memory.
- [ ] When total allocated memory > threshold, a log message `"Published memory event"` appears.
- [ ] If subscriber is running, it prints a formatted alert within the monitoring interval.
- [ ] Pressing Ctrl+C stops the server gracefully.

---

## 📝 Example `go.mod`

```go
module web

go 1.23

require (
    google.golang.org/grpc v1.68.0
    google.golang.org/protobuf v1.34.2
)
```

---

## 📚 References

- [gRPC Go Quickstart](https://grpc.io/docs/languages/go/quickstart/)
- [Go `net/http` package](https://pkg.go.dev/net/http)
- [Go `runtime` package (memory stats)](https://pkg.go.dev/runtime#MemStats)
- Assignment specification – Part 2 (Web VM), Part 3 (Pub/Sub & memory monitoring)

---

**Author:** Taha Majlesi  
**Course:** Fundamentals of Distributed Computing – University of Tehran  
**Spring 1405 (2026)**