#!/bin/bash
#
# Part 3 Testing Script - Final Version
# Tests the Dockerized arithmetic microservice for:
# - missing parameters
# - invalid op
# - non-numeric a or b
# - division/modulo by zero
# - wrong HTTP method
#

set -euo pipefail

###############################################
# Color definitions (ANSI-safe)
###############################################
RED=$(printf '\033[0;31m')
GREEN=$(printf '\033[0;32m')
YELLOW=$(printf '\033[1;33m')
BLUE=$(printf '\033[0;34m')
NC=$(printf '\033[0m')

###############################################
# Configuration
###############################################
IMAGE_NAME="compute-service"
CONTAINER_NAME="compute-container"
HOST_PORT="8080"
BASE_URL="http://localhost:${HOST_PORT}"
LOG_DIR="logs_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$LOG_DIR"

###############################################
# Utility helpers
###############################################
print_section() { echo -e "\n${BLUE}========== $1 ==========${NC}"; }
print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_error()   { echo -e "${RED}✗ $1${NC}"; }
print_info()    { echo -e "${YELLOW}→ $1${NC}"; }

save_log() {
    local name=$1
    echo "$2" > "${LOG_DIR}/${name}.log"
}

###############################################
# Phase 1: Environment checks
###############################################
check_docker() {
    print_section "Checking Docker Availability"
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker is not running."
        exit 1
    fi
    print_success "Docker daemon is running"
}

validate_dockerfile() {
    print_section "Validating Dockerfile"
    if [[ ! -f Dockerfile ]]; then
        print_error "No Dockerfile found in current directory."
        exit 1
    fi

    if ! grep -qi '^FROM' Dockerfile; then
        print_error "Dockerfile missing a FROM statement — may be invalid."
        exit 1
    fi
    print_success "Dockerfile appears valid"
}

###############################################
# Phase 2: Container lifecycle
###############################################
clean_container() {
    print_section "Cleaning Previous Containers"
    if docker ps -aq -f name="^${CONTAINER_NAME}$" >/dev/null 2>&1; then
        docker stop "${CONTAINER_NAME}" >/dev/null 2>&1 || true
        docker rm "${CONTAINER_NAME}" >/dev/null 2>&1 || true
        print_success "Cleaned old container"
    else
        print_info "No previous container found"
    fi
}

build_image() {
    print_section "Building Docker Image"
    if docker build -t "${IMAGE_NAME}" .; then
        print_success "Image built successfully"
    else
        print_error "Image build failed"
        exit 1
    fi
}

run_container() {
    print_section "Starting Container"
    if docker run -d -p "${HOST_PORT}:8080" --name "${CONTAINER_NAME}" "${IMAGE_NAME}" >/dev/null; then
        print_success "Container started"
    else
        print_error "Failed to start container"
        exit 1
    fi
}

wait_for_service() {
    print_section "Waiting for Service Startup"
    local attempt=1
    while [[ $attempt -le 30 ]]; do
        if curl -s -f "${BASE_URL}/health" >/dev/null 2>&1; then
            print_success "Service responsive (after ${attempt} attempts)"
            return
        fi
        attempt=$((attempt + 1))
        sleep 1
    done
    print_error "Service failed to start"
    docker logs "${CONTAINER_NAME}" || true
    exit 1
}

###############################################
# Phase 3: Test Case Helpers
###############################################
pretty_json() { jq '.' 2>/dev/null || echo "$1"; }

# name, url, expected_code, expected_substring
test_endpoint() {
    local name=$1 url=$2 expected_code=$3 expected_substring=$4

    echo -n "  Test: ${name}... "

    local response http_code body
    response=$(curl -s -w "\n%{http_code}" "$url" || true)
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [[ "$http_code" == "$expected_code" ]]; then
        if echo "$body" | grep -qi "$expected_substring"; then
            print_success "$name (HTTP $http_code)"
            save_log "$name" "$body"
        else
            print_error "$name: missing '$expected_substring' in body"
            echo "      Body:"
            pretty_json "$body" | sed 's/^/        /'
            save_log "${name}_FAIL" "$body"
            return 1
        fi
    else
        print_error "$name: expected $expected_code, got $http_code"
        echo "      Body:"
        pretty_json "$body" | sed 's/^/        /'
        save_log "${name}_FAIL" "$body"
        return 1
    fi
}

###############################################
# Phase 4: Test Suite (covers all required errors)
###############################################
run_tests() {
    print_section "Running API Test Suite"
    local failed=0

    # Health check
    test_endpoint "Health Check"          "${BASE_URL}/health"                     200 '"status":"ok"'              || failed=$((failed+1))

    # Successful operations
    test_endpoint "Addition"              "${BASE_URL}/compute?op=add&a=8&b=3"     200 '"result":11'                || failed=$((failed+1))
    test_endpoint "Subtraction"           "${BASE_URL}/compute?op=sub&a=10&b=4"    200 '"result":6'                 || failed=$((failed+1))
    test_endpoint "Multiplication"        "${BASE_URL}/compute?op=mul&a=6&b=7"     200 '"result":42'                || failed=$((failed+1))
    test_endpoint "Division"              "${BASE_URL}/compute?op=div&a=20&b=4"    200 '"result":5'                 || failed=$((failed+1))
    test_endpoint "Modulo"                "${BASE_URL}/compute?op=mod&a=20&b=3"    200 '"result":2'                 || failed=$((failed+1))

    # 1) Missing parameters
    test_endpoint "Missing Params"        "${BASE_URL}/compute?op=add&a=5"         400 "missing required parameters" || failed=$((failed+1))

    # 2) Invalid op
    test_endpoint "Invalid Operation"     "${BASE_URL}/compute?op=pow&a=2&b=3"     400 "unsupported operation"      || failed=$((failed+1))

    # 3) Non-numeric a/b
    test_endpoint "Invalid Number A"      "${BASE_URL}/compute?op=add&a=abc&b=2"   400 "invalid numeric parameters" || failed=$((failed+1))
    test_endpoint "Invalid Number B"      "${BASE_URL}/compute?op=add&a=2&b=xyz"   400 "invalid numeric parameters" || failed=$((failed+1))

    # 4) Division by zero
    test_endpoint "Division by Zero"      "${BASE_URL}/compute?op=div&a=10&b=0"    400 "division by zero"           || failed=$((failed+1))

    #    Modulo by zero (هم خانواده تقسیم بر صفر)
    test_endpoint "Modulo by Zero"        "${BASE_URL}/compute?op=mod&a=10&b=0"    400 "modulo by zero"             || failed=$((failed+1))

    # 5) Wrong HTTP method
    echo -n "  Test: Wrong HTTP method (POST /compute)... "
    local resp code body
    resp=$(curl -s -X POST -w "\n%{http_code}" "${BASE_URL}/compute?op=add&a=1&b=2")
    code=$(echo "$resp" | tail -n1)
    body=$(echo "$resp" | sed '$d')
    if [[ "$code" -eq 405 ]] && echo "$body" | grep -qi "method not allowed"; then
        print_success "Wrong method correctly returned 405 + 'method not allowed'"
        save_log "WrongMethod" "$body"
    else
        print_error "Wrong method: expected 405 and 'method not allowed', got $code"
        pretty_json "$body" | sed 's/^/        /'
        save_log "WrongMethod_FAIL" "$body"
        failed=$((failed+1))
    fi

    echo
    if [[ $failed -eq 0 ]]; then
        print_success "All tests passed (including all required error cases)."
    else
        print_error "$failed test(s) failed. Check logs in $LOG_DIR."
    fi
}

###############################################
# Phase 5: Logs & Cleanup
###############################################
show_logs() {
    print_section "Docker Logs (last 20 lines)"
    docker logs "${CONTAINER_NAME}" --tail 20 || true
}

cleanup() {
    print_section "Cleanup"
    docker stop "${CONTAINER_NAME}" >/dev/null 2>&1 || true
    docker rm "${CONTAINER_NAME}" >/dev/null 2>&1 || true
    print_success "Container stopped & removed"
}

###############################################
# Main Entry Point
###############################################
main() {
    print_section "Part 3 Automated Test Framework"

    check_docker
    validate_dockerfile
    clean_container
    build_image
    run_container
    wait_for_service
    run_tests
    show_logs

    echo
    read -p "Remove container? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cleanup
    else
        print_info "Container left running. Stop manually with:"
        echo "        docker stop ${CONTAINER_NAME} && docker rm ${CONTAINER_NAME}"
    fi

    print_info "Logs saved to $LOG_DIR/"
}

###############################################
# Optional arguments
###############################################
if [[ "${1:-}" == "--clean" ]]; then
    clean_container
    exit 0
elif [[ "${1:-}" == "--help" ]]; then
    echo "Usage: $0 [--clean|--help]"
    exit 0
else
    main
fi
