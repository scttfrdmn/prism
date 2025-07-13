# CloudWorkstation 0.2.0 Release

## Release Summary

CloudWorkstation 0.2.0 focuses on two major improvements:

1. **AMI Builder System** - Replaces slow UserData scripts with pre-built AMIs
2. **Terminal User Interface (TUI)** - Provides an intuitive interface for managing workstations

These enhancements significantly improve the user experience and performance of CloudWorkstation, reducing instance launch times from 10+ minutes to under 60 seconds.

## AMI Builder System

The AMI Builder System represents a major architectural improvement, moving away from UserData scripts that run during instance launch to a system of pre-built AMIs that are ready to use immediately.

### Key Components

1. **YAML Template Format**
   - Human-readable and version-controllable template definitions
   - Structured build steps and validation tests
   - Multi-architecture support (x86_64, ARM64)
   - Enhanced documentation and metadata

2. **GitHub Actions Workflow**
   - Automated AMI building on schedule or template changes
   - Multi-region and multi-architecture build matrix
   - Validation testing for built AMIs
   - Artifact uploads for logs and test results

3. **Template Registry**
   - Storage for built AMI references
   - Region and architecture-specific lookups
   - Fallback to hard-coded AMIs when registry unavailable
   - CLI commands for registry management

4. **Validation Framework**
   - JSON Schema validation for templates
   - Runtime validation tests to verify AMI functionality
   - Comprehensive error reporting

5. **CLI Integration**
   - `cws ami build <template>` - Build an AMI from a template
   - `cws ami validate <template>` - Validate a template definition
   - `cws ami list [template]` - List available AMIs
   - `cws ami publish <template> <ami-id>` - Register AMI in registry
   - `cws registry` commands for template management

## Terminal User Interface (TUI)

The Terminal User Interface provides a modern, intuitive way to interact with CloudWorkstation, making it accessible to a wider range of users.

### Key Features

1. **Dashboard View**
   - At-a-glance overview of system status
   - Cost tracking and instance summary
   - Quick access to common actions

2. **Instances View**
   - List of all workstations with status and details
   - Management capabilities (launch, stop, start, delete)
   - Connection information and SSH commands

3. **Templates View**
   - Browse and select from available templates
   - Filter by category and search functionality
   - Detailed template information and requirements

4. **Storage Management**
   - EFS and EBS volume management
   - Create, attach, detach, and delete operations
   - Size and type selection

5. **Settings Page**
   - Theme switching (dark/light mode)
   - Configuration management
   - Region and profile selection

6. **User Experience Enhancements**
   - Notification system for async operations
   - Search functionality across all list views
   - Help panels with keyboard shortcuts
   - Progress indicators for long-running tasks

## Template Expansion

The 0.2.0 release includes an expanded template library with YAML definitions:

1. **Basic Templates**
   - `basic-ubuntu` - Plain Ubuntu 22.04 for general use
   - `desktop-research` - Ubuntu Desktop with NICE DCV

2. **Research Templates**
   - `r-research` - R + RStudio Server + tidyverse
   - `python-research` - Python + Jupyter + data science stack

3. **Specialized Research Templates**
   - `neuroimaging` - FSL + AFNI + ANTs + MRtrix
   - `bioinformatics` - BWA + GATK + Samtools + R Bioconductor
   - `gis-research` - QGIS + GRASS + PostGIS + Python geospatial

## Performance Improvements

- **Launch Time**: Reduced from 10+ minutes to under 60 seconds
- **Reliability**: Eliminated UserData script failures
- **Consistency**: All instances of the same template are identical
- **Updates**: AMIs can be pre-built and tested before deployment

## Installation and Upgrade

### Fresh Installation

```bash
# Download the latest release
curl -L https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.2.0/cws-0.2.0-$(uname -s)-$(uname -m).tar.gz | tar xz

# Move binaries to your PATH
sudo mv cws cwsd /usr/local/bin/
```

### Upgrade from 0.1.x

```bash
# Stop the daemon if running
cws daemon stop

# Download and install the new version
curl -L https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.2.0/cws-0.2.0-$(uname -s)-$(uname -m).tar.gz | tar xz
sudo mv cws cwsd /usr/local/bin/

# Start the daemon with the new version
cws daemon start
```

## Known Issues

1. **TUI Integration Tests**: Integration tests between TUI and daemon are still in development.
2. **Tab Bar Component**: The custom TabBar component is pending submission to the charmbracelet/bubbles repository.
3. **Documentation**: User documentation for TUI features is still in progress.

## Next Steps

1. **Complete TUI Integration**
   - Finalize integration tests between TUI and daemon
   - Create comprehensive user documentation
   - Submit TabBar component PR to charmbracelet/bubbles

2. **Prepare for Phase 3**
   - Develop more specialized research templates
   - Implement multi-stack template architecture
   - Add desktop environments with NICE DCV
   - Create idle detection and smart cost controls

## Contributors

Thank you to all the contributors who made this release possible:

- CloudWorkstation Team
- Open source contributors
- Beta testers and early adopters

## License

CloudWorkstation is released under the MIT License.

---

For more information, visit the [CloudWorkstation documentation](https://docs.cloudworkstation.io) or [GitHub repository](https://github.com/scttfrdmn/cloudworkstation).