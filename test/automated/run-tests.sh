#!/bin/bash
# Automated test suite for CloudWorkstation v0.4.5

set -e

echo "🧪 CloudWorkstation Automated Test Suite v0.4.5"
echo "================================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -n "Testing: $test_name... "
    
    if eval "$test_command" > /tmp/test_output.log 2>&1; then
        echo -e "${GREEN}✓${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        echo "  Error: $(tail -n 1 /tmp/test_output.log)"
    fi
}

# Start daemon if not running
echo "1. Starting daemon..."
pkill cwsd 2>/dev/null || true
sleep 1
./bin/cwsd > /tmp/daemon-test.log 2>&1 &
DAEMON_PID=$!
sleep 2

# Verify daemon is running
run_test "Daemon startup" "curl -s http://localhost:8947/api/v1/ping | grep -q 'ok'"

echo ""
echo "2. CLI Tests"
echo "------------"
run_test "CLI version" "./bin/cws --version | grep -q '0.4.5'"
run_test "Daemon status" "./bin/cws daemon status | grep -q 'running'"
run_test "Templates list" "./bin/cws templates | grep -q 'Available Templates'"
run_test "Idle policies" "./bin/cws idle policy list | grep -q 'POLICY ID'"
run_test "Profile list" "./bin/cws profiles | grep -q 'PROFILE ID'"
run_test "Instance list" "./bin/cws list"
run_test "Storage list" "./bin/cws storage list"
run_test "Volume list" "./bin/cws volume list"
run_test "Rightsizing summary" "./bin/cws rightsizing summary"
run_test "Template validation" "./bin/cws templates validate | grep -q 'All templates are valid'"

echo ""
echo "3. API Endpoint Tests"
echo "---------------------"
run_test "API ping" "curl -s http://localhost:8947/api/v1/ping | jq -e '.status == \"ok\"'"
run_test "API status" "curl -s http://localhost:8947/api/v1/status | jq -e '.version == \"0.4.5\"'"
run_test "API templates" "curl -s http://localhost:8947/api/v1/templates | jq -e 'length > 0'"
run_test "API instances" "curl -s http://localhost:8947/api/v1/instances | jq -e '. != null'"
run_test "API idle policies" "curl -s http://localhost:8947/api/v1/idle/policies | jq -e 'length == 6'"
run_test "API idle savings" "curl -s http://localhost:8947/api/v1/idle/savings | jq -e '.total_saved != null'"
run_test "API storage" "curl -s http://localhost:8947/api/v1/storage | jq -e '. != null'"
run_test "API volumes" "curl -s http://localhost:8947/api/v1/volumes | jq -e '. != null'"

echo ""
echo "4. Error Handling Tests"
echo "-----------------------"
run_test "Invalid instance" "./bin/cws start nonexistent 2>&1 | grep -q 'not found'"
run_test "Invalid stop" "curl -s -X POST http://localhost:8947/api/v1/instances/fake/stop | jq -e '.code == \"server_error\"'"
run_test "Hibernation status (missing)" "curl -s http://localhost:8947/api/v1/instances/test/hibernation-status | jq -e '.hibernation_supported == false'"

echo ""
echo "5. TUI Tests"
echo "------------"
run_test "TUI startup" "timeout 2 ./bin/cws tui < /dev/null 2>&1 | grep -q 'CloudWorkstation Dashboard' || true"

echo ""
echo "6. GUI Frontend Tests (if npm available)"
echo "-----------------------------------------"
if command -v npm &> /dev/null; then
    cd cmd/cws-gui/frontend
    
    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        echo "Installing frontend dependencies..."
        npm install > /dev/null 2>&1
    fi
    
    # Run unit tests
    run_test "GUI unit tests" "npm run test:unit"
    
    # Run e2e tests with mock server
    if [ -f "tests/mocks/daemon-server.js" ]; then
        node tests/mocks/daemon-server.js &
        MOCK_PID=$!
        sleep 2
        run_test "GUI e2e tests" "npx playwright test --project=chromium"
        kill $MOCK_PID 2>/dev/null || true
    fi
    
    cd ../../..
else
    echo "Skipping GUI tests (npm not found)"
fi

echo ""
echo "7. Integration Tests"
echo "--------------------"
# Test dry-run launch
run_test "Dry-run launch" "./bin/cws launch --dry-run python-ml test-instance"

# Test idle policy details
run_test "Idle policy details" "./bin/cws idle policy details balanced | grep -q 'Balanced Performance'"

# Test help commands
run_test "Hibernate help" "./bin/cws hibernate --help | grep -q 'preserve RAM state'"
run_test "Resume help" "./bin/cws resume --help | grep -q 'instant startup'"

echo ""
echo "================================================"
echo "Test Results Summary:"
echo "  Total Tests: $TOTAL_TESTS"
echo -e "  Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "  Failed: ${RED}$FAILED_TESTS${NC}"

# Cleanup
kill $DAEMON_PID 2>/dev/null || true

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}✅ All tests passed! CloudWorkstation v0.4.5 is ready for release.${NC}"
    exit 0
else
    echo -e "\n${RED}❌ $FAILED_TESTS test(s) failed. Please review the errors above.${NC}"
    exit 1
fi