#!/bin/bash

# Run all TUI component tests
echo "Running TUI component tests..."

# Navigate to the test directory
cd "$(dirname "$0")"

# Run tests with verbose output and color
go test -v -count=1 ./...

# Check test result
if [ $? -eq 0 ]; then
  echo -e "\n\033[32mAll TUI component tests passed!\033[0m"
else
  echo -e "\n\033[31mSome TUI component tests failed\033[0m"
  exit 1
fi