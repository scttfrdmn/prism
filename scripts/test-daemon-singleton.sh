#!/bin/bash
# Test daemon singleton enforcement

set -e

# Cleanup function
cleanup() {
    pkill -f "bin/cwsd" 2>/dev/null || true
    rm -f /tmp/daemon-singleton-*.log /tmp/daemon-singleton-*.pid
}

trap cleanup EXIT

echo "Testing daemon singleton enforcement..."

# Start first daemon
./bin/cwsd > /tmp/daemon-singleton-1.log 2>&1 &
PID1=$!
echo "Started first daemon (PID: $PID1)"
sleep 3

# Check first daemon is running
if ! ps -p $PID1 > /dev/null; then
    echo "❌ First daemon failed to start"
    exit 1
fi

# Start second daemon (should replace first)
./bin/cwsd > /tmp/daemon-singleton-2.log 2>&1 &
PID2=$!
echo "Started second daemon (PID: $PID2)"
sleep 3

# Check second daemon is running
if ! ps -p $PID2 > /dev/null; then
    echo "❌ Second daemon failed to start"
    exit 1
fi

# Check first daemon was shut down
if ps -p $PID1 > /dev/null 2>&1; then
    echo "❌ First daemon still running (singleton failed)"
    exit 1
fi

echo "✅ Daemon singleton enforcement working"
