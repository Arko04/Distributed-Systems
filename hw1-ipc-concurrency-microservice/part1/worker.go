// worker.go
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
)

const (
	reqPipePath  = "/tmp/hw1_req.pipe"
	respPipePath = "/tmp/hw1_resp.pipe"
)

type Request struct {
	Op string  `json:"op"`
	A  float64 `json:"a"`
	B  float64 `json:"b"`
}

type Response struct {
	Status   string   `json:"status"`       
	Result   *float64 `json:"result,omitempty"`   
	ErrorMsg string   `json:"error,omitempty"`   
}

func parseLine(line string) (Request, error) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return Request{}, errors.New("empty_request")
	}
	if len(fields) != 3 {
		return Request{}, fmt.Errorf("invalid_argument_count: got %d, want 3", len(fields))
	}

	op := strings.ToUpper(fields[0])

	a, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return Request{}, fmt.Errorf("invalid_number_a: %v", err)
	}

	b, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return Request{}, fmt.Errorf("invalid_number_b: %v", err)
	}

	return Request{
		Op: op,
		A:  a,
		B:  b,
	}, nil
}

func compute(req Request) (float64, error) {
	switch req.Op {
	case "ADD":
		return req.A + req.B, nil
	case "SUB":
		return req.A - req.B, nil
	case "MUL":
		return req.A * req.B, nil
	case "DIV":
		if req.B == 0 {
			return 0, errors.New("division_by_zero")
		}
		return req.A / req.B, nil
	case "MOD":
		if req.B == 0 {
			return 0, errors.New("mod_by_zero")
		}
		return float64(int64(req.A) % int64(req.B)), nil
	default:
		return 0, fmt.Errorf("unknown_operation: %s", req.Op)
	}
}

func ensurePipe(path string) error {
	if info, err := os.Stat(path); err == nil {
		if (info.Mode() & os.ModeNamedPipe) != 0 {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}

	if err := syscall.Mkfifo(path, 0600); err != nil && !os.IsExist(err) {
		return fmt.Errorf("mkfifo %s: %w", path, err)
	}
	return nil
}

func main() {
	log.SetPrefix("[WORKER] ")
	log.Println("starting worker...")

	if err := ensurePipe(reqPipePath); err != nil {
		log.Fatalf("cannot ensure req pipe: %v", err)
	}
	if err := ensurePipe(respPipePath); err != nil {
		log.Fatalf("cannot ensure resp pipe: %v", err)
	}

	reqFile, err := os.OpenFile(reqPipePath, os.O_RDONLY, 0600)
	if err != nil {
		log.Fatalf("failed to open req pipe: %v", err)
	}
	defer reqFile.Close()

	respFile, err := os.OpenFile(respPipePath, os.O_WRONLY, 0600)
	if err != nil {
		log.Fatalf("failed to open resp pipe: %v", err)
	}
	defer respFile.Close()

	reqReader := bufio.NewScanner(reqFile)
	respWriter := bufio.NewWriter(respFile)

	log.Println("worker is listening for requests...")

	for {
		if !reqReader.Scan() {
			// EOF یا قطع شدن pipe
			if err := reqReader.Err(); err != nil {
				log.Printf("error reading from pipe: %v", err)
			} else {
				log.Println("request pipe closed, exiting worker.")
			}
			break
		}
		line := strings.TrimSpace(reqReader.Text())
		if line == "" {
			continue
		}
		log.Printf("received request line: %q", line)

		req, err := parseLine(line)
		if err != nil {
			log.Printf("parse error: %v", err)
			resp := Response{
				Status:   "ERR",
				ErrorMsg: err.Error(),
			}
			writeResponse(respWriter, resp)
			continue
		}

		result, err := compute(req)
		if err != nil {
			log.Printf("compute error: %v", err)
			resp := Response{
				Status:   "ERR",
				ErrorMsg: err.Error(),
			}
			writeResponse(respWriter, resp)
			continue
		}

		resp := Response{
			Status: "OK",
			Result: &result,
		}
		writeResponse(respWriter, resp)
	}

	log.Println("worker stopped.")
}

func writeResponse(w *bufio.Writer, resp Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("failed to marshal response: %v", err)
		fmt.Fprintf(w, "ERR internal_error\n")
		_ = w.Flush()
		return
	}
	_, _ = w.WriteString(string(data) + "\n")
	if err := w.Flush(); err != nil {
		log.Printf("failed to flush response: %v", err)
	}
}
