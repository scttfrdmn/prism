# Prism Distribution Guide

![Prism Logo](images/prism.png)

This document outlines the different ways to install Prism on your system.

## Installation Methods

Prism can be installed using multiple methods depending on your platform and preferences:

### Direct Download (All Platforms)

Download the latest binaries directly from our GitHub releases page:

```bash
# Example for Linux x86_64
curl -L https://github.com/yourusername/prism/releases/download/v0.4.1/cws-linux-amd64 -o cws
chmod +x cws
sudo mv prism /usr/local/bin/
```

Available binaries:
- `cws-linux-amd64` - Linux Intel/AMD
- `cws-linux-arm64` - Linux ARM64
- `cws-macos-amd64` - macOS Intel
- `cws-macos-arm64` - macOS Apple Silicon
- `cws-windows-amd64.exe` - Windows Intel/AMD

### Homebrew (macOS and Linux)

For macOS and Linux users with Homebrew installed:

```bash
# Add our tap (only needed the first time)
brew tap yourusername/prism

# Install Prism
brew install prism
```

This automatically installs the correct binary for your architecture (Intel or ARM).

### Chocolatey (Windows)

For Windows users with Chocolatey installed:

```powershell
# Install Prism
choco install prism
```

This adds Prism to your PATH and creates desktop shortcuts.

### Conda (All Platforms)

For researchers already using the Conda package manager:

```bash
# Install from conda-forge channel
conda install prism -c conda-forge
```

This is particularly useful for scientific computing environments where Conda is commonly used.

## Verifying Your Installation

To verify Prism is correctly installed, run:

```bash
prism version
```

You should see output indicating the installed version, for example:
```
Prism v0.4.1
```

## Setting Up AWS Credentials

Prism requires AWS credentials to function. If you haven't already configured your AWS credentials:

1. Create an AWS account if you don't have one
2. Create an IAM user with appropriate permissions
3. Configure credentials using one of these methods:

```bash
# Option 1: AWS CLI
aws configure

# Option 2: Environment variables
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_DEFAULT_REGION=us-west-2

# Option 3: Prism config
prism config profile my-profile
prism config region us-west-2
```

## Updating Prism

To update to the latest version:

### Direct Download
Download the latest binary and replace your existing installation.

### Homebrew
```bash
brew upgrade prism
```

### Chocolatey
```bash
choco upgrade prism
```

### Conda
```bash
conda update prism -c conda-forge
```

## Troubleshooting Installation Issues

### Common Issues

1. **Permission denied error**:
   ```
   -bash: /usr/local/bin/cws: Permission denied
   ```
   Fix: `chmod +x /usr/local/bin/cws`

2. **Command not found**:
   ```
   cws: command not found
   ```
   Fix: Ensure the installation directory is in your PATH

3. **Dependency issues**:
   If dependencies are missing, install them based on your distribution:
   ```
   # Ubuntu/Debian
   apt-get install libssl-dev

   # Red Hat/Fedora
   dnf install openssl-devel
   ```

4. **Architecture mismatch**:
   Make sure you're using the correct binary for your system's architecture.

### Getting Help

If you encounter persistent installation issues:

1. Visit our GitHub issues page
2. Check the detailed installation guides in the documentation
3. Contact support with details about your system and the issue you're experiencing