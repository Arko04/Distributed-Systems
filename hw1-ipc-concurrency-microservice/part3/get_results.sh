#!/bin/bash

OUTPUT_FILE="results.txt"

# Clear previous results
> "$OUTPUT_FILE"

echo "=== Part 3: Compute Service Results ===" >> "$OUTPUT_FILE"
echo "Timestamp: $(date)" >> "$OUTPUT_FILE"
echo "" >> "$OUTPUT_FILE"

# Health check
echo "--- Health Check ---" >> "$OUTPUT_FILE"
curl -s http://localhost:8080/health >> "$OUTPUT_FILE"
echo -e "\n" >> "$OUTPUT_FILE"

# Arithmetic operations
echo "--- Addition: 8 + 3 ---" >> "$OUTPUT_FILE"
curl -s "http://localhost:8080/compute?op=add&a=8&b=3" >> "$OUTPUT_FILE"
echo -e "\n" >> "$OUTPUT_FILE"

echo "--- Subtraction: 10 - 4 ---" >> "$OUTPUT_FILE"
curl -s "http://localhost:8080/compute?op=sub&a=10&b=4" >> "$OUTPUT_FILE"
echo -e "\n" >> "$OUTPUT_FILE"

echo "--- Multiplication: 6 * 7 ---" >> "$OUTPUT_FILE"
curl -s "http://localhost:8080/compute?op=mul&a=6&b=7" >> "$OUTPUT_FILE"
echo -e "\n" >> "$OUTPUT_FILE"

echo "--- Division: 20 / 4 ---" >> "$OUTPUT_FILE"
curl -s "http://localhost:8080/compute?op=div&a=20&b=4" >> "$OUTPUT_FILE"
echo -e "\n" >> "$OUTPUT_FILE"

# Error cases
echo "--- Division by zero (10/0) ---" >> "$OUTPUT_FILE"
curl -s "http://localhost:8080/compute?op=div&a=10&b=0" >> "$OUTPUT_FILE"
echo -e "\n" >> "$OUTPUT_FILE"

echo "--- Invalid operation (mod) ---" >> "$OUTPUT_FILE"
curl -s "http://localhost:8080/compute?op=mod&a=5&b=2" >> "$OUTPUT_FILE"
echo -e "\n" >> "$OUTPUT_FILE"

echo "--- Missing parameters (only op) ---" >> "$OUTPUT_FILE"
curl -s "http://localhost:8080/compute?op=add&a=5" >> "$OUTPUT_FILE"
echo -e "\n" >> "$OUTPUT_FILE"

echo "--- Non-numeric a ---" >> "$OUTPUT_FILE"
curl -s "http://localhost:8080/compute?op=add&a=abc&b=2" >> "$OUTPUT_FILE"
echo -e "\n" >> "$OUTPUT_FILE"

echo "Results saved to $OUTPUT_FILE"