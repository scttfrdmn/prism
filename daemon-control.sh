#!/bin/bash
# CloudWorkstation Daemon Control Script
# Prevents multiple daemon instances and ensures proper development mode

set -e

# Source environment variables
if [[ -f .env ]]; then
    source .env
    echo "✅ Loaded environment from .env"
fi

# Ensure development mode is enabled
export CLOUDWORKSTATION_DEV=true
echo "✅ Development mode enabled (no keychain prompts)"

# Function to stop all existing daemons
stop_all_daemons() {
    echo "🛑 Stopping all existing cwsd processes..."
    pkill -f cwsd 2>/dev/null || echo "No existing cwsd processes found"
    sleep 2
    
    # Verify all stopped
    if pgrep -f cwsd >/dev/null 2>&1; then
        echo "⚠️  Some cwsd processes still running, force killing..."
        pkill -9 -f cwsd 2>/dev/null || true
        sleep 2
    fi
    echo "✅ All cwsd processes stopped"
}

# Function to start daemon safely
start_daemon() {
    echo "🚀 Starting CloudWorkstation daemon..."
    
    # Check if daemon is already running
    if pgrep -f cwsd >/dev/null 2>&1; then
        echo "⚠️  Daemon already running, stopping first..."
        stop_all_daemons
    fi
    
    # Start daemon with proper environment
    ./bin/cwsd &
    DAEMON_PID=$!
    
    # Wait a moment and verify it started
    sleep 3
    if kill -0 $DAEMON_PID 2>/dev/null; then
        echo "✅ Daemon started successfully (PID: $DAEMON_PID)"
        echo "🔗 API available at: http://localhost:8947"
    else
        echo "❌ Daemon failed to start"
        exit 1
    fi
}

# Function to show daemon status
show_status() {
    echo "📊 CloudWorkstation Daemon Status:"
    if pgrep -f cwsd >/dev/null 2>&1; then
        echo "✅ Daemon is running:"
        ps aux | grep cwsd | grep -v grep | head -5
        echo ""
        echo "🔗 Testing API connection..."
        if curl -s http://localhost:8947/api/v1/ping >/dev/null 2>&1; then
            echo "✅ API is responding (health check passed)"
        else
            echo "⚠️  API not responding"
        fi
    else
        echo "❌ No daemon processes running"
    fi
    echo ""
    echo "🔧 Environment: CLOUDWORKSTATION_DEV=$CLOUDWORKSTATION_DEV"
}

# Main command handling
case "${1:-status}" in
    start)
        start_daemon
        ;;
    stop)
        stop_all_daemons
        ;;
    restart)
        stop_all_daemons
        start_daemon
        ;;
    status)
        show_status
        ;;
    clean)
        echo "🧹 Cleaning up all CloudWorkstation processes..."
        stop_all_daemons
        # Clean up any stale state files
        rm -f ~/.cloudworkstation/daemon.pid 2>/dev/null || true
        echo "✅ Cleanup complete"
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|clean}"
        echo ""
        echo "Commands:"
        echo "  start   - Start daemon (stops existing ones first)"
        echo "  stop    - Stop all daemon processes"
        echo "  restart - Stop and start daemon"
        echo "  status  - Show daemon status and API health"
        echo "  clean   - Full cleanup of processes and state"
        exit 1
        ;;
esac