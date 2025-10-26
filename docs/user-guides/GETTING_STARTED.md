# Getting Started with Prism

## Quick Start (5 minutes)

Prism provides pre-configured research environments without complex setup requirements.

### 1. Installation

See the main [Installation Guide](../index.md#installation) for detailed installation instructions for your platform (macOS, Linux, Windows, Conda).

Quick install:
```bash
# macOS/Linux
brew tap scttfrdmn/prism
brew install prism
```

```powershell
# Windows
scoop bucket add scttfrdmn https://github.com/scttfrdmn/scoop-bucket
scoop install prism
```

### 2. AWS Setup

Prism uses your existing AWS credentials. If you don't have AWS CLI configured:

```bash
aws configure
# Enter your AWS Access Key ID, Secret Access Key, and default region
```

For detailed AWS setup including IAM permissions, see the [Administrator Guide](../admin-guides/ADMINISTRATOR_GUIDE.md) or [AWS IAM Permissions](../admin-guides/AWS_IAM_PERMISSIONS.md).

### 3. Launch Your First Environment
```bash
# See available templates
prism templates

# Launch a Python ML environment
prism launch python-ml my-first-project

# Get connection info
prism connect my-first-project
```

That's it! Your research environment is ready.

---

## Choose Your Interface

Prism offers three ways to interact:

### 🖥️ **GUI (Desktop App)**
Perfect for visual management and one-click operations.
```bash
cws-gui
```

### 📱 **TUI (Terminal Interface)**  
Keyboard-driven interface for remote work and SSH sessions.
```bash
prism tui
```

### 💻 **CLI (Command Line)**
Scriptable interface for automation and power users.
```bash
prism launch python-ml my-project --size L
```

---

## Essential Commands

### Template Management
```bash
prism templates                    # List available environments
prism templates info python-ml    # Get template details
prism launch python-ml my-project # Launch environment
```

### Instance Management
```bash
prism list                        # Show running instances
prism connect my-project          # Get connection info
prism stop my-project             # Stop when not in use
prism start my-project            # Resume later
prism delete my-project           # Remove completely
```

### Cost Optimization
```bash
prism hibernate my-project        # Preserve RAM, reduce costs
prism resume my-project           # Resume hibernated instance
prism idle enable                 # Auto-hibernate idle instances
```

---

## Common Research Workflows

### Data Science Project
```bash
# Launch Jupyter environment
prism launch python-ml data-analysis --size L

# Create shared storage
prism volume create shared-datasets

# Connect and start working
prism connect data-analysis
# Opens: ssh user@ip-address -L 8888:localhost:8888
# Jupyter: http://localhost:8888
```

### R Statistical Analysis
```bash
# Launch R + RStudio environment
prism launch r-research stats-project

# Get RStudio connection
prism connect stats-project
# Opens: http://ip-address:8787 (RStudio Server)
```

### Custom Environment
```bash
# Start with base template
prism launch basic-ubuntu my-custom

# Customize your setup
prism connect my-custom
# Install packages, configure tools

# Save for reuse
prism save my-custom custom-template
```

---

## Troubleshooting

### "Daemon not running"
```bash
# Check daemon status
prism daemon status

# Restart daemon if troubleshooting (rarely needed - daemon auto-starts)
prism daemon stop
# Next command will auto-start fresh daemon
prism templates
```

### "AWS credentials not found"
```bash
# Verify AWS configuration
aws sts get-caller-identity

# Reconfigure if needed
aws configure
```

### "Permission denied" errors
Make sure your AWS user has the required permissions. See our [AWS IAM Permissions](../admin-guides/AWS_IAM_PERMISSIONS.md) for complete IAM policies, or run:

```bash
./scripts/setup-iam-permissions.sh
```

### Instance launch fails
```bash
# Check AWS region and availability
aws ec2 describe-availability-zones

# Try different region
prism launch python-ml my-project --region us-east-1
```

---

## Next Steps

- **Browse Templates**: Explore research environments with `prism templates`
- **Join Community**: Share templates and get help
- **Read Guides**: Detailed documentation in `/docs` folder
- **Cost Optimization**: Learn about hibernation and spot instances
- **Team Collaboration**: Set up shared storage and project management

**Need Help?** Open an issue on [GitHub](https://github.com/scttfrdmn/prism/issues) or check our documentation.

---

## Advanced Features

### Template Stacking
```bash
# Build on existing templates
prism apply gpu-drivers my-project    # Add GPU support
prism apply docker-tools my-project   # Add Docker
```

### Project Management
```bash
# Create research project
prism project create brain-study --budget 500

# Launch in project context
prism launch neuroimaging analysis --project brain-study
```

### Custom AMIs
```bash
# Build optimized AMI from template
prism ami build python-ml --region us-west-2

# Save running instance as template
prism ami save my-project custom-env
```

**🎯 Key Principle**: Prism defaults to success. Most commands work without options, with smart defaults for research computing.