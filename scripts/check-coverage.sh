#!/bin/bash

# CloudWorkstation coverage enforcement script
# Checks test coverage meets minimum requirements

set -e

echo "📊 Checking CloudWorkstation test coverage requirements..."

# Color functions
red() { echo -e "\033[31m$*\033[0m"; }
green() { echo -e "\033[32m$*\033[0m"; }
yellow() { echo -e "\033[33m$*\033[0m"; }
blue() { echo -e "\033[34m$*\033[0m"; }

# Coverage thresholds
MIN_TOTAL_COVERAGE=60    # Minimum total coverage percentage
MIN_PKG_COVERAGE=40      # Minimum per-package coverage percentage
EXCLUDE_PACKAGES="cmd/cws-gui"  # Packages to exclude from coverage requirements

# Generate coverage report
echo "Generating coverage report..."
go test ./... -coverprofile=total-coverage.out -covermode=atomic > /dev/null 2>&1

if [ ! -f "total-coverage.out" ]; then
    red "❌ Failed to generate coverage report"
    exit 1
fi

# Get total coverage
total_coverage=$(go tool cover -func=total-coverage.out | grep "total:" | awk '{print $3}' | sed 's/%//')
total_coverage_int=$(printf "%.0f" "$total_coverage")

echo ""
blue "Coverage Report Summary:"
echo "========================="

# Check total coverage
if (( $(echo "$total_coverage_int >= $MIN_TOTAL_COVERAGE" | bc -l) )); then
    green "✓ Total coverage: $total_coverage% (>= $MIN_TOTAL_COVERAGE%)"
    total_pass=true
else
    red "✗ Total coverage: $total_coverage% (< $MIN_TOTAL_COVERAGE%)"
    total_pass=false
fi

echo ""
echo "Package Coverage Details:"
echo "========================="

# Check per-package coverage
failed_packages=0
go tool cover -func=total-coverage.out | grep -v "total:" | while read line; do
    if [[ $line =~ ^([^:]+):[0-9]+:[[:space:]]*[^[:space:]]+[[:space:]]+[0-9]+\.[0-9]+% ]]; then
        # Skip individual function lines, we want package summaries
        continue
    fi
    
    # Package summary lines
    if [[ $line =~ ([0-9]+\.[0-9]+)% ]]; then
        coverage="${BASH_REMATCH[1]}"
        coverage_int=$(printf "%.0f" "$coverage")
        package=$(echo "$line" | awk '{print $1}' | sed 's|github.com/scttfrdmn/cloudworkstation/||')
        
        # Skip excluded packages
        skip=false
        for exclude in $EXCLUDE_PACKAGES; do
            if [[ $package == *"$exclude"* ]]; then
                yellow "- $package: $coverage% (excluded)"
                skip=true
                break
            fi
        done
        
        if [ "$skip" = false ]; then
            if (( $(echo "$coverage_int >= $MIN_PKG_COVERAGE" | bc -l) )); then
                green "✓ $package: $coverage%"
            else
                red "✗ $package: $coverage% (< $MIN_PKG_COVERAGE%)"
                ((failed_packages++))
            fi
        fi
    fi
done

# Note: The while loop runs in a subshell, so we need to check coverage again
pkg_failures=$(go tool cover -func=total-coverage.out | grep -v "total:" | grep -E "[0-9]+\.[0-9]+%" | awk -v min="$MIN_PKG_COVERAGE" -v exclude="$EXCLUDE_PACKAGES" '
BEGIN { failures = 0 }
{
    # Extract coverage percentage
    if (match($0, /([0-9]+\.[0-9]+)%/, arr)) {
        coverage = arr[1]
        package = $1
        gsub(/github\.com\/scttfrdmn\/cloudworkstation\//, "", package)
        
        # Check if package should be excluded
        excluded = 0
        split(exclude, exclude_list, " ")
        for (i in exclude_list) {
            if (index(package, exclude_list[i]) > 0) {
                excluded = 1
                break
            }
        }
        
        if (!excluded && coverage < min) {
            failures++
        }
    }
}
END { print failures }
')

echo ""
echo "Coverage Summary:"
echo "=================="
echo "Total Coverage: $total_coverage%"
echo "Minimum Required: $MIN_TOTAL_COVERAGE%"
echo "Failed Packages: $pkg_failures"

# Generate HTML report for detailed analysis
go tool cover -html=total-coverage.out -o coverage.html
echo ""
green "📋 Detailed coverage report: coverage.html"

# Final result
echo ""
if [ "$total_pass" = true ] && [ "$pkg_failures" -eq 0 ]; then
    green "🎉 All coverage requirements met!"
    echo ""
    echo "✅ Total coverage: $total_coverage% >= $MIN_TOTAL_COVERAGE%"
    echo "✅ All packages meet minimum coverage requirements"
    exit 0
else
    red "❌ Coverage requirements not met"
    echo ""
    if [ "$total_pass" = false ]; then
        echo "❌ Total coverage: $total_coverage% < $MIN_TOTAL_COVERAGE%"
    fi
    if [ "$pkg_failures" -gt 0 ]; then
        echo "❌ $pkg_failures packages below minimum coverage ($MIN_PKG_COVERAGE%)"
    fi
    echo ""
    echo "💡 Tips to improve coverage:"
    echo "  - Add unit tests for uncovered functions"
    echo "  - Test error handling paths"
    echo "  - Add integration tests for complex workflows"
    echo "  - Use 'go test -cover -v ./pkg/...' to see detailed coverage"
    exit 1
fi