#!/bin/bash
# Real AWS Validation Script for Real Testers
# Purpose: Validate critical user workflows against real AWS before tester release
# Usage: ./scripts/validate_real_aws.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
RESULTS_DIR="./test-results/aws-validation-$(date +%Y%m%d_%H%M%S)"
LOG_FILE="$RESULTS_DIR/validation.log"
FINDINGS_FILE="$RESULTS_DIR/FINDINGS.md"
TEST_PREFIX="real-test-$(date +%s)"

# Create results directory
mkdir -p "$RESULTS_DIR"

# Initialize findings document
cat > "$FINDINGS_FILE" << EOF
# AWS Validation Findings
**Date**: $(date)
**Test Run ID**: $TEST_PREFIX

## Summary
- **Total Tests**: TBD
- **Passed**: TBD
- **Failed**: TBD
- **Warnings**: TBD

---

## Test Results

EOF

# Logging functions
log() {
    echo -e "${BLUE}[$(date +%H:%M:%S)]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[$(date +%H:%M:%S)] ✅ $1${NC}" | tee -a "$LOG_FILE"
    echo "- ✅ **PASS**: $1" >> "$FINDINGS_FILE"
}

log_fail() {
    echo -e "${RED}[$(date +%H:%M:%S)] ❌ $1${NC}" | tee -a "$LOG_FILE"
    echo "- ❌ **FAIL**: $1" >> "$FINDINGS_FILE"
    echo "  - Details: $2" >> "$FINDINGS_FILE"
}

log_warn() {
    echo -e "${YELLOW}[$(date +%H:%M:%S)] ⚠️  $1${NC}" | tee -a "$LOG_FILE"
    echo "- ⚠️ **WARN**: $1" >> "$FINDINGS_FILE"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."

    if [ -z "$AWS_PROFILE" ]; then
        log_fail "AWS_PROFILE not set" "Export AWS_PROFILE=aws before running"
        exit 1
    fi

    if ! command -v aws &> /dev/null; then
        log_fail "AWS CLI not installed" "Install AWS CLI v2"
        exit 1
    fi

    if [ ! -f "./bin/cws" ]; then
        log_fail "CloudWorkstation binary not found" "Run 'make build' first"
        exit 1
    fi

    # Test AWS credentials
    if ! aws sts get-caller-identity --profile "$AWS_PROFILE" &> /dev/null; then
        log_fail "AWS credentials invalid" "Check AWS_PROFILE=$AWS_PROFILE credentials"
        exit 1
    fi

    log_success "Prerequisites check passed"
}

# Test 1: First-Time Setup Experience
test_first_time_setup() {
    log "========================================="
    log "TEST 1: First-Time Setup Experience"
    log "========================================="

    # Backup existing config
    if [ -d ~/.cloudworkstation ]; then
        log "Backing up existing config..."
        mv ~/.cloudworkstation ~/.cloudworkstation.backup.$(date +%s)
    fi

    # Test: Can list templates without setup
    log "Testing: List templates on fresh install..."
    if ./bin/prism templates &> "$RESULTS_DIR/test1_templates.log"; then
        log_success "Templates list worked on first run"
    else
        log_fail "Templates list failed on first run" "See $RESULTS_DIR/test1_templates.log"
        cat "$RESULTS_DIR/test1_templates.log" >> "$FINDINGS_FILE"
    fi

    # Test: Daemon auto-started?
    log "Testing: Daemon auto-start..."
    if ./bin/prism daemon status &> "$RESULTS_DIR/test1_daemon.log"; then
        log_success "Daemon auto-started successfully"
    else
        log_warn "Daemon not running (may require manual start)"
        cat "$RESULTS_DIR/test1_daemon.log" >> "$FINDINGS_FILE"
    fi
}

# Test 2: Launch First Instance (Critical Path)
test_launch_first_instance() {
    log "========================================="
    log "TEST 2: Launch First Instance"
    log "========================================="

    INSTANCE_NAME="${TEST_PREFIX}-launch"

    log "Launching instance: $INSTANCE_NAME"
    log "Template: template (Basic Research - simplest, fastest)"

    START_TIME=$(date +%s)

    if timeout 600 ./bin/prism launch template "$INSTANCE_NAME" --size S &> "$RESULTS_DIR/test2_launch.log"; then
        END_TIME=$(date +%s)
        DURATION=$((END_TIME - START_TIME))
        log_success "Instance launched successfully in ${DURATION}s"

        # Wait a bit for instance to be fully ready
        log "Waiting 60s for instance to be fully ready..."
        sleep 60

        # Test: Can we get instance info?
        if ./bin/prism list | grep -q "$INSTANCE_NAME"; then
            log_success "Instance appears in list"
        else
            log_fail "Instance not in list" "See $RESULTS_DIR/test2_list.log"
        fi

        # Store instance name for later tests
        echo "$INSTANCE_NAME" > "$RESULTS_DIR/launched_instance.txt"

    else
        END_TIME=$(date +%s)
        DURATION=$((END_TIME - START_TIME))
        log_fail "Instance launch failed or timed out after ${DURATION}s" "See $RESULTS_DIR/test2_launch.log"
        cat "$RESULTS_DIR/test2_launch.log" >> "$FINDINGS_FILE"
        return 1
    fi
}

# Test 3: Instance Lifecycle (Stop/Start/Delete)
test_instance_lifecycle() {
    log "========================================="
    log "TEST 3: Instance Lifecycle Management"
    log "========================================="

    # Get instance from previous test
    if [ ! -f "$RESULTS_DIR/launched_instance.txt" ]; then
        log_warn "No instance from previous test, creating new one..."
        INSTANCE_NAME="${TEST_PREFIX}-lifecycle"
        if ! timeout 600 ./bin/prism launch template "$INSTANCE_NAME" --size S &> "$RESULTS_DIR/test3_launch.log"; then
            log_fail "Could not launch instance for lifecycle test"
            return 1
        fi
        sleep 60
    else
        INSTANCE_NAME=$(cat "$RESULTS_DIR/launched_instance.txt")
    fi

    log "Testing lifecycle for instance: $INSTANCE_NAME"

    # Test: Stop
    log "Testing: Stop instance..."
    if timeout 180 ./bin/prism stop "$INSTANCE_NAME" &> "$RESULTS_DIR/test3_stop.log"; then
        log_success "Instance stopped successfully"
        sleep 30  # Wait for stop to propagate
    else
        log_fail "Instance stop failed" "See $RESULTS_DIR/test3_stop.log"
    fi

    # Test: Start
    log "Testing: Start instance..."
    if timeout 180 ./bin/prism start "$INSTANCE_NAME" &> "$RESULTS_DIR/test3_start.log"; then
        log_success "Instance started successfully"
        sleep 60  # Wait for start to complete
    else
        log_fail "Instance start failed" "See $RESULTS_DIR/test3_start.log"
    fi

    # Test: Delete
    log "Testing: Delete instance..."
    if timeout 180 ./bin/prism delete "$INSTANCE_NAME" --force &> "$RESULTS_DIR/test3_delete.log"; then
        log_success "Instance deleted successfully"
        rm -f "$RESULTS_DIR/launched_instance.txt"
    else
        log_fail "Instance delete failed" "See $RESULTS_DIR/test3_delete.log"
    fi
}

# Test 4: EFS Storage Persistence
test_efs_storage() {
    log "========================================="
    log "TEST 4: EFS Storage Persistence"
    log "========================================="

    VOLUME_NAME="${TEST_PREFIX}-vol"
    INSTANCE1="${TEST_PREFIX}-efs1"
    INSTANCE2="${TEST_PREFIX}-efs2"
    TEST_FILE="test-$(date +%s).txt"
    TEST_DATA="CloudWorkstation Test Data - $(date)"

    # Create volume
    log "Creating EFS volume: $VOLUME_NAME..."
    if timeout 300 ./bin/prism volume create "$VOLUME_NAME" &> "$RESULTS_DIR/test4_create_volume.log"; then
        log_success "EFS volume created"
    else
        log_fail "EFS volume creation failed" "See $RESULTS_DIR/test4_create_volume.log"
        return 1
    fi

    # Launch first instance
    log "Launching first instance: $INSTANCE1..."
    if ! timeout 600 ./bin/prism launch template "$INSTANCE1" --size S &> "$RESULTS_DIR/test4_launch1.log"; then
        log_fail "First instance launch failed"
        return 1
    fi
    sleep 60

    # Mount volume
    log "Mounting volume to first instance..."
    if timeout 120 ./bin/prism volume mount "$VOLUME_NAME" "$INSTANCE1" &> "$RESULTS_DIR/test4_mount1.log"; then
        log_success "Volume mounted to first instance"
    else
        log_fail "Volume mount failed" "See $RESULTS_DIR/test4_mount1.log"
    fi

    log_warn "MANUAL STEP REQUIRED: SSH to $INSTANCE1 and create test file:"
    log_warn "  echo '$TEST_DATA' > /mnt/${VOLUME_NAME}/${TEST_FILE}"
    log_warn "Press Enter when done..."
    read -r

    # Delete first instance
    log "Deleting first instance..."
    timeout 180 ./bin/prism delete "$INSTANCE1" --force &> "$RESULTS_DIR/test4_delete1.log"

    # Launch second instance
    log "Launching second instance: $INSTANCE2..."
    if ! timeout 600 ./bin/prism launch template "$INSTANCE2" --size S &> "$RESULTS_DIR/test4_launch2.log"; then
        log_fail "Second instance launch failed"
        return 1
    fi
    sleep 60

    # Mount same volume
    log "Mounting same volume to second instance..."
    if timeout 120 ./bin/prism volume mount "$VOLUME_NAME" "$INSTANCE2" &> "$RESULTS_DIR/test4_mount2.log"; then
        log_success "Volume mounted to second instance"
    else
        log_fail "Volume mount to second instance failed" "See $RESULTS_DIR/test4_mount2.log"
    fi

    log_warn "MANUAL VERIFICATION REQUIRED: SSH to $INSTANCE2 and verify file exists:"
    log_warn "  cat /mnt/${VOLUME_NAME}/${TEST_FILE}"
    log_warn "Expected: $TEST_DATA"
    log_warn "Did file exist with correct content? (y/n)"
    read -r answer

    if [ "$answer" = "y" ]; then
        log_success "EFS persistence verified"
    else
        log_fail "EFS persistence verification failed" "Data not preserved across instances"
    fi

    # Cleanup
    log "Cleaning up test resources..."
    timeout 180 ./bin/prism delete "$INSTANCE2" --force &> "$RESULTS_DIR/test4_delete2.log"
    timeout 180 ./bin/prism volume delete "$VOLUME_NAME" --force &> "$RESULTS_DIR/test4_delete_volume.log"
}

# Test 5: Template Validation
test_templates() {
    log "========================================="
    log "TEST 5: Template Validation"
    log "========================================="

    TEMPLATES=("template" "python-ml")

    for template in "${TEMPLATES[@]}"; do
        log "Testing template: $template"
        INSTANCE_NAME="${TEST_PREFIX}-tmpl-${template}"

        # Launch
        log "Launching $template template..."
        if timeout 600 ./bin/prism launch "$template" "$INSTANCE_NAME" --size S &> "$RESULTS_DIR/test5_${template}_launch.log"; then
            log_success "Template $template launched"

            log_warn "MANUAL VERIFICATION: SSH to $INSTANCE_NAME and verify software"
            log_warn "Press Enter when done..."
            read -r

            # Cleanup
            timeout 180 ./bin/prism delete "$INSTANCE_NAME" --force &> "$RESULTS_DIR/test5_${template}_delete.log"
        else
            log_fail "Template $template launch failed" "See $RESULTS_DIR/test5_${template}_launch.log"
        fi
    done
}

# Test 6: Error Handling
test_error_handling() {
    log "========================================="
    log "TEST 6: Error Handling"
    log "========================================="

    # Test: Non-existent instance
    log "Testing: Operation on non-existent instance..."
    if ./bin/prism stop "nonexistent-instance-$(date +%s)" &> "$RESULTS_DIR/test6_nonexistent.log"; then
        log_fail "Should have failed on non-existent instance"
    else
        # Check if error message is user-friendly
        if grep -q "not found" "$RESULTS_DIR/test6_nonexistent.log"; then
            log_success "Good error message for non-existent instance"
        else
            log_warn "Error message could be more user-friendly"
            cat "$RESULTS_DIR/test6_nonexistent.log" >> "$FINDINGS_FILE"
        fi
    fi
}

# Main execution
main() {
    log "========================================="
    log "CloudWorkstation Real AWS Validation"
    log "========================================="
    log "Test Run ID: $TEST_PREFIX"
    log "Results Directory: $RESULTS_DIR"
    log "AWS Profile: $AWS_PROFILE"
    log "AWS Region: ${AWS_TEST_REGION:-default}"
    log "========================================="

    check_prerequisites

    echo ""
    log "Starting validation tests..."
    log "NOTE: Some tests require manual verification"
    echo ""

    test_first_time_setup
    echo ""

    test_launch_first_instance
    echo ""

    test_instance_lifecycle
    echo ""

    # Ask if user wants to run longer tests
    log "Continue with longer tests (EFS, Templates)? (y/n)"
    read -r answer
    if [ "$answer" = "y" ]; then
        test_efs_storage
        echo ""

        test_templates
        echo ""
    fi

    test_error_handling
    echo ""

    log "========================================="
    log "Validation Complete!"
    log "========================================="
    log "Results saved to: $RESULTS_DIR"
    log "Findings document: $FINDINGS_FILE"
    log "========================================="

    # Update findings summary
    PASS_COUNT=$(grep -c "✅ \*\*PASS\*\*" "$FINDINGS_FILE" || echo "0")
    FAIL_COUNT=$(grep -c "❌ \*\*FAIL\*\*" "$FINDINGS_FILE" || echo "0")
    WARN_COUNT=$(grep -c "⚠️ \*\*WARN\*\*" "$FINDINGS_FILE" || echo "0")

    sed -i.bak "s/- \*\*Total Tests\*\*: TBD/- **Total Tests**: $((PASS_COUNT + FAIL_COUNT + WARN_COUNT))/" "$FINDINGS_FILE"
    sed -i.bak "s/- \*\*Passed\*\*: TBD/- **Passed**: $PASS_COUNT/" "$FINDINGS_FILE"
    sed -i.bak "s/- \*\*Failed\*\*: TBD/- **Failed**: $FAIL_COUNT/" "$FINDINGS_FILE"
    sed -i.bak "s/- \*\*Warnings\*\*: TBD/- **Warnings**: $WARN_COUNT/" "$FINDINGS_FILE"
    rm "$FINDINGS_FILE.bak"

    log "Summary: $PASS_COUNT passed, $FAIL_COUNT failed, $WARN_COUNT warnings"

    if [ "$FAIL_COUNT" -gt 0 ]; then
        log_fail "Validation failed - $FAIL_COUNT critical issues found"
        log "Review findings at: $FINDINGS_FILE"
        exit 1
    else
        log_success "Validation passed - Ready for real testers!"
    fi
}

# Run main
main
