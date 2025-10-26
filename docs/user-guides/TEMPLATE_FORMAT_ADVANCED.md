# Prism Template Format - Advanced Guide

This document describes the technical details of the YAML template format used by Prism to define research environment templates.

## Overview

Templates define the steps needed to build an Amazon Machine Image (AMI) for a specific research environment. Templates are written in YAML format and include metadata, build steps, and validation tests.

## Template Structure

A template consists of the following sections:

```yaml
name: template-name
description: A description of the template
base: base-image-name
architecture: x86_64  # or arm64
build_steps:
  - name: Step name
    script: |
      # Commands to run
    timeout_seconds: 600  # Optional
validation:
  - name: Test name
    script: |
      # Commands to run for validation
```

### Required Fields

| Field | Description |
|-------|-------------|
| `name` | A unique identifier for the template |
| `description` | A human-readable description of the environment |
| `base` | The base AMI to start from (e.g., ubuntu-22.04-server-lts) |
| `architecture` | The CPU architecture (x86_64 or arm64) |
| `build_steps` | A list of build steps to create the environment |

### Build Steps

Each build step consists of:

| Field | Description |
|-------|-------------|
| `name` | A descriptive name for the step |
| `script` | The shell script to execute |
| `timeout_seconds` | (Optional) Maximum execution time in seconds (default: 600) |

### Validation Tests

Validation tests verify that the environment was built correctly:

| Field | Description |
|-------|-------------|
| `name` | A descriptive name for the test |
| `script` | The shell script to execute for validation |

## Example Template

```yaml
name: python-ml
description: Python environment with machine learning libraries
base: ubuntu-22.04-server-lts
architecture: x86_64

build_steps:
  - name: Update system packages
    script: |
      apt-get update
      apt-get upgrade -y
    timeout_seconds: 300
    
  - name: Install system dependencies
    script: |
      apt-get install -y build-essential python3-pip git curl
    timeout_seconds: 600
    
  - name: Install Python packages
    script: |
      pip3 install numpy pandas scikit-learn tensorflow torch
    timeout_seconds: 1200

validation:
  - name: Verify Python installation
    script: python3 --version
    
  - name: Verify ML libraries
    script: |
      python3 -c "import numpy; import pandas; import sklearn; import tensorflow; import torch; print('All libraries loaded')"
```

## Best Practices

### General Tips

1. **Idempotent Scripts**: Ensure your scripts are idempotent (can be run multiple times safely)
2. **Error Handling**: Include error checking in critical scripts
3. **Timeouts**: Set appropriate timeouts for long-running operations
4. **Clear Names**: Use descriptive names for steps and tests
5. **Comments**: Add comments to explain complex operations
6. **Dependencies**: Install all required dependencies explicitly
7. **Validation**: Include comprehensive validation tests

### Build Step Recommendations

1. Start with system updates
2. Install system packages before language-specific packages
3. Use non-interactive installation flags where possible (`-y`, `DEBIAN_FRONTEND=noninteractive`, etc.)
4. For large installations, split into multiple build steps
5. Specify versions for critical software components
6. Clean up temporary files to reduce AMI size

### Validation Recommendations

1. Test every major component installed
2. Verify configurations are correct
3. Check that services are running if applicable
4. Test actual functionality, not just presence of binaries
5. Keep validation scripts simple and focused

## Template Organization

Prism templates are organized by research domain:

- `/templates/python-research.yaml`: Python data science environment
- `/templates/neuroimaging.yaml`: Neuroimaging tools (FSL, AFNI, etc.)
- `/templates/bioinformatics.yaml`: Bioinformatics tools (BWA, GATK, etc.)
- `/templates/gis-research.yaml`: GIS and spatial analysis tools

## Common Base Images

Prism supports multiple base images:

- `ubuntu-22.04-server-lts`: Standard Ubuntu 22.04 LTS server
- `ubuntu-22.04-server-lts-arm64`: ARM64 version of Ubuntu 22.04 LTS

## Adding New Templates

To add a new template:

1. Create a YAML file in the `/templates` directory
2. Follow the format described above
3. Test your template with `prism ami validate my-template.yaml`
4. Build the AMI with `prism ami build my-template.yaml`

## Testing Templates

Test your template before building:

```bash
# Validate the template format
prism ami validate my-template.yaml

# Test with dry run
prism ami build my-template.yaml --dry-run

# Build the AMI
prism ami build my-template.yaml
```