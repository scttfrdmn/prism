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

### 🚀 User Guides

- [Quick Start Guide](GETTING_STARTED.md) - Get up and running fast
- [Terminal Interface Guide](TUI_USER_GUIDE.md) - Command-line power users  
- [Desktop Interface Guide](GUI_USER_GUIDE.md) - Point-and-click interface
- [Multiple AWS Accounts](MULTI_PROFILE_GUIDE.md) - Work with different AWS accounts
- [Template System](TEMPLATE_INHERITANCE.md) - Understanding research templates

### 🔧 Administrator Documentation

- [Administrator Guide](ADMINISTRATOR_GUIDE.md) - Managing CloudWorkstation
- [Security & Invitations](SECURE_INVITATION_ARCHITECTURE.md) - User access control
- [Batch Management](BATCH_INVITATION_GUIDE.md) - Managing multiple users
- [Security Hardening](SECURITY_HARDENING_GUIDE.md) - Enterprise security

### 📦 Installation & Distribution

- [Linux Installation](LINUX_INSTALLATION.md) - Linux-specific setup
- [macOS DMG Installation](MACOS_DMG_INSTALLATION.md) - macOS installer
- [Homebrew Tap](HOMEBREW_TAP.md) - macOS/Linux package management
- [Chocolatey Package](CHOCOLATEY_PACKAGE.md) - Windows package management
- [Windows MSI Installer](../packaging/windows/README.md) - Enterprise Windows install

### 🛠️ Advanced Features

- [Template Format](TEMPLATE_FORMAT.md) - Creating custom templates
- [Advanced Templates](TEMPLATE_FORMAT_ADVANCED.md) - Complex template features
- [EFS File Sharing](EFS_SHARING_IMPLEMENTATION.md) - Multi-instance collaboration
- [Cost Optimization](IDLE_DETECTION.md) - Hibernation and idle detection
- [Profile Management](PROFILE_EXPORT_IMPORT.md) - Importing/exporting profiles

### 🏗️ Developer Documentation

- [GUI Architecture](GUI_ARCHITECTURE.md) - Desktop application design
- [Plugin Architecture](PLUGIN_ARCHITECTURE.md) - Extensibility system
- [API Reference](DAEMON_API_REFERENCE.md) - REST API documentation
- [Template Implementation](TEMPLATE_SYSTEM_IMPLEMENTATION.md) - How templates work

### 🔮 Future Planning

- [Phase 5 Development](PHASE_5_DEVELOPMENT_PLAN.md) - Roadmap for v0.5.0
- [Multi-User Planning](MULTI_USER_PLANNING_v0.5.0.md) - Collaborative features
- [Research Architecture](RESEARCH_USER_ARCHITECTURE.md) - Academic integration

### 📚 Archive

Historical documentation and completed implementation details are archived in [docs/archive/](archive/README.md).

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

- [Troubleshooting Guide](TROUBLESHOOTING.md) - Common issues and solutions
- [Report issues on GitHub](https://github.com/scttfrdmn/cloudworkstation/issues)
- [Read the documentation](https://cloudworkstation.io/docs)

## Security

CloudWorkstation takes security seriously:

[![Security Scan](https://img.shields.io/badge/Security%20Scan-Passing-brightgreen)](https://github.com/scttfrdmn/cloudworkstation/actions)
[![Dependency Check](https://img.shields.io/badge/Dependencies-No%20Known%20Vulnerabilities-brightgreen)](https://github.com/scttfrdmn/cloudworkstation/actions)
[![Code Coverage](https://img.shields.io/badge/Code%20Coverage-87%25-brightgreen)](https://github.com/scttfrdmn/cloudworkstation/actions)