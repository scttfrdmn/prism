# CloudWorkstation Demo Tester Setup Guide

**Quick Start**: Get CloudWorkstation running on your AWS account in under 10 minutes.

## Prerequisites

- AWS account with billing enabled
- Basic command-line familiarity
- Homebrew (macOS/Linux) or direct binary download

## 1. AWS Account Setup

### Required AWS Permissions

CloudWorkstation needs the following AWS permissions. Create an IAM user or use an existing one with these policies:

#### Minimal IAM Policy (JSON)
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:*",
                "efs:*",
                "iam:GetRole",
                "iam:CreateRole",
                "iam:AttachRolePolicy",
                "iam:PassRole",
                "iam:CreateInstanceProfile",
                "iam:AddRoleToInstanceProfile"
            ],
            "Resource": "*"
        }
    ]
}
```

#### AWS CLI Configuration

1. **Install AWS CLI**: 
   ```bash
   curl "https://awscli.amazonaws.com/AWSCLIV2.pkg" -o "AWSCLIV2.pkg"
   sudo installer -pkg AWSCLIV2.pkg -target /
   ```

2. **Configure credentials**:
   ```bash
   aws configure
   # Enter your Access Key ID, Secret Access Key, and default region
   # Recommended regions: us-west-2, us-east-1, eu-west-1
   ```

3. **Verify access**:
   ```bash
   aws sts get-caller-identity
   aws ec2 describe-regions
   ```

### Default VPC Setup (Optional)

CloudWorkstation works with your default VPC. If you don't have one:

```bash
aws ec2 create-default-vpc
```

## 2. Install CloudWorkstation

### Option A: Homebrew (Recommended)
```bash
# Add CloudWorkstation tap
brew tap scttfrdmn/cloudworkstation

# Install CloudWorkstation
brew install cloudworkstation

# Verify installation
cws version
```

### Option B: Direct Binary Download
```bash
# Download latest release for your platform
# macOS ARM64
curl -L -o cws https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cws-darwin-arm64
curl -L -o cwsd https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cwsd-darwin-arm64

# macOS AMD64
curl -L -o cws https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cws-darwin-amd64
curl -L -o cwsd https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cwsd-darwin-amd64

# Linux AMD64
curl -L -o cws https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cws-linux-amd64
curl -L -o cwsd https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cwsd-linux-amd64

# Make executable and move to PATH
chmod +x cws cwsd
sudo mv cws cwsd /usr/local/bin/
```

### Option C: Build from Source
```bash
git clone https://github.com/scttfrdmn/cloudworkstation.git
cd cloudworkstation
make build

# Binaries will be in ./bin/
export PATH=$PATH:$(pwd)/bin
```

## 3. Initial Setup

### Start CloudWorkstation Daemon
```bash
# Start the background service
cws daemon start

# Verify it's running
cws daemon status
```

### Test Connection
```bash
# List available research templates
cws templates

# Should show templates like:
# 🏗️  Python Machine Learning
# 🏗️  R Research Environment  
# 🏗️  Web Development
```

## 4. Your First Research Environment

### Launch a Test Environment
```bash
# Launch Python ML environment (takes 2-3 minutes)
cws launch "Python Machine Learning" demo-ml-test

# Monitor progress
cws list
```

### Connect to Your Environment
```bash
# Get connection details
cws connect demo-ml-test

# Should provide SSH command and Jupyter URL
```

### Clean Up (IMPORTANT - Prevents Charges)
```bash
# Stop the instance
cws stop demo-ml-test

# Or hibernate (preserves state)
cws hibernate demo-ml-test

# Delete completely
cws delete demo-ml-test
```

## 5. Cost Management

### Default Cost Controls
CloudWorkstation includes several cost-saving features:

- **Spot instances** by default where appropriate
- **Automatic hibernation** after idle periods
- **Instance sizing optimization** per research domain

### View Costs
```bash
# See estimated daily costs
cws list

# View pricing configuration
cws pricing show
```

### Set Budget Alerts (Optional)
```bash
# Create project with budget limit
cws project create my-research --budget 100

# Launch instances under project
cws launch python-ml my-analysis --project my-research
```

## 6. Advanced Configuration (Optional)

### Custom AWS Profile
```bash
# Use specific AWS profile
export AWS_PROFILE=research
cws templates
```

### Custom Region
```bash
# Launch in specific region
cws launch python-ml my-test --region us-east-1
```

### Instance Sizing
```bash
# Use larger instance for intensive work
cws launch python-ml gpu-training --size GPU-L
```

## 7. Troubleshooting

### Common Issues

**"daemon not running" error**:
```bash
# Check daemon status
cws daemon status

# Restart daemon
cws daemon stop
cws daemon start
```

**AWS permission errors**:
```bash
# Verify AWS credentials
aws sts get-caller-identity

# Check if you have EC2 permissions
aws ec2 describe-regions
```

**Template launch failures**:
```bash
# Check AWS region has sufficient capacity
aws ec2 describe-availability-zones

# Try different region
cws launch python-ml test --region us-west-2
```

**Unexpected AWS charges**:
```bash
# List all running instances
cws list

# Stop everything
cws stop $(cws list --format names)
```

### Support

- **Documentation**: [GitHub Repository](https://github.com/scttfrdmn/cloudworkstation)
- **Issues**: [Report bugs and feature requests](https://github.com/scttfrdmn/cloudworkstation/issues)
- **Community**: [Discussions](https://github.com/scttfrdmn/cloudworkstation/discussions)

## 8. What to Test

### Core Functionality
1. **Template System**: Try different research templates
2. **Instance Management**: Launch, stop, start, hibernate
3. **Storage**: Create and attach EFS volumes
4. **Projects**: Create projects with budgets
5. **Cost Tracking**: Monitor spending estimates

### Advanced Features
1. **Multi-user Projects**: Add team members to projects
2. **Template Inheritance**: Explore how templates build on each other  
3. **Pricing Integration**: Test institutional discount configurations
4. **GUI Interface**: Try `cws-gui` for visual management

### Provide Feedback On
- **Ease of initial setup** - How long from AWS account to working environment?
- **Template usefulness** - Do the research environments meet your needs?
- **Cost transparency** - Are costs clear and predictable?
- **Error messages** - Are failures explained clearly with next steps?
- **Documentation gaps** - What information was missing or confusing?

---

**Expected Demo Timeline**: 
- Initial setup: 5-10 minutes
- First environment launch: 3-5 minutes  
- Total time to productive research environment: < 15 minutes

**Cost Warning**: Running CloudWorkstation will incur AWS charges. Demo environments cost approximately $0.10-$2.00 per hour depending on instance size. Always clean up test resources!