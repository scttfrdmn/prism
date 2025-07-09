# CloudWorkstation 0.1.0-alpha Release Notes

We're excited to announce the first alpha release of CloudWorkstation, a command-line tool that allows academic researchers to launch pre-configured cloud workstations in seconds.

## Overview

CloudWorkstation 0.1.0-alpha is our first public release. It features a distributed architecture with a REST API backend daemon and a lightweight CLI client, implementing our core design principles:

1. **Default to Success**: Every template works out of the box
2. **Optimize by Default**: Automatic selection of instance types
3. **Transparent Fallbacks**: Clear communication when ideal configurations aren't available
4. **Helpful Warnings**: Guidance for suboptimal choices
5. **Zero Surprises**: Clear previews and cost estimates
6. **Progressive Disclosure**: Simple by default, detailed when needed

## Key Features

### Core Functionality
- **Template-based Provisioning**: Launch environments with a single command
- **Instance Management**: Launch, list, connect, stop, start, and delete instances
- **Storage Management**: Create and manage EFS/EBS volumes
- **Cost Awareness**: Clear cost estimates for all resources

### Research Templates
- **python-research**: Python with scientific and ML libraries
- **neuroimaging**: FSL, AFNI, ANTs and MRtrix
- **bioinformatics**: BWA, GATK, Samtools and R Bioconductor
- **gis-research**: QGIS, GRASS, PostGIS and geospatial libraries
- **desktop-research**: Ubuntu Desktop with NICE DCV

### Architecture
- **Distributed System**: REST API daemon + CLI client
- **State Management**: Local JSON state persistence
- **AWS Integration**: Direct AWS SDK integration
- **Cross-Platform**: macOS, Linux, Windows support

## Installation

```bash
# Clone the repository
git clone https://github.com/username/cloudworkstation.git

# Build the binaries
cd cloudworkstation
make build

# Copy binaries to path
cp bin/cws bin/cwsd /usr/local/bin/
```

## Basic Usage

```bash
# Start the daemon
cws daemon start

# List available templates
cws templates

# Launch a new workstation
cws launch python-research my-project

# List running instances
cws list

# Get connection information
cws connect my-project

# Stop an instance
cws stop my-project

# Delete an instance
cws delete my-project
```

## What's Next

This alpha release establishes our core architecture and functionality. In our upcoming releases, we plan to add:

- **Terminal User Interface (TUI)**: Interactive terminal interface (v0.2.0)
- **Graphical User Interface (GUI)**: System tray and desktop application
- **Template Registry**: Online repository for sharing research environments
- **Collaboration Features**: Shared workspaces and resource management
- **Advanced Cost Controls**: Budget management and optimization

## Feedback and Contributions

We welcome feedback and contributions\! Please file issues and pull requests on our GitHub repository.
