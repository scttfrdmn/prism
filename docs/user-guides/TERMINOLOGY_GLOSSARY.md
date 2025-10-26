# Prism Terminology Glossary

## Overview

Prism uses researcher-friendly terminology while leveraging AWS infrastructure. This glossary helps users familiar with AWS understand how Prism terms map to AWS concepts.

## Design Philosophy

**For Researchers**: Prism prioritizes clarity and accessibility over technical precision.
**For AWS Experts**: Use the `--verbose` flag on CLI commands to see AWS technical details.

---

## Core Terminology Mapping

### Workspaces (User Term) → EC2 Instances (AWS Term)

**Prism Term**: **Workspace**
**AWS Equivalent**: EC2 Instance
**Why the Change**: "Workspace" conveys the researcher's mental model - a complete computing environment for their work.

**Examples**:
- Prism: "Launch a workspace"
- AWS: "Launch an EC2 instance"
- Prism: "List my workspaces"
- AWS: "List my EC2 instances"

**When to Use Which**:
- ✅ **Workspace** - All user-facing Prism interfaces (CLI, TUI, GUI)
- ✅ **Instance** - When discussing AWS infrastructure directly (e.g., "EC2 instance types", "spot instances")

---

## Storage Terminology

### Current Terms (v0.5.6)

| Prism Term | AWS Equivalent | Description |
|----------------------|----------------|-------------|
| **EBS Volume** | EBS Volume | Block storage attached to workspaces |
| **EFS Filesystem** | EFS Filesystem | Shared network filesystem |
| **S3 Bucket** | S3 Bucket | Object storage for datasets |

### Planned Simplification (v0.5.7 - Issue #66)

| New Prism Term | AWS Equivalent | Description |
|--------------------------|----------------|-------------|
| **Local Storage** | EBS Volume | Workspace-local persistent storage |
| **Shared Storage** | EFS Filesystem | Network filesystem for collaboration |
| **Cloud Storage** | S3 Bucket | Scalable object storage |

**Use `--verbose` to see AWS details**:
```bash
# User-friendly output (default)
prism storage list
# → Local Storage:   my-data-L (500GB)
# → Shared Storage:  lab-shared (1TB)

# AWS technical details
prism storage list --verbose
# → Local Storage (EBS gp3):   my-data-L (vol-abc123, 500GB, 3000 IOPS)
# → Shared Storage (EFS):      lab-shared (fs-def456, 1TB, General Purpose)
```

---

## Compute Terminology

### Instance Sizing

**Prism Sizes** (Simple):
```bash
prism launch python-ml my-project --size L
```

Sizes: `XS`, `S`, `M`, `L`, `XL`
- **XS**: 1 vCPU, 2GB RAM, 100GB storage
- **S**: 2 vCPU, 4GB RAM, 500GB storage
- **M**: 2 vCPU, 8GB RAM, 1TB storage
- **L**: 4 vCPU, 16GB RAM, 2TB storage
- **XL**: 8 vCPU, 32GB RAM, 4TB storage

**AWS Instance Types** (Precise):
```bash
prism launch python-ml my-project --instance-type t3.xlarge
```

**Use `--verbose` to see AWS instance type**:
```bash
# Default output
prism list
# → my-project   running   Size: L   $2.40/day

# AWS details
prism list --verbose
# → my-project   running   t3.xlarge (4vCPU, 16GB)   $2.40/day
```

---

## Cost & Optimization Terminology

### Hibernation

**Prism**: Hibernate a workspace
**AWS**: Hibernate an EC2 instance

Both terms refer to the same AWS hibernation feature - pausing compute while preserving RAM state.

### Spot Workspaces

**Prism**: Spot workspace
**AWS**: Spot instance

Uses AWS EC2 Spot Instances for 60-90% cost savings (with potential interruption).

---

## Technical Reference Terms

### Terms that Remain AWS-Specific

Some terms are inherently technical and remain AWS-specific in Prism:

| Term | Context | Why Unchanged |
|------|---------|---------------|
| **EC2 Instance Types** | Technical sizing | Industry-standard classification (t3.large, m5.xlarge, etc.) |
| **Spot Instances** | Cost optimization | Established AWS pricing model |
| **Instance ID** | System internals | AWS resource identifier (i-1234567890abcdef0) |
| **AMI** | Template optimization | AWS Machine Image - technical artifact |
| **VPC/Subnet** | Network configuration | AWS networking concepts |
| **Security Groups** | Network security | AWS firewall rules |
| **IAM Roles** | Authentication | AWS identity and access management |

**When you see these terms**: They refer to AWS infrastructure concepts and are intentionally preserved for technical accuracy.

---

## Region & Availability

**Prism**: Region
**AWS**: AWS Region
**Same meaning**: Geographic location of AWS data centers

**Examples**:
- `us-west-2` - US West (Oregon)
- `us-east-1` - US East (N. Virginia)
- `eu-west-1` - Europe (Ireland)

---

## Templates & AMIs

### Templates

**Prism Term**: Template
**What it is**: Pre-configured research environment (e.g., "Python Machine Learning", "R Research")
**Contains**: Software packages, system configuration, user setup

**AWS Equivalent**: Combination of AMI, user data (cloud-init), and configuration

### AMIs (Advanced)

**Prism**: AMI
**AWS**: Amazon Machine Image
**Purpose**: Pre-built snapshot of a template for faster launching (30s vs 5-8 minutes)

**When you see "AMI"**: This is an advanced performance optimization feature. Most users only need templates.

---

## Lifecycle States

| Prism State | AWS EC2 State | Meaning |
|------------------------|---------------|---------|
| **Running** | running | Workspace is active and billable |
| **Stopped** | stopped | Workspace is paused (only storage billed) |
| **Hibernated** | stopped (hibernated) | Workspace paused with RAM preserved |
| **Terminated** | terminated | Workspace permanently deleted |
| **Pending** | pending | Workspace is starting up |
| **Stopping** | stopping | Workspace is shutting down |

---

## Configuration & Profiles

### Prism Profiles

**Prism Term**: Profile
**Purpose**: Manages AWS credentials, region, and configuration
**Not to be confused with**: AWS profiles (in `~/.aws/credentials`)

**How they relate**:
```bash
# Prism profile references an AWS profile
prism profile create research \
  --aws-profile my-aws-creds \
  --region us-west-2
```

**Prism profile** = AWS profile + region + Prism settings

---

## Finding Technical Details

### CLI `--verbose` Flag

Add `--verbose` to any command to see AWS technical details:

```bash
# Simple output
prism list
# → my-ml-project   running   Size: L

# Technical details
prism list --verbose
# → my-ml-project   running   t3.xlarge (i-abc123, us-west-2a)
```

### GUI Technical Mode

**Settings → Advanced → Show AWS Technical Details**

Enables:
- Instance types in workspace list
- Instance IDs in connection info
- AWS service names (EBS, EFS, S3)
- VPC/subnet information

### TUI Technical View

Press `t` in any TUI view to toggle technical details.

---

## Why This Matters

### For Researchers (Majority of Users)

- **Focus on work, not infrastructure**: "Workspace" and "storage" make sense for research computing
- **Gentle learning curve**: AWS complexity hidden by default, revealed progressively
- **Confidence**: Clear terminology reduces fear of misconfiguration or unexpected costs

### For DevOps/IT (Power Users)

- **Precise control available**: `--instance-type`, `--vpc`, `--subnet` flags for exact AWS resource specification
- **Transparency**: `--verbose` flag reveals all AWS technical details
- **Compatibility**: Can use Prism alongside native AWS tools (AWS CLI, Console)

### For AWS Experts (Contributors)

- **Code uses AWS terminology internally**: Variable names, API calls use `instance`, `instanceId`, etc.
- **Documentation for mixed audiences**: User guides use "workspace", admin guides use both terms
- **Clear distinction**: User-facing vs. implementation terminology

---

## Progressive Disclosure

Prism follows **progressive disclosure** - simple by default, detailed when needed:

| User Level | Experience | Tools |
|------------|------------|-------|
| **Beginner Researcher** | "Launch a workspace for Python ML" | GUI, simple CLI commands |
| **Intermediate** | "Configure storage and networking" | CLI with common flags |
| **Advanced** | "Fine-tune instance types and optimize costs" | CLI with `--instance-type`, cost analysis |
| **Expert** | "Full AWS infrastructure control" | CLI with all AWS flags, `--verbose`, direct AWS API access |

---

## Examples in Context

### Scenario: Launching a Workspace

**Beginner (Simple)**:
```bash
prism launch python-ml my-research
```

**Intermediate (Sized)**:
```bash
prism launch python-ml my-research --size L
```

**Advanced (Spot + Storage)**:
```bash
prism launch python-ml my-research \
  --size L \
  --spot \
  --attach-storage my-data
```

**Expert (Full Control)**:
```bash
prism launch python-ml my-research \
  --instance-type c5.4xlarge \
  --spot \
  --subnet subnet-abc123 \
  --security-group sg-def456 \
  --verbose
```

### Scenario: Checking Workspace Status

**Beginner**:
```bash
prism list
# → my-research   running   $2.40/day
```

**Advanced**:
```bash
prism list --verbose
# → my-research   running   c5.4xlarge (i-abc123456789, us-west-2b)   $2.40/day
```

**Expert**:
```bash
aws ec2 describe-instances --instance-ids i-abc123456789
# → Full AWS EC2 API response
```

---

## Summary

| **For Most Users** | **For AWS Experts** |
|--------------------|---------------------|
| Use **workspaces**, **storage**, **regions** | Add `--verbose` to see AWS details |
| Focus on research, not infrastructure | Use `--instance-type` for precise control |
| GUI and simple CLI commands | Full AWS flag support in CLI |
| Prism handles AWS complexity | Direct AWS API access available |

**Key Principle**: Prism meets users where they are - simple for researchers, powerful for experts.

---

**See Also**:
- [Issue #15 - Instances → Workspaces Rename](https://github.com/scttfrdmn/prism/issues/15)
- [Issue #66 - Storage Terminology Simplification](https://github.com/scttfrdmn/prism/issues/66)
- [Design Principles](../DESIGN_PRINCIPLES.md)
- [User Requirements](../USER_REQUIREMENTS.md)
