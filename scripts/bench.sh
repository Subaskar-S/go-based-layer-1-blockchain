#!/bin/bash

# Benchmark script for Layer-1 blockchain

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo "Running Layer-1 Blockchain Benchmarks"
echo "======================================"

cd "$ROOT_DIR"

# Run Go benchmarks
echo ""
echo "Running Go benchmarks..."
go test -bench=. -benchmem -benchtime=10s ./... | tee benchmarks.txt

echo ""
echo "======================================"
echo "Benchmark Results Summary"
echo "======================================"

# Parse and display key metrics
if [ -f benchmarks.txt ]; then
    echo ""
    echo "Transaction Processing:"
    grep -i "BenchmarkTx" benchmarks.txt || echo "  No tx benchmarks found"
    
    echo ""
    echo "Block Processing:"
    grep -i "BenchmarkBlock" benchmarks.txt || echo "  No block benchmarks found"
    
    echo ""
    echo "VM Execution:"
    grep -i "BenchmarkVM" benchmarks.txt || echo "  No VM benchmarks found"
    
    echo ""
    echo "State Operations:"
    grep -i "BenchmarkState" benchmarks.txt || echo "  No state benchmarks found"
fi

echo ""
echo "Full results saved to: benchmarks.txt"

