# Welcome to CloudWorkstation

<p align="center">
  <img src="images/cloudworkstation.png" alt="CloudWorkstation Logo" width="200">
</p>

CloudWorkstation helps researchers launch cloud computers with just a few clicks. No more spending hours setting up research tools - we've done the hard work for you!

## Getting Started

1. [Install CloudWorkstation](#installation)
2. Choose a research environment template
3. Give your project a name
4. Click "Launch"
5. Start working in seconds!

## Installation

You can install CloudWorkstation in different ways:

### macOS

```bash
# Using Homebrew
brew tap scttfrdmn/cloudworkstation
brew install cloudworkstation
```

### Windows

```bash
# Using Chocolatey
choco install cloudworkstation --source="'https://package.cloudworkstation.org/chocolatey'"
```

### Linux

```bash
# Using Homebrew on Linux
brew tap scttfrdmn/cloudworkstation
brew install cloudworkstation
```

### Using Conda (Any Platform)

```bash
# Add our channel
conda config --add channels scttfrdmn
conda install cloudworkstation
```

### Direct Download

You can also download the right version for your computer:

- [macOS Intel (x86_64)](https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cloudworkstation-darwin-amd64.tar.gz)
- [macOS Apple Silicon (M1/M2)](https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cloudworkstation-darwin-arm64.tar.gz)
- [Windows](https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cloudworkstation-windows-amd64.zip)
- [Linux](https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cloudworkstation-linux-amd64.tar.gz)

## Documentation Index

### User Guides

- [Quick Start Guide](GETTING_STARTED.md)
- [Terminal User Interface Guide](TUI_USER_GUIDE.md)
- [Graphical User Interface Guide](GUI_USER_GUIDE.md)
- [Multiple AWS Accounts Guide](MULTI_PROFILE_GUIDE.md)
- [Research Templates Guide](TEMPLATE_FORMAT.md)

### Administrator Documentation
- [Administrator Guide](ADMINISTRATOR_GUIDE.md)
- [Batch Administration](ADMINISTRATOR_GUIDE_BATCH.md)
- [Security and Invitations](SECURE_INVITATION_ARCHITECTURE.md)

### Feature Documentation
- [Templates](TEMPLATE_FORMAT.md) - Template creation guide
- [Advanced Templates](TEMPLATE_FORMAT_ADVANCED.md) - Advanced template features  
- [Template Inheritance](TEMPLATE_INHERITANCE.md) - Template inheritance system
- [Repositories](REPOSITORIES.md) - Repository management
- [Idle Detection](IDLE_DETECTION.md) - Hibernation and cost optimization
- [Profile Export/Import](PROFILE_EXPORT_IMPORT.md) - Profile operations

### Developer Documentation
- [GUI Architecture](GUI_ARCHITECTURE.md) - GUI technical architecture
- [API Authentication](API_AUTHENTICATION.md) - API security documentation
- [Daemon API Integration](DAEMON_API_INTEGRATION.md) - API integration guide
- [Testing Infrastructure](TESTING_INFRASTRUCTURE.md) - Testing framework
- [Template System Implementation](TEMPLATE_SYSTEM_IMPLEMENTATION.md) - Implementation details

### Distribution & Packaging
- [Distribution Guide](DISTRIBUTION.md) - Package distribution
- [Homebrew Tap](HOMEBREW_TAP.md) - macOS distribution
- [Chocolatey Package](CHOCOLATEY_PACKAGE.md) - Windows distribution
- [Conda Package](CONDA_PACKAGE.md) - Cross-platform distribution

### Implementation Plans  
- [Implementation Plan v0.4.2](IMPLEMENTATION_PLAN_V0.4.2.md) - Current development plan

## Features

CloudWorkstation lets you:

- **Launch research environments** with common tools pre-installed
- **Save money** by automatically choosing the right computer size
- **Access your work** from anywhere with internet
- **Share files** between different cloud computers
- **Monitor costs** to avoid surprise bills
- **Use multiple AWS accounts** for different projects or classes

## Get Help

If you need help:

- [Read our documentation](https://docs.cloudworkstation.org)
- [Join our community forum](https://community.cloudworkstation.org)
- [Report issues on GitHub](https://github.com/scttfrdmn/cloudworkstation/issues)

## Security

CloudWorkstation takes security seriously:

[![Security Scan](https://img.shields.io/badge/Security%20Scan-Passing-brightgreen)](https://github.com/scttfrdmn/cloudworkstation/actions)
[![Dependency Check](https://img.shields.io/badge/Dependencies-No%20Known%20Vulnerabilities-brightgreen)](https://github.com/scttfrdmn/cloudworkstation/actions)
[![Code Coverage](https://img.shields.io/badge/Code%20Coverage-87%25-brightgreen)](https://github.com/scttfrdmn/cloudworkstation/actions)