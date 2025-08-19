# CloudWorkstation v0.4.3 Installation Guide

## Quick Start (Homebrew - Recommended)

### macOS and Linux
```bash
# Step 1: Add the CloudWorkstation tap
brew tap scttfrdmn/cloudworkstation

# Step 2: Install CloudWorkstation
brew install cloudworkstation

# Step 3: Verify installation
cws --version
cwsd --version
```

## Alternative Installation Methods

### GitHub Releases (Direct Download)
1. Download the appropriate binary for your platform from [GitHub Releases](https://github.com/scttfrdmn/cloudworkstation/releases)
2. Extract the archive
3. Move binaries to your PATH (e.g., `/usr/local/bin/`)

### Source Build (Full Features)
```bash
# Clone the repository
git clone https://github.com/scttfrdmn/cloudworkstation.git
cd cloudworkstation

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
cws daemon start

# Launch your first workstation
cws launch "Python Machine Learning (Simplified)" my-research

# Connect to your workstation
cws connect my-research
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
- **Basic Support**: CLI, daemon
- **TUI/GUI**: Planned for future releases

## Configuration

### Development Mode (Optional)
To avoid keychain password prompts during development:
```bash
export CLOUDWORKSTATION_DEV=true
# Or add to ~/.bashrc or ~/.zshrc
```

### AWS Credentials

CloudWorkstation requires AWS credentials to launch cloud workstations. See the **[AWS Setup Guide](AWS_SETUP_GUIDE.md)** for complete configuration instructions.

**Quick setup:**
```bash
# Configure with your preferred AWS profile name
aws configure --profile aws  # or any name you prefer

# Point CloudWorkstation to your profile
export AWS_PROFILE=aws
export AWS_REGION=us-west-2

# Make permanent by adding to ~/.bashrc or ~/.zshrc
echo 'export AWS_PROFILE=aws' >> ~/.zshrc
```

**Need help?** The [AWS Setup Guide](AWS_SETUP_GUIDE.md) covers:
- AWS account setup and permissions
- Using non-default profiles (like 'aws' instead of 'default')
- Regional configuration
- CloudWorkstation profile management
- Troubleshooting common issues

## Getting Help

- **Documentation**: [https://docs.cloudworkstation.dev](https://docs.cloudworkstation.dev)
- **Issues**: [GitHub Issues](https://github.com/scttfrdmn/cloudworkstation/issues)
- **CLI Help**: `cws --help`
- **Demo**: Run `./demo.sh` in the repository

## Upgrading

### Homebrew
```bash
brew update
brew upgrade cloudworkstation
```

### Manual
Download the latest release and replace the existing binaries.

## Uninstalling

### Homebrew
```bash
brew uninstall cloudworkstation
brew untap scttfrdmn/cloudworkstation
```

### Manual
```bash
# Remove binaries
sudo rm -f /usr/local/bin/cws /usr/local/bin/cwsd /usr/local/bin/cws-gui

# Remove configuration (optional)
rm -rf ~/.cloudworkstation
```

## Troubleshooting

### Common Issues

**Daemon won't start:**
```bash
cws daemon stop
cws daemon start
```

**Keychain password prompts:**
```bash
export CLOUDWORKSTATION_DEV=true
```

**AWS permission errors:**
```bash
aws sts get-caller-identity
cws doctor
```

For more help, see the [Troubleshooting Guide](https://github.com/scttfrdmn/cloudworkstation/blob/main/TROUBLESHOOTING.md).