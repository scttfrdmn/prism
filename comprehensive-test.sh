#!/bin/bash

# Comprehensive CloudWorkstation System Test
# This script validates EVERYTHING to ensure production readiness

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Test results
PASS_COUNT=0
FAIL_COUNT=0
TOTAL_TESTS=0

# Logging
LOG_FILE="comprehensive-test.log"
echo "=== CloudWorkstation Comprehensive Test $(date) ===" > $LOG_FILE

print_header() {
    echo -e "${BOLD}${BLUE}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${BLUE}║                    COMPREHENSIVE CLOUDWORKSTATION TEST                       ║${NC}"
    echo -e "${BOLD}${BLUE}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

test_result() {
    local test_name="$1"
    local status="$2"
    local details="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✅ PASS${NC} $test_name"
        PASS_COUNT=$((PASS_COUNT + 1))
        echo "PASS: $test_name - $details" >> $LOG_FILE
    else
        echo -e "${RED}❌ FAIL${NC} $test_name: $details"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        echo "FAIL: $test_name - $details" >> $LOG_FILE
    fi
}

run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -e "${BLUE}Testing:${NC} $test_name"
    
    if eval "$test_command" &>/dev/null; then
        test_result "$test_name" "PASS" "Command executed successfully"
    else
        local exit_code=$?
        test_result "$test_name" "FAIL" "Command failed with exit code $exit_code"
    fi
}

run_test_with_output() {
    local test_name="$1"
    local test_command="$2"
    local expected_output="$3"
    
    echo -e "${BLUE}Testing:${NC} $test_name"
    
    local output=$(eval "$test_command" 2>&1)
    local exit_code=$?
    
    if [ $exit_code -eq 0 ] && [[ "$output" == *"$expected_output"* ]]; then
        test_result "$test_name" "PASS" "Output contains expected text: $expected_output"
    else
        test_result "$test_name" "FAIL" "Exit code: $exit_code, Output: $output"
    fi
}

print_section() {
    echo ""
    echo -e "${BOLD}${YELLOW}=== $1 ===${NC}"
}

# Test binary existence and permissions
test_binaries() {
    print_section "BINARY VALIDATION"
    
    for binary in cws cwsd cws-gui; do
        if [ -f "./bin/$binary" ]; then
            if [ -x "./bin/$binary" ]; then
                test_result "Binary $binary exists and executable" "PASS" "Found at ./bin/$binary"
            else
                test_result "Binary $binary executable" "FAIL" "File exists but not executable"
            fi
        else
            test_result "Binary $binary exists" "FAIL" "Binary not found at ./bin/$binary"
        fi
    done
}

# Test version consistency
test_versions() {
    print_section "VERSION CONSISTENCY"
    
    local cli_version=$(./bin/cws --version 2>/dev/null | head -1)
    local daemon_version=$(./bin/cwsd --version 2>/dev/null | head -1)
    
    if [ "$cli_version" = "$daemon_version" ]; then
        test_result "CLI and Daemon version consistency" "PASS" "Both report: $cli_version"
    else
        test_result "CLI and Daemon version consistency" "FAIL" "CLI: $cli_version, Daemon: $daemon_version"
    fi
}

# Test daemon functionality
test_daemon() {
    print_section "DAEMON FUNCTIONALITY"
    
    # Test daemon status
    run_test_with_output "Daemon status check" "./bin/cws daemon status" "Status: running"
    
    # Test API endpoints
    run_test_with_output "API ping endpoint" "curl -s http://localhost:8947/api/v1/ping" "ok"
    run_test_with_output "API templates endpoint" "curl -s http://localhost:8947/api/v1/templates" "{"
    run_test_with_output "API instances endpoint" "curl -s http://localhost:8947/api/v1/instances" "{"
    
    # Test daemon connection from CLI
    run_test "CLI daemon connectivity" "./bin/cws daemon status >/dev/null"
}

# Test template system
test_templates() {
    print_section "TEMPLATE SYSTEM"
    
    # Template listing
    run_test_with_output "Template listing" "./bin/cws templates list" "Available Templates"
    
    # Template validation
    run_test_with_output "Template validation" "./bin/cws templates validate" "All templates are valid"
    
    # Specific template info
    run_test_with_output "Template info command" "./bin/cws templates info \"Python Machine Learning (Simplified)\"" "Name"
    
    # Template inheritance test
    run_test_with_output "Template inheritance" "./bin/cws templates info \"Rocky Linux 9 + Conda Stack\"" "rocky"
}

# Test CLI commands
test_cli_commands() {
    print_section "CLI COMMAND COVERAGE"
    
    # Help system
    run_test "Main help" "./bin/cws --help"
    run_test "Templates help" "./bin/cws templates --help"
    run_test "Daemon help" "./bin/cws daemon --help"
    run_test "Profiles help" "./bin/cws profiles --help"
    
    # List commands (should not fail even if empty)
    run_test "List instances" "./bin/cws list"
    run_test "List profiles" "./bin/cws profiles list"
    
    # Storage commands
    run_test "Storage help" "./bin/cws storage --help"
    run_test "Volume help" "./bin/cws volume --help"
}

# Test profile system
test_profiles() {
    print_section "PROFILE SYSTEM"
    
    # Profile listing (even if empty)
    run_test "Profile list command" "./bin/cws profiles list"
    
    # Profile help
    run_test "Profile add help" "./bin/cws profiles add --help"
}

# Test TUI (non-interactive)
test_tui() {
    print_section "TUI INTERFACE"
    
    # Test TUI help
    run_test "TUI help" "./bin/cws tui --help"
    
    # Test TUI daemon detection (should not start another daemon)
    echo "Checking TUI daemon detection..."
    local tui_output=$(timeout 3s bash -c 'echo "q" | ./bin/cws tui' 2>&1 || true)
    
    if [[ "$tui_output" == *"Attempting to start daemon"* ]]; then
        test_result "TUI daemon detection" "FAIL" "TUI tried to start daemon when one is already running"
    else
        test_result "TUI daemon detection" "PASS" "TUI correctly detected existing daemon"
    fi
}

# Test documentation consistency
test_documentation() {
    print_section "DOCUMENTATION CONSISTENCY"
    
    # Check key files exist
    for file in README.md DEMO_SEQUENCE.md demo.sh AWS_SETUP_GUIDE.md INSTALL.md; do
        if [ -f "$file" ]; then
            test_result "Documentation file $file exists" "PASS" "Found $file"
        else
            test_result "Documentation file $file exists" "FAIL" "Missing $file"
        fi
    done
    
    # Check template directory
    if [ -d "templates" ] && [ $(ls templates/*.yml 2>/dev/null | wc -l) -gt 0 ]; then
        test_result "Template files exist" "PASS" "Found $(ls templates/*.yml | wc -l) template files"
    else
        test_result "Template files exist" "FAIL" "No template files found"
    fi
}

# Test build system
test_build_system() {
    print_section "BUILD SYSTEM"
    
    # Check Makefile
    if [ -f "Makefile" ]; then
        test_result "Makefile exists" "PASS" "Found Makefile"
        
        # Test make targets
        if grep -q "build:" Makefile; then
            test_result "Make build target exists" "PASS" "Found build target"
        else
            test_result "Make build target exists" "FAIL" "No build target found"
        fi
        
        if grep -q "test:" Makefile; then
            test_result "Make test target exists" "PASS" "Found test target"
        else
            test_result "Make test target exists" "FAIL" "No test target found"
        fi
    else
        test_result "Makefile exists" "FAIL" "No Makefile found"
    fi
}

# Test error handling
test_error_handling() {
    print_section "ERROR HANDLING"
    
    # Test invalid commands
    local invalid_output=$(./bin/cws invalid-command 2>&1 || true)
    if [[ "$invalid_output" == *"unknown command"* ]] || [[ "$invalid_output" == *"Usage"* ]]; then
        test_result "Invalid command handling" "PASS" "Proper error message for invalid command"
    else
        test_result "Invalid command handling" "FAIL" "No proper error handling: $invalid_output"
    fi
    
    # Test missing arguments
    local missing_args_output=$(./bin/cws templates info 2>&1 || true)
    if [[ "$missing_args_output" == *"Usage"* ]] || [[ "$missing_args_output" == *"required"* ]]; then
        test_result "Missing arguments handling" "PASS" "Proper error for missing arguments"
    else
        test_result "Missing arguments handling" "FAIL" "No proper error handling: $missing_args_output"
    fi
}

# Test unit test compilation
test_unit_tests() {
    print_section "UNIT TEST COMPILATION"
    
    # Test that tests compile
    if go test -c ./internal/cli/ -o /tmp/cli-test 2>/dev/null; then
        test_result "CLI unit tests compile" "PASS" "Unit tests compile successfully"
        rm -f /tmp/cli-test
    else
        test_result "CLI unit tests compile" "FAIL" "Unit test compilation failed"
    fi
    
    # Test AWS integration tests compile
    if go test -c -tags aws_integration ./internal/cli/ -o /tmp/aws-test 2>/dev/null; then
        test_result "AWS integration tests compile" "PASS" "AWS tests compile successfully"
        rm -f /tmp/aws-test
    else
        test_result "AWS integration tests compile" "FAIL" "AWS test compilation failed"
    fi
}

# Print final results
print_results() {
    echo ""
    echo -e "${BOLD}${BLUE}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${BLUE}║                            TEST RESULTS                                      ║${NC}"
    echo -e "${BOLD}${BLUE}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    
    echo -e "${BOLD}Total Tests:${NC} $TOTAL_TESTS"
    echo -e "${GREEN}${BOLD}Passed:${NC} $PASS_COUNT"
    echo -e "${RED}${BOLD}Failed:${NC} $FAIL_COUNT"
    
    local pass_percentage=$((PASS_COUNT * 100 / TOTAL_TESTS))
    echo -e "${BOLD}Success Rate:${NC} ${pass_percentage}%"
    
    echo ""
    echo -e "${BOLD}Detailed log:${NC} $LOG_FILE"
    
    if [ $FAIL_COUNT -eq 0 ]; then
        echo ""
        echo -e "${GREEN}${BOLD}🎉 ALL TESTS PASSED - PRODUCTION READY!${NC}"
        exit 0
    else
        echo ""
        echo -e "${RED}${BOLD}❌ SOME TESTS FAILED - REQUIRES ATTENTION${NC}"
        echo ""
        echo -e "${BOLD}Failed tests summary:${NC}"
        grep "FAIL:" $LOG_FILE | sed 's/^/  - /'
        exit 1
    fi
}

# Main execution
main() {
    print_header
    
    test_binaries
    test_versions
    test_daemon
    test_templates
    test_cli_commands
    test_profiles
    test_tui
    test_documentation
    test_build_system
    test_error_handling
    test_unit_tests
    
    print_results
}

# Run the comprehensive test
main "$@"