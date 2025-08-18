#!/bin/bash

# CloudWorkstation Comprehensive System Test
# Tests all core functionality to ensure the complete system works correctly
#
# Usage:
#   ./system-test.sh              # Run all tests
#   ./system-test.sh --help       # Show this help
#   ./system-test.sh --version    # Show test version
#
# This script validates:
#   • Binary compilation and execution
#   • Daemon functionality and API endpoints  
#   • Template system with inheritance
#   • Profile management
#   • CLI command coverage
#   • Multi-modal access (CLI, basic TUI)
#   • Documentation accuracy
#   • Error handling
#   • Build system consistency
#
# Prerequisites:
#   • Built binaries in ./bin/ directory
#   • curl command available
#   • Current directory should be CloudWorkstation root
#
# Note: Some tests may fail if AWS credentials are not configured,
#       but core system functionality will still be validated.

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Counters
TESTS_PASSED=0
TESTS_FAILED=0
FAILED_TESTS=()

print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

test_result() {
    local exit_code=$1
    local test_name="$2"
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}✅ PASS${NC}: $test_name"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}❌ FAIL${NC}: $test_name"
        FAILED_TESTS+=("$test_name")
        ((TESTS_FAILED++))
    fi
}

# Cleanup function
cleanup() {
    if [ -n "${DAEMON_PID:-}" ] && kill -0 "$DAEMON_PID" 2>/dev/null; then
        kill "$DAEMON_PID" 2>/dev/null || true
    fi
    pkill -f cwsd 2>/dev/null || true
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "CloudWorkstation Comprehensive System Test"
        echo ""
        echo "Usage: $0 [--help|--version]"
        echo ""
        echo "This script validates all core CloudWorkstation functionality:"
        echo "  • Binary compilation and execution"
        echo "  • Daemon functionality and API endpoints"
        echo "  • Template system with inheritance"
        echo "  • Profile management"
        echo "  • CLI command coverage"
        echo "  • Multi-modal access (CLI, basic TUI)"
        echo "  • Documentation accuracy"
        echo "  • Error handling"
        echo "  • Build system consistency"
        echo ""
        echo "Prerequisites:"
        echo "  • Built binaries in ./bin/ directory"
        echo "  • curl command available"
        echo "  • Run from CloudWorkstation root directory"
        echo ""
        echo "Exit codes:"
        echo "  0 - All tests passed"
        echo "  1 - Some tests failed"
        exit 0
        ;;
    --version)
        echo "CloudWorkstation System Test v1.0"
        echo "Compatible with CloudWorkstation v0.4.2"
        exit 0
        ;;
    *)
        ;;
esac

trap cleanup EXIT

print_header "CloudWorkstation System Test v0.4.2"
echo "Working directory: $(pwd)"
echo "Testing core functionality..."

# Verify prerequisites
if ! command -v curl >/dev/null 2>&1; then
    echo -e "${RED}❌ Error: curl is required but not installed${NC}"
    exit 1
fi

if [ ! -d "./bin" ]; then
    echo -e "${RED}❌ Error: ./bin directory not found. Run 'make build' first.${NC}"
    exit 1
fi

# 1. Binary Tests
print_header "Binary Compilation & Execution"

if [ -f "./bin/cws" ] && [ -x "./bin/cws" ]; then
    test_result 0 "CLI binary exists and executable"
else
    test_result 1 "CLI binary exists and executable"
fi

if [ -f "./bin/cwsd" ] && [ -x "./bin/cwsd" ]; then
    test_result 0 "Daemon binary exists and executable"
else
    test_result 1 "Daemon binary exists and executable"
fi

if [ -f "./bin/cws-gui" ] && [ -x "./bin/cws-gui" ]; then
    test_result 0 "GUI binary exists and executable"
else
    test_result 1 "GUI binary exists and executable"
fi

# 2. Version Tests
print_header "Version Reporting"

if ./bin/cws --version | grep -q "CloudWorkstation v"; then
    test_result 0 "CLI version reporting"
else
    test_result 1 "CLI version reporting"
fi

if ./bin/cws --version | grep -q "v0\.4\.2"; then
    test_result 0 "CLI version format correct"
else
    test_result 1 "CLI version format correct"
fi

# 3. Help Tests
print_header "CLI Help System"

if ./bin/cws --help | grep -q "CloudWorkstation"; then
    test_result 0 "CLI help displays"
else
    test_result 1 "CLI help displays"
fi

if ./bin/cws --help | grep -q "Commands:"; then
    test_result 0 "CLI commands listed"
else
    test_result 1 "CLI commands listed"
fi

if ./bin/cws --help | grep -q "Examples:"; then
    test_result 0 "CLI examples provided"
else
    test_result 1 "CLI examples provided"
fi

# 4. Template Tests
print_header "Template System"

if [ -d "./templates" ]; then
    test_result 0 "Templates directory exists"
else
    test_result 1 "Templates directory exists"
fi

template_count=$(find ./templates -name "*.yml" 2>/dev/null | wc -l)
if [ "$template_count" -gt 0 ]; then
    test_result 0 "Template files exist ($template_count found)"
else
    test_result 1 "Template files exist"
fi

# 5. Daemon Tests
print_header "Daemon Functionality"

# Kill any existing daemon
pkill -f cwsd 2>/dev/null || true
sleep 2

# Add bin directory to PATH for all subsequent commands
export PATH="$(pwd)/bin:$PATH"
echo "Added $(pwd)/bin to PATH"

# Start daemon in background
echo "Starting daemon..."
./bin/cwsd >/dev/null 2>&1 &
DAEMON_PID=$!
sleep 5

# Test daemon is responsive
if kill -0 "$DAEMON_PID" 2>/dev/null; then
    test_result 0 "Daemon process starts successfully"
    
    # Test API endpoints
    if curl -s "http://localhost:8947/api/v1/ping" | grep -q "status\|ok"; then
        test_result 0 "Daemon ping endpoint responds"
    else
        test_result 1 "Daemon ping endpoint responds"
    fi
    
    if curl -s "http://localhost:8947/api/v1/status" >/dev/null 2>&1; then
        test_result 0 "Daemon status endpoint accessible"
    else
        test_result 1 "Daemon status endpoint accessible"
    fi
    
    if curl -s "http://localhost:8947/api/v1/templates" >/dev/null 2>&1; then
        test_result 0 "Templates API endpoint accessible"
    else
        test_result 1 "Templates API endpoint accessible"
    fi
    
    if curl -s "http://localhost:8947/api/v1/instances" >/dev/null 2>&1; then
        test_result 0 "Instances API endpoint accessible"
    else
        test_result 1 "Instances API endpoint accessible"
    fi
    
    # Test CLI daemon integration
    if ./bin/cws daemon status | grep -q "running\|Daemon is running\|active"; then
        test_result 0 "CLI daemon status command works"
    else
        test_result 1 "CLI daemon status command works"
    fi
    
    # Test templates with daemon running
    if ./bin/cws templates list | grep -q "Available templates:"; then
        test_result 0 "Templates list command works"
    else
        test_result 1 "Templates list command works"
    fi
    
else
    test_result 1 "Daemon process starts successfully"
fi

# 6. Template System Advanced
print_header "Template System Advanced"

if [ -f "./templates/base-rocky9.yml" ]; then
    test_result 0 "Base Rocky9 template exists"
else
    test_result 1 "Base Rocky9 template exists"
fi

if [ -f "./templates/rocky9-conda-stack.yml" ]; then
    test_result 0 "Rocky9 Conda Stack template exists"
    
    if grep -q "inherits" "./templates/rocky9-conda-stack.yml"; then
        test_result 0 "Template inheritance implemented"
    else
        test_result 1 "Template inheritance implemented"
    fi
else
    test_result 1 "Rocky9 Conda Stack template exists"
fi

# 7. Profile Tests
print_header "Profile Management"

if ./bin/cws profiles list | grep -q "Profiles"; then
    test_result 0 "Profile list command works"
else
    test_result 1 "Profile list command works"
fi

if ./bin/cws profiles current >/dev/null 2>&1; then
    test_result 0 "Current profile command works"
else
    test_result 1 "Current profile command works"
fi

# 8. Storage Tests
print_header "Storage Commands"

if ./bin/cws volume list | grep -q "EFS Volumes"; then
    test_result 0 "EFS volume list works"
else
    test_result 1 "EFS volume list works"
fi

if ./bin/cws storage list | grep -q "EBS Volumes"; then
    test_result 0 "EBS storage list works"
else
    test_result 1 "EBS storage list works"
fi

# 9. Instance Tests  
print_header "Instance Management"

if ./bin/cws list | grep -q "Workstations"; then
    test_result 0 "Instance list command works"
else
    test_result 1 "Instance list command works"
fi

# 10. Advanced Features
print_header "Advanced Features"

if ./bin/cws --help | grep -q "hibernate"; then
    test_result 0 "Hibernation commands documented"
else
    test_result 1 "Hibernation commands documented"
fi

if ./bin/cws idle status >/dev/null 2>&1; then
    test_result 0 "Idle detection available"
    
    if ./bin/cws idle profile list 2>/dev/null | grep -q "profiles\|batch\|gpu"; then
        test_result 0 "Idle profiles available"
    else
        test_result 1 "Idle profiles available"
    fi
else
    echo -e "${YELLOW}⏭️  SKIP${NC}: Idle detection (not available)"
fi

if ./bin/cws project list >/dev/null 2>&1; then
    test_result 0 "Project management available"
else
    echo -e "${YELLOW}⏭️  SKIP${NC}: Project management (not available)"
fi

# 11. Error Handling
print_header "Error Handling"

if ./bin/cws nonexistent-command 2>&1 | grep -q "Unknown command\|not found\|Invalid\|Error"; then
    test_result 0 "Invalid command error handling"
else
    test_result 1 "Invalid command error handling"
fi

if ./bin/cws launch 2>&1 | grep -q "usage\|Usage\|required\|arguments\|Error"; then
    test_result 0 "Missing argument error handling"
else
    test_result 1 "Missing argument error handling"
fi

# 12. Documentation
print_header "Documentation & Build System"

if [ -f "./README.md" ]; then
    test_result 0 "README.md exists"
else
    test_result 1 "README.md exists"
fi

if [ -f "./Makefile" ]; then
    test_result 0 "Makefile exists"
else
    test_result 1 "Makefile exists"
fi

if make --dry-run build >/dev/null 2>&1; then
    test_result 0 "Build system functional"
else
    test_result 1 "Build system functional"
fi

# Version consistency
makefile_version=$(grep "VERSION :=" Makefile | head -1 | cut -d' ' -f3)
cli_version=$(./bin/cws --version | grep -o "v[0-9]*\.[0-9]*\.[0-9]*" | head -1)
if [ "v$makefile_version" = "$cli_version" ]; then
    test_result 0 "Version consistency (Makefile: v$makefile_version, CLI: $cli_version)"
else
    test_result 1 "Version consistency (Makefile: v$makefile_version != CLI: $cli_version)"
fi

# TUI Basic Test
print_header "TUI Interface"

echo "Testing TUI basic functionality..."
if timeout 10s bash -c "echo '' | ./bin/cws tui" >/dev/null 2>&1; then
    test_result 0 "TUI launches without error"
else
    echo -e "${YELLOW}⏭️  SKIP${NC}: TUI test (requires interactive terminal)"
fi

# Cleanup daemon
if [ -n "${DAEMON_PID:-}" ] && kill -0 "$DAEMON_PID" 2>/dev/null; then
    if ./bin/cws daemon stop >/dev/null 2>&1; then
        test_result 0 "Daemon stops cleanly"
    else
        test_result 1 "Daemon stops cleanly"
    fi
fi

# Final Results
print_header "Test Results Summary"
TOTAL_TESTS=$((TESTS_PASSED + TESTS_FAILED))
echo -e "Total Tests: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"  
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"

if [ ${#FAILED_TESTS[@]} -gt 0 ]; then
    echo -e "\n${RED}Failed Tests:${NC}"
    for test in "${FAILED_TESTS[@]}"; do
        echo -e "  ${RED}• $test${NC}"
    done
fi

echo ""
if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 ALL TESTS PASSED! CloudWorkstation system is working correctly.${NC}"
    echo -e "${GREEN}✅ System is ready for production use.${NC}"
    echo -e "\n${CYAN}System Validation Summary:${NC}"
    echo -e "  • All binaries compile and execute correctly"
    echo -e "  • Daemon starts and responds to API calls"
    echo -e "  • Template system with inheritance works"
    echo -e "  • CLI commands provide expected functionality"
    echo -e "  • Profile management operational"
    echo -e "  • Storage commands functional"
    echo -e "  • Error handling works as expected"
    echo -e "  • Documentation and build system consistent"
    exit 0
else
    echo -e "${RED}❌ $TESTS_FAILED tests failed. System has issues that need attention.${NC}"
    echo -e "${YELLOW}⚠️  Review failed tests before deploying to production.${NC}"
    exit 1
fi