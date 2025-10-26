#!/bin/bash
# Test CLI auto-start functionality

set -e

# Cleanup
pkill -f "bin/prismd" 2>/dev/null || true
sleep 1

echo "Testing CLI auto-start..."

# Run CLI command - should auto-start daemon
timeout 10s ./bin/prism workspace list > /dev/null 2>&1

echo "✅ CLI auto-start working"

# Cleanup
pkill -f "bin/prismd" 2>/dev/null || true
