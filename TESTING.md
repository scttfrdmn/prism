# CloudWorkstation Testing Guide

This guide covers the comprehensive testing strategy for CloudWorkstation, including unit tests, integration tests with LocalStack, and code coverage analysis.

## Test Coverage Targets

- **pkg/aws**: 85% (most critical package handling money and cloud resources)
- **pkg/daemon**: 80% (HTTP API server)
- **pkg/api**: 75% (API client library)
- **Overall project**: 75%

## Current Coverage Status

| Package | Current Coverage | Target | Status |
|---------|------------------|---------|---------|
| pkg/aws | 49.5% | 85% | 🟡 In Progress |
| pkg/daemon | 27.8% | 80% | 🟡 In Progress |
| pkg/api | 58.3% | 75% | 🟡 Approaching |
| pkg/state | 76.1% | 75% | ✅ Complete |
| pkg/types | 100% | 75% | ✅ Complete |

## Test Types

### 1. Unit Tests

**Location**: `*_test.go` files alongside source code
**Command**: `go test ./...`

Unit tests cover:
- Helper functions and utilities
- Pricing calculations and discounts
- Template validation
- Error handling
- Business logic without external dependencies

**Key Test Files:**
- `pkg/aws/manager_test.go` - Comprehensive AWS manager tests
- `pkg/daemon/server_test.go` - HTTP handler tests
- `pkg/state/manager_test.go` - State management tests
- `pkg/types/types_test.go` - Type validation tests

### 2. Integration Tests

**Location**: `pkg/aws/integration_test.go`
**Command**: `INTEGRATION_TESTS=1 go test ./pkg/aws -tags=integration -v`

Integration tests use LocalStack to provide:
- Real AWS API testing without actual cloud costs
- Complete instance lifecycle testing (launch, stop, start, delete)
- EBS and EFS volume operations
- Error handling with real AWS errors
- Multi-instance management

**Prerequisites:**
- Docker and Docker Compose installed
- LocalStack container running

### 3. Test Coverage Analysis

**Command**: `go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out`

Generates detailed HTML coverage reports showing:
- Line-by-line coverage
- Function coverage
- Package-level summaries
- Uncovered code paths

## Running Tests

### Basic Unit Tests
```bash
# Run all unit tests
go test ./...

# Run with coverage
go test ./... -coverprofile=coverage.out -covermode=atomic

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

### Integration Tests with LocalStack

```bash
# Start LocalStack
docker-compose -f docker-compose.test.yml up -d localstack

# Wait for LocalStack to be ready (10 seconds)
sleep 10

# Run integration tests
INTEGRATION_TESTS=1 go test ./pkg/aws -tags=integration -v

# Stop LocalStack
docker-compose -f docker-compose.test.yml down
```

### Individual Package Testing

```bash
# Test specific packages
go test ./pkg/aws -coverprofile=aws_coverage.out
go test ./pkg/daemon -coverprofile=daemon_coverage.out
go test ./pkg/api -coverprofile=api_coverage.out
```

## LocalStack Setup

LocalStack provides a local AWS cloud stack for testing:

**Services Used:**
- EC2 (instance management)
- EFS (file system volumes) 
- STS (security token service)
- IAM (basic permissions)

**Configuration** (`docker-compose.test.yml`):
```yaml
services:
  localstack:
    image: localstack/localstack:3.0
    ports:
      - "127.0.0.1:4566:4566"
    environment:
      - SERVICES=ec2,efs,sts,iam
      - AWS_DEFAULT_REGION=us-east-1
      - AWS_ACCESS_KEY_ID=test
      - AWS_SECRET_ACCESS_KEY=test
```

**Endpoint**: http://localhost:4566

## Test Categories

### 1. AWS Manager Tests (`pkg/aws/manager_test.go`)

**Pricing Tests:**
- Regional pricing multipliers
- Instance type cost calculations
- Volume pricing (EBS, EFS)
- Discount combinations
- Cost caching logic

**Template Tests:**
- Template validation
- Architecture mapping
- AMI selection by region
- Instance type selection

**Helper Function Tests:**
- Size parsing (XS, S, M, L, XL → GB)
- Performance parameter calculation
- User data manipulation
- Error handling

### 2. Daemon Server Tests (`pkg/daemon/server_test.go`)

**HTTP Handler Tests:**
- Method validation (GET, POST, etc.)
- Request routing
- JSON request/response handling
- Error response formatting
- Path parsing

**API Endpoint Tests:**
- `/api/v1/ping` - Health check
- `/api/v1/status` - Daemon status
- `/api/v1/instances` - Instance operations
- `/api/v1/volumes` - Volume operations
- `/api/v1/storage` - Storage operations

### 3. Integration Tests (`pkg/aws/integration_test.go`)

**Instance Lifecycle:**
- Launch instances with different templates
- Start/stop/delete operations
- Connection info retrieval
- Multi-instance management

**Volume Operations:**
- EFS volume creation/deletion
- EBS volume creation/deletion
- Storage attachment/detachment

**Error Handling:**
- Invalid template handling
- Nonexistent resource operations
- AWS API error propagation

## Coverage Improvement Strategies

### For AWS Package (Target: 85%)

**Currently Tested (49.5%):**
✅ Pricing calculations and regional multipliers
✅ Template validation and architecture mapping
✅ Helper functions (parsing, validation)
✅ Discount application logic
✅ Billing information handling

**Needs Integration Testing:**
🔄 Instance launch/management operations
🔄 Volume creation/management operations
🔄 AWS API error handling
🔄 Network and security group creation

**Strategy**: Use LocalStack integration tests to cover the actual AWS operations that require API calls.

### For Daemon Package (Target: 80%)

**Currently Tested (27.8%):**
✅ HTTP method validation
✅ Request routing and path parsing
✅ JSON error responses
✅ Basic handler functionality

**Needs More Coverage:**
🔄 Complete request/response cycles
🔄 State management integration
🔄 AWS manager integration
🔄 Middleware functionality

**Strategy**: Add comprehensive handler tests with mock dependencies.

## Continuous Integration

**Recommended CI Pipeline:**
1. **Lint**: `golangci-lint run`
2. **Unit Tests**: `go test ./... -coverprofile=coverage.out`
3. **Integration Tests**: LocalStack + `INTEGRATION_TESTS=1 go test ./pkg/aws -tags=integration`
4. **Coverage Analysis**: Fail if below targets
5. **Build**: Ensure all binaries build successfully

**Environment Variables:**
- `INTEGRATION_TESTS=1` - Enable integration tests
- `AWS_ENDPOINT_URL=http://localhost:4566` - LocalStack endpoint

## Debugging Tests

### Verbose Output
```bash
go test ./pkg/aws -v  # Verbose test output
go test ./pkg/aws -v -run TestSpecificFunction  # Run specific test
```

### LocalStack Debugging
```bash
# View LocalStack logs
docker-compose -f docker-compose.test.yml logs -f localstack

# Check LocalStack health
curl http://localhost:4566/health

# List LocalStack services
curl http://localhost:4566/_localstack/health
```

### Coverage Debugging
```bash
# Show uncovered functions
go tool cover -func=coverage.out | grep -v "100.0%"

# Generate coverage profile for specific package
go test ./pkg/aws -coverprofile=aws.out -covermode=count
go tool cover -func=aws.out
```

## Best Practices

1. **Test Structure**: Use table-driven tests for multiple scenarios
2. **Isolation**: Each test should be independent and clean up after itself
3. **Mocking**: Use LocalStack for AWS integration, avoid complex mocking
4. **Coverage**: Focus on critical paths and error conditions
5. **Performance**: Keep unit tests fast (<1s each), integration tests can be slower
6. **Documentation**: Test names should clearly describe what they test

## Future Improvements

1. **Fuzzing**: Add fuzz tests for input validation
2. **Benchmarks**: Add performance benchmarks for critical paths
3. **Property Testing**: Add property-based tests for complex algorithms
4. **Load Testing**: Add load tests for daemon server
5. **End-to-End**: Add full workflow tests with real AWS (optional)

## Troubleshooting

**Common Issues:**
- LocalStack container not starting: Check Docker daemon and port 4566
- Integration tests failing: Ensure LocalStack is fully initialized (wait 10s)
- Coverage reports not generating: Check file permissions and output directory
- AWS SDK errors: Verify LocalStack endpoint configuration

**Debug Commands:**
```bash
# Check LocalStack status
docker ps | grep localstack

# Test LocalStack connectivity
curl -v http://localhost:4566/health

# Validate test build tags
go list -tags=integration ./pkg/aws
```