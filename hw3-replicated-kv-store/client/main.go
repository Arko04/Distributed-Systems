// ============================================================================
// CLIENT - Distributed Replicated Key-Value Store Client
// ============================================================================
// This file implements a command-line client for interacting with the
// replicated key-value store. It supports both interactive mode and
// command-line arguments for scripting.
//
// Features:
//   - Interactive command-line interface
//   - Command-line mode for scripting tests
//   - PUT, GET, and GETALL operations
//   - Replica health checking
//   - Start/stop replicas for failure simulation
//   - Automatic response time measurement
// ============================================================================

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// ============================================================================
// CLIENT STRUCTURE
// ============================================================================

// Client represents a test client for the replicated key-value store.
// It maintains connections to all replicas and provides methods for
// interacting with the distributed system.
type Client struct {
	// replicas is a list of replica URLs (e.g., ["http://localhost:8001", ...])
	replicas []string
	
	// client is an HTTP client used for all requests
	client *http.Client
}

// ============================================================================
// REQUEST/RESPONSE STRUCTURES
// ============================================================================

// PUTRequest is the JSON body sent for PUT operations
type PUTRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// PUTResponse is the JSON response received after a PUT operation
type PUTResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Version   int    `json:"version"`
	UpdatedBy string `json:"updated_by"`
}

// GETResponse is the JSON response received after a GET operation
type GETResponse struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Version   int    `json:"version"`
	UpdatedBy string `json:"updated_by"`
	Timestamp int64  `json:"timestamp"`
	Found     bool   `json:"found"`
}

// ============================================================================
// CONSTRUCTOR
// ============================================================================

// NewClient creates a new Client instance configured to connect to
// the specified replicas.
func NewClient(replicas []string) *Client {
	return &Client{
		replicas: replicas,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ============================================================================
// CORE OPERATIONS
// ============================================================================

// PUT sends a PUT request to store a key-value pair on a specific replica.
// It measures and returns the response time for performance analysis.
//
// Parameters:
//   - replicaURL: The full URL of the replica (e.g., "http://localhost:8001")
//   - key: The key to store
//   - value: The value to associate with the key
//
// Returns:
//   - *PUTResponse: The parsed response from the replica
//   - time.Duration: The time taken for the request
//   - error: Any error that occurred
func (c *Client) PUT(replicaURL, key, value string) (*PUTResponse, time.Duration, error) {
	reqBody := PUTRequest{
		Key:   key,
		Value: value,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Measure request time
	start := time.Now()
	resp, err := c.client.Post(
		replicaURL+"/put",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	duration := time.Since(start)

	if err != nil {
		return nil, duration, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, duration, fmt.Errorf("failed to read response: %w", err)
	}

	var putResp PUTResponse
	if err := json.Unmarshal(body, &putResp); err != nil {
		return nil, duration, fmt.Errorf("failed to parse response: %w", err)
	}

	return &putResp, duration, nil
}

// GET sends a GET request to retrieve a value from a specific replica.
// It measures and returns the response time.
//
// Parameters:
//   - replicaURL: The full URL of the replica
//   - key: The key to retrieve
//
// Returns:
//   - *GETResponse: The parsed response containing the value and metadata
//   - time.Duration: The time taken for the request
//   - error: Any error that occurred
func (c *Client) GET(replicaURL, key string) (*GETResponse, time.Duration, error) {
	start := time.Now()
	resp, err := c.client.Get(replicaURL + "/get?key=" + key)
	duration := time.Since(start)

	if err != nil {
		return nil, duration, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, duration, fmt.Errorf("failed to read response: %w", err)
	}

	var getResp GETResponse
	if err := json.Unmarshal(body, &getResp); err != nil {
		return nil, duration, fmt.Errorf("failed to parse response: %w", err)
	}

	return &getResp, duration, nil
}

// GetAllReplicas retrieves a value from ALL replicas simultaneously.
// This is useful for observing replication status and detecting inconsistencies.
//
// Returns a map from replica URL to its response.
func (c *Client) GetAllReplicas(key string) map[string]*GETResponse {
	results := make(map[string]*GETResponse)
	
	for _, replica := range c.replicas {
		resp, _, err := c.GET(replica, key)
		if err != nil {
			// Store a "not found" response for unreachable replicas
			results[replica] = &GETResponse{
				Key:   key,
				Found: false,
			}
		} else {
			results[replica] = resp
		}
	}
	
	return results
}

// ============================================================================
// REPLICA MANAGEMENT
// ============================================================================

// StopReplica sends a request to stop a specific replica.
// This simulates a replica failure for testing purposes.
func (c *Client) StopReplica(replicaURL string) error {
	resp, err := c.client.Post(replicaURL+"/stop", "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to stop replica: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// StartReplica sends a request to start a previously stopped replica.
func (c *Client) StartReplica(replicaURL string) error {
	resp, err := c.client.Post(replicaURL+"/start", "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to start replica: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// CheckHealth queries all replicas and displays their status.
// Shows ID, status (healthy/stopped/unreachable), consistency model, and delay.
func (c *Client) CheckHealth() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                           REPLICA HEALTH STATUS                          ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════╣")
	
	for _, replica := range c.replicas {
		resp, err := c.client.Get(replica + "/health")
		if err != nil {
			fmt.Printf("║  %-40s │ %-28s ║\n", replica, "❌ UNREACHABLE")
			continue
		}
		defer resp.Body.Close()
		
		body, _ := io.ReadAll(resp.Body)
		var health map[string]interface{}
		json.Unmarshal(body, &health)
		
		status := "✅ healthy"
		if s, ok := health["status"].(string); ok && s != "healthy" {
			status = "🛑 " + s
		}
		
		id := ""
		model := ""
		delay := float64(0)
		
		if v, ok := health["id"].(string); ok {
			id = v
		}
		if v, ok := health["model"].(string); ok {
			model = v
		}
		if v, ok := health["delay"].(float64); ok {
			delay = v
		}
		
		fmt.Printf("║  %-20s │ %-10s │ %-8s │ %4.0fms delay ║\n",
			id, status, model, delay)
	}
	
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════╝")
}

// ============================================================================
// DISPLAY HELPERS
// ============================================================================

// printPUTResult formats and displays a PUT operation result
func printPUTResult(resp *PUTResponse, duration time.Duration) {
	if resp.Success {
		fmt.Printf("✅ PUT SUCCESS (took %v)\n", duration)
		fmt.Printf("   Key:      (stored)\n")
		fmt.Printf("   Version:  %d\n", resp.Version)
		fmt.Printf("   Updated:  %s\n", resp.UpdatedBy)
		fmt.Printf("   Message:  %s\n", resp.Message)
	} else {
		fmt.Printf("❌ PUT FAILED (took %v)\n", duration)
		fmt.Printf("   Message:  %s\n", resp.Message)
	}
}

// printGETResult formats and displays a GET operation result
func printGETResult(resp *GETResponse, duration time.Duration) {
	fmt.Printf("📖 GET RESULT (took %v)\n", duration)
	if resp.Found {
		fmt.Printf("   Key:      %s\n", resp.Key)
		fmt.Printf("   Value:    %s\n", resp.Value)
		fmt.Printf("   Version:  %d\n", resp.Version)
		fmt.Printf("   Updated:  %s\n", resp.UpdatedBy)
	} else {
		fmt.Printf("   Key '%s' NOT FOUND\n", resp.Key)
	}
}

// printGetAllResult formats and displays results from all replicas
func printGetAllResult(key string, results map[string]*GETResponse) {
	fmt.Printf("\n📊 GETALL RESULTS for key '%s':\n", key)
	fmt.Println("─────────────────────────────────────────────────────────────")
	
	// Check for inconsistency
	values := make(map[string]int)
	for _, resp := range results {
		if resp.Found {
			values[resp.Value]++
		}
	}
	
	for replica, resp := range results {
		if resp.Found {
			fmt.Printf("  %-30s │ value='%s' │ v%d │ by %s\n",
				replica, resp.Value, resp.Version, resp.UpdatedBy)
		} else {
			fmt.Printf("  %-30s │ NOT FOUND\n", replica)
		}
	}
	
	fmt.Println("─────────────────────────────────────────────────────────────")
	
	// Detect and report inconsistency
	if len(values) > 1 {
		fmt.Println("⚠️  INCONSISTENCY DETECTED: Different replicas have different values!")
		fmt.Println("   Values found:")
		for val, count := range values {
			fmt.Printf("     '%s': %d replica(s)\n", val, count)
		}
	} else if len(values) == 1 {
		fmt.Println("✅ All replicas are CONSISTENT")
	} else {
		fmt.Println("📭 Key not found on any replica")
	}
}

// ============================================================================
// COMMAND-LINE MODE
// ============================================================================

// runCommandMode processes a single command from command-line arguments.
// This mode is useful for scripting and automated testing.
func runCommandMode(client *Client, replicas []string, osArgs []string) {
	if len(osArgs) < 2 {
		fmt.Println("Usage: client <command> [args...]")
		fmt.Println("Commands: put, get, getall, health, stop, start")
		return
	}

	command := osArgs[1]
	
	switch command {
	case "put":
		if len(osArgs) != 5 {
			fmt.Println("Usage: client put <replica-index> <key> <value>")
			fmt.Println("Example: client put 1 mykey myvalue")
			return
		}
		replicaIdx := parseReplicaIndex(osArgs[2], len(replicas))
		if replicaIdx < 0 {
			return
		}
		
		resp, duration, err := client.PUT(replicas[replicaIdx], osArgs[3], osArgs[4])
		if err != nil {
			log.Printf("Error: %v", err)
			return
		}
		printPUTResult(resp, duration)

	case "get":
		if len(osArgs) != 4 {
			fmt.Println("Usage: client get <replica-index> <key>")
			fmt.Println("Example: client get 1 mykey")
			return
		}
		replicaIdx := parseReplicaIndex(osArgs[2], len(replicas))
		if replicaIdx < 0 {
			return
		}
		
		resp, duration, err := client.GET(replicas[replicaIdx], osArgs[3])
		if err != nil {
			log.Printf("Error: %v", err)
			return
		}
		printGETResult(resp, duration)

	case "getall":
		if len(osArgs) != 3 {
			fmt.Println("Usage: client getall <key>")
			return
		}
		results := client.GetAllReplicas(osArgs[2])
		printGetAllResult(osArgs[2], results)

	case "health":
		client.CheckHealth()

	case "stop":
		if len(osArgs) != 3 {
			fmt.Println("Usage: client stop <replica-index>")
			return
		}
		replicaIdx := parseReplicaIndex(osArgs[2], len(replicas))
		if replicaIdx < 0 {
			return
		}
		if err := client.StopReplica(replicas[replicaIdx]); err != nil {
			log.Printf("Error stopping replica: %v", err)
		} else {
			fmt.Printf("🛑 Replica %d stopped\n", replicaIdx+1)
		}

	case "start":
		if len(osArgs) != 3 {
			fmt.Println("Usage: client start <replica-index>")
			return
		}
		replicaIdx := parseReplicaIndex(osArgs[2], len(replicas))
		if replicaIdx < 0 {
			return
		}
		if err := client.StartReplica(replicas[replicaIdx]); err != nil {
			log.Printf("Error starting replica: %v", err)
		} else {
			fmt.Printf("🟢 Replica %d started\n", replicaIdx+1)
		}

	default:
		fmt.Printf("Unknown command: '%s'\n", command)
		fmt.Println("Available commands: put, get, getall, health, stop, start")
	}
}

// ============================================================================
// INTERACTIVE MODE
// ============================================================================

// runInteractiveMode starts the interactive command-line interface.
// The user can type commands repeatedly until they type "quit" or "exit".
func runInteractiveMode(client *Client, replicas []string) {
	printWelcomeMessage(replicas)
	
	scanner := bufio.NewScanner(os.Stdin)
	
	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}
		
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		
		if input == "quit" || input == "exit" {
			fmt.Println("👋 Goodbye!")
			break
		}
		
		processInteractiveCommand(client, replicas, input)
	}
}

// processInteractiveCommand parses and executes a single command in interactive mode
func processInteractiveCommand(client *Client, replicas []string, input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "help", "?":
		printHelpMessage()
		
	case "put":
		if len(parts) != 4 {
			fmt.Println("❌ Usage: put <replica> <key> <value>")
			fmt.Println("   Example: put 1 username alice")
			return
		}
		replicaIdx := parseReplicaIndex(parts[1], len(replicas))
		if replicaIdx < 0 {
			return
		}
		
		resp, duration, err := client.PUT(replicas[replicaIdx], parts[2], parts[3])
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		printPUTResult(resp, duration)

	case "get":
		if len(parts) != 3 {
			fmt.Println("❌ Usage: get <replica> <key>")
			fmt.Println("   Example: get 1 username")
			return
		}
		replicaIdx := parseReplicaIndex(parts[1], len(replicas))
		if replicaIdx < 0 {
			return
		}
		
		resp, duration, err := client.GET(replicas[replicaIdx], parts[2])
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		printGETResult(resp, duration)

	case "getall":
		if len(parts) != 2 {
			fmt.Println("❌ Usage: getall <key>")
			return
		}
		results := client.GetAllReplicas(parts[1])
		printGetAllResult(parts[1], results)

	case "health", "status":
		client.CheckHealth()

	case "stop":
		if len(parts) != 2 {
			fmt.Println("❌ Usage: stop <replica>")
			fmt.Println("   Example: stop 3")
			return
		}
		replicaIdx := parseReplicaIndex(parts[1], len(replicas))
		if replicaIdx < 0 {
			return
		}
		if err := client.StopReplica(replicas[replicaIdx]); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Printf("🛑 Replica %d has been stopped\n", replicaIdx+1)
		}

	case "start":
		if len(parts) != 2 {
			fmt.Println("❌ Usage: start <replica>")
			fmt.Println("   Example: start 3")
			return
		}
		replicaIdx := parseReplicaIndex(parts[1], len(replicas))
		if replicaIdx < 0 {
			return
		}
		if err := client.StartReplica(replicas[replicaIdx]); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Printf("🟢 Replica %d has been started\n", replicaIdx+1)
		}

	case "demo":
		runDemo(client, replicas)

	case "scenario1":
		runScenario1(client, replicas)
		
	case "scenario2":
		runScenario2(client, replicas)
		
	case "scenario3":
		runScenario3(client, replicas)
		
	case "scenario4":
		runScenario4(client, replicas)

	default:
		fmt.Printf("❌ Unknown command: '%s'\n", parts[0])
		fmt.Println("Type 'help' for available commands")
	}
}

// ============================================================================
// BUILT-IN TEST SCENARIOS
// ============================================================================

// runScenario1 demonstrates temporary inconsistency in eventual consistency mode
func runScenario1(client *Client, replicas []string) {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  SCENARIO 1: TEMPORARY INCONSISTENCY OBSERVATION                         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("This scenario demonstrates that in eventual consistency mode,")
	fmt.Println("a read immediately after a write may return stale data.")
	fmt.Println()
	
	// Step 1: Write to Replica 1
	fmt.Println("Step 1: Writing x=10 to Replica 1...")
	resp, dur, err := client.PUT(replicas[0], "x", "10")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	printPUTResult(resp, dur)
	
	// Step 2: Immediately read from Replica 2
	fmt.Println("\nStep 2: Immediately reading x from Replica 2...")
	resp2, dur2, err2 := client.GET(replicas[1], "x")
	if err2 != nil {
		fmt.Printf("Error: %v\n", err2)
		return
	}
	printGETResult(resp2, dur2)
	
	if !resp2.Found || resp2.Value != "10" {
		fmt.Println("📝 OBSERVATION: Replica 2 returned stale/missing data!")
		fmt.Println("   This is because replication from Replica 1 hasn't completed yet.")
	}
	
	// Step 3: Wait and read again
	fmt.Println("\nStep 3: Waiting 3 seconds for replication to complete...")
	time.Sleep(3 * time.Second)
	
	fmt.Println("Step 4: Reading x from Replica 2 again...")
	resp3, dur3, err3 := client.GET(replicas[1], "x")
	if err3 != nil {
		fmt.Printf("Error: %v\n", err3)
		return
	}
	printGETResult(resp3, dur3)
	
	if resp3.Found && resp3.Value == "10" {
		fmt.Println("📝 OBSERVATION: Replica 2 now has the correct value!")
		fmt.Println("   The system has EVENTUALLY become consistent.")
	}
	
	// Show all replicas
	fmt.Println("\nFinal state across all replicas:")
	results := client.GetAllReplicas("x")
	printGetAllResult("x", results)
}

// runScenario2 demonstrates replica failure behavior
func runScenario2(client *Client, replicas []string) {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  SCENARIO 2: REPLICA FAILURE BEHAVIOR                                    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	// Step 1: Stop Replica 3
	fmt.Println("Step 1: Stopping Replica 3...")
	if err := client.StopReplica(replicas[2]); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println("Replica 3 stopped.")
	
	// Step 2: Write a value
	fmt.Println("\nStep 2: Writing y=20 to Replica 1...")
	resp, dur, err := client.PUT(replicas[0], "y", "20")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		printPUTResult(resp, dur)
		
		if resp.Success {
			fmt.Println("📝 OBSERVATION: Write succeeded even with one replica down.")
			fmt.Println("   (If using strong consistency, this requires majority still available)")
		} else {
			fmt.Println("📝 OBSERVATION: Write failed - strong consistency requires majority.")
		}
	}
	
	// Step 3: Check remaining replicas
	fmt.Println("\nStep 3: Checking remaining replicas...")
	results := client.GetAllReplicas("y")
	printGetAllResult("y", results)
	
	// Step 4: Restart Replica 3
	fmt.Println("\nStep 4: Restarting Replica 3...")
	if err := client.StartReplica(replicas[2]); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println("Replica 3 restarted.")
	
	time.Sleep(1 * time.Second)
	
	// Step 5: Check Replica 3
	fmt.Println("\nStep 5: Checking Replica 3 after restart...")
	resp3, _, _ := client.GET(replicas[2], "y")
	if resp3.Found {
		fmt.Printf("Replica 3 has value: %s (v%d)\n", resp3.Value, resp3.Version)
		fmt.Println("📝 OBSERVATION: Replica 3 may have missed the update while it was down.")
	} else {
		fmt.Println("Replica 3 does NOT have the value - it missed the update.")
	}
}

// runScenario3 demonstrates concurrent conflict resolution
func runScenario3(client *Client, replicas []string) {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  SCENARIO 3: CONCURRENT CONFLICT RESOLUTION                              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("This scenario demonstrates conflict resolution when two replicas")
	fmt.Println("receive different values for the same key simultaneously.")
	fmt.Println()
	
	// Step 1: Write different values to different replicas
	fmt.Println("Step 1: Writing z=100 to Replica 1...")
	resp1, _, _ := client.PUT(replicas[0], "z", "100")
	printPUTResult(resp1, time.Duration(0))
	
	fmt.Println("\nStep 2: Immediately writing z=200 to Replica 2...")
	resp2, _, _ := client.PUT(replicas[1], "z", "200")
	printPUTResult(resp2, time.Duration(0))
	
	// Step 3: Wait for replication and conflict resolution
	fmt.Println("\nStep 3: Waiting for replication and conflict resolution...")
	time.Sleep(3 * time.Second)
	
	// Step 4: Check all replicas
	fmt.Println("\nStep 4: Checking final state across all replicas...")
	results := client.GetAllReplicas("z")
	printGetAllResult("z", results)
	
	fmt.Println("\n📝 OBSERVATION: The conflict was resolved using Last-Write-Wins (LWW).")
	fmt.Println("   The value with the higher timestamp won.")
	fmt.Println("   All replicas should have converged to the same value.")
}

// runScenario4 demonstrates network delay impact
func runScenario4(client *Client, replicas []string) {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  SCENARIO 4: NETWORK DELAY IMPACT                                        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("This scenario demonstrates how network delay affects convergence.")
	fmt.Println("(Note: Configure network_delay in replica config files before running)")
	fmt.Println()
	
	// Test 1: Write and measure convergence
	fmt.Println("Test: Writing w=500 to Replica 1...")
	start := time.Now()
	resp, _, err := client.PUT(replicas[0], "w", "500")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	printPUTResult(resp, time.Since(start))
	
	// Poll until all replicas converge or timeout
	fmt.Println("\nMonitoring convergence...")
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	converged := false
	var convergeTime time.Duration
	
	for !converged {
		select {
		case <-timeout:
			fmt.Println("⏰ Timeout: Replicas did not converge within 10 seconds")
			goto showResults
		case <-ticker.C:
			results := client.GetAllReplicas("w")
			allMatch := true
			var firstValue string
			firstSet := false
			
			for _, resp := range results {
				if !resp.Found {
					allMatch = false
					break
				}
				if !firstSet {
					firstValue = resp.Value
					firstSet = true
				} else if resp.Value != firstValue {
					allMatch = false
					break
				}
			}
			
			if allMatch && firstSet {
				converged = true
				convergeTime = time.Since(start)
				fmt.Printf("\n✅ All replicas converged after %v\n", convergeTime)
			} else {
				fmt.Print(".")
			}
		}
	}
	
showResults:
	results := client.GetAllReplicas("w")
	printGetAllResult("w", results)
	
	fmt.Printf("\n📊 Metrics:\n")
	fmt.Printf("   Convergence time: %v\n", convergeTime)
}

// runDemo runs a quick demonstration of all features
func runDemo(client *Client, replicas []string) {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  QUICK DEMONSTRATION                                                     ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════╝")
	
	// Health check
	client.CheckHealth()
	
	// Basic operations
	fmt.Println("\n--- Basic PUT and GET ---")
	client.PUT(replicas[0], "demo_key", "demo_value")
	time.Sleep(100 * time.Millisecond)
	results := client.GetAllReplicas("demo_key")
	printGetAllResult("demo_key", results)
	
	// Inconsistency demo
	fmt.Println("\n--- Inconsistency Demo ---")
	client.PUT(replicas[0], "fast_key", "value1")
	fmt.Println("(Reading immediately from another replica...)")
	time.Sleep(50 * time.Millisecond)
	resp, _, _ := client.GET(replicas[1], "fast_key")
	if !resp.Found {
		fmt.Println("📝 Replica 2 doesn't have the value yet - demonstrating eventual consistency!")
	}
	time.Sleep(2 * time.Second)
	results = client.GetAllReplicas("fast_key")
	printGetAllResult("fast_key", results)
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// parseReplicaIndex converts a string index to an integer and validates it
func parseReplicaIndex(input string, numReplicas int) int {
	idx := 0
	fmt.Sscanf(input, "%d", &idx)
	idx-- // Convert from 1-based to 0-based
	
	if idx < 0 || idx >= numReplicas {
		fmt.Printf("❌ Invalid replica index. Use 1-%d\n", numReplicas)
		return -1
	}
	return idx
}

// printWelcomeMessage displays the initial welcome screen
func printWelcomeMessage(replicas []string) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║           DISTRIBUTED KEY-VALUE STORE CLIENT                             ║")
	fmt.Println("║           University of Tehran - Distributed Computing                   ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Connected to %d replicas:                                              ║\n", len(replicas))
	for i, r := range replicas {
		fmt.Printf("║    Replica %d: %-52s ║\n", i+1, r)
	}
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Type 'help' for available commands                                      ║")
	fmt.Println("║  Type 'demo' for a quick demonstration                                   ║")
	fmt.Println("║  Type 'scenario1' to run test scenario 1                                 ║")
	fmt.Println("║  Type 'quit' to exit                                                     ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════╝")
}

// printHelpMessage displays available commands
func printHelpMessage() {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  AVAILABLE COMMANDS                                                      ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════╣")
	fmt.Println("║                                                                          ║")
	fmt.Println("║  DATA OPERATIONS:                                                        ║")
	fmt.Println("║    put <replica> <key> <value>  - Store a key-value pair                ║")
	fmt.Println("║    get <replica> <key>          - Retrieve a value                      ║")
	fmt.Println("║    getall <key>                 - Get value from ALL replicas           ║")
	fmt.Println("║                                                                          ║")
	fmt.Println("║  REPLICA MANAGEMENT:                                                     ║")
	fmt.Println("║    health                       - Check status of all replicas          ║")
	fmt.Println("║    stop <replica>               - Stop a replica (simulate failure)     ║")
	fmt.Println("║    start <replica>              - Restart a stopped replica             ║")
	fmt.Println("║                                                                          ║")
	fmt.Println("║  TEST SCENARIOS:                                                         ║")
	fmt.Println("║    scenario1                    - Temporary inconsistency test          ║")
	fmt.Println("║    scenario2                    - Replica failure test                  ║")
	fmt.Println("║    scenario3                    - Concurrent conflict test              ║")
	fmt.Println("║    scenario4                    - Network delay test                    ║")
	fmt.Println("║    demo                         - Quick demonstration                   ║")
	fmt.Println("║                                                                          ║")
	fmt.Println("║  OTHER:                                                                  ║")
	fmt.Println("║    help                         - Show this help message                ║")
	fmt.Println("║    quit / exit                  - Exit the client                       ║")
	fmt.Println("║                                                                          ║")
	fmt.Println("║  Note: <replica> is 1, 2, or 3 (not 0-based)                            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════╝")
}

// ============================================================================
// MAIN FUNCTION
// ============================================================================

func main() {
	// Default replica URLs - modify these if your replicas are on different ports/hosts
	replicas := []string{
		"http://localhost:8001",
		"http://localhost:8002",
		"http://localhost:8003",
	}

	// Allow custom replicas via environment variable
	if envReplicas := os.Getenv("REPLICAS"); envReplicas != "" {
		replicas = strings.Split(envReplicas, ",")
	}

	client := NewClient(replicas)

	// If command-line arguments are provided, run in command mode
	// Otherwise, run in interactive mode
	if len(os.Args) > 1 {
		runCommandMode(client, replicas, os.Args)
	} else {
		runInteractiveMode(client, replicas)
	}
}