# Distributed Systems

Coursework for **Fundamentals of Distributed Computing** at the University of Tehran, Faculty of Electrical and Computer Engineering (Spring 2026, Dr. Mohammadreza Shournia).

All implementation work is in **Go**. It covers inter-process communication, concurrency, RPC across multiple machines, publish/subscribe messaging, and replication with tunable consistency.

## Projects

| # | Project | What it does | Tech |
|---|---------|--------------|------|
| HW1 | [IPC, Concurrency & Microservice](hw1-ipc-concurrency-microservice/) | A calculator split into two processes that talk over **named pipes**; a benchmark of goroutine scheduling and context switching under CPU-bound and mixed workloads; a containerised HTTP compute service with health checks and tests | Go, named pipes, Docker, Python (plots) |
| HW2 | [Multi-VM RPC System](hw2-multi-vm-rpc-system/) | A web app spread across three VMs: a **gRPC** authentication service, a file server, and a web front end that publishes memory alerts through a custom **TCP pub/sub broker** | Go, gRPC / Protocol Buffers, Docker Compose, UTM VMs |
| HW3 | [Replicated Key-Value Store](hw3-replicated-kv-store/) | Three independent replicas with switchable **strong (majority quorum)** and **eventual** consistency, versioning, last-write-wins conflict resolution, failure and network-delay simulation, and an automated benchmark suite | Go, HTTP/JSON, Bash, Python (matplotlib) |
| Project | [Distributed Deep Learning](project-distributed-deep-learning/) | Paper study and presentation series: *On Improving Efficiency of Distributed Deep Learning Training* | — |

## Highlights

- **Replicated KV store (HW3):** writes in strong mode wait for a majority (2 of 3) and keep working with one replica down. Eventual mode acknowledges immediately and converges in the background. The `results/` and `report/charts/` folders compare latency, stale reads and convergence time at 0 ms, 500 ms and 2000 ms of injected delay.
- **Multi-VM system (HW2):** services talk only over real network addresses (no `localhost`). It was tested on three UTM virtual machines, and a Docker Compose setup is included for running it locally.

## Running

Each project folder has its own README with full instructions. A quick taste with HW3:

```bash
cd hw3-replicated-kv-store
# terminal 1-3: start the replicas
(cd replica && go run main.go ../configs/replica1.json)
(cd replica && go run main.go ../configs/replica2.json)
(cd replica && go run main.go ../configs/replica3.json)
# terminal 4: use the client
(cd client && go run main.go health)
(cd client && go run main.go put 1 name Alice)
(cd client && go run main.go getall name)
```

Or run every scenario automatically with `./run_all_tests.sh`.

> Note: several folders contain more than one `package main` file (for example `broker.go` and `subscriber.go`). Build them one file at a time, as the scripts and Dockerfiles do: `go build broker.go`.

## Team

HW1–HW3 and the project were done together with **Taha Majlesi**.

## Repository layout

```
hw1-ipc-concurrency-microservice/   part1 (named pipes), part2 (scheduling benchmark), part3 (HTTP service)
hw2-multi-vm-rpc-system/            codes/, report/ (implementation report), rpc-study/ (RPC theory report)
hw3-replicated-kv-store/            replica/, client/, configs/, results/, report/
project-distributed-deep-learning/  presentation slides
```
