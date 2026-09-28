
## Part 3 – HTTP Microservice, Containerization, and Testing

### 1. Service Overview

In Part 3, a simple HTTP arithmetic microservice is implemented in Go and then containerized using Docker. The service exposes two endpoints:

- `GET /health`  
  Returns the current health status of the service in JSON format.

- `GET /compute?op=<op>&a=<num>&b=<num>`  
  Performs an arithmetic operation on two numbers and returns a JSON response.

The service listens on port `8080` inside the container and is mapped to the host port `8080` using Docker.

---

### 2. Implemented Endpoints and Behavior

#### 2.1 `/health`

- **Method:** `GET`
- **URL:** `http://localhost:8080/health`
- **Example Response:**

  ```json
  {
    "status": "ok",
    "time": "2026-04-22T05:15:21Z"
  }
  ```

- **Behavior:**
  - If the method is **not** `GET` (e.g. `POST`), the service returns:
    - HTTP status: `405 Method Not Allowed`
    - JSON body: `{"error": "method not allowed"}`

This endpoint is used by the test script to check that the service has started successfully and is responsive.

---

#### 2.2 `/compute`

- **Method:** `GET`
- **URL pattern:**

  ```text
  /compute?op=<op>&a=<num>&b=<num>
  ```

- **Supported operations:**

  | op value | Description        |
  |----------|--------------------|
  | `add`    | addition (`a + b`) |
  | `sub`    | subtraction (`a - b`) |
  | `mul`    | multiplication (`a * b`) |
  | `div`    | division (`a / b`) |
  | `mod`    | modulo (`a % b`, using integer cast) |

- **Normal success response format:**

  ```json
  {
    "operation": "add",
    "a": 8,
    "b": 3,
    "result": 11
  }
  ```

- **Error response format (for all failures):**

  For `/compute`, errors are returned as:

  ```json
  {
    "operation": "div",
    "a": 10,
    "b": 0,
    "error": "division by zero"
  }
  ```

  or, for general validation errors (e.g. in method/health):

  ```json
  {
    "error": "method not allowed"
  }
  ```

---

### 3. Required Error Handling (Assignment Requirements)

The assignment explicitly requires handling the following error scenarios:

1. Missing parameters (`op`, `a`, or `b`)
2. Invalid `op` (unsupported operation)
3. Non-numeric `a` or `b`
4. Division by zero (and modulo by zero)
5. Wrong HTTP method

These are all implemented and verified.

---

### 4. Implementation of Error Handling in `main.go`

Below is a summary of how each error is handled in the code:

#### 4.1 Missing parameters

In `computeHandler`, the presence of all three parameters is checked:

```go
op := strings.ToLower(query.Get("op"))
aStr := query.Get("a")
bStr := query.Get("b")

if op == "" || aStr == "" || bStr == "" {
    writeJSON(w, http.StatusBadRequest, errorResponse{
        Error: "missing required parameters: op, a, b",
    })
    return
}
```

- **HTTP status:** `400 Bad Request`
- **Body contains:** `"missing required parameters: op, a, b"`

---

#### 4.2 Non-numeric `a` or `b`

Parsing is done with `strconv.ParseFloat`:

```go
a, err1 := strconv.ParseFloat(aStr, 64)
b, err2 := strconv.ParseFloat(bStr, 64)
if err1 != nil || err2 != nil {
    writeJSON(w, http.StatusBadRequest, errorResponse{
        Error: "invalid numeric parameters for a or b",
    })
    return
}
```

- **HTTP status:** `400 Bad Request`
- **Body contains:** `"invalid numeric parameters for a or b"`

---

#### 4.3 Invalid `op` (unsupported operation)

If `op` is not one of `add`, `sub`, `mul`, `div`, `mod`:

```go
default:
    resp.Error = "unsupported operation (supported: add, sub, mul, div, mod)"
    writeJSON(w, http.StatusBadRequest, resp)
    return
```

- **HTTP status:** `400 Bad Request`
- **Body contains:** `"unsupported operation (supported: add, sub, mul, div, mod)"`

---

#### 4.4 Division / modulo by zero

Division:

```go
case OpDiv:
    if b == 0 {
        resp.Error = "division by zero"
        writeJSON(w, http.StatusBadRequest, resp)
        return
    }
    resp.Result = a / b
```

Modulo:

```go
case OpMod:
    ai := int64(a)
    bi := int64(b)
    if bi == 0 {
        resp.Error = "modulo by zero"
        writeJSON(w, http.StatusBadRequest, resp)
        return
    }
    resp.Result = float64(ai % bi)
```

- **HTTP status:** `400 Bad Request`
- **Body contains:** `"division by zero"` or `"modulo by zero"`

---

#### 4.5 Wrong HTTP method

For both `/health` and `/compute`:

```go
if r.Method != http.MethodGet {
    writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
    return
}
```

- **HTTP status:** `405 Method Not Allowed`
- **Body contains:** `"method not allowed"`

---

### 5. Containerization

The service is containerized using the following `Dockerfile` (multi-stage build):

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

**Key points:**

- First stage (`builder`): builds a static Go binary (`server`) inside `golang:1.21-alpine`.
- Second stage (`alpine:latest`): only includes the compiled binary, making the final image small.
- Port `8080` is exposed and used by the test script via `-p 8080:8080`.

Build and run commands:

```bash
docker build -t compute-service .
docker run -d -p 8080:8080 --name compute-container compute-service
```

---

### 6. Automated Testing with `test_part3.sh`

A fully automated test script (`test_part3.sh` / `run_part3_tests.sh`) is used to:

1. Check Docker availability.
2. Validate the presence and basic structure of the `Dockerfile`.
3. Build the Docker image (`compute-service`).
4. Run the container (`compute-container`) on port `8080`.
5. Wait for the `/health` endpoint to become responsive.
6. Run a comprehensive test suite against `/health` and `/compute`.
7. Show the last lines of the Docker logs.
8. Optionally clean up the container.
9. Save all responses in a timestamped logs directory, e.g. `logs_20260422_084510/`.

**Sample execution output (already obtained):**

```text
========== Part 3 Automated Test Framework ==========

========== Checking Docker Availability ==========
✓ Docker daemon is running

========== Validating Dockerfile ==========
✓ Dockerfile appears valid

========== Cleaning Previous Containers ==========
✓ Cleaned old container

========== Building Docker Image ==========
✓ Image built successfully

========== Starting Container ==========
✓ Container started

========== Waiting for Service Startup ==========
✓ Service responsive (after 1 attempts)

========== Running API Test Suite ==========
  Test: Health Check... ✓ Health Check (HTTP 200)
  Test: Addition... ✓ Addition (HTTP 200)
  Test: Subtraction... ✓ Subtraction (HTTP 200)
  Test: Multiplication... ✓ Multiplication (HTTP 200)
  Test: Division... ✓ Division (HTTP 200)
  Test: Modulo... ✓ Modulo (HTTP 200)
  Test: Missing Params... ✓ Missing Params (HTTP 400)
  Test: Invalid Operation... ✓ Invalid Operation (HTTP 400)
  Test: Invalid Number A... ✓ Invalid Number A (HTTP 400)
  Test: Invalid Number B... ✓ Invalid Number B (HTTP 400)
  Test: Division by Zero... ✓ Division by Zero (HTTP 400)
  Test: Modulo by Zero... ✓ Modulo by Zero (HTTP 400)
  Test: Wrong HTTP method (POST /compute)... ✓ Wrong method correctly returned 405 + 'method not allowed'

✓ All tests passed (including all required error cases).

========== Docker Logs (last 20 lines) ==========
2026/04/22 05:15:21 Starting HTTP server on :8080
2026/04/22 05:15:21 GET /health from 192.168.65.1:61301
...
2026/04/22 05:15:21 POST /compute from 192.168.65.1:36361 (query: op=add&a=1&b=2)

Remove container? (y/n): y

========== Cleanup ==========
✓ Container stopped & removed
→ Logs saved to logs_20260422_084510/
```

---

### 7. Detailed Test Coverage Based on the Results

Each test in the output corresponds to a specific requirement:

| Test name                         | URL example                                               | Expected status | What it verifies                                  |
|----------------------------------|-----------------------------------------------------------|-----------------|---------------------------------------------------|
| Health Check                     | `/health`                                                | 200             | Service up and alive                              |
| Addition                         | `/compute?op=add&a=8&b=3`                                | 200             | Correct addition                                  |
| Subtraction                      | `/compute?op=sub&a=10&b=4`                               | 200             | Correct subtraction                               |
| Multiplication                   | `/compute?op=mul&a=6&b=7`                                | 200             | Correct multiplication                            |
| Division                         | `/compute?op=div&a=20&b=4`                               | 200             | Correct division                                  |
| Modulo                           | `/compute?op=mod&a=20&b=3`                               | 200             | Correct modulo behavior                           |
| Missing Params                   | `/compute?op=add&a=5`                                    | 400             | Missing `b` → missing parameters error            |
| Invalid Operation                | `/compute?op=pow&a=2&b=3`                                | 400             | Unsupported `op`                                  |
| Invalid Number A                 | `/compute?op=add&a=abc&b=2`                              | 400             | Non-numeric `a`                                   |
| Invalid Number B                 | `/compute?op=add&a=2&b=xyz`                              | 400             | Non-numeric `b`                                   |
| Division by Zero                 | `/compute?op=div&a=10&b=0`                               | 400             | Division by zero error handling                   |
| Modulo by Zero                   | `/compute?op=mod&a=10&b=0`                               | 400             | Modulo by zero error handling                     |
| Wrong HTTP method (POST)        | `POST /compute?op=add&a=1&b=2`                           | 405             | Method not allowed for `/compute`                 |

All of them passed, as visible in the script output (`✓ ...`).

---

### 8. Conclusion

- The Go HTTP microservice correctly implements the two required endpoints (`/health` and `/compute`) with JSON responses.
- All mandatory error cases from the assignment are handled:
  - missing parameters,
  - invalid `op`,
  - non-numeric operands,
  - division/modulo by zero,
  - wrong HTTP method.
- The service is properly containerized using a two-stage Dockerfile and runs successfully inside a Docker container.
- The automated test script compiles the image, runs the container, exercises all success and error scenarios, and confirms that:
  - HTTP status codes are correct,
  - error messages are descriptive and consistent,
  - the service is stable and does not crash for invalid input.


