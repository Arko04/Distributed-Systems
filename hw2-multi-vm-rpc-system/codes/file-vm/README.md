# File VM – Static HTTP File Server

This is **VM3** in the distributed system. It provides a simple, lightweight HTTP server that serves static files (images, documents, etc.) from a configurable directory. The web service (VM1) fetches images from this server to display on the dashboard.

The server is built using only the Go standard library (`net/http`), has no external dependencies, and includes:

- Configurable port and directory (via command-line flags)
- Graceful shutdown on `SIGINT`/`SIGTERM`
- Request logging (method, path, client IP)
- Timeouts to prevent slowloris attacks
- Health check endpoint (`/health`)
- Path traversal protection (built into `http.Dir`)

---

## 📁 Folder Structure (after setup)

```
file-vm/
├── main.go                    # HTTP file server implementation
├── go.mod                     # Go module definition (optional, no external deps)
├── go.sum                     # Empty (no external dependencies)
├── files/                     # Directory containing served files
│   ├── sample.jpg             # Example image (downloaded by install.sh)
│   └── ... (any other files)
├── install.sh                 # One-time setup script (creates files/, downloads sample)
├── run.sh                     # Simple launcher script
├── Makefile                   # Build automation
├── test_client.go             # Optional HTTP client for testing
└── README.md                  # This file
```

---

## 🛠️ Prerequisites

- **Go 1.23+** installed on VM3.
- No external dependencies – only the standard library.
- The directory you intend to serve must exist and be readable.

---

## 📦 Installation & Setup

### 1. Copy the `file-vm/` folder to VM3.

### 2. Make scripts executable:

```bash
chmod +x install.sh run.sh
```

### 3. Run the setup script:

```bash
./install.sh
```

This script will:
- Create the `files/` directory (if missing).
- Download a sample image (`sample.jpg`) from the internet (or use a local fallback).
- Initialize a Go module (if `go.mod` missing).
- Run `go mod tidy` (no external packages, but keeps things clean).

### 4. (Optional) Add your own files:

```bash
cp /path/to/your/image.jpg files/
cp /path/to/document.pdf files/
```

---

## 🚀 Running the File Server

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
go run main.go -port 8081 -dir ./files
```

### Command-line flags

| Flag              | Default       | Description                                      |
|-------------------|---------------|--------------------------------------------------|
| `-port`           | `8081`        | Port to listen on (e.g., `8081`, `:8081`).       |
| `-dir`            | `./files`     | Directory to serve files from (relative or absolute). |
| `-read-timeout`   | `5s`          | Maximum duration for reading the entire request. |
| `-write-timeout`  | `10s`         | Maximum duration for writing the response.       |
| `-idle-timeout`   | `30s`         | Maximum time to keep idle connections alive.     |

> **Note:** If `-port` does not start with a colon, it is automatically prefixed (e.g., `8081` → `:8081`).  
> The server serves files under the `/files/` path prefix. For example, a file `sample.jpg` in `./files/` is accessible at `http://<VM3_IP>:8081/files/sample.jpg`.

---

## 🧪 Testing the File Server

### 1. After starting the server, test with `curl`:

```bash
curl -I http://127.0.0.1:8081/files/sample.jpg
```

Expected output:
```
HTTP/1.1 200 OK
Content-Type: image/jpeg
...
```

### 2. Download the file:

```bash
curl http://127.0.0.1:8081/files/sample.jpg -o downloaded.jpg
```

### 3. Open in a browser:

```
http://<VM3_IP>:8081/files/sample.jpg
```

### 4. Health check endpoint:

```bash
curl http://127.0.0.1:8081/health
```

Returns `OK` (HTTP 200).

### 5. Using the optional test client (`test_client.go`):

```bash
go run test_client.go -url http://localhost:8081/files/sample.jpg -output test.jpg
```

---

## 🛡️ Security Features

| Feature                      | How it works                                                                 |
|------------------------------|------------------------------------------------------------------------------|
| **Path traversal protection** | `http.Dir` automatically cleans paths and prevents `../` attacks.           |
| **Directory listing disabled** | The server serves only explicit files; directory indexes are disabled.      |
| **Timeouts**                  | `ReadTimeout`, `WriteTimeout`, `IdleTimeout` prevent slowloris and hung connections. |
| **Logging**                   | Each request is logged (method, path, client IP) for auditing.              |
| **No directory browsing**     | If a client requests a directory, they get `404 Not Found` (unless an `index.html` exists – we don't have one). |

---

## ⚙️ Graceful Shutdown

The server listens for `SIGINT` (Ctrl+C) and `SIGTERM` (systemd stop). On receiving a signal:

1. It stops accepting new connections.
2. It waits for existing requests to finish (up to 10 seconds).
3. It exits cleanly.

Example output:

```
^C
🛑 Shutdown signal received, stopping gracefully...
✅ File VM stopped cleanly
```

---

## 🔧 Configuration with Environment Variables (run.sh)

The `run.sh` script respects the following environment variables (or you can edit the script):

| Variable    | Default   | Description                |
|-------------|-----------|----------------------------|
| `PORT`      | `8081`    | Port to listen on          |
| `DIR`       | `./files` | Directory to serve files   |

Example:

```bash
export PORT=9090
export DIR=/var/www
./run.sh
```

---

## 🧩 Integration with the Assignment

- **VM1 (Web VM)** fetches images from this server using the base URL provided by the `-file` flag (e.g., `http://192.168.56.12:8081`).  
  The dashboard then constructs the full URL: `<baseURL>/files/sample.jpg`.
- The File VM is **independent** – it does not call any other service.

---

## 🔥 Troubleshooting

| Problem | Likely Cause | Solution |
|---------|--------------|----------|
| `404 Not Found` when requesting a file | File missing or wrong path | Ensure the file exists inside the `-dir` directory and the URL path starts with `/files/`. |
| `connection refused` | Server not running or wrong port | Check if the server is running (`lsof -i :8081`). Ensure the port is not blocked by a firewall (`sudo ufw allow 8081`). |
| `open ./files: no such file or directory` | The `-dir` does not exist | Create the directory: `mkdir -p ./files`. |
| `permission denied` | File permissions too restrictive | Make the file readable: `chmod 644 files/*`. |
| `go: go.mod file not found` | No Go module | Run `go mod init file` in the directory. |
| `bind: address already in use` | Another process using the same port | Change the port with `-port 8082` or kill the existing process. |

---

## 📝 Makefile Targets

| Target   | Action                                                                 |
|----------|------------------------------------------------------------------------|
| `make run`   | Run the server with default flags (port 8081, dir ./files).          |
| `make build` | Compile the server binary into `bin/file-server`.                    |
| `make test`  | Test the running server with `curl` (requires server to be running). |
| `make clean` | Remove the `bin/` directory and `go.sum`.                            |
| `make help`  | Show available targets.                                              |

Example:

```bash
make build
./bin/file-server -port 8081 -dir ./files
```

---

## 📄 Complete `main.go` (code summary)

The server is implemented in a single file `main.go`. Key features:

- **Command-line flags** using the `flag` package.
- **Logging middleware** that prints each request.
- **Health check** endpoint (`/health`).
- **Graceful shutdown** with a 10‑second timeout.
- **Customisable timeouts** to avoid resource exhaustion.

The full code is available in the submission archive. Below is a functional summary:

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
    "syscall"
    "time"
)

var (
    port        = flag.String("port", "8081", "Port to listen on")
    dir         = flag.String("dir", "./files", "Directory to serve")
    readTimeout = flag.Duration("read-timeout", 5*time.Second, "Read timeout")
    writeTimeout = flag.Duration("write-timeout", 10*time.Second, "Write timeout")
    idleTimeout  = flag.Duration("idle-timeout", 30*time.Second, "Idle timeout")
)

func main() {
    flag.Parse()
    addr := *port
    if addr[0] != ':' {
        addr = ":" + addr
    }
    absDir, _ := filepath.Abs(*dir)
    log.Printf("📁 Serving files from: %s", absDir)

    fs := http.FileServer(http.Dir(*dir))
    mux := http.NewServeMux()
    mux.Handle("/files/", http.StripPrefix("/files/", loggingMiddleware(fs)))
    mux.HandleFunc("/health", healthHandler)

    srv := &http.Server{Addr: addr, Handler: mux, ReadTimeout: *readTimeout, WriteTimeout: *writeTimeout, IdleTimeout: *idleTimeout}

    go func() {
        log.Printf("🚀 File VM serving on %s", addr)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()

    stop := make(chan os.Signal, 1)
    signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
    <-stop
    log.Println("🛑 Shutting down...")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
    log.Println("✅ Stopped cleanly")
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("📥 %s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
        next.ServeHTTP(w, r)
    })
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}
```

---

## ✅ Verification Checklist

- [ ] `./install.sh` completes without errors.
- [ ] `./run.sh` starts the server and logs `🚀 File VM serving on :8081`.
- [ ] `curl http://127.0.0.1:8081/files/sample.jpg` returns the image (or `404` if not present).
- [ ] `curl http://127.0.0.1:8081/health` returns `OK`.
- [ ] The Web VM (VM1) can fetch and display the image using the configured `-file` URL.
- [ ] Pressing Ctrl+C stops the server gracefully.

---

## 📚 References

- [Go `net/http` package documentation](https://pkg.go.dev/net/http)
- [Go `http.FileServer` example](https://pkg.go.dev/net/http#FileServer)
- [Graceful shutdown in Go](https://pkg.go.dev/net/http#Server.Shutdown)

---

**Author:** Taha Majlesi  
**Course:** Fundamentals of Distributed Computing – University of Tehran  
**Spring 1405 (2026)**