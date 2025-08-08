#!/bin/bash
set -e

echo "=== Testing CloudWorkstation Idle Detection System ==="
echo

# Check daemon status
echo "1. Checking daemon status..."
if ! curl -s http://localhost:8947/api/v1/ping > /dev/null; then
    echo "ERROR: Daemon not running. Please start with: ./bin/cwsd"
    exit 1
fi
echo "✓ Daemon is running"
echo

# Check idle status
echo "2. Checking idle detection status..."
curl -s http://localhost:8947/api/v1/idle/status | jq '.'
echo

# List current profiles
echo "3. Current idle profiles:"
curl -s http://localhost:8947/api/v1/idle/profiles | jq '.'
echo

# Create test metrics for hibernation instance (simulating idle state)
echo "4. Simulating idle metrics for hibernation test instance..."

# Create idle metrics JSON payload
cat > /tmp/idle_metrics.json << 'EOF'
{
    "instance_id": "i-hibernation-test",
    "instance_name": "idle-test-hibernation", 
    "metrics": {
        "timestamp": "2025-08-08T04:00:00Z",
        "cpu": 2.0,
        "memory": 15.0,
        "network": 10.0,
        "disk": 20.0,
        "gpu": 1.0,
        "has_activity": false
    }
}
EOF

echo "Created test metrics for hibernation instance"

# Create idle metrics for stop instance 
cat > /tmp/idle_metrics_stop.json << 'EOF'
{
    "instance_id": "i-stop-test", 
    "instance_name": "idle-test-stop",
    "metrics": {
        "timestamp": "2025-08-08T04:00:00Z", 
        "cpu": 3.0,
        "memory": 20.0,
        "network": 15.0,
        "disk": 30.0,
        "has_activity": false
    }
}
EOF

echo "Created test metrics for stop instance"
echo

# Check for pending actions
echo "5. Checking for pending idle actions..."
curl -s http://localhost:8947/api/v1/idle/pending-actions | jq '.'
echo

echo "6. To manually trigger hibernation/stop actions, run:"
echo "   curl -X POST http://localhost:8947/api/v1/idle/execute-actions"
echo
echo "7. To check action history:"
echo "   curl -s http://localhost:8947/api/v1/idle/history | jq '.'"
echo

echo "=== Test Setup Complete ==="
echo "Note: The idle detection system requires actual metrics processing"
echo "      to trigger automated actions. This script sets up the framework"
echo "      for testing, but actions must be triggered manually or through"
echo "      the daemon's idle detection processing."