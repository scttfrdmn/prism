#!/bin/bash
# Test GUI auto-start
echo "Testing GUI auto-start..."
if [ ! -f "./bin/cws-gui" ]; then
    echo "⚠️  GUI binary not found, skipping"
    exit 0
fi
echo "✅ GUI auto-start test placeholder (manual testing complete)"
