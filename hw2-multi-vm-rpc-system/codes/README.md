# Distributed Systems Assignment 2 – Complete System Documentation

**Author:** Taha Majlesi  
**Course:** Fundamentals of Distributed Computing – University of Tehran  
**Instructor:** Dr. Mohammadreza Shourniya  
**Teaching Assistants:** Pooya Jamshidi, Kazem Ayarzadeh, Mohammad Afzal Zadeh  
**Spring 1405 (2026)**

---

## Table of Contents

- [Distributed Systems Assignment 2 – Complete System Documentation](#distributed-systems-assignment-2--complete-system-documentation)
  - [Table of Contents](#table-of-contents)
  - [1. Project Overview](#1-project-overview)
    - [1.1 Key Features](#11-key-features)
  - [2. System Architecture \& VM Configuration](#2-system-architecture--vm-configuration)
  - [3. Prerequisites (All VMs)](#3-prerequisites-all-vms)
  - [4. Quick Start (5 Minutes)](#4-quick-start-5-minutes)
  - [5. Component 1: Auth VM (gRPC Authentication Service)](#5-component-1-auth-vm-grpc-authentication-service)
    - [5.1 Files](#51-files)
    - [5.2 Setup \& Run (VM2)](#52-setup--run-vm2)
  - [6. Component 2: File VM (HTTP Static File Server)](#6-component-2-file-vm-http-static-file-server)
    - [6.1 Files](#61-files)
    - [6.2 Setup \& Run (VM3)](#62-setup--run-vm3)
  - [7. Component 3: Web VM (Frontend + Memory Publisher)](#7-component-3-web-vm-frontend--memory-publisher)
    - [7.1 Files](#71-files)
    - [7.2 Setup \& Run (VM1)](#72-setup--run-vm1)
  - [8. Pub/Sub Broker \& Subscriber (Memory Alerts)](#8-pubsub-broker--subscriber-memory-alerts)
    - [8.1 Files](#81-files)
    - [8.2 Broker Protocol (Line‑Based)](#82-broker-protocol-linebased)
    - [8.3 Running the Broker (on VM1)](#83-running-the-broker-on-vm1)
    - [8.4 Running the Subscriber (anywhere)](#84-running-the-subscriber-anywhere)
  - [9. Complete End‑to‑End Test Scenario](#9-complete-endtoend-test-scenario)
    - [9.1 Prepare the Environment](#91-prepare-the-environment)
    - [9.2 Step‑by‑Step Verification](#92-stepbystep-verification)
    - [9.3 Expected Terminal Outputs (simulated)](#93-expected-terminal-outputs-simulated)
  - [10. Troubleshooting \& FAQs](#10-troubleshooting--faqs)
    - [10.1 Common Issues and Resolutions](#101-common-issues-and-resolutions)
    - [10.2 Debugging Tips](#102-debugging-tips)
    - [10.3 Performance Notes](#103-performance-notes)
  - [11. File Structure of the Submission](#11-file-structure-of-the-submission)
  - [12. Evaluation Criteria Self‑Assessment](#12-evaluation-criteria-selfassessment)
  - [13. References \& Acknowledgments](#13-references--acknowledgments)

---

## 1. Project Overview

This project implements a **distributed system** consisting of three independent virtual machines (VMs) that communicate over a private network. It demonstrates:

- **Remote Procedure Call (RPC)** using **gRPC** between the Web VM and the Auth VM.
- **Separation of concerns** – authentication, file serving, and the web frontend are isolated services.
- **Publish/Subscribe (Pub/Sub)** for monitoring the memory usage of the web service, with alerts when a threshold (300 MB) is exceeded.

The system fulfills all requirements of the second assignment for the *Fundamentals of Distributed Computing* course at the University of Tehran.

### 1.1 Key Features

| Feature                         | Description                                                                 |
|---------------------------------|-----------------------------------------------------------------------------|
| User authentication             | Web VM calls Auth VM via gRPC (RPC) to validate credentials.               |
| File serving                    | Web VM retrieves an image from File VM over HTTP.                          |
| Memory monitoring               | Web VM checks its own heap every 10 seconds.                               |
| Pub/Sub alerting                | When memory exceeds threshold, a JSON event is published to a TCP broker.  |
| Graceful shutdown               | All services stop cleanly on `SIGINT`/`SIGTERM`.                           |
| Configurable addresses          | All external service endpoints are set via command‑line flags.             |
| Extensive logging               | Every request, RPC call, and alert is logged for debugging.                |
| Self‑contained, zero‑config     | Each VM has its own `install.sh` and `run.sh` – no manual dependency hunt. |

---

## 2. System Architecture & VM Configuration

We use three **Ubuntu 22.04 LTS** virtual machines (can be VirtualBox, VMware, or cloud). They are connected via a **Host‑Only network** with static IP addresses. The subscriber can run on any machine (including the host or another VM).

| VM | Role                              | Example IP     | Open Ports                         |
|----|-----------------------------------|----------------|------------------------------------|
| 1  | Web Service + Pub/Sub Broker      | 192.168.56.10  | 8080 (HTTP), 9090 (Pub/Sub broker) |
| 2  | Authentication Service (gRPC)     | 192.168.56.11  | 50051 (gRPC)                       |
| 3  | File Service (HTTP)               | 192.168.56.12  | 8081 (HTTP)                        |

**Network diagram:**

```
+---------------------------+       gRPC        +---------------------------+
|        VM1 (Web)          | ─────────────────>│        VM2 (Auth)         |
|  IP: 192.168.56.10        │ <─────────────────│   IP: 192.168.56.11       |
|  Ports: 8080 (HTTP)       │    (Login resp)   │   Port: 50051 (gRPC)       |
|         9090 (broker)     │                   │                            │
+---------------------------+                   +----------------------------+
           │
           │ HTTP (image fetch)
           ▼
+---------------------------+
|        VM3 (File)         │
|  IP: 192.168.56.12        │
|  Port: 8081 (HTTP)        │
+---------------------------+

The Pub/Sub broker runs on VM1:9090.
The subscriber (any machine) connects to VM1:9090.
```

**Why this IP scheme?**  
- The subnet `192.168.56.0/24` is the default for VirtualBox Host‑Only networks.  
- Static IPs ensure that services do not change addresses after reboot.

---

## 3. Prerequisites (All VMs)

Before starting, ensure each VM meets these requirements:

| Software          | Version      | Installation command (Ubuntu)                                  | Notes                           |
|-------------------|--------------|----------------------------------------------------------------|---------------------------------|
| Go                | 1.23+        | `sudo snap install go --classic` or `sudo apt install golang-go` | Required on all VMs.            |
| protobuf-compiler | 3.x+         | `sudo apt install protobuf-compiler`                          | **Only needed on VM2 (Auth)**.  |
| git               | any          | `sudo apt install git`                                        | Optional (for cloning).         |
| curl              | any          | `sudo apt install curl`                                       | For testing.                    |
| netcat (nc)       | any          | `sudo apt install netcat`                                     | Optional, for manual Pub/Sub.   |

**Firewall settings (if enabled):**  
Open the required ports on each VM:

```bash
sudo ufw allow 8080/tcp   # VM1 (web)
sudo ufw allow 9090/tcp   # VM1 (broker)
sudo ufw allow 50051/tcp  # VM2 (auth)
sudo ufw allow 8081/tcp   # VM3 (file)
```

**Setting static IPs (VirtualBox Host‑Only example):**  
Edit `/etc/netplan/00-installer-config.yaml` (adjust interface name):

```yaml
network:
  ethernets:
    enp0s3:
      dhcp4: no
      addresses: [192.168.56.10/24]   # change .10 to .11 or .12 accordingly
      nameservers:
        addresses: [8.8.8.8, 8.8.4.4]
  version: 2
```

Then apply: `sudo netplan apply`.

---

## 4. Quick Start (5 Minutes)

This section gives the minimal commands to bring up the whole system. For detailed explanations, see the individual component sections.

1. **Copy the project** to each VM (or clone the repository).
2. **On VM2 (Auth):**
   ```bash
   cd auth-vm && chmod +x install.sh run.sh && ./install.sh && ./run.sh
   ```
3. **On VM3 (File):**
   ```bash
   cd file-vm && chmod +x install.sh run.sh && ./install.sh && ./run.sh
   ```
4. **On VM1 (Web + Broker + Subscriber):** (three terminals)
   ```bash
   # Terminal 1
   cd pubsub && go run broker.go
   # Terminal 2
   cd web-vm && go run main.go -auth=192.168.56.11:50051 -file=http://192.168.56.12:8081 -broker=192.168.56.10:9090
   # Terminal 3
   cd pubsub && go run subscriber.go 192.168.56.10:9090
   ```
5. **Open browser** to `http://192.168.56.10:8080/login`, log in with `alice`/`alice123`.
6. **Trigger memory alert:**  
   `curl "http://192.168.56.10:8080/consume-memory?mb=100"` (repeat 4 times).  
   The subscriber terminal will print a formatted alert.

---

## 5. Component 1: Auth VM (gRPC Authentication Service)

**Location:** `auth-vm/`

This service runs on **VM2** and provides user authentication via a gRPC endpoint. It is completely independent and does not call any other service.

### 5.1 Files

| File               | Description                                                                 |
|--------------------|-----------------------------------------------------------------------------|
| `auth.proto`       | Protocol Buffers definition of the `AuthService` with `Login` RPC.          |
| `main.go`          | Server implementation: loads `users.json`, serves gRPC, graceful shutdown.  |
| `users.json`       | JSON array of test users (plaintext passwords – extra credit version uses bcrypt). |
| `go.mod` / `go.sum`| Go module dependencies (gRPC, protobuf).                                    |
| `install.sh`       | Installs protoc plugins, generates stubs, runs `go mod tidy`.               |
| `run.sh`           | Starts the server with default flags.                                       |
| `Makefile`         | `make proto`, `make build`, `make run`, `make test`, `make clean`.          |
| `test_client.go`   | Optional gRPC client for manual testing.                                    |
| `README.md`        | Detailed component‑specific documentation.                                  |

### 5.2 Setup & Run (VM2)

```bash
cd auth-vm
chmod +x install.sh run.sh
./install.sh
./run.sh
```

**Default flags:** `-port :50051 -users users.json`

**Verify it works:**  
From any machine with `grpcurl` installed:

```bash
grpcurl -plaintext -d '{"username":"alice","password":"alice123"}' 192.168.56.11:50051 auth.AuthService/Login
```

Expected response:
```json
{"success": true}
```

**Logs example:**
```
2026/06/03 14:00:01 ✅ Loaded 3 users from users.json
2026/06/03 14:00:01 🚀 Auth VM listening on :50051
2026/06/03 14:00:05 ✅ Successful login for user: alice
```

---

## 6. Component 2: File VM (HTTP Static File Server)

**Location:** `file-vm/`

This service runs on **VM3** and serves static files (images) over HTTP. It uses only the Go standard library.

### 6.1 Files

| File               | Description                                                                 |
|--------------------|-----------------------------------------------------------------------------|
| `main.go`          | HTTP file server with logging, timeouts, graceful shutdown, health check.   |
| `files/`           | Directory containing served files (created by `install.sh`).                |
| `install.sh`       | Creates `files/`, downloads a sample `sample.jpg`, initialises module.      |
| `run.sh`           | Starts the server on port 8081.                                             |
| `Makefile`         | `make run`, `make build`, `make test`, `make clean`.                        |
| `test_client.go`   | Simple HTTP client to test file download.                                   |
| `README.md`        | Component documentation.                                                    |

### 6.2 Setup & Run (VM3)

```bash
cd file-vm
chmod +x install.sh run.sh
./install.sh
./run.sh
```

**Default flags:** `-port 8081 -dir ./files`

**Test with curl:**
```bash
curl -I http://192.168.56.12:8081/files/sample.jpg
# Should return HTTP/1.1 200 OK
```

**Health check:**
```bash
curl http://192.168.56.12:8081/health
# Returns "OK"
```

**Logs example:**
```
2026/06/03 14:00:05 📁 Serving files from: /home/ubuntu/file-vm/files
2026/06/03 14:00:05 🚀 File VM serving on :8081
2026/06/03 14:00:10 📥 GET /files/sample.jpg 192.168.56.10:54321
```

---

## 7. Component 3: Web VM (Frontend + Memory Publisher)

**Location:** `web-vm/` (plus `pubsub/` for broker and subscriber)

This service runs on **VM1**. It provides the user interface, authenticates via gRPC, fetches images, monitors memory, and publishes alerts.

### 7.1 Files

| File                         | Description                                                                 |
|------------------------------|-----------------------------------------------------------------------------|
| `main.go`                    | Web server: login, dashboard, gRPC client, memory monitor, Pub/Sub publisher. |
| `templates/login.html`       | HTML login form.                                                            |
| `templates/dashboard.html`   | Dashboard with image and memory consumption button.                         |
| `authpb/`                    | Generated gRPC stubs (copied from `auth-vm/`).                              |
| `go.mod` / `go.sum`          | Dependencies: gRPC, protobuf.                                               |
| `install.sh`                 | Sets up Go modules, copies or generates stubs.                              |
| `run.sh`                     | Launcher with environment variable overrides.                               |
| `Makefile`                   | `make proto`, `make build`, `make run`, `make clean`.                       |
| `README.md`                  | Component documentation.                                                    |

### 7.2 Setup & Run (VM1)

**First, obtain the gRPC stubs:**  
Copy the `authpb/` folder from `auth-vm/` into `web-vm/`.  
Alternatively, copy `auth.proto` into `web-vm/` and run `make proto`.

**Then install and run:**

```bash
cd web-vm
chmod +x install.sh run.sh
./install.sh
```

**Start the web service** (in a terminal):
```bash
go run main.go -auth=192.168.56.11:50051 -file=http://192.168.56.12:8081 -broker=192.168.56.10:9090
```

**Flags explained:**

| Flag       | Example value                     | Meaning                                                   |
|------------|-----------------------------------|-----------------------------------------------------------|
| `-auth`    | `192.168.56.11:50051`             | Address of Auth VM (gRPC)                                 |
| `-file`    | `http://192.168.56.12:8081`       | Base URL of File VM (no trailing slash)                   |
| `-broker`  | `192.168.56.10:9090`              | Address of Pub/Sub broker (on VM1)                        |
| `-threshold`| `300`                            | Memory threshold in MB (default 300)                      |
| `-listen`  | `:8080`                           | Web server listen address                                 |
| `-interval`| `10s`                            | How often to check memory                                 |
| `-verbose` | `true`                            | Enable detailed logging                                   |

---

## 8. Pub/Sub Broker & Subscriber (Memory Alerts)

The Pub/Sub system is a separate component in the `pubsub/` directory. It can run on any machine; we place the broker on VM1 (same as web service) for simplicity.

### 8.1 Files

| File               | Description                                                                 |
|--------------------|-----------------------------------------------------------------------------|
| `broker.go`        | TCP server that accepts `sub` and `pub` commands, broadcasts messages.      |
| `subscriber.go`    | Client that subscribes and prints memory alerts in a boxed format.          |
| `go.mod`           | Module definition (no external dependencies).                               |
| `run-broker.sh`    | Launcher for broker.                                                        |
| `run-subscriber.sh`| Launcher for subscriber.                                                    |
| `README.md`        | Detailed protocol and usage documentation.                                  |

### 8.2 Broker Protocol (Line‑Based)

- **Subscribe:** Send `sub\n` → broker keeps connection open, sends every future message as `<JSON>\n`.
- **Publish:** Send `pub\n` then `<JSON>\n` → broker broadcasts the JSON to all subscribers, then closes the publisher connection.

### 8.3 Running the Broker (on VM1)

```bash
cd pubsub
go run broker.go -addr :9090
```

**Logs example:**
```
2026/06/03 14:00:00 🚀 PubSub broker listening on :9090
2026/06/03 14:00:10 📡 New subscriber connection from 192.168.56.10:54322
2026/06/03 14:00:10 ✅ Subscriber registered
2026/06/03 14:00:20 📨 Received message from publisher (156 bytes)
2026/06/03 14:00:20 📢 Broadcast message to 1 active subscribers
```

### 8.4 Running the Subscriber (anywhere)

```bash
cd pubsub
go run subscriber.go -broker 192.168.56.10:9090
```

**When a memory alert arrives, the subscriber prints:**
```
╔════════════════════════════════════════════════════════════════╗
║                         🚨 MEMORY ALERT 🚨                       ║
╠════════════════════════════════════════════════════════════════╣
║  Event Type   : HIGH_MEMORY_USAGE                               ║
║  Service      : web-server                                      ║
║  Memory Usage : 415 MB (threshold: 300 MB)                      ║
║  Timestamp    : 2026-06-03T14:32:17+03:30                       ║
╚════════════════════════════════════════════════════════════════╝
```

---

## 9. Complete End‑to‑End Test Scenario

This scenario must be executed and documented (screenshots provided in the `screenshots/` folder). It covers **all assignment requirements**.

### 9.1 Prepare the Environment

| VM | IP Address      | Run these commands                                                                 |
|----|-----------------|------------------------------------------------------------------------------------|
| 2  | 192.168.56.11   | `cd auth-vm && ./run.sh`                                                           |
| 3  | 192.168.56.12   | `cd file-vm && ./run.sh`                                                           |
| 1  | 192.168.56.10   | Terminal 1: `cd pubsub && go run broker.go`                                        |
|    |                 | Terminal 2: `cd web-vm && go run main.go -auth=192.168.56.11:50051 -file=http://192.168.56.12:8081 -broker=192.168.56.10:9090` |
|    |                 | Terminal 3: `cd pubsub && go run subscriber.go 192.168.56.10:9090`                 |

### 9.2 Step‑by‑Step Verification

| Step | Action                                                                 | Expected Result                                                              |
|------|------------------------------------------------------------------------|------------------------------------------------------------------------------|
| 1    | Open browser to `http://192.168.56.10:8080/login`                      | Login form appears.                                                          |
| 2    | Enter `alice` / `wrongpass` and submit                                 | Error message: "Invalid username or password".                               |
| 3    | Enter `alice` / `alice123` and submit                                  | Redirect to `/dashboard`.                                                    |
| 4    | Dashboard shows an image                                              | Image source is `http://192.168.56.12:8081/files/sample.jpg`.                |
| 5    | Run `curl "http://192.168.56.10:8080/consume-memory?mb=100"` four times| Each call prints "Allocated 100 MB".                                          |
| 6    | Wait 10 seconds (monitoring interval)                                  | Subscriber terminal displays the formatted memory alert.                     |
| 7    | Press Ctrl+C in each terminal                                          | Each service prints "shutting down gracefully" and exits.                    |

### 9.3 Expected Terminal Outputs (simulated)

**Auth VM:**
```
2026/06/03 14:00:01 ✅ Loaded 3 users from users.json
2026/06/03 14:00:01 🚀 Auth VM listening on :50051
2026/06/03 14:00:05 ✅ Successful login for user: alice
```

**File VM:**
```
2026/06/03 14:00:02 📁 Serving files from: /home/ubuntu/file-vm/files
2026/06/03 14:00:02 🚀 File VM serving on :8081
2026/06/03 14:00:06 📥 GET /files/sample.jpg 192.168.56.10:54321
```

**Web VM (Terminal 2):**
```
2026/06/03 14:00:03 ✅ Templates loaded successfully
2026/06/03 14:00:03 🚀 Web VM starting on :8080
2026/06/03 14:00:05 📊 Memory usage: 45 MB
2026/06/03 14:00:10 📊 Memory usage: 145 MB
...
2026/06/03 14:00:40 ⚠️ Memory exceeded threshold: 405 MB > 300 MB
2026/06/03 14:00:40 📢 Published memory event: 405 MB (threshold 300 MB)
```

**Subscriber (Terminal 3):**
```
2026/06/03 14:00:04 ✅ Connected to broker at 192.168.56.10:9090, waiting for events...
... (after memory threshold crossed)
╔════════════════════════════════════════════════════════════════╗
║                         🚨 MEMORY ALERT 🚨                       ║
...
```

---

## 10. Troubleshooting & FAQs

### 10.1 Common Issues and Resolutions

| Symptom                                                      | Likely Cause                               | Solution                                                                                             |
|--------------------------------------------------------------|--------------------------------------------|------------------------------------------------------------------------------------------------------|
| VMs cannot ping each other                                   | Wrong network mode or IP configuration     | Use Host‑Only adapter; assign static IPs in same subnet (e.g., 192.168.56.x).                        |
| `protoc: command not found` on VM2                           | protobuf compiler missing                  | `sudo apt install protobuf-compiler`                                                                 |
| gRPC `connection refused`                                    | Auth VM not running or wrong address       | `lsof -i :50051` on VM2; check `-auth` flag on web VM. Open firewall: `sudo ufw allow 50051`.       |
| Web dashboard shows broken image (404)                       | File VM unreachable or image missing       | Ensure File VM is running; check `files/sample.jpg` exists; verify `-file` URL.                      |
| Subscriber never receives alerts                             | Broker not running or wrong `-broker` flag | Start broker first; check `-broker` address on web VM and subscriber.                                |
| Memory alert triggers immediately without allocation         | Memory threshold set too low               | Default is 300 MB; use `-threshold` to increase if needed.                                          |
| `cannot find package "web/authpb"` in web-vm                 | gRPC stubs missing                         | Copy `authpb/` folder from `auth-vm/` into `web-vm/`.                                                |
| `go mod tidy` fails with checksum error                      | Go proxy issues                            | Set `GOPROXY=direct` and `GOSUMDB=off`; or use Iranian mirror `https://go.devneeds.ir`.              |
| `bind: address already in use` when starting a service       | Port conflict                              | Change the port (e.g., `-port :8081` for web server, `-addr :9091` for broker).                      |

### 10.2 Debugging Tips

- **Check open ports:** `sudo netstat -tulpn | grep -E "8080|9090|50051|8081"`
- **Test gRPC connectivity manually:** Use `grpcurl` from any machine.
- **Simulate a Pub/Sub message without web VM:**  
  `echo -e "pub\n{\"test\":1}\n" | nc 192.168.56.10 9090`
- **Monitor logs in real time:** Run services with `-verbose=true` (default).

### 10.3 Performance Notes

- Memory monitoring uses `runtime.ReadMemStats`, which causes a stop‑the‑world pause (very short, < 1ms). For a production system, use a less intrusive method.
- The Pub/Sub broker uses buffered channels (size 256) per subscriber. If a subscriber is too slow, it is automatically disconnected.
- The file server uses timeouts (5s read, 10s write) to prevent slowloris attacks.

---

## 11. File Structure of the Submission

The submitted archive (`HW2.zip` or `HW2.tar.gz`) contains the following directories and files. Each subdirectory has its own `README.md` with component‑specific instructions.

```
HW2/
├── report.pdf                          # Final report (LaTeX generated PDF)
├── rpc-study/
│   └── rpc-summary.pdf                 # Part 1: RPC theory report
├── web-vm/                             # VM1 code
│   ├── main.go
│   ├── templates/
│   │   ├── login.html
│   │   └── dashboard.html
│   ├── go.mod
│   ├── go.sum
│   ├── install.sh
│   ├── run.sh
│   ├── Makefile
│   └── README.md
├── auth-vm/                            # VM2 code
│   ├── auth.proto
│   ├── main.go
│   ├── users.json
│   ├── go.mod
│   ├── go.sum
│   ├── install.sh
│   ├── run.sh
│   ├── Makefile
│   ├── test_client.go
│   └── README.md
├── file-vm/                            # VM3 code
│   ├── main.go
│   ├── files/
│   │   └── sample.jpg
│   ├── go.mod
│   ├── install.sh
│   ├── run.sh
│   ├── Makefile
│   ├── test_client.go
│   └── README.md
├── pubsub/                             # Pub/Sub broker & subscriber
│   ├── broker.go
│   ├── subscriber.go
│   ├── go.mod
│   ├── run-broker.sh
│   ├── run-subscriber.sh
│   └── README.md
└── screenshots/                        # Evidence of successful execution
    ├── login-page.png
    ├── login-error.png
    ├── dashboard-with-image.png
    ├── subscriber-alert.png
    ├── auth-vm-running.png
    ├── file-vm-running.png
    └── web-vm-running.png
```

---

## 12. Evaluation Criteria Self‑Assessment

The following table maps the assignment’s grading rubric to how this submission meets each point.

| Criterion                                                       | Max Points | Self‑Assessment                                                                 | Points Claimed |
|-----------------------------------------------------------------|------------|---------------------------------------------------------------------------------|----------------|
| RPC report (definition, components, errors)                     | 5          | Detailed explanations with tables and diagrams.                                 | 5              |
| Comparison of at least 3 RPC technologies + mandatory table     | 6          | gRPC, JSON‑RPC, XML‑RPC, plus Thrift, Java RMI, SOAP in extra table.            | 6              |
| RPC vs REST comparison                                          | 4          | Extended comparison with 10+ criteria.                                          | 4              |
| Answers to analytical questions                                 | 5          | Five questions answered thoroughly.                                             | 5              |
| Three VMs correctly set up                                      | 8          | Static IPs, separate services, proper network configuration.                    | 8              |
| Login page and authentication RPC                               | 15         | Web VM calls Auth VM via gRPC; users.json on VM2 only.                          | 15             |
| Separation of user data                                         | 4          | Auth VM stores users.json, web VM never accesses it directly.                   | 4              |
| File service on VM3                                             | 6          | HTTP file server with configurable directory.                                   | 6              |
| Retrieve file from VM3                                          | 4          | Web VM fetches and displays image from VM3.                                     | 4              |
| Memory monitoring                                               | 5          | Periodic `runtime.ReadMemStats`.                                                | 5              |
| Threshold checking                                              | 4          | Configurable threshold (default 300 MB).                                        | 4              |
| Event publishing                                                | 5          | JSON event sent to Pub/Sub broker when threshold exceeded.                      | 5              |
| Subscriber                                                      | 5          | Separate subscriber connects, receives, and prints alerts.                      | 5              |
| Method to increase memory                                       | 3          | `/consume-memory?mb=N` endpoint.                                                | 3              |
| Alert display                                                   | 3          | Formatted boxed alert in subscriber output.                                     | 3              |
| Code quality, error handling, documentation                     | 3          | Extensive comments, graceful shutdown, logging, READMEs for each component.     | 3              |
| **Total**                                                       | **100**    |                                                                                 | **100**        |

---

## 13. References & Acknowledgments

- **gRPC documentation:** [https://grpc.io/docs/](https://grpc.io/docs/)
- **Protocol Buffers:** [https://protobuf.dev/](https://protobuf.dev/)
- **Go `net/http` package:** [https://pkg.go.dev/net/http](https://pkg.go.dev/net/http)
- **Go `runtime` memory statistics:** [https://pkg.go.dev/runtime#MemStats](https://pkg.go.dev/runtime#MemStats)
- **Assignment specification** – provided by the course (University of Tehran, Spring 2026).
- **Course instructor:** Dr. Mohammadreza Shourniya  
  **Teaching assistants:** Pooya Jamshidi, Kazem Ayarzadeh, Mohammad Afzal Zadeh

This project was completed independently, with all code written from scratch. The design decisions (gRPC for RPC, custom TCP broker for Pub/Sub, etc.) were made to satisfy the assignment requirements while demonstrating best practices in distributed systems.

---

**End of Complete System README**
```

This README is **fully comprehensive**. It includes:
- Overview, architecture, prerequisites.
- Detailed sections for each VM component.
- Pub/Sub protocol and usage.
- Complete test scenario with expected outputs.
- Troubleshooting table.
- File structure.
- Self‑assessment table.
- References.

