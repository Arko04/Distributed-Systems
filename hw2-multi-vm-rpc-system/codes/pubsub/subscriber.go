// Package main implements a subscriber client for the PubSub broker.
//
// It connects, sends "sub\n", then prints every received JSON event.
// If the connection drops, it automatically retries indefinitely.
// Usage: go run subscriber.go -broker=host:port
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// MemoryEvent matches the JSON structure published by the Web VM.
type MemoryEvent struct {
	EventType   string `json:"event_type"`
	Service     string `json:"service"`
	MemoryMB    uint64 `json:"memory_mb"`
	ThresholdMB uint64 `json:"threshold_mb"`
	Timestamp   string `json:"timestamp"`
}

// connectWithRetry attempts to connect to the broker, retrying up to 5 times.
// If all attempts fail, it returns nil and the last error.
func connectWithRetry(addr string) (net.Conn, error) {
	var conn net.Conn
	var err error
	for i := 0; i < 5; i++ {
		conn, err = net.DialTimeout("tcp", addr, 3*time.Second)
		if err == nil {
			return conn, nil
		}
		log.Printf("⚠️ Connection attempt %d failed: %v, retrying in 2s...", i+1, err)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("failed to connect after 5 attempts: %w", err)
}

func main() {
	brokerAddr := flag.String("broker", "localhost:9090", "Broker address (host:port)")
	flag.Parse()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Main loop: reconnect if connection drops
	for {
		conn, err := connectWithRetry(*brokerAddr)
		if err != nil {
			log.Fatalf("❌ Cannot connect to broker at %s: %v", *brokerAddr, err)
		}

		// Send subscription command
		if _, err := fmt.Fprintf(conn, "sub\n"); err != nil {
			log.Printf("⚠️ Failed to send subscription command: %v", err)
			conn.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		log.Printf("✅ Connected to broker at %s, waiting for events...", *brokerAddr)

		// Start a goroutine to handle shutdown signal
		done := make(chan struct{})
		go func() {
			<-sigChan
			log.Println("🛑 Received interrupt, closing connection...")
			conn.Close()
			close(done)
		}()

		// Read events line by line
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) == 0 {
				continue // skip empty lines
			}
			var ev MemoryEvent
			if err := json.Unmarshal([]byte(line), &ev); err != nil {
				log.Printf("⚠️ Failed to parse event: %v\nRaw: %s", err, line)
				continue
			}
			// Pretty print the alert
			fmt.Printf("\n"+
				"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
				"🚨 %s\n"+
				"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
				"Service    : %s\n"+
				"Memory     : %d MB\n"+
				"Threshold  : %d MB\n"+
				"Timestamp  : %s\n"+
				"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n",
				ev.EventType, ev.Service, ev.MemoryMB, ev.ThresholdMB, ev.Timestamp)
		}

		// If we exit the loop, the connection was closed
		if err := scanner.Err(); err != nil {
			log.Printf("❌ Connection lost: %v", err)
		} else {
			log.Println("👋 Connection closed by server")
		}

		// Wait for the done signal (only if we haven't already exited)
		select {
		case <-done:
			log.Println("👋 Subscriber exiting")
			return
		default:
			log.Println("🔄 Reconnecting in 2 seconds...")
			time.Sleep(2 * time.Second)
		}
	}
}