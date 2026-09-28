// Package main implements a simple TCP-based publish-subscribe broker.
package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"net"
	"os/signal"
	"sync"
	"syscall"
)

var (
	subscribers = make(map[net.Conn]chan []byte)
	mu          sync.RWMutex
	verbose     bool
)

func logDebug(format string, v ...interface{}) {
	if verbose {
		log.Printf("[DEBUG] "+format, v...)
	}
}

func addSubscriber(conn net.Conn) {
	log.Printf("📡 New subscriber from %s", conn.RemoteAddr())
	send := make(chan []byte, 256)

	mu.Lock()
	subscribers[conn] = send
	mu.Unlock()

	// Writer goroutine: delivers messages to this subscriber
	go func() {
		defer func() {
			mu.Lock()
			delete(subscribers, conn)
			mu.Unlock()
			conn.Close()
			log.Printf("🔌 Subscriber %s removed", conn.RemoteAddr())
		}()
		for msg := range send {
			if _, err := conn.Write(msg); err != nil {
				log.Printf("⚠️ Write error to %s: %v", conn.RemoteAddr(), err)
				return
			}
			if _, err := conn.Write([]byte("\n")); err != nil {
				log.Printf("⚠️ Failed to write newline to %s: %v", conn.RemoteAddr(), err)
				return
			}
			logDebug("Sent %d bytes to %s", len(msg), conn.RemoteAddr())
		}
	}()

	// Reader goroutine: detect when subscriber disconnects (read returns error)
	go func() {
		buf := make([]byte, 1)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				mu.Lock()
				close(send)
				delete(subscribers, conn)
				mu.Unlock()
				logDebug("Subscriber %s disconnected", conn.RemoteAddr())
				return
			}
		}
	}()
}

func publishMessage(msg []byte) {
	mu.RLock()
	defer mu.RUnlock()
	log.Printf("📨 Broadcasting %d bytes to %d subscribers", len(msg), len(subscribers))
	for conn, send := range subscribers {
		select {
		case send <- msg:
			logDebug("Queued for %s", conn.RemoteAddr())
		default:
			log.Printf("⚠️ Subscriber %s channel full, removing", conn.RemoteAddr())
			close(send)
			delete(subscribers, conn)
			conn.Close()
		}
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	// Read command
	cmdLine, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("❌ Read error from %s: %v", conn.RemoteAddr(), err)
		return
	}
	cmd := cmdLine[:len(cmdLine)-1]

	switch cmd {
	case "sub":
		addSubscriber(conn)
		// Block forever – keep the connection open
		select {}
	case "pub":
		msgLine, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("❌ Failed to read message from publisher %s: %v", conn.RemoteAddr(), err)
			return
		}
		msg := []byte(msgLine[:len(msgLine)-1])
		if len(msg) == 0 {
			log.Printf("⚠️ Empty message from publisher %s", conn.RemoteAddr())
			return
		}
		logDebug("Received %d bytes from publisher %s", len(msg), conn.RemoteAddr())
		publishMessage(msg)
	default:
		log.Printf("❌ Unknown command '%s' from %s", cmd, conn.RemoteAddr())
	}
}

func startBroker(addr string, ctx context.Context) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	log.Printf("🚀 PubSub broker listening on %s", addr)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ln.Close()
				return
			default:
				conn, err := ln.Accept()
				if err != nil {
					select {
					case <-ctx.Done():
						return
					default:
						log.Printf("Accept error: %v", err)
						continue
					}
				}
				go handleConnection(conn)
			}
		}
	}()

	<-ctx.Done()
	log.Println("🛑 Shutting down broker...")
	mu.Lock()
	defer mu.Unlock()
	for conn, send := range subscribers {
		close(send)
		conn.Close()
	}
	subscribers = make(map[net.Conn]chan []byte)
	return nil
}

func main() {
	addr := flag.String("addr", ":9090", "listen address")
	verboseFlag := flag.Bool("verbose", true, "verbose logging")
	flag.Parse()
	verbose = *verboseFlag

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := startBroker(*addr, ctx); err != nil {
		log.Fatalf("Broker error: %v", err)
	}
	log.Println("✅ Broker stopped")
}