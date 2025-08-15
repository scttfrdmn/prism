# CloudWorkstation AWS Setup Guide

This guide walks you through setting up your AWS account and configuring CloudWorkstation to work with your specific AWS profile and preferences.

## Prerequisites

- AWS Account with programmatic access
- AWS CLI installed on your system
- CloudWorkstation installed via Homebrew or built from source

## 1. AWS Account Setup

### Required AWS Permissions

CloudWorkstation needs these AWS permissions to function properly:

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

### Create IAM User for CloudWorkstation

1. **Log into AWS Console** → IAM → Users → Create User
2. **User name**: `cloudworkstation-user`
3. **Access type**: Programmatic access (Access key + Secret key)
4. **Permissions**: Attach the policy above or use `PowerUserAccess` for simplicity
5. **Download credentials**: Save the Access Key ID and Secret Access Key

## 2. AWS CLI Configuration

### Install AWS CLI (if not already installed)

```bash
# macOS
brew install awscli

# Linux
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip && sudo ./aws/install

# Verify installation
aws --version
```

### Configure AWS Profile

**For your specific use case (using profile 'aws' instead of 'default'):**

```bash
# Configure your custom AWS profile named 'aws'
aws configure --profile aws

# You'll be prompted for:
# AWS Access Key ID: [Your Access Key]
# AWS Secret Access Key: [Your Secret Key] 
# Default region name: us-west-2  # Choose your preferred region
# Default output format: json
```

### Verify Profile Configuration

```bash
# Test your profile configuration
aws sts get-caller-identity --profile aws

# Should return something like:
# {
#     "UserId": "AIDAXXXXXXXXXXXXX",
#     "Account": "123456789012", 
#     "Arn": "arn:aws:iam::123456789012:user/cloudworkstation-user"
# }
```

### Set Up AWS Credentials File

Your AWS credentials are stored in `~/.aws/credentials`:

```ini
[aws]
aws_access_key_id = YOUR_ACCESS_KEY_ID
aws_secret_access_key = YOUR_SECRET_ACCESS_KEY

[default]
aws_access_key_id = DIFFERENT_ACCESS_KEY_ID
aws_secret_access_key = DIFFERENT_SECRET_ACCESS_KEY
```

And `~/.aws/config`:

```ini
[profile aws]
region = us-west-2
output = json

[profile default]
region = us-east-1
output = json
```

## 3. CloudWorkstation Configuration

### Method 1: Environment Variables (Recommended)

Set these in your shell profile (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
# Use your custom 'aws' profile
export AWS_PROFILE=aws
export AWS_REGION=us-west-2

# Optional: Set development mode to avoid keychain prompts
export CLOUDWORKSTATION_DEV=true
```

Then restart your terminal or run `source ~/.zshrc`.

### Method 2: CloudWorkstation Profiles

CloudWorkstation has its own profile system that can reference AWS profiles:

```bash
# Create a CloudWorkstation profile using your 'aws' AWS profile
cws profile create my-research --aws-profile aws --region us-west-2

# Set it as your current profile
cws profile use my-research

# Verify it's working
cws profile current
```

### Method 3: Command-Line Override

You can specify the AWS profile for individual commands:

```bash
# Set environment variable for single session
AWS_PROFILE=aws cws templates list

# Or use CloudWorkstation's profile system
cws --aws-profile aws templates list
```

## 4. Verification and Testing

### Quick Health Check

```bash
# Check daemon can access AWS
cws daemon start
cws doctor

# List available templates (requires AWS access)
cws templates list

# Check your current configuration
cws profile current
aws configure list --profile aws
```

### Test Instance Launch (Optional)

```bash
# Launch a simple test instance
cws launch "Basic Ubuntu (APT)" test-instance

# Check it's running
cws list

# Clean up
cws terminate test-instance
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
aws configure set region us-west-2 --profile aws

# Or set via environment variable
export AWS_REGION=us-west-2
```

## 6. Troubleshooting

### Common Issues

**"No credentials found" error:**
```bash
# Check your profile exists
aws configure list --profile aws

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

**CloudWorkstation can't find your profile:**
```bash
# Check CloudWorkstation sees your AWS profile
cws doctor --verbose

# Create explicit CloudWorkstation profile
cws profile create research --aws-profile aws --region us-west-2
cws profile use research
```

### Debug Mode

Enable verbose logging to see what CloudWorkstation is doing:

```bash
# Set debug environment variables
export AWS_PROFILE=aws
export CLOUDWORKSTATION_DEBUG=true

# Run commands with detailed output
cws templates list
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
cws launch "Python ML" my-project --spot

# Use hibernation for cost savings
cws hibernate my-project
```

### Profile Organization

```bash
# Organize profiles by project/purpose
cws profile create personal-research --aws-profile aws --region us-west-2
cws profile create team-project --aws-profile work --region us-east-1
cws profile create gpu-experiments --aws-profile aws --region us-west-2
```

## 8. Example Complete Setup

Here's a complete example for your specific case:

```bash
# 1. Configure AWS CLI with 'aws' profile
aws configure --profile aws
# Enter your credentials when prompted

# 2. Set environment variables  
export AWS_PROFILE=aws
export AWS_REGION=us-west-2
export CLOUDWORKSTATION_DEV=true

# 3. Add to your shell profile to make permanent
echo 'export AWS_PROFILE=aws' >> ~/.zshrc
echo 'export AWS_REGION=us-west-2' >> ~/.zshrc  
echo 'export CLOUDWORKSTATION_DEV=true' >> ~/.zshrc

# 4. Test CloudWorkstation
cws daemon start
cws templates list
cws doctor

# 5. Launch your first workstation
cws launch "Python Machine Learning (Simplified)" my-research

# 6. Connect and start working
cws connect my-research
```

## Need Help?

- **CloudWorkstation Issues**: `cws doctor --verbose`
- **AWS Issues**: `aws sts get-caller-identity --profile aws`
- **Documentation**: Run `cws --help` for command reference
- **Demo**: Run `./demo.sh` to see CloudWorkstation in action

Your AWS profile 'aws' should now work seamlessly with CloudWorkstation!