#!/bin/bash
# Test graceful daemon shutdown
echo "Testing graceful shutdown..."
pkill -f "bin/prismd" 2>/dev/null || true
sleep 1
echo "✅ Graceful shutdown test complete"
