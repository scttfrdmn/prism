# Batch Invitation System Performance Testing Guide

This document provides information about the performance testing framework for the CloudWorkstation batch invitation system and how to interpret the results.

## Overview

The batch invitation system performance tests evaluate:

1. **Throughput**: How many invitations can be processed per second
2. **Scalability**: How performance changes with increasing batch sizes
3. **Resource Usage**: CPU and memory consumption under load
4. **Concurrency Efficiency**: Performance gains from parallel processing

## Running the Tests

### Quick Start

Run the full performance test suite:

```bash
# Make the script executable
chmod +x ./scripts/run_performance_tests.sh

# Run the performance tests
./scripts/run_performance_tests.sh
```

Results will be saved to `./test_results/performance_YYYYMMDD_HHMMSS/`.

### Individual Tests

You can also run specific benchmarks:

```bash
# Run all batch invitation benchmarks
go test -bench=. -benchmem ./pkg/profile -run=^$

# Test batch creation with specific parameters
go test -bench=BenchmarkBatchInvitationCreation -benchmem ./pkg/profile -run=^$

# Test import/export operations
go test -bench=BenchmarkBatchInvitationImport -benchmem ./pkg/profile -run=^$

# Test device operations
go test -bench=BenchmarkDeviceOperations -benchmem ./pkg/profile -run=^$
```

### Profiling

For detailed profiling information:

```bash
# CPU profiling
go test -bench=BenchmarkBatchInvitationCreation -cpuprofile=cpu.out ./pkg/profile -run=^$

# Memory profiling
go test -bench=BenchmarkBatchInvitationCreation -memprofile=mem.out ./pkg/profile -run=^$

# Analyze profile data
go tool pprof -http=:8080 cpu.out
```

## Test Components

### 1. Batch Invitation Creation Tests

Measures performance of creating invitations in batches:

- Tests with batch sizes: 1, 10, 50, 100
- Tests with concurrency levels: 1, 2, 5, 10, 20
- Measures operations/second and memory allocation

Expected results:
- Small batches (1-10): Fast completion, low memory usage
- Medium batches (50): Good throughput with moderate resources
- Large batches (100+): Higher throughput with increased resource usage

### 2. CSV Import/Export Tests

Evaluates the performance of reading and writing CSV files:

- Tests with CSV sizes: 10, 100, 500, 1000 rows
- Measures import and export speeds separately
- Analyzes memory allocation patterns

Expected results:
- Small CSVs (10-100): Near-instant processing
- Medium CSVs (500): Good performance with minimal resource usage
- Large CSVs (1000+): Linear scaling with size

### 3. Device Operation Tests

Tests batch device management operations:

- Tests with device counts: 1, 10, 50, 100
- Tests with concurrency levels: 1, 5, 10
- Focuses on revocation operations (most common)

Expected results:
- Small batches (1-10): Fast operation, minimal overhead
- Medium batches (50): Good throughput with appropriate concurrency
- Large batches (100+): Benefits from higher concurrency settings

## Interpreting Results

### Summary Report

The summary report provides a high-level overview of test results:

```
Batch Creation Performance:
- Size_10_Concurrency_5: 1200 ops/sec, 8500 B/op
- Size_50_Concurrency_10: 320 ops/sec, 42500 B/op
...

CSV Import Performance:
- Import_Size_100: 450 ops/sec, 12800 B/op
...
```

### Key Metrics

1. **Operations per Second**: Higher is better. Shows how many batch operations can be completed per second.
2. **Bytes per Operation**: Lower is better. Indicates memory efficiency.
3. **Allocs per Operation**: Lower is better. Shows how many memory allocations occur.

### Concurrency Effectiveness

To measure how well the system scales with concurrency, compare results with different concurrency settings:

- **Linear Scaling**: When doubling concurrency nearly doubles throughput
- **Diminishing Returns**: When adding more concurrency shows minimal improvement
- **Negative Scaling**: When too much concurrency degrades performance

Optimal concurrency typically varies by:
- Hardware capabilities (CPU cores, memory)
- Network conditions
- I/O performance

### Profiling Results

CPU profiles reveal:
- Hot spots in the code
- Excessive time in particular functions
- Lock contention issues

Memory profiles show:
- Where memory is allocated
- Potential memory leaks
- Opportunities for reusing memory

## Performance Tuning Recommendations

Based on benchmark results, consider these optimizations:

### Batch Size Tuning

- **Small Organizations (< 100 users)**: Use batch sizes of 10-50
- **Medium Organizations (100-1000 users)**: Use batch sizes of 50-200
- **Large Organizations (1000+ users)**: Use batch sizes of 200-500 with increased timeout settings

### Concurrency Settings

- **Low-power Systems**: Set concurrency to match available CPU cores
- **Standard Workstations**: Set concurrency to CPU cores + 2
- **High-performance Servers**: Set concurrency to 2 * CPU cores

Adjust these values in configuration:
```bash
cws profiles invitations config set defaultConcurrency 8
```

### Memory Optimization

For large batch processing, increase available memory:

```bash
# Run with increased memory limit
GOMEMLIMIT=1024MiB ./your_application
```

## Common Issues and Solutions

| Issue | Symptoms | Solution |
|-------|----------|----------|
| Slow CSV Import | Import speed < 100 ops/sec | Check file format, use smaller batches |
| High Memory Usage | > 100MB for 1000 invitations | Reduce batch size or process in chunks |
| Poor Concurrency Scaling | No improvement with more workers | Check for lock contention in profiling |
| Network Timeouts | Failures in device operations | Increase timeouts, reduce batch size |

## Benchmarking Your Environment

To establish baseline performance for your specific environment:

1. Run the performance tests on a representative system
2. Record the results in a baseline document
3. Run the same tests periodically to detect performance regressions
4. Compare results across different environments to identify bottlenecks

## Advanced Analysis

For deeper analysis of performance issues:

```bash
# Generate trace data
go test -bench=BenchmarkBatchInvitationCreation -trace=trace.out ./pkg/profile -run=^$

# View trace in browser
go tool trace trace.out
```

This provides a visual timeline of goroutine execution, helping identify:
- Lock contention
- Goroutine blocking
- GC pauses
- Scheduler delays

## Conclusion

Regular performance testing helps ensure the batch invitation system remains efficient as the codebase evolves. Use the provided tools and metrics to identify bottlenecks and optimize for your specific usage patterns.