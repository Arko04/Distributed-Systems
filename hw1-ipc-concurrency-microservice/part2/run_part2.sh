#!/usr/bin/env bash
set -euo pipefail

# --------------------------------------------------------------------
# Configuration
# --------------------------------------------------------------------
GOFILE="main.go"
BINARY="./part2_bench"
RESULT_CSV="part2_results.csv"

# Colors
RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
BLUE="\033[34m"
RESET="\033[0m"

log() {
  printf "%b\n" "$1"
}

section() {
  local title="$1"
  echo
  printf "%b\n" "${BLUE}==================================================${RESET}"
  printf "%b\n" "${BLUE}>>> ${title}${RESET}"
  printf "%b\n" "${BLUE}==================================================${RESET}"
}

# --------------------------------------------------------------------
# 1. Build
# --------------------------------------------------------------------
section "Building Part 2 benchmark program"

if [[ ! -f "${GOFILE}" ]]; then
  printf "%b\n" "${RED}Error: ${GOFILE} not found in current directory.${RESET}"
  printf "%b\n" "${YELLOW}Make sure you run this script from the folder containing ${GOFILE}.${RESET}"
  exit 1
fi

if [[ -f "${BINARY}" ]]; then
  log "Removing old binary ${BINARY} ..."
  rm -f "${BINARY}"
fi

log "Running: go build -o ${BINARY} ${GOFILE}"
if go build -o "${BINARY}" "${GOFILE}"; then
  printf "%b\n" "${GREEN}Build succeeded.${RESET}"
else
  printf "%b\n" "${RED}Build failed.${RESET}"
  exit 1
fi

# --------------------------------------------------------------------
# 2. Run benchmark
# --------------------------------------------------------------------
section "Running benchmark (this may take some seconds)"

if [[ -f "${RESULT_CSV}" ]]; then
  log "Removing old results file ${RESULT_CSV} ..."
  rm -f "${RESULT_CSV}"
fi

log "Running: ${BINARY} > ${RESULT_CSV}"
"${BINARY}" > "${RESULT_CSV}"

printf "%b\n" "${GREEN}Benchmark completed. Results saved to ${RESULT_CSV}.${RESET}"

# --------------------------------------------------------------------
# 3. Quick summary preview
# --------------------------------------------------------------------
section "Quick summary (first few lines of CSV)"

# Show header + first 10 lines
head -n 11 "${RESULT_CSV}"

echo
printf "%b\n" "${YELLOW}You can open ${RESULT_CSV} in Excel, Google Sheets, or process it with Python/R for graphs and deeper analysis.${RESET}"

section "Done"
printf "%b\n" "${GREEN}Part 2 tests executed successfully.${RESET}"
