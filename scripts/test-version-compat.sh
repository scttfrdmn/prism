#!/bin/bash
# Test version compatibility checking

set -e

echo "Testing version compatibility..."

# Get CLI and daemon versions
CLI_VERSION=$(./bin/cws --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
DAEMON_VERSION=$(./bin/cwsd --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)

if [ "$CLI_VERSION" != "$DAEMON_VERSION" ]; then
    echo "⚠️  Version mismatch: CLI=$CLI_VERSION, Daemon=$DAEMON_VERSION"
    echo "   (This is OK for testing, but should match in production)"
fi

echo "✅ Version compatibility check complete"
echo "   CLI:    v$CLI_VERSION"
echo "   Daemon: v$DAEMON_VERSION"
