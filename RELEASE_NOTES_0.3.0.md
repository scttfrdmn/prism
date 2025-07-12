# CloudWorkstation 0.3.0 Release Notes

## Overview

CloudWorkstation 0.3.0 introduces significant enhancements to the platform, focusing on research domain specialization, infrastructure optimization, and quality of life improvements. This version completes the transition to a distributed architecture, separating the backend daemon from the client interfaces while preserving backward compatibility.

## Key Features

### Research Domain Templates

CloudWorkstation now includes a comprehensive template system with specialized environments for different research domains:

- **Bioinformatics** - Complete genomics research environment with BWA, GATK, Samtools, R Bioconductor, and Galaxy
- **Python Research** - Python + Jupyter + data science packages for research and analysis
- **R Research** - R + RStudio Server + tidyverse packages for statistical analysis
- **GIS Research** - GIS research environment with QGIS, GRASS, PostGIS and geospatial libraries
- **Neuroimaging** - Neuroimaging research environment with FSL, AFNI, ANTs and MRtrix
- **Desktop Research** - Ubuntu Desktop with NICE DCV, common research tools, and GUI applications
- **Scientific Visualization** - Comprehensive scientific visualization environment with ParaView, VisIt, VTK, and related tools

### Idle Detection System

The new idle detection system helps optimize cloud resource usage:

- Intelligent monitoring of instance resource utilization
- Configurable idle thresholds and actions (stop, hibernate, or terminate)
- Domain-specific idle profiles for different research workloads
- Activity detection for desktop environments using NICE DCV
- CLI commands for idle policy management
- Automatic cost optimization without disrupting research workflows

### Multi-Repository Support

Enhanced template management through a repository system:

- Support for multiple template repositories
- Priority-based template resolution
- Dependency resolution between templates
- Default core repository with maintained templates
- Support for private and specialized template repositories
- CLI commands for repository management

### Distributed Architecture

The distributed architecture is now fully implemented:

- Complete separation of backend daemon (cwsd) and client (cws)
- Full REST API coverage for all operations
- API versioning for backward compatibility
- Context support for all API methods
- Background state synchronization
- Cross-platform support
- Foundational work for future GUI client

## Other Improvements

### Security and Dependency Management

- SemVer 2.0 versioning implemented
- Keep a Changelog format for release notes
- Automated dependency scanning and vulnerability detection
- Supply chain security improvements with SBOM generation
- Go modules updated to latest stable versions
- Improved error handling and input validation

### Testing and Quality Assurance

- Comprehensive test suite with 85%+ coverage
- Integration tests for all core components
- Functional tests for CLI commands
- Mock implementations for AWS services
- Improved CI/CD pipeline with GitHub Actions

### Documentation

- Updated CLI reference documentation
- Template authoring guide
- API reference documentation
- AWS integration documentation
- Troubleshooting guide

## Breaking Changes

- The `CloudWorkstationAPI` interface now requires context parameters
- Template format has been updated to include validation steps
- AWS credential handling has been unified across components

## Upgrading

To upgrade from CloudWorkstation 0.2.x:

1. Stop any running CloudWorkstation daemon: `cws daemon stop`
2. Install the new version: `go install github.com/scttfrdmn/cloudworkstation@0.3.0`
3. Start the daemon: `cws daemon start`

Your existing state, instances, and templates will be preserved.

## Coming in Future Releases

- GUI client with menubar/system tray integration
- Template marketplace with community contributions
- Enhanced cost management and budget controls
- Extended cloud provider support
- Collaborative workspaces

## Contributors

- Scott Friedman (@scttfrdmn)
- CloudWorkstation Team

## License

CloudWorkstation is licensed under the Apache License 2.0.