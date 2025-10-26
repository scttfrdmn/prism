# AWS Validation Findings
**Date**: Mon Oct 13 11:42:21 PDT 2025
**Test Run ID**: real-test-1760380941

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
- ❌ **FAIL**: Instance launch failed or timed out after 2s
  - Details: See ./test-results/aws-validation-20251013_114221/test2_launch.log
Error: launch instance real-test-1760380941-launch failed

API error 500 for POST /api/v1/instances: {"code":"server_error","message":"AWS operation failed: failed to launch instance: operation error EC2: RunInstances, https response error StatusCode: 400, RequestID: c7341e98-f5c6-41eb-a12d-93e0f2871505, api error InvalidParameterValue: The architecture 'x86_64' of the specified instance type does not match the architecture 'arm64' of the specified AMI. Specify an instance type and an AMI that have matching architectures, and try again. You can use 'describe-instance-types' or 'describe-images' to discover the architecture of the instance type or AMI.","status_code":500}


Need help?
1. Check our troubleshooting guide:
   https://github.com/scttfrdmn/prism/blob/main/TROUBLESHOOTING.md

2. Verify daemon status:
   prism daemon status

3. Check AWS credentials:
   aws sts get-caller-identity

4. Open an issue: https://github.com/scttfrdmn/prism/issues
Usage:
  prism launch <template> <name> [flags]

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

Error: instance launch failed

Prism couldn't launch your research environment.

🔧 Common solutions:
1. Try different region: prism launch template-name instance-name --region us-west-2
2. Use different size: prism launch template-name instance-name --size S
3. Check template availability: prism templates

🔍 Advanced troubleshooting:
- Verify AWS quotas: aws service-quotas get-service-quota --service-code ec2 --quota-code L-1216C47A
- Check template validation: prism templates validate
- Try different instance type: prism launch template-name instance-name --instance-type t3.medium

Need template help? Each template shows its requirements with 'cws templates'

Original error: launch instance real-test-1760380941-launch failed

API error 500 for POST /api/v1/instances: {"code":"server_error","message":"AWS operation failed: failed to launch instance: operation error EC2: RunInstances, https response error StatusCode: 400, RequestID: c7341e98-f5c6-41eb-a12d-93e0f2871505, api error InvalidParameterValue: The architecture 'x86_64' of the specified instance type does not match the architecture 'arm64' of the specified AMI. Specify an instance type and an AMI that have matching architectures, and try again. You can use 'describe-instance-types' or 'describe-images' to discover the architecture of the instance type or AMI.","status_code":500}


Need help?
1. Check our troubleshooting guide:
   https://github.com/scttfrdmn/prism/blob/main/TROUBLESHOOTING.md

2. Verify daemon status:
   prism daemon status

3. Check AWS credentials:
   aws sts get-caller-identity

4. Open an issue: https://github.com/scttfrdmn/prism/issues
