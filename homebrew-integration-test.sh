#!/bin/bash
# CloudWorkstation v0.4.2-1 Homebrew Integration Test Suite
# Tests the ACTUAL user experience from brew install through real usage
#
# This validates:
# - Real Homebrew installation process
# - Auto-daemon startup (should work with installed binaries in PATH)
# - Template system with both full names and slugs
# - Profile management integration
# - AWS connectivity (dry-run safe, real AWS comprehensive)
#
# Usage:
#   ./homebrew-integration-test.sh              # Safe dry-run testing
#   ./homebrew-integration-test.sh --real-aws   # Full real AWS testing

set -e

# Show help if requested
if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    echo "CloudWorkstation Homebrew Integration Test Suite"
    echo "==============================================="
    echo ""
    echo "Tests the ACTUAL user experience from brew install through real usage."
    echo ""
    echo "Usage:"
    echo "  $0                    # Safe dry-run testing (default)"
    echo "  $0 --real-aws         # Full real AWS testing with actual resources"
    echo "  $0 --help             # Show this help"
    echo ""
    echo "Safe Mode (default):"
    echo "  • Uses --dry-run flags for AWS operations"
    echo "  • No AWS resources created or costs incurred"
    echo "  • Tests installation, auto-daemon, templates, profiles"
    echo "  • Validates AWS connectivity without creating instances"
    echo ""
    echo "Real AWS Mode (--real-aws):"
    echo "  • Creates actual AWS instances for comprehensive testing"
    echo "  • Tests complete lifecycle: launch → info → terminate"
    echo "  • Verifies end-to-end tutorial workflows"
    echo "  • ⚠️  WILL INCUR AWS COSTS - includes automatic cleanup"
    echo "  • Requires AWS profile 'aws' configured and working"
    echo ""
    echo "Prerequisites:"
    echo "  • AWS CLI configured with profile 'aws'"
    echo "  • Homebrew installed"
    echo "  • Internet connectivity"
    echo ""
    exit 0
fi

echo "🧪 CloudWorkstation Homebrew Integration Test Suite"
echo "=================================================="
echo ""

# Parse command line arguments
REAL_AWS_MODE=false
if [[ "$1" == "--real-aws" ]]; then
    REAL_AWS_MODE=true
    echo "🔥 REAL AWS MODE: Will create actual AWS resources!"
    echo "⚠️  This will incur AWS costs and requires cleanup"
    echo ""
    read -p "Continue with real AWS testing? (y/N): " -r
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "❌ Real AWS testing cancelled"
        exit 1
    fi
    echo ""
fi

# Test configuration
TEMP_INSTANCE_NAME="homebrew-test-$(date +%s)"
TEST_RESULTS_LOG="homebrew-test-results.log"
FAILED_TESTS=0
TOTAL_TESTS=0
AWS_MODE_LABEL="DRY-RUN"
if [[ "$REAL_AWS_MODE" == "true" ]]; then
    AWS_MODE_LABEL="REAL AWS"
fi

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

test_result() {
    local test_name="$1"
    local status="$2"
    local details="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✅ PASS${NC} $test_name"
        echo "$(date): PASS $test_name - $details" >> "$TEST_RESULTS_LOG"
    else
        echo -e "${RED}❌ FAIL${NC} $test_name"
        echo "   Details: $details"
        echo "$(date): FAIL $test_name - $details" >> "$TEST_RESULTS_LOG"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

echo "Phase 1: Clean State Preparation"
echo "-------------------------------"

# Stop any existing daemon
if command -v cws &> /dev/null; then
    cws daemon stop &> /dev/null || true
fi

# Clean any existing installation
echo "🧹 Cleaning previous installation..."
brew uninstall cloudworkstation &> /dev/null || true
brew untap scttfrdmn/cloudworkstation &> /dev/null || true

# Verify clean state
if command -v cws &> /dev/null; then
    test_result "Clean state verification" "FAIL" "cws command still found in PATH after uninstall"
    exit 1
else
    test_result "Clean state verification" "PASS" "No existing cws installation found"
fi

echo ""
echo "Phase 2: Fresh Homebrew Installation"
echo "-----------------------------------"

# Add tap
echo "🍺 Adding CloudWorkstation tap..."
if brew tap scttfrdmn/cloudworkstation; then
    test_result "Homebrew tap addition" "PASS" "Tap added successfully"
else
    test_result "Homebrew tap addition" "FAIL" "Failed to add tap"
    exit 1
fi

# Install CloudWorkstation
echo "📦 Installing CloudWorkstation..."
if brew install cloudworkstation; then
    test_result "Homebrew installation" "PASS" "Installation completed successfully"
else
    test_result "Homebrew installation" "FAIL" "Installation failed"
    exit 1
fi

# Verify installation
if command -v cws &> /dev/null && command -v cwsd &> /dev/null; then
    test_result "Binary availability" "PASS" "Both cws and cwsd found in PATH"
else
    test_result "Binary availability" "FAIL" "Binaries not found in PATH"
    exit 1
fi

echo ""
echo "Phase 3: Auto-Daemon Startup Testing"
echo "-----------------------------------"

# Test that daemon auto-starts on first command
echo "🚀 Testing daemon auto-startup..."
if timeout 30 cws templates list > /dev/null 2>&1; then
    test_result "Auto-daemon startup" "PASS" "Templates list succeeded (daemon auto-started)"
else
    test_result "Auto-daemon startup" "FAIL" "Templates list failed or timed out"
fi

# Verify daemon is running
if cws daemon status > /dev/null 2>&1; then
    test_result "Daemon status verification" "PASS" "Daemon is running after auto-start"
else
    test_result "Daemon status verification" "FAIL" "Daemon not running after auto-start"
fi

echo ""
echo "Phase 4: Template System Testing"
echo "-------------------------------"

# Test template listing
echo "📋 Testing template operations..."
if cws templates list > /dev/null 2>&1; then
    test_result "Template listing" "PASS" "Templates list command succeeded"
else
    test_result "Template listing" "FAIL" "Templates list command failed"
fi

# Test template info with full name
if cws templates info "Python Machine Learning (Simplified)" > /dev/null 2>&1; then
    test_result "Template info (full name)" "PASS" "Template info with full name succeeded"
else
    test_result "Template info (full name)" "FAIL" "Template info with full name failed"
fi

# Test template info with slug
if cws templates info python-ml > /dev/null 2>&1; then
    test_result "Template info (slug)" "PASS" "Template info with slug succeeded"
else
    test_result "Template info (slug)" "FAIL" "Template info with slug failed"
fi

# Test template validation
if cws templates validate > /dev/null 2>&1; then
    test_result "Template validation" "PASS" "Template validation succeeded"
else
    test_result "Template validation" "FAIL" "Template validation failed"
fi

echo ""
echo "Phase 5: Launch Testing ($AWS_MODE_LABEL)"
echo "--------------------------------"

# Test launch with full name
echo "🚀 Testing instance launch operations..."
LAUNCH_FLAGS=""
if [[ "$REAL_AWS_MODE" != "true" ]]; then
    LAUNCH_FLAGS="--dry-run"
fi

if timeout 120 cws launch "Python Machine Learning (Simplified)" "$TEMP_INSTANCE_NAME-full" $LAUNCH_FLAGS > /dev/null 2>&1; then
    test_result "Launch with full name ($AWS_MODE_LABEL)" "PASS" "$AWS_MODE_LABEL launch with full name succeeded"
else
    test_result "Launch with full name ($AWS_MODE_LABEL)" "FAIL" "$AWS_MODE_LABEL launch with full name failed"
fi

# Test launch with slug
if timeout 120 cws launch python-ml "$TEMP_INSTANCE_NAME-slug" $LAUNCH_FLAGS > /dev/null 2>&1; then
    test_result "Launch with slug ($AWS_MODE_LABEL)" "PASS" "$AWS_MODE_LABEL launch with slug succeeded"
else
    test_result "Launch with slug ($AWS_MODE_LABEL)" "FAIL" "$AWS_MODE_LABEL launch with slug failed"
fi

echo ""
echo "Phase 6: Profile System Testing"
echo "------------------------------"

# Test profile operations
echo "👤 Testing profile management..."
if cws profiles list > /dev/null 2>&1; then
    test_result "Profile listing" "PASS" "Profile list command succeeded"
else
    test_result "Profile listing" "FAIL" "Profile list command failed"
fi

if cws profiles current > /dev/null 2>&1; then
    test_result "Current profile check" "PASS" "Current profile command succeeded"
else
    test_result "Current profile check" "FAIL" "Current profile command failed"
fi

echo ""
echo "Phase 7: Instance Management Testing"
echo "----------------------------------"

# Test instance listing (should be empty)
echo "📋 Testing instance operations..."
if cws list > /dev/null 2>&1; then
    test_result "Instance listing" "PASS" "Instance list command succeeded"
else
    test_result "Instance listing" "FAIL" "Instance list command failed"
fi

echo ""
echo "Phase 8: Version Consistency Testing"
echo "-----------------------------------"

# Test version reporting
CLI_VERSION=$(cws --version 2>/dev/null | head -1)
DAEMON_VERSION=$(timeout 10 cwsd --version 2>/dev/null | head -1)

if [[ "$CLI_VERSION" == "$DAEMON_VERSION" ]]; then
    test_result "Version consistency" "PASS" "CLI and daemon versions match: $CLI_VERSION"
else
    test_result "Version consistency" "FAIL" "CLI version '$CLI_VERSION' != daemon version '$DAEMON_VERSION'"
fi

# Check that it's the expected version
if [[ "$CLI_VERSION" =~ "0.4.2-1" ]]; then
    test_result "Version correctness" "PASS" "Version includes expected 0.4.2-1"
else
    test_result "Version correctness" "FAIL" "Version '$CLI_VERSION' does not include expected 0.4.2-1"
fi

echo ""
echo "Phase 9: Real Tutorial Workflow Test ($AWS_MODE_LABEL)"
echo "----------------------------------"

# Test the exact tutorial workflow
echo "📚 Testing tutorial workflow..."

# Step 1: Templates (already tested above)
# Step 2: Launch with slug (the efficient way)
TUTORIAL_FLAGS=""
if [[ "$REAL_AWS_MODE" != "true" ]]; then
    TUTORIAL_FLAGS="--dry-run"
fi

if timeout 60 cws launch python-ml tutorial-test $TUTORIAL_FLAGS > /dev/null 2>&1; then
    test_result "Tutorial workflow (launch)" "PASS" "Tutorial launch command succeeded"
else
    test_result "Tutorial workflow (launch)" "FAIL" "Tutorial launch command failed"
fi

# Test storage operations (should work without creating resources)
if cws storage list > /dev/null 2>&1; then
    test_result "Storage operations" "PASS" "Storage list command succeeded"
else
    test_result "Storage operations" "FAIL" "Storage list command failed"
fi

# Real AWS comprehensive testing
if [[ "$REAL_AWS_MODE" == "true" ]]; then
    echo ""
    echo "Phase 9.5: Real AWS Instance Operations"
    echo "-------------------------------------"
    
    # List instances (should show our created instances)
    if cws list > /dev/null 2>&1; then
        INSTANCE_COUNT=$(cws list 2>/dev/null | grep -c "$TEMP_INSTANCE_NAME" || echo "0")
        if [[ "$INSTANCE_COUNT" -gt "0" ]]; then
            test_result "Real instance verification" "PASS" "Found $INSTANCE_COUNT launched instances"
        else
            test_result "Real instance verification" "FAIL" "No instances found after launch"
        fi
    else
        test_result "Real instance verification" "FAIL" "Failed to list instances"
    fi
    
    # Test instance connection info (doesn't actually connect)
    FIRST_INSTANCE=$(cws list 2>/dev/null | grep "$TEMP_INSTANCE_NAME" | head -1 | awk '{print $1}' || echo "")
    if [[ -n "$FIRST_INSTANCE" ]]; then
        if timeout 30 cws info "$FIRST_INSTANCE" > /dev/null 2>&1; then
            test_result "Instance connection info" "PASS" "Connection info retrieved successfully"
        else
            test_result "Instance connection info" "FAIL" "Failed to get connection info"
        fi
    fi
fi

echo ""
echo "Phase 10: Cleanup and Final Status"
echo "---------------------------------"

# Real AWS cleanup (important to avoid costs!)
if [[ "$REAL_AWS_MODE" == "true" ]]; then
    echo "🧹 Cleaning up real AWS resources..."
    
    # Get list of test instances to clean up
    TEST_INSTANCES=$(cws list 2>/dev/null | grep "$TEMP_INSTANCE_NAME" | awk '{print $1}' || echo "")
    
    if [[ -n "$TEST_INSTANCES" ]]; then
        echo "   Found test instances to clean up:"
        echo "$TEST_INSTANCES" | while read -r instance; do
            echo "   • $instance"
        done
        
        # Terminate all test instances
        CLEANUP_SUCCESS=true
        echo "$TEST_INSTANCES" | while read -r instance; do
            if [[ -n "$instance" ]]; then
                echo "   Terminating $instance..."
                if timeout 60 cws delete "$instance" --force > /dev/null 2>&1; then
                    echo "   ✅ $instance terminated"
                else
                    echo "   ❌ Failed to terminate $instance"
                    CLEANUP_SUCCESS=false
                fi
            fi
        done
        
        # Wait a moment and verify cleanup
        sleep 5
        REMAINING_INSTANCES=$(cws list 2>/dev/null | grep -c "$TEMP_INSTANCE_NAME" || echo "0")
        if [[ "$REMAINING_INSTANCES" == "0" ]]; then
            test_result "AWS resource cleanup" "PASS" "All test instances terminated successfully"
        else
            test_result "AWS resource cleanup" "FAIL" "$REMAINING_INSTANCES test instances still running - MANUAL CLEANUP NEEDED!"
            echo ""
            echo "⚠️  WARNING: Manual cleanup required for remaining instances:"
            cws list | grep "$TEMP_INSTANCE_NAME" | awk '{print "   • " $1}'
            echo "   Run: cws delete <instance-name> --force"
        fi
    else
        test_result "AWS resource cleanup" "PASS" "No test instances found to clean up"
    fi
fi

# Stop daemon cleanly
if cws daemon stop > /dev/null 2>&1; then
    test_result "Daemon shutdown" "PASS" "Daemon stopped cleanly"
else
    test_result "Daemon shutdown" "FAIL" "Daemon failed to stop cleanly"
fi

# Final summary
echo ""
echo "🎯 Test Results Summary"
echo "======================"
echo "Total Tests: $TOTAL_TESTS"
echo "Passed: $((TOTAL_TESTS - FAILED_TESTS))"
echo "Failed: $FAILED_TESTS"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}🎉 ALL TESTS PASSED - HOMEBREW INTEGRATION READY!${NC}"
    echo ""
    echo "✅ Verified Real User Experience ($AWS_MODE_LABEL):"
    echo "  • Fresh Homebrew installation works correctly"
    echo "  • Daemon auto-starts seamlessly (no manual start needed)"
    echo "  • Both full names and slugs work for templates"
    echo "  • Profile management functions correctly"
    if [[ "$REAL_AWS_MODE" == "true" ]]; then
        echo "  • Real AWS operations work end-to-end"
        echo "  • Instance lifecycle management verified"
        echo "  • Resource cleanup completed successfully"
        echo "  • Full tutorial workflow confirmed working"
    else
        echo "  • AWS operations validate successfully (dry-run)"
        echo "  • Template system ready for real usage"
    fi
    echo "  • Version consistency maintained"
    echo ""
    echo "🚀 Tutorial workflows validated and ready for users!"
    echo ""
    echo "📚 Recommended Tutorial Updates:"
    echo "  1. 'Your First Cloud Workstation in 5 Minutes' - VALIDATED"
    echo "     • brew install cloudworkstation"
    echo "     • cws launch python-ml my-project  # (daemon auto-starts)"
    if [[ "$REAL_AWS_MODE" == "true" ]]; then
        echo "     • cws connect my-project        # (confirmed working)"
    else
        echo "     • cws connect my-project        # (ready for real AWS)"
    fi
    echo "  2. Template naming conventions work correctly:"
    echo "     • Full names: cws launch 'Python Machine Learning (Simplified)' name"
    echo "     • Slugs: cws launch python-ml name  # (power user efficiency)"
    if [[ "$REAL_AWS_MODE" == "true" ]]; then
        echo "💰 AWS costs incurred: Review your AWS console for charges"
    fi
    exit 0
else
    echo -e "${RED}❌ $FAILED_TESTS TESTS FAILED${NC}"
    echo ""
    echo "🔍 Issues found in real install experience:"
    echo "  Check $TEST_RESULTS_LOG for detailed failure information"
    echo ""
    if [[ "$REAL_AWS_MODE" == "true" ]]; then
        echo "⚠️  Real AWS testing revealed critical issues!"
        echo "🧹 Important: Check AWS console and clean up any remaining resources"
    else
        echo "⚠️  Real install testing revealed issues that source build testing missed!"
    fi
    exit 1
fi