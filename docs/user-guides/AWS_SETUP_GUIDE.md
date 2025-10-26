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
#     "Arn": "arn:aws:iam::123456789012:user/prism-user"
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
export CLOUDWORKSTATION_DEV=true
```

Then restart your terminal or run `source ~/.zshrc`.

### Method 3: Command-Line Override

You can specify the AWS profile for individual commands:

```bash
# Set environment variable for single session
AWS_PROFILE=aws prism templates list

# Or use Prism's profile system
prism --aws-profile aws templates list
```

## 4. Verification and Testing

### Quick Health Check

```bash
# Check daemon can access AWS
prism daemon start
prism doctor

# List available templates (requires AWS access)
prism templates list

# Check your current configuration
prism profile current
aws configure list --profile aws
```

### Test Instance Launch (Optional)

```bash
# Launch a simple test instance
prism launch "Basic Ubuntu (APT)" test-instance

# Check it's running
prism list

# Clean up
prism terminate test-instance
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

**Prism can't find your profile:**
```bash
# Check Prism sees your AWS profile
prism doctor --verbose

# Create explicit Prism profile
prism profile create research --aws-profile aws --region us-west-2
prism profile use research
```

### Debug Mode

Enable verbose logging to see what Prism is doing:

```bash
# Set debug environment variables
export AWS_PROFILE=aws
export CLOUDWORKSTATION_DEBUG=true

# Run commands with detailed output
prism templates list
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
prism launch "Python ML" my-project --spot

# Use hibernation for cost savings
prism hibernate my-project
```

### Profile Organization

```bash
# Organize profiles by project/purpose
prism profile create personal-research --aws-profile aws --region us-west-2
prism profile create team-project --aws-profile work --region us-east-1
prism profile create gpu-experiments --aws-profile aws --region us-west-2
```

## 8. Example Complete Setup

Here's a complete example for your specific case using Prism profiles:

```bash
# 1. Configure AWS CLI with 'aws' profile
aws configure --profile aws
# Enter your credentials when prompted

# 2. Create Prism profile (RECOMMENDED METHOD)
prism daemon start
prism profiles add personal my-research --aws-profile aws --region us-west-2
prism profiles switch aws  # Switch to use your 'aws' profile

# 3. Verify configuration
prism profiles current
prism doctor

# 4. Launch your first workstation
prism launch "Python Machine Learning (Simplified)" my-research

# 5. Connect and start working
prism connect my-research
```

### Alternative Setup (Environment Variables)

If you prefer environment variables:

```bash
# 1. Configure AWS CLI with 'aws' profile  
aws configure --profile aws

# 2. Set environment variables
export AWS_PROFILE=aws
export AWS_REGION=us-west-2
export CLOUDWORKSTATION_DEV=true

# 3. Make permanent
echo 'export AWS_PROFILE=aws' >> ~/.zshrc
echo 'export AWS_REGION=us-west-2' >> ~/.zshrc

# 4. Test and launch
prism daemon start
prism templates list
prism launch "Python ML" my-research
```

## Need Help?

- **Prism Issues**: `prism doctor --verbose`
- **AWS Issues**: `aws sts get-caller-identity --profile aws`
- **Documentation**: Run `prism --help` for command reference
- **Demo**: Run `./demo.sh` to see Prism in action

Your AWS profile 'aws' should now work seamlessly with Prism!