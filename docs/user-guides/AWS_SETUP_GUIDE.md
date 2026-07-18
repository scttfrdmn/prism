# Prism AWS Setup Guide

This guide walks you through setting up your AWS account and configuring Prism to work with your specific AWS profile and preferences.

## Prerequisites

- AWS Account with programmatic access
- AWS CLI installed on your system
- Prism installed via Homebrew or built from source

## 1. AWS Account Setup

### Required AWS Permissions

Prism needs these AWS permissions to function properly:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:*",
        "efs:*",
        "ssm:*",
        "iam:PassRole",
        "iam:CreateRole",
        "iam:AttachRolePolicy",
        "iam:CreateInstanceProfile",
        "iam:AddRoleToInstanceProfile"
      ],
      "Resource": "*"
    }
  ]
}
```

### Create IAM User for Prism

1. **Log into AWS Console** → IAM → Users → Create User
2. **User name**: `prism-user`
3. **Access type**: Programmatic access (Access key + Secret key)
4. **Permissions**: Attach the policy above or use `PowerUserAccess` for simplicity
5. **Download credentials**: Save the Access Key ID and Secret Access Key

## 2. AWS CLI Authentication

### Install AWS CLI v2

```bash
# macOS
brew install awscli

# Linux
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip && sudo ./aws/install

# Verify (must be v2.32+ for aws login)
aws --version
```

### Option A: Browser-based login (recommended)

`aws login` authenticates via your browser using IAM user or federated identity credentials — no key management required. Requires AWS CLI v2.32+.

```bash
# Authenticate (opens browser)
aws login

# For a named profile
aws login --profile prism-research

# On headless/remote machines
aws login --remote
```

Credentials are cached in `~/.aws/cli/cache/` for up to 12 hours and auto-refresh. Re-run `aws login` when they expire.

Verify:
```bash
aws sts get-caller-identity
```

### Option B: Long-term access keys

For CI/CD pipelines, headless servers, or users who prefer static credentials:

```bash
aws configure --profile prism-research
# AWS Access Key ID:     [Your Access Key]
# AWS Secret Access Key: [Your Secret Key]
# Default region name:   us-west-2
# Default output format: json
```

This writes to `~/.aws/credentials`:

```ini
[prism-research]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

And `~/.aws/config`:

```ini
[profile prism-research]
region = us-west-2
output = json
```

Verify:
```bash
aws sts get-caller-identity --profile prism-research
```

## 3. Prism Configuration

### Method 1: Prism Profiles (Recommended)

Prism has its own profile system for managing different AWS accounts and configurations:

```bash
# Create a Prism profile using your 'aws' AWS profile
prism profiles add personal my-research --aws-profile aws --region us-west-2

# Switch to your new profile
prism profiles switch aws  # Use the AWS profile name as the profile ID

# Verify it's active
prism profiles current
prism profiles list
```

**This is the cleanest method** - Prism remembers your settings and you don't need environment variables.

### Method 2: Environment Variables

Set these in your shell profile (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
# Use your custom 'aws' profile
export AWS_PROFILE=aws
export AWS_REGION=us-west-2

# Optional: Set development mode to avoid keychain prompts
export PRISM_DEV=true
```

Then restart your terminal or run `source ~/.zshrc`.

### Method 3: Command-Line Override

You can specify the AWS profile for individual commands:

```bash
# Set environment variable for single session
AWS_PROFILE=aws prism templates

# Or use Prism's profile system
prism --aws-profile aws templates list
```

## 4. Verification and Testing

### Quick Health Check

```bash
# Check daemon status (daemon auto-starts as needed)
prism admin daemon status

# List available templates (requires AWS access)
prism templates

# Check your current configuration
prism profiles current
aws configure list --profile prism-research
```

### Test Instance Launch (Optional)

```bash
# Launch a simple test instance
prism workspace launch "Basic Ubuntu (APT)" test-instance

# Check it's running
prism workspace list

# Clean up
prism workspace delete test-instance
```

## 5. Regional Configuration

### Choose Your Region

Consider these factors when selecting your AWS region:

- **Cost**: Pricing varies by region
- **Latency**: Choose closer to your location  
- **Available Instance Types**: Some regions have better GPU/specialized instance availability
- **Data Residency**: Regulatory requirements

### Popular Regions for Research

```bash
# US West (Oregon) - Good for West Coast, often cheapest
export AWS_REGION=us-west-2

# US East (Virginia) - Good for East Coast, most services available
export AWS_REGION=us-east-1

# EU (Ireland) - Good for European users
export AWS_REGION=eu-west-1

# Asia Pacific (Sydney) - Good for APAC users
export AWS_REGION=ap-southeast-2
```

### Set Default Region

```bash
# Update your AWS profile's default region
aws configure set region us-west-2 --profile prism-research

# Or set via environment variable
export AWS_REGION=us-west-2
```

## 6. Troubleshooting

### Common Issues

**"No credentials found" error:**
```bash
# Check your profile exists
aws configure list --profile prism-research

# Verify environment variable
echo $AWS_PROFILE

# Test credentials manually
aws sts get-caller-identity --profile aws
```

**"Permission denied" errors:**
```bash
# Check your IAM permissions
aws iam get-user --profile aws

# Test EC2 access specifically
aws ec2 describe-instances --profile aws --region us-west-2
```

**Prism can't find your profile:**
```bash
# Create explicit Prism profile
prism profiles add personal research --aws-profile aws --region us-west-2
prism profiles switch research

# Verify it's active
prism profiles current
```

### Debug Mode

Enable verbose logging to see what Prism is doing:

```bash
# Set debug environment variables
export AWS_PROFILE=aws
export PRISM_DEBUG=true

# Run commands with detailed output
prism templates
```

## 7. Production Recommendations

### Security Best Practices

1. **Use IAM Roles**: For EC2 instances that need AWS access
2. **Rotate Keys**: Regularly rotate your access keys
3. **Least Privilege**: Only grant necessary permissions
4. **MFA**: Enable Multi-Factor Authentication on your AWS account

### Cost Management

```bash
# Set up billing alerts in AWS Console
# Enable Cost Explorer
# Use spot instances for non-critical workloads
prism workspace launch "Python Machine Learning (Simplified)" my-project --spot

# Use hibernation for cost savings
prism workspace hibernate my-project
```

### Profile Organization

```bash
# Organize profiles by project/purpose
prism profiles add personal personal-research --aws-profile aws --region us-west-2
prism profiles add personal team-project --aws-profile work --region us-east-1
prism profiles add personal gpu-experiments --aws-profile aws --region us-west-2
```

## 8. Example Complete Setup

Here's a complete example for your specific case using Prism profiles:

```bash
# 1. Configure AWS CLI with 'aws' profile
aws login --profile prism-research  # or: aws configure --profile prism-research
# Enter your credentials when prompted

# 2. Create Prism profile (RECOMMENDED METHOD)
prism profiles add personal my-research --aws-profile aws --region us-west-2
prism profiles switch aws  # Switch to use your 'aws' profile

# 3. Verify configuration
prism profiles current

# 4. Launch your first workstation
prism workspace launch "Python Machine Learning (Simplified)" my-research

# 5. Connect and start working
prism workspace connect my-research
```

### Alternative Setup (Environment Variables)

If you prefer environment variables:

```bash
# 1. Configure AWS CLI with 'aws' profile  
aws login --profile prism-research  # or: aws configure --profile prism-research

# 2. Set environment variables
export AWS_PROFILE=aws
export AWS_REGION=us-west-2
export PRISM_DEV=true

# 3. Make permanent
echo 'export AWS_PROFILE=aws' >> ~/.zshrc
echo 'export AWS_REGION=us-west-2' >> ~/.zshrc

# 4. Test and launch (daemon auto-starts)
prism templates
prism workspace launch "Python Machine Learning (Simplified)" my-research
```

## Need Help?

- **AWS Issues**: `aws sts get-caller-identity --profile aws`
- **Documentation**: Run `prism --help` for command reference
- **GitHub Issues**: https://github.com/scttfrdmn/prism/issues

Your AWS profile 'aws' should now work seamlessly with Prism!