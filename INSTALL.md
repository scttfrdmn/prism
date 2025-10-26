# Prism v0.5.4 Installation Guide

## Quick Start (Package Managers - Recommended)

### macOS and Linux (Homebrew)
```bash
# Step 1: Install Prism
brew install scttfrdmn/tap/prism

# Step 2: Verify installation
prism --version
cwsd --version
```

### Windows (Scoop)
```powershell
# Step 1: Add the Prism bucket
scoop bucket add scttfrdmn https://github.com/scttfrdmn/scoop-bucket

# Step 2: Install Prism
scoop install prism

# Step 3: Verify installation
prism --version
cwsd --version
```

## Alternative Installation Methods

### GitHub Releases (Direct Download)
1. Download the appropriate binary for your platform from [GitHub Releases](https://github.com/scttfrdmn/prism/releases)
2. Extract the archive
3. Move binaries to your PATH (e.g., `/usr/local/bin/`)

### Source Build (Full Features)
```bash
# Clone the repository
git clone https://github.com/scttfrdmn/prism.git
cd prism

# Set development mode (avoids keychain prompts during build)
cp .env.example .env

# Build all components (includes GUI)
make build

# Optional: Install to system
make install
```

## First Launch

```bash
# Start the daemon
prism daemon start

# Launch your first workstation
prism launch "Python Machine Learning (Simplified)" my-research

# Connect to your workstation
prism connect my-research
```

## Platform-Specific Notes

### macOS
- **Full Support**: CLI, TUI, daemon, GUI (when built from source)
- **Native Keychain**: Automatic integration with macOS Keychain
- **Architecture**: Both Intel (x86_64) and Apple Silicon (arm64) supported

### Linux
- **Core Features**: CLI, TUI, daemon
- **GUI**: Available when built from source or via Docker
- **Architecture**: Both x86_64 and arm64 supported

### Windows
- **Full Support**: CLI, TUI, daemon
- **GUI**: Available when built from source or via Docker
- **Installation**: Use Scoop package manager for easy installation

## Configuration

### Development Mode (Optional)
To avoid keychain password prompts during development:
```bash
export CLOUDWORKSTATION_DEV=true
# Or add to ~/.bashrc or ~/.zshrc
```

### AWS Credentials

Prism requires AWS credentials to launch cloud workstations. See the **[AWS Setup Guide](AWS_SETUP_GUIDE.md)** for complete configuration instructions.

**Quick setup:**
```bash
# Configure with your preferred AWS profile name
aws configure --profile aws  # or any name you prefer

# Point Prism to your profile
export AWS_PROFILE=aws
export AWS_REGION=us-west-2

# Make permanent by adding to ~/.bashrc or ~/.zshrc
echo 'export AWS_PROFILE=aws' >> ~/.zshrc
```

**Need help?** The [AWS Setup Guide](AWS_SETUP_GUIDE.md) covers:
- AWS account setup and permissions
- Using non-default profiles (like 'aws' instead of 'default')
- Regional configuration
- Prism profile management
- Troubleshooting common issues

## Getting Help

- **Documentation**: [https://docs.prism.dev](https://docs.prism.dev)
- **Issues**: [GitHub Issues](https://github.com/scttfrdmn/prism/issues)
- **CLI Help**: `prism --help`
- **Demo**: Run `./demo.sh` in the repository

## Upgrading

### Homebrew (macOS/Linux)
```bash
brew update
brew upgrade scttfrdmn/tap/prism
```

### Scoop (Windows)
```powershell
scoop update
scoop update prism
```

### Manual
Download the latest release and replace the existing binaries.

## Uninstalling

### Homebrew (macOS/Linux)
```bash
brew uninstall scttfrdmn/tap/prism
brew untap scttfrdmn/tap
```

### Scoop (Windows)
```powershell
scoop uninstall prism
scoop bucket rm scttfrdmn
```

### Manual
```bash
# Remove binaries
sudo rm -f /usr/local/bin/cws /usr/local/bin/cwsd /usr/local/bin/cws-gui

# Remove configuration (optional)
rm -rf ~/.prism
```

## Troubleshooting

### Common Issues

**Daemon won't start:**
```bash
prism daemon stop
prism daemon start
```

**Keychain password prompts:**
```bash
export CLOUDWORKSTATION_DEV=true
```

**AWS permission errors:**
```bash
aws sts get-caller-identity
prism doctor
```

For more help, see the [Troubleshooting Guide](https://github.com/scttfrdmn/prism/blob/main/TROUBLESHOOTING.md).