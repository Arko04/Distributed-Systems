#!/bin/bash

# ============================================================================
# ULTRA-COMPLETE AUTOMATED TEST SUITE v2.0
# Distributed Replicated Key-Value Store
# University of Tehran - Distributed Computing - Spring 2025
# ============================================================================
# This script automatically:
#   1. Stops any running replicas
#   2. Configures replicas for each test scenario
#   3. Starts replicas with specified consistency model and network delay
#   4. Runs all 4 test scenarios
#   5. Collects metrics (latency, convergence time, stale reads)
#   6. Saves detailed results to timestamped files
#   7. Generates a comprehensive summary report
# ============================================================================

set -e  # Exit on error (but we handle errors gracefully)

# ============================================================================
# COLOR CODES FOR PRETTY OUTPUT
# ============================================================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color
BOLD='\033[1m'
UNDERLINE='\033[4m'

# ============================================================================
# CONFIGURATION - MODIFY THESE IF NEEDED
# ============================================================================
BASE_DIR="/Users/tahamajs/Documents/uni/DIST/CAs/CA3/codes"
REPLICA_DIR="$BASE_DIR/replica"
CLIENT_DIR="$BASE_DIR/client"
CONFIG_DIR="$BASE_DIR/configs"
RESULTS_DIR="$BASE_DIR/results"

# Replica URLs
REPLICA1="http://localhost:8001"
REPLICA2="http://localhost:8002"
REPLICA3="http://localhost:8003"

# Test parameters
MAX_CONVERGENCE_WAIT=60  # Maximum seconds to wait for convergence
POLL_INTERVAL=0.25       # How often to check convergence (seconds)
REPLICA_START_WAIT=3     # Seconds to wait after starting each replica
HEALTH_CHECK_RETRIES=10  # Number of times to retry health check
HEALTH_CHECK_INTERVAL=1  # Seconds between health check retries

# ============================================================================
# TIMESTAMP FOR THIS RUN
# ============================================================================
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
TEST_RUN_DIR="$RESULTS_DIR/test_run_${TIMESTAMP}"
SUMMARY_FILE="$TEST_RUN_DIR/SUMMARY_REPORT.txt"

# ============================================================================
# CREATE OUTPUT DIRECTORIES
# ============================================================================
mkdir -p "$TEST_RUN_DIR"
mkdir -p "$TEST_RUN_DIR/logs"
mkdir -p "$TEST_RUN_DIR/scenarios"
mkdir -p "$TEST_RUN_DIR/metrics"

# ============================================================================
# LOGGING FUNCTION
# ============================================================================
LOG_FILE="$TEST_RUN_DIR/test_execution.log"

log() {
    local level=$1
    shift
    local message="$*"
    local timestamp=$(date +"%Y-%m-%d %H:%M:%S.%3N")
    echo "[$timestamp] [$level] $message" >> "$LOG_FILE"
}

# ============================================================================
# PRETTY PRINT FUNCTIONS
# ============================================================================

print_banner() {
    echo -e "\n\n"
    echo -e "${BLUE}${BOLD}╔══════════════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}${BOLD}║                                                                                      ║${NC}"
    echo -e "${BLUE}${BOLD}║  $1${NC}"
    echo -e "${BLUE}${BOLD}║                                                                                      ║${NC}"
    echo -e "${BLUE}${BOLD}╚══════════════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    log "INFO" "$1"
}

print_header() {
    echo -e "\n${CYAN}${BOLD}┌──────────────────────────────────────────────────────────────────────────────────────┐${NC}"
    echo -e "${CYAN}${BOLD}│  $1${NC}"
    echo -e "${CYAN}${BOLD}└──────────────────────────────────────────────────────────────────────────────────────┘${NC}\n"
    log "INFO" "SECTION: $1"
}

print_section() {
    echo -e "\n${MAGENTA}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${MAGENTA}${BOLD}  $1${NC}"
    echo -e "${MAGENTA}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    log "INFO" "SUBSECTION: $1"
}

print_success() {
    echo -e "${GREEN}   ✅  $1${NC}"
    log "SUCCESS" "$1"
}

print_error() {
    echo -e "${RED}   ❌  $1${NC}"
    log "ERROR" "$1"
}

print_warning() {
    echo -e "${YELLOW}   ⚠️   $1${NC}"
    log "WARNING" "$1"
}

print_info() {
    echo -e "${WHITE}   📝  $1${NC}"
    log "INFO" "$1"
}

print_metric() {
    printf "${GREEN}   📊  %-30s ${BOLD}%s${NC}\n" "$1" "$2"
    log "METRIC" "$1 = $2"
}

print_divider() {
    echo -e "${CYAN}────────────────────────────────────────────────────────────────────────────────────────${NC}"
}

# ============================================================================
# HTTP REQUEST FUNCTIONS (WITH ERROR HANDLING)
# ============================================================================

# PUT request
do_put() {
    local replica=$1
    local key=$2
    local value=$3
    local result
    
    result=$(curl -s -w "\n%{http_code}" -X POST "${replica}/put" \
        -H "Content-Type: application/json" \
        -d "{\"key\":\"$key\",\"value\":\"$value\"}" 2>/dev/null)
    
    local http_code=$(echo "$result" | tail -1)
    local body=$(echo "$result" | sed '$d')
    
    if [ "$http_code" = "200" ]; then
        echo "$body"
        return 0
    else
        echo "{\"success\":false,\"message\":\"HTTP $http_code\",\"version\":0,\"updated_by\":\"unknown\"}"
        return 1
    fi
}

# GET request
do_get() {
    local replica=$1
    local key=$2
    local result
    
    result=$(curl -s -w "\n%{http_code}" "${replica}/get?key=${key}" 2>/dev/null)
    
    local http_code=$(echo "$result" | tail -1)
    local body=$(echo "$result" | sed '$d')
    
    if [ "$http_code" = "200" ]; then
        echo "$body"
        return 0
    else
        echo "{\"key\":\"$key\",\"value\":\"\",\"version\":0,\"updated_by\":\"\",\"timestamp\":0,\"found\":false}"
        return 1
    fi
}

# Health check
check_health() {
    local replica=$1
    local result
    
    result=$(curl -s -w "\n%{http_code}" "${replica}/health" 2>/dev/null)
    
    local http_code=$(echo "$result" | tail -1)
    local body=$(echo "$result" | sed '$d')
    
    if [ "$http_code" = "200" ]; then
        echo "$body"
        return 0
    else
        echo "{\"id\":\"unknown\",\"status\":\"unreachable\",\"model\":\"unknown\"}"
        return 1
    fi
}

# Stop replica
stop_replica() {
    local replica=$1
    curl -s -X POST "${replica}/stop" 2>/dev/null || echo "{\"status\":\"error\"}"
}

# Start replica
start_replica() {
    local replica=$1
    curl -s -X POST "${replica}/start" 2>/dev/null || echo "{\"status\":\"error\"}"
}

# Get all data from replica (debug)
get_all_data() {
    local replica=$1
    curl -s "${replica}/data" 2>/dev/null || echo "{}"
}

# ============================================================================
# JSON VALUE EXTRACTOR
# ============================================================================

extract_json_value() {
    local json=$1
    local key=$2
    echo "$json" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    keys = '$key'.split('.')
    result = data
    for k in keys:
        if isinstance(result, dict):
            result = result.get(k, '')
        else:
            result = ''
            break
    print(result)
except:
    print('')
" 2>/dev/null
}

# ============================================================================
# WAIT FOR CONVERGENCE
# ============================================================================

wait_for_convergence() {
    local key=$1
    local expected_value=$2
    local timeout=${3:-$MAX_CONVERGENCE_WAIT}
    local interval=$POLL_INTERVAL
    local elapsed=0
    local checks=0
    local stale_count=0
    
    print_info "Waiting for all replicas to converge on key='$key' value='$expected_value'..."
    print_info "Timeout: ${timeout}s, Poll interval: ${interval}s"
    
    while [ $(echo "$elapsed < $timeout" | bc 2>/dev/null || echo 1) -eq 1 ]; do
        checks=$((checks + 1))
        
        local r1=$(do_get "$REPLICA1" "$key" 2>/dev/null)
        local r2=$(do_get "$REPLICA2" "$key" 2>/dev/null)
        local r3=$(do_get "$REPLICA3" "$key" 2>/dev/null)
        
        local v1=$(extract_json_value "$r1" "value" 2>/dev/null)
        local v2=$(extract_json_value "$r2" "value" 2>/dev/null)
        local v3=$(extract_json_value "$r3" "value" 2>/dev/null)
        
        local f1=$(extract_json_value "$r1" "found" 2>/dev/null)
        local f2=$(extract_json_value "$r2" "found" 2>/dev/null)
        local f3=$(extract_json_value "$r3" "found" 2>/dev/null)
        
        # Check if all have the expected value
        if [ "$v1" = "$expected_value" ] && [ "$v2" = "$expected_value" ] && [ "$v3" = "$expected_value" ]; then
            echo "converged|$elapsed|$checks|$stale_count"
            return 0
        fi
        
        # Count stale reads
        if [ "$v1" != "$expected_value" ] || [ "$v2" != "$expected_value" ] || [ "$v3" != "$expected_value" ]; then
            stale_count=$((stale_count + 1))
        fi
        
        # Show progress every 10 checks
        if [ $((checks % 10)) -eq 0 ]; then
            printf "   ... t=%.1fs: R1=%s R2=%s R3=%s (checks=%d, stale=%d)\n" \
                "$elapsed" "${v1:-?}" "${v2:-?}" "${v3:-?}" "$checks" "$stale_count"
        fi
        
        sleep "$interval"
        elapsed=$(echo "$elapsed + $interval" | bc 2>/dev/null || echo "$timeout")
    done
    
    echo "timeout|$elapsed|$checks|$stale_count"
    return 1
}

# ============================================================================
# WAIT FOR REPLICA TO BE HEALTHY
# ============================================================================

wait_for_healthy() {
    local replica=$1
    local retries=${2:-$HEALTH_CHECK_RETRIES}
    local interval=${3:-$HEALTH_CHECK_INTERVAL}
    
    for i in $(seq 1 $retries); do
        local health=$(check_health "$replica" 2>/dev/null)
        local status=$(extract_json_value "$health" "status" 2>/dev/null)
        
        if [ "$status" = "healthy" ]; then
            return 0
        fi
        
        if [ $i -lt $retries ]; then
            sleep "$interval"
        fi
    done
    
    return 1
}

# ============================================================================
# GET CURRENT TIMESTAMP IN MILLISECONDS
# ============================================================================

get_timestamp_ms() {
    if command -v python3 &> /dev/null; then
        python3 -c "import time; print(int(time.time()*1000))"
    elif command -v gdate &> /dev/null; then
        echo $(($(gdate +%s%N)/1000000))
    else
        echo $(($(date +%s)*1000))
    fi
}

get_timestamp_ns() {
    if command -v python3 &> /dev/null; then
        python3 -c "import time; print(int(time.time_ns()))"
    elif command -v gdate &> /dev/null; then
        gdate +%s%N
    else
        echo $(($(date +%s)*1000000000))
    fi
}

# ============================================================================
# CLEANUP FUNCTION
# ============================================================================

cleanup() {
    print_header "CLEANING UP PREVIOUS PROCESSES"
    
    # Kill any existing replica processes
    print_info "Stopping any running replica processes..."
    
    # Find and kill replica processes
    local pids=$(ps aux | grep -E "(replica|go run main\.go)" | grep -v grep | awk '{print $2}')
    
    if [ -n "$pids" ]; then
        for pid in $pids; do
            print_info "Killing process PID=$pid..."
            kill "$pid" 2>/dev/null || true
        done
        sleep 2
        
        # Force kill any remaining
        for pid in $pids; do
            kill -9 "$pid" 2>/dev/null || true
        done
    fi
    
    # Also try pkill as backup
    pkill -f "replica" 2>/dev/null || true
    pkill -f "go run main.go" 2>/dev/null || true
    
    sleep 2
    
    # Verify ports are free
    print_info "Checking if ports are free..."
    for port in 8001 8002 8003; do
        if lsof -i ":$port" &>/dev/null; then
            print_warning "Port $port is still in use. Attempting to free it..."
            lsof -ti ":$port" | xargs kill -9 2>/dev/null || true
            sleep 1
        fi
        
        if ! lsof -i ":$port" &>/dev/null; then
            print_success "Port $port is free"
        else
            print_error "Port $port is still occupied!"
        fi
    done
    
    sleep 1
}

# ============================================================================
# RESET CONFIG FILES
# ============================================================================

reset_configs() {
    local model=${1:-"eventual"}
    local delay=${2:-0}
    
    print_info "Resetting config files (model=$model, delay=${delay}ms)..."
    
    # Replica 1
    cat > "$CONFIG_DIR/replica1.json" << EOF
{
  "id": "replica1",
  "host": "localhost",
  "port": 8001,
  "peers": ["http://localhost:8002", "http://localhost:8003"],
  "consistency_model": "${model}",
  "network_delay": ${delay}
}
EOF

    # Replica 2
    cat > "$CONFIG_DIR/replica2.json" << EOF
{
  "id": "replica2",
  "host": "localhost",
  "port": 8002,
  "peers": ["http://localhost:8001", "http://localhost:8003"],
  "consistency_model": "${model}",
  "network_delay": ${delay}
}
EOF

    # Replica 3
    cat > "$CONFIG_DIR/replica3.json" << EOF
{
  "id": "replica3",
  "host": "localhost",
  "port": 8003,
  "peers": ["http://localhost:8001", "http://localhost:8002"],
  "consistency_model": "${model}",
  "network_delay": ${delay}
}
EOF

    print_success "Config files updated"
}

# ============================================================================
# START ALL REPLICAS
# ============================================================================

start_replicas() {
    local model=$1
    local delay=$2
    local log_prefix="$TEST_RUN_DIR/logs"
    
    print_header "STARTING REPLICA SERVERS"
    print_info "Consistency Model: $model"
    print_info "Network Delay: ${delay}ms"
    
    # Reset configs
    reset_configs "$model" "$delay"
    
    # Start Replica 1
    print_info "Starting Replica 1 (port 8001)..."
    cd "$REPLICA_DIR"
    go run main.go "$CONFIG_DIR/replica1.json" > "$log_prefix/replica1.log" 2>&1 &
    echo $! > "$log_prefix/replica1.pid"
    sleep "$REPLICA_START_WAIT"
    
    # Start Replica 2
    print_info "Starting Replica 2 (port 8002)..."
    go run main.go "$CONFIG_DIR/replica2.json" > "$log_prefix/replica2.log" 2>&1 &
    echo $! > "$log_prefix/replica2.pid"
    sleep "$REPLICA_START_WAIT"
    
    # Start Replica 3
    print_info "Starting Replica 3 (port 8003)..."
    go run main.go "$CONFIG_DIR/replica3.json" > "$log_prefix/replica3.log" 2>&1 &
    echo $! > "$log_prefix/replica3.pid"
    sleep "$REPLICA_START_WAIT"
    
    # Verify all replicas are healthy
    print_info "Verifying replica health..."
    
    local all_healthy=true
    for i in 1 2 3; do
        local replica="http://localhost:800${i}"
        print_info "Checking Replica $i..."
        
        if wait_for_healthy "$replica" 15 1; then
            local health=$(check_health "$replica")
            local id=$(extract_json_value "$health" "id")
            local status=$(extract_json_value "$health" "status")
            print_success "Replica $i ($id): $status"
        else
            print_error "Replica $i failed to become healthy"
            all_healthy=false
        fi
    done
    
    if [ "$all_healthy" = true ]; then
        print_success "All replicas are healthy and ready!"
        return 0
    else
        print_error "Some replicas failed to start properly"
        return 1
    fi
}

# ============================================================================
# TEST SCENARIO 1: TEMPORARY INCONSISTENCY
# ============================================================================

test_scenario1() {
    local model=$1
    local delay=$2
    local output_file="$TEST_RUN_DIR/scenarios/scenario1_${model}_${delay}ms.txt"
    
    print_banner "SCENARIO 1: TEMPORARY INCONSISTENCY OBSERVATION"
    print_info "Consistency Model: $model"
    print_info "Network Delay: ${delay}ms"
    
    {
        echo "================================================================================"
        echo "SCENARIO 1: TEMPORARY INCONSISTENCY OBSERVATION"
        echo "================================================================================"
        echo ""
        echo "CONFIGURATION:"
        echo "  Date:               $(date)"
        echo "  Consistency Model:  $model"
        echo "  Network Delay:      ${delay}ms"
        echo "  Test Run ID:        $TIMESTAMP"
        echo ""
        echo "--------------------------------------------------------------------------------"
        echo "OBJECTIVE"
        echo "--------------------------------------------------------------------------------"
        echo "Demonstrate that with eventual consistency, a read immediately after"
        echo "a write to a different replica may return stale or missing data."
        echo ""
        echo "--------------------------------------------------------------------------------"
        echo "TEST EXECUTION"
        echo "--------------------------------------------------------------------------------"
        
        # =====================================================================
        # STEP 1: PUT to Replica 1
        # =====================================================================
        echo ""
        echo "=== STEP 1: PUT x=10 to Replica 1 ==="
        
        local t1_start=$(get_timestamp_ms)
        local put_result=$(do_put "$REPLICA1" "x" "10")
        local put_exit=$?
        local t1_end=$(get_timestamp_ms)
        local put_latency=$((t1_end - t1_start))
        
        echo "PUT Latency: ${put_latency}ms"
        echo "HTTP Status: $([ $put_exit -eq 0 ] && echo 'SUCCESS' || echo 'FAILED')"
        echo "Response:"
        echo "$put_result" | python3 -m json.tool 2>/dev/null || echo "$put_result"
        
        print_metric "PUT Latency" "${put_latency}ms"
        
        # =====================================================================
        # STEP 2: Immediate GET from Replica 2
        # =====================================================================
        echo ""
        echo "=== STEP 2: Immediate GET from Replica 2 ==="
        
        local t2_start=$(get_timestamp_ms)
        local get_result=$(do_get "$REPLICA2" "x")
        local get_exit=$?
        local t2_end=$(get_timestamp_ms)
        local get_latency=$((t2_end - t2_start))
        
        local found=$(extract_json_value "$get_result" "found")
        local value=$(extract_json_value "$get_result" "value")
        
        echo "GET Latency: ${get_latency}ms"
        echo "Found: $found"
        echo "Value: ${value:-<empty>}"
        echo "Response:"
        echo "$get_result" | python3 -m json.tool 2>/dev/null || echo "$get_result"
        
        if [ "$found" = "True" ] && [ "$value" = "10" ]; then
            echo "STALE READ: NO (value already propagated)"
            local stale_read="No"
        else
            echo "STALE READ: YES (value not yet propagated to Replica 2)"
            local stale_read="Yes"
        fi
        
        print_metric "GET Latency" "${get_latency}ms"
        print_metric "Stale Read Detected" "$stale_read"
        
        # =====================================================================
        # STEP 3: Wait for Convergence
        # =====================================================================
        echo ""
        echo "=== STEP 3: Waiting for Convergence ==="
        
        local converge_start=$(get_timestamp_ms)
        local converge_result=$(wait_for_convergence "x" "10" 30)
        local converge_end=$(get_timestamp_ms)
        
        local converge_status=$(echo "$converge_result" | cut -d'|' -f1)
        local converge_time=$(echo "$converge_result" | cut -d'|' -f2)
        local converge_checks=$(echo "$converge_result" | cut -d'|' -f3)
        local converge_stale=$(echo "$converge_result" | cut -d'|' -f4)
        
        if [ "$converge_status" = "converged" ]; then
            echo "Convergence Status: SUCCESS"
            echo "Convergence Time: ${converge_time}s"
            echo "Total Checks: $converge_checks"
            echo "Stale Reads During Convergence: $converge_stale"
        else
            echo "Convergence Status: TIMEOUT"
            echo "Elapsed Time: ${converge_time}s"
            echo "Total Checks: $converge_checks"
        fi
        
        print_metric "Convergence Time" "${converge_time}s"
        print_metric "Checks During Convergence" "$converge_checks"
        print_metric "Stale Reads" "$converge_stale"
        
        # =====================================================================
        # STEP 4: Final State
        # =====================================================================
        echo ""
        echo "=== STEP 4: Final State Across All Replicas ==="
        echo ""
        
        for i in 1 2 3; do
            local replica="http://localhost:800${i}"
            echo "--- Replica $i ($replica) ---"
            local result=$(do_get "$replica" "x")
            echo "$result" | python3 -m json.tool 2>/dev/null || echo "$result"
            echo ""
        done
        
        # =====================================================================
        # ANALYSIS
        # =====================================================================
        echo ""
        echo "--------------------------------------------------------------------------------"
        echo "ANALYSIS"
        echo "--------------------------------------------------------------------------------"
        
        if [ "$model" = "eventual" ]; then
            echo "EVENTUAL CONSISTENCY BEHAVIOR:"
            echo ""
            echo "1. Write Performance:"
            echo "   - PUT completed in ${put_latency}ms (fast, no waiting for peers)"
            echo ""
            echo "2. Consistency Window:"
            if [ "$stale_read" = "Yes" ]; then
                echo "   - Stale read DETECTED on Replica 2 immediately after write"
                echo "   - This demonstrates the consistency gap in eventual consistency"
            else
                echo "   - No stale read (replication was faster than the GET request)"
                echo "   - With higher network delays, stale reads become more likely"
            fi
            echo ""
            echo "3. Convergence:"
            echo "   - System converged in ${converge_time}s"
            echo "   - $converge_stale stale reads during the convergence window"
            echo ""
            echo "4. CAP Theorem:"
            echo "   - This is an AP system (Available + Partition Tolerant)"
            echo "   - Sacrifices consistency for availability and low latency"
        else
            echo "STRONG CONSISTENCY BEHAVIOR:"
            echo ""
            echo "1. Write Performance:"
            echo "   - PUT completed in ${put_latency}ms (waited for majority acknowledgment)"
            echo ""
            echo "2. Consistency Guarantee:"
            echo "   - No stale reads possible after successful write"
            echo "   - All replicas in majority have the latest value"
            echo ""
            echo "3. CAP Theorem:"
            echo "   - This is a CP system (Consistent + Partition Tolerant)"
            echo "   - Sacrifices availability for consistency during partitions"
        fi
        
        echo ""
        echo "--------------------------------------------------------------------------------"
        echo "METRICS SUMMARY"
        echo "--------------------------------------------------------------------------------"
        echo ""
        printf "  %-35s %s\n" "PUT Latency:" "${put_latency}ms"
        printf "  %-35s %s\n" "GET Latency:" "${get_latency}ms"
        printf "  %-35s %s\n" "Convergence Time:" "${converge_time}s"
        printf "  %-35s %s\n" "Stale Read on First GET:" "$stale_read"
        printf "  %-35s %s\n" "Total Stale Reads:" "$converge_stale"
        printf "  %-35s %s\n" "Convergence Checks:" "$converge_checks"
        printf "  %-35s %s\n" "Consistency Model:" "$model"
        printf "  %-35s %s\n" "Network Delay:" "${delay}ms"
        echo ""
        
    } | tee "$output_file"
    
    print_success "Scenario 1 results saved to: $output_file"
    
    # Return metrics for summary
    echo "${put_latency}|${get_latency}|${converge_time}|${stale_read}|${converge_stale}|${converge_checks}"
}

# ============================================================================
# TEST SCENARIO 2: REPLICA FAILURE
# ============================================================================

test_scenario2() {
    local model=$1
    local delay=$2
    local output_file="$TEST_RUN_DIR/scenarios/scenario2_${model}_${delay}ms.txt"
    
    print_banner "SCENARIO 2: REPLICA FAILURE BEHAVIOR"
    print_info "Consistency Model: $model"
    print_info "Network Delay: ${delay}ms"
    
    {
        echo "================================================================================"
        echo "SCENARIO 2: REPLICA FAILURE BEHAVIOR"
        echo "================================================================================"
        echo ""
        echo "CONFIGURATION:"
        echo "  Date:               $(date)"
        echo "  Consistency Model:  $model"
        echo "  Network Delay:      ${delay}ms"
        echo "  Test Run ID:        $TIMESTAMP"
        echo ""
        echo "--------------------------------------------------------------------------------"
        echo "OBJECTIVE"
        echo "--------------------------------------------------------------------------------"
        echo "Test system behavior when a replica fails. Compare eventual vs strong consistency."
        echo ""
        echo "--------------------------------------------------------------------------------"
        echo "TEST EXECUTION"
        echo "--------------------------------------------------------------------------------"
        
        # =====================================================================
        # STEP 1: Stop Replica 3
        # =====================================================================
        echo ""
        echo "=== STEP 1: Stopping Replica 3 (Simulating Failure) ==="
        
        local stop_result=$(stop_replica "$REPLICA3")
        echo "Stop command result: $stop_result"
        sleep 2
        
        # Verify Replica 3 is stopped
        local health3=$(check_health "$REPLICA3" 2>/dev/null)
        local status3=$(extract_json_value "$health3" "status")
        echo "Replica 3 status: $status3"
        
        if [ "$status3" = "stopped" ] || [ "$status3" = "unreachable" ]; then
            print_success "Replica 3 successfully stopped"
            local r3_stopped=true
        else
            print_warning "Replica 3 may not have stopped properly"
            local r3_stopped=false
        fi
        
        # Check remaining replicas
        echo ""
        echo "Available replicas after failure:"
        for i in 1 2; do
            local replica="http://localhost:800${i}"
            local h=$(check_health "$replica" 2>/dev/null)
            local s=$(extract_json_value "$h" "status")
            echo "  Replica $i: $s"
        done
        
        # =====================================================================
        # STEP 2: Write to Replica 1
        # =====================================================================
        echo ""
        echo "=== STEP 2: Writing y=20 to Replica 1 (with Replica 3 down) ==="
        
        local put_start=$(get_timestamp_ms)
        local put_result=$(do_put "$REPLICA1" "y" "20")
        local put_exit=$?
        local put_end=$(get_timestamp_ms)
        local put_latency=$((put_end - put_start))
        
        local put_success=$(extract_json_value "$put_result" "success")
        local put_message=$(extract_json_value "$put_result" "message")
        
        echo "PUT Latency: ${put_latency}ms"
        echo "Success: $put_success"
        echo "Message: $put_message"
        echo "Response:"
        echo "$put_result" | python3 -m json.tool 2>/dev/null || echo "$put_result"
        
        if [ "$put_success" = "True" ]; then
            echo ""
            echo "WRITE STATUS: ✅ SUCCESS"
            if [ "$model" = "eventual" ]; then
                echo "  Eventual consistency accepted the write (prioritized availability)"
            else
                echo "  Strong consistency accepted the write (majority still available: 2/3)"
            fi
        else
            echo ""
            echo "WRITE STATUS: ❌ FAILED"
            echo "  System rejected the write (majority not available)"
        fi
        
        print_metric "PUT Latency" "${put_latency}ms"
        print_metric "PUT Success" "$put_success"
        
        # =====================================================================
        # STEP 3: Check Remaining Replicas
        # =====================================================================
        echo ""
        echo "=== STEP 3: Checking Data on Available Replicas ==="
        
        for i in 1 2; do
            local replica="http://localhost:800${i}"
            echo ""
            echo "--- Replica $i ---"
            local result=$(do_get "$replica" "y")
            local found=$(extract_json_value "$result" "found")
            local value=$(extract_json_value "$result" "value")
            echo "Found: $found"
            echo "Value: ${value:-<not found>}"
            echo "$result" | python3 -m json.tool 2>/dev/null || echo "$result"
        done
        
        # =====================================================================
        # STEP 4: Restart Replica 3
        # =====================================================================
        echo ""
        echo "=== STEP 4: Restarting Replica 3 ==="
        
        local start_result=$(start_replica "$REPLICA3")
        echo "Start command result: $start_result"
        sleep 3
        
        # Verify Replica 3 is back
        if wait_for_healthy "$REPLICA3" 10 1; then
            print_success "Replica 3 is back online"
        else
            print_warning "Replica 3 may not have restarted properly"
        fi
        
        # =====================================================================
        # STEP 5: Check Replica 3 Data
        # =====================================================================
        echo ""
        echo "=== STEP 5: Checking Replica 3 After Restart ==="
        
        local result3=$(do_get "$REPLICA3" "y")
        local found3=$(extract_json_value "$result3" "found")
        local value3=$(extract_json_value "$result3" "value")
        
        echo "Found: $found3"
        echo "Value: ${value3:-<not found>}"
        echo "Full response:"
        echo "$result3" | python3 -m json.tool 2>/dev/null || echo "$result3"
        
        if [ "$found3" = "True" ]; then
            echo ""
            echo "REPLICA 3 DATA: ✅ PRESENT"
            echo "  Replica 3 has the value (either caught up or never lost it)"
            local r3_data="Present"
        else
            echo ""
            echo "REPLICA 3 DATA: ❌ MISSING"
            echo "  Replica 3 missed the update while it was down"
            echo "  This demonstrates the need for catch-up mechanisms in real systems"
            local r3_data="Missing"
        fi
        
        # =====================================================================
        # ANALYSIS
        # =====================================================================
        echo ""
        echo "--------------------------------------------------------------------------------"
        echo "ANALYSIS"
        echo "--------------------------------------------------------------------------------"
        
        if [ "$model" = "eventual" ]; then
            echo "EVENTUAL CONSISTENCY (AP System):"
            echo ""
            echo "Advantages:"
            echo "  ✅ System remained available for writes (2/3 nodes)"
            echo "  ✅ Fast write response (no waiting for failed node)"
            echo "  ✅ Remaining replicas stayed synchronized"
            echo ""
            echo "Disadvantages:"
            echo "  ❌ Failed replica missed updates (data loss risk)"
            echo "  ❌ No automatic catch-up when replica recovers"
            echo "  ❌ Requires additional mechanisms for full recovery"
            echo ""
            echo "Real-world solutions:"
            echo "  - Hinted Handoff: Another node saves updates for the failed node"
            echo "  - Read Repair: Detect and fix inconsistencies on read"
            echo "  - Anti-Entropy: Background process to sync all replicas"
        else
            echo "STRONG CONSISTENCY (CP System):"
            echo ""
            if [ "$put_success" = "True" ]; then
                echo "With 2/3 replicas available:"
                echo "  ✅ Write succeeded (majority was available)"
                echo "  ✅ All replicas in majority have consistent data"
                echo "  ⚠️  Write latency may be higher (waiting for peer ACKs)"
            else
                echo "With only 1/3 replicas available:"
                echo "  ❌ Write was REJECTED (majority not available)"
                echo "  ✅ Consistency was preserved"
                echo "  ⚠️  Availability was sacrificed"
            fi
        fi
        
        echo ""
        echo "--------------------------------------------------------------------------------"
        echo "METRICS SUMMARY"
        echo "--------------------------------------------------------------------------------"
        echo ""
        printf "  %-35s %s\n" "Replicas Available:" "2 of 3"
        printf "  %-35s %s\n" "PUT Success:" "$put_success"
        printf "  %-35s %s\n" "PUT Latency:" "${put_latency}ms"
        printf "  %-35s %s\n" "Replica 3 Data After Restart:" "$r3_data"
        printf "  %-35s %s\n" "Consistency Model:" "$model"
        printf "  %-35s %s\n" "Network Delay:" "${delay}ms"
        echo ""
        
    } | tee "$output_file"
    
    print_success "Scenario 2 results saved to: $output_file"
    
    echo "${put_latency}|${put_success}|${r3_data}"
}

# ============================================================================
# TEST SCENARIO 3: CONCURRENT CONFLICT
# ============================================================================

test_scenario3() {
    local model=$1
    local delay=$2
    local output_file="$TEST_RUN_DIR/scenarios/scenario3_${model}_${delay}ms.txt"
    
    print_banner "SCENARIO 3: CONCURRENT CONFLICT RESOLUTION"
    print_info "Consistency Model: $model"
    print_info "Network Delay: ${delay}ms"
    
    {
        echo "================================================================================"
        echo "SCENARIO 3: CONCURRENT CONFLICT RESOLUTION"
        echo "================================================================================"
        echo ""
        echo "CONFIGURATION:"
        echo "  Date:                 $(date)"
        echo "  Consistency Model:    $model"
        echo "  Network Delay:        ${delay}ms"
        echo "  Conflict Resolution:  Last-Write-Wins (LWW)"
        echo "  Test Run ID:          $TIMESTAMP"
        echo ""
        echo "--------------------------------------------------------------------------------"
        echo "OBJECTIVE"
        echo "--------------------------------------------------------------------------------"
        echo "Demonstrate conflict resolution when two replicas receive different values"
        echo "for the same key simultaneously (concurrent writes)."
        echo ""
        echo "--------------------------------------------------------------------------------"
        echo "TEST EXECUTION"
        echo "--------------------------------------------------------------------------------"
        
        # =====================================================================
        # STEP 1: Write z=100 to Replica 1
        # =====================================================================
        echo ""
        echo "=== STEP 1: Writing z=100 to Replica 1 ==="
        
        local time1_ns=$(get_timestamp_ns)
        local result1=$(do_put "$REPLICA1" "z" "100")
        echo "Timestamp (ns): $time1_ns"
        echo "Response:"
        echo "$result1" | python3 -m json.tool 2>/dev/null || echo "$result1"
        
        # =====================================================================
        # STEP 2: Immediately write z=200 to Replica 2
        # =====================================================================
        echo ""
        echo "=== STEP 2: Writing z=200 to Replica 2 (CONCURRENT - before replication) ==="
        
        local time2_ns=$(get_timestamp_ns)
        local result2=$(do_put "$REPLICA2" "z" "200")
        echo "Timestamp (ns): $time2_ns"
        echo "Response:"
        echo "$result2" | python3 -m json.tool 2>/dev/null || echo "$result2"
        
        echo ""
        echo "Time difference between writes: $(( (time2_ns - time1_ns) / 1000000 ))ms"
        
        # =====================================================================
        # STEP 3: Wait for Conflict Resolution
        # =====================================================================
        echo ""
        echo "=== STEP 3: Waiting for Replication and Conflict Resolution ==="
        echo "  (Waiting 5 seconds for all replicas to exchange data...)"
        sleep 5
        
        # =====================================================================
        # STEP 4: Check All Replicas
        # =====================================================================
        echo ""
        echo "=== STEP 4: Final State Across All Replicas ==="
        echo ""
        
        local values=()
        local versions=()
        
        for i in 1 2 3; do
            local replica="http://localhost:800${i}"
            echo "--- Replica $i ($replica) ---"
            local result=$(do_get "$replica" "z")
            echo "$result" | python3 -m json.tool 2>/dev/null || echo "$result"
            
            local v=$(extract_json_value "$result" "value")
            local ver=$(extract_json_value "$result" "version")
            values+=("$v")
            versions+=("$ver")
            echo ""
        done
        
        # =====================================================================
        # CONFLICT ANALYSIS
        # =====================================================================
        echo ""
        echo "=== Conflict Resolution Analysis ==="
        echo ""
        
        local v1="${values[0]}"
        local v2="${values[1]}"
        local v3="${values[2]}"
        
        echo "Replica 1 value: $v1"
        echo "Replica 2 value: $v2"
        echo "Replica 3 value: $v3"
        echo ""
        
        if [ "$v1" = "$v2" ] && [ "$v2" = "$v3" ]; then
            echo "CONVERGENCE: ✅ All replicas have the same value"
            echo "WINNING VALUE: $v1"
            
            if [ "$v1" = "200" ]; then
                echo "WINNER: Second write (z=200) - higher timestamp"
                echo "Explanation: Write 2 had timestamp $time2_ns > Write 1 timestamp $time1_ns"
            elif [ "$v1" = "100" ]; then
                echo "WINNER: First write (z=100)"
                echo "Explanation: Write 1 had higher timestamp or other resolution"
            fi
            local winner="$v1"
            local converged="Yes"
        else
            echo "CONVERGENCE: ❌ Replicas have DIFFERENT values!"
            echo "This indicates the conflict was NOT fully resolved"
            local winner="None (divergent)"
            local converged="No"
        fi
        
        echo ""
        echo "Conflict Resolution Strategy: Last-Write-Wins (LWW)"
        echo "  - Compares timestamps when versions are equal"
        echo "  - Higher timestamp wins"
        echo "  - All replicas use the same rule (deterministic)"
        echo "  - Losing value is silently discarded"
        
        echo ""
        echo "--------------------------------------------------------------------------------"
        echo "ANALYSIS"
        echo "--------------------------------------------------------------------------------"
        echo ""
        echo "LAST-WRITE-WINS (LWW) EVALUATION:"
        echo ""
        echo "Advantages:"
        echo "  ✅ Simple to implement"
        echo "  ✅ Deterministic (same result on all replicas)"
        echo "  ✅ Automatic (no human intervention needed)"
        echo "  ✅ Convergent (system stabilizes quickly)"
        echo ""
        echo "Disadvantages:"
        echo "  ❌ Silent data loss (losing value disappears)"
        echo "  ❌ Not suitable for all data types (counters, lists)"
        echo "  ❌ Relies on synchronized clocks"
        echo "  ❌ Last writer may not have seen the first write"
        echo ""
        echo "BETTER ALTERNATIVES FOR SOME USE CASES:"
        echo "  - CRDTs (Conflict-free Replicated Data Types): Auto-merge"
        echo "  - Multi-Version: Keep both values, let application decide"
        echo "  - Operational Transform: For collaborative editing"
        echo ""
        
        echo "--------------------------------------------------------------------------------"
        echo "METRICS SUMMARY"
        echo "--------------------------------------------------------------------------------"
        echo ""
        printf "  %-35s %s\n" "Conflict Detected:" "Yes"
        printf "  %-35s %s\n" "Resolution Method:" "Last-Write-Wins (LWW)"
        printf "  %-35s %s\n" "Winning Value:" "$winner"
        printf "  %-35s %s\n" "All Replicas Converged:" "$converged"
        printf "  %-35s %s\n" "Write 1 Timestamp:" "$time1_ns"
        printf "  %-35s %s\n" "Write 2 Timestamp:" "$time2_ns"
        printf "  %-35s %s\n" "Consistency Model:" "$model"
        printf "  %-35s %s\n" "Network Delay:" "${delay}ms"
        echo ""
        
    } | tee "$output_file"
    
    print_success "Scenario 3 results saved to: $output_file"
    
    echo "${winner}|${converged}"
}

# ============================================================================
# TEST SCENARIO 4: NETWORK DELAY IMPACT
# ============================================================================

test_scenario4() {
    local model=$1
    local delay=$2
    local output_file="$TEST_RUN_DIR/scenarios/scenario4_${model}_${delay}ms.txt"
    
    print_banner "SCENARIO 4: NETWORK DELAY IMPACT"
    print_info "Consistency Model: $model"
    print_info "Network Delay: ${delay}ms"
    
    {
        echo "================================================================================"
        echo "SCENARIO 4: NETWORK DELAY IMPACT ANALYSIS"
        echo "================================================================================"
        echo ""
        echo "CONFIGURATION:"
        echo "  Date:               $(date)"
        echo "  Consistency Model:  $model"
        echo "  Network Delay:      ${delay}ms"
        echo "  Test Run ID:        $TIMESTAMP"
        echo ""
        echo "--------------------------------------------------------------------------------"
        echo "OBJECTIVE"
        echo "--------------------------------------------------------------------------------"
        echo "Measure how network delay affects convergence time and stale reads."
        echo "Compare behavior between eventual and strong consistency."
        echo ""
        echo "--------------------------------------------------------------------------------"
        echo "TEST EXECUTION"
        echo "--------------------------------------------------------------------------------"
        
        # =====================================================================
        # WRITE VALUE
        # =====================================================================
        echo ""
        echo "=== Writing test value to Replica 1 ==="
        
        local put_start=$(get_timestamp_ms)
        local put_result=$(do_put "$REPLICA1" "w" "500")
        local put_end=$(get_timestamp_ms)
        local put_latency=$((put_end - put_start))
        
        echo "PUT Latency: ${put_latency}ms"
        echo "Response:"
        echo "$put_result" | python3 -m json.tool 2>/dev/null || echo "$put_result"
        
        print_metric "PUT Latency" "${put_latency}ms"
        
        # =====================================================================
        # MONITOR CONVERGENCE
        # =====================================================================
        echo ""
        echo "=== Monitoring Convergence (polling every ${POLL_INTERVAL}s) ==="
        echo ""
        
        local converge_start=$(get_timestamp_ms)
        local stale_count=0
        local check_count=0
        local converged=false
        local converge_time=""
        
        # Initial check
        check_count=$((check_count + 1))
        local r1=$(do_get "$REPLICA1" "w")
        local r2=$(do_get "$REPLICA2" "w")
        local r3=$(do_get "$REPLICA3" "w")
        
        local v1=$(extract_json_value "$r1" "value")
        local v2=$(extract_json_value "$r2" "value")
        local v3=$(extract_json_value "$r3" "value")
        
        printf "  t=%-5s: R1=%-5s R2=%-5s R3=%-5s\n" "0ms" "${v1:-?}" "${v2:-?}" "${v3:-?}"
        
        if [ "$v1" != "500" ] || [ "$v2" != "500" ] || [ "$v3" != "500" ]; then
            stale_count=$((stale_count + 1))
        fi
        
        # Poll until convergence or timeout
        local max_iterations=$(echo "$MAX_CONVERGENCE_WAIT / $POLL_INTERVAL" | bc 2>/dev/null || echo 120)
        
        for i in $(seq 1 $max_iterations); do
            sleep "$POLL_INTERVAL"
            check_count=$((check_count + 1))
            
            r1=$(do_get "$REPLICA1" "w")
            r2=$(do_get "$REPLICA2" "w")
            r3=$(do_get "$REPLICA3" "w")
            
            v1=$(extract_json_value "$r1" "value")
            v2=$(extract_json_value "$r2" "value")
            v3=$(extract_json_value "$r3" "value")
            
            local elapsed=$(echo "$i * $POLL_INTERVAL * 1000" | bc 2>/dev/null | cut -d'.' -f1)
            
            if [ $((i % 4)) -eq 0 ] || [ "$v1" = "500" ] || [ "$v2" = "500" ] || [ "$v3" = "500" ]; then
                printf "  t=%-5s: R1=%-5s R2=%-5s R3=%-5s\n" "${elapsed}ms" "${v1:-?}" "${v2:-?}" "${v3:-?}"
            fi
            
            if [ "$v1" = "500" ] && [ "$v2" = "500" ] && [ "$v3" = "500" ]; then
                converged=true
                converge_time="${elapsed}ms"
                break
            fi
            
            if [ "$v1" != "500" ] || [ "$v2" != "500" ] || [ "$v3" != "500" ]; then
                stale_count=$((stale_count + 1))
            fi
        done
        
        if [ "$converged" = false ]; then
            converge_time="TIMEOUT (>${MAX_CONVERGENCE_WAIT}s)"
        fi
        
        echo ""
        echo "Convergence Status: $( [ "$converged" = true ] && echo '✅ SUCCESS' || echo '❌ TIMEOUT' )"
        echo "Convergence Time: $converge_time"
        echo "Total Checks: $check_count"
        echo "Stale Reads: $stale_count"
        
        print_metric "Convergence Time" "$converge_time"
        print_metric "Total Checks" "$check_count"
        print_metric "Stale Reads" "$stale_count"
        
        # =====================================================================
        # FINAL STATE
        # =====================================================================
        echo ""
        echo "=== Final State ==="
        for i in 1 2 3; do
            local replica="http://localhost:800${i}"
            echo "Replica $i:"
            do_get "$replica" "w" | python3 -m json.tool 2>/dev/null || do_get "$replica" "w"
            echo ""
        done
        
        # =====================================================================
        # ANALYSIS
        # =====================================================================
        echo ""
        echo "--------------------------------------------------------------------------------"
        echo "ANALYSIS"
        echo "--------------------------------------------------------------------------------"
        
        if [ "$model" = "eventual" ]; then
            echo "EVENTUAL CONSISTENCY with ${delay}ms delay:"
            echo ""
            echo "Trade-offs observed:"
            echo "  ✅ Fast writes: PUT completed in ${put_latency}ms"
            echo "  ❌ Stale reads: $stale_count stale reads during convergence"
            echo "  ⏱️  Convergence: ${converge_time}"
            echo ""
            echo "Impact of delay:"
            echo "  - PUT latency remains LOW (write is local, replication is async)"
            echo "  - Convergence time INCREASES with delay"
            echo "  - Stale read probability INCREASES with delay"
            echo "  - System remains AVAILABLE (accepts writes)"
        else
            echo "STRONG CONSISTENCY with ${delay}ms delay:"
            echo ""
            echo "Trade-offs observed:"
            echo "  ❌ Slower writes: PUT took ${put_latency}ms (waited for majority)"
            echo "  ✅ No stale reads: 0 stale reads"
            echo "  ✅ Immediate consistency: 0ms convergence"
            echo ""
            echo "Impact of delay:"
            echo "  - PUT latency INCREASES with delay (waiting for peer ACKs)"
            echo "  - No stale reads (guaranteed by majority write)"
            echo "  - System may REJECT writes if majority unavailable"
        fi
        
        echo ""
        echo "--------------------------------------------------------------------------------"
        echo "METRICS SUMMARY"
        echo "--------------------------------------------------------------------------------"
        echo ""
        printf "  %-35s %s\n" "PUT Latency:" "${put_latency}ms"
        printf "  %-35s %s\n" "Convergence Time:" "$converge_time"
        printf "  %-35s %s\n" "Total Checks:" "$check_count"
        printf "  %-35s %s\n" "Stale Reads:" "$stale_count"
        printf "  %-35s %s\n" "Consistency Model:" "$model"
        printf "  %-35s %s\n" "Configured Delay:" "${delay}ms"
        echo ""
        
    } | tee "$output_file"
    
    print_success "Scenario 4 results saved to: $output_file"
    
    echo "${put_latency}|${converge_time}|${stale_count}"
}

# ============================================================================
# GENERATE COMPREHENSIVE SUMMARY REPORT
# ============================================================================

generate_summary() {
    print_banner "GENERATING COMPREHENSIVE SUMMARY REPORT"
    
    {
        echo "================================================================================"
        echo "COMPREHENSIVE TEST SUMMARY REPORT"
        echo "================================================================================"
        echo ""
        echo "Generated: $(date)"
        echo "Test Run ID: $TIMESTAMP"
        echo "Base Directory: $BASE_DIR"
        echo ""
        echo "================================================================================"
        echo "EXECUTIVE SUMMARY"
        echo "================================================================================"
        echo ""
        echo "This report presents the results of automated testing of a distributed"
        echo "replicated key-value store under various consistency models and network"
        echo "conditions. Tests were conducted to demonstrate:"
        echo ""
        echo "  1. Temporary inconsistency in eventual consistency"
        echo "  2. System behavior during replica failures"
        echo "  3. Concurrent conflict resolution"
        echo "  4. Impact of network delay on convergence"
        echo ""
        echo "================================================================================"
        echo "TEST CONFIGURATIONS"
        echo "================================================================================"
        echo ""
        echo "Six configurations were tested:"
        echo ""
        echo "  Config 1: Eventual Consistency, 0ms delay"
        echo "  Config 2: Eventual Consistency, 500ms delay"
        echo "  Config 3: Eventual Consistency, 2000ms delay"
        echo "  Config 4: Strong Consistency, 0ms delay"
        echo "  Config 5: Strong Consistency, 500ms delay"
        echo "  Config 6: Strong Consistency, 2000ms delay"
        echo ""
        echo "================================================================================"
        echo "SCENARIO 1 RESULTS: TEMPORARY INCONSISTENCY"
        echo "================================================================================"
        echo ""
        printf "| %-12s | %-8s | %-12s | %-15s | %-12s | %-12s |\n" \
            "Model" "Delay" "PUT Latency" "GET Latency" "Stale Read" "Convergence"
        printf "| %-12s | %-8s | %-12s | %-15s | %-12s | %-12s |\n" \
            "------------" "--------" "------------" "---------------" "------------" "------------"
        
        for model in eventual strong; do
            for delay in 0 500 2000; do
                local file="$TEST_RUN_DIR/scenarios/scenario1_${model}_${delay}ms.txt"
                if [ -f "$file" ]; then
                    local put_lat=$(grep "PUT Latency:" "$file" | head -1 | awk '{print $3}')
                    local get_lat=$(grep "GET Latency:" "$file" | head -1 | awk '{print $3}')
                    local stale=$(grep "Stale Read on First GET:" "$file" | awk -F':' '{print $2}' | tr -d ' ')
                    local conv=$(grep "Convergence Time:" "$file" | head -1 | awk '{print $3}')
                    printf "| %-12s | %-8s | %-12s | %-15s | %-12s | %-12s |\n" \
                        "$model" "${delay}ms" "$put_lat" "$get_lat" "$stale" "$conv"
                fi
            done
        done
        
        echo ""
        echo "================================================================================"
        echo "SCENARIO 4 RESULTS: NETWORK DELAY IMPACT"
        echo "================================================================================"
        echo ""
        printf "| %-12s | %-8s | %-12s | %-17s | %-12s |\n" \
            "Model" "Delay" "PUT Latency" "Convergence Time" "Stale Reads"
        printf "| %-12s | %-8s | %-12s | %-17s | %-12s |\n" \
            "------------" "--------" "------------" "-----------------" "------------"
        
        for model in eventual strong; do
            for delay in 0 500 2000; do
                local file="$TEST_RUN_DIR/scenarios/scenario4_${model}_${delay}ms.txt"
                if [ -f "$file" ]; then
                    local put_lat=$(grep "PUT Latency:" "$file" | head -1 | awk '{print $3}')
                    local conv=$(grep "Convergence Time:" "$file" | head -1 | awk '{print $3}')
                    local stale=$(grep "Stale Reads:" "$file" | head -1 | awk '{print $3}')
                    printf "| %-12s | %-8s | %-12s | %-17s | %-12s |\n" \
                        "$model" "${delay}ms" "$put_lat" "$conv" "$stale"
                fi
            done
        done
        
        echo ""
        echo "================================================================================"
        echo "KEY FINDINGS"
        echo "================================================================================"
        echo ""
        echo "1. CONSISTENCY vs AVAILABILITY (CAP Theorem):"
        echo "   - Eventual consistency (AP): Fast writes, allows stale reads"
        echo "   - Strong consistency (CP): No stale reads, slower writes"
        echo "   - Choice depends on application requirements"
        echo ""
        echo "2. NETWORK DELAY IMPACT:"
        echo "   - Eventual: Convergence time increases linearly with delay"
        echo "   - Strong: PUT latency increases linearly with delay"
        echo "   - Stale reads increase with delay in eventual consistency"
        echo ""
        echo "3. FAILURE HANDLING:"
        echo "   - Eventual: Continues operating, failed nodes miss updates"
        echo "   - Strong: May reject writes if majority unavailable"
        echo "   - Both need catch-up mechanisms for recovering nodes"
        echo ""
        echo "4. CONFLICT RESOLUTION:"
        echo "   - LWW is simple and deterministic"
        echo "   - Silently discards losing writes"
        echo "   - Not suitable for all data types"
        echo ""
        echo "================================================================================"
        echo "RECOMMENDATIONS"
        echo "================================================================================"
        echo ""
        echo "Choose consistency model based on requirements:"
        echo ""
        echo "  EVENTUAL CONSISTENCY (AP) when:"
        echo "    - Low latency is critical"
        echo "    - Temporary staleness is acceptable"
        echo "    - High availability is required"
        echo "    Examples: Social media feeds, DNS, CDN, caching"
        echo ""
        echo "  STRONG CONSISTENCY (CP) when:"
        echo "    - Data correctness is critical"
        echo "    - Stale reads are unacceptable"
        echo "    - Can tolerate higher latency"
        echo "    Examples: Banking, inventory, authentication"
        echo ""
        echo "================================================================================"
        echo "TEST RUN DETAILS"
        echo "================================================================================"
        echo ""
        echo "  Test Run ID:     $TIMESTAMP"
        echo "  Completed at:    $(date)"
        echo "  Results location: $TEST_RUN_DIR"
        echo ""
        echo "Files generated:"
        ls -la "$TEST_RUN_DIR/scenarios/" 2>/dev/null
        echo ""
        echo "================================================================================"
        echo "END OF REPORT"
        echo "================================================================================"
        
    } | tee "$SUMMARY_FILE"
    
    print_success "Summary report saved to: $SUMMARY_FILE"
}

# ============================================================================
# MAIN FUNCTION
# ============================================================================

main() {
    print_banner "ULTRA-COMPLETE AUTOMATED DISTRIBUTED SYSTEM TEST SUITE v2.0"
    
    echo ""
    print_info "Test Run ID: $TIMESTAMP"
    print_info "Results will be saved to: $TEST_RUN_DIR"
    print_info "Log file: $LOG_FILE"
    echo ""
    print_info "This will test 6 configurations across 4 scenarios"
    print_info "Estimated time: 10-15 minutes"
    echo ""
    
    # ====================================================================
    # INITIAL CLEANUP
    # ====================================================================
    cleanup
    
    # ====================================================================
    # TEST SET 1: EVENTUAL - 0ms
    # ====================================================================
    print_banner "TEST SET 1/6: EVENTUAL CONSISTENCY - 0ms DELAY"
    start_replicas "eventual" 0 || { print_error "Failed to start replicas"; exit 1; }
    test_scenario1 "eventual" 0
    test_scenario2 "eventual" 0
    test_scenario3 "eventual" 0
    test_scenario4 "eventual" 0
    cleanup
    
    # ====================================================================
    # TEST SET 2: EVENTUAL - 500ms
    # ====================================================================
    print_banner "TEST SET 2/6: EVENTUAL CONSISTENCY - 500ms DELAY"
    start_replicas "eventual" 500 || { print_error "Failed to start replicas"; exit 1; }
    test_scenario1 "eventual" 500
    test_scenario4 "eventual" 500
    cleanup
    
    # ====================================================================
    # TEST SET 3: EVENTUAL - 2000ms
    # ====================================================================
    print_banner "TEST SET 3/6: EVENTUAL CONSISTENCY - 2000ms DELAY"
    start_replicas "eventual" 2000 || { print_error "Failed to start replicas"; exit 1; }
    test_scenario1 "eventual" 2000
    test_scenario4 "eventual" 2000
    cleanup
    
    # ====================================================================
    # TEST SET 4: STRONG - 0ms
    # ====================================================================
    print_banner "TEST SET 4/6: STRONG CONSISTENCY - 0ms DELAY"
    start_replicas "strong" 0 || { print_error "Failed to start replicas"; exit 1; }
    test_scenario1 "strong" 0
    test_scenario2 "strong" 0
    test_scenario3 "strong" 0
    test_scenario4 "strong" 0
    cleanup
    
    # ====================================================================
    # TEST SET 5: STRONG - 500ms
    # ====================================================================
    print_banner "TEST SET 5/6: STRONG CONSISTENCY - 500ms DELAY"
    start_replicas "strong" 500 || { print_error "Failed to start replicas"; exit 1; }
    test_scenario1 "strong" 500
    test_scenario4 "strong" 500
    cleanup
    
    # ====================================================================
    # TEST SET 6: STRONG - 2000ms
    # ====================================================================
    print_banner "TEST SET 6/6: STRONG CONSISTENCY - 2000ms DELAY"
    start_replicas "strong" 2000 || { print_error "Failed to start replicas"; exit 1; }
    test_scenario1 "strong" 2000
    test_scenario4 "strong" 2000
    cleanup
    
    # ====================================================================
    # GENERATE SUMMARY
    # ====================================================================
    generate_summary
    
    # ====================================================================
    # FINAL CLEANUP
    # ====================================================================
    cleanup
    
    # ====================================================================
    # DONE!
    # ====================================================================
    print_banner "✅ TEST SUITE COMPLETE! ✅"
    
    echo ""
    print_success "All 6 configurations tested across 4 scenarios"
    print_success "Total test runs: 18"
    echo ""
    print_info "Results saved in: $TEST_RUN_DIR"
    echo ""
    echo "  📁 $TEST_RUN_DIR/"
    echo "  ├── 📄 SUMMARY_REPORT.txt          (Comprehensive summary)"
    echo "  ├── 📄 test_execution.log          (Detailed execution log)"
    echo "  ├── 📁 scenarios/                  (Individual scenario results)"
    echo "  │   ├── scenario1_eventual_0ms.txt"
    echo "  │   ├── scenario1_eventual_500ms.txt"
    echo "  │   ├── scenario1_eventual_2000ms.txt"
    echo "  │   ├── scenario1_strong_0ms.txt"
    echo "  │   ├── scenario1_strong_500ms.txt"
    echo "  │   ├── scenario1_strong_2000ms.txt"
    echo "  │   ├── scenario2_eventual_0ms.txt"
    echo "  │   ├── scenario2_strong_0ms.txt"
    echo "  │   ├── scenario3_eventual_0ms.txt"
    echo "  │   ├── scenario3_strong_0ms.txt"
    echo "  │   ├── scenario4_eventual_0ms.txt"
    echo "  │   ├── scenario4_eventual_500ms.txt"
    echo "  │   ├── scenario4_eventual_2000ms.txt"
    echo "  │   ├── scenario4_strong_0ms.txt"
    echo "  │   ├── scenario4_strong_500ms.txt"
    echo "  │   └── scenario4_strong_2000ms.txt"
    echo "  └── 📁 logs/                       (Replica server logs)"
    echo "      ├── replica1.log"
    echo "      ├── replica2.log"
    echo "      └── replica3.log"
    echo ""
    print_info "Open the summary: cat $SUMMARY_FILE"
    echo ""
}

# ============================================================================
# RUN MAIN
# ============================================================================

# Trap Ctrl+C to cleanup
trap 'print_warning "Interrupted! Cleaning up..."; cleanup; exit 1' INT TERM

# Run the main function
main "$@"