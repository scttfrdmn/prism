#!/bin/bash

# benchmark-launch.sh - Performance benchmarking for CloudWorkstation launch operations
# Usage: ./benchmark-launch.sh [options]

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PRISM_BINARY="${PROJECT_ROOT}/bin/cws"
CWSD_BINARY="${PROJECT_ROOT}/bin/prismd"

# Default values
ITERATIONS=5
TEMPLATES="python-ml,r-research,ubuntu"
DRY_RUN="false"
VERBOSE="false"
OUTPUT_FILE=""
TIMEOUT="600" # 10 minutes

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
print_usage() {
    cat << EOF
CloudWorkstation Launch Performance Benchmark

Usage: $0 [OPTIONS]

Options:
    --iterations, -i    Number of iterations per template (default: 5)
    --templates, -t     Comma-separated list of templates to test (default: python-ml,r-research,ubuntu)
    --dry-run, -d       Don't actually launch instances, just time template resolution
    --verbose, -v       Verbose output with detailed timing
    --output, -o        Output results to file (JSON format)
    --timeout           Timeout for each launch in seconds (default: 600)
    --help, -h          Show this help message

Examples:
    $0 --iterations 10 --templates python-ml
    $0 --dry-run --verbose --output benchmark-results.json
    $0 -i 3 -t "r-research,ubuntu" --timeout 300

EOF
}

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $*${NC}" >&2
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $*${NC}" >&2
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $*${NC}" >&2
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --iterations|-i)
            ITERATIONS="$2"
            shift 2
            ;;
        --templates|-t)
            TEMPLATES="$2"
            shift 2
            ;;
        --dry-run|-d)
            DRY_RUN="true"
            shift
            ;;
        --verbose|-v)
            VERBOSE="true"
            shift
            ;;
        --output|-o)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --help|-h)
            print_usage
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            print_usage
            exit 1
            ;;
    esac
done

# Validation
if ! [[ "$ITERATIONS" =~ ^[0-9]+$ ]] || [ "$ITERATIONS" -lt 1 ]; then
    error "Iterations must be a positive integer"
    exit 1
fi

if ! [[ "$TIMEOUT" =~ ^[0-9]+$ ]] || [ "$TIMEOUT" -lt 30 ]; then
    error "Timeout must be at least 30 seconds"
    exit 1
fi

if [ ! -x "$PRISM_BINARY" ]; then
    error "CloudWorkstation CLI not found at $PRISM_BINARY. Run 'make build' first."
    exit 1
fi

if [ ! -x "$CWSD_BINARY" ]; then
    error "CloudWorkstation daemon not found at $CWSD_BINARY. Run 'make build' first."
    exit 1
fi

# Check if daemon is running
if ! pgrep -f "$CWSD_BINARY" >/dev/null; then
    log "Starting CloudWorkstation daemon..."
    "$CWSD_BINARY" &
    sleep 5
    
    # Wait for daemon to be ready
    for i in {1..10}; do
        if "$PRISM_BINARY" daemon status &>/dev/null; then
            break
        fi
        if [ "$i" -eq 10 ]; then
            error "Daemon failed to start within 10 seconds"
            exit 1
        fi
        sleep 1
    done
fi

log "Starting CloudWorkstation launch performance benchmark"
log "Configuration:"
log "  Iterations: $ITERATIONS"
log "  Templates: $TEMPLATES"
log "  Dry run: $DRY_RUN"
log "  Timeout: ${TIMEOUT}s"

# Convert templates to array
IFS=',' read -ra TEMPLATE_ARRAY <<< "$TEMPLATES"

# Initialize results
declare -A results
declare -A template_stats

start_benchmark=$(date +%s)

# Run benchmarks for each template
for template in "${TEMPLATE_ARRAY[@]}"; do
    log "Benchmarking template: $template"
    
    template_times=()
    failed_launches=0
    
    for i in $(seq 1 "$ITERATIONS"); do
        instance_name="benchmark-${template}-$(date +%s)-${i}"
        
        log "  Iteration $i/$ITERATIONS - Instance: $instance_name"
        
        start_time=$(date +%s.%N)
        
        if [ "$DRY_RUN" = "true" ]; then
            # Dry run - just resolve template
            if timeout "$TIMEOUT" "$PRISM_BINARY" templates info "$template" &>/dev/null; then
                end_time=$(date +%s.%N)
                duration=$(echo "$end_time - $start_time" | bc -l)
                template_times+=("$duration")
                
                if [ "$VERBOSE" = "true" ]; then
                    log "    Template resolution time: ${duration}s"
                fi
            else
                warn "    Template resolution failed for $template"
                ((failed_launches++))
            fi
        else
            # Full launch
            if timeout "$TIMEOUT" "$PRISM_BINARY" launch "$template" "$instance_name" --no-connect &>/dev/null; then
                end_time=$(date +%s.%N)
                duration=$(echo "$end_time - $start_time" | bc -l)
                template_times+=("$duration")
                
                if [ "$VERBOSE" = "true" ]; then
                    log "    Launch time: ${duration}s"
                fi
                
                # Clean up instance
                log "    Cleaning up instance: $instance_name"
                "$PRISM_BINARY" delete "$instance_name" --force &>/dev/null || warn "Failed to delete $instance_name"
            else
                warn "    Launch failed for $instance_name"
                ((failed_launches++))
                
                # Try to clean up anyway
                "$PRISM_BINARY" delete "$instance_name" --force &>/dev/null 2>&1 || true
            fi
        fi
        
        # Brief pause between iterations
        sleep 2
    done
    
    # Calculate statistics for this template
    if [ ${#template_times[@]} -gt 0 ]; then
        # Calculate mean
        mean=$(printf '%s\n' "${template_times[@]}" | awk '{sum+=$1} END {print sum/NR}')
        
        # Calculate min and max
        min=$(printf '%s\n' "${template_times[@]}" | sort -n | head -n1)
        max=$(printf '%s\n' "${template_times[@]}" | sort -n | tail -n1)
        
        # Calculate median
        median=$(printf '%s\n' "${template_times[@]}" | sort -n | awk 'NR==int((NR+1)/2)')
        
        results["${template}_mean"]=$mean
        results["${template}_min"]=$min
        results["${template}_max"]=$max
        results["${template}_median"]=$median
        results["${template}_failed"]=$failed_launches
        results["${template}_successful"]=$((ITERATIONS - failed_launches))
        
        log "  Results for $template:"
        log "    Successful: $((ITERATIONS - failed_launches))/$ITERATIONS"
        log "    Mean time: ${mean}s"
        log "    Min time: ${min}s" 
        log "    Max time: ${max}s"
        log "    Median time: ${median}s"
    else
        warn "  No successful launches for template: $template"
        results["${template}_mean"]="N/A"
        results["${template}_failed"]=$ITERATIONS
        results["${template}_successful"]=0
    fi
done

end_benchmark=$(date +%s)
total_benchmark_time=$((end_benchmark - start_benchmark))

# Print summary
echo
log "=== BENCHMARK SUMMARY ==="
log "Total benchmark time: ${total_benchmark_time}s"
log "Mode: $(if [ "$DRY_RUN" = "true" ]; then echo "Template resolution only"; else echo "Full launch"; fi)"

for template in "${TEMPLATE_ARRAY[@]}"; do
    echo
    log "Template: $template"
    log "  Success rate: ${results["${template}_successful"]}/$ITERATIONS"
    if [ "${results["${template}_mean"]}" != "N/A" ]; then
        log "  Average time: ${results["${template}_mean"]}s"
        log "  Best time: ${results["${template}_min"]}s"
        log "  Worst time: ${results["${template}_max"]}s"
        log "  Median time: ${results["${template}_median"]}s"
    fi
done

# Generate JSON output if requested
if [ -n "$OUTPUT_FILE" ]; then
    log "Writing results to: $OUTPUT_FILE"
    
    cat > "$OUTPUT_FILE" << EOF
{
  "benchmark_metadata": {
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "iterations": $ITERATIONS,
    "templates": [$(printf '"%s",' "${TEMPLATE_ARRAY[@]}" | sed 's/,$//')],
    "dry_run": $DRY_RUN,
    "timeout": $TIMEOUT,
    "total_duration_seconds": $total_benchmark_time
  },
  "results": {
EOF

    first_template=true
    for template in "${TEMPLATE_ARRAY[@]}"; do
        if [ "$first_template" = true ]; then
            first_template=false
        else
            echo "," >> "$OUTPUT_FILE"
        fi
        
        cat >> "$OUTPUT_FILE" << EOF
    "$template": {
      "successful_launches": ${results["${template}_successful"]},
      "failed_launches": ${results["${template}_failed"]},
      "mean_time_seconds": "${results["${template}_mean"]}",
      "min_time_seconds": "${results["${template}_min"]}",
      "max_time_seconds": "${results["${template}_max"]}",
      "median_time_seconds": "${results["${template}_median"]}"
    }
EOF
    done
    
    cat >> "$OUTPUT_FILE" << EOF
  }
}
EOF
    
    log "Results written to: $OUTPUT_FILE"
fi

# Performance recommendations
echo
log "=== PERFORMANCE RECOMMENDATIONS ==="

for template in "${TEMPLATE_ARRAY[@]}"; do
    if [ "${results["${template}_mean"]}" != "N/A" ]; then
        mean_time=$(echo "${results["${template}_mean"]}" | bc -l)
        
        # Thresholds for different modes
        if [ "$DRY_RUN" = "true" ]; then
            # Template resolution should be under 5 seconds
            threshold=5.0
            operation="template resolution"
        else
            # Full launch should be under 5 minutes
            threshold=300.0
            operation="launch"
        fi
        
        if (( $(echo "$mean_time > $threshold" | bc -l) )); then
            warn "$template average $operation time (${mean_time}s) exceeds recommended threshold (${threshold}s)"
            if [ "$DRY_RUN" = "false" ]; then
                log "  Consider enabling template caching or using pre-built AMIs"
            fi
        else
            log "$template $operation performance is within acceptable range"
        fi
    fi
done

log "Benchmark complete!"

exit 0