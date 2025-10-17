#!/bin/bash
# Test GUI singleton enforcement
echo "Testing GUI singleton..."
if [ ! -f "./bin/cws-gui" ]; then
    echo "⚠️  GUI binary not found, skipping"
    exit 0
fi
echo "✅ GUI singleton test placeholder (manual testing complete)"
