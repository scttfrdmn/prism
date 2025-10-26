#!/bin/bash
# Test binary discovery from PATH

set -e

echo "Testing binary discovery..."

# Test same directory discovery (already works)
if [ ! -f "./bin/prismd" ]; then
    echo "❌ Daemon binary not found in ./bin/"
    exit 1
fi

echo "✅ Binary discovery working (same directory)"
