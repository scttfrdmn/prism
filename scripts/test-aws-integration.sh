#!/bin/bash

# CloudWorkstation AWS Integration Test Runner
# Runs AWS integration tests with proper environment setup

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Default configuration
AWS_PROFILE="${AWS_PROFILE:-aws}"
AWS_TEST_REGION="${AWS_TEST_REGION:-us-east-1}"
TEST_TIMEOUT="${AWS_TEST_TIMEOUT:-15}"
TEST_SCOPE="${1:-quick}"

echo "🚀 CloudWorkstation AWS Integration Test Runner"
echo "=============================================="
echo ""
log_info "Configuration:"
echo "  Test Scope: $TEST_SCOPE"
echo "  AWS Profile: $AWS_PROFILE"
echo "  AWS Region: $AWS_TEST_REGION"
echo "  Timeout: ${TEST_TIMEOUT}m"
echo ""

# Check if AWS tests are enabled
if [ "$RUN_AWS_TESTS" != "true" ]; then
    log_error "AWS tests not enabled. Set RUN_AWS_TESTS=true"
    exit 1
fi

# Verify build
log_info "Building CloudWorkstation..."
if ! make build > /dev/null 2>&1; then
    log_error "Build failed"
    exit 1
fi
log_success "Build completed"

# Check daemon
log_info "Checking daemon..."
if ! curl -s http://localhost:8947/api/v1/ping > /dev/null 2>&1; then
    log_warning "Daemon not running, starting..."
    ./bin/prismd &
    DAEMON_PID=$!
    sleep 3
    if ! curl -s http://localhost:8947/api/v1/ping > /dev/null 2>&1; then
        log_error "Failed to start daemon"
        kill $DAEMON_PID 2>/dev/null || true
        exit 1
    fi
    log_success "Daemon started"
    CLEANUP_DAEMON=true
else
    log_success "Daemon is running"
    CLEANUP_DAEMON=false
fi

# Function to cleanup daemon if we started it
cleanup() {
    if [ "$CLEANUP_DAEMON" = "true" ] && [ -n "$DAEMON_PID" ]; then
        log_info "Stopping daemon..."
        kill $DAEMON_PID 2>/dev/null || true
        sleep 1
    fi
}

# Set up cleanup trap
trap cleanup EXIT

# Run tests based on scope
case $TEST_SCOPE in
    "quick")
        log_info "Running quick AWS integration tests..."
        go test -v -tags=aws_integration ./internal/cli/ \
            -run "TestAWSTemplate|TestAWSDaemon|TestAWSError" \
            -timeout="${TEST_TIMEOUT}m"
        ;;
    "full")
        log_info "Running full AWS integration tests..."
        go test -v -tags=aws_integration ./internal/cli/ \
            -run "TestAWS" \
            -timeout="${TEST_TIMEOUT}m"
        ;;
    "lifecycle")
        log_info "Running instance lifecycle tests..."
        go test -v -tags=aws_integration ./internal/cli/ \
            -run "TestAWSInstanceLifecycle" \
            -timeout="${TEST_TIMEOUT}m"
        ;;
    "storage")
        log_info "Running storage tests..."
        go test -v -tags=aws_integration ./internal/cli/ \
            -run "TestAWSStorage" \
            -timeout="${TEST_TIMEOUT}m"
        ;;
    *)
        log_info "Running specific test pattern: $TEST_SCOPE"
        go test -v -tags=aws_integration ./internal/cli/ \
            -run "$TEST_SCOPE" \
            -timeout="${TEST_TIMEOUT}m"
        ;;
esac

# Success
log_success "AWS integration tests completed successfully!"
echo ""
log_warning "Remember to check your AWS account for any remaining test resources"
log_warning "Monitor AWS costs in your test account billing dashboard"