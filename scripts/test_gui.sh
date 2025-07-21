#!/bin/bash

# Test script for CloudWorkstation GUI

# Set error handling
set -e

# Print test header
echo "==== CloudWorkstation GUI Tests ===="
echo "Running tests for the GUI components"
echo "====================================="

# Check if we're in CI environment
if [ -n "$CI" ]; then
  echo "Detected CI environment. Using headless mode."
  # Set environment variables for headless testing
  export FYNE_THEME=light
  export FYNE_SCALE=1
fi

# Create directory for test results if it doesn't exist
mkdir -p test_results

# Go to project root to ensure paths are correct
cd "$(dirname "$0")/.."

# Run the GUI tests
echo "Running GUI tests..."
go test -v ./cmd/cws-gui/tests/... -count=1 | tee test_results/gui_test_results.log

# Check exit status
if [ "${PIPESTATUS[0]}" -eq 0 ]; then
  echo "✅ GUI tests passed!"
else
  echo "❌ GUI tests failed!"
  exit 1
fi

# Run with coverage
echo "Running GUI tests with coverage..."
go test -v ./cmd/cws-gui/tests/... -coverprofile=test_results/gui_coverage.out

# Generate coverage report
go tool cover -html=test_results/gui_coverage.out -o test_results/gui_coverage.html
echo "Coverage report generated at test_results/gui_coverage.html"

echo "==== GUI Tests Complete ===="