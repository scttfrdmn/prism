# Real Tester AWS Validation Plan

**Priority**: CRITICAL - Real testers (non-experts) incoming
**Timeline**: Before tester access
**Focus**: Functional validation against real AWS

---

## Executive Summary

**Current Status**:
- ✅ Unit tests: 96.8% passing (mocks work)
- ✅ Integration test framework: EXISTS and comprehensive
- ⚠️ **CRITICAL GAP**: Need to RUN integration tests against real AWS
- ⚠️ **BLOCKING**: Real testers need working functionality, not passing mocks

**Priority Actions**:
1. Run existing AWS integration tests (HIGH)
2. Create real tester validation checklist (HIGH)
3. Test critical user workflows end-to-end (CRITICAL)
4. Document any AWS-specific issues found (HIGH)

---

## Phase 1: Run Existing AWS Integration Tests (2-3 hours)

### Status: Integration tests already exist but haven't been run

**Files**:
- `/internal/cli/integration_aws_test.go` - CLI against real AWS
- `/pkg/aws/manager_integration_test.go` - AWS manager integration
- `/pkg/aws/integration_test.go` - AWS operations
- `/pkg/ami/integration_test.go` - AMI operations
- `/pkg/research/integration_test.go` - Research user operations

### Run Integration Tests

```bash
# Set up AWS environment
export AWS_PROFILE=aws  # Your test AWS profile
export AWS_TEST_REGION=us-east-1  # Or your preferred region
export RUN_AWS_TESTS=true

# Run all AWS integration tests
go test -tags aws_integration ./internal/cli/ -v -run TestAWS
go test -tags aws_integration ./pkg/aws/ -v
go test -tags aws_integration ./pkg/ami/ -v

# Check for failures and document issues
```

### Safety Features (Already Built In)
- ✅ Test resources use `cwstest-` prefix
- ✅ Tagged: `CreatedBy=CloudWorkstationIntegrationTest`
- ✅ Automatic cleanup on teardown
- ✅ Cost-conscious (smallest instances)
- ✅ Resource limits to prevent runaway costs

### Expected Outcomes
1. Identify AWS permission issues
2. Find real-world API failures
3. Discover timing/race conditions
4. Validate cleanup works properly

---

## Phase 2: Critical User Workflows for Real Testers (4-6 hours)

### Workflow 1: First-Time Setup (CRITICAL)
**User**: Never used CloudWorkstation before, not AWS expert

**Test Steps**:
1. Install binary (CLI or GUI)
2. First run - daemon auto-starts?
3. AWS credential detection works?
4. Region selection intuitive?
5. No cryptic error messages?

**Success Criteria**:
- ✅ Works without manual daemon management
- ✅ Clear error messages for missing AWS credentials
- ✅ Helpful guidance for first-time setup
- ✅ No systemd knowledge required

**Test Script**:
```bash
# Clean state
rm -rf ~/.cloudworkstation

# First run
./bin/cws templates

# Expected: Daemon auto-starts, templates display, no errors
# Reality: Document what actually happens
```

### Workflow 2: Launch First Instance (CRITICAL)
**User**: Researcher wants to start working immediately

**Test Steps**:
1. Browse templates
2. Pick a template (e.g., Python ML)
3. Launch instance: `cws launch python-ml my-project`
4. Wait for instance to be ready
5. Connect via SSH
6. Verify software works (Python, Jupyter, etc.)

**Success Criteria**:
- ✅ Launch completes in 5-10 minutes
- ✅ Progress feedback is clear
- ✅ Connection info is provided
- ✅ Software actually works as advertised
- ✅ No AWS knowledge needed

**Test Script**:
```bash
# Launch
time ./bin/cws launch python-ml test-instance

# Expected output:
# - Clear progress messages
# - Estimated time
# - Connection info when ready
# - No errors

# Connect
./bin/cws connect test-instance

# Verify
ssh <connection-info>
python3 --version
jupyter --version
```

### Workflow 3: Stop/Start/Delete Lifecycle (HIGH)
**User**: Needs to manage costs, stop when not using

**Test Steps**:
1. Stop instance: `cws stop test-instance`
2. List instances (verify stopped)
3. Start instance: `cws start test-instance`
4. Wait for ready
5. Delete instance: `cws delete test-instance`
6. Verify cleanup

**Success Criteria**:
- ✅ Stop/start work reliably
- ✅ State transitions are clear
- ✅ Delete removes everything (no orphaned resources)
- ✅ Cost implications are clear

**Test Script**:
```bash
# Stop
./bin/cws stop test-instance
# Verify: Instance state shows "stopped"

# List
./bin/cws list
# Verify: Instance shows as stopped, not running

# Start
./bin/cws start test-instance
# Verify: Instance comes back up

# Delete
./bin/cws delete test-instance
# Verify: Instance gone, volumes cleaned up
```

### Workflow 4: Storage - EFS Volume (HIGH)
**User**: Needs persistent data across instances

**Test Steps**:
1. Create EFS volume: `cws volume create shared-data`
2. Mount to instance: `cws volume mount shared-data test-instance`
3. SSH in, create file in mounted directory
4. Delete instance
5. Launch new instance
6. Mount same volume
7. Verify file still exists

**Success Criteria**:
- ✅ Volume creation is fast
- ✅ Mounting works reliably
- ✅ Data persists across instances
- ✅ Multi-instance sharing works

**Test Script**:
```bash
# Create volume
./bin/cws volume create shared-data

# Launch instance
./bin/cws launch ubuntu-base test1

# Mount volume
./bin/cws volume mount shared-data test1

# Create test file via SSH
ssh <test1-connection> "echo 'test data' > /mnt/shared-data/test.txt"

# Delete instance
./bin/cws delete test1

# Launch new instance
./bin/cws launch ubuntu-base test2

# Mount same volume
./bin/cws volume mount shared-data test2

# Verify file exists
ssh <test2-connection> "cat /mnt/shared-data/test.txt"
# Expected: "test data"
```

### Workflow 5: Template Validation (MEDIUM)
**User**: Wants to know templates actually work

**Test Steps**:
1. Pick 3-5 core templates
2. Launch each template
3. SSH in and verify software
4. Run basic functionality tests

**Templates to Test**:
- `ubuntu-base` - Basic Ubuntu
- `python-ml` - Python + Jupyter + ML libraries
- `r-research` - R + RStudio + tidyverse
- `collaborative-workspace` - Multi-user setup
- One GPU template (if available)

**Success Criteria**:
- ✅ All advertised software is installed
- ✅ Software versions are reasonable
- ✅ Services start automatically (Jupyter, RStudio)
- ✅ Ports are accessible

**Test Script**:
```bash
# For each template:
./bin/cws launch <template-name> test-<template>

# SSH in and verify
ssh <connection-info>

# Python ML template
python3 -c "import numpy, pandas, matplotlib, jupyter; print('All imports work')"
jupyter lab --version

# R Research template
R --version
rstudio-server status  # or check service

# etc.
```

### Workflow 6: Hibernation & Cost Management (HIGH)
**User**: Worried about AWS costs, wants to hibernate

**Test Steps**:
1. Launch instance
2. Hibernate: `cws hibernate test-instance`
3. Verify hibernation status
4. Resume: `cws resume test-instance`
5. Verify work resumed (check for running processes)

**Success Criteria**:
- ✅ Hibernation works (not just stop)
- ✅ Resume restores RAM state
- ✅ Faster than cold start
- ✅ Cost savings are clear

**Test Script**:
```bash
# Launch
./bin/cws launch python-ml test-hibernate

# Start some work via SSH
ssh <connection> "python3 -c 'import time; time.sleep(1000)' &"

# Hibernate
./bin/cws hibernate test-hibernate

# Wait a bit
sleep 60

# Resume
./bin/cws resume test-hibernate

# Check if process still running
ssh <connection> "ps aux | grep python"
# Expected: Python process still there
```

---

## Phase 3: GUI Real AWS Validation (3-4 hours)

### GUI Workflow 1: Visual Instance Management
**User**: Prefers GUI, not comfortable with CLI

**Test Steps**:
1. Launch GUI: `./bin/cws-gui`
2. Browse templates visually
3. Launch instance via GUI
4. Monitor instance status in GUI
5. Stop/start via GUI buttons
6. Delete via GUI

**Success Criteria**:
- ✅ GUI reflects real AWS state
- ✅ Polling/refresh works
- ✅ Actions trigger real AWS operations
- ✅ Error messages are user-friendly
- ✅ No need to drop to CLI

### GUI Workflow 2: Multi-Instance Dashboard
**User**: Running multiple projects simultaneously

**Test Steps**:
1. Launch 3-4 instances
2. View in GUI dashboard
3. Verify real-time status updates
4. Manage multiple instances simultaneously
5. Check cost estimates

**Success Criteria**:
- ✅ Dashboard shows all instances
- ✅ Real-time updates work
- ✅ Can manage multiple instances
- ✅ Cost tracking is accurate

---

## Phase 4: Error Handling & Edge Cases (2-3 hours)

### Test: AWS Permission Issues
**Simulate**: Remove permissions, test error messages

**Test Cases**:
1. No EC2 permissions
2. No EFS permissions
3. Invalid AWS credentials
4. Expired session token
5. Region with no capacity

**Success Criteria**:
- ✅ Clear error messages (not AWS API jargon)
- ✅ Helpful suggestions for resolution
- ✅ No crashes or stack traces shown to user

### Test: Network Issues
**Simulate**: Network interruptions during operations

**Test Cases**:
1. Launch instance, kill network mid-launch
2. Stop instance, disconnect during operation
3. SSH connection fails

**Success Criteria**:
- ✅ Graceful failure handling
- ✅ State recovers correctly
- ✅ Retry logic works

### Test: Resource Limits
**Simulate**: AWS account limits hit

**Test Cases**:
1. EC2 instance limit reached
2. EFS mount target limit reached
3. VPC limits
4. IAM limits

**Success Criteria**:
- ✅ Clear error about limit
- ✅ Guidance on how to fix
- ✅ No partial resources left behind

---

## Phase 5: Real Tester Checklist (Pre-Release)

### Before Giving Access to Real Testers

**Required**:
- [ ] All Phase 2 critical workflows pass
- [ ] At least 2 templates fully validated
- [ ] Daemon auto-start works on fresh install
- [ ] Error messages are user-friendly
- [ ] Documentation for common issues

**Recommended**:
- [ ] GUI tested against real AWS
- [ ] Hibernation/resume validated
- [ ] Multi-instance scenarios work
- [ ] Cost tracking is accurate
- [ ] Cleanup works (no orphaned resources)

**Nice to Have**:
- [ ] All templates validated
- [ ] Edge cases handled gracefully
- [ ] Performance benchmarks documented

### Tester Onboarding Materials Needed

1. **Quick Start Guide** (non-expert friendly)
   - How to install
   - How to configure AWS credentials (simple)
   - First launch walkthrough
   - Common issues & solutions

2. **Troubleshooting Guide** (real issues found during testing)
   - Daemon won't start → solution
   - AWS connection fails → solution
   - Instance won't launch → solution
   - Can't connect via SSH → solution

3. **Feedback Template**
   - What were you trying to do?
   - What happened?
   - What did you expect?
   - Error messages (copy/paste)

---

## Phase 6: Automated Real AWS Test Suite (Future)

### CI/CD Integration
**Goal**: Run AWS integration tests automatically

**Implementation**:
```bash
# GitHub Actions workflow (future)
name: AWS Integration Tests

on:
  push:
    branches: [main]
  schedule:
    - cron: '0 6 * * *'  # Daily at 6 AM

jobs:
  aws-integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: aws-actions/configure-aws-credentials@v2
        with:
          role-to-assume: ${{ secrets.AWS_TEST_ROLE }}
          aws-region: us-east-1
      - name: Run Integration Tests
        run: |
          export RUN_AWS_TESTS=true
          go test -tags aws_integration ./... -v
```

**Benefits**:
- Catch regressions immediately
- Validate against real AWS daily
- No manual testing needed for each change

---

## Timeline & Priorities

### IMMEDIATE (Before Real Testers - 8-12 hours)

**Day 1 (4-5 hours)**:
1. Run existing AWS integration tests (2-3 hours)
2. Test Workflow 1: First-Time Setup (1 hour)
3. Test Workflow 2: Launch First Instance (1-2 hours)

**Day 2 (4-6 hours)**:
1. Test Workflow 3: Stop/Start/Delete (1-2 hours)
2. Test Workflow 4: Storage - EFS Volume (2-3 hours)
3. Test Workflow 5: Template Validation (2-3 core templates)

**Day 3 (Optional - if time permits)**:
1. GUI validation against real AWS (3-4 hours)
2. Error handling edge cases (2-3 hours)
3. Create tester documentation (2-3 hours)

### SHORT-TERM (First week with testers)

1. Monitor tester feedback closely
2. Fix critical issues immediately
3. Update documentation based on real issues
4. Iterate on error messages

### MEDIUM-TERM (1-2 months)

1. Add more templates based on tester needs
2. Improve error handling based on real scenarios
3. Performance optimizations
4. Set up automated AWS integration testing

---

## Success Criteria for Real Tester Release

### Minimum Bar (Must Have)
- ✅ First-time setup works without CLI expertise
- ✅ Launch instance workflow is smooth
- ✅ Stop/start/delete work reliably
- ✅ At least 2 templates fully functional
- ✅ Error messages are helpful, not cryptic
- ✅ Daemon auto-start works
- ✅ No AWS knowledge required for basic use

### Good Bar (Should Have)
- ✅ 5+ templates validated
- ✅ EFS storage works end-to-end
- ✅ Hibernation works reliably
- ✅ GUI fully functional against real AWS
- ✅ Cost tracking accurate
- ✅ Multi-instance scenarios work

### Great Bar (Nice to Have)
- ✅ All templates validated
- ✅ Edge cases handled gracefully
- ✅ Performance is excellent
- ✅ Comprehensive documentation
- ✅ Automated testing in place

---

## Risk Assessment

### High Risk Areas
1. **Daemon Auto-Start**: Most likely to confuse non-expert users
2. **AWS Credentials**: Common pain point for new users
3. **Template Provision Scripts**: May fail in various ways
4. **Network/SSH Issues**: Hard to debug for non-experts
5. **Cost Surprises**: Users might not understand AWS billing

### Mitigation Strategies
1. **Extensive Testing**: Run every workflow multiple times
2. **Clear Error Messages**: User-facing, not developer-facing
3. **Good Documentation**: Step-by-step for common issues
4. **Quick Response**: Monitor tester feedback closely
5. **Rollback Plan**: Easy way to downgrade if issues found

---

## Commands for Immediate Action

```bash
# 1. Set up AWS test environment
export AWS_PROFILE=aws
export AWS_TEST_REGION=us-east-1
export RUN_AWS_TESTS=true

# 2. Run existing integration tests
echo "Running AWS integration tests..."
go test -tags aws_integration ./internal/cli/ -v -run TestAWS 2>&1 | tee aws_tests_$(date +%Y%m%d_%H%M%S).log

# 3. Test critical workflow - First launch
echo "Testing first-time launch workflow..."
rm -rf ~/.cloudworkstation  # Clean state
./bin/cws templates
./bin/cws launch python-ml real-test-$(date +%s)

# 4. Test stop/start/delete
echo "Testing instance lifecycle..."
INSTANCE_NAME="real-test-lifecycle-$(date +%s)"
./bin/cws launch ubuntu-base $INSTANCE_NAME
sleep 300  # Wait for launch
./bin/cws stop $INSTANCE_NAME
sleep 60
./bin/cws start $INSTANCE_NAME
sleep 120
./bin/cws delete $INSTANCE_NAME

# 5. Test EFS storage
echo "Testing EFS storage workflow..."
VOLUME_NAME="real-test-volume-$(date +%s)"
INSTANCE1="real-test-i1-$(date +%s)"
INSTANCE2="real-test-i2-$(date +%s)"

./bin/cws volume create $VOLUME_NAME
./bin/cws launch ubuntu-base $INSTANCE1
./bin/cws volume mount $VOLUME_NAME $INSTANCE1
# Manual: SSH in, create test file
./bin/cws delete $INSTANCE1
./bin/cws launch ubuntu-base $INSTANCE2
./bin/cws volume mount $VOLUME_NAME $INSTANCE2
# Manual: SSH in, verify file exists

# 6. Document all findings
echo "Create findings document at: ./docs/AWS_VALIDATION_FINDINGS_$(date +%Y%m%d).md"
```

---

## Next Steps

1. **IMMEDIATE**: Run the commands above
2. **DOCUMENT**: Create findings document with any failures
3. **FIX**: Address critical issues found
4. **ITERATE**: Re-test until workflows are smooth
5. **RELEASE**: Only after critical workflows pass

**Target**: Real testers should have a smooth, AWS-expertise-free experience

---

**Status**: READY TO EXECUTE
**Priority**: CRITICAL
**Owner**: You
**Deadline**: Before real tester access
**Success**: All critical workflows pass against real AWS
