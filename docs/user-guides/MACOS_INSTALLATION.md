# macOS Installation Guide

Prism can be installed on macOS via Homebrew for a streamlined command-line and GUI experience.

## Quick Start

```bash
# Install via Homebrew
brew tap scttfrdmn/prism
brew install prism

# Verify installation
prism version
```

## Installation Methods

### Method 1: Homebrew (Recommended)

**Best for:** Most macOS users, provides automatic updates and easy management.

```bash
# Add Prism tap
brew tap scttfrdmn/prism

# Install Prism
brew install prism

# Verify installation
prism version
cwsd version
```

**Includes:**
- `cws` CLI tool
- `cwsd` daemon
- `prism-gui` desktop application (if GUI support is available)
- Automatic PATH configuration
- Easy updates via `brew upgrade`

### Method 2: Direct Binary Download

**Best for:** Users who prefer manual installation or don't use Homebrew.

```bash
# Intel Macs
curl -L https://github.com/scttfrdmn/prism/releases/latest/download/prism-darwin-amd64.tar.gz | tar xz

# Apple Silicon Macs
curl -L https://github.com/scttfrdmn/prism/releases/latest/download/prism-darwin-arm64.tar.gz | tar xz

# Move binaries to PATH
sudo mv prism cwsd /usr/local/bin/
```

## Post-Installation Setup

### 1. Configure AWS Credentials

Prism requires AWS credentials to manage cloud resources:

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
# Check Prism version
prism version

# List available templates
prism templates

# Check daemon status
prism daemon status
```

## Using Prism on macOS

### Launch GUI Application

```bash
# Start the desktop application
cws-gui
```

### Launch Terminal Interface

```bash
# Start the interactive terminal UI
prism tui
```

### Use Command Line

```bash
# See all available commands
prism --help

# Launch your first environment
prism launch python-ml my-first-project
```

## Updating Prism

```bash
# Update via Homebrew
brew update
brew upgrade prism

# Verify new version
prism version
```

## Troubleshooting

### "Command not found: cws"

```bash
# Check if Prism is installed
brew list prism

# Reinstall if needed
brew reinstall prism

# Verify PATH includes Homebrew bin
echo $PATH | grep homebrew
```

### Permission Issues

```bash
# Ensure proper permissions on binaries
ls -la $(which cws)
ls -la $(which cwsd)

# Fix permissions if needed
brew reinstall prism
```

### Daemon Connection Issues

```bash
# Check daemon status
prism daemon status

# Restart daemon if needed
prism daemon stop
# Next command will auto-start daemon
prism templates
```

## Uninstalling

```bash
# Remove Prism via Homebrew
brew uninstall prism
brew untap scttfrdmn/prism

# Remove configuration (optional)
rm -rf ~/.prism
```

## Next Steps

- See [Getting Started Guide](GETTING_STARTED.md) for first-time usage
- Read [User Guide v0.5.x](USER_GUIDE_v0.5.x.md) for complete CLI reference
- Explore [Template Format](TEMPLATE_FORMAT.md) to create custom environments
- Check [Troubleshooting](TROUBLESHOOTING.md) for common issues
