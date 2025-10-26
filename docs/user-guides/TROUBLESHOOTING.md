# Prism Troubleshooting Guide

## Quick Fixes for Common Issues

### 🚨 "daemon not running" Error

**What you see:**
```
Error: daemon not running. Start with: prism daemon start
```

**Quick fix:**
```bash
# Start the daemon
prism daemon start

# Verify it's running
prism daemon status
```

**If daemon won't start:**
```bash
# Check if something is using port 8947
lsof -i :8947

# Kill conflicting process if found
kill -9 <PID>

# Try starting again
prism daemon start
```

---

### 🔐 AWS Credential Issues

**What you see:**
```
Error: AWS credentials not found
Error: UnauthorizedOperation
```

**Quick fix:**
```bash
# Check current credentials
aws sts get-caller-identity

# Configure if needed
aws configure
```

**If you have AWS credentials but Prism can't find them:**
```bash
# Check AWS profile
echo $AWS_PROFILE

# Set profile if needed
export AWS_PROFILE=your-profile-name

# Or specify directly
prism launch python-ml my-project --profile your-profile-name
```

---

### 🏗️ Template Launch Failures

**What you see:**
```
Error: failed to launch instance
Error: VPC not found
Error: subnet not available
```

**Quick fix:**
```bash
# Prism auto-discovers VPC/subnet (new feature!)
prism launch python-ml my-project

# If auto-discovery fails, check your VPC setup
aws ec2 describe-vpcs --query 'Vpcs[?IsDefault==`true`]'
```

**If you don't have a default VPC:**
```bash
# Create a default VPC
aws ec2 create-default-vpc

# Or specify manually (advanced)
prism ami build python-ml --vpc vpc-12345 --subnet subnet-67890
```

---

### 💰 Cost and Pricing Concerns

**What you see:**
```
Instance cost seems high
Unexpected AWS charges
```

**Quick fix:**
```bash
# Check current instances and costs
prism list

# Stop unused instances
prism stop instance-name

# Use hibernation to preserve work and reduce costs
prism hibernate instance-name

# Enable auto-hibernation for idle instances
prism idle enable
```

**Cost optimization commands:**
```bash
# Use smaller instance sizes
prism launch python-ml my-project --size S

# Use spot instances (up to 90% savings)
prism launch python-ml my-project --spot

# Check institutional pricing discounts
prism pricing show
```

---

### 🔌 Connection Problems

**What you see:**
```
Connection timeout
SSH connection refused
Can't access Jupyter/RStudio
```

**Quick fix:**
```bash
# Check instance status
prism list

# Ensure instance is running
prism start instance-name

# Get fresh connection info
prism connect instance-name
```

**If SSH still fails:**
```bash
# Check security group settings
prism list --verbose

# Wait for instance to fully boot (can take 2-3 minutes)
# Then try connecting again
```

---

### 🧠 Memory and Performance Issues

**What you see:**
```
Instance running slowly
Out of memory errors
Jupyter kernel crashes
```

**Quick fix:**
```bash
# Use larger instance size
prism stop instance-name
prism launch python-ml instance-name --size L

# Or add more storage
prism storage create extra-space XL
prism storage attach extra-space instance-name
```

---

### 📦 Template and Package Issues

**What you see:**
```
Package not found
Template validation failed
Command not available in instance
```

**Quick fix:**
```bash
# Validate template before launching
prism templates validate python-ml

# Check template contents
prism templates info python-ml

# Apply additional packages to running instance
prism apply docker-tools instance-name
```

**If template seems broken:**
```bash
# Force refresh template cache
rm -rf ~/.prism/templates
prism templates

# Use AMI-based templates for reliability
prism templates | grep "(AMI)"
```

---

### 🌍 Region and Availability Issues

**What you see:**
```
Insufficient capacity
Instance type not available
AMI not found in region
```

**Quick fix:**
```bash
# Try different region
prism launch python-ml my-project --region us-east-1

# Use different instance size
prism launch python-ml my-project --size M

# Check region availability
aws ec2 describe-availability-zones --region us-west-2
```

---

### 🔧 GUI and Interface Issues

**What you see:**
```
GUI won't start
TUI looks broken
Interface unresponsive
```

**Quick fix:**
```bash
# For GUI issues on macOS
# Allow keychain access when prompted (now shows "Prism")

# For TUI display issues
export TERM=xterm-256color
prism tui

# For interface problems, use CLI as backup
prism list
prism connect instance-name
```

---

## Advanced Troubleshooting

### Enable Debug Logging
```bash
# Set debug mode
export PRISM_DEBUG=1

# Check daemon logs
prism daemon logs

# Or run commands with verbose output
prism launch python-ml my-project --verbose
```

### Check System Requirements
```bash
# Verify AWS CLI version (need v2+)
aws --version

# Check Prism version
prism version

# Verify network connectivity
curl -I https://ec2.amazonaws.com
```

### Reset Prism
```bash
# Stop daemon
prism daemon stop

# Clear cache and state
rm -rf ~/.prism/

# Restart fresh
prism daemon start
```

---

## Getting Help

### Before Opening an Issue

1. **Check daemon status**: `prism daemon status`
2. **Verify AWS credentials**: `aws sts get-caller-identity`  
3. **Try CLI interface**: Sometimes GUI/TUI have display issues
4. **Check recent changes**: Did you update AWS credentials or change regions?

### Include This Information

When asking for help, please include:

```bash
# Prism version
prism version

# Daemon status
prism daemon status

# AWS account info (no credentials)
aws sts get-caller-identity --query 'Account'

# Operating system
uname -a

# Error message (full text)
```

### Community Support

- **GitHub Issues**: [Report bugs and request features](https://github.com/scttfrdmn/prism/issues)
- **Discussions**: [Get help from the community](https://github.com/scttfrdmn/prism/discussions)
- **Documentation**: [Complete guides in `/docs` folder](docs/)

---

## Emergency Recovery

### Instance Stuck in Bad State
```bash
# Force stop and restart
prism stop instance-name --force
prism start instance-name

# Or delete and recreate
prism save instance-name backup-template  # Save work first
prism delete instance-name
prism launch backup-template instance-name-new
```

### Accidentally Deleted Important Instance
```bash
# Check if hibernated (data may be preserved)
prism list --all

# Look for recent AMI snapshots
prism ami list

# Contact AWS support for EBS snapshot recovery if critical
```

### Unexpected High AWS Bills
```bash
# Immediately stop all instances
prism list | grep running | awk '{print $1}' | xargs -I {} prism stop {}

# Check what's still running
prism list

# Review hibernation options for the future
prism idle profile list
```

**Remember**: Prism is designed to "default to success." Most issues have simple solutions, and the error messages are designed to guide you to the fix.