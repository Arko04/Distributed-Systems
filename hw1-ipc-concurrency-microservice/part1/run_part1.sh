#!/usr/bin/env bash
set -euo pipefail

INTERFACE_BIN="./interface"
LOG_FILE="part1_test.log"

# colors
RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
BLUE="\033[34m"
RESET="\033[0m"

log() {
  printf "%b\n" "$1" | tee -a "${LOG_FILE}"
}

section() {
  local title="$1"
  log ""
  log "=================================================="
  log ">>> ${title}"
  log "=================================================="
}

# header
printf "%b\n" "${BLUE}==== Part 1 Test — Interface Only (Assumes WORKER is already running) ====${RESET}"
echo "All output will be stored in ${LOG_FILE}."
echo

rm -f "${LOG_FILE}"

# build interface
section "Build interface"

if [[ -f "${INTERFACE_BIN}" ]]; then
  log "Removing previous interface binary..."
  rm -f "${INTERFACE_BIN}"
fi

log "Building interface.go ..."
if go build -o "${INTERFACE_BIN}" interface.go 2>&1 | tee -a "${LOG_FILE}"; then
  printf "%b\n" "${GREEN}✓ interface build succeeded.${RESET}" | tee -a "${LOG_FILE}"
else
  printf "%b\n" "${RED}✗ interface build failed. Tests aborted.${RESET}" | tee -a "${LOG_FILE}"
  exit 1
fi

# ---------- Section 1: Normal operations ----------
section "Section 1 - Normal operations (ADD, SUB, MUL, DIV, MOD)"

{
  echo "ADD 5 7"      # OK 12
  echo "SUB 10 3"     # OK 7
  echo "MUL 4 2.5"    # OK 10
  echo "DIV 9 2"      # OK 4.5
  echo "MOD 10 3"     # OK 1
  echo "exit"
} | "${INTERFACE_BIN}" 2>&1 | tee -a "${LOG_FILE}"

# ---------- Section 2: Unknown operation ----------
section "Section 2 - Unknown operation"

{
  echo "FOO 1 2"
  echo "exit"
} | "${INTERFACE_BIN}" 2>&1 | tee -a "${LOG_FILE}"

# ---------- Section 3: Wrong argument count ----------
section "Section 3 - Wrong argument count"

{
  echo "ADD 1"
  echo "SUB 1 2 3"
  echo "MUL"
  echo "exit"
} | "${INTERFACE_BIN}" 2>&1 | tee -a "${LOG_FILE}"

# ---------- Section 4: Invalid numeric input ----------
section "Section 4 - Invalid numeric input"

{
  echo "ADD a 2"
  echo "DIV 10 b"
  echo "MUL one two"
  echo "exit"
} | "${INTERFACE_BIN}" 2>&1 | tee -a "${LOG_FILE}"

# ---------- Section 5: Division by zero ----------
section "Section 5 - Division by zero"

{
  echo "DIV 10 0"
  echo "MOD 7 0"
  echo "exit"
} | "${INTERFACE_BIN}" 2>&1 | tee -a "${LOG_FILE}"

# ---------- Section 6: Simulating worker/pipe disconnection ----------
section "Section 6 - Simulating worker/pipe disconnection"

printf "%b\n" "${YELLOW}The interface will send several requests now. Please go to the WORKER terminal and press Ctrl+C in the middle to simulate a crash.${RESET}" | tee -a "${LOG_FILE}"

{
  echo "ADD 1 1"
  echo "SUB 5 2"
  sleep 2
  echo "MUL 2 3"
  echo "DIV 4 2"
  echo "exit"
} | "${INTERFACE_BIN}" 2>&1 | tee -a "${LOG_FILE}"

section "End of tests"

printf "%b\n" "${GREEN}All scenarios executed. Full output saved to ${LOG_FILE}.${RESET}" | tee -a "${LOG_FILE}"
