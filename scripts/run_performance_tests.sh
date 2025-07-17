#!/bin/bash
# Comprehensive performance testing for batch invitation system

set -e

# Color codes for better readability
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Starting Batch Invitation System Performance Tests${NC}"

# Create results directory
RESULTS_DIR="./test_results/performance_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$RESULTS_DIR"
echo "Saving results to: ${RESULTS_DIR}"

# Function to run a test and save results
run_test() {
    local test_name=$1
    local test_args=$2
    local output_file="${RESULTS_DIR}/${test_name}.txt"
    
    echo -e "${BLUE}Running test: ${test_name}${NC}"
    echo "Command: go test $test_args"
    
    # Run the test and capture output
    echo "=== Test: $test_name ===" > "$output_file"
    echo "Command: go test $test_args" >> "$output_file"
    echo "Started: $(date)" >> "$output_file"
    echo "=======================================" >> "$output_file"
    
    if go test $test_args >> "$output_file" 2>&1; then
        echo -e "${GREEN}✓ Test passed${NC}"
        echo "=======================================" >> "$output_file"
        echo "Status: PASSED" >> "$output_file"
    else
        echo -e "${RED}✗ Test failed${NC}"
        echo "=======================================" >> "$output_file"
        echo "Status: FAILED" >> "$output_file"
    fi
    
    echo "Ended: $(date)" >> "$output_file"
}

# 1. Run standard benchmarks
echo -e "\n${YELLOW}Running Standard Benchmarks${NC}"
run_test "standard_benchmarks" "-bench=. -benchmem ./pkg/profile -run=^$ -timeout=30m"

# 2. Run focused batch creation benchmarks with various concurrency settings
echo -e "\n${YELLOW}Running Batch Creation Concurrency Tests${NC}"
run_test "batch_creation_concurrency" "-bench=BenchmarkBatchInvitationCreation -benchmem ./pkg/profile -run=^$ -timeout=15m"

# 3. Run import/export benchmarks with various CSV sizes
echo -e "\n${YELLOW}Running Import/Export Performance Tests${NC}"
run_test "batch_import_export" "-bench=BenchmarkBatchInvitationImport -benchmem ./pkg/profile -run=^$ -timeout=15m"

# 4. Run device operation benchmarks
echo -e "\n${YELLOW}Running Device Operation Tests${NC}"
run_test "device_operations" "-bench=BenchmarkDeviceOperations -benchmem ./pkg/profile -run=^$ -timeout=15m"

# 5. Run CPU profiling test on batch creation (most intensive operation)
echo -e "\n${YELLOW}Running CPU Profiling on Batch Creation${NC}"
CPU_PROFILE_PATH="${RESULTS_DIR}/cpu_profile.out"
run_test "cpu_profiling" "-bench=BenchmarkBatchInvitationCreation -cpuprofile=${CPU_PROFILE_PATH} ./pkg/profile -run=^$ -timeout=15m"

# 6. Run memory profiling test on batch creation
echo -e "\n${YELLOW}Running Memory Profiling on Batch Creation${NC}"
MEM_PROFILE_PATH="${RESULTS_DIR}/mem_profile.out"
run_test "mem_profiling" "-bench=BenchmarkBatchInvitationCreation -memprofile=${MEM_PROFILE_PATH} ./pkg/profile -run=^$ -timeout=15m"

# 7. Generate profile reports if pprof is available
if command -v go tool pprof &> /dev/null; then
    echo -e "\n${YELLOW}Generating Profile Reports${NC}"
    
    # Generate CPU profile report
    if [ -f "${CPU_PROFILE_PATH}" ]; then
        echo "Generating CPU profile report..."
        go tool pprof -text "${CPU_PROFILE_PATH}" > "${RESULTS_DIR}/cpu_profile_report.txt"
    fi
    
    # Generate Memory profile report
    if [ -f "${MEM_PROFILE_PATH}" ]; then
        echo "Generating Memory profile report..."
        go tool pprof -text "${MEM_PROFILE_PATH}" > "${RESULTS_DIR}/mem_profile_report.txt"
    fi
fi

# 8. Run scaling tests with increasing load
echo -e "\n${YELLOW}Running Scaling Tests${NC}"

# Test with different batch sizes to measure scaling
for size in 10 50 100 500 1000; do
    if [ $size -le 100 ]; then
        # For smaller sizes, use more iterations for better measurement
        count=10
    else
        # For larger sizes, use fewer iterations
        count=3
    fi
    
    echo -e "${BLUE}Testing batch size: ${size} with ${count} iterations${NC}"
    test_name="scaling_test_${size}"
    run_test "$test_name" "-bench=BenchmarkBatchInvitationCreation/Size_${size} -benchtime=${count}x -benchmem ./pkg/profile -run=^$ -timeout=15m"
done

# 9. Generate summary report
echo -e "\n${YELLOW}Generating Summary Report${NC}"
SUMMARY_PATH="${RESULTS_DIR}/summary.txt"

echo "Batch Invitation System Performance Test Summary" > "$SUMMARY_PATH"
echo "================================================" >> "$SUMMARY_PATH"
echo "Date: $(date)" >> "$SUMMARY_PATH"
echo "" >> "$SUMMARY_PATH"

echo "Test Results:" >> "$SUMMARY_PATH"
for result_file in "${RESULTS_DIR}"/*.txt; do
    if [ "$(basename "$result_file")" != "summary.txt" ]; then
        test_name=$(basename "$result_file" .txt)
        status=$(grep "Status:" "$result_file" | cut -d ' ' -f 2)
        echo "- $test_name: $status" >> "$SUMMARY_PATH"
    fi
done

echo "" >> "$SUMMARY_PATH"
echo "Performance Highlights:" >> "$SUMMARY_PATH"

# Extract key benchmark results for the summary
echo "Batch Creation Performance:" >> "$SUMMARY_PATH"
grep -h "BenchmarkBatchInvitationCreation/Size_" "${RESULTS_DIR}/standard_benchmarks.txt" | \
    sed 's/BenchmarkBatchInvitationCreation\///' | \
    awk '{ printf "- %s: %s ops/sec, %s\n", $1, $3, $4 }' >> "$SUMMARY_PATH"

echo "" >> "$SUMMARY_PATH"
echo "CSV Import Performance:" >> "$SUMMARY_PATH"
grep -h "BenchmarkBatchInvitationImport/Import_Size_" "${RESULTS_DIR}/batch_import_export.txt" | \
    sed 's/BenchmarkBatchInvitationImport\///' | \
    awk '{ printf "- %s: %s ops/sec, %s\n", $1, $3, $4 }' >> "$SUMMARY_PATH"

echo "" >> "$SUMMARY_PATH"
echo "Device Operation Performance:" >> "$SUMMARY_PATH"
grep -h "BenchmarkDeviceOperations/Revoke_Size_" "${RESULTS_DIR}/device_operations.txt" | \
    sed 's/BenchmarkDeviceOperations\///' | \
    awk '{ printf "- %s: %s ops/sec, %s\n", $1, $3, $4 }' >> "$SUMMARY_PATH"

echo -e "${GREEN}All performance tests completed!${NC}"
echo "Results saved to: ${RESULTS_DIR}"
echo "Summary report: ${SUMMARY_PATH}"