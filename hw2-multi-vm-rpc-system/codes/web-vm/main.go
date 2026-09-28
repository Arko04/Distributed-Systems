// Package main implements the Web VM for the distributed system assignment.
//
// It provides:
//   - A login page that authenticates users via gRPC (Auth VM).
//   - A dashboard that displays an image from the File VM (VM3) and shows service IPs.
//   - A memory monitoring goroutine that publishes events to a Pub/Sub broker
//     when memory usage exceeds the threshold (default 300 MB).
//   - An endpoint /consume-memory to manually allocate memory for testing alerts.
//
// All external service addresses are configurable via command-line flags or
// environment variables, making it easy to run on separate VMs with dynamic IPs.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	// gRPC stubs generated from auth.proto; located in ./authpb/
	pb "web/authpb"
)

// ----------------------------------------------------------------------
// Configuration (command-line flags + environment variable fallback)
// ----------------------------------------------------------------------

var (
	// Service addresses
	authAddr   = flag.String("auth", getEnvDefault("WEB_AUTH_ADDR", "127.0.0.1:50051"), "Auth VM gRPC address (host:port)")
	fileAddr   = flag.String("file", getEnvDefault("WEB_FILE_BASE_URL", "http://127.0.0.1:8081"), "File VM HTTP base URL")
	brokerAddr = flag.String("broker", getEnvDefault("WEB_BROKER_ADDR", "127.0.0.1:9090"), "Pub/Sub broker address (host:port)")

	// Memory threshold in MB
	thresholdMB = flag.Uint("threshold", getEnvIntDefault("WEB_THRESHOLD_MB", 300), "Memory usage threshold in MB (alerts when exceeded)")

	// Web server listen address
	listenAddr = flag.String("listen", getEnvDefault("WEB_LISTEN_ADDR", ":8080"), "Web server listen address (e.g., :8080, 0.0.0.0:8080)")

	// Memory check interval
	monitorInterval = flag.Duration("interval", 10*time.Second, "Memory check interval (e.g., 10s, 30s)")

	// Verbose logging
	verbose = flag.Bool("verbose", true, "Enable verbose logging")
)

func getEnvDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func getEnvIntDefault(key string, defaultValue uint) uint {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.ParseUint(val, 10, 64); err == nil {
			return uint(parsed)
		}
	}
	return defaultValue
}

// ----------------------------------------------------------------------
// Global state
// ----------------------------------------------------------------------

var (
	allocatedMemory [][]byte           // holds manually allocated chunks to prevent GC
	loginTmpl       *template.Template // parsed login.html
	dashboardTmpl   *template.Template // parsed dashboard.html
)

// ----------------------------------------------------------------------
// Utility functions
// ----------------------------------------------------------------------

func logDebug(format string, v ...interface{}) {
	if *verbose {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// resolveServiceIP returns the IP address of a given hostname (Docker service name)
func resolveServiceIP(serviceHost string) string {
	ips, err := net.LookupIP(serviceHost)
	if err != nil {
		log.Printf("⚠️ Failed to resolve %s: %v", serviceHost, err)
		return "unknown"
	}
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.String()
		}
	}
	return "unknown"
}

// ----------------------------------------------------------------------
// Pub/Sub Publisher
// ----------------------------------------------------------------------

func publishMemoryEvent(memMB uint64) {
	event := map[string]interface{}{
		"event_type":   "HIGH_MEMORY_USAGE",
		"service":      "web-server",
		"memory_mb":    memMB,
		"threshold_mb": *thresholdMB,
		"timestamp":    time.Now().Format(time.RFC3339),
	}
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("❌ Failed to marshal event: %v", err)
		return
	}

	conn, err := net.DialTimeout("tcp", *brokerAddr, 2*time.Second)
	if err != nil {
		log.Printf("⚠️ Cannot connect to broker at %s: %v", *brokerAddr, err)
		return
	}
	defer conn.Close()

	// Combine "pub\n", JSON, and newline into one write
	msg := append([]byte("pub\n"), append(data, '\n')...)
	if _, err := conn.Write(msg); err != nil {
		log.Printf("⚠️ Failed to write to broker: %v", err)
		return
	}
	log.Printf("📢 Published memory event: %d MB (threshold %d MB)", memMB, *thresholdMB)
}

// ----------------------------------------------------------------------
// Memory Monitor (goroutine)
// ----------------------------------------------------------------------

func monitorMemory(ctx context.Context) {
	ticker := time.NewTicker(*monitorInterval)
	defer ticker.Stop()

	logDebug("Memory monitor started, checking every %v", *monitorInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Memory monitor stopped")
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			memMB := m.Alloc / 1024 / 1024
			sysMB := m.Sys / 1024 / 1024
			logDebug("Memory stats: Alloc = %d MB, Sys = %d MB", memMB, sysMB)

			if memMB > uint64(*thresholdMB) {
				log.Printf("⚠️ Memory exceeded threshold: %d MB > %d MB", memMB, *thresholdMB)
				publishMemoryEvent(memMB)
			}
		}
	}
}

// ----------------------------------------------------------------------
// HTTP Handlers
// ----------------------------------------------------------------------

func consumeMemoryHandler(w http.ResponseWriter, r *http.Request) {
	mbStr := r.URL.Query().Get("mb")
	if mbStr == "" {
		http.Error(w, "❌ Missing 'mb' parameter", http.StatusBadRequest)
		return
	}
	mb, err := strconv.Atoi(mbStr)
	if err != nil || mb <= 0 {
		http.Error(w, "❌ 'mb' must be a positive integer", http.StatusBadRequest)
		return
	}

	chunk := make([]byte, mb*1024*1024)
	for i := range chunk {
		chunk[i] = byte(i & 0xFF)
	}
	allocatedMemory = append(allocatedMemory, chunk)

	log.Printf("💾 Manually allocated %d MB (total chunks held: %d)", mb, len(allocatedMemory))
	fmt.Fprintf(w, "✅ Allocated %d MB. Total allocated chunks: %d\n", mb, len(allocatedMemory))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if err := loginTmpl.Execute(w, nil); err != nil {
			log.Printf("❌ Failed to execute login template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		loginTmpl.Execute(w, map[string]string{"Error": "Invalid form data"})
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		loginTmpl.Execute(w, map[string]string{"Error": "Username and password required"})
		return
	}

	conn, err := grpc.Dial(*authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("❌ Cannot connect to Auth VM (%s): %v", *authAddr, err)
		loginTmpl.Execute(w, map[string]string{"Error": "Authentication service unavailable"})
		return
	}
	defer conn.Close()

	client := pb.NewAuthServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := client.Login(ctx, &pb.LoginRequest{Username: username, Password: password})
	if err != nil {
		log.Printf("❌ RPC error from Auth VM: %v", err)
		loginTmpl.Execute(w, map[string]string{"Error": "Authentication service error"})
		return
	}
	if !resp.Success {
		log.Printf("❌ Failed login attempt for user: %s", username)
		loginTmpl.Execute(w, map[string]string{"Error": "Invalid username or password"})
		return
	}

	log.Printf("✅ User %s logged in successfully", username)
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "authenticated",
		Path:   "/",
		MaxAge: 3600,
	})
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
    // Check session cookie
    cookie, err := r.Cookie("session")
    if err != nil || cookie.Value != "authenticated" {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }

    imageURL := *fileAddr + "/files/sample.jpg"

    // Resolve IPs of all services (Docker service names)
    authIP := resolveServiceIP("auth-vm")
    fileIP := resolveServiceIP("file-vm")
    brokerIP := resolveServiceIP("broker-vm")
    subscriberIP := resolveServiceIP("subscriber-vm")
    webIP := resolveServiceIP("web-vm")

    data := map[string]interface{}{
        "ImageURL":      imageURL,
        "ImageError":    "", // no error – assume image works
        "AuthIP":        authIP,
        "FileIP":        fileIP,
        "BrokerIP":      brokerIP,
        "SubscriberIP":  subscriberIP,
        "WebIP":         webIP,
        "ThresholdMB":   *thresholdMB,
    }

    if err := dashboardTmpl.Execute(w, data); err != nil {
        log.Printf("❌ Failed to execute dashboard template: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}


// ----------------------------------------------------------------------
// Template Initialisation
// ----------------------------------------------------------------------

func initTemplates() {
	var err error
	loginTmpl, err = template.ParseFiles("templates/login.html")
	if err != nil {
		log.Fatalf("❌ Missing or invalid templates/login.html: %v", err)
	}
	dashboardTmpl, err = template.ParseFiles("templates/dashboard.html")
	if err != nil {
		log.Fatalf("❌ Missing or invalid templates/dashboard.html: %v", err)
	}
	log.Println("✅ Templates loaded successfully")
}

// ----------------------------------------------------------------------
// Main Entry Point
// ----------------------------------------------------------------------

func main() {
	flag.Parse()

	if *authAddr == "" {
		log.Fatal("❌ -auth address must be set")
	}
	if *fileAddr == "" {
		log.Fatal("❌ -file base URL must be set")
	}
	if *brokerAddr == "" {
		log.Fatal("❌ -broker address must be set")
	}

	initTemplates()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go monitorMemory(ctx)

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/dashboard", dashboardHandler)
	http.HandleFunc("/consume-memory", consumeMemoryHandler)

	srv := &http.Server{
		Addr:         *listenAddr,
		Handler:      nil,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	go func() {
		log.Printf("🚀 Web VM starting on %s", *listenAddr)
		log.Printf("   Auth VM address: %s", *authAddr)
		log.Printf("   File VM base URL: %s", *fileAddr)
		log.Printf("   Pub/Sub broker: %s", *brokerAddr)
		log.Printf("   Memory threshold: %d MB, check interval: %v", *thresholdMB, *monitorInterval)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ HTTP server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("🛑 Shutdown signal received, stopping gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("⚠️ HTTP server shutdown error: %v", err)
	} else {
		log.Println("✅ HTTP server stopped cleanly")
	}
}