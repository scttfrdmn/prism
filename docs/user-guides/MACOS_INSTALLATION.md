# macOS Installation Guide

CloudWorkstation can be installed on macOS via Homebrew for a streamlined command-line and GUI experience.

## Quick Start

```bash
# Install via Homebrew
brew tap scttfrdmn/cloudworkstation
brew install cloudworkstation

# Verify installation
cws version
```

## Installation Methods

### Method 1: Homebrew (Recommended)

**Best for:** Most macOS users, provides automatic updates and easy management.

```bash
# Add CloudWorkstation tap
brew tap scttfrdmn/cloudworkstation

# Install CloudWorkstation
brew install cloudworkstation

# Verify installation
cws version
cwsd version
```

**Includes:**
- `cws` CLI tool
- `cwsd` daemon
- `cws-gui` desktop application (if GUI support is available)
- Automatic PATH configuration
- Easy updates via `brew upgrade`

### Method 2: Direct Binary Download

**Best for:** Users who prefer manual installation or don't use Homebrew.

```bash
# Intel Macs
curl -L https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cloudworkstation-darwin-amd64.tar.gz | tar xz

# Apple Silicon Macs
curl -L https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cloudworkstation-darwin-arm64.tar.gz | tar xz

# Move binaries to PATH
sudo mv cws cwsd /usr/local/bin/
```

## Post-Installation Setup

### 1. Configure AWS Credentials

CloudWorkstation requires AWS credentials to manage cloud resources:

```bash
# Install AWS CLI if needed
brew install awscli

# Configure your AWS credentials
aws configure
# Enter your AWS Access Key ID
# Enter your AWS Secret Access Key
# Enter your default region (e.g., us-west-2)
# Enter output format (json)
```

### 2. Verify Installation

```bash
# Check CloudWorkstation version
cws version

# List available templates
cws templates

# Check daemon status
cws daemon status
```

## Using CloudWorkstation on macOS

### Launch GUI Application

```bash
# Start the desktop application
cws-gui
```

### Launch Terminal Interface

```bash
# Start the interactive terminal UI
cws tui
```

### Use Command Line

```bash
# See all available commands
cws --help

# Launch your first environment
cws launch python-ml my-first-project
```

## Updating CloudWorkstation

```bash
# Update via Homebrew
brew update
brew upgrade cloudworkstation

# Verify new version
cws version
```

## Troubleshooting

### "Command not found: cws"

```bash
# Check if CloudWorkstation is installed
brew list cloudworkstation

# Reinstall if needed
brew reinstall cloudworkstation

# Verify PATH includes Homebrew bin
echo $PATH | grep homebrew
```

### Permission Issues

```bash
# Ensure proper permissions on binaries
ls -la $(which cws)
ls -la $(which cwsd)

# Fix permissions if needed
brew reinstall cloudworkstation
```

### Daemon Connection Issues

```bash
# Check daemon status
cws daemon status

# Restart daemon if needed
cws daemon stop
# Next command will auto-start daemon
cws templates
```

## Uninstalling

```bash
# Remove CloudWorkstation via Homebrew
brew uninstall cloudworkstation
brew untap scttfrdmn/cloudworkstation

# Remove configuration (optional)
rm -rf ~/.cloudworkstation
```

## Next Steps

- See [Getting Started Guide](GETTING_STARTED.md) for first-time usage
- Read [User Guide v0.5.x](USER_GUIDE_v0.5.x.md) for complete CLI reference
- Explore [Template Format](TEMPLATE_FORMAT.md) to create custom environments
- Check [Troubleshooting](TROUBLESHOOTING.md) for common issues
