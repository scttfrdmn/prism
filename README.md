# CloudWorkstation

<p align="center">
  <img src="docs/images/cloudworkstation.png" alt="CloudWorkstation Logo" width="200">
</p>

<p align="center"><strong>Academic Research Computing Platform - Pre-configured cloud environments for researchers</strong></p>

<p align="center">
  <a href="https://github.com/scttfrdmn/cloudworkstation/releases/latest">
    <img alt="GitHub release (latest by date)" src="https://img.shields.io/github/v/release/scttfrdmn/cloudworkstation">
  </a>
  <a href="https://github.com/scttfrdmn/cloudworkstation/blob/main/LICENSE">
    <img alt="License" src="https://img.shields.io/github/license/scttfrdmn/cloudworkstation">
  </a>
  <a href="https://goreportcard.com/report/github.com/scttfrdmn/cloudworkstation">
    <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/scttfrdmn/cloudworkstation">
  </a>
  <a href="https://github.com/scttfrdmn/cloudworkstation/security/policy">
    <img alt="Security Policy" src="https://img.shields.io/badge/security-policy-brightgreen">
  </a>
</p>

## What is CloudWorkstation?

CloudWorkstation is an **academic research computing platform** that provides researchers with pre-configured cloud environments for data analysis, machine learning, and computational research.

From individual researchers to institutional deployments, CloudWorkstation scales seamlessly while maintaining the simplicity that makes research computing accessible to everyone.

## 🎯 Core Design Principles

- **🎯 Default to Success**: Every template works out of the box in every supported region
- **⚡ Optimize by Default**: Smart instance sizing and cost-performance optimization  
- **🔍 Transparent Fallbacks**: Clear communication when configurations change
- **💡 Helpful Warnings**: Gentle guidance for optimal choices
- **🚫 Zero Surprises**: Users always know what they're getting
- **📈 Progressive Disclosure**: Simple by default, detailed when needed

## 🚀 Quick Start - Zero Setup Experience

### ✨ The CloudWorkstation Promise

**Zero setup, maximum productivity.** CloudWorkstation automatically handles:
- 🚀 **Daemon auto-start**: No manual service management needed
- 🔑 **Smart credential discovery**: Finds your AWS credentials automatically
- 🌍 **Intelligent region selection**: Uses your default AWS region
- 📦 **Template validation**: Ensures all templates work in your region
- 🔧 **Automatic fallbacks**: Seamlessly handles unavailable resources

### macOS Installation

**🍎 DMG Installer (Recommended for Desktop Users)**

```bash
# Download and install via DMG
curl -L -O https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/CloudWorkstation-v0.4.5.dmg
open CloudWorkstation-v0.4.5.dmg
# Drag CloudWorkstation.app to Applications folder
```

**Features:** Native macOS app, GUI + CLI, automatic PATH setup, daemon auto-start
**Guide:** [macOS DMG Installation Guide](docs/MACOS_DMG_INSTALLATION.md)

**📦 Homebrew (Traditional)**

```bash
# Add the CloudWorkstation tap
brew tap scttfrdmn/cloudworkstation

# Install CloudWorkstation  
brew install cloudworkstation

# Verify installation
cws --version
```

### Linux Enterprise Distributions

**🐧 Native Package Installation (Recommended for Servers)**

**Ubuntu/Debian:**
```bash
# Download and install DEB package
wget https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cloudworkstation_0.4.5-1_amd64.deb
sudo dpkg -i cloudworkstation_0.4.5-1_amd64.deb
sudo apt-get install -f  # Fix any dependency issues

# Start service
sudo systemctl enable --now cloudworkstation
```

**RHEL/CentOS/Fedora:**
```bash
# Download and install RPM package  
wget https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cloudworkstation-0.4.5-1.x86_64.rpm
sudo dnf install cloudworkstation-0.4.5-1.x86_64.rpm

# Start service
sudo systemctl enable --now cloudworkstation
```

**Features:** Native systemd service, automatic startup, enterprise-grade security, comprehensive logging
**Guide:** [Linux Installation Guide](docs/LINUX_INSTALLATION.md)

### Linux/Other Platforms (Manual)

```bash
# Download binary for your platform
curl -L https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cloudworkstation-linux-$(uname -m).tar.gz | tar xz

# Make executable and add to PATH
chmod +x cws cwsd
sudo mv cws cwsd /usr/local/bin/
```

### AWS Credentials - Automatic Discovery

CloudWorkstation automatically discovers your AWS credentials in this order:

1. **Environment variables** (AWS_PROFILE, AWS_ACCESS_KEY_ID, etc.)
2. **AWS CLI configuration** (~/.aws/credentials)
3. **CloudWorkstation profiles** (for multi-account management)

```bash
# Option 1: Already have AWS CLI? You're done!
# CloudWorkstation will use your existing credentials

# Option 2: Need to set up AWS?
aws configure  # Follow prompts for access key and region

# Option 3: Multiple AWS accounts? Use profiles
cws profiles add research --aws-profile research-account --region us-west-2
cws profiles switch research
```

**→ Advanced setup options:** [AWS_SETUP_GUIDE.md](AWS_SETUP_GUIDE.md)

### Your First Workstation - Zero Configuration Required

```bash
# Launch a Python ML workstation - that's it!
cws launch "Python Machine Learning (Simplified)" my-research

# What happens automatically:
# ✅ Daemon starts if not running
# ✅ AWS credentials discovered from ~/.aws/credentials
# ✅ Default region selected from AWS config
# ✅ Optimal instance type chosen for ML workload
# ✅ Template validated for your region
# ✅ Security groups and networking configured
# ✅ SSH keys generated and managed

# Connect to your workstation
cws connect my-research

# When done, use idle policy to save costs (preserves state)
cws idle policy apply my-research balanced
```

**That's it!** No configuration files, no setup scripts, no manual steps.

## 🌟 Key Features (v0.4.5)

### 🏢 Enterprise Research Management
- **Project-Based Organization**: Complete project lifecycle management with role-based access control
- **Advanced Budget Management**: Project-specific budgets with real-time tracking and automated controls
- **Cost Analytics**: Detailed cost breakdowns, hibernation savings, and resource utilization metrics
- **Multi-User Collaboration**: Project member management with granular permissions
- **Enterprise API**: Full REST API for project management, budget monitoring, and cost analysis

### 💰 Intelligent Cost Optimization
- **Complete Idle Policy Ecosystem**: Manual controls + automated idle detection policies
- **Session Preservation**: Full work environment state maintained through idle actions (hibernate/stop)
- **Smart Policies**: Domain-specific idle profiles (batch jobs, GPU instances, cost-optimized)
- **Cost Transparency**: Clear audit trail of idle actions and cost savings

### 🏗️ Template System with Inheritance
- **Stackable Templates**: Build complex environments through template composition
- **Smart Inheritance**: Intelligent merging of packages, users, services, and configurations
- **Battle-Tested Defaults**: Templates include optimized configurations for their use cases
- **Regional Fallbacks**: Transparent handling of regional/architecture limitations

### 📱 Multi-Modal Access
- **CLI**: Power users, automation, scripting - maximum efficiency
- **TUI**: Interactive terminal interface with keyboard-first navigation  
- **GUI**: Desktop application with system tray integration (when built from source)
- **REST API**: Complete HTTP API on port 8947 for integrations

## 📦 Available Templates

### Core Research Environments
| Template | Description | Use Cases |
|----------|-------------|-----------|
| **Python Machine Learning (Simplified)** | Python + Jupyter + ML packages | Data science, AI research |
| **R Research Environment (Simplified)** | R + RStudio + tidyverse | Statistical analysis, bioinformatics |
| **Rocky Linux 9 + Conda Stack** | Enterprise Linux + data science stack | HPC environments, institutional research |
| **AWS Deep Learning AMI** | Pre-built GPU-optimized ML environment | Deep learning, neural networks |

### Enterprise & Development
| Template | Description | Use Cases |
|----------|-------------|-----------|
| **Enterprise Server (DNF)** | Full enterprise stack with Docker | Server development, DevOps |
| **Web Development (APT)** | Node.js + Docker + development tools | Full-stack development |
| **Basic Ubuntu (APT)** | Minimal development environment | General computing, testing |

### Template Inheritance Example
```bash
# Base template provides foundation
"Rocky Linux 9 Base" → System tools + rocky user

# Stacked template inherits and extends
"Rocky Linux 9 + Conda Stack" → Inherits base + adds conda + datascientist user + jupyter

# Result: Combined environment with both users, all packages, merged services
cws launch "Rocky Linux 9 + Conda Stack" my-analysis
```

## 🎛️ Interface Options

### Command Line Interface (CLI)
```bash
cws launch "Python Machine Learning (Simplified)" ml-project --size L
cws list                    # Show all instances
cws hibernate ml-project    # Save costs while preserving state
cws resume ml-project       # Resume when needed
cws project create brain-study --budget 1000  # Enterprise features
```

### Terminal User Interface (TUI)
```bash
cws tui
# Navigate: 1=Dashboard, 2=Instances, 3=Templates, 4=Storage, 5=Settings
```

### Desktop GUI (when built from source)
```bash
cws-gui  # Launch desktop application with system tray
```

### REST API Integration
```bash
# Templates
curl http://localhost:8947/api/v1/templates

# Instances  
curl http://localhost:8947/api/v1/instances

# Projects
curl http://localhost:8947/api/v1/projects
```

## 🏢 Enterprise Features

### Project-Based Research Organization
```bash
# Create research project with budget
cws project create "machine-learning-research" --budget 500.00

# Add team members with roles
cws project member add machine-learning-research researcher@university.edu --role member
cws project member add machine-learning-research advisor@university.edu --role admin

# Launch instances within project
cws launch "Python Machine Learning (Simplified)" analysis --project machine-learning-research

# Track costs in real-time
cws project cost machine-learning-research --breakdown
```

### Budget Management & Cost Control
```bash
# Set budget limits and alerts
cws project budget machine-learning-research set --monthly-limit 500.00 --alert-threshold 0.8

# View cost analytics
cws project cost machine-learning-research --savings

# Automated budget actions (hibernation when limits approached)
```

### Hibernation & Cost Optimization
```bash
# Manual hibernation (preserves RAM state)
cws hibernate my-instance
cws resume my-instance

# Automated hibernation policies
cws idle profile list                           # Show available policies
cws idle instance my-gpu-workstation --profile gpu  # Apply GPU-optimized policy
cws idle history                                # View hibernation audit trail
```

## 🌍 Platform Support

### Distributed Binaries (Homebrew/GitHub Releases)
- **macOS (Intel/ARM)**: CLI + TUI + daemon + native keychain integration
- **Linux (x64/ARM64)**: CLI + TUI + daemon with secure file storage  
- **Windows (x64)**: CLI + daemon (TUI support planned)

### Development Builds (Source)
- **Full GUI Support**: Available when building from source on all platforms
- **Native Features**: Complete keychain integration, hardware acceleration

## 📖 Documentation

- **[AWS Setup Guide](AWS_SETUP_GUIDE.md)**: Complete AWS account and profile configuration
- **[Installation Guide](INSTALL.md)**: Comprehensive installation instructions
- **[Demo Sequence](DEMO_SEQUENCE.md)**: 15-minute comprehensive demo script
- **[Demo Script](demo.sh)**: Quick 5-minute executable demo
- **[Demo Results](DEMO_RESULTS.md)**: Testing results and replication guide

### Quick Help
```bash
cws --help              # Show all commands
cws templates           # List available templates  
cws templates info <template>  # Detailed template information
cws doctor              # System health check
cws daemon status       # Check daemon status
```

## 🔐 Security & Reliability

- **66 comprehensive test files** ensuring production reliability
- **Automated security scanning** of all dependencies and builds
- **Signed releases** with checksums for verification
- **Platform-specific implementations** with proper build constraints
- **Development mode** with keychain optimization (`CLOUDWORKSTATION_DEV=true`)

## 🗓️ Version History

### v0.4.5 (Current) - Enhanced Security and Testing Platform
**🎉 PHASE 4 COMPLETE**: Full enterprise research platform with:
- ✅ Project-based organization with role-based access control
- ✅ Advanced budget management with real-time tracking
- ✅ Complete hibernation ecosystem with automated policies  
- ✅ Template inheritance system with intelligent merging
- ✅ Multi-modal access (CLI, TUI, GUI, API) with feature parity
- ✅ Professional package management via Homebrew tap

### Previous Versions
- **v0.4.1**: Multi-modal access with GUI and package management
- **v0.4.0**: Terminal User Interface (TUI) and enhanced templates
- **v0.3.x**: Core CLI functionality and template system

## 🚀 Future Roadmap

### Phase 5: AWS-Native Research Ecosystem Expansion
- **Template Marketplace**: Community-contributed research environments
- **Advanced Storage**: OpenZFS/FSx integration for specialized workloads
- **Research Workflows**: Integration with research data management and CI/CD
- **Enhanced Networking**: Private VPC networking and data transfer optimization
- **AWS Research Services**: Deep integration with ParallelCluster, Batch, SageMaker

## 🤝 Contributing

CloudWorkstation is open source! We welcome contributions:

1. **Issues**: Report bugs or request features via [GitHub Issues](https://github.com/scttfrdmn/cloudworkstation/issues)
2. **Pull Requests**: Submit improvements via GitHub
3. **Templates**: Contribute new research environment templates
4. **Documentation**: Help improve guides and documentation

### Development Setup
```bash
git clone https://github.com/scttfrdmn/cloudworkstation.git
cd cloudworkstation

# Set development mode (avoids keychain prompts)
cp .env.example .env

# Configure git hooks for automated testing
./scripts/setup-git-hooks.sh

# Build all components
make build

# Run tests
make test
```

### Automated Testing
CloudWorkstation uses git hooks to ensure code quality:
- **Pre-commit**: Fast checks (formatting, build, unit tests)
- **Pre-push**: Comprehensive validation (all tests, E2E, integration)

To bypass hooks in emergencies: `git commit/push --no-verify`

## 📄 License

CloudWorkstation is released under the [MIT License](LICENSE).

## 🆘 Support

- **AWS Setup**: [AWS Setup Guide](AWS_SETUP_GUIDE.md) for account and profile configuration
- **Documentation**: [Installation Guide](INSTALL.md), [Demo Guide](DEMO_SEQUENCE.md)
- **Issues**: [GitHub Issues](https://github.com/scttfrdmn/cloudworkstation/issues)  
- **Quick Demo**: Run `./demo.sh` in the repository
- **System Check**: `cws doctor`

---

**CloudWorkstation v0.4.5** - From individual researchers to institutional deployments, research computing made simple, scalable, and cost-effective.