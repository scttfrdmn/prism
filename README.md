# Cloud Workstation 

<p align="center">
  <img src="docs/images/cloudworkstation.png" alt="CloudWorkstation Logo" width="200">
</p>

<p align="center"><strong>Research computers in the cloud - ready in seconds!</strong></p>

<p align="center">
  <a href="https://github.com/scttfrdmn/cloudworkstation/actions/workflows/dependency-scan.yml">
    <img src="https://github.com/scttfrdmn/cloudworkstation/actions/workflows/dependency-scan.yml/badge.svg" alt="Dependency Scan">
  </a>
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

CloudWorkstation helps you create powerful computers in the cloud for your research projects! It's like having a supercomputer you can turn on and off whenever you need it.

No more waiting hours to set up software - CloudWorkstation comes with everything you need already installed!

## Cool Things You Can Do

- **Launch a Python computer** with all the science tools already installed
- **Create an R statistics workstation** with RStudio ready to go
- **Set up a powerful computer for brain research** with all the special tools scientists use
- **Start and stop your cloud computer** whenever you want
- **Pay only for what you use** - turn it off when you're not using it!

## Getting Started in 3 Easy Steps

### Step 1: Install CloudWorkstation

#### With Package Managers (Coming in v0.4.1)

```bash
# macOS and Linux
brew install scttfrdmn/cloudworkstation/cloudworkstation

# Windows
choco install cloudworkstation

# Via Conda (all platforms)
conda install cloudworkstation -c scttfrdmn
```

#### Manual Installation

```bash
# Download and install
git clone https://github.com/scttfrdmn/cloudworkstation
cd cloudworkstation
go build -o cws

# Move it so you can use it from anywhere
sudo mv cws /usr/local/bin/
```

### Step 2: Launch Your First Cloud Computer

```bash
# Launch a Python research computer named "my-project"
cws launch python-research my-project
```

That's it! CloudWorkstation handles everything else automatically.

### Step 3: Connect and Start Working!

```bash
# See your running computer
cws list

# Connect to your computer
cws connect my-project

# When you're done, turn it off to save money
cws stop my-project
```

## Ways to Use CloudWorkstation

### Command Line Interface (CLI)
Simple commands you can type to control everything:
```bash
cws launch python-research my-project  # Create a new computer
cws list                               # See all your computers
cws connect my-project                 # Connect to your computer
cws stop my-project                    # Turn off your computer
cws delete my-project                  # Delete your computer when done
```

### Terminal User Interface (TUI) - NEW!
A colorful screen-based interface you can navigate with arrow keys:
```bash
cws tui
```

<p align="center">
  <img src="https://via.placeholder.com/800x400?text=CloudWorkstation+TUI" alt="TUI Screenshot" width="600">
</p>

### GUI Coming Soon!
A point-and-click interface is coming in the next version!

## Cool Science Environments Available

Pick the perfect computer for your research:

| Environment | What's Included | Great For |
|-------------|----------------|-----------|
| **python-research** | Python, Jupyter, pandas, numpy, scikit-learn | Data analysis, machine learning |
| **r-research** | R, RStudio, tidyverse, ggplot2 | Statistics, data visualization |
| **neuroimaging** | FSL, AFNI, ANTs | Brain research |
| **bioinformatics** | BWA, GATK, Samtools | DNA/RNA analysis |
| **gis-research** | QGIS, GRASS, PostGIS | Map making, geography |
| **desktop-research** | Full Ubuntu desktop | General research with GUI |

## Security

CloudWorkstation takes security seriously:

- All templates are regularly scanned for vulnerabilities
- Dependencies are automatically monitored and updated
- Releases include signed binaries with checksums
- We follow secure coding practices and conduct regular reviews
- All builds undergo automated security scanning

## Need Help?

Try these commands:
```bash
# Learn how to use CloudWorkstation
cws help

# Test if everything is working correctly
cws test

# See detailed information about available templates
cws templates
```

## New in Version 0.4.0!

- **Terminal User Interface (TUI)** - Colorful, interactive screens
- **Dashboard** - See all your computers and costs at a glance
- **Smart templates** - CloudWorkstation picks the best settings automatically
- **Keyboard shortcuts** - Work faster with quick commands
- **Tab navigation** - Easily switch between different sections

**Coming in Version 0.4.1: GUI interface!**