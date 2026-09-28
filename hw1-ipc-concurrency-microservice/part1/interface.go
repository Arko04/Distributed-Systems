// interface.go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	reqPipePath  = "/tmp/hw1_req.pipe"
	respPipePath = "/tmp/hw1_resp.pipe"
)

// Response باید با ساختار worker یکسان باشد
type Response struct {
	Status   string   `json:"status"`
	Result   *float64 `json:"result,omitempty"`
	ErrorMsg string   `json:"error,omitempty"`
}

func main() {
	log.SetPrefix("[INTERFACE] ")
	log.Println("starting interface...")

	// چک وجود worker pipeها (نه ساختن آنها)
	if _, err := os.Stat(reqPipePath); err != nil {
		log.Fatalf("request pipe not found (%s). is worker running? err=%v", reqPipePath, err)
	}
	if _, err := os.Stat(respPipePath); err != nil {
		log.Fatalf("response pipe not found (%s). is worker running? err=%v", respPipePath, err)
	}

	reqFile, err := os.OpenFile(reqPipePath, os.O_WRONLY, 0600)
	if err != nil {
		log.Fatalf("failed to open req pipe for writing: %v", err)
	}
	defer reqFile.Close()

	respFile, err := os.OpenFile(respPipePath, os.O_RDONLY, 0600)
	if err != nil {
		log.Fatalf("failed to open resp pipe for reading: %v", err)
	}
	defer respFile.Close()

	reqWriter := bufio.NewWriter(reqFile)
	respReader := bufio.NewScanner(respFile)
	stdinReader := bufio.NewScanner(os.Stdin)

	fmt.Println("Simple IPC Calculator Interface")
	fmt.Println("Format: OP A B (e.g., ADD 5 7)")
	fmt.Println("Supported ops: ADD, SUB, MUL, DIV, MOD")
	fmt.Println("Type 'exit' to quit.")

	for {
		fmt.Print("> ")
		if !stdinReader.Scan() {
			fmt.Println("\ninput closed, exiting.")
			break
		}
		line := strings.TrimSpace(stdinReader.Text())
		if line == "" {
			continue
		}
		if strings.EqualFold(line, "exit") {
			fmt.Println("bye.")
			break
		}

		// ارسال درخواست به worker
		_, err := reqWriter.WriteString(line + "\n")
		if err != nil {
			log.Printf("failed to write to req pipe: %v", err)
			fmt.Println("ERR failed_to_send_request (pipe issue)")
			break
		}
		if err := reqWriter.Flush(); err != nil {
			log.Printf("failed to flush req pipe: %v", err)
			fmt.Println("ERR failed_to_send_request (pipe flush issue)")
			break
		}

		// منتظر پاسخ
		if !respReader.Scan() {
			// worker ظاهراً مرده یا pipe قطع شده
			if err := respReader.Err(); err != nil {
				log.Printf("error reading response: %v", err)
			} else {
				log.Println("response pipe closed by worker.")
			}
			fmt.Println("ERR worker_unavailable")
			break
		}

		respLine := strings.TrimSpace(respReader.Text())
		if respLine == "" {
			fmt.Println("ERR empty_response")
			continue
		}

		var resp Response
		if err := json.Unmarshal([]byte(respLine), &resp); err != nil {
			// اگر JSON نبود، به عنوان متن ساده چاپ می‌کنیم
			log.Printf("failed to parse JSON response: %v, raw=%q", err, respLine)
			fmt.Println("Raw response from worker:", respLine)
			continue
		}

		if strings.EqualFold(resp.Status, "OK") && resp.Result != nil {
			fmt.Printf("OK %v\n", *resp.Result)
		} else {
			fmt.Printf("ERR %s\n", resp.ErrorMsg)
		}
	}

	log.Println("interface stopped.")
}
