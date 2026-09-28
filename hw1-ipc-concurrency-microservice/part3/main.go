package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	OpAdd = "add"
	OpSub = "sub"
	OpMul = "mul"
	OpDiv = "div"
	OpMod = "mod"
)

type response struct {
	Operation string  `json:"operation"`
	A         float64 `json:"a"`
	B         float64 `json:"b"`
	Result    float64 `json:"result,omitempty"`
	Error     string  `json:"error,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	resp := map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, resp)
}

func computeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	query := r.URL.Query()
	op := strings.ToLower(query.Get("op"))
	aStr := query.Get("a")
	bStr := query.Get("b")

	if op == "" || aStr == "" || bStr == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "missing required parameters: op, a, b",
		})
		return
	}

	// 2) Parse numbers.
	a, err1 := strconv.ParseFloat(aStr, 64)
	b, err2 := strconv.ParseFloat(bStr, 64)
	if err1 != nil || err2 != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid numeric parameters for a or b",
		})
		return
	}

	resp := response{
		Operation: op,
		A:         a,
		B:         b,
	}

	switch op {
	case OpAdd:
		resp.Result = a + b
	case OpSub:
		resp.Result = a - b
	case OpMul:
		resp.Result = a * b
	case OpDiv:
		if b == 0 {
			resp.Error = "division by zero"
			writeJSON(w, http.StatusBadRequest, resp)
			return
		}
		resp.Result = a / b
	case OpMod:
		ai := int64(a)
		bi := int64(b)
		if bi == 0 {
			resp.Error = "modulo by zero"
			writeJSON(w, http.StatusBadRequest, resp)
			return
		}
		resp.Result = float64(ai % bi)
	default:
		resp.Error = "unsupported operation (supported: add, sub, mul, div, mod)"
		writeJSON(w, http.StatusBadRequest, resp)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		healthHandler(w, r)
	})

	http.HandleFunc("/compute", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s from %s (query: %s)", r.Method, r.URL.Path, r.RemoteAddr, r.URL.RawQuery)
		computeHandler(w, r)
	})

	addr := ":8080"
	log.Printf("Starting HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
