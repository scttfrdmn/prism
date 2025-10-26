# AWS Integration Testing Guide

This document describes Prism's AWS integration testing framework, which allows comprehensive testing against real AWS resources using the 'aws' profile.

## Overview

The AWS integration testing framework provides:
- **Safe Testing**: Automatic resource cleanup and cost limits
- **Real AWS Integration**: Tests against actual AWS services
- **Comprehensive Coverage**: Instance lifecycle, storage, networking, hibernation
- **Cost Control**: Built-in limits to prevent runaway costs
- **Parallel Execution**: Multiple tests can run concurrently

## Quick Start

### Prerequisites

1. **AWS Account**: Test account with appropriate permissions
2. **AWS Profile**: Configure 'aws' profile with test account credentials
3. **Prism Daemon**: Running locally on port 8947
4. **Go Build Tags**: Tests use `aws_integration` build tag

### Basic Usage

```bash
# Mock tests only (default - no AWS resources)
go test ./internal/cli/...

# Include AWS integration tests
RUN_AWS_TESTS=true AWS_PROFILE=aws go test ./internal/cli/...

# AWS tests only  
RUN_AWS_TESTS=true AWS_PROFILE=aws go test ./internal/cli/ -run TestAWS

# Verbose output with test details
RUN_AWS_TESTS=true AWS_PROFILE=aws go test -v ./internal/cli/ -run TestAWS
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `RUN_AWS_TESTS` | `false` | Enable AWS integration tests |
| `AWS_PROFILE` | `aws` | AWS CLI profile for authentication |
| `AWS_TEST_REGION` | `us-east-1` | AWS region for testing |
| `AWS_TEST_TIMEOUT` | `10` | Test timeout in minutes |
| `AWS_TEST_MAX_INSTANCES` | `5` | Maximum concurrent instances |
| `AWS_TEST_MAX_VOLUMES` | `3` | Maximum concurrent volumes |
| `AWS_TEST_MAX_HOURLY_COST` | `5.0` | Maximum hourly spend limit ($) |
| `AWS_TEST_PREFIX` | `cwstest` | Prefix for test resource names |

### Advanced Configuration

```bash
# Custom test configuration
export RUN_AWS_TESTS=true
export AWS_PROFILE=aws
export AWS_TEST_REGION=us-west-2
export AWS_TEST_TIMEOUT=15
export AWS_TEST_MAX_INSTANCES=3
export AWS_TEST_MAX_VOLUMES=2
export AWS_TEST_MAX_HOURLY_COST=3.0
export AWS_TEST_PREFIX=mytest

go test -v ./internal/cli/ -run TestAWS
```

## Test Coverage

### Instance Lifecycle Tests (`TestAWSInstanceLifecycle`)
- ✅ Instance launch with template application
- ✅ State transitions (pending → running → stopped → running)
- ✅ Hibernation and resume (if supported)
- ✅ Connection information generation
- ✅ Proper cleanup and termination

### Template Operations (`TestAWSTemplateOperations`)
- ✅ Template discovery and listing
- ✅ Template validation
- ✅ Template inheritance verification
- ✅ AMI lookup and compatibility

### Storage Management (`TestAWSStorageOperations`)
- ✅ EFS volume creation and lifecycle
- ✅ EBS storage creation with GP3 optimization
- ✅ Storage attachment and mounting
- ✅ Volume state monitoring

### Network Operations (`TestAWSNetworkOperations`)
- ✅ VPC and subnet discovery
- ✅ Security group management
- ✅ Public IP assignment
- ✅ SSH connectivity setup

### Hibernation Features (`TestAWSHibernationFeatures`)
- ✅ Hibernation support detection
- ✅ Instance hibernation and resume
- ✅ State preservation verification
- ✅ Fallback to stop/start when hibernation unsupported

### Project Management (`TestAWSProjectManagement`)
- ✅ Project-based resource organization
- ✅ Budget tracking and cost analysis
- ✅ Multi-user collaboration features
- ✅ Resource tagging and filtering

### Idle Detection (`TestAWSIdleDetection`)
- ✅ Idle profile management
- ✅ Automated hibernation policies
- ✅ Usage monitoring and thresholds
- ✅ Cost optimization tracking

### Daemon Integration (`TestAWSDaemonIntegration`)
- ✅ Full CLI-to-daemon-to-AWS workflow
- ✅ Profile-based authentication
- ✅ Real-time status monitoring
- ✅ Error handling and recovery

## Safety Features

### Automatic Resource Cleanup
- All test resources tagged with `CreatedBy=PrismIntegrationTest`
- Unique naming: `cwstest-[testid]-[resource]-[timestamp]`
- Automatic cleanup in test teardown (even on failure)
- Orphaned resource detection and cleanup

### Cost Protection
- Instance limits: Maximum 5 concurrent instances
- Volume limits: Maximum 3 concurrent volumes
- Hourly cost limit: $5.00 maximum spend
- Smallest instance types used (t3.nano, t3.micro)
- Conservative storage sizes (100GB default)

### Resource Isolation
- Test-specific resource naming prevents conflicts
- Proper AWS resource tagging for identification
- Region isolation for parallel test execution
- Time-based cleanup for orphaned resources

### Error Handling
- Retry logic for transient AWS failures
- Graceful degradation when services unavailable
- Comprehensive error logging and debugging
- Resource cleanup on test interruption

## AWS Setup Requirements

### IAM Permissions

Test account needs these minimum permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:*",
                "efs:*",
                "iam:PassRole",
                "sts:GetCallerIdentity"
            ],
            "Resource": "*",
            "Condition": {
                "StringEquals": {
                    "aws:RequestedRegion": ["us-east-1", "us-west-2"]
                }
            }
        }
    ]
}
```

### AWS Profile Configuration

```bash
# Configure AWS profile for testing
aws configure --profile aws
# AWS Access Key ID: [test-account-key]
# AWS Secret Access Key: [test-account-secret] 
# Default region name: us-east-1
# Default output format: json

# Verify profile setup
aws sts get-caller-identity --profile aws
```

### Account Limits

Ensure test account has sufficient limits:
- EC2 instances: At least 10 instances
- EBS volumes: At least 10 volumes
- EFS file systems: At least 5 file systems
- VPC resources: Default limits sufficient

## Troubleshooting

### Common Issues

**Tests Skip with "AWS tests disabled"**
```bash
# Ensure RUN_AWS_TESTS is set
export RUN_AWS_TESTS=true
```

**"Failed to load AWS config"**
```bash
# Check AWS profile exists and has valid credentials
aws configure list --profile aws
aws sts get-caller-identity --profile aws
```

**"Prism daemon not running"**
```bash
# Start the daemon
./bin/cwsd &

# Or check if running on different port
export DAEMON_URL=http://localhost:8947
```

**"Cost limit exceeded"**
```bash
# Increase cost limit or reduce resource usage
export AWS_TEST_MAX_HOURLY_COST=10.0
export AWS_TEST_MAX_INSTANCES=3
```

**"Instance failed to reach running state"**
- Check AWS account limits and quotas
- Verify selected region has capacity
- Review CloudWatch logs for launch failures

### Debug Mode

```bash
# Enable verbose test output
RUN_AWS_TESTS=true AWS_PROFILE=aws go test -v ./internal/cli/ -run TestAWS

# Enable AWS SDK debug logging
export AWS_SDK_LOAD_CONFIG=1
export AWS_LOG_LEVEL=debug
```

### Cleanup Verification

```bash
# Check for orphaned resources
aws ec2 describe-instances --filters "Name=tag:CreatedBy,Values=PrismIntegrationTest" --profile aws
aws efs describe-file-systems --profile aws | grep cwstest
```

## Performance Considerations

### Test Execution Times
- Instance lifecycle: 8-12 minutes
- Storage operations: 2-3 minutes  
- Template validation: 30-60 seconds
- Network setup: 1-2 minutes

### Optimization Tips
- Run tests in parallel where possible
- Use fastest AWS regions (us-east-1, us-west-2)
- Pre-warm AMIs in test account
- Cache template validations

### CI/CD Integration
```yaml
# GitHub Actions example
- name: AWS Integration Tests
  env:
    RUN_AWS_TESTS: true
    AWS_PROFILE: aws
    AWS_TEST_TIMEOUT: 15
  run: go test -v ./internal/cli/ -run TestAWS
```

## Contributing

### Adding New Tests
1. Use `//go:build aws_integration` build tag
2. Implement proper resource cleanup
3. Add cost tracking for new resources
4. Include comprehensive error handling
5. Add documentation for new test cases

### Best Practices
- Keep tests idempotent and isolated
- Use helper functions for common operations
- Implement proper timeouts and retries
- Add comprehensive logging for debugging
- Follow existing naming conventions

## Security Considerations

- Test account should be isolated from production
- Use minimal IAM permissions required
- Rotate test account credentials regularly  
- Monitor test account usage and costs
- Implement resource cleanup automation

## Cost Management

### Monitoring
```bash
# Check current test costs
aws ce get-cost-and-usage \
  --time-period Start=2024-01-01,End=2024-01-02 \
  --granularity DAILY \
  --metrics BlendedCost \
  --group-by Type=DIMENSION,Key=SERVICE \
  --profile aws
```

### Budget Alerts
Set up AWS Budget alerts for test account to monitor integration test costs and prevent overruns.

## Support

For issues with AWS integration testing:
1. Check this documentation
2. Review test logs and error messages
3. Verify AWS account setup and permissions
4. Check Prism daemon status
5. Open GitHub issue with full error details