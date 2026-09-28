# PubSub – Simple TCP Broker & Subscriber

This is a lightweight **publish‑subscribe** system over raw TCP.  
- The **broker** accepts two commands: `pub` (publish a message) and `sub` (subscribe to all future messages).  
- The **subscriber** connects, sends `sub`, and prints every received message as a nicely formatted alert.

It is used in this assignment to forward **memory usage alerts** from the Web VM (publisher) to a monitoring terminal (subscriber).

---

## 📁 Folder Structure

```
pubsub/
├── broker.go             # TCP broker implementation
├── subscriber.go         # Subscriber client implementation
├── go.mod                # Go module definition
├── go.sum                # Dependency checksums (none, pure stdlib)
└── README.md             # This file
```

---

## 🛠️ Prerequisites

- **Go 1.23+** (recommended 1.25)
- No external dependencies – uses only the standard library.
- The broker and subscriber can run on **any machine** as long as they can reach each other over the network.

---

## 📦 Installation & Setup

### 1. Copy the `pubsub` folder to your target machines (or a single machine).

### 2. (Optional) Set Go module proxy (not required for stdlib, but may be needed for other tools):
```bash
go env -w GOPROXY=https://go.devneeds.ir,direct
go env -w GOSUMDB=off
```

### 3. Initialize module (if `go.mod` missing):
```bash
go mod init pubsub
go mod tidy
```

---

## 🚀 Running the Broker

The broker listens for incoming TCP connections, accepts `pub` and `sub` commands, and broadcasts published messages to all active subscribers.

### Basic run (default port `:9090`):
```bash
go run broker.go
```

### With custom listen address:
```bash
go run broker.go -addr :9090
# or with specific IP:
go run broker.go -addr 192.168.1.10:9090
```

### Command‑line flags

| Flag   | Default | Description                                     |
|--------|---------|-------------------------------------------------|
| `-addr` | `:9090` | TCP address to listen on (e.g., `:9090`, `0.0.0.0:9090`, `192.168.1.10:9090`). |

When the broker starts, you will see:
```
🚀 PubSub broker listening on :9090
```

### Graceful shutdown
Press `Ctrl+C` – the broker will close all subscriber connections and exit cleanly.

---

## 🚀 Running the Subscriber

The subscriber connects to the broker, sends the `sub` command, and prints every received JSON event as a formatted alert.

### Basic run (connects to `localhost:9090`):
```bash
go run subscriber.go
```

### With custom broker address:
```bash
go run subscriber.go -broker 192.168.1.10:9090
```

### Command‑line flags

| Flag     | Default           | Description                               |
|----------|-------------------|-------------------------------------------|
| `-broker` | `localhost:9090` | Broker address (host:port) to connect to. |

After connecting, the subscriber waits silently. It prints:
```
✅ Connected to broker at localhost:9090, waiting for events...
```

When a message is received, it shows:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🚨 HIGH_MEMORY_USAGE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Service    : web-server
Memory     : 400 MB
Threshold  : 300 MB
Timestamp  : 2026-05-11T04:32:10Z
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### Graceful shutdown
Press `Ctrl+C` – the subscriber closes the connection and exits.

---

## 🔌 Protocol Specification

The broker uses a **line-based** protocol over TCP:

1. **Subscribe** – Client sends:
   ```
   sub\n
   ```
   Then the broker keeps the connection open and sends every future message as:
   ```
   <JSON>\n
   ```

2. **Publish** – Client sends:
   ```
   pub\n
   <JSON>\n
   ```
   The broker then forwards the `<JSON>` line to **all currently connected subscribers**. The publishing connection is closed immediately after the message is sent.

> **Note:** The JSON message must be a single line (no embedded newlines). The Web VM already formats its event correctly.

---

## 🧪 Testing the Broker & Subscriber Manually

You can test the system without the Web VM using `nc` (netcat) or `telnet`.

### Terminal 1 – Start the broker:
```bash
go run broker.go -addr :9090
```

### Terminal 2 – Start a subscriber:
```bash
go run subscriber.go -broker 127.0.0.1:9090
```

### Terminal 3 – Publish a test message using `nc`:
```bash
echo -e "pub\n{\"event_type\":\"TEST\",\"service\":\"manual\",\"memory_mb\":123,\"threshold_mb\":300,\"timestamp\":\"now\"}\n" | nc 127.0.0.1 9090
```

The subscriber (Terminal 2) will immediately print the formatted alert.

---

## 🔥 Troubleshooting

| Problem | Likely cause | Solution |
|---------|--------------|----------|
| `connection refused` when starting subscriber | Broker not running or wrong address | Start broker first; verify `-broker` address. |
| `unknown command` in broker logs | Client sent something other than `pub` or `sub` | Ensure the first line is exactly `pub\n` or `sub\n`. |
| Subscriber prints `bad event: unexpected end of JSON input` | Received empty line or malformed JSON | The Web VM may send an extra newline; this is harmless. |
| Subscriber disconnects after a few seconds | Network timeout or broker restart | Use a more robust retry logic (not included, but subscriber exits gracefully). |
| `address already in use` | Another process using the same port | Change the port with `-addr :9091` or kill the old process. |

---

## 📝 Example `go.mod`

```go
module pubsub

go 1.23
```

(No external dependencies.)

---

## 📞 Integration with the Assignment

- **Web VM** (publisher) – configured with the `-broker` flag (default `127.0.0.1:9090`). When memory exceeds the threshold, it connects, sends `pub\n<JSON>\n`, and closes.
- **Subscriber** – run on any VM (e.g., VM2 or a monitoring machine) to receive and print alerts.
- **Broker** – must be running before any publisher or subscriber can connect.

**Recommended deployment:**
- Broker on the same machine as the Web VM (VM1) for low latency.
- Subscriber on a monitoring terminal (e.g., VM2 or the instructor’s machine).

---

## ✅ Verification Checklist

- [ ] `go run broker.go` prints `PubSub broker listening on :9090`.
- [ ] `go run subscriber.go -broker 127.0.0.1:9090` connects successfully.
- [ ] Using `nc` to publish a test message produces an alert in the subscriber terminal.
- [ ] When the Web VM allocates memory (`curl .../consume-memory?mb=400`), the subscriber prints the memory alert within 10 seconds.

---

## 📚 References

- [TCP protocol design](https://en.wikipedia.org/wiki/Transmission_Control_Protocol)
- [JSON format](https://www.json.org/)
- Assignment spec – Part 3 (Pub/Sub and memory monitoring)

---

**Author:** Taha Majlesi  
**Course:** Fundamentals of Distributed Computing – University of Tehran  
**Spring 1405 (2026)**
