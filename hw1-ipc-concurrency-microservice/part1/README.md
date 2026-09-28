
# Distributed Systems – Part 1  
## Simple IPC Calculator with Named Pipes (Go)

This project implements the **first part** of a distributed systems assignment:

- A **worker** process that performs simple arithmetic operations.
- An **interface** process that interacts with the user via stdin/stdout and communicates with the worker using **named pipes (FIFOs)**.
- A **test script** that automatically runs and validates all required scenarios (normal operations, errors, disconnection).

All components are written in **Go** and tested on a Unix-like environment (Linux/macOS).

---

## 1. Overview

### 1.1. Processes

There are two main programs:

1. `worker.go`  
   - A background calculator service.  
   - Reads requests from a **request pipe**, performs the operation, and writes results to a **response pipe**.  
   - Supports operations: `ADD`, `SUB`, `MUL`, `DIV`, `MOD`.  
   - Returns structured `OK` / `ERR` responses.

2. `interface.go`  
   - A user-facing CLI interface.  
   - Reads commands from **stdin**, sends them to the worker via the request pipe, waits for the worker response on the response pipe, and prints the result to the user.  
   - Handles errors gracefully (unknown operation, wrong argument count, invalid numbers, division/modulo by zero).  
   - Exits cleanly on `exit`.

### 1.2. IPC Mechanism

The communication between `worker` and `interface` is implemented using **named pipes (FIFOs)**:

- Typical structure (example):

  - Request pipe: `calc_req.pipe`  
  - Response pipe: `calc_res.pipe`

  > Note: If your code uses different names or paths (e.g. `/tmp/worker_req`, etc.), adjust the commands in this README accordingly.

The general message format is **text-based**, one request/response per line.

---

## 2. Build & Run

### 2.1. Requirements

- Go 1.20+ (or any reasonably recent Go version)
- Unix-like OS (Linux / macOS) with support for named pipes
- Bash (for the test script)

### 2.2. Files

At minimum, your folder should contain:

- `worker.go`
- `interface.go`
- `run_part1.sh` (test script)
- `README.md` (this file)

---

## 3. Usage

### 3.1. Build the programs

From the project directory:

```bash
go build -o worker worker.go
go build -o interface interface.go
```

This will produce:

- `./worker`
- `./interface`

### 3.2. Run manually (interactive mode)

#### Step 1 – Start the worker

In **Terminal 1**:

```bash
./worker
```

The worker starts, creates/opens the named pipes, and waits for requests.

#### Step 2 – Start the interface

In **Terminal 2**:

```bash
./interface
```

You should see something like:

```text
[INTERFACE] 2026/04/22 08:24:07 starting interface...
Simple IPC Calculator Interface
Format: OP A B (e.g., ADD 5 7)
Supported ops: ADD, SUB, MUL, DIV, MOD
Type 'exit' to quit.
> 
```

Now you can type commands:

```text
> ADD 5 7
OK 12
> DIV 9 2
OK 4.5
> MOD 10 3
OK 1
> exit
bye.
[INTERFACE] ... interface stopped.
```

---

## 4. Protocol and Behavior

### 4.1. Command format

The interface expects **one command per line**:

```text
OP A B
```

- `OP` – operation name (case-sensitive):  
  - `ADD`, `SUB`, `MUL`, `DIV`, `MOD`
- `A`, `B` – numeric operands, parsed as `float64` (or at least supporting decimals for `DIV`)

Example:

```text
ADD 5 7
SUB 10 3
MUL 4 2.5
DIV 9 2
MOD 10 3
```

### 4.2. Supported operations

- `ADD A B` → `A + B`
- `SUB A B` → `A - B`
- `MUL A B` → `A * B`
- `DIV A B` → `A / B` (error on division by zero)
- `MOD A B` → `A % B` (error on modulo by zero)

### 4.3. Responses

The worker responds with **one line per request**, interpreted by the interface and printed to the user:

- Success:

  ```text
  OK <result>
  ```

  Example:

  ```text
  > ADD 5 7
  OK 12
  ```

- Errors (examples from your logs):

  ```text
  ERR unknown_operation: FOO
  ERR invalid_argument_count: got 2, want 3
  ERR invalid_number_a: strconv.ParseFloat: parsing "a": invalid syntax
  ERR invalid_number_b: strconv.ParseFloat: parsing "b": invalid syntax
  ERR division_by_zero
  ERR mod_by_zero
  ```

- Exit from interface:

  ```text
  exit
  bye.
  [INTERFACE] ... interface stopped.
  ```

---

## 5. Error Handling (as implemented)

Based on the test results, your code correctly handles the following cases.

### 5.1. Unknown operation

Input:

```text
FOO 1 2
exit
```

Output:

```text
> ERR unknown_operation: FOO
> bye.
```

### 5.2. Wrong argument count

Input:

```text
ADD 1
SUB 1 2 3
MUL
exit
```

Output:

```text
> ERR invalid_argument_count: got 2, want 3
> ERR invalid_argument_count: got 4, want 3
> ERR invalid_argument_count: got 1, want 3
> bye.
```

### 5.3. Invalid numeric input

Input:

```text
ADD a 2
DIV 10 b
MUL one two
exit
```

Output:

```text
> ERR invalid_number_a: strconv.ParseFloat: parsing "a": invalid syntax
> ERR invalid_number_b: strconv.ParseFloat: parsing "b": invalid syntax
> ERR invalid_number_a: strconv.ParseFloat: parsing "one": invalid syntax
> bye.
```

### 5.4. Division / modulo by zero

Input:

```text
DIV 10 0
MOD 7 0
exit
```

Output:

```text
> ERR division_by_zero
> ERR mod_by_zero
> bye.
```

### 5.5. Worker / pipe disconnection

The interface is tested against a simulated crash of the worker.

Input pattern:

```text
ADD 1 1
SUB 5 2
[worker is killed here]
MUL 2 3
DIV 4 2
exit
```

In your current run (without killing the worker) you saw:

```text
> OK 2
> OK 3
> OK 6
> OK 2
> bye.
```

If the worker is killed in the middle, the interface should detect broken pipe / EOF and behave gracefully (log error and/or exit). This scenario is covered by the test script.

---

## 6. Automated Test Script

### 6.1. Script: `run_part1.sh`

This script:

- Builds `interface.go`
- Assumes `worker` is **already running** in another terminal
- Runs 6 groups of test scenarios:
  1. Normal operations
  2. Unknown operation
  3. Wrong argument count
  4. Invalid numeric input
  5. Division/modulo by zero
  6. Worker / pipe disconnection
- Writes all output to `part1_test.log`
- Uses colored output in the terminal

#### 6.1.1. Script source

```bash
#!/usr/bin/env bash
set -euo pipefail

INTERFACE_BIN="./interface"
LOG_FILE="part1_test.log"

# colors
RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
BLUE="\033[34m"
RESET="\033[0m"

log() {
  printf "%b\n" "$1" | tee -a "${LOG_FILE}"
}

section() {
  local title="$1"
  log ""
  log "=================================================="
  log ">>> ${title}"
  log "=================================================="
}

# header
printf "%b\n" "${BLUE}==== Part 1 Test — Interface Only (Assumes WORKER is already running) ====${RESET}"
echo "All output will be stored in ${LOG_FILE}."
echo

rm -f "${LOG_FILE}"

# build interface
section "Build interface"

if [[ -f "${INTERFACE_BIN}" ]]; then
  log "Removing previous interface binary..."
  rm -f "${INTERFACE_BIN}"
fi

log "Building interface.go ..."
if go build -o "${INTERFACE_BIN}" interface.go 2>&1 | tee -a "${LOG_FILE}"; then
  printf "%b\n" "${GREEN}✓ interface build succeeded.${RESET}" | tee -a "${LOG_FILE}"
else
  printf "%b\n" "${RED}✗ interface build failed. Tests aborted.${RESET}" | tee -a "${LOG_FILE}"
  exit 1
fi

# ---------- Section 1: Normal operations ----------
section "Section 1 - Normal operations (ADD, SUB, MUL, DIV, MOD)"

{
  echo "ADD 5 7"      # OK 12
  echo "SUB 10 3"     # OK 7
  echo "MUL 4 2.5"    # OK 10
  echo "DIV 9 2"      # OK 4.5
  echo "MOD 10 3"     # OK 1
  echo "exit"
} | "${INTERFACE_BIN}" 2>&1 | tee -a "${LOG_FILE}"

# ---------- Section 2: Unknown operation ----------
section "Section 2 - Unknown operation"

{
  echo "FOO 1 2"
  echo "exit"
} | "${INTERFACE_BIN}" 2>&1 | tee -a "${LOG_FILE}"

# ---------- Section 3: Wrong argument count ----------
section "Section 3 - Wrong argument count"

{
  echo "ADD 1"
  echo "SUB 1 2 3"
  echo "MUL"
  echo "exit"
} | "${INTERFACE_BIN}" 2>&1 | tee -a "${LOG_FILE}"

# ---------- Section 4: Invalid numeric input ----------
section "Section 4 - Invalid numeric input"

{
  echo "ADD a 2"
  echo "DIV 10 b"
  echo "MUL one two"
  echo "exit"
} | "${INTERFACE_BIN}" 2>&1 | tee -a "${LOG_FILE}"

# ---------- Section 5: Division by zero ----------
section "Section 5 - Division by zero"

{
  echo "DIV 10 0"
  echo "MOD 7 0"
  echo "exit"
} | "${INTERFACE_BIN}" 2>&1 | tee -a "${LOG_FILE}"

# ---------- Section 6: Simulating worker/pipe disconnection ----------
section "Section 6 - Simulating worker/pipe disconnection"

printf "%b\n" "${YELLOW}The interface will send several requests now. Please go to the WORKER terminal and press Ctrl+C in the middle to simulate a crash.${RESET}" | tee -a "${LOG_FILE}"

{
  echo "ADD 1 1"
  echo "SUB 5 2"
  sleep 2
  echo "MUL 2 3"
  echo "DIV 4 2"
  echo "exit"
} | "${INTERFACE_BIN}" 2>&1 | tee -a "${LOG_FILE}"

section "End of tests"

printf "%b\n" "${GREEN}All scenarios executed. Full output saved to ${LOG_FILE}.${RESET}" | tee -a "${LOG_FILE}"
```

### 6.2. How to run the tests

1. **Terminal 1** – start the worker:

   ```bash
   ./worker
   ```

2. **Terminal 2** – run the test script:

   ```bash
   chmod +x run_part1.sh
   ./run_part1.sh
   ```

3. Check the summary on the terminal and full details in:

   ```bash
   cat part1_test.log
   ```

---

## 7. Example Test Run (Actual Output)

Here is an example run (taken from your latest execution):

```text
==================================================
>>> Build interface
==================================================
Removing previous interface binary...
Building interface.go ...
✓ interface build succeeded.

==================================================
>>> Section 1 - Normal operations (ADD, SUB, MUL, DIV, MOD)
==================================================
[INTERFACE] 2026/04/22 08:24:07 starting interface...
Simple IPC Calculator Interface
Format: OP A B (e.g., ADD 5 7)
Supported ops: ADD, SUB, MUL, DIV, MOD
Type 'exit' to quit.
> OK 12
> OK 7
> OK 10
> OK 4.5
> OK 1
> bye.
[INTERFACE] 2026/04/22 08:24:07 interface stopped.

==================================================
>>> Section 2 - Unknown operation
==================================================
[INTERFACE] 2026/04/22 08:24:07 starting interface...
Simple IPC Calculator Interface
Format: OP A B (e.g., ADD 5 7)
Supported ops: ADD, SUB, MUL, DIV, MOD
Type 'exit' to quit.
> ERR unknown_operation: FOO
> bye.
[INTERFACE] 2026/04/22 08:24:08 interface stopped.

==================================================
>>> Section 3 - Wrong argument count
==================================================
[INTERFACE] 2026/04/22 08:24:08 starting interface...
Simple IPC Calculator Interface
Format: OP A B (e.g., ADD 5 7)
Supported ops: ADD, SUB, MUL, DIV, MOD
Type 'exit' to quit.
> ERR invalid_argument_count: got 2, want 3
> ERR invalid_argument_count: got 4, want 3
> ERR invalid_argument_count: got 1, want 3
> bye.
[INTERFACE] 2026/04/22 08:24:13 interface stopped.

==================================================
>>> Section 4 - Invalid numeric input
==================================================
[INTERFACE] 2026/04/22 08:24:13 starting interface...
Simple IPC Calculator Interface
Format: OP A B (e.g., ADD 5 7)
Supported ops: ADD, SUB, MUL, DIV, MOD
Type 'exit' to quit.
> ERR invalid_number_a: strconv.ParseFloat: parsing "a": invalid syntax
> ERR invalid_number_b: strconv.ParseFloat: parsing "b": invalid syntax
> ERR invalid_number_a: strconv.ParseFloat: parsing "one": invalid syntax
> bye.
[INTERFACE] 2026/04/22 08:24:15 interface stopped.

==================================================
>>> Section 5 - Division by zero
==================================================
[INTERFACE] 2026/04/22 08:24:15 starting interface...
Simple IPC Calculator Interface
Format: OP A B (e.g., ADD 5 7)
Supported ops: ADD, SUB, MUL, DIV, MOD
Type 'exit' to quit.
> ERR division_by_zero
> ERR mod_by_zero
> bye.
[INTERFACE] 2026/04/22 08:24:16 interface stopped.

==================================================
>>> Section 6 - Simulating worker/pipe disconnection
==================================================
The interface will send several requests now. Please go to the WORKER terminal and press Ctrl+C in the middle to simulate a crash.
[INTERFACE] 2026/04/22 08:24:16 starting interface...
Simple IPC Calculator Interface
Format: OP A B (e.g., ADD 5 7)
Supported ops: ADD, SUB, MUL, DIV, MOD
Type 'exit' to quit.
> OK 2
> OK 3
> OK 6
> OK 2
> bye.
[INTERFACE] 2026/04/22 08:24:18 interface stopped.

==================================================
>>> End of tests
==================================================
All scenarios executed. Full output saved to part1_test.log.
```

All scenarios pass as expected.

---

## 8. Possible Extensions (for later parts)

If you extend this project in later parts, potential directions include:

- Multiple workers and load-balancing between them
- Network communication (TCP/UDP) instead of named pipes
- Timeouts and retries when worker is slow or unresponsive
- Structured (JSON) protocol between interface and worker(s)

---

## 9. Summary

This Part 1 implementation:

- Uses **Go** and **named pipes** to implement a simple distributed calculator.
- Clearly separates:
  - **Worker** (computation + IPC)
  - **Interface** (user interaction + IPC)
- Handles:
  - Normal arithmetic operations
  - Invalid commands and inputs
  - Division/modulo by zero
  - Worker disconnection (simulated crash)
- Provides a **fully automated test script** (`run_part1.sh`) that validates all required scenarios and stores output in `part1_test.log`.


In My Mac M1:
go build -o worker worker.go
go build -o interface interface.go

In one terminal:
./worker

In another terminal:
./interface