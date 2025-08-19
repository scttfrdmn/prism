# CloudWorkstation AMI User Guide

## Quick Start - Building AMIs Made Simple

CloudWorkstation automatically discovers your AWS VPC and subnet configuration, so you can build AMIs with minimal setup:

```bash
# Simple AMI build - auto-discovers VPC and subnet
cws ami build my-template

# Override auto-discovery if needed
cws ami build my-template --vpc vpc-12345 --subnet subnet-67890
```

## Commands

### Build AMI from Template
```bash
# Auto-discovery (recommended)
cws ami build python-ml

# With specific region
cws ami build python-ml --region us-west-2

# For ARM architecture
cws ami build python-ml --arch arm64

# Override VPC/subnet auto-discovery
cws ami build python-ml --vpc vpc-12345 --subnet subnet-67890

# Dry run (test without creating resources)
cws ami build python-ml --dry-run
```

### Save Running Instance as Template
```bash
# Convert your customized instance to a reusable template
cws ami save my-instance my-custom-template --description "My research environment"

# Copy to multiple regions
cws ami save my-instance my-template --copy-to-regions us-west-1,eu-west-1
```

### List and Manage AMIs
```bash
# List all available templates
cws ami list

# List AMIs for specific template
cws ami list python-ml

# Validate template before building
cws ami validate my-template
```

## AWS Permissions Required

Your AWS user/role needs these permissions for AMI operations:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeVpcs",
                "ec2:DescribeSubnets", 
                "ec2:DescribeRouteTables",
                "ec2:DescribeImages",
                "ec2:DescribeInstances",
                "ec2:DescribeInstanceAttribute",
                "ec2:DescribeSecurityGroups",
                "ec2:RunInstances",
                "ec2:StopInstances",
                "ec2:StartInstances", 
                "ec2:TerminateInstances",
                "ec2:CreateImage",
                "ec2:CopyImage",
                "ec2:CreateTags",
                "ssm:SendCommand",
                "ssm:GetCommandInvocation",
                "ssm:DescribeInstanceInformation",
                "ssm:GetParameter",
                "ssm:PutParameter",
                "ssm:GetParametersByPath"
            ],
            "Resource": "*"
        }
    ]
}
```

## How Auto-Discovery Works

CloudWorkstation automatically finds your AWS infrastructure:

1. **VPC Discovery**: Finds your default VPC in the current region
2. **Subnet Discovery**: Locates a public subnet with internet gateway access
3. **Fallback**: Uses first available subnet if public detection fails

You'll see output like:
```
🔍 Auto-discovering default VPC and subnet...
   ✅ Using default VPC: vpc-12345678
   ✅ Using public subnet: subnet-87654321
```

## Troubleshooting

### No Default VPC Found
```
Error: no default VPC found - please create one or specify --vpc
```
**Solution**: Create a default VPC or specify `--vpc vpc-your-id`

### No Subnets Available  
```
Error: no subnets found in VPC vpc-12345
```
**Solution**: Create subnets in your VPC or use a different VPC

### AMI Build Timeout
```
Error: timeout waiting for instance to be ready for SSM commands
```
**Solution**: Check security group allows outbound internet access for SSM

## Best Practices

1. **Use Auto-Discovery**: Let CloudWorkstation find your VPC/subnet automatically
2. **Test Templates**: Use `cws ami validate` before building
3. **Dry Run First**: Use `--dry-run` to test without creating resources
4. **Multi-Region**: Copy AMIs to regions where you'll launch instances
5. **Descriptive Names**: Use clear template names for easy identification

## Example Workflows

### Research Workflow
```bash
# Validate template
cws ami validate python-ml

# Build AMI
cws ami build python-ml --region us-east-1

# Launch instance from AMI (fast!)
cws launch python-ml my-research
```

### Custom Environment Workflow  
```bash
# Launch base instance
cws launch python-ml my-custom

# Customize the instance (install packages, configure tools)
# ... work in your instance ...

# Save customizations as new template
cws ami save my-custom my-custom-ml-env --description "My custom ML environment"

# Share with team
cws launch my-custom-ml-env team-project
```