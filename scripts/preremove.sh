#!/bin/bash
# Pre-removal script for CloudWorkstation Linux packages

# Stop daemon gracefully if it's running
if command -v cws >/dev/null 2>&1; then
    echo "Stopping CloudWorkstation daemon..."
    cws daemon stop 2>/dev/null || true
fi

echo "CloudWorkstation will be removed."
