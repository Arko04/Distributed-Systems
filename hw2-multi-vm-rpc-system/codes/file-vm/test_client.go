// Simple HTTP client to test the file service.
// Usage: go run test_client.go -url http://localhost:8081/files/sample.jpg
package main

import (
    "flag"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "time"
)

func main() {
    url := flag.String("url", "http://localhost:8081/files/sample.jpg", "URL to fetch")
    output := flag.String("output", "downloaded.jpg", "Output filename")
    flag.Parse()

    client := http.Client{Timeout: 10 * time.Second}
    resp, err := client.Get(*url)
    if err != nil {
        log.Fatalf("Failed to fetch: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        log.Fatalf("HTTP error: %s", resp.Status)
    }

    out, err := os.Create(*output)
    if err != nil {
        log.Fatalf("Cannot create output file: %v", err)
    }
    defer out.Close()

    written, err := io.Copy(out, resp.Body)
    if err != nil {
        log.Fatalf("Write error: %v", err)
    }

    fmt.Printf("✅ Downloaded %d bytes from %s to %s\n", written, *url, *output)
}