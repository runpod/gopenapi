#!/bin/bash

echo "Running GopenAPI vs Stock HTTP ServeMux Benchmarks"
echo "=================================================="
echo ""

echo "Running correctness tests first..."
go test -run=TestBenchmarkCorrectness -v
if [ $? -ne 0 ]; then
    echo "Correctness tests failed!"
    exit 1
fi

echo ""
echo "Running performance benchmarks..."
echo ""

# Run benchmarks with multiple iterations for more stable results
go test -bench=. -benchmem -count=3 -run=^$ -benchtime=1s

echo ""
echo "Benchmark completed! Check BENCHMARK_RESULTS.md for detailed analysis." 