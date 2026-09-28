# Distributed Systems Assignment 2 – Multi‑VM RPC System

**Students:** Taha Majlesi, Alireza Karimi  
**Course:** Fundamentals of Distributed Computing – University of Tehran  
**Instructor:** Mohammadreza Shourniya  
**Spring 2026**

---

## 🎯 Overview

This project implements a **fully distributed system** that spans **three independent virtual machines** (or containers). All components communicate exclusively over the network using their **real IP addresses** – no `localhost` shortcuts are used, as required by the assignment.

The system demonstrates four key distributed computing concepts:

1. **Remote Procedure Call (RPC)** – using gRPC between the Web VM and the Auth VM.  
2. **Service separation** – authentication, file serving, and web logic each run on their own VM.  
3. **Publish/Subscribe pattern** – a custom TCP broker monitors memory usage and raises alerts.  
4. **Memory monitoring** – the Web VM periodically checks its own heap usage and publishes an event when it exceeds 300 MB.

The technologies used: **Go 1.25**, **gRPC/Protocol Buffers**, **HTTP**, and a custom TCP‑based Pub/Sub broker. The system is fully reproducible and has been tested on **UTM with Apple Virtualization (aarch64)**.

---

## 🧠 System Architecture (Explained)

The architecture consists of three dedicated virtual machines (plus an optional fourth for the broker, though we co‑locate it on the Web VM for simplicity). The following diagram shows the communication flow:

```
Browser
   │
   ▼
┌─────────────────┐      gRPC (Login)      ┌─────────────────┐
│  VM1 (Web)      │ ─────────────────────► │  VM2 (Auth)     │
│  192.168.1.116  │ ◄───────────────────── │  192.168.1.118  │
│  - HTTP :8080   │      LoginResponse     │  - gRPC :50051  │
│  - gRPC client  │                        │  - users.json   │
│  - Pub/Sub pub  │                        └─────────────────┘
└─────────────────┘
         │  HTTP (image)
         ▼
┌─────────────────┐      Pub/Sub (TCP)     ┌─────────────────┐
│  VM3 (File)     │                        │  VM1 (Broker)   │
│  192.168.1.117  │                        │  - TCP :9090    │
│  - HTTP :8081   │                        └─────────────────┘
│  - static files │                                 │
└─────────────────┘                                 │ sub
                                                    ▼
                                            ┌─────────────────┐
                                            │  VM1 (Subscriber)│
                                            │  prints alerts  │
                                            └─────────────────┘
```

**Why this architecture?**  
- **Separation of concerns** – each VM has a single responsibility, making the system easier to debug and scale.  
- **RPC requirement** – the Web VM calls the Auth VM via gRPC, fulfilling the core RPC requirement.  
- **File service** – a simple HTTP file server on VM3 demonstrates that the Web VM can retrieve resources from another machine.  
- **Pub/Sub** – memory monitoring and alerting are decoupled; the Web VM only publishes events, and a separate subscriber handles the alert display.  

All IP addresses shown above are examples from our test environment. You must replace them with the actual IPs obtained from `ip a` on your VMs.

---

## 📁 Project Structure

```
HW2/
├── README.md                     (this file – complete guide)
├── report.pdf                    (final report with screenshots and analysis)
├── rpc-study/
│   └── rpc-summary.pdf           (Part 1 – theoretical RPC study)
├── screenshots/                  (all required screenshots for the report)
│
├── auth-vm/                      # Authentication service (gRPC server)
│   ├── main.go
│   ├── auth.proto
│   ├── users.json
│   ├── go.mod
│   └── (generated: auth.pb.go, auth_grpc.pb.go)
│
├── file-vm/                      # File service (HTTP server)
│   ├── main.go
│   ├── files/
│   │   └── sample.jpg
│   └── go.mod
│
├── web-vm/                       # Web service (HTTP + gRPC client + memory monitor)
│   ├── main.go
│   ├── templates/
│   │   ├── login.html
│   │   └── dashboard.html
│   ├── authpb/                   (copied from auth-vm generated files)
│   └── go.mod
│
└── pubsub/                       # Pub/Sub broker and subscriber
    ├── broker.go
    ├── subscriber.go
    └── go.mod
```

Each service folder contains its own `go.mod` and all necessary code. The `authpb/` directory in `web-vm` is a copy of the generated gRPC stubs from `auth-vm` – this avoids having to regenerate them on the Web VM.

---

## ✅ Prerequisites (Before You Start)

- **3 Linux VMs** (or containers with separate IP addresses).  
  *The assignment explicitly forbids running all services on a single machine or using `localhost`.*  
- **Go 1.25** (or at least 1.23). Earlier versions (e.g., 1.19) do not support required packages like `slices` and `maps`.  
  *Install Go manually if your distribution provides an older version.*  
- **Network connectivity** between the VMs:  
  - Web VM must reach Auth VM on port `50051`.  
  - Web VM must reach File VM on port `8081`.  
  - Web VM must reach the PubSub broker on port `9090`.  
- (Optional) `protoc` – only needed if you regenerate the gRPC stubs. Pre‑generated files are included.  
- **Firewall** – ensure ports are open (on each VM: `sudo ufw allow 50051`, etc. if using `ufw`).

---

## 🔧 Detailed Setup & Configuration

We assume you have already set up three VMs with IP addresses. The instructions are **per VM**.

> **Important:** All commands must be executed **inside the respective VM** (unless specified otherwise). Replace `<IP...>` placeholders with the actual IPs from your environment.

---

### Step 1: Environment Variables (Go Module Proxy)

Because some networks block the default Go proxy (`proxy.golang.org`), we set a working mirror (e.g., the Iranian mirror `go.devneeds.ir`). Run this on **each VM** before any `go` commands:

```bash
go env -w GOPROXY=https://go.devneeds.ir,direct
go env -w GOSUMDB=off    # optional, only if you encounter checksum errors
```

If you prefer a different proxy (e.g., `https://goproxy.cn,direct`), use that instead.

---

### Step 2: Auth VM (VM2)

**Purpose:** Runs the gRPC authentication service.  
**IP:** e.g., `192.168.1.118` (example – use your actual IP).

1. Copy the `auth-vm/` folder to VM2.  
2. Open a terminal inside `auth-vm/`.  
3. **Download dependencies** and run the service:

   ```bash
   cd auth-vm
   go mod tidy
   go run -mod=mod main.go
   ```

   Expected output:
   ```
   2026/06/03 10:00:01 Loaded 3 users from users.json
   2026/06/03 10:00:01 Auth VM listening on :50051
   ```

4. **Do not close this terminal** – the Auth VM must stay running.

> **Note about users.json:** The file contains three test users (alice, bob, admin). Passwords are stored in plaintext for simplicity, as allowed by the assignment. You can add or modify users by editing the JSON file and restarting the service.

> **Regenerating gRPC stubs (if needed):**  
> If you modify `auth.proto`, regenerate the Go code with:
> ```bash
> protoc --go_out=. --go-grpc_out=. auth.proto
> ```
> Then copy the entire `authpb/` folder to `web-vm/authpb/` so that the web service can use the updated stubs.

---

### Step 3: File VM (VM3)

**Purpose:** Serves static files (images) over HTTP.  
**IP:** e.g., `192.168.1.117`.

1. Copy the `file-vm/` folder to VM3.  
2. Ensure there is at least one image inside `files/` (e.g., `sample.jpg`). You can add your own image or use the provided one.  
3. Run the file server:

   ```bash
   cd file-vm
   go run main.go
   ```

   Expected output:
   ```
   2026/06/03 10:00:05 File VM serving on :8081
   Serving files from /root/Desktop/file-vm/files
   ```

4. Keep this terminal running.

> **How it works:** The service uses `http.FileServer` to serve the `files/` directory. Any file placed there becomes available at `http://<VM3_IP>:8081/files/<filename>`.

---

### Step 4: Pub/Sub Broker (usually on VM1 or a dedicated VM)

**Purpose:** Accepts `pub` and `sub` commands over TCP and broadcasts messages to all subscribers.  
**IP:** e.g., `192.168.1.116` (same as Web VM, or a separate VM).

1. Copy the `pubsub/` folder to the machine that will host the broker.  
2. Run the broker:

   ```bash
   cd pubsub
   go run broker.go
   ```

   Expected output:
   ```
   2026/06/03 10:00:10 PubSub broker listening on :9090
   ```

3. Keep this terminal open.

**Broker protocol:**  
- `sub` – register the connection as a subscriber (connection stays open).  
- `pub <message>` – broadcast the message (JSON) to all subscribers.

---

### Step 5: Subscriber (can run on the same machine as the broker)

**Purpose:** Connects to the broker, listens for events, and prints memory alerts.

1. On the same or another VM, open a new terminal.  
2. Navigate to the `pubsub/` folder.  
3. Run the subscriber, providing the broker’s IP and port:

   ```bash
   cd pubsub
   go run subscriber.go <broker_IP>:9090
   ```

   For example: `go run subscriber.go 192.168.1.116:9090`

   Expected output:
   ```
   2026/06/03 10:00:15 Connected to broker at 192.168.1.116:9090, waiting for events...
   ```

4. The subscriber will remain silent until it receives a `HIGH_MEMORY_USAGE` event.

---

### Step 6: Web VM (VM1)

**Purpose:** Provides the login page, calls the Auth VM via gRPC, fetches an image from the File VM, monitors memory, and publishes events.

**IP:** e.g., `192.168.1.116`.

1. Copy the `web-vm/` folder to VM1.  
2. **Ensure the `authpb/` folder exists** – if not, copy it from `auth-vm`:

   ```bash
   cp -r ../auth-vm/authpb .   # assuming both folders are in the same parent directory
   ```

3. **Edit the IP addresses** in `web-vm/main.go`. Look for the constants at the top of the file:

   ```go
   const (
       authAddr    = "192.168.1.118:50051"   // change to your Auth VM IP
       fileAddr    = "http://192.168.1.117:8081" // change to your File VM IP
       brokerAddr  = "192.168.1.116:9090"    // change to your broker IP
       thresholdMB = 300
   )
   ```

   Replace the example IPs with the actual IPs of your VMs.

4. **Download dependencies** and run the web service:

   ```bash
   cd web-vm
   go mod tidy
   go run -mod=mod main.go
   ```

   Expected output:
   ```
   2026/06/03 10:00:20 Web VM starting on :8080
   2026/06/03 10:00:20 Templates loaded successfully
   2026/06/03 10:00:20 [DEBUG] Memory monitor started, checking every 10s
   ```

5. Keep this terminal running.

The Web VM is now ready to accept HTTP requests on port `8080`.

---

## 🧪 Complete Test Scenario (Mandatory for TAs)

Follow these steps exactly as described. All required screenshots must be taken and included in `report.pdf`.

### A. Verify IP addresses
On each VM, run `ip a` and take a screenshot showing the IP address (e.g., `inet 192.168.1.x/24`). These must match the IPs you configured in the Web VM.

### B. Start all services in order
1. Auth VM (VM2) – terminal window 1  
2. File VM (VM3) – terminal window 2  
3. PubSub broker – terminal window 3 (on any VM, but often VM1)  
4. Subscriber – terminal window 4 (same or different VM)  
5. Web VM – terminal window 5 (VM1)

Keep all terminals visible for screenshots.

### C. Test authentication
1. Open a browser on any machine that can reach VM1’s port 8080 (you can use a browser inside VM1 itself, or your host if networking allows).  
2. Visit `http://<VM1_IP>:8080/login`.  
   **Screenshot:** The login page (URL visible).  
3. Enter wrong credentials, e.g., `alice` / `wrongpass`.  
   **Screenshot:** Error message “Invalid username or password”.  
4. Enter correct credentials: `alice` / `alice123`.  
   **Screenshot:** Dashboard appears, showing the image retrieved from the File VM.

### D. Trigger memory alert
1. From any terminal that can reach VM1’s port 8080 (e.g., inside VM1 itself), run:
   ```bash
   curl "http://192.168.1.116:8080/consume-memory?mb=100"
   ```
   Repeat this command **4 times** (total 400 MB).  
   *Why 4 times?* Each call allocates 100 MB and keeps it alive. After the fourth call, memory exceeds 300 MB.  
2. Wait up to 10 seconds (the monitoring interval).  
3. Observe the **subscriber terminal** – it should print an alert similar to:
   ```
   *** ALERT ***
   Service: web-server
   Memory: 415 MB (threshold 300 MB)
   Timestamp: 2026-06-03T10:05:23+03:30
   ```
4. **Screenshot:** The subscriber terminal showing the alert.

### E. Verify continuous monitoring
If you keep the Web VM running, the subscriber will print a new alert every 10 seconds as long as memory stays above 300 MB. You can also allocate more memory (e.g., `?mb=200` twice) to increase the value.

---

## 🛠 Troubleshooting Common Issues

| Problem | Likely Cause | Solution |
|---------|--------------|----------|
| `go mod tidy` fails with `zip: not a valid zip file` | Corrupted proxy download. | Set `GOPROXY=direct` or use a different mirror (e.g., `https://goproxy.cn,direct`). |
| `connection refused` when Web VM tries to call Auth VM | Firewall or service not running. | Check that Auth VM is running (`ps aux | grep main`). Test connectivity with `telnet <IP> 50051`. Open ports with `sudo ufw allow 50051`. |
| Subscriber shows `bad event: unexpected end of JSON input` | Empty lines or extra newlines in TCP stream. | This is harmless; the subscriber ignores non‑JSON lines. It does not affect alert detection. |
| Web VM cannot find `authpb` package | The `authpb/` folder is missing. | Copy it from `auth-vm`: `cp -r ../auth-vm/authpb .` inside `web-vm/`. |
| Memory monitor never triggers even after `curl` calls | Garbage collector freed the memory. | Make sure the `memoryHog` global slice is used (it is). Each `consume-memory` appends to it, preventing GC. |
| `go: inconsistent vendoring` | A `vendor/` folder exists but `go.mod` has changed. | Delete the `vendor/` folder and use `-mod=mod`. |
| `protoc` command not found | The Protocol Buffers compiler is not installed. | Pre‑generated stubs are included; you don’t need `protoc` unless you change `auth.proto`. |
| Browser cannot reach Web VM on port 8080 | Network isolation (e.g., Emulated VLAN in UTM). | Use a browser **inside VM1** (install a minimal GUI like Xfce) or add a second network interface in `Shared` mode for host access. |

---

## 📦 Dependency Management

All Go modules are managed via `go.mod` files. The recommended `GOPROXY` is `https://go.devneeds.ir,direct` (Iranian mirror) because it is reliable behind local restrictions. To update dependencies to their latest compatible versions:

```bash
go get -u google.golang.org/grpc
go get -u google.golang.org/protobuf
go mod tidy
```

If you ever encounter checksum mismatches, you can temporarily disable checksum verification with:

```bash
go env -w GOSUMDB=off
```

But only do this if you trust the proxy.

---

## 📸 Required Screenshots for the Report

The `screenshots/` folder must contain at least the following images (exact names can vary, but they should be clearly labelled in the report):

1. `ip-addresses.png` – output of `ip a` on each VM (three separate screenshots or one combined).  
2. `auth-running.png` – Auth VM terminal showing “Auth VM listening on :50051”.  
3. `file-running.png` – File VM terminal showing “File VM serving on :8081”.  
4. `broker-running.png` – Broker terminal showing “PubSub broker listening on :9090”.  
5. `subscriber-waiting.png` – Subscriber terminal showing “waiting for events”.  
6. `web-running.png` – Web VM terminal showing “Web VM starting on :8080”.  
7. `login-page.png` – Browser showing the login form (URL visible).  
8. `login-error.png` – Browser showing the error after wrong credentials.  
9. `dashboard.png` – Browser showing the dashboard with the image from File VM.  
10. `alert.png` – Subscriber terminal showing the high‑memory alert.  

Optionally, also include a combined screenshot of all terminals to prove they are running on separate VMs (different IPs).

---

## 📁 Deliverables (What to Submit)

- **Complete source code** of all services (the entire folder structure as shown above).  
- **This README.md** file (the final version).  
- **Final report** (`report.pdf`) containing:  
  - Architecture diagram (can be a drawing or text).  
  - Comparison tables (RPC technologies, RPC vs REST).  
  - Answers to the five analytical questions.  
  - Step‑by‑step test scenario with all required screenshots.  
  - Self‑assessment of evaluation criteria.  
- **Screenshots folder** with all images referenced in the report.  
- **Optional:** Separate README files inside each component folder (not mandatory but nice).

The submission must be packaged as a ZIP archive named `HW2_<studentIDs>.zip` (or similar according to course rules).

---

## 👤 Authors

- **Taha Majlesi**  
- **Alireza Karimi**  

University of Tehran – Spring 2026  
For any questions, contact the course teaching assistants or the instructor, Dr. Mohammadreza Shourniya.

---

## 📚 References

- gRPC Documentation: [https://grpc.io/docs/](https://grpc.io/docs/)  
- Protocol Buffers: [https://protobuf.dev/](https://protobuf.dev/)  
- Go `runtime` package: [https://pkg.go.dev/runtime](https://pkg.go.dev/runtime)  
- UTM virtualisation for Apple Silicon: [https://mac.getutm.app/](https://mac.getutm.app/)  
- Assignment specification (provided by the course).

---

**© 2026 – Distributed Systems Course, University of Tehran**
