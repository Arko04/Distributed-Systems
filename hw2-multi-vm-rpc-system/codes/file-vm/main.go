// Package main implements a simple HTTP file server for the distributed system.
// It serves static files from the ./files directory on a configurable port.
// Graceful shutdown ensures all active connections finish before exiting.
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
    "syscall"
    "time"
)

var (
    // Command-line flags
    port        = flag.String("port", "8081", "Port to listen on (e.g., 8081, :8081)")
    dir         = flag.String("dir", "./files", "Directory to serve files from")
    readTimeout = flag.Duration("read-timeout", 5*time.Second, "HTTP read timeout")
    writeTimeout = flag.Duration("write-timeout", 10*time.Second, "HTTP write timeout")
    idleTimeout  = flag.Duration("idle-timeout", 30*time.Second, "HTTP idle timeout")
)

// loggingMiddleware wraps an http.Handler and logs each request.
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("📥 %s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
        next.ServeHTTP(w, r)
    })
}

// serveFiles starts the HTTP file server and blocks until shutdown.
func serveFiles(addr, dirPath string) error {
    // Validate directory
    absPath, err := filepath.Abs(dirPath)
    if err != nil {
        return fmt.Errorf("invalid directory path: %w", err)
    }
    info, err := os.Stat(absPath)
    if err != nil {
        return fmt.Errorf("cannot access directory %s: %w", absPath, err)
    }
    if !info.IsDir() {
        return fmt.Errorf("%s is not a directory", absPath)
    }
    log.Printf("📁 Serving files from: %s", absPath)

    // Create file server
    fs := http.FileServer(http.Dir(absPath))
    handler := http.StripPrefix("/files/", fs)

    // Apply logging middleware
    mux := http.NewServeMux()
    mux.Handle("/files/", loggingMiddleware(handler))

    // Health check endpoint (optional, for monitoring)
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    srv := &http.Server{
        Addr:         addr,
        Handler:      mux,
        ReadTimeout:  *readTimeout,
        WriteTimeout: *writeTimeout,
        IdleTimeout:  *idleTimeout,
    }

    // Run server in a goroutine so we can handle shutdown
    go func() {
        log.Printf("🚀 File VM serving on %s", addr)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("❌ HTTP server error: %v", err)
        }
    }()

    // Wait for shutdown signal
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()
    <-ctx.Done()

    // Graceful shutdown with timeout
    log.Println("🛑 Shutdown signal received, stopping gracefully...")
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := srv.Shutdown(shutdownCtx); err != nil {
        log.Printf("⚠️ HTTP server shutdown error: %v", err)
        return err
    }
    log.Println("✅ File VM stopped cleanly")
    return nil
}

func main() {
    flag.Parse()

    // Build address (if port doesn't start with ':', add it)
    addr := *port
    if len(addr) > 0 && addr[0] != ':' {
        addr = ":" + addr
    }

    if err := serveFiles(addr, *dir); err != nil {
        log.Fatalf("❌ Failed to start file server: %v", err)
    }
}