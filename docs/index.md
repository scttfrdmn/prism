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

```powershell
# Using Scoop
scoop bucket add scttfrdmn https://github.com/scttfrdmn/scoop-bucket
scoop install cloudworkstation
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

### 👥 Persona Walkthroughs (Start Here!)

**🎯 These walkthroughs are our north star for feature development and product direction.**

They prioritize usability and clarity by showing complete end-to-end workflows with real commands, expected outputs, and best practices. When we add features or make design decisions, we validate them against these scenarios to ensure CloudWorkstation remains focused on real researcher needs.

**User Scenarios:**

- [Solo Researcher Walkthrough](USER_SCENARIOS/01_SOLO_RESEARCHER_WALKTHROUGH.md) - Individual research projects
- [Lab Environment Walkthrough](USER_SCENARIOS/02_LAB_ENVIRONMENT_WALKTHROUGH.md) - Team collaboration
- [University Class Walkthrough](USER_SCENARIOS/03_UNIVERSITY_CLASS_WALKTHROUGH.md) - Teaching and coursework
- [Conference Workshop Walkthrough](USER_SCENARIOS/04_CONFERENCE_WORKSHOP_WALKTHROUGH.md) - Workshops and tutorials
- [Cross-Institutional Collaboration](USER_SCENARIOS/05_CROSS_INSTITUTIONAL_COLLABORATION_WALKTHROUGH.md) - Multi-institution projects

💡 **For Contributors**: Before implementing a feature, check if it improves one of these workflows. If it doesn't clearly benefit a persona scenario, it may not be the right priority.

### 🚀 User Guides

- [Getting Started](user-guides/ZERO_SETUP_GUIDE.md) - Quickest path to first launch
- [User Guide v0.5.x](user-guides/USER_GUIDE_v0.5.x.md) - Complete CLI guide
- [Desktop Interface Guide](user-guides/GUI_USER_GUIDE.md) - Point-and-click interface
- [Terminal Interface Guide](user-guides/TUI_USER_GUIDE.md) - Command-line power users
- [Linux Installation](user-guides/LINUX_INSTALLATION.md) - Linux setup
- [macOS Installation](user-guides/MACOS_INSTALLATION.md) - macOS setup via Homebrew
- [Multiple AWS Accounts](user-guides/MULTI_PROFILE_GUIDE.md) - Profile management
- [Template Format](user-guides/TEMPLATE_FORMAT.md) - Creating templates
- [Template Marketplace](user-guides/TEMPLATE_MARKETPLACE_USER_GUIDE.md) - Community templates
- [Web Services](user-guides/WEB_SERVICES_INTEGRATION_GUIDE.md) - Jupyter, RStudio
- [Troubleshooting](user-guides/TROUBLESHOOTING.md) - Common issues

### 🔧 Administrator Documentation

- [Administrator Guide](admin-guides/ADMINISTRATOR_GUIDE.md) - System administration
- [Batch Management](admin-guides/BATCH_INVITATION_GUIDE.md) - Multiple users
- [Research User Management](admin-guides/RESEARCH_USER_MANAGEMENT_GUIDE.md) - Multi-user
- [Security Hardening](admin-guides/SECURITY_HARDENING_GUIDE.md) - Enterprise security
- [NIST 800-171 Compliance](admin-guides/NIST_800_171_COMPLIANCE.md) - Compliance
- [AWS IAM Permissions](admin-guides/AWS_IAM_PERMISSIONS.md) - Required permissions
- [Policy Examples](admin-guides/BASIC_POLICY_EXAMPLES.md) - Policy configuration

### 🏗️ Architecture & Design

- [Vision](VISION.md) - Project vision and goals
- [Design Principles](DESIGN_PRINCIPLES.md) - Core philosophy
- [User Requirements](USER_REQUIREMENTS.md) - User scenarios
- [GUI Architecture](architecture/GUI_ARCHITECTURE.md) - Desktop app design
- [API Reference](architecture/DAEMON_API_REFERENCE.md) - REST API docs
- [Dual User Architecture](architecture/DUAL_USER_ARCHITECTURE.md) - User system
- [Template Marketplace](architecture/TEMPLATE_MARKETPLACE_ARCHITECTURE.md) - Distribution
- [Auto-AMI System](architecture/AUTO_AMI_SYSTEM.md) - AMI automation
- [Idle Detection](architecture/IDLE_DETECTION.md) - Cost optimization

### 💻 Development

- [Development Setup](development/DEVELOPMENT_SETUP.md) - Dev environment
- [Testing Guide](development/TESTING.md) - Running tests
- [Code Quality](development/CODE_QUALITY_BEST_PRACTICES.md) - Best practices
- [Release Process](development/RELEASE_PROCESS.md) - Creating releases
- [Distribution](development/DISTRIBUTION.md) - Package distribution
- [Template Implementation](development/TEMPLATE_SYSTEM_IMPLEMENTATION.md) - How templates work

### 📋 Releases

- [Release Notes](releases/RELEASE_NOTES.md) - All versions
- [v0.5.2](releases/RELEASE_NOTES_v0.5.2.md) - Template Marketplace
- [v0.5.1](releases/RELEASE_NOTES_v0.5.1.md) - Command updates

### 📚 Archive

Historical documentation, session summaries, and obsolete plans are archived in [docs/archive/](archive/README.md).

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