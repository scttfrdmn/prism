# Prism

<p align="center">
  <img src="docs/images/prism-transparent.png" alt="Prism Logo" width="200">
</p>

<p align="center"><strong>Academic Research Computing Platform - Pre-configured cloud environments made simple</strong></p>

<p align="center">
  <a href="https://github.com/scttfrdmn/prism/actions/workflows/ci.yml">
    <img alt="CI" src="https://github.com/scttfrdmn/prism/actions/workflows/ci.yml/badge.svg">
  </a>
  <a href="https://github.com/scttfrdmn/prism/releases/latest">
    <img alt="GitHub release (latest by date)" src="https://img.shields.io/github/v/release/scttfrdmn/prism">
  </a>
  <a href="https://github.com/scttfrdmn/prism/blob/main/LICENSE">
    <img alt="License" src="https://img.shields.io/github/license/scttfrdmn/prism">
  </a>
  <a href="https://goreportcard.com/report/github.com/scttfrdmn/prism">
    <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/scttfrdmn/prism?style=flat&cachebust=1">
  </a>
  <a href="https://github.com/scttfrdmn/prism/blob/main/cmd/prism-gui/frontend/eslint.config.js">
    <img alt="ESLint" src="https://img.shields.io/badge/ESLint-A%2B-success?logo=eslint&logoColor=white">
  </a>
</p>

## What is Prism?

Prism provides researchers with **pre-configured cloud workstations** for data analysis, machine learning, and computational research. Launch production-ready environments without manual configuration.

**From individual researchers to institutional deployments** - research computing made simple, scalable, and cost-effective.

**Learn more at [prismcloud.io](https://prismcloud.io)**

## 🎯 Core Design Principles

- **🎯 Default to Success**: Every template works out of the box in every supported region
- **⚡ Optimize by Default**: Smart instance sizing and cost-performance optimization  
- **🔍 Transparent Fallbacks**: Clear communication when configurations change
- **💡 Helpful Warnings**: Gentle guidance for optimal choices
- **🚫 Zero Surprises**: Users always know what they're getting
- **📈 Progressive Disclosure**: Simple by default, detailed when needed

## 🚀 Installation

### macOS

**Homebrew (Recommended)**

```bash
brew install scttfrdmn/tap/prism
```

**Desktop app (GUI)**

Download the latest `.dmg` (macOS) or `.msi` (Windows) from the
[releases page](https://github.com/scttfrdmn/prism/releases/latest).

### Linux

Download the latest release archive from the
[releases page](https://github.com/scttfrdmn/prism/releases/latest), extract, and
move the binaries onto your `PATH`:

```bash
# Replace the URL with the asset for your platform from the releases page
tar xz -f prism_<version>_linux_amd64.tar.gz
sudo mv prism prismd /usr/local/bin/
```

Or build from source (see [Build from Source](#build-from-source) below).

### Windows

**Scoop**
```powershell
scoop bucket add scttfrdmn https://github.com/scttfrdmn/scoop-bucket
scoop install prism
```

**Manual Installation**
```powershell
# Download from GitHub releases
# https://github.com/scttfrdmn/prism/releases/latest
# Extract and add to PATH
```

## 🚀 Quick Start

### First-Time Users: Interactive Wizard (⏱️ 30 seconds)

Launch your first workspace with a guided wizard!

```bash
# Configure AWS credentials (if not already set up)
aws configure

# Launch the interactive wizard
prism init
```

The wizard will guide you through:
1. ✅ **Template Selection**: Browse by category (ML/AI, Data Science, Bioinformatics, Web)
2. ✅ **Workspace Configuration**: Name and size with cost estimates
3. ✅ **Review**: See exactly what you're launching
4. ✅ **Launch**: Real-time progress tracking
5. ✅ **Connect**: SSH connection details and next steps

**Time to first workspace: ~30 seconds** 🚀

---

### Advanced: Direct Commands

For experienced users or automation:

```bash
# View available templates
prism templates

# Launch a Python ML environment
prism workspace launch python-ml my-research

# Connect via SSH
prism workspace connect my-research

# View running workspaces
prism workspace list
```

**Automatic Features:**
- ✅ Daemon starts if not running
- ✅ Optimal instance type selected
- ✅ Security groups configured
- ✅ SSH keys generated and managed
- ✅ Template provisioned and ready

**Credential Discovery** (automatic):
- Environment variables (AWS_PROFILE, AWS_ACCESS_KEY_ID)
- AWS CLI configuration (~/.aws/credentials)
- Prism profiles (for multi-account management)

## 🌟 Key Features

### 💰 Cost Optimization
- **Hibernation**: Preserve state while reducing costs by 90%
- **Idle Detection**: Automated hibernation policies with configurable thresholds
- **Budget Management**: Project-level cost tracking and alerts
- **Cost Analytics**: Real-time spending reports and forecasts

### 🏗️ Research Templates
- **21+ Pre-configured Environments**: Python ML, R, bioinformatics, web dev, and more
- **Template Inheritance**: Compose complex environments from simple building blocks
- **Smart Defaults**: Optimal instance sizing and cost-performance ratios
- **Regional Fallbacks**: Automatic handling of availability constraints

### 🏢 Enterprise & Collaboration
- **Project-Based Organization**: Multi-user projects with role-based access
- **Research User System**: Persistent identities across workspaces
- **Multi-Account Support**: Manage multiple AWS profiles efficiently
- **Template Marketplace**: Share and discover community templates

### 📱 Multi-Modal Access
- **CLI**: Fast, scriptable command-line interface
- **TUI**: Interactive terminal interface with keyboard navigation
- **GUI**: Desktop application (available when building from source)
- **REST API**: Complete HTTP API on port 8947

## 📦 Templates

**Prism ships with minimal base OS templates** (as of v0.7.0):
- Amazon Linux 2023 (x86_64, ARM64)
- Ubuntu 22.04 LTS (x86_64, ARM64)
- Ubuntu 24.04 LTS (x86_64, ARM64)

**Application templates are community-contributed:**
- **Python ML**: Jupyter, scikit-learn, TensorFlow, PyTorch
- **R Research**: RStudio, tidyverse, Bioconductor
- **Bioinformatics**: BLAST, bowtie2, samtools, bedtools
- **Web Development**: Node.js, Docker, nginx
- **Desktop Environments**: Full GUI with browsers and dev tools

```bash
# View all templates (base + community)
prism templates

# Launch a base OS template
prism workspace launch ubuntu-24-04-x86 my-instance

# Launch a community template with applications
prism workspace launch python-ml my-ml-project

# Get detailed template info
prism templates info python-ml
```

**Template Structure**: See [templates/README.md](templates/README.md) for details on base/, community/, and custom templates.

## 💻 Usage Examples

### Basic Workspace Management
```bash
# Launch a workspace
prism workspace launch python-ml my-project

# List running workspaces
prism workspace list

# Connect via SSH
prism workspace connect my-project

# Stop workspace
prism workspace stop my-project
```

### Cost Optimization
```bash
# Hibernate to preserve state while saving costs
prism workspace hibernate my-workspace
prism workspace resume my-workspace

# Automated idle policies
prism idle profile list
prism idle workspace my-gpu --profile gpu
```

### Project Management
```bash
# Create project with budget
prism project create ml-research --budget 500

# Add team members
prism project member add ml-research user@example.com --role member

# Launch workspace in project
prism workspace launch python-ml analysis --project ml-research
```

### Multi-Modal Access
```bash
# Command line
prism templates

# REST API
curl http://localhost:8947/api/v1/instances
```

## 📖 Documentation

**📚 [Complete Documentation Site](https://scttfrdmn.github.io/prism/)** - User guides, architecture docs, and persona walkthroughs

```bash
prism --help                      # Show all commands
prism templates                   # List available templates
prism templates info <template>   # Detailed template info
prism admin daemon status         # Daemon / system health check
```

**Guides:**
- [AWS Setup Guide](docs/user-guides/AWS_SETUP_GUIDE.md) - AWS account and credential configuration
- [Collaboration Quickstart](docs/user-guides/COLLABORATION_QUICKSTART.md) - Add collaborators in 5 minutes (simplified guide)
- [Multi-User Instance Setup](docs/user-guides/MULTI_USER_INSTANCE_SETUP.md) - Comprehensive multi-user collaboration guide
- [Custom AMI Workflow](docs/user-guides/CUSTOM_AMI_WORKFLOW.md) - Creating and managing custom AMIs for faster launches
- [AMI Best Practices](docs/user-guides/AMI_BEST_PRACTICES.md) - Best practices for AMI management, security, and cost optimization
- [Installation Guide](INSTALL.md) - Comprehensive installation instructions
- [Budget System Philosophy](docs/BUDGET_PHILOSOPHY.md) - Multi-budget system design and conceptual model (v0.5.10+)
- [Budget Banking Philosophy](docs/BUDGET_BANKING_PHILOSOPHY.md) - Surplus tracking and burst budgeting
- [Resource Tagging](docs/RESOURCE_TAGGING.md) - Cost optimization and zombie resource cleanup
- [Compliance Matrix](docs/admin-guides/COMPLIANCE_MATRIX.md) - NIST 800-171, HIPAA, and framework support
- [Changelog](CHANGELOG.md) - Version history and release notes

## 🗓️ Version History

See **[CHANGELOG.md](CHANGELOG.md)** for the full, up-to-date version history
following [Keep a Changelog](https://keepachangelog.com/) and semantic versioning.

## 🚀 Roadmap

**Phase 5 (Current)**: Multi-user collaboration and template marketplace
**Phase 6**: Advanced storage (FSx, S3 integration) and AWS research services
**Phase 7**: Enterprise authentication (OAuth, LDAP, SAML) and TUI enhancements

## 🤝 Contributing

Prism is open source and welcomes contributions!

- **Issues**: [Report bugs or request features](https://github.com/scttfrdmn/prism/issues)
- **Pull Requests**: Submit code improvements
- **Templates**: [Contribute research environment templates](docs/development/COMMUNITY_TEMPLATE_GUIDE.md) 🎯
- **Documentation**: Help improve guides

**Contributing Templates:**
See our [Community Template Contribution Guide](docs/development/COMMUNITY_TEMPLATE_GUIDE.md) for step-by-step instructions on creating and submitting templates to the marketplace.

**Development:**
```bash
git clone https://github.com/scttfrdmn/prism.git
cd prism
make build
make test
```

## 📄 License

[Apache License 2.0](LICENSE) - Free for academic and commercial use

## 🆘 Support

- **Documentation**: [Complete docs site](https://docs.prismcloud.host/) or `prism --help`
- **System Check**: `prism admin daemon status`
- **Issues**: [GitHub Issues](https://github.com/scttfrdmn/prism/issues)
- **Discussions**: [GitHub Discussions](https://github.com/scttfrdmn/prism/discussions)
- **AWS Setup**: See [AWS Setup Guide](docs/user-guides/AWS_SETUP_GUIDE.md)

---

**Prism** - Research computing environments made accessible | [prismcloud.host](https://prismcloud.host)