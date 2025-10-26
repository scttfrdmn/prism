# Prism AWS IAM Permissions

**Last Updated**: October 17, 2025

This document defines the minimum AWS IAM permissions required for Prism to function properly.

---

## Overview

Prism requires AWS credentials with sufficient permissions to manage EC2 instances, EFS filesystems, IAM roles, and Systems Manager (SSM) operations. Users must have an AWS account with either:

1. **AWS Access Keys** (Access Key ID + Secret Access Key) stored in `~/.aws/credentials`
2. **AWS IAM Role** attached to the machine running Prism (for EC2/ECS deployments)
3. **AWS SSO credentials** configured via `aws sso login`

---

## Quick Start: Recommended Setup

For new users, Prism provides a **managed IAM policy** that grants all necessary permissions:

```bash
# Option 1: Attach AWS managed policy (if available in future)
aws iam attach-user-policy \
  --user-name YOUR_USERNAME \
  --policy-arn arn:aws:iam::aws:policy/PrismFullAccess

# Option 2: Create custom policy from this document
aws iam create-policy \
  --policy-name PrismAccess \
  --policy-document file://prism-policy.json

aws iam attach-user-policy \
  --user-name YOUR_USERNAME \
  --policy-arn arn:aws:iam::YOUR_ACCOUNT_ID:policy/PrismAccess
```

---

## Minimum Required Permissions

Prism requires the following AWS service permissions:

### 1. **EC2 (Elastic Compute Cloud)** - Core Instance Management

**Required Actions:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "EC2InstanceManagement",
      "Effect": "Allow",
      "Action": [
        "ec2:RunInstances",
        "ec2:TerminateInstances",
        "ec2:StartInstances",
        "ec2:StopInstances",
        "ec2:DescribeInstances",
        "ec2:DescribeInstanceTypes",
        "ec2:DescribeInstanceTypeOfferings",
        "ec2:DescribeImages",
        "ec2:DescribeVolumes",
        "ec2:CreateVolume",
        "ec2:DeleteVolume",
        "ec2:CreateTags",
        "ec2:DescribeTags"
      ],
      "Resource": "*"
    },
    {
      "Sid": "EC2NetworkManagement",
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeVpcs",
        "ec2:DescribeSubnets",
        "ec2:DescribeSecurityGroups",
        "ec2:CreateSecurityGroup",
        "ec2:AuthorizeSecurityGroupIngress"
      ],
      "Resource": "*"
    },
    {
      "Sid": "EC2KeyPairManagement",
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeKeyPairs",
        "ec2:ImportKeyPair",
        "ec2:DeleteKeyPair"
      ],
      "Resource": "*"
    }
  ]
}
```

**Why These Permissions:**
- **RunInstances**: Launch new Prism instances
- **Stop/Start/Terminate**: Instance lifecycle management
- **DescribeInstances**: List and monitor running instances
- **DescribeImages**: Find optimal AMIs for templates
- **CreateSecurityGroup**: Automatic security group creation for SSH/web access
- **ImportKeyPair**: Manage SSH keys for instance access

### 2. **EFS (Elastic File System)** - Persistent Storage

**Required Actions:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "EFSVolumeManagement",
      "Effect": "Allow",
      "Action": [
        "elasticfilesystem:CreateFileSystem",
        "elasticfilesystem:DeleteFileSystem",
        "elasticfilesystem:DescribeFileSystems",
        "elasticfilesystem:DescribeMountTargets",
        "elasticfilesystem:CreateMountTarget",
        "elasticfilesystem:DeleteMountTarget",
        "elasticfilesystem:CreateTags",
        "elasticfilesystem:DescribeTags"
      ],
      "Resource": "*"
    }
  ]
}
```

**Why These Permissions:**
- **CreateFileSystem**: Create shared EFS volumes for research data
- **CreateMountTarget**: Attach EFS to instances across availability zones
- **DescribeMountTargets**: Monitor volume attachments
- Multi-instance file sharing for collaborative research

### 3. **IAM (Identity and Access Management)** - Instance Profiles

**Required Actions:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "IAMInstanceProfileManagement",
      "Effect": "Allow",
      "Action": [
        "iam:GetInstanceProfile",
        "iam:CreateRole",
        "iam:AttachRolePolicy",
        "iam:PutRolePolicy",
        "iam:CreateInstanceProfile",
        "iam:AddRoleToInstanceProfile",
        "iam:PassRole"
      ],
      "Resource": [
        "arn:aws:iam::*:role/Prism-Instance-Profile-Role",
        "arn:aws:iam::*:instance-profile/Prism-Instance-Profile"
      ]
    }
  ]
}
```

**Why These Permissions:**
- **CreateRole**: Auto-create Prism-Instance-Profile for SSM access
- **AttachRolePolicy**: Attach AmazonSSMManagedInstanceCore for Systems Manager
- **PutRolePolicy**: Create inline policy for autonomous idle detection
- **PassRole**: Allow EC2 to assume the Prism role

**What This Enables:**
- **SSM Access**: Remote command execution without SSH keys
- **Autonomous Idle Detection**: Instances can stop themselves when idle
- **Secure Management**: No SSH keys exposed, all commands via AWS Systems Manager

### 4. **SSM (Systems Manager)** - Remote Command Execution

**Required Actions:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "SSMCommandExecution",
      "Effect": "Allow",
      "Action": [
        "ssm:SendCommand",
        "ssm:GetCommandInvocation",
        "ssm:DescribeInstanceInformation"
      ],
      "Resource": "*"
    }
  ]
}
```

**Why These Permissions:**
- **SendCommand**: Execute remote scripts for software installation, user provisioning
- **GetCommandInvocation**: Monitor command execution status
- Used for EFS mounting, template provisioning, research user setup

### 5. **STS (Security Token Service)** - Identity Verification

**Required Actions:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "STSIdentityVerification",
      "Effect": "Allow",
      "Action": [
        "sts:GetCallerIdentity"
      ],
      "Resource": "*"
    }
  ]
}
```

**Why These Permissions:**
- Verify AWS credentials are valid
- Retrieve AWS account ID for resource naming

---

## Complete IAM Policy

Here is the **complete Prism IAM policy** combining all permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "EC2InstanceManagement",
      "Effect": "Allow",
      "Action": [
        "ec2:RunInstances",
        "ec2:TerminateInstances",
        "ec2:StartInstances",
        "ec2:StopInstances",
        "ec2:DescribeInstances",
        "ec2:DescribeInstanceStatus",
        "ec2:DescribeInstanceTypes",
        "ec2:DescribeInstanceTypeOfferings",
        "ec2:DescribeImages",
        "ec2:DescribeVolumes",
        "ec2:CreateVolume",
        "ec2:DeleteVolume",
        "ec2:AttachVolume",
        "ec2:DetachVolume",
        "ec2:CreateTags",
        "ec2:DescribeTags"
      ],
      "Resource": "*"
    },
    {
      "Sid": "EC2NetworkManagement",
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeVpcs",
        "ec2:DescribeSubnets",
        "ec2:DescribeAvailabilityZones",
        "ec2:DescribeSecurityGroups",
        "ec2:CreateSecurityGroup",
        "ec2:AuthorizeSecurityGroupIngress",
        "ec2:RevokeSecurityGroupIngress"
      ],
      "Resource": "*"
    },
    {
      "Sid": "EC2KeyPairManagement",
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeKeyPairs",
        "ec2:ImportKeyPair",
        "ec2:DeleteKeyPair"
      ],
      "Resource": "*"
    },
    {
      "Sid": "EFSVolumeManagement",
      "Effect": "Allow",
      "Action": [
        "elasticfilesystem:CreateFileSystem",
        "elasticfilesystem:DeleteFileSystem",
        "elasticfilesystem:DescribeFileSystems",
        "elasticfilesystem:DescribeMountTargets",
        "elasticfilesystem:CreateMountTarget",
        "elasticfilesystem:DeleteMountTarget",
        "elasticfilesystem:CreateTags",
        "elasticfilesystem:DescribeTags"
      ],
      "Resource": "*"
    },
    {
      "Sid": "IAMInstanceProfileManagement",
      "Effect": "Allow",
      "Action": [
        "iam:GetRole",
        "iam:GetInstanceProfile",
        "iam:CreateRole",
        "iam:TagRole",
        "iam:AttachRolePolicy",
        "iam:PutRolePolicy",
        "iam:CreateInstanceProfile",
        "iam:TagInstanceProfile",
        "iam:AddRoleToInstanceProfile",
        "iam:PassRole"
      ],
      "Resource": [
        "arn:aws:iam::*:role/Prism-Instance-Profile-Role",
        "arn:aws:iam::*:instance-profile/Prism-Instance-Profile"
      ]
    },
    {
      "Sid": "SSMCommandExecution",
      "Effect": "Allow",
      "Action": [
        "ssm:SendCommand",
        "ssm:GetCommandInvocation",
        "ssm:ListCommands",
        "ssm:ListCommandInvocations",
        "ssm:DescribeInstanceInformation"
      ],
      "Resource": "*"
    },
    {
      "Sid": "STSIdentityVerification",
      "Effect": "Allow",
      "Action": [
        "sts:GetCallerIdentity"
      ],
      "Resource": "*"
    }
  ]
}
```

---

## Permission Tiers

Prism supports different permission levels based on institutional requirements:

### **Tier 1: Basic Usage** (Minimum Required)
- EC2 instance management (launch, stop, terminate)
- EC2 networking (VPC, security groups, subnets)
- SSH key pair management
- STS identity verification

**Missing Features Without Tier 2:**
- No persistent EFS storage (only local instance storage)
- No SSM access (must use SSH keys)
- No autonomous idle detection

### **Tier 2: Full Features** (Recommended)
- All Tier 1 permissions
- EFS filesystem creation and management
- IAM instance profile auto-creation
- SSM remote command execution

**Enables:**
- Multi-instance shared storage
- Zero-configuration SSM access
- Autonomous cost optimization (idle detection)

### **Tier 3: Institutional Deployment** (Future)
- All Tier 2 permissions
- Additional permissions for:
  - CloudFormation stack creation (one-click institutional setup)
  - AWS Cost Explorer API access (detailed cost analytics)
  - AWS Organizations integration (multi-account management)

---

## Security Best Practices

### 1. **Use IAM Users, Not Root Credentials**
Never use AWS root account credentials with Prism. Create a dedicated IAM user:

```bash
aws iam create-user --user-name prism-admin
aws iam create-access-key --user-name prism-admin
aws iam attach-user-policy \
  --user-name prism-admin \
  --policy-arn arn:aws:iam::YOUR_ACCOUNT_ID:policy/PrismAccess
```

### 2. **Restrict Permissions with Resource Tags**
Limit Prism to only manage its own resources:

```json
{
  "Condition": {
    "StringEquals": {
      "aws:ResourceTag/ManagedBy": "Prism"
    }
  }
}
```

### 3. **Use AWS Profiles for Multi-Account Management**
Separate research projects into different AWS accounts:

```bash
# ~/.aws/credentials
[research-project-1]
aws_access_key_id = AKIA...
aws_secret_access_key = ...

[research-project-2]
aws_access_key_id = AKIA...
aws_secret_access_key = ...

# Prism profiles
prism profiles add project1 proj1-profile --aws-profile research-project-1 --region us-west-2
prism profiles add project2 proj2-profile --aws-profile research-project-2 --region us-east-1
```

### 4. **Enable CloudTrail Logging**
Track all AWS API calls made by Prism:

```bash
aws cloudtrail create-trail \
  --name prism-audit \
  --s3-bucket-name YOUR_AUDIT_BUCKET
```

---

## Common Permission Issues

### **Error: "You are not authorized to perform this operation"**

**Cause**: Missing required IAM permissions

**Solution**: Attach the Prism IAM policy to your IAM user/role

### **Error: "User is not authorized to perform: iam:CreateRole"**

**Cause**: User lacks IAM permissions for instance profile auto-creation

**Impact**: SSM access and autonomous idle detection will be unavailable

**Solutions**:
1. **Recommended**: Request IAM permissions from AWS administrator
2. **Workaround**: Manually create Prism-Instance-Profile in AWS console
3. **Fallback**: Continue without IAM profile (SSH-only access, no autonomous features)

### **Error: "Failed to create EFS filesystem: AccessDeniedException"**

**Cause**: Missing EFS permissions

**Impact**: Cannot create persistent shared storage volumes

**Solution**: Add EFS permissions to IAM policy

---

## Verification

Test your IAM permissions with Prism:

```bash
# Test EC2 permissions
prism templates  # Should list available templates
prism launch ubuntu test-instance --dry-run  # Should show what would be created

# Test EFS permissions (if you have them)
prism volume create test-volume  # Should create EFS filesystem

# Test IAM permissions (if you have them)
# Prism will automatically create instance profile on first launch
prism launch ubuntu test-instance
# Check logs for: "✅ Successfully created IAM instance profile 'Prism-Instance-Profile'"
```

---

## Getting Help

If you encounter permission issues:

1. **Check AWS IAM Policy Simulator**: https://policysim.aws.amazon.com/
2. **Review CloudTrail logs**: See which API calls are being denied
3. **Contact AWS Support**: For enterprise/educational account assistance
4. **Prism Issues**: https://github.com/anthropics/prism/issues

---

## Future Enhancements

### Planned Improvements
- **AWS CloudFormation Template**: One-click IAM setup for institutions
- **Least-Privilege Policies**: More restrictive resource-level permissions
- **AWS Organizations Integration**: Multi-account research management
- **Cost Explorer Integration**: Detailed cost analytics and budget tracking

---

**Summary**: Prism requires EC2, EFS, IAM, SSM, and STS permissions for full functionality. Basic usage requires only EC2 permissions, but EFS and IAM permissions enable persistent storage and autonomous features. Users should create a dedicated IAM user with the Prism policy for secure access.
