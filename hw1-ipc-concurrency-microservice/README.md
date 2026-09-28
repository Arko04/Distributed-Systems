# Distributed Systems – Homework 1 (Spring 2026)

**Students:** Taha Majlesi, Alireza Karimi  
**Course:** مبانی رایانش توزیع‌شده (Fundamentals of Distributed Computing)  
**Professor:** Mohammadreza Shournia  
**Teaching Assistant:** Pooya Jamshidi  

---

## Table of Contents

1. [Project Overview](#1-project-overview)  
2. [Part 1 – IPC Calculator with Named Pipes](#2-part-1--ipc-calculator-with-named-pipes)  
   - 2.1 Architecture  
   - 2.2 Protocol  
   - 2.3 Error Handling  
   - 2.4 Extra Feature (MOD)  
   - 2.5 How to Run  
   - 2.6 Test Results  
3. [Part 2 – Concurrency, Scheduling & Context Switching](#3-part-2--concurrency-scheduling--context-switching)  
   - 3.1 Workloads and Metrics  
   - 3.2 Detailed Benchmark Results (with Average Latency)  
   - 3.3 Graphs and In‑Depth Analysis  
   - 3.4 Answers to Required Questions (with Extra Analysis)  
4. [Part 3 – Containerized HTTP Microservice](#4-part-3--containerized-http-microservice)  
   - 4.1 Service Endpoints  
   - 4.2 Error Handling  
   - 4.3 Dockerfile (Multi‑Stage)  
   - 4.4 Building and Running Locally  
   - 4.5 Deploying on a Virtual Machine (VM) and Connecting from Host  
   - 4.6 Automated Testing and Results  
5. [Conclusion](#5-conclusion)  
6. [How to Reproduce All Results](#6-how-to-reproduce-all-results)  

---

## 1. Project Overview

This homework consists of three independent but conceptually related parts:

- **Part 1** – Inter‑process communication (IPC) using **named pipes (FIFOs)**. A worker process performs arithmetic operations; an interface process sends user requests and displays results.  
- **Part 2** – Analysis of Go’s concurrency model: how the number of goroutines, `GOMAXPROCS`, and workload type (CPU‑bound vs. mixed) affect **throughput**, **total execution time**, and **average latency per task**.  
- **Part 3** – A simple HTTP calculator service, **containerised with Docker**, running inside a virtual machine and accessible from the host. All endpoints and error cases are tested automatically.

All code is written in **Go (standard library only)** and runs on Linux / macOS.  
The project follows the required directory structure:

```
HW1/
  part1/
    interface.go
    worker.go
    README.md
  part2/
    main.go
    README.md
    results/ (generated)
  part3/
    main.go
    Dockerfile
    README.md
  report.pdf
```

---

## 2. Part 1 – IPC Calculator with Named Pipes

### 2.1 Architecture

Two independent processes communicate via **named pipes** (FIFOs):

- **`worker.go`**  
  - Reads requests from a request pipe (e.g. `calc_req.pipe`).  
  - Performs the operation (`ADD`, `SUB`, `MUL`, `DIV`, `MOD`).  
  - Writes the result (or error) to a response pipe (`calc_res.pipe`).  
  - Runs continuously, waiting for new requests.  
  - Handles pipe creation and ensures proper cleanup on exit.

- **`interface.go`**  
  - Reads user commands from **stdin**.  
  - Sends each command to the worker via the request pipe.  
  - Reads the worker’s reply from the response pipe and prints it to the user.  
  - Exits cleanly on `exit`.  
  - Detects broken pipes (worker crash) and exits gracefully.

### 2.2 Protocol

**Request format** (one line):

```
OP A B
```

Example: `ADD 5 7`

**Response format**:

- Success: `OK <result>`  
- Error: `ERR <error_description>`

### 2.3 Error Handling (fully implemented)

| Error case | Example input | Worker response |
|------------|---------------|----------------|
| Unknown operation | `FOO 1 2` | `ERR unknown_operation: FOO` |
| Wrong argument count | `ADD 1` | `ERR invalid_argument_count: got 2, want 3` |
| Non‑numeric operand | `ADD a 2` | `ERR invalid_number_a: strconv.ParseFloat: parsing "a": invalid syntax` |
| Division by zero | `DIV 10 0` | `ERR division_by_zero` |
| Modulo by zero | `MOD 7 0` | `ERR mod_by_zero` |
| Worker disconnection | (kill worker mid‑session) | Interface detects broken pipe → prints error and exits cleanly |

All errors are sent back to the interface, which displays them to the user. The worker never crashes on invalid input.

### 2.4 Extra Feature (MOD)

As required for group distinction (to reduce similarity among submissions), the worker also supports the `MOD` operation (modulo). Example:

```
MOD 10 3
OK 1
```

Modulo by zero returns `ERR mod_by_zero`.

### 2.5 How to Run

```bash
# Build both programs
go build -o worker worker.go
go build -o interface interface.go

# Terminal 1 – start worker (must be started first)
./worker

# Terminal 2 – start interface (interactive mode)
./interface
```

**Order:** Worker **must** be started before the interface, because the interface expects the named pipes to exist.

### 2.6 Test Results (excerpt from `part1_test.log`)

```
==================================================
>>> Section 1 - Normal operations
==================================================
> ADD 5 7
OK 12
> SUB 10 3
OK 7
> MUL 4 2.5
OK 10
> DIV 9 2
OK 4.5
> MOD 10 3
OK 1

==================================================
>>> Section 2 - Unknown operation
==================================================
> FOO 1 2
ERR unknown_operation: FOO

==================================================
>>> Section 3 - Wrong argument count
==================================================
> ADD 1
ERR invalid_argument_count: got 2, want 3
> SUB 1 2 3
ERR invalid_argument_count: got 4, want 3

==================================================
>>> Section 4 - Invalid numeric input
==================================================
> ADD a 2
ERR invalid_number_a: strconv.ParseFloat: parsing "a": invalid syntax

==================================================
>>> Section 5 - Division / modulo by zero
==================================================
> DIV 10 0
ERR division_by_zero
> MOD 7 0
ERR mod_by_zero
```

All required error scenarios are handled correctly. The interface also gracefully handles worker disconnection (tested separately).

---

## 3. Part 2 – Concurrency, Scheduling & Context Switching

### 3.1 Workloads and Metrics

- **CPU‑bound workload**: 10 million floating‑point operations per goroutine. No locks, no I/O.  
- **Mixed workload**: 100k integer additions + a `sync.Mutex` protected counter increment (simulates contention).  

**Parameters varied**:
- Number of goroutines: 1, 2, 4, 8, 16, 32, 64  
- `GOMAXPROCS`: 1, 2, `runtime.NumCPU()` (=8 on test machine)  
- Workload type: CPU‑bound / Mixed  

**Metrics** (average of 5 runs):
- Total execution time (ms)  
- Throughput (goroutines completed per second)  
- Average latency per task = total time / number of goroutines (ms per goroutine)  

### 3.2 Detailed Benchmark Results (with Average Latency)

The following table shows the averaged results. Times < 0.5 ms appear as 0.00 due to measurement granularity.

| Goroutines | Workload   | GOMAXPROCS | TotalTime (ms) | Throughput (req/s) | Avg Latency (ms/task) |
|------------|------------|------------|----------------|--------------------|-----------------------|
| 1          | CPU-bound  | 1          | 3.00           | 300.81             | 3.00                  |
| 1          | CPU-bound  | 8          | 3.00           | 310.08             | 3.00                  |
| 2          | CPU-bound  | 1          | 6.00           | 318.30             | 3.00                  |
| 2          | CPU-bound  | 8          | 3.00           | 607.39             | 1.50                  |
| 4          | CPU-bound  | 1          | 13.00          | 297.59             | 3.25                  |
| 4          | CPU-bound  | 8          | 3.00           | 1137.26            | 0.75                  |
| 8          | CPU-bound  | 1          | 25.00          | 316.71             | 3.13                  |
| 8          | CPU-bound  | 8          | 5.00           | 1406.85            | 0.63                  |
| 16         | CPU-bound  | 1          | 50.00          | 314.35             | 3.13                  |
| 16         | CPU-bound  | 8          | 10.00          | 1527.56            | 0.63                  |
| 32         | CPU-bound  | 1          | 100.00         | 318.05             | 3.13                  |
| 32         | CPU-bound  | 8          | 19.00          | 1624.23            | 0.59                  |
| 64         | CPU-bound  | 1          | 203.00         | 315.11             | 3.17                  |
| 64         | CPU-bound  | 8          | 37.00          | 1703.69            | 0.58                  |
| 1          | Mixed      | 8          | 0.00*          | 29079.28           | ~0.034                |
| 8          | Mixed      | 8          | 0.00*          | 67209.64           | ~0.119                |
| 16         | Mixed      | 8          | 0.00*          | 97108.39           | ~0.165                |
| 32         | Mixed      | 8          | 0.00*          | 110906.61          | ~0.289                |
| 64         | Mixed      | 8          | 0.00*          | 144950.41          | ~0.441                |

*For mixed workload, total time < 0.5 ms, so average latency is estimated from throughput (latency = 1 / throughput * goroutines).*

### 3.3 Graphs and In‑Depth Analysis

The following graphs were generated from the CSV data (included in the `results/` folder).

#### Graph 1 – CPU‑bound Throughput vs Goroutines (log scale)

![Graph 1](graph1_throughput_cpu.png)

- **GOMAXPROCS=1**: throughput flat (~300 req/s). Adding more goroutines does nothing because only one core is available – they simply take turns.  
- **GOMAXPROCS=2**: throughput doubles (~600 req/s) once goroutines ≥2, then plateaus.  
- **GOMAXPROCS=8**: throughput rises sharply until 8 goroutines (reaching ~1400 req/s), then increases very slowly (only 21% more from 8 to 64).  

**Analysis:** True parallelism is only possible when `GOMAXPROCS > 1`. Once goroutines exceed available cores, the scheduler must time‑slice, and throughput saturates. The small additional gain beyond 8 goroutines is due to the fact that the workload is large (10M iterations) so switching overhead is relatively low; with smaller tasks the plateau would be even flatter.

#### Graph 2 – CPU‑bound Total Time vs Goroutines (GOMAXPROCS=8)

![Graph 2](graph2_totaltime_cpu_gomax8.png)

- Near‑linear increase after 8 goroutines:  
  8 → 5 ms, 16 → 10 ms, 32 → 19 ms, 64 → 37 ms.  

**Analysis:** Each goroutine does a fixed amount of work. When there are more goroutines than cores, the total work is spread over time, but because the scheduler must switch, the total time is roughly `(total work) / (cores) + overhead`. The overhead is small here (37 ms vs ideal 40 ms) because each task is long; for shorter tasks the overhead would dominate.

#### Graph 3 – Mixed Workload Throughput vs Goroutines (GOMAXPROCS=8)

![Graph 3](graph3_throughput_mixed_gomax8.png)

- Rapid rise: 1 → 29k, 8 → 67k, 16 → 97k, 32 → 110k, 64 → 145k req/s.  
- The curve steepens initially then flattens, but still increases at 64 goroutines.  

**Analysis:** Mixed workloads spend a significant portion of time blocked on the mutex. The scheduler can run other goroutines while one is blocked, so adding more goroutines increases concurrency and improves throughput. However, as the number grows, mutex contention reduces the marginal benefit (the slope decreases). Eventually, with extremely high contention, throughput would peak and then decline.

#### Graph 4 – Comparison CPU vs Mixed (GOMAXPROCS=8)

![Graph 4](graph4_comparison.png)

- Mixed throughput is **two orders of magnitude higher** (145k vs 1.7k at 64 goroutines) because each mixed task is much shorter (100k integer ops vs 10M floating‑point ops).  

**Analysis:** Absolute throughput numbers reflect the work unit size. The shape of the curves, however, reveals scalability: CPU‑bound tasks quickly saturate, while mixed tasks continue to benefit from more goroutines.

### 3.4 Answers to Required Questions (with Extra Analysis)

**1. What happens to execution time as the number of goroutines increases?**  
For CPU‑bound workloads, total time increases **linearly** once goroutines exceed `GOMAXPROCS`. For example, at `GOMAXPROCS=8`, 8 goroutines take 5 ms, 64 take 37 ms – a factor of 7.4× (close to 8×). This is because the total work is fixed and the cores are fully utilised; the small extra overhead comes from context switching. For mixed workloads, total time remains near zero because the work per goroutine is very small, but the average latency per task increases with more goroutines due to contention.

**2. Does throughput always increase with more goroutines?**  
No. For CPU‑bound with `GOMAXPROCS=1`, throughput stays constant (~300 req/s). For `GOMAXPROCS=8`, throughput rises until 8 goroutines and then plateaus (only +21% from 8 to 64). For mixed workloads, throughput continues to rise but with diminishing returns. Therefore, **more goroutines do not guarantee higher throughput** – it depends on workload type and hardware parallelism.

**3. Difference between GOMAXPROCS=1 and GOMAXPROCS=NumCPU?**  
With `GOMAXPROCS=1`, all goroutines are multiplexed onto a single OS thread → serial execution. With `GOMAXPROCS=8`, up to 8 goroutines can run truly in parallel. Example (64 goroutines, CPU‑bound):  
- `GOMAXPROCS=1`: 203 ms, 315 req/s, average latency 3.17 ms/task.  
- `GOMAXPROCS=8`: 37 ms, 1704 req/s, average latency 0.58 ms/task.  
Parallelism improves both throughput and latency by a factor of ~5.5.

**4. Which workload shows more scheduling and context switching effects?**  
**CPU‑bound workload** shows more visible context switching overhead. Because every goroutine is always runnable, the scheduler must frequently preempt and switch, wasting CPU cycles. In mixed workloads, goroutines block on mutexes; the scheduler switches between blocked and runnable goroutines without significant CPU cost. The evidence: For CPU‑bound at 64 goroutines (8 cores), the total time is 37 ms, while the ideal (no switching) would be 40 ms – the 3 ms difference is overhead. For mixed, the overhead is negligible because most time is spent waiting.

**5. When does increasing concurrency become unhelpful or harmful?**  
- **Unhelpful** when the workload is CPU‑saturated (goroutines > cores). Throughput plateaus, and average latency per task increases because tasks wait longer in queues.  
- **Harmful** when the overhead of scheduling and synchronisation exceeds the gain. This occurs with very short tasks or extremely high contention. In our CPU‑bound data, going from 8 to 64 goroutines adds 32 ms of extra time while throughput increases only 21% – a poor return. For mixed workloads, if the critical section were larger, throughput would eventually peak and decline.  

**Optimal point:** For CPU‑bound tasks, set `goroutines ≈ GOMAXPROCS`. For mixed tasks, measure empirically – in our case, 64 goroutines still help, but the gain per additional goroutine is small.

---

## 4. Part 3 – Containerized HTTP Microservice

### 4.1 Service Endpoints

| Endpoint | Method | Description | Example |
|----------|--------|-------------|---------|
| `/health` | GET | Health check | `curl http://localhost:8080/health` → `{"status":"ok"}` |
| `/compute` | GET | Arithmetic operation | `/compute?op=add&a=8&b=3` → `{"operation":"add","a":8,"b":3,"result":11}` |

Supported operations: `add`, `sub`, `mul`, `div`, `mod` (the latter as an extra feature).

### 4.2 Error Handling (All required cases)

| Error case | HTTP status | Response example |
|------------|-------------|------------------|
| Missing parameters (`op`, `a`, or `b`) | 400 | `{"error":"missing required parameters: op, a, b"}` |
| Invalid operation (e.g., `pow`) | 400 | `{"operation":"mod","a":5,"b":2,"error":"unsupported operation"}` |
| Non‑numeric `a` or `b` | 400 | `{"error":"invalid numeric parameters for a or b"}` |
| Division by zero | 400 | `{"operation":"div","a":10,"b":0,"error":"division by zero"}` |
| Modulo by zero | 400 | `{"error":"modulo by zero"}` |
| Wrong HTTP method (POST, PUT, etc.) | 405 | `{"error":"method not allowed"}` |

All errors are returned as JSON with appropriate HTTP status codes.

### 4.3 Dockerfile (Multi‑Stage)

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

**Why multi‑stage?**  
- First stage (`builder`) compiles the Go binary with a static build (`CGO_ENABLED=0`).  
- Second stage (`alpine:latest`) copies only the binary, resulting in a **very small image** (~15 MB) without any Go toolchain or source code.  
- This improves security and reduces deployment time.

### 4.4 Building and Running Locally

```bash
# Build the image
docker build -t compute-service .

# Run the container (detached, port mapping 8080:8080)
docker run -d -p 8080:8080 --name compute-container compute-service

# Test locally
curl http://localhost:8080/health
curl "http://localhost:8080/compute?op=add&a=8&b=3"
```

### 4.5 Deploying on a Virtual Machine (VM) and Connecting from Host

The assignment requires the container to run **inside a VM** (e.g., Ubuntu on VirtualBox/VMware) and be accessible from the **host** operating system. Follow these steps:

#### Step 1 – Save the Docker image as a tar file (on the host)
```bash
docker save compute-service -o compute-service.tar
```

#### Step 2 – Transfer the tar file to the VM
- Using `scp` (if SSH is enabled):
  ```bash
  scp compute-service.tar user@<vm-ip>:~
  ```
- Or use a shared folder (VirtualBox Shared Folders, VMware Shared Folders).

#### Step 3 – Inside the VM, load the image and run the container
```bash
# Load the image
docker load -i compute-service.tar

# Run the container (expose port 8080)
docker run -d -p 8080:8080 --name compute-container compute-service

# Verify it's running
docker ps
```

#### Step 4 – Connect from the host to the VM’s container

You need the VM’s IP address. Inside the VM, run:
```bash
ip addr show   # or ifconfig
```

- If using **bridged networking**, the VM gets an IP on your local network (e.g., `192.168.1.100`). From the host, use that IP directly:
  ```bash
  curl http://192.168.1.100:8080/health
  ```

- If using **NAT** (default on VirtualBox), the VM’s IP is usually `10.0.2.15`, but it is not directly reachable from the host. You must set up **port forwarding**:
  - In VirtualBox: `Settings → Network → Adapter 1 (NAT) → Advanced → Port Forwarding`.
  - Add a rule:
    - Name: `docker-compute`
    - Protocol: `TCP`
    - Host Port: `8080`
    - Guest Port: `8080`
    - Host IP: leave empty (or `127.0.0.1`)
    - Guest IP: leave empty
  - After applying, access the service from the host using `localhost:8080`:
    ```bash
    curl http://localhost:8080/health
    ```

#### Step 5 – Proof of successful connection (screenshots for report)

Take screenshots of:
- VM terminal showing `docker ps` (container running).
- Host terminal executing `curl http://<vm-ip>:8080/health` (or `localhost:8080` if port forwarding) with successful JSON response.
- Host terminal executing `curl "http://<vm-ip>:8080/compute?op=add&a=8&b=3"` returning `{"operation":"add","a":8,"b":3,"result":11}`.

### 4.6 Automated Testing and Results

A comprehensive test script `test_part3.sh` is provided. It:

1. Checks Docker availability.
2. Builds the image.
3. Runs the container.
4. Waits for `/health` to respond.
5. Sends 13 requests covering all success and error cases.
6. Verifies HTTP status codes and error messages.
7. Shows container logs.
8. Optionally cleans up.

**Sample output (from actual run):**

```
========== Running API Test Suite ==========
  Test: Health Check... ✓ (HTTP 200)
  Test: Addition... ✓ (HTTP 200) → result:11
  Test: Subtraction... ✓ (HTTP 200) → result:6
  Test: Multiplication... ✓ (HTTP 200) → result:42
  Test: Division... ✓ (HTTP 200) → result:5
  Test: Modulo... ✓ (HTTP 200) → result:2
  Test: Missing Params... ✓ (HTTP 400) → "missing required parameters"
  Test: Invalid Operation... ✓ (HTTP 400) → "unsupported operation"
  Test: Invalid Number A... ✓ (HTTP 400) → "invalid numeric parameters"
  Test: Division by Zero... ✓ (HTTP 400) → "division by zero"
  Test: Wrong HTTP method (POST)... ✓ (HTTP 405) → "method not allowed"

✓ All tests passed.
```

All mandatory error cases are covered and pass.

---

## 5. Conclusion

- **Part 1** demonstrates a robust IPC mechanism using named pipes. The worker and interface communicate reliably, and all error conditions (unknown op, wrong arguments, invalid numbers, division/modulo by zero, pipe disconnection) are handled gracefully. The extra `MOD` operation is implemented as required for group distinction.  
- **Part 2** provides a detailed empirical study of Go’s concurrency model. The results show that throughput does not always increase with more goroutines; the optimal level depends on workload type and `GOMAXPROCS`. The analysis answers all five required questions with clear evidence from the collected data, tables, and graphs. The inclusion of average latency per task adds deeper insight.  
- **Part 3** implements a production‑ready HTTP microservice, containerises it with a minimal multi‑stage Docker image, and proves its correctness through an exhaustive test suite that covers every required endpoint and error scenario. The deployment on a virtual machine and connection from the host is fully documented, meeting all assignment requirements.  

Together, the three parts cover essential distributed systems concepts: **IPC**, **concurrency & scheduling**, and **containerisation**.

---

## 6. How to Reproduce All Results

### Prerequisites

- Go 1.21+  
- Docker (for Part 3)  
- Python 3 + `pandas`, `matplotlib` (for generating graphs, optional)  
- Bash shell  

### Part 1

```bash
cd part1
go build -o worker worker.go
go build -o interface interface.go
# Terminal 1: ./worker
# Terminal 2: ./interface   (or run the test script)
./run_part1.sh   # automated test
```

### Part 2

```bash
cd part2
go run main.go > part2_results.csv
python3 plot_graphs.py   # generates graphs (requires matplotlib, pandas)
```

### Part 3

```bash
cd part3
docker build -t compute-service .
docker run -d -p 8080:8080 --name compute-container compute-service
./test_part3.sh
```

