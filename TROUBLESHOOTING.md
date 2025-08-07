# CloudWorkstation Troubleshooting Guide

## Quick Fixes for Common Issues

### 🚨 "daemon not running" Error

**What you see:**
```
Error: daemon not running. Start with: cws daemon start
```

**Quick fix:**
```bash
# Start the daemon
cws daemon start

# Verify it's running
cws daemon status
```

**If daemon won't start:**
```bash
# Check if something is using port 8947
lsof -i :8947

# Kill conflicting process if found
kill -9 <PID>

# Try starting again
cws daemon start
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

**If you have AWS credentials but CloudWorkstation can't find them:**
```bash
# Check AWS profile
echo $AWS_PROFILE

# Set profile if needed
export AWS_PROFILE=your-profile-name

# Or specify directly
cws launch python-ml my-project --profile your-profile-name
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
# CloudWorkstation auto-discovers VPC/subnet (new feature!)
cws launch python-ml my-project

# If auto-discovery fails, check your VPC setup
aws ec2 describe-vpcs --query 'Vpcs[?IsDefault==`true`]'
```

**If you don't have a default VPC:**
```bash
# Create a default VPC
aws ec2 create-default-vpc

# Or specify manually (advanced)
cws ami build python-ml --vpc vpc-12345 --subnet subnet-67890
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
cws list

# Stop unused instances
cws stop instance-name

# Use hibernation to preserve work and reduce costs
cws hibernate instance-name

# Enable auto-hibernation for idle instances
cws idle enable
```

**Cost optimization commands:**
```bash
# Use smaller instance sizes
cws launch python-ml my-project --size S

# Use spot instances (up to 90% savings)
cws launch python-ml my-project --spot

# Check institutional pricing discounts
cws pricing show
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
cws list

# Ensure instance is running
cws start instance-name

# Get fresh connection info
cws connect instance-name
```

**If SSH still fails:**
```bash
# Check security group settings
cws list --verbose

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
cws stop instance-name
cws launch python-ml instance-name --size L

# Or add more storage
cws storage create extra-space XL
cws storage attach extra-space instance-name
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
cws templates validate python-ml

# Check template contents
cws templates info python-ml

# Apply additional packages to running instance
cws apply docker-tools instance-name
```

**If template seems broken:**
```bash
# Force refresh template cache
rm -rf ~/.cloudworkstation/templates
cws templates

# Use AMI-based templates for reliability
cws templates | grep "(AMI)"
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
cws launch python-ml my-project --region us-east-1

# Use different instance size
cws launch python-ml my-project --size M

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
# Allow keychain access when prompted (now shows "CloudWorkstation")

# For TUI display issues
export TERM=xterm-256color
cws tui

# For interface problems, use CLI as backup
cws list
cws connect instance-name
```

---

## Advanced Troubleshooting

### Enable Debug Logging
```bash
# Set debug mode
export CWS_DEBUG=1

# Check daemon logs
cws daemon logs

# Or run commands with verbose output
cws launch python-ml my-project --verbose
```

### Check System Requirements
```bash
# Verify AWS CLI version (need v2+)
aws --version

# Check CloudWorkstation version
cws version

# Verify network connectivity
curl -I https://ec2.amazonaws.com
```

### Reset CloudWorkstation
```bash
# Stop daemon
cws daemon stop

# Clear cache and state
rm -rf ~/.cloudworkstation/

# Restart fresh
cws daemon start
```

---

## Getting Help

### Before Opening an Issue

1. **Check daemon status**: `cws daemon status`
2. **Verify AWS credentials**: `aws sts get-caller-identity`  
3. **Try CLI interface**: Sometimes GUI/TUI have display issues
4. **Check recent changes**: Did you update AWS credentials or change regions?

### Include This Information

When asking for help, please include:

```bash
# CloudWorkstation version
cws version

# Daemon status
cws daemon status

# AWS account info (no credentials)
aws sts get-caller-identity --query 'Account'

# Operating system
uname -a

# Error message (full text)
```

### Community Support

- **GitHub Issues**: [Report bugs and request features](https://github.com/scttfrdmn/cloudworkstation/issues)
- **Discussions**: [Get help from the community](https://github.com/scttfrdmn/cloudworkstation/discussions)
- **Documentation**: [Complete guides in `/docs` folder](docs/)

---

## Emergency Recovery

### Instance Stuck in Bad State
```bash
# Force stop and restart
cws stop instance-name --force
cws start instance-name

# Or delete and recreate
cws save instance-name backup-template  # Save work first
cws delete instance-name
cws launch backup-template instance-name-new
```

### Accidentally Deleted Important Instance
```bash
# Check if hibernated (data may be preserved)
cws list --all

# Look for recent AMI snapshots
cws ami list

# Contact AWS support for EBS snapshot recovery if critical
```

### Unexpected High AWS Bills
```bash
# Immediately stop all instances
cws list | grep running | awk '{print $1}' | xargs -I {} cws stop {}

# Check what's still running
cws list

# Review hibernation options for the future
cws idle profile list
```

**Remember**: CloudWorkstation is designed to "default to success." Most issues have simple solutions, and the error messages are designed to guide you to the fix.