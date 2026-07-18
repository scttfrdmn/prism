# Prism Zero-Setup Guide

## 🎯 The Zero-Setup Philosophy

Prism is designed to work **immediately** after installation, with zero configuration required. This guide explains how the zero-setup experience works and what happens automatically behind the scenes.

## ✨ What is Zero-Setup?

Zero-setup means you can go from installation to running workstation in **one command**:

```bash
# Install Prism (see the Installation guide for all platforms)
brew install scttfrdmn/tap/prism   # macOS/Linux
# or: scoop install prism          # Windows

# Launch a workstation - that's it!
prism workspace launch python-ml my-research
```

For all install methods, see the [Installation guide](INSTALLATION.md).

**No configuration files. No setup scripts. No manual steps.**

## 🚀 Automatic Features

### 1. Daemon Auto-Start

The Prism daemon (`prismd`) starts automatically when needed:

```bash
prism workspace launch template my-instance
# ✅ Daemon starts automatically if not running
# ✅ No need for: prism admin daemon start
# ✅ No systemd/launchd configuration needed
```

**How it works:**
- CLI checks if daemon is running on port 8947
- If not running, starts daemon in background
- Daemon continues running for future commands
- Gracefully shuts down when idle

### 2. AWS Credential Discovery

Prism automatically finds your AWS credentials:

```bash
# Already have AWS CLI configured?
aws configure  # Your existing setup

# Prism finds credentials automatically:
prism workspace launch template my-instance
# ✅ Checks AWS_PROFILE environment variable
# ✅ Checks ~/.aws/credentials
# ✅ Checks ~/.aws/config for region
# ✅ Uses IAM instance profile if on EC2
```

**Discovery order:**
1. Environment variables (`AWS_PROFILE`, `AWS_ACCESS_KEY_ID`)
2. AWS CLI configuration (`~/.aws/credentials`)
3. Prism profiles (`~/.prism/profiles.yaml`)
4. IAM instance profile (when running on EC2)

### 3. Intelligent Region Selection

No need to specify regions - Prism figures it out:

```bash
prism workspace launch template my-instance
# ✅ Uses region from AWS config
# ✅ Falls back to us-west-2 if not set
# ✅ Validates template works in selected region
# ✅ Suggests alternatives if resources unavailable
```

**Region precedence:**
1. Command line flag: `--region us-east-1`
2. Prism profile setting
3. AWS_DEFAULT_REGION environment variable
4. AWS CLI config file (`~/.aws/config`)
5. Default: `us-west-2`

### 4. Template Validation & Fallbacks

Templates automatically adapt to your environment:

```bash
prism workspace launch "Python Machine Learning (Simplified)" my-ml
# ✅ Checks if GPU instances available in region
# ✅ Falls back to CPU instance if needed
# ✅ Validates AMIs exist in region
# ✅ Adjusts for regional pricing differences
```

**Automatic fallbacks:**
- GPU → CPU instances if GPUs unavailable
- ARM → x86 architecture if ARM unavailable
- Larger → smaller instance sizes if capacity limited
- Always communicates changes clearly

### 5. SSH Key Management

SSH keys are generated and managed automatically:

```bash
prism workspace connect my-instance
# ✅ SSH key generated on first use
# ✅ Stored securely in ~/.ssh/
# ✅ Uploaded to AWS automatically
# ✅ Permissions set correctly (600)
```

**Key management:**
- Key name: `prism-<profile>-key`
- Location: `~/.ssh/prism-<profile>-key`
- AWS KeyPair created automatically
- Reused across instances in same profile

### 6. Network Configuration

VPC and security groups configured automatically:

```bash
prism workspace launch template my-instance
# ✅ Uses default VPC if available
# ✅ Creates security group with proper rules
# ✅ Opens only required ports (22, 443, template-specific)
# ✅ Configures public IP for access
```

**Network setup:**
- Discovers default VPC
- Creates `prism-sg` security group
- Adds rules for SSH and template services
- Enables public IP assignment

### 7. Storage Configuration

Storage optimized automatically:

```bash
prism workspace launch template my-instance --size L
# ✅ SSD (gp3) storage by default
# ✅ Size adjusted based on template needs
# ✅ Encryption enabled for security
# ✅ Snapshot on termination for safety
```

**Storage defaults:**
- Type: `gp3` (latest generation SSD)
- Size: Template-specific (20-100GB)
- Encryption: Enabled by default
- Delete on termination: Yes (with snapshot)

## 🎨 Progressive Disclosure

Start simple, add complexity only when needed:

### Level 1: Absolute Beginner
```bash
# Just launch with defaults
prism workspace launch "R Research Environment (Simplified)" my-analysis
```

### Level 2: Basic Customization
```bash
# Specify size
prism workspace launch "R Research Environment (Simplified)" my-analysis --size L
```

### Level 3: Advanced Options
```bash
# Full control
prism workspace launch "R Research Environment (Simplified)" my-analysis \
  --size XL \
  --region eu-west-1 \
  --spot \
  --idle-policy
```

### Level 4: Expert Mode
```bash
# Complete customization
prism workspace launch template my-instance \
  --instance-type r6i.2xlarge \
  --subnet subnet-abc123 \
  --security-group sg-def456 \
  --volume 500 \
  --param notebook_password=secret
```

## 🔍 Troubleshooting Zero-Setup

### Issue: "AWS credentials not found"

**Solution:** Authenticate once (browser-based):
```bash
aws login
# Opens a browser to sign in; caches credentials for ~12h.
# Prefer long-term keys? Use `aws configure` instead.
```

### Issue: "No default VPC in region"

**Solution:** Prism will prompt to create one:
```bash
prism workspace launch template my-instance
# ⚠️  No default VPC found in us-west-2
# Would you like to create one? [Y/n]: Y
# ✅ Default VPC created
```

### Issue: "Instance type not available"

**Solution:** Automatic fallback with notification:
```bash
prism workspace launch gpu-template my-training
# ⚠️  GPU instance g4dn.xlarge not available in us-west-2
# ✅ Using g4dn.xlarge in us-east-1 instead
# Proceed? [Y/n]: Y
```

## 📚 Advanced Configuration

While zero-setup works for most users, power users can customize:

### Prism Profiles

Manage multiple AWS accounts:
```bash
# Add a research account
prism profiles add personal research \
  --aws-profile research \
  --region eu-central-1

# Add a personal account  
prism profiles add personal personal \
  --aws-profile personal \
  --region us-west-2

# Switch between them
prism profiles switch research
```

### Configuration File

Optional configuration (`~/.prism/config.yaml`):
```yaml
defaults:
  region: us-west-2
  instance_size: M
  enable_spot: false
  idle_policy: balanced

daemon:
  port: 8947
  auto_start: true
  log_level: info
```

### Environment Variables

Override any setting:
```bash
export PRISM_DEFAULT_REGION=eu-west-1
export PRISM_DEFAULT_SIZE=L
export PRISM_DAEMON_PORT=8948
export PRISM_AUTO_START=false
```

## 🚀 Quick Examples

### Data Science Workstation
```bash
# One command to productivity
prism workspace launch "Python Machine Learning (Simplified)" notebook

# What happens automatically:
# ✅ Starts daemon
# ✅ Finds AWS credentials
# ✅ Selects optimal GPU instance
# ✅ Configures Jupyter
# ✅ Sets up SSH access
# ✅ Returns connection info
```

### R Statistical Analysis
```bash
# Launch RStudio environment
prism workspace launch "R Research Environment (Simplified)" stats

# Automatic setup:
# ✅ Memory-optimized instance selection
# ✅ RStudio Server configuration
# ✅ Required R packages installation
# ✅ Persistent storage setup
```

### Development Environment
```bash
# Web development setup
prism workspace launch "Web Development (APT)" webapp

# Zero-config features:
# ✅ Docker pre-installed
# ✅ Node.js configured
# ✅ Ports 3000, 8080 open
# ✅ VS Code Server ready
```

## 💡 Best Practices

1. **Start with defaults** - They're optimized for most use cases
2. **Use templates** - Pre-configured for specific workflows
3. **Enable idle policies** - Automatic cost optimization
4. **Trust the fallbacks** - Prism makes smart choices
5. **Check status regularly** - `prism workspace list` shows all instances

## 🎯 The Zero-Setup Promise

Prism maintains its zero-setup promise by:

- **Sensible defaults** that work for 90% of use cases
- **Automatic discovery** of existing configurations
- **Intelligent fallbacks** when ideal resources aren't available
- **Clear communication** about what's happening
- **Progressive disclosure** of advanced features

You should be doing research, not configuring infrastructure. Prism makes that possible.

## 📚 Learn More

- Quick Start Guide
- [Administrator Guide](../admin-guides/ADMINISTRATOR_GUIDE.md) (for manual AWS configuration)
- [Template Format](TEMPLATE_FORMAT.md) (creating custom templates)
- [Getting Started Guide](QUICK_START.md) (complete CLI reference)