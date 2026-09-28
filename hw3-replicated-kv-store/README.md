# Distributed Replicated Key-Value Store

## Complete Project Documentation and Conceptual Guide

---

## 📚 Table of Contents

1. [Project Overview](#project-overview)
2. [Theoretical Concepts](#theoretical-concepts)
   - [Why Replication?](#why-replication)
   - [Replication vs Performance vs Fault Tolerance](#replication-for-performance-vs-fault-tolerance)
   - [The Consistency Problem](#the-consistency-problem)
   - [Strong Consistency vs Eventual Consistency](#strong-vs-eventual-consistency)
   - [Client-Centric Consistency Models](#client-centric-consistency-models)
   - [The CAP Theorem](#the-cap-theorem)
3. [System Architecture](#system-architecture)
4. [Project Structure](#project-structure)
5. [Installation & Prerequisites](#installation--prerequisites)
6. [How to Run](#how-to-run)
7. [Client Commands Reference](#client-commands-reference)
8. [Testing Scenarios](#testing-scenarios)
9. [API Reference](#api-reference)
10. [Data Model & Versioning](#data-model--versioning)
11. [Conflict Resolution Strategy](#conflict-resolution-strategy)
12. [Consistency Models Implementation](#consistency-models-implementation)
13. [Metrics Collection](#metrics-collection)
14. [Troubleshooting](#troubleshooting)
15. [Expected Learning Outcomes](#expected-learning-outcomes)

---

## Project Overview

This project implements a **Distributed Replicated Key-Value Store** as part of the Distributed Computing Fundamentals course at the University of Tehran. The system demonstrates fundamental distributed systems concepts including:

- **Data Replication** across multiple independent nodes
- **Consistency Models** (Eventual and Strong)
- **Versioning** of stored data
- **Conflict Detection and Resolution**
- **Fault Tolerance** and availability trade-offs
- **Network Delay** impact on system behavior

The system consists of **three independent replica servers** that communicate over HTTP and a **command-line client** that interacts with these replicas. Each replica maintains its own local copy of the data and synchronizes with other replicas through replication messages.

---

## Theoretical Concepts

### Why Replication?

**Replication** is the process of maintaining multiple copies of the same data across different nodes (machines, servers, or processes) in a distributed system. Data is replicated for several critical reasons:

#### 1. **Increased Availability**
When data is replicated across multiple nodes, the system can continue to serve requests even if some nodes fail. If one replica crashes or becomes unreachable due to network issues, clients can still access the data from another replica. This is crucial for building fault-tolerant systems.

**Example:** In a banking system with three replicas, if one server goes down for maintenance, customers can still check their balances and make transactions through the remaining two replicas.

#### 2. **Improved Performance and Scalability**
Replication allows the system to distribute read requests across multiple replicas, reducing the load on any single node. Additionally, replicas can be placed geographically closer to users, reducing network latency.

**Example:** A global social media platform replicates user profiles across data centers in North America, Europe, and Asia. Users in Tokyo can read from the Tokyo replica instead of waiting for data to travel from New York.

#### 3. **Fault Tolerance and Data Durability**
With multiple copies of data, the system can tolerate failures without losing data. As long as at least one replica survives, the data remains accessible and can be used to restore other replicas.

**Example:** If a hard drive fails on one server, the data is not lost because it exists on other replicas. The failed server can be replaced and its data restored from the surviving replicas.

#### 4. **Disaster Recovery**
Geographically distributed replicas protect against regional disasters (power outages, natural disasters) that could destroy a single data center.

#### 5. **Offline Operations**
In some systems (like mobile applications), local replicas allow users to continue working even when disconnected from the network. Changes are synchronized when connectivity is restored.

---

### Replication for Performance vs Fault Tolerance

While both purposes involve creating copies of data, the design goals and trade-offs differ significantly:

| Aspect | Performance Replication | Fault Tolerance Replication |
|--------|------------------------|----------------------------|
| **Primary Goal** | Reduce latency, increase throughput | Ensure data durability and availability |
| **Placement** | Close to users (geographic distribution) | Distributed for redundancy |
| **Consistency** | Often relaxed (eventual consistency acceptable) | May require stronger guarantees |
| **Write Handling** | May use asynchronous propagation | Often requires synchronous writes to multiple replicas |
| **Read Handling** | Read from nearest replica | Read from any available replica |
| **Failure Impact** | Performance degrades if replica fails | System continues operating if replica fails |
| **Example** | CDN (Content Delivery Network) | Database replication for disaster recovery |
| **Trade-off** | Optimizes for speed, accepts some staleness | Optimizes for safety, may sacrifice some performance |

**Detailed Explanation:**

**Performance-Focused Replication:**
- **Caching Systems:** CDNs replicate static content (images, videos, web pages) to edge servers worldwide. The goal is to serve content from the closest server to the user, minimizing latency. If an edge server fails, users are redirected to the next nearest server.
- **Read-Heavy Workloads:** In applications where reads vastly outnumber writes, multiple read replicas can handle read queries while a single master handles writes. This scales the system without complex synchronization.

**Fault Tolerance-Focused Replication:**
- **State Machine Replication:** Each replica maintains an identical copy of the application state by applying the same sequence of operations in the same order. This ensures that even if some replicas fail, the system continues to function correctly.
- **Quorum-Based Systems:** Writes must be acknowledged by a majority of replicas before being considered committed. This ensures that even if some replicas fail, the data is preserved and consistent.

**The key insight:** Performance replication often uses **asynchronous** updates (fire-and-forget), while fault tolerance replication typically requires **synchronous** confirmation from multiple replicas. This trade-off directly impacts consistency guarantees.

---

### The Consistency Problem

The fundamental challenge of replication is the **consistency problem**: keeping all replicas in sync when updates occur.

#### The Root Cause
When a client updates data on one replica, there is an inevitable **time gap** before other replicas receive and apply the update. During this gap:

- Different replicas may hold **different values** for the same key
- A client reading from a replica that hasn't received the update will see **stale (outdated) data**
- Concurrent updates on different replicas can create **conflicts** that must be resolved

#### Types of Inconsistency

1. **Stale Reads:** Reading an old value that has been updated on another replica but the update hasn't propagated yet.

   ```
   Time: ----|----|----|----|----
   Client 1: PUT x=5 to Replica A  (t=0)
   Client 2: GET x from Replica B  (t=1) -> Returns old value or null
   Replication: A -> B happens at   (t=3)
   Client 2: GET x from Replica B  (t=4) -> Returns x=5
   ```

2. **Write Conflicts:** Two clients update the same key on different replicas simultaneously, creating two divergent versions.

   ```
   Time: ----|----|----|----|----
   Client 1: PUT x=10 on Replica A
   Client 2: PUT x=20 on Replica B  (nearly simultaneously)
   Replica A now has x=10, Replica B has x=20
   When they exchange data, which value is correct?
   ```

3. **Lost Updates:** An update is overwritten by another update that didn't see the first one.

   ```
   Time: ----|----|----|----|----
   Client reads x=5 from Replica A
   Client increments to x=6 and writes to Replica A
   Meanwhile, another client reads x=5 from Replica B
   Other client increments to x=6 and writes to Replica B
   The final value should be x=7, but both replicas show x=6
   ```

#### Why Consistency is Hard
- **Network delays** are unpredictable and can vary from milliseconds to seconds
- **Network partitions** can isolate replicas from each other
- **Concurrent operations** happen naturally in distributed systems
- **No global clock** exists to determine the exact order of events across different machines
- **Replica failures** can occur at any time, potentially during updates

---

### Strong vs Eventual Consistency

These are two fundamental approaches to handling the consistency problem, representing different points on the consistency spectrum.

#### Strong Consistency

**Definition:** After an update completes, all subsequent reads (from any replica) will return the updated value. The system behaves as if there is only one copy of the data.

**Also known as:**
- Linearizability
- Atomic consistency
- Immediate consistency

**How it works in our implementation:**
1. Client sends a `PUT` request to one replica
2. The receiving replica forwards the update to all other replicas **synchronously**
3. The write is only considered successful when a **majority of replicas** (2 out of 3) confirm they have applied the update
4. Only then is success returned to the client
5. Any subsequent `GET` from any replica that was part of the majority will return the new value

**Advantages:**
- Simple programming model – the system behaves predictably
- No stale reads – always see the latest data
- Easier reasoning about system state
- Critical for applications like banking, inventory management, user authentication

**Disadvantages:**
- Higher latency for writes (must wait for multiple confirmations)
- Reduced availability during network partitions (may refuse writes if majority unavailable)
- Cannot scale writes linearly (every replica must process every write)
- Performance bottleneck – the slowest replica determines write speed

**Code Example from our implementation:**
```go
func (r *Replica) replicateStrong(entry DataEntry) bool {
    // ... 
    // Need majority (including self)
    totalNodes := len(r.config.Peers) + 1
    majorityNeeded := totalNodes/2 + 1  // For 3 nodes: 3/2 + 1 = 2
    acknowledged := 1 // Self is already acknowledged

    // Send to all peers and wait for responses
    for _, peer := range r.config.Peers {
        // ... send replication request and wait ...
    }
    
    // Only succeed if majority acknowledged
    success := acknowledged >= majorityNeeded
    return success
}
```

#### Eventual Consistency

**Definition:** If no new updates are made to a data item, eventually all replicas will converge to the same value. However, immediately after an update, reads from different replicas may return different (stale) values.

**How it works in our implementation:**
1. Client sends a `PUT` request to one replica
2. The receiving replica immediately applies the update and returns success to the client
3. The update is propagated to other replicas **asynchronously** (in the background)
4. During the propagation delay, other replicas still have the old value
5. Over time, all replicas receive and apply the update, converging to the same state

**Advantages:**
- Low write latency (write to one replica and return immediately)
- High availability (can always accept writes, even during partitions)
- Better write throughput (no waiting for other replicas)
- Works well for applications where staleness is acceptable (social media feeds, DNS, search indexes)

**Disadvantages:**
- Complex programming model – must handle stale data
- Temporary inconsistencies are visible to users
- May require conflict resolution mechanisms
- Harder to debug and reason about system state

**Code Example from our implementation:**
```go
func (r *Replica) replicateEventual(entry DataEntry) {
    // ... prepare replication request ...
    
    // Send asynchronously - don't wait for responses
    for _, peer := range r.config.Peers {
        go func(peerURL string) {
            r.sendReplication(peerURL+"/replicate", jsonData)
        }(peer)
    }
    // Returns immediately - client already got success
}
```

#### Visual Comparison

```
Strong Consistency:
Client PUT --> [Replica A] --sync--> [Replica B] --sync--> [Replica C]
               Wait for 2/3 ACK <--OK--              <--OK--
Return Success (all replicas now have the same value)

Eventual Consistency:
Client PUT --> [Replica A] 
Return Success (immediately)
               |--async--> [Replica B] (receives update after delay)
               |--async--> [Replica C] (receives update after delay)
```

| Property | Strong Consistency | Eventual Consistency |
|----------|-------------------|---------------------|
| Write Latency | Higher (wait for majority) | Lower (write to one) |
| Read Consistency | Always latest | May be stale temporarily |
| Availability During Partition | May refuse writes | Always accepts writes |
| Conflict Likelihood | Low | Higher |
| Use Cases | Banking, inventory | Social media, caching |
| Implementation Complexity | Higher | Lower |

---

### Client-Centric Consistency Models

These are consistency guarantees provided to individual clients, even when the system as a whole only provides eventual consistency. They ensure that a single client's view of the data is "consistent enough" for their needs.

#### Read-Your-Writes Consistency

**Definition:** Once a client writes a value, any subsequent reads by that same client will return the written value (or a newer one), never an older version.

**Why it matters:** Imagine posting a comment on a social media post. After clicking "Post," you expect to see your comment immediately when the page refreshes. Without read-your-writes, you might see the page without your comment, making you think the post failed.

**Implementation approaches:**
- Track which replica received the write and always read from it
- Use version numbers to ensure reads return at least the written version
- Use timestamps to reject older values

**In our system:** The version number mechanism supports this. If a client writes version 5 of a key, it can request "version 5 or higher" in its read. Our current implementation doesn't enforce this automatically, but the versioning infrastructure supports it.

#### Monotonic Reads

**Definition:** If a client reads a value, subsequent reads will never return an older value. The client's view of the data only moves forward in time.

**Why it matters:** Consider refreshing a news feed. You see 10 new posts. If you refresh again (now connected to a different, stale replica), you might see only 5 posts – the 10 you saw earlier seem to have "disappeared." This is confusing and violates user expectations.

**Implementation approaches:**
- Track the highest version seen by each client
- Always read from replicas that are at least as current as what the client has seen
- Use session affinity (always route same client to same replica)

**In our system:** The `/getall` command demonstrates this by showing version differences across replicas. A client could track the highest version it has seen and only accept values with that version or higher.

#### Bounded Staleness

**Definition:** Reads may return stale data, but the staleness is bounded by a time limit (e.g., "data is at most 5 seconds old").

**Why it matters:** Some applications can tolerate slightly stale data but not arbitrarily old data. A weather app might be fine showing data up to 5 minutes old, but a stock trading app might require data no older than 1 second.

**Implementation:** This requires tracking timestamps and either rejecting reads that are too old or ensuring replication happens within the bound.

---

### The CAP Theorem

The **CAP theorem** (also known as Brewer's Theorem) states that a distributed data store can provide only two of the following three guarantees simultaneously:

#### C - Consistency
Every read receives the most recent write or an error. All nodes see the same data at the same time.

#### A - Availability
Every request receives a (non-error) response, without guarantee that it contains the most recent write. The system continues to operate even when some nodes are down.

#### P - Partition Tolerance
The system continues to operate despite network partitions (messages between nodes are lost or delayed).

#### Why Can't We Have All Three?

The fundamental tension becomes clear during a **network partition**:

```
[Replica A] ----X---- [Replica B]
   |                     |
[Client 1]          [Client 2]
```

**Scenario:** The network between Replica A and Replica B is broken. Client 1 writes x=5 to Replica A. Client 2 reads x from Replica B.

**The Choice:**

1. **Choose Consistency over Availability (CP System):**
   - Replica B refuses to respond to Client 2's read because it cannot confirm it has the latest data
   - The system remains consistent but is not available
   - **Example:** Traditional relational databases with synchronous replication, HBase, MongoDB (configured for strong consistency)

2. **Choose Availability over Consistency (AP System):**
   - Replica B responds with whatever value it has (which is stale)
   - The system remains available but is inconsistent
   - **Example:** Cassandra, DynamoDB, CouchDB, DNS

3. **What about CA?**
   - CA systems (Consistency + Availability, no Partition Tolerance) are only possible when there is no network partition
   - In practice, partitions are inevitable in distributed systems
   - True CA systems are essentially single-node databases

#### CAP in Our System

**Strong Consistency Model = CP:**
- During a partition, if majority is not available, writes are refused
- Prioritizes consistency over availability

**Eventual Consistency Model = AP:**
- During a partition, writes are accepted on any available replica
- Prioritizes availability over consistency (accepts temporary inconsistency)

**Real-World Guidance:**
- The CAP theorem is a framework for understanding trade-offs
- In practice, systems can be tuned along a spectrum (not binary choices)
- Modern systems often allow per-operation configuration (e.g., some reads/writes can be strongly consistent, others eventually consistent)

---

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         CLIENT                              │
│  (Command-line interface that can connect to any replica)   │
└──────┬──────────────┬──────────────┬───────────────────────┘
       │              │              │
       │ HTTP         │ HTTP         │ HTTP
       ▼              ▼              ▼
┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│  REPLICA 1  │ │  REPLICA 2  │ │  REPLICA 3  │
│  Port 8001  │ │  Port 8002  │ │  Port 8003  │
│             │◄├────────────►├◄├─────────────┤
│  In-Memory  │ │  In-Memory  │ │  In-Memory  │
│  Data Store │ │  Data Store │ │  Data Store │
└─────────────┘ └─────────────┘ └─────────────┘
       ▲              ▲              ▲
       └──────────────┴──────────────┘
            Replication Messages
         (HTTP POST to /replicate)
```

### Component Details

#### 1. Replica Servers

Each replica is an **independent HTTP server** running in its own process. They are not goroutines or threads within a single program – they are separate OS processes that communicate over TCP/HTTP.

**Responsibilities:**
- Accept `PUT` and `GET` requests from clients
- Maintain a local in-memory key-value store
- Propagate updates to other replicas
- Handle replication requests from peers
- Enforce consistency models
- Track data versions

**Endpoints:**
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/put` | POST | Store a key-value pair |
| `/get?key=<key>` | GET | Retrieve a value by key |
| `/replicate` | POST | Receive replication data from peer |
| `/health` | GET | Health check and status |
| `/stop` | POST | Simulate failure by stopping requests |
| `/start` | POST | Resume after being stopped |
| `/data` | GET | Debug endpoint to view all stored data |

#### 2. Client

A command-line tool that can connect to any replica and perform operations. It supports both interactive mode and command-line arguments for scripting tests.

**Features:**
- Connect to any replica
- Perform `PUT`, `GET`, and `getall` operations
- Measure response times
- Start/stop replicas for testing
- Check health of all replicas

#### 3. Configuration Files

JSON files that define each replica's identity, network address, peer list, and behavior:

```json
{
  "id": "replica1",
  "host": "localhost",
  "port": 8001,
  "peers": [
    "http://localhost:8002",
    "http://localhost:8003"
  ],
  "consistency_model": "eventual",
  "network_delay": 0
}
```

---

## Project Structure

```
hw3/
├── README.md                           # This documentation file
├── configs/                            # Configuration files for each replica
│   ├── replica1.json                   # Replica 1 configuration (port 8001)
│   ├── replica2.json                   # Replica 2 configuration (port 8002)
│   └── replica3.json                   # Replica 3 configuration (port 8003)
├── replica/                            # Replica server source code
│   ├── go.mod                          # Go module definition
│   ├── go.sum                          # Dependency checksums
│   └── main.go                         # Replica server implementation
├── client/                             # Client source code
│   ├── go.mod                          # Go module definition
│   ├── go.sum                          # Dependency checksums
│   └── main.go                         # Client implementation
└── results/                            # Test scenario results
    ├── scenario1.txt                   # Temporary inconsistency test results
    ├── scenario2.txt                   # Replica failure test results
    ├── scenario3.txt                   # Concurrent conflict test results
    └── scenario4.txt                   # Network delay test results
```

### File Descriptions

#### `configs/replica1.json`, `replica2.json`, `replica3.json`
Configuration files specifying each replica's identity, network address, peers, consistency model, and artificial network delay. Modifying these files changes replica behavior without code changes.

#### `replica/main.go`
The complete replica server implementation including:
- **Data structures:** `Config`, `DataEntry`, request/response types
- **HTTP handlers:** `handlePUT`, `handleGET`, `handleReplicate`, `handleHealth`, `handleStop`, `handleStart`, `handleData`
- **Replication logic:** `replicateEventual` (async) and `replicateStrong` (synchronous with majority)
- **Concurrency control:** `sync.RWMutex` for thread-safe data access
- **Conflict resolution:** Version comparison and timestamp-based LWW

#### `client/main.go`
The client implementation providing:
- **HTTP client** for communicating with replicas
- **Interactive mode** with command parsing
- **Command mode** for scriptable testing
- **Response time measurement**
- **Health checking** and replica management

#### `results/scenario*.txt`
Templates for recording test results with:
- Expected behavior descriptions
- Step-by-step instructions
- Placeholders for actual results
- Analysis sections

---

## Installation & Prerequisites

### System Requirements
- **Operating System:** Linux, macOS, or Windows
- **Go:** Version 1.21 or higher
- **Available Ports:** 8001, 8002, 8003 (must be free)
- **Terminal:** Multiple terminal windows/tabs for running replicas

### Installing Go

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install golang-go
```

**macOS (with Homebrew):**
```bash
brew install go
```

**Windows:**
Download the installer from [https://golang.org/dl/](https://golang.org/dl/)

**Verify Installation:**
```bash
go version
# Should output: go version go1.21.x or higher
```

### Project Setup

```bash
# Clone or create the project directory
mkdir -p hw3
cd hw3

# Create directory structure
mkdir -p configs replica client results

# Copy the source files into their respective directories
# (replica/main.go, client/main.go, config files, etc.)

# Initialize Go modules (if not already done)
cd replica
go mod init hw3/replica
go mod tidy

cd ../client
go mod init hw3/client
go mod tidy

cd ..
```

### Port Availability Check

Before starting, ensure ports 8001-8003 are free:

```bash
# Linux/macOS
lsof -i :8001
lsof -i :8002
lsof -i :8003

# If ports are in use, kill the processes or change ports in config files
```

---

## How to Run

### Step 1: Start the Replica Servers

Open **three separate terminal windows** (or tabs). In each terminal, navigate to the project directory and start one replica:

**Terminal 1 - Replica 1:**
```bash
cd replica
go run main.go ../configs/replica1.json
```

Expected output:
```
Replica replica1 starting on localhost:8001 (model: eventual, delay: 0ms)
```

**Terminal 2 - Replica 2:**
```bash
cd replica
go run main.go ../configs/replica2.json
```

Expected output:
```
Replica replica2 starting on localhost:8002 (model: eventual, delay: 0ms)
```

**Terminal 3 - Replica 3:**
```bash
cd replica
go run main.go ../configs/replica3.json
```

Expected output:
```
Replica replica3 starting on localhost:8003 (model: eventual, delay: 0ms)
```

### Step 2: Start the Client

Open a **fourth terminal window** for the client:

**Interactive Mode:**
```bash
cd client
go run main.go
```

You'll see:
```
=== Distributed Key-Value Store Client ===
Connected to replicas: [http://localhost:8001 http://localhost:8002 http://localhost:8003]
Commands:
  put <replica> <key> <value>  - Store a value
  get <replica> <key>          - Retrieve a value
  getall <key>                 - Get from all replicas
  stop <replica>               - Stop a replica
  start <replica>              - Start a replica
  health                       - Check all replicas
  quit                         - Exit

>
```

### Step 3: Build Standalone Binaries (Optional)

For easier use, you can build standalone executables:

```bash
# Build the replica server
cd replica
go build -o replica main.go

# Build the client
cd client
go build -o client main.go

# Now you can run them directly:
./replica/replica configs/replica1.json
./client/client
```

---

## Client Commands Reference

### Interactive Mode Commands

Once the client is running in interactive mode, you can use these commands:

#### PUT - Store a Value
```
> put <replica-index> <key> <value>
```

**Parameters:**
- `replica-index`: The replica to send the request to (1, 2, or 3)
- `key`: The key name (any string without spaces)
- `value`: The value to store (any string without spaces)

**Example:**
```
> put 1 name Ali
Response (took 5.2ms): success=true, version=1, Value stored successfully
```

**What happens:**
1. Client sends `POST /put` with `{"key":"name","value":"Ali"}` to Replica 1
2. Replica 1 stores the value with version 1
3. Replica 1 returns success immediately (eventual) or waits for majority (strong)
4. Replica 1 propagates the update to Replicas 2 and 3

#### GET - Retrieve a Value
```
> get <replica-index> <key>
```

**Example:**
```
> get 1 name
Response (took 1.8ms): value='Ali', version=1, updated_by=replica1
```

**What happens:**
1. Client sends `GET /get?key=name` to the specified replica
2. The replica returns the value, version, and metadata

#### GETALL - Check All Replicas
```
> getall <key>
```

**Example:**
```
> getall name
Results for key 'name':
  http://localhost:8001: value='Ali', version=1, updated_by=replica1
  http://localhost:8002: value='Ali', version=1, updated_by=replica1
  http://localhost:8003: value='Ali', version=1, updated_by=replica1
```

**Use this to:** Observe replication status and detect inconsistencies.

#### HEALTH - Check Replica Status
```
> health
```

**Example:**
```
Replica http://localhost:8001: map[id:replica1 model:eventual status:healthy]
Replica http://localhost:8002: map[id:replica2 model:eventual status:healthy]
Replica http://localhost:8003: map[id:replica3 model:eventual status:healthy]
```

#### STOP - Simulate Replica Failure
```
> stop <replica-index>
```

**Example:**
```
> stop 3
Replica 3 stopped
```

After stopping:
```
> get 3 name
Error: ... (replica is unavailable)
```

#### START - Restart a Stopped Replica
```
> start <replica-index>
```

**Example:**
```
> start 3
Replica 3 started
```

### Command-Line Mode (for scripting)

The client can also be used with command-line arguments, useful for automated testing:

```bash
# PUT operation
go run main.go put <replica-index> <key> <value>
# Example:
go run main.go put 1 mykey myvalue

# GET operation
go run main.go get <replica-index> <key>
# Example:
go run main.go get 1 mykey

# GETALL operation
go run main.go getall <key>
# Example:
go run main.go getall mykey

# HEALTH check
go run main.go health

# STOP replica
go run main.go stop <replica-index>

# START replica
go run main.go start <replica-index>
```

---

## Testing Scenarios

### Scenario 1: Temporary Inconsistency Observation

**Objective:** Demonstrate that in eventual consistency, replicas can have different values temporarily.

**Prerequisites:** All replicas running with `consistency_model: "eventual"` and some network delay (e.g., 100ms).

**Steps:**
1. Ensure all three replicas are running and healthy
2. Write a value to Replica 1:
   ```
   > put 1 username Alice
   ```
3. Immediately (as fast as possible) read from Replica 2:
   ```
   > get 2 username
   ```
4. Wait 2-3 seconds for replication to complete
5. Read from Replica 2 again:
   ```
   > get 2 username
   ```

**Expected Results:**
- Step 3: Replica 2 returns "NOT FOUND" or an old value (if the key existed before)
- Step 5: Replica 2 returns `username=Alice`

**Why this happens:**
The `PUT` operation on Replica 1 returns immediately. The replication message to Replica 2 is sent asynchronously and takes time to arrive and be processed. During this gap, Replica 2 hasn't received the update yet.

**Analysis points:**
- Measure the time between step 2 and step 5 (the convergence time)
- This demonstrates the fundamental trade-off: low write latency vs. read consistency
- Consider: Is this acceptable for a social media application? For a banking application?

---

### Scenario 2: Replica Failure

**Objective:** Understand how the system behaves when a replica crashes, and compare eventual vs. strong consistency.

**Part A: Eventual Consistency**

**Steps:**
1. Stop Replica 3:
   ```
   > stop 3
   ```
2. Write a value to Replica 1:
   ```
   > put 1 email user@example.com
   ```
3. Read from Replica 2:
   ```
   > get 2 email
   ```
4. Start Replica 3:
   ```
   > start 3
   ```
5. After a few seconds, check Replica 3:
   ```
   > get 3 email
   ```

**Expected Results:**
- Step 2: PUT succeeds immediately
- Step 3: Replica 2 returns `email=user@example.com` (it received the replication)
- Step 5: Replica 3 returns NOT FOUND (it was down during the update and missed it)

**Part B: Strong Consistency**

**Setup:** Change `consistency_model` to `"strong"` in all config files and restart replicas.

**Steps:**
1. Stop Replica 3:
   ```
   > stop 3
   ```
2. Write a value to Replica 1:
   ```
   > put 1 email user@example.com
   ```

**Expected Results:**
- The PUT should still succeed because 2 out of 3 replicas (majority) are available
- If you stop two replicas (leaving only one), the PUT should fail with "Failed to achieve majority consensus"

**Analysis:**
- Eventual consistency prioritizes availability (accepts writes even when replicas are down)
- Strong consistency requires a quorum, balancing consistency with partial availability
- What happens to the failed replica when it comes back online? (In our implementation, it doesn't automatically catch up)

---

### Scenario 3: Concurrent Conflict

**Objective:** Observe what happens when two clients update the same key simultaneously on different replicas.

**Steps:**
1. Open two client terminals
2. In Terminal 1, write to Replica 1:
   ```
   > put 1 counter 100
   ```
3. In Terminal 2, **immediately** (before replication completes) write to Replica 2:
   ```
   > put 2 counter 200
   ```
4. Wait for replication to complete (2-3 seconds)
5. Check all replicas:
   ```
   > getall counter
   ```

**Expected Results:**
- All replicas should converge to the same value
- The winning value depends on the conflict resolution strategy (LWW: the write with the higher timestamp wins)
- Check the replica logs for conflict detection messages

**Our Conflict Resolution (Last-Write-Wins):**
```
Replica 1 creates counter=100 with timestamp T1
Replica 2 creates counter=200 with timestamp T2
If T2 > T1: Both replicas converge to counter=200
If T1 > T2: Both replicas converge to counter=100
```

**Analysis:**
- This demonstrates why some applications need stronger conflict resolution (e.g., CRDTs, operational transforms)
- LWW can silently lose updates – the "loser" value disappears
- Consider: Would this be acceptable for a shopping cart? For a bank balance?

---

### Scenario 4: Network Delay Impact

**Objective:** Measure how network delay affects convergence time and stale reads.

**Setup:** Test with three different delay configurations by modifying `network_delay` in config files.

**Test A: No Delay (0ms)**
1. Configure all replicas with `"network_delay": 0`
2. PUT a value: `> put 1 data test1`
3. Immediately GET from Replica 2: `> get 2 data`
4. Record the convergence time

**Test B: Medium Delay (500ms)**
1. Configure all replicas with `"network_delay": 500`
2. PUT a value: `> put 1 data test2`
3. Immediately GET from Replica 2: `> get 2 data`
4. Record the convergence time

**Test C: High Delay (2000ms)**
1. Configure all replicas with `"network_delay": 2000`
2. PUT a value: `> put 1 data test3`
3. Immediately GET from Replica 2: `> get 2 data`
4. Record the convergence time

**Metrics to Record:**

| Delay | PUT Latency | Convergence Time | Stale Reads Observed |
|-------|-------------|-------------------|---------------------|
| 0ms | ___ ms | ___ ms | Yes/No |
| 500ms | ___ ms | ___ ms | Yes/No |
| 2000ms | ___ ms | ___ ms | Yes/No |

**Strong Consistency Comparison:**
Repeat with strong consistency. The PUT latency should increase with delay because the system waits for majority acknowledgment.

**Analysis:**
- How does delay affect the probability of stale reads?
- At what delay does the system become unusable?
- What are real-world causes of such delays (geographic distance, network congestion)?

---

## API Reference

### Replica Server Endpoints

#### `POST /put`
Store a key-value pair.

**Request:**
```json
{
  "key": "username",
  "value": "alice"
}
```

**Response (Success):**
```json
{
  "success": true,
  "message": "Value stored successfully",
  "version": 1,
  "updated_by": "replica1"
}
```

**Response (Failure - Strong Consistency, Majority Not Available):**
```json
{
  "success": false,
  "message": "Failed to achieve majority consensus"
}
```

**Status Codes:**
- `200 OK` - Value stored successfully
- `400 Bad Request` - Invalid request body or missing key
- `503 Service Unavailable` - Replica is stopped

---

#### `GET /get?key=<key>`
Retrieve a value by key.

**Request:**
```
GET /get?key=username
```

**Response (Found):**
```json
{
  "key": "username",
  "value": "alice",
  "version": 1,
  "updated_by": "replica1",
  "timestamp": 1702345678000000000,
  "found": true
}
```

**Response (Not Found):**
```json
{
  "key": "nonexistent",
  "found": false
}
```

**Status Codes:**
- `200 OK` - Query processed (check `found` field)
- `400 Bad Request` - Missing key parameter
- `503 Service Unavailable` - Replica is stopped

---

#### `POST /replicate`
Internal endpoint for receiving replication data from peer replicas.

**Request:**
```json
{
  "entry": {
    "key": "username",
    "value": "alice",
    "version": 1,
    "updated_by": "replica1",
    "timestamp": 1702345678000000000
  },
  "origin_id": "replica1",
  "timestamp": 1702345678000000000
}
```

**Response:**
```json
{
  "success": true,
  "message": "Replication processed"
}
```

**Processing Logic:**
1. Apply artificial network delay if configured
2. Compare version with existing data:
   - Newer version → Accept update
   - Same version → Use LWW (compare timestamps)
   - Older version → Reject (don't overwrite newer data)
3. Log conflict if detected

**Status Codes:**
- `200 OK` - Replication processed
- `400 Bad Request` - Invalid request body
- `503 Service Unavailable` - Replica is stopped

---

#### `GET /health`
Check the health and status of a replica.

**Response:**
```json
{
  "id": "replica1",
  "status": "healthy",
  "model": "eventual"
}
```

**Status Codes:**
- `200 OK` - Always returns status (even if stopped, status field will say "stopped")

---

#### `POST /stop`
Simulate a replica failure by stopping request processing.

**Request:** Empty body

**Response:**
```json
{
  "status": "stopped",
  "message": "Replica replica1 has been stopped"
}
```

**Effect:** The replica stops processing PUT, GET, and REPLICATE requests (returns 503). Health endpoint still works.

**Status Codes:**
- `200 OK` - Replica stopped successfully

---

#### `POST /start`
Restart a previously stopped replica.

**Request:** Empty body

**Response:**
```json
{
  "status": "started",
  "message": "Replica replica1 has been started"
}
```

**Effect:** The replica resumes normal operation.

**Status Codes:**
- `200 OK` - Replica started successfully

---

#### `GET /data`
Debug endpoint to view all stored data in the replica.

**Response:**
```json
{
  "username": {
    "key": "username",
    "value": "alice",
    "version": 1,
    "updated_by": "replica1",
    "timestamp": 1702345678000000000
  },
  "email": {
    "key": "email",
    "value": "alice@example.com",
    "version": 2,
    "updated_by": "replica2",
    "timestamp": 1702345679000000000
  }
}
```

**Status Codes:**
- `200 OK` - Returns all stored data

---

## Data Model & Versioning

### Data Structure

Each key-value pair is stored as a `DataEntry` object:

```go
type DataEntry struct {
    Key       string `json:"key"`        // The key name
    Value     string `json:"value"`      // The stored value
    Version   int    `json:"version"`    // Monotonically increasing version number
    UpdatedBy string `json:"updated_by"` // ID of the replica that last updated this entry
    Timestamp int64  `json:"timestamp"`  // Nanosecond timestamp of the update
}
```

### JSON Representation
```json
{
  "key": "username",
  "value": "alice",
  "version": 3,
  "updated_by": "replica1",
  "timestamp": 1702345678000000000
}
```

### Versioning Rules

1. **Initial Creation:** When a key is first created, its version is set to 1.

2. **Subsequent Updates:** Each time a key's value is changed on any replica, the version is incremented by 1.

3. **Version Comparison During Replication:**
   - If received version > current version → **Accept the update**
   - If received version < current version → **Reject the update** (the received data is stale)
   - If received version == current version → **Conflict detected**, use conflict resolution

4. **Version Monotonicity:** Version numbers only increase. A replica never decrements a version number.

### Why Versioning Matters

**Problem Without Versioning:**
```
Timeline:
t1: Replica A gets x=10 (no version tracking)
t2: Replica B gets x=20 (no version tracking)
t3: A replicates x=10 to B → B now has x=10 (the OLDER update overwrote the newer one!)
t4: B replicates x=20 to A → A now has x=20
t5: System oscillates between values, never stabilizes
```

**Solution With Versioning:**
```
Timeline:
t1: Replica A stores x=10, version=1
t2: Replica B stores x=20, version=1 (concurrent - conflict!)
t3: A sends {x=10, v=1} to B → B sees v=1 == v=1, conflict resolution chooses x=20 (higher timestamp)
t4: B sends {x=20, v=1} to A → A sees v=1 == v=1, conflict resolution chooses x=20
t5: System converges to x=20 on all replicas
```

### Timestamp Field

The `timestamp` field uses `time.Now().UnixNano()` for high-precision ordering:

- **Purpose:** Break ties when versions are equal (concurrent updates)
- **Resolution:** Nanoseconds (highly unlikely to have exact same timestamp)
- **Usage in LWW:** When two updates have the same version, the one with the higher timestamp wins

---

## Conflict Resolution Strategy

### What is a Conflict?

A conflict occurs when two replicas receive updates for the same key **concurrently** (before they can synchronize with each other). Both updates have the same version number but different values.

```
     Replica A          Replica B
        |                   |
   PUT x=100           PUT x=200
   version=5           version=5
        |                   |
        |--- replication --->|
        |<-- replication ----|
        |                   |
   CONFLICT: Both have version 5, but different values!
```

### Our Strategy: Last-Write-Wins (LWW)

**Rule:** When two updates have the same version, the one with the **higher timestamp** wins.

**Implementation in code:**
```go
if repReq.Entry.Version > currentEntry.Version {
    // Newer version - always accept
    shouldUpdate = true
} else if repReq.Entry.Version == currentEntry.Version {
    // Same version - conflict!
    if repReq.Entry.Timestamp > currentEntry.Timestamp {
        // Received data has higher timestamp - accept it
        shouldUpdate = true
        log.Printf("CONFLICT resolved for key '%s' using LWW (new version)", key)
    } else {
        // Current data has higher timestamp - reject
        log.Printf("CONFLICT detected for key '%s', keeping current version", key)
    }
} else {
    // Older version - reject
}
```

### Why LWW?

**Advantages:**
1. **Simple to implement:** Just compare timestamps
2. **Deterministic:** Same result on all replicas (assuming timestamps are unique)
3. **Automatic resolution:** No human intervention needed
4. **Convergent:** All replicas will eventually agree on the same value

**Disadvantages:**
1. **Silent data loss:** The "losing" value disappears without notification to clients
2. **Clock skew vulnerability:** Relies on synchronized clocks across machines
3. **Not suitable for all data types:** Works poorly for counters, lists, or sets where merging would be better

### Alternative Conflict Resolution Strategies

| Strategy | Description | Use Case |
|----------|-------------|----------|
| **Last-Write-Wins (LWW)** | Higher timestamp wins | Simple key-value stores |
| **Replica Priority** | Higher replica ID wins | When timestamp reliability is low |
| **Merge/CRDT** | Automatically merge values | Counters, sets, collaborative editing |
| **Application-Level** | Return both values to client | When business logic must decide |
| **Multi-Version** | Keep both versions | When history matters |

### Conflict Resolution in Our Implementation

**Scenario Walkthrough:**
```
1. Replica A receives PUT x=100 at timestamp 1000 → stores {v:5, ts:1000, val:100}
2. Replica B receives PUT x=200 at timestamp 1001 → stores {v:5, ts:1001, val:200}
3. A sends replication to B: {v:5, ts:1000, val:100}
   B compares: v=5 == v=5, ts=1000 < ts=1001 → REJECT (keeps val:200)
4. B sends replication to A: {v:5, ts:1001, val:200}
   A compares: v=5 == v=5, ts=1001 > ts=1000 → ACCEPT (updates to val:200)
5. Both replicas now have x=200 ✓
```

---

## Consistency Models Implementation

### Eventual Consistency (Default)

**Configuration:**
```json
{
  "consistency_model": "eventual"
}
```

**Write Flow:**
```
1. Client sends PUT to Replica A
2. Replica A:
   a. Acquires write lock (sync.RWMutex)
   b. Updates local data store with new version
   c. Releases lock
   d. Returns success to client IMMEDIATELY
   e. Spawns goroutines to replicate to peers ASYNCHRONOUSLY
3. Client receives response (very fast, ~1-5ms local operation)
4. Replication happens in background:
   - Each peer receives POST /replicate
   - Peers apply update independently
   - If a peer is down, replication silently fails
```

**Characteristics:**
- **Write Latency:** Low (only local write)
- **Consistency Window:** Non-zero (stale reads possible until replication completes)
- **Availability:** High (accepts writes even if peers are down)
- **Durability:** Lower (data could be lost if the receiving replica crashes before replication)

**When to use:**
- Social media feeds (temporary staleness acceptable)
- DNS records (eventual propagation is expected)
- Cache invalidation
- Shopping cart (non-critical)
- Real-time analytics (approximate counts acceptable)

---

### Strong Consistency (Simplified)

**Configuration:**
```json
{
  "consistency_model": "strong"
}
```

**Write Flow:**
```
1. Client sends PUT to Replica A
2. Replica A:
   a. Acquires write lock
   b. Updates local data store
   c. Releases lock
   d. Calculates majority needed: (total_nodes / 2) + 1
      - With 3 replicas: 3/2 + 1 = 2 (need self + 1 peer)
   e. Sends replication to ALL peers SYNCHRONOUSLY
   f. Waits for responses
   g. Counts successful acknowledgments
   h. If acknowledgments >= majority:
      - Returns success to client
   i. If acknowledgments < majority:
      - Returns failure to client
      - (Note: local update is already applied - could rollback if needed)
3. Client receives response (slower, depends on slowest peer in majority)
```

**Majority Calculation:**
```
Total Nodes (N) = Number of Peers + Self
Majority = floor(N/2) + 1

With 3 replicas (self + 2 peers):
  N = 3
  Majority = floor(3/2) + 1 = 1 + 1 = 2

With 5 replicas:
  N = 5
  Majority = floor(5/2) + 1 = 2 + 1 = 3
```

**Availability Matrix:**

| Running Replicas | Majority Needed | Can Accept Writes? |
|------------------|-----------------|---------------------|
| 3 of 3 | 2 | ✅ Yes |
| 2 of 3 | 2 | ✅ Yes |
| 1 of 3 | 2 | ❌ No (cannot reach majority) |
| 3 of 5 | 3 | ✅ Yes |
| 2 of 5 | 3 | ❌ No |

**Characteristics:**
- **Write Latency:** Higher (must wait for peer acknowledgments)
- **Consistency Window:** Zero (once write succeeds, all replicas in majority have the value)
- **Availability:** Lower (writes fail if majority not available)
- **Durability:** Higher (data exists on multiple nodes before success is returned)

**When to use:**
- Banking transactions
- Inventory management
- User authentication state
- Leader election
- Critical configuration updates

---

### Switching Between Models

**Method 1: Configuration File (All Replicas)**
```json
// configs/replica1.json, replica2.json, replica3.json
{
  "consistency_model": "strong"  // Change from "eventual" to "strong"
}
```
Then restart all replicas.

**Method 2: Mixed Models (Advanced)**
You can run different replicas with different models, but this creates complex behavior and is not recommended for this exercise:
```json
// replica1.json: "strong"
// replica2.json: "eventual"  
// replica3.json: "eventual"
```

---

## Metrics Collection

### Metrics to Record

For each test scenario, record the following metrics:

#### 1. PUT Latency
**Definition:** Time from client sending PUT request to receiving response.

**How to measure:** The client automatically measures and displays this:
```
> put 1 key value
Response (took 5.2ms): success=true, version=1
```

**Record in table:**
| Consistency Model | Delay | PUT Latency |
|-------------------|-------|-------------|
| Eventual | 0ms | 5ms |
| Eventual | 500ms | 7ms |
| Strong | 0ms | 12ms |
| Strong | 500ms | 520ms |

#### 2. GET Latency
**Definition:** Time from client sending GET request to receiving response.

**How to measure:** Client automatically displays:
```
> get 1 key
Response (took 1.8ms): value='test', version=1
```

#### 3. Convergence Time
**Definition:** Time from PUT completion until all replicas return the same value.

**How to measure:**
1. Note the time when PUT completes
2. Repeatedly run `getall <key>` until all replicas show the same value
3. Calculate the difference

**Script approach:**
```bash
# In client command mode:
time {
  go run main.go put 1 testkey testvalue
  while true; do
    result=$(go run main.go getall testkey 2>&1)
    if echo "$result" | grep -q "version=1" && [ $(echo "$result" | grep -c "version=1") -eq 3 ]; then
      break
    fi
    sleep 0.1
  done
}
```

#### 4. Number of Updated Replicas After PUT
**Definition:** How many replicas have the new value immediately after PUT returns.

**How to measure:** Immediately after PUT, run `getall`:
```
> put 1 key value
> getall key
```

**Record:**
- Eventual consistency: Usually only 1 (the receiving replica)
- Strong consistency: At least majority (2 out of 3)

#### 5. Stale Read Count
**Definition:** Number of times a GET returns an older version than the latest committed version.

**How to measure:** In scenarios 1 and 4, count how many reads return stale data before convergence.

### Sample Metrics Table

| Model | Scenario | PUT Latency | GET Latency | Convergence Time | Replicas Updated | Stale Reads |
|-------|----------|-------------|-------------|------------------|------------------|-------------|
| Eventual | 1 (0ms delay) | 5ms | 2ms | 50ms | 1 of 3 | 2 |
| Eventual | 1 (500ms delay) | 7ms | 2ms | 550ms | 1 of 3 | 5 |
| Eventual | 1 (2000ms delay) | 8ms | 2ms | 2050ms | 1 of 3 | 10 |
| Strong | 1 (0ms delay) | 15ms | 2ms | 0ms | 2 of 3 | 0 |
| Strong | 1 (500ms delay) | 520ms | 2ms | 0ms | 2 of 3 | 0 |
| Eventual | 2 (failure) | 6ms | 2ms | N/A (one replica down) | 2 of 3 | 0 |
| Strong | 2 (failure) | 15ms | 2ms | N/A (one replica down) | 2 of 3 | 0 |
| Strong | 2 (2 failures) | FAIL | - | - | - | - |
| Eventual | 3 (conflict) | 5ms | 2ms | 100ms | 3 of 3 | 0 |
| Eventual | 4 (0ms delay) | 5ms | 2ms | 50ms | 1 of 3 | 3 |
| Eventual | 4 (500ms delay) | 7ms | 2ms | 550ms | 1 of 3 | 8 |
| Eventual | 4 (2000ms delay) | 8ms | 2ms | 2050ms | 1 of 3 | 15 |

---

## Troubleshooting

### Common Issues and Solutions

#### 1. "Address already in use" Error

**Symptom:**
```
listen tcp :8001: bind: address already in use
```

**Cause:** Another process is using one of the required ports (8001, 8002, or 8003).

**Solution:**
```bash
# Find the process using the port
lsof -i :8001

# Kill the process (replace PID with actual process ID)
kill -9 <PID>

# Or change the port in the config file:
// configs/replica1.json
{
  "port": 8004  // Change to an available port
}
```

#### 2. Replicas Cannot Communicate

**Symptom:**
```
[replica1] Failed to replicate to http://localhost:8002: connection refused
```

**Cause:** One or more replicas are not running.

**Solution:**
1. Verify all replicas are running in separate terminals
2. Check the port numbers in config files match the running instances
3. Run `health` command from client to verify:
   ```
   > health
   ```

#### 3. Client Cannot Connect

**Symptom:**
```
Error: Post "http://localhost:8001/put": dial tcp: connection refused
```

**Cause:** The replica at the specified address is not running.

**Solution:**
1. Start the replica server
2. Verify the port number matches

#### 4. PUT Always Fails in Strong Consistency Mode

**Symptom:**
```
Response: success=false, Failed to achieve majority consensus
```

**Cause:** Not enough replicas are running to form a majority.

**Solution:**
- For 3 replicas, at least 2 must be running
- Check with `health` command
- Start the missing replica(s)

#### 5. Values Not Replicating

**Symptom:** After PUT, `getall` shows value only on one replica.

**Cause:** Network delay or replica communication issue.

**Solution:**
1. Check replica logs for replication errors
2. Verify peer URLs in config files are correct
3. Check that all replicas are running
4. Wait longer (replication may be delayed)
5. Check consistency model setting

#### 6. Go Module Errors

**Symptom:**
```
go: cannot find main module
```

**Solution:**
```bash
cd replica
go mod init hw3/replica
# OR if go.mod exists but is corrupted:
rm go.mod go.sum
go mod init hw3/replica
```

---

## Expected Learning Outcomes

After completing this exercise, you should be able to:

### 1. Explain Replication Fundamentals
- Articulate why distributed systems replicate data (availability, performance, fault tolerance)
- Distinguish between replication for performance vs. fault tolerance
- Identify the consistency-availability trade-off

### 2. Understand Consistency Models
- Define and compare strong consistency and eventual consistency
- Explain client-centric consistency guarantees (read-your-writes, monotonic reads)
- Choose appropriate consistency models for different application requirements

### 3. Apply the CAP Theorem
- State the CAP theorem and its implications
- Classify systems as CP or AP based on their behavior during partitions
- Understand why perfect consistency, availability, and partition tolerance cannot coexist

### 4. Implement Distributed Systems
- Build independent services that communicate over HTTP
- Implement synchronous and asynchronous replication
- Design RESTful APIs for distributed systems
- Handle concurrent access with proper locking

### 5. Manage Data Versioning
- Implement version numbers for tracking data changes
- Use versions to detect stale updates
- Understand why versioning is necessary for conflict detection

### 6. Resolve Conflicts
- Implement Last-Write-Wins conflict resolution
- Understand alternative strategies (CRDTs, multi-version, application-level)
- Recognize the limitations of simple conflict resolution

### 7. Test Distributed Systems
- Design test scenarios for distributed systems
- Simulate failures (stopping replicas)
- Simulate network delays
- Measure and analyze system behavior under different conditions

### 8. Analyze Trade-offs
- Interpret latency, convergence time, and consistency metrics
- Make informed design decisions based on application requirements
- Understand the practical implications of theoretical concepts

### 9. Debug Distributed Systems
- Use logging to trace replication messages
- Identify causes of inconsistency
- Troubleshoot communication issues between services

### 10. Document System Design
- Write clear documentation explaining architecture decisions
- Create meaningful test scenarios and record results
- Present findings with supporting metrics and analysis

---

## Practical Skills Gained

| Skill | How Acquired |
|-------|---------------|
| **Go Programming** | Writing replica server and client code |
| **HTTP/REST APIs** | Designing and implementing API endpoints |
| **Concurrent Programming** | Goroutines for async replication, mutexes for thread safety |
| **JSON Serialization** | Data exchange between replicas and client |
| **System Design** | Architecting a distributed system with multiple components |
| **Testing Methodology** | Designing and executing test scenarios |
| **Performance Measurement** | Recording latency, convergence time, staleness |
| **Documentation** | Writing comprehensive README and analysis |

---

## Advanced Topics for Further Exploration

### 1. Vector Clocks
Instead of simple integer versions, use vector clocks to track causality between updates across replicas. This enables detection of concurrent vs. causally-related updates.

### 2. CRDTs (Conflict-Free Replicated Data Types)
For specific data types (counters, sets, maps), use mathematically-guaranteed merge operations that always converge without conflict.

### 3. Gossip Protocols
Instead of direct replication to all peers, use gossip-based dissemination where replicas randomly exchange updates, providing better scalability.

### 4. Read Repair
When a read detects inconsistency, trigger immediate repair by updating stale replicas with the latest value.

### 5. Hinted Handoff
When a replica is down, store updates intended for it on another replica. When it comes back online, forward the stored updates.

### 6. Anti-Entropy
Periodic background process that compares data between replicas and synchronizes differences, ensuring eventual convergence even if some replication messages were lost.

### 7. Quorum Reads
Extend strong consistency to reads by requiring a majority of replicas to agree on the returned value, not just writes.

### 8. Leader-Based Replication
Designate one replica as the leader that handles all writes, simplifying conflict resolution but creating a single point of failure (requiring leader election).

---

## References

1. **CAP Theorem:** Brewer, E. (2000). "Towards Robust Distributed Systems"
2. **Consistency Models:** Tanenbaum, A. & Van Steen, M. "Distributed Systems: Principles and Paradigms"
3. **Dynamo Paper:** DeCandia, G. et al. (2007). "Dynamo: Amazon's Highly Available Key-value Store" (Eventual consistency in practice)
4. **Paxos:** Lamport, L. (2001). "Paxos Made Simple" (Consensus algorithm for strong consistency)
5. **Raft:** Ongaro, D. & Ousterhout, J. (2014). "In Search of an Understandable Consensus Algorithm"

---

## License

This project is part of the Distributed Computing Fundamentals course at the University of Tehran, Faculty of Electrical and Computer Engineering, Spring 2025.

---

**End of Documentation**