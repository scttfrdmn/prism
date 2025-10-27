# Persona-Based Integration Tests

This directory contains integration tests based on the 5 persona walkthroughs documented in `docs/USER_SCENARIOS/`.

## Purpose

These tests validate real-world user workflows against actual AWS infrastructure, ensuring that the documented persona scenarios work end-to-end.

## Test Configuration

**AWS Profile**: `aws`
**AWS Region**: `us-west-2`

## Running Tests

```bash
# Run all persona integration tests
go test ./test/integration -v -timeout 30m

# Run specific persona
go test ./test/integration -v -timeout 30m -run TestSoloResearcherPersona

# Skip integration tests (default in CI)
go test ./... -short
```

## Test Structure

Each persona test follows this structure:

1. **Setup**: Configure test environment with AWS profile and region
2. **Execute**: Run through the documented workflow step-by-step
3. **Verify**: Check that each step produces expected results
4. **Cleanup**: Delete all created resources (instances, volumes, etc.)

## Test Personas

### 1. Solo Researcher (Dr. Sarah Chen)
**File**: `personas_test.go::TestSoloResearcherPersona`
**Workflow**: Launch workspace → Enable hibernation → Daily work cycle → Cost tracking
**Duration**: ~10 minutes
**Resources**: 1 instance, 1 hibernation profile

### 2. Lab Environment (Prof. Martinez)
**File**: `personas_test.go::TestLabEnvironmentPersona`
**Workflow**: Multi-user setup → Shared storage → Team collaboration
**Duration**: ~15 minutes
**Resources**: 3 instances, 1 EFS volume, 3 research users

### 3. University Class (Prof. Thompson)
**File**: `personas_test.go::TestUniversityClassPersona`
**Workflow**: Bulk launch → Student access → Template standardization
**Duration**: ~20 minutes
**Resources**: 25 instances (simulated), shared template

### 4. Conference Workshop (Dr. Patel)
**File**: `personas_test.go::TestConferenceWorkshopPersona`
**Workflow**: Rapid deployment → Public access → Time-limited workspaces
**Duration**: ~10 minutes
**Resources**: 10 instances (simulated), auto-termination

### 5. Cross-Institutional Collaboration (Dr. Kim)
**File**: `personas_test.go::TestCrossInstitutionalPersona`
**Workflow**: Multi-profile setup → Shared EFS → Budget tracking
**Duration**: ~15 minutes
**Resources**: 2 instances, 1 shared EFS, 2 AWS profiles

## Important Notes

### Cost Considerations
These tests create real AWS resources and incur real costs. Estimated cost per full test run: ~$1.50

### Resource Cleanup
All tests include comprehensive cleanup in `defer` statements. If tests fail mid-execution, resources may remain. Check with:
```bash
./bin/prism list
./bin/prism storage list
```

### Test Isolation
Each test uses unique names with timestamps to avoid conflicts when running tests in parallel.

### CI/CD Integration
Integration tests are skipped by default in CI (`testing.Short()`). Run manually with:
```bash
go test ./test/integration -v -timeout 30m
```

## Troubleshooting

### Authentication Errors
Ensure AWS profile 'aws' is configured:
```bash
aws configure --profile aws
aws sts get-caller-identity --profile aws
```

### Timeout Errors
Some workflows (instance launch) can take 5-8 minutes. Tests have 30-minute timeout. Increase if needed:
```bash
go test ./test/integration -v -timeout 60m
```

### Resource Conflicts
If test fails to cleanup, manually delete resources:
```bash
./bin/prism delete test-* --force
./bin/prism storage delete test-* --force
```

## Adding New Tests

1. Read persona walkthrough in `docs/USER_SCENARIOS/`
2. Create test function following naming pattern: `TestXXXPersona`
3. Use helper functions from `helpers.go`
4. Include comprehensive cleanup
5. Document expected duration and resource usage

## Success Criteria

A successful persona test should:
- ✅ Complete all documented workflow steps
- ✅ Verify outputs match expected values
- ✅ Clean up all created resources
- ✅ Complete within expected timeframe
- ✅ Work on first run (no manual setup required)
