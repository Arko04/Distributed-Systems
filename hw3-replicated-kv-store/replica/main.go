// ============================================================================
// REPLICA SERVER - Distributed Replicated Key-Value Store
// ============================================================================
// This file implements a replica server that is part of a distributed
// key-value store system. Each replica is an independent HTTP server
// that maintains its own copy of the data and synchronizes with other
// replicas through replication messages.
//
// Features:
//   - Independent HTTP server for each replica
//   - In-memory key-value storage with versioning
//   - Two consistency models: Eventual and Strong
//   - Conflict resolution using Last-Write-Wins (LWW)
//   - Simulated network delays for testing
//   - Start/stop functionality for failure simulation
//   - Health checking and debugging endpoints
// ============================================================================

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// ============================================================================
// CONFIGURATION STRUCTURES
// ============================================================================

// Config represents the configuration for a replica loaded from a JSON file.
// This structure defines how the replica identifies itself, where it listens,
// which peers it communicates with, and its behavioral parameters.
type Config struct {
	// ID is the unique identifier for this replica (e.g., "replica1", "replica2")
	ID string `json:"id"`
	
	// Host is the network interface to bind to (e.g., "localhost", "0.0.0.0")
	Host string `json:"host"`
	
	// Port is the TCP port to listen on (e.g., 8001, 8002, 8003)
	Port int `json:"port"`
	
	// Peers is a list of URLs for the other replicas in the system.
	// This replica will send replication messages to these peers.
	Peers []string `json:"peers"`
	
	// ConsistencyModel determines the replication strategy:
	//   "eventual" - Asynchronous replication (AP system)
	//   "strong"   - Synchronous replication with majority (CP system)
	ConsistencyModel string `json:"consistency_model"`
	
	// NetworkDelay is the artificial delay (in milliseconds) added to
	// replication message processing. Used for testing the impact of
	// network latency on system behavior.
	NetworkDelay int `json:"network_delay"`
}

// ============================================================================
// DATA STRUCTURES
// ============================================================================

// DataEntry represents a single key-value pair stored in the system.
// Each entry includes versioning information for conflict detection
// and resolution, along with metadata about the last update.
type DataEntry struct {
	// Key is the unique identifier for this data entry
	Key string `json:"key"`
	
	// Value is the stored data associated with the key
	Value string `json:"value"`
	
	// Version is a monotonically increasing number that tracks updates.
	// Higher version numbers indicate more recent updates.
	// Used to detect stale updates and conflicts.
	Version int `json:"version"`
	
	// UpdatedBy identifies which replica last modified this entry.
	// Useful for debugging and tracing update origins.
	UpdatedBy string `json:"updated_by"`
	
	// Timestamp is the time when this entry was last updated (in nanoseconds).
	// Used for Last-Write-Wins conflict resolution when versions are equal.
	Timestamp int64 `json:"timestamp"`
}

// ============================================================================
// REPLICA STRUCTURE
// ============================================================================

// Replica represents a single replica node in the distributed system.
// It encapsulates the server configuration, the local data store,
// concurrency controls, and the HTTP client for peer communication.
type Replica struct {
	// config contains all configuration parameters loaded from JSON
	config Config
	
	// data is the in-memory key-value store.
	// Maps key strings to their corresponding DataEntry.
	// Protected by mu (RWMutex) for concurrent access.
	data map[string]DataEntry
	
	// mu is a read-write mutex that protects the data map.
	// RLock/RUnlock for reads (GET), Lock/Unlock for writes (PUT, replicate).
	// This ensures thread-safe access when handling concurrent HTTP requests.
	mu sync.RWMutex
	
	// client is an HTTP client used for sending replication messages to peers.
	// Configured with a timeout to prevent hanging on unresponsive peers.
	client *http.Client
	
	// isRunning indicates whether this replica is currently accepting requests.
	// When false, the replica returns 503 Service Unavailable.
	// Used for simulating replica failures in testing.
	isRunning bool
}

// ============================================================================
// REQUEST/RESPONSE STRUCTURES
// ============================================================================

// PUTRequest is the expected JSON body for PUT operations.
// Clients send this structure when storing a key-value pair.
type PUTRequest struct {
	Key   string `json:"key"`   // The key to store
	Value string `json:"value"` // The value to associate with the key
}

// PUTResponse is the JSON response returned after a PUT operation.
// It indicates whether the operation succeeded and provides version info.
type PUTResponse struct {
	Success   bool   `json:"success"`    // Whether the PUT was successful
	Message   string `json:"message"`    // Human-readable status message
	Version   int    `json:"version"`    // The version number of the stored entry
	UpdatedBy string `json:"updated_by"` // Which replica performed the update
}

// GETResponse is the JSON response returned after a GET operation.
// It contains the requested data (if found) along with metadata.
type GETResponse struct {
	Key       string `json:"key"`        // The requested key
	Value     string `json:"value"`      // The stored value (empty if not found)
	Version   int    `json:"version"`    // The version of the stored entry
	UpdatedBy string `json:"updated_by"` // Which replica last updated this entry
	Timestamp int64  `json:"timestamp"`  // When the entry was last updated
	Found     bool   `json:"found"`      // Whether the key was found
}

// ReplicationRequest is the JSON body sent between replicas during replication.
// It contains the data entry to replicate plus metadata about the origin.
type ReplicationRequest struct {
	// Entry is the data entry being replicated
	Entry DataEntry `json:"entry"`
	
	// OriginID identifies which replica initiated this replication
	OriginID string `json:"origin_id"`
	
	// Timestamp is when the replication message was created
	Timestamp int64 `json:"timestamp"`
}

// ReplicationResponse is the JSON response returned after processing replication.
type ReplicationResponse struct {
	Success bool   `json:"success"` // Whether replication was accepted
	Message string `json:"message"` // Status message
}

// ============================================================================
// CONSTRUCTOR
// ============================================================================

// NewReplica creates a new Replica instance with the given configuration.
// It initializes the data store, HTTP client, and sets the replica as running.
func NewReplica(config Config) *Replica {
	return &Replica{
		config: config,
		data:   make(map[string]DataEntry),
		client: &http.Client{
			Timeout: 10 * time.Second, // 10-second timeout for peer communication
		},
		isRunning: true, // Start in running state
	}
}

// ============================================================================
// HTTP HANDLERS
// ============================================================================

// handlePUT processes a PUT request from a client.
// It stores the key-value pair locally, then replicates to peers
// either synchronously (strong consistency) or asynchronously (eventual).
//
// Endpoint: POST /put
// Body: {"key": "string", "value": "string"}
// Response: {"success": bool, "message": "string", "version": int, "updated_by": "string"}
func (r *Replica) handlePUT(w http.ResponseWriter, req *http.Request) {
	// Check if replica is running (for failure simulation)
	if !r.isRunning {
		http.Error(w, "Replica is stopped", http.StatusServiceUnavailable)
		return
	}

	// Read and parse the request body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var putReq PUTRequest
	if err := json.Unmarshal(body, &putReq); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate that key is not empty
	if putReq.Key == "" {
		http.Error(w, "Key cannot be empty", http.StatusBadRequest)
		return
	}

	// CRITICAL SECTION: Update local data store
	// Using write lock to ensure atomic read-modify-write
	r.mu.Lock()
	
	// Determine the new version number
	currentEntry, exists := r.data[putReq.Key]
	newVersion := 1
	if exists {
		// Key already exists, increment the version
		newVersion = currentEntry.Version + 1
	}
	
	// Create the new data entry with current timestamp
	newEntry := DataEntry{
		Key:       putReq.Key,
		Value:     putReq.Value,
		Version:   newVersion,
		UpdatedBy: r.config.ID,
		Timestamp: time.Now().UnixNano(), // Nanosecond precision for LWW
	}
	
	// Store in local data map
	r.data[putReq.Key] = newEntry
	r.mu.Unlock()
	// END CRITICAL SECTION

	// Handle replication based on consistency model
	if r.config.ConsistencyModel == "strong" {
		// STRONG CONSISTENCY: Synchronous replication to majority
		success := r.replicateStrong(newEntry)
		if !success {
			// Failed to achieve majority consensus
			// In a production system, you might rollback the local write here
			response := PUTResponse{
				Success: false,
				Message: "Failed to achieve majority consensus. Need at least " +
					fmt.Sprintf("%d replicas.", (len(r.config.Peers)+1)/2+1),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK) // Still 200, but success=false
			json.NewEncoder(w).Encode(response)
			return
		}
	} else {
		// EVENTUAL CONSISTENCY: Asynchronous replication
		// Launch goroutine to replicate in background
		go r.replicateEventual(newEntry)
	}

	// Return success response
	response := PUTResponse{
		Success:   true,
		Message:   "Value stored successfully",
		Version:   newVersion,
		UpdatedBy: r.config.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	
	// Log the operation for debugging
	log.Printf("[%s] PUT %s=%s (version %d, model: %s)",
		r.config.ID, putReq.Key, putReq.Value, newVersion, r.config.ConsistencyModel)
}

// handleGET processes a GET request from a client.
// It retrieves the value for the specified key from the local data store.
//
// Endpoint: GET /get?key=<key>
// Response: {"key": "string", "value": "string", "version": int, "found": bool, ...}
func (r *Replica) handleGET(w http.ResponseWriter, req *http.Request) {
	// Check if replica is running
	if !r.isRunning {
		http.Error(w, "Replica is stopped", http.StatusServiceUnavailable)
		return
	}

	// Extract key from query parameters
	key := req.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Key parameter is required", http.StatusBadRequest)
		return
	}

	// Read from local data store with read lock
	r.mu.RLock()
	entry, exists := r.data[key]
	r.mu.RUnlock()

	// Build response based on whether key was found
	var response GETResponse
	if exists {
		response = GETResponse{
			Key:       entry.Key,
			Value:     entry.Value,
			Version:   entry.Version,
			UpdatedBy: entry.UpdatedBy,
			Timestamp: entry.Timestamp,
			Found:     true,
		}
	} else {
		response = GETResponse{
			Key:   key,
			Found: false,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	
	// Log the operation
	if exists {
		log.Printf("[%s] GET %s -> %s (version %d)",
			r.config.ID, key, entry.Value, entry.Version)
	} else {
		log.Printf("[%s] GET %s -> NOT FOUND", r.config.ID, key)
	}
}

// handleReplicate processes a replication request from a peer replica.
// This is the internal endpoint used for data synchronization between replicas.
// It applies version checking and conflict resolution before accepting updates.
//
// Endpoint: POST /replicate
// Body: {"entry": {...}, "origin_id": "string", "timestamp": int64}
// Response: {"success": bool, "message": "string"}
func (r *Replica) handleReplicate(w http.ResponseWriter, req *http.Request) {
	// Check if replica is running
	if !r.isRunning {
		http.Error(w, "Replica is stopped", http.StatusServiceUnavailable)
		return
	}

	// Read and parse the replication request
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var repReq ReplicationRequest
	if err := json.Unmarshal(body, &repReq); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Apply artificial network delay for testing
	// This simulates network latency between replicas
	if r.config.NetworkDelay > 0 {
		time.Sleep(time.Duration(r.config.NetworkDelay) * time.Millisecond)
	}

	// CRITICAL SECTION: Apply replication with version checking
	r.mu.Lock()
	
	currentEntry, exists := r.data[repReq.Entry.Key]
	shouldUpdate := false
	action := ""

	if !exists {
		// Key doesn't exist locally - always accept new entry
		shouldUpdate = true
		action = "CREATED"
	} else {
		// Key exists - compare versions to decide whether to accept
		if repReq.Entry.Version > currentEntry.Version {
			// Received version is newer - accept the update
			shouldUpdate = true
			action = "UPDATED"
		} else if repReq.Entry.Version == currentEntry.Version {
			// SAME VERSION = CONFLICT DETECTED
			// Two replicas updated the same key concurrently
			// Use Last-Write-Wins (LWW) based on timestamp
			if repReq.Entry.Timestamp > currentEntry.Timestamp {
				// Received entry has higher timestamp - accept it
				shouldUpdate = true
				action = "CONFLICT_RESOLVED_LWW"
				log.Printf("[%s] ⚠️  CONFLICT on key '%s': values '%s' vs '%s' - "+
					"resolved using LWW (newer timestamp wins)",
					r.config.ID, repReq.Entry.Key,
					currentEntry.Value, repReq.Entry.Value)
			} else if repReq.Entry.Timestamp == currentEntry.Timestamp {
				// Extremely rare: same version AND same timestamp
				// Use replica ID as tiebreaker
				if repReq.OriginID > r.config.ID {
					shouldUpdate = true
					action = "CONFLICT_RESOLVED_REPLICA_ID"
				} else {
					action = "CONFLICT_KEPT_CURRENT_REPLICA_ID"
				}
				log.Printf("[%s] ⚠️  CONFLICT on key '%s': same version and timestamp - "+
					"resolved using replica ID tiebreaker",
					r.config.ID, repReq.Entry.Key)
			} else {
				// Current entry has higher timestamp - reject the incoming update
				action = "REJECTED_CONFLICT"
				log.Printf("[%s] ⚠️  CONFLICT on key '%s': keeping current version "+
					"(our timestamp %d > received %d)",
					r.config.ID, repReq.Entry.Key,
					currentEntry.Timestamp, repReq.Entry.Timestamp)
			}
		} else {
			// Received version is older - reject (stale update)
			action = "REJECTED_STALE"
		}
	}

	// Apply the update if decided
	if shouldUpdate {
		r.data[repReq.Entry.Key] = repReq.Entry
	}
	r.mu.Unlock()
	// END CRITICAL SECTION

	// Log the replication action
	switch action {
	case "CREATED":
		log.Printf("[%s] 📥 REPLICATED: created key '%s' = '%s' (v%d) from %s",
			r.config.ID, repReq.Entry.Key, repReq.Entry.Value,
			repReq.Entry.Version, repReq.OriginID)
	case "UPDATED":
		log.Printf("[%s] 📥 REPLICATED: updated key '%s' = '%s' (v%d) from %s",
			r.config.ID, repReq.Entry.Key, repReq.Entry.Value,
			repReq.Entry.Version, repReq.OriginID)
	case "CONFLICT_RESOLVED_LWW", "CONFLICT_RESOLVED_REPLICA_ID":
		log.Printf("[%s] 📥 REPLICATED: conflict resolved for '%s', "+
			"now = '%s' (v%d) from %s",
			r.config.ID, repReq.Entry.Key, repReq.Entry.Value,
			repReq.Entry.Version, repReq.OriginID)
	case "REJECTED_STALE":
		log.Printf("[%s] ❌ REJECTED: stale update for '%s' "+
			"(received v%d <= current v%d)",
			r.config.ID, repReq.Entry.Key,
			repReq.Entry.Version, currentEntry.Version)
	case "REJECTED_CONFLICT":
		log.Printf("[%s] ❌ REJECTED: conflict on '%s', keeping current value",
			r.config.ID, repReq.Entry.Key)
	case "CONFLICT_KEPT_CURRENT_REPLICA_ID":
		log.Printf("[%s] ❌ REJECTED: conflict on '%s', keeping current (replica ID tiebreaker)",
			r.config.ID, repReq.Entry.Key)
	}

	// Return success response
	response := ReplicationResponse{
		Success: true,
		Message: fmt.Sprintf("Replication processed: %s", action),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHealth provides a health check endpoint.
// Returns the replica's ID, status, and consistency model.
//
// Endpoint: GET /health
// Response: {"id": "string", "status": "string", "model": "string"}
func (r *Replica) handleHealth(w http.ResponseWriter, req *http.Request) {
	status := "healthy"
	if !r.isRunning {
		status = "stopped"
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     r.config.ID,
		"status": status,
		"model":  r.config.ConsistencyModel,
		"delay":  r.config.NetworkDelay,
		"peers":  r.config.Peers,
	})
}

// handleStop allows stopping the replica to simulate failure.
// When stopped, the replica returns 503 for PUT, GET, and REPLICATE requests.
//
// Endpoint: POST /stop
// Response: {"status": "stopped", "message": "string"}
func (r *Replica) handleStop(w http.ResponseWriter, req *http.Request) {
	r.isRunning = false
	log.Printf("[%s] 🛑 Replica STOPPED (simulating failure)", r.config.ID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "stopped",
		"message": fmt.Sprintf("Replica %s has been stopped", r.config.ID),
	})
}

// handleStart allows restarting a previously stopped replica.
//
// Endpoint: POST /start
// Response: {"status": "started", "message": "string"}
func (r *Replica) handleStart(w http.ResponseWriter, req *http.Request) {
	r.isRunning = true
	log.Printf("[%s] 🟢 Replica STARTED (resuming operation)", r.config.ID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "started",
		"message": fmt.Sprintf("Replica %s has been started", r.config.ID),
	})
}

// handleData is a debug endpoint that returns all stored data.
// Useful for inspecting the state of a replica during testing.
//
// Endpoint: GET /data
// Response: {"key1": {...}, "key2": {...}, ...}
func (r *Replica) handleData(w http.ResponseWriter, req *http.Request) {
	r.mu.RLock()
	// Create a copy of the data to avoid holding lock during JSON encoding
	dataCopy := make(map[string]DataEntry, len(r.data))
	for k, v := range r.data {
		dataCopy[k] = v
	}
	r.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dataCopy)
}

// ============================================================================
// REPLICATION METHODS
// ============================================================================

// replicateEventual performs asynchronous replication to all peers.
// This is used in EVENTUAL CONSISTENCY mode.
// It spawns goroutines for each peer and returns immediately.
//
// Characteristics:
//   - Non-blocking: returns before peers acknowledge
//   - Best-effort: failures are logged but don't affect the client
//   - Low latency: client gets response after local write only
func (r *Replica) replicateEventual(entry DataEntry) {
	repReq := ReplicationRequest{
		Entry:     entry,
		OriginID:  r.config.ID,
		Timestamp: time.Now().UnixNano(),
	}

	jsonData, err := json.Marshal(repReq)
	if err != nil {
		log.Printf("[%s] Error marshaling replication request: %v", r.config.ID, err)
		return
	}

	// Send to all peers concurrently
	var wg sync.WaitGroup
	for _, peer := range r.config.Peers {
		wg.Add(1)
		go func(peerURL string) {
			defer wg.Done()
			r.sendReplication(peerURL+"/replicate", jsonData)
		}(peer)
	}
	
	// Wait for all goroutines to complete (they may take time due to network)
	// Note: We don't report success/failure to the original client
	wg.Wait()
	
	log.Printf("[%s] 📤 Eventual replication complete for key '%s' (v%d)",
		r.config.ID, entry.Key, entry.Version)
}

// replicateStrong performs synchronous replication requiring majority acknowledgment.
// This is used in STRONG CONSISTENCY mode.
// It blocks until either a majority of replicas acknowledge or all peers respond.
//
// Characteristics:
//   - Blocking: waits for peer responses
//   - Majority-based: requires ceil(N/2) acknowledgments (including self)
//   - Higher latency: client waits for network round-trips
//   - Better consistency: guarantees majority has the update before returning
//
// Returns true if majority was achieved, false otherwise.
func (r *Replica) replicateStrong(entry DataEntry) bool {
	repReq := ReplicationRequest{
		Entry:     entry,
		OriginID:  r.config.ID,
		Timestamp: time.Now().UnixNano(),
	}

	jsonData, err := json.Marshal(repReq)
	if err != nil {
		log.Printf("[%s] Error marshaling replication request: %v", r.config.ID, err)
		return false
	}

	// Calculate majority needed
	// Total nodes = peers + self
	totalNodes := len(r.config.Peers) + 1
	// Majority = floor(totalNodes/2) + 1
	// For 3 nodes: floor(3/2) + 1 = 1 + 1 = 2
	// For 5 nodes: floor(5/2) + 1 = 2 + 1 = 3
	majorityNeeded := totalNodes/2 + 1
	
	// Self is already acknowledged (we already stored the value locally)
	acknowledged := 1

	// Channel to collect results from peers
	type ackResult struct {
		success bool
		peer    string
	}
	ackChan := make(chan ackResult, len(r.config.Peers))

	// Send replication to all peers concurrently
	for _, peer := range r.config.Peers {
		go func(peerURL string) {
			resp, err := r.client.Post(
				peerURL+"/replicate",
				"application/json",
				bytes.NewBuffer(jsonData),
			)
			if err != nil {
				log.Printf("[%s] Failed to replicate to %s: %v",
					r.config.ID, peerURL, err)
				ackChan <- ackResult{success: false, peer: peerURL}
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var repResp ReplicationResponse
				if err := json.NewDecoder(resp.Body).Decode(&repResp); err == nil && repResp.Success {
					ackChan <- ackResult{success: true, peer: peerURL}
					return
				}
			}
			ackChan <- ackResult{success: false, peer: peerURL}
		}(peer)
	}

	// Wait for all peer responses
	for i := 0; i < len(r.config.Peers); i++ {
		result := <-ackChan
		if result.success {
			acknowledged++
			log.Printf("[%s] ✅ Majority ACK received from %s (%d/%d)",
				r.config.ID, result.peer, acknowledged, totalNodes)
		} else {
			log.Printf("[%s] ❌ No ACK from %s (%d/%d)",
				r.config.ID, result.peer, acknowledged, totalNodes)
		}
	}

	success := acknowledged >= majorityNeeded
	log.Printf("[%s] Strong replication result: %d/%d acknowledged (need %d) - %s",
		r.config.ID, acknowledged, totalNodes, majorityNeeded,
		map[bool]string{true: "SUCCESS ✅", false: "FAILED ❌"}[success])
	
	return success
}

// sendReplication sends a replication request to a single peer.
// This is a helper method used by both eventual and strong replication.
func (r *Replica) sendReplication(peerURL string, jsonData []byte) {
	resp, err := r.client.Post(peerURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[%s] Failed to send replication to %s: %v",
			r.config.ID, peerURL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var repResp ReplicationResponse
		if err := json.NewDecoder(resp.Body).Decode(&repResp); err == nil && repResp.Success {
			log.Printf("[%s] Successfully replicated to %s: %s",
				r.config.ID, peerURL, repResp.Message)
		} else {
			log.Printf("[%s] Replication to %s returned unexpected response",
				r.config.ID, peerURL)
		}
	} else {
		log.Printf("[%s] Replication to %s failed with status %d",
			r.config.ID, peerURL, resp.StatusCode)
	}
}

// ============================================================================
// MIDDLEWARE
// ============================================================================

// corsMiddleware adds Cross-Origin Resource Sharing (CORS) headers to all responses.
// This allows the client to make requests from browser-based tools if needed.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs all incoming HTTP requests with method, path, and timing.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Call the next handler
		next.ServeHTTP(w, r)
		
		// Log the request
		log.Printf("[%s] %s %s - %v",
			r.RemoteAddr, r.Method, r.URL.Path, time.Since(start))
	})
}

// ============================================================================
// MAIN FUNCTION
// ============================================================================

func main() {
	// Check command-line arguments
	if len(os.Args) < 2 {
		log.Fatal("Usage: replica <config-file>\nExample: replica ../configs/replica1.json")
	}

	configFile := os.Args[1]
	
	// Read configuration file
	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to read config file '%s': %v", configFile, err)
	}

	// Parse configuration
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatalf("Failed to parse config file '%s': %v", configFile, err)
	}

	// Validate configuration
	if config.ID == "" {
		log.Fatal("Configuration error: 'id' is required")
	}
	if config.Port == 0 {
		log.Fatal("Configuration error: 'port' is required")
	}
	if len(config.Peers) == 0 {
		log.Println("⚠️  Warning: No peers configured. This replica will operate in isolation.")
	}
	if config.ConsistencyModel != "eventual" && config.ConsistencyModel != "strong" {
		log.Fatalf("Configuration error: 'consistency_model' must be 'eventual' or 'strong', got '%s'",
			config.ConsistencyModel)
	}

	// Create the replica instance
	replica := NewReplica(config)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/put", replica.handlePUT)
	mux.HandleFunc("/get", replica.handleGET)
	mux.HandleFunc("/replicate", replica.handleReplicate)
	mux.HandleFunc("/health", replica.handleHealth)
	mux.HandleFunc("/stop", replica.handleStop)
	mux.HandleFunc("/start", replica.handleStart)
	mux.HandleFunc("/data", replica.handleData)

	// Apply middleware
	handler := corsMiddleware(loggingMiddleware(mux))

	// Start the HTTP server
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	
	log.Printf("╔══════════════════════════════════════════════════════════════╗")
	log.Printf("║  REPLICA SERVER                                             ║")
	log.Printf("╠══════════════════════════════════════════════════════════════╣")
	log.Printf("║  ID:              %-42s ║", config.ID)
	log.Printf("║  Address:         %-42s ║", addr)
	log.Printf("║  Consistency:     %-42s ║", config.ConsistencyModel)
	log.Printf("║  Network Delay:   %-38d ms ║", config.NetworkDelay)
	log.Printf("║  Peers:           %-42s ║", fmt.Sprintf("%d peers", len(config.Peers)))
	for i, peer := range config.Peers {
		log.Printf("║    Peer %d:         %-38s ║", i+1, peer)
	}
	log.Printf("╚══════════════════════════════════════════════════════════════╝")
	
	log.Printf("Server starting on %s...", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}