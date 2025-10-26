# Template File Provisioning Guide (v0.5.7)

Complete guide to provisioning files in Prism templates using S3-backed transfers.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [File Configuration](#file-configuration)
- [Template Examples](#template-examples)
- [S3 Setup](#s3-setup)
- [IAM Permissions](#iam-permissions)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

Prism v0.5.7 introduces S3-backed file provisioning for templates, enabling:

- **Large Dataset Distribution**: Multi-GB datasets automatically downloaded during launch
- **Binary Distribution**: Commercial software binaries and pre-compiled tools
- **Configuration Deployment**: Standardized configuration files across instances
- **Pre-trained Models**: ML model weights and checkpoints
- **Conditional Provisioning**: Architecture-specific or optional files

### Key Features

- ✅ **Multipart transfers** for files up to 5TB
- ✅ **Progress tracking** with real-time updates
- ✅ **MD5 checksum verification** for data integrity
- ✅ **Conditional provisioning** based on architecture
- ✅ **Required vs optional** files with graceful fallback
- ✅ **Auto-cleanup** from S3 after download
- ✅ **File permissions** and ownership configuration

## Quick Start

### 1. Upload Files to S3

```bash
# Create S3 bucket for datasets
aws s3 mb s3://my-prism-datasets --region us-west-2

# Upload dataset
aws s3 cp imagenet_subset.tar.gz s3://my-prism-datasets/datasets/imagenet.tar.gz

# Verify upload
aws s3 ls s3://my-prism-datasets/datasets/
```

### 2. Add Files to Template

```yaml
name: "Python ML with ImageNet"
slug: python-ml-imagenet
base: ubuntu-22.04
package_manager: conda

packages:
  conda:
    - python=3.11
    - pytorch
    - torchvision

# File provisioning
files:
  - s3_bucket: my-prism-datasets
    s3_key: datasets/imagenet.tar.gz
    destination_path: /home/ubuntu/datasets/imagenet.tar.gz
    description: "ImageNet validation subset (10GB)"
    owner: ubuntu
    group: ubuntu
    permissions: "0644"
    checksum: true
    required: true
```

### 3. Launch Instance

```bash
prism launch python-ml-imagenet my-ml-work
```

The files will be automatically downloaded during instance launch!

## File Configuration

### Required Fields

```yaml
files:
  - s3_bucket: bucket-name       # S3 bucket containing the file
    s3_key: path/to/file.tar.gz  # S3 object key (path within bucket)
    destination_path: /path/on/instance  # Where to place file on instance
```

### Optional Fields

```yaml
files:
  - # ... required fields ...

    # File Properties
    owner: ubuntu              # File owner (default: ubuntu)
    group: ubuntu              # File group (default: owner)
    permissions: "0644"        # Octal permissions (e.g., 0644, 0755)

    # Transfer Options
    checksum: true             # Enable MD5 verification (default: true)
    auto_cleanup: false        # Delete from S3 after download (default: false)

    # Conditional Provisioning
    required: true             # Fail launch if download fails (default: true)
    only_if: "arch == 'x86_64'"  # Conditional expression

    # Documentation
    description: "Human-readable description"
```

## Template Examples

### Example 1: Large Dataset

```yaml
files:
  - s3_bucket: research-datasets
    s3_key: imagenet/val_10gb.tar.gz
    destination_path: /home/ubuntu/datasets/imagenet.tar.gz
    description: "ImageNet validation set (10GB)"
    owner: ubuntu
    group: ubuntu
    permissions: "0644"
    checksum: true
    required: true  # Critical dataset - fail if unavailable

post_install: |
  #!/bin/bash
  # Extract dataset after download
  cd /home/ubuntu/datasets
  tar -xzf imagenet.tar.gz
  rm imagenet.tar.gz
```

### Example 2: Architecture-Specific Binaries

```yaml
files:
  # x86_64 binary
  - s3_bucket: prism-binaries
    s3_key: tools/analyzer_x86_64
    destination_path: /usr/local/bin/analyzer
    description: "Data analyzer (x86_64)"
    owner: root
    group: root
    permissions: "0755"
    checksum: true
    only_if: "arch == 'x86_64'"

  # ARM64 binary
  - s3_bucket: prism-binaries
    s3_key: tools/analyzer_arm64
    destination_path: /usr/local/bin/analyzer
    description: "Data analyzer (ARM64)"
    owner: root
    group: root
    permissions: "0755"
    checksum: true
    only_if: "arch == 'arm64'"
```

### Example 3: Optional Configuration Files

```yaml
files:
  # Required configuration
  - s3_bucket: my-configs
    s3_key: jupyter/jupyter_notebook_config.py
    destination_path: /home/ubuntu/.jupyter/jupyter_notebook_config.py
    description: "Jupyter configuration"
    owner: ubuntu
    permissions: "0644"
    required: true

  # Optional pre-loaded notebooks
  - s3_bucket: my-configs
    s3_key: notebooks/tutorial.ipynb
    destination_path: /home/ubuntu/notebooks/tutorial.ipynb
    description: "Tutorial notebook"
    owner: ubuntu
    permissions: "0644"
    required: false  # Continue if missing
```

### Example 4: Pre-trained Model Weights

```yaml
files:
  - s3_bucket: ml-models
    s3_key: weights/resnet50_imagenet.pth
    destination_path: /home/ubuntu/models/resnet50.pth
    description: "Pre-trained ResNet50 weights (500MB)"
    owner: ubuntu
    permissions: "0644"
    checksum: true
    auto_cleanup: true  # Remove from S3 after download
```

## S3 Setup

### Bucket Organization

Recommended S3 bucket structure:

```
my-prism-datasets/
├── datasets/
│   ├── imagenet/
│   │   └── val_10gb.tar.gz
│   ├── coco/
│   │   └── val_2017.zip
│   └── custom/
│       └── research_data.h5
├── models/
│   ├── resnet50.pth
│   └── bert_base.bin
├── configs/
│   └── jupyter_config.py
└── binaries/
    ├── tool_x86_64
    └── tool_arm64
```

### Bucket Policy

Grant read access to EC2 instances:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": "*",
      "Action": [
        "s3:GetObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::my-prism-datasets/*",
        "arn:aws:s3:::my-prism-datasets"
      ],
      "Condition": {
        "StringEquals": {
          "aws:PrincipalAccount": "YOUR_AWS_ACCOUNT_ID"
        }
      }
    }
  ]
}
```

## IAM Permissions

### Instance Profile

Prism instances need S3 read permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::my-prism-datasets/*",
        "arn:aws:s3:::my-prism-datasets"
      ]
    }
  ]
}
```

### Optional: Auto-Cleanup Permission

If using `auto_cleanup: true`:

```json
{
  "Effect": "Allow",
  "Action": "s3:DeleteObject",
  "Resource": "arn:aws:s3:::my-prism-datasets/*"
}
```

## Best Practices

### File Sizes

- **Small (<100MB)**: Perfect for configs, scripts, small datasets
- **Medium (100MB-1GB)**: Good for pre-trained models, medium datasets
- **Large (1GB-10GB)**: Suitable for research datasets, large models
- **Very Large (>10GB)**: Use with `estimated_launch_time` adjustment

### Storage Costs

Consider S3 storage costs vs. repeated downloads:

```yaml
files:
  - s3_bucket: my-datasets
    s3_key: frequently_used_dataset.tar.gz
    # ...
    auto_cleanup: false  # Keep in S3 for multiple launches
```

For one-time use datasets:

```yaml
files:
  - s3_bucket: my-datasets
    s3_key: ephemeral_dataset.tar.gz
    # ...
    auto_cleanup: true  # Remove after download
```

### Error Handling

```yaml
files:
  # Critical file - halt if missing
  - s3_bucket: my-datasets
    s3_key: required_dataset.tar.gz
    required: true  # EXIT 1 on failure

  # Optional file - continue if missing
  - s3_bucket: my-datasets
    s3_key: optional_dataset.tar.gz
    required: false  # WARNING on failure, continue
```

### Launch Time Estimates

Update `estimated_launch_time` for file provisioning:

```yaml
files:
  - # 10GB file @ ~100MB/s = ~2 minutes
  - # 1GB file @ ~100MB/s = ~10 seconds
  - # Total: ~3 minutes for files

estimated_launch_time: 15  # 10 min baseline + 5 min files
```

## Troubleshooting

### Files Not Downloading

**Check cloud-init logs**:
```bash
prism connect my-instance
sudo tail -f /var/log/cloud-init-output.log
```

Look for:
- `Starting file provisioning from S3...`
- `✓ Downloaded: <description>`
- `✗ Failed to download: <description>`

### Permission Denied

Verify IAM instance profile:
```bash
aws sts get-caller-identity
aws s3 ls s3://my-bucket/
```

### Checksum Failures

AWS CLI automatically verifies checksums. If failures occur:

1. Re-upload file to S3
2. Verify file integrity locally before upload
3. Check for S3 cross-region replication issues

### Slow Downloads

- Use same region for S3 bucket and instance
- Consider S3 Transfer Acceleration for global access
- Check instance network performance (enhanced networking)

### Conditional Not Working

Architecture detection:
```bash
# On instance
uname -m
# x86_64 -> x86_64
# aarch64 -> arm64
```

Update condition:
```yaml
only_if: "arch == 'x86_64'"  # For Intel/AMD
only_if: "arch == 'arm64'"   # For ARM
```

## Advanced Features

### Post-Download Processing

```yaml
files:
  - s3_bucket: my-datasets
    s3_key: compressed_data.tar.gz
    destination_path: /tmp/data.tar.gz
    # ... config ...

post_install: |
  #!/bin/bash
  # Extract after download
  cd /home/ubuntu/datasets
  tar -xzf /tmp/data.tar.gz
  rm /tmp/data.tar.gz

  # Process data
  python preprocess.py

  # Set permissions
  chown -R ubuntu:ubuntu /home/ubuntu/datasets
```

### Multiple Files with Dependencies

```yaml
files:
  # Download in order specified
  - s3_bucket: my-data
    s3_key: base_dataset.tar.gz
    destination_path: /data/base.tar.gz
    required: true

  - s3_bucket: my-data
    s3_key: supplement_dataset.tar.gz
    destination_path: /data/supplement.tar.gz
    required: false  # OK if missing
```

### Size Optimization

Use compression:
```bash
# Before upload
tar -czf dataset.tar.gz dataset/
# 10GB -> 3GB (3x smaller, faster transfer)
```

## API Reference

### FileConfig Struct

```go
type FileConfig struct {
    S3Bucket        string  // Required: S3 bucket name
    S3Key           string  // Required: S3 object key
    DestinationPath string  // Required: Target path on instance
    Owner           string  // Optional: File owner
    Group           string  // Optional: File group
    Permissions     string  // Optional: Octal permissions
    Checksum        bool    // Optional: MD5 verification
    AutoCleanup     bool    // Optional: Remove from S3
    Required        bool    // Optional: Fail on error
    OnlyIf          string  // Optional: Conditional expression
    Description     string  // Optional: Human description
}
```

### Validation

```go
import "github.com/scttfrdmn/prism/pkg/templates"

// Validate single file
if err := templates.ValidateFileConfig(fileConfig); err != nil {
    log.Fatal(err)
}

// Validate all files in template
if err := templates.ValidateTemplateFiles(template); err != nil {
    log.Fatal(err)
}
```

### Script Generation

```go
import "github.com/scttfrdmn/prism/pkg/templates"

// Generate provisioning script
script := templates.GenerateFileProvisioningScript(
    template.Files,
    "us-west-2",  // Instance region
)

// Append to user data
userData += script
```

## See Also

- [S3 Transfer API Documentation](S3_TRANSFER_API.md)
- [Template Schema Reference](TEMPLATE_SCHEMA.md)
- [IAM Best Practices](https://docs.aws.amazon.com/IAM/latest/UserGuide/best-practices.html)
- [S3 Pricing](https://aws.amazon.com/s3/pricing/)

## Support

For issues or questions:
- GitHub Issues: https://github.com/scttfrdmn/prism/issues
- Related: Issue #64 (Template Asset Management)
- Related: Issue #92 (v0.5.7 Planning)
