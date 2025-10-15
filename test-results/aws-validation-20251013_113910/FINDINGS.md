# AWS Validation Findings
**Date**: Mon Oct 13 11:39:10 PDT 2025
**Test Run ID**: real-test-1760380750

## Summary
- **Total Tests**: TBD
- **Passed**: TBD
- **Failed**: TBD
- **Warnings**: TBD

---

## Test Results

- ✅ **PASS**: Prerequisites check passed
- ✅ **PASS**: Templates list worked on first run
- ✅ **PASS**: Daemon auto-started successfully
- ❌ **FAIL**: Instance launch failed or timed out after 1s
  - Details: See ./test-results/aws-validation-20251013_113910/test2_launch.log
Error: template not found

The specified template doesn't exist. To fix this:

1. List available templates:
   cws templates

2. Check template name spelling
3. Refresh template cache:
   rm -rf ~/.cloudworkstation/templates && cws templates

Original error: API error 500 for POST /api/v1/instances: {"code":"server_error","message":"AWS operation failed: failed to get template: template not found: ubuntu-base","status_code":500}

Usage:
  cws launch <template> <name> [flags]

Flags:
      --dry-run                Validate configuration without launching
  -h, --help                   help for launch
      --hibernation            Enable hibernation support
      --param stringArray      Template parameter in format name=value
      --project string         Associate with project
      --research-user string   Automatically create and provision research user on instance
      --size string            Instance size: XS=1vCPU,2GB+100GB | S=2vCPU,4GB+500GB | M=2vCPU,8GB+1TB | L=4vCPU,16GB+2TB | XL=8vCPU,32GB+4TB
      --spot                   Use spot instances
      --subnet string          Specify subnet ID
      --vpc string             Specify VPC ID
      --wait                   Wait and display launch progress in real-time

Error: template not found

The specified template doesn't exist. To fix this:

1. List available templates:
   cws templates

2. Check template name spelling
3. Refresh template cache:
   rm -rf ~/.cloudworkstation/templates && cws templates

Original error: template not found

The specified template doesn't exist. To fix this:

1. List available templates:
   cws templates

2. Check template name spelling
3. Refresh template cache:
   rm -rf ~/.cloudworkstation/templates && cws templates

Original error: API error 500 for POST /api/v1/instances: {"code":"server_error","message":"AWS operation failed: failed to get template: template not found: ubuntu-base","status_code":500}

