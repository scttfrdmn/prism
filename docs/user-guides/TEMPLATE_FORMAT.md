# Templates

Templates are YAML files that define what software is installed when you launch a workspace.

## Using templates

```bash
# List available templates
prism templates

# Details on a specific template
prism templates info python-ml

# Launch a workspace from a template
prism workspace launch python-ml my-project
prism workspace launch r-research stats-project --size L
```

## Built-in templates

| Template | What's included |
|----------|-----------------|
| `python-ml` | Python, PyTorch, TensorFlow, Jupyter |
| `r-research` | R, RStudio Server, Bioconductor |
| `genomics` | GATK, BWA, samtools, STAR |
| `bioinformatics` | Conda, Snakemake, BioPython |
| `deep-learning` | CUDA, cuDNN, GPU-ready PyTorch |
| `data-science` | Pandas, scikit-learn, DuckDB |
| `hpc-base` | MPI, OpenMP, GCC, CMake |

---

## Template YAML format

Templates live in the `templates/` directory and are written in YAML.

### Minimal example

```yaml
name: my-template
description: Python environment with data science tools
base: ubuntu-22.04-server-lts
architecture: x86_64

packages:
  - python3
  - python3-pip
  - jupyter

build_steps:
  - name: Install Python packages
    script: |
      pip3 install numpy pandas scikit-learn matplotlib

validation:
  - name: Check Python
    script: python3 --version
  - name: Check Jupyter
    script: jupyter --version
```

### Full schema

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique template name |
| `description` | Yes | Human-readable description |
| `base` | Yes | Base OS image (e.g. `ubuntu-22.04-server-lts`) |
| `architecture` | No | `x86_64` (default) or `arm64` |
| `inherits` | No | Parent template name (see Inheritance below) |
| `package_manager` | No | `apt`, `conda`, or `dnf` |
| `packages` | No | List of packages to install |
| `users` | No | Additional OS users to create |
| `services` | No | Services to start (e.g. `jupyter`, `rstudio`) |
| `ports` | No | Ports to open in the security group |
| `build_steps` | No | Ordered list of setup scripts |
| `validation` | No | Tests run after build to verify the template works |

### Build steps

Each build step runs a shell script:

```yaml
build_steps:
  - name: Install R packages
    script: |
      Rscript -e "install.packages(c('tidyverse', 'ggplot2'), repos='https://cran.r-project.org')"
    timeout: 1800    # seconds; default 600
```

### Validation

Validation runs after build. A non-zero exit code marks the template as broken:

```yaml
validation:
  - name: R is installed
    script: R --version
  - name: tidyverse loads
    script: Rscript -e "library(tidyverse)"
```

---

## Template inheritance

Templates can inherit from a parent. The child adds to (not replaces) the parent's packages, users, services, and ports. The child's `package_manager` replaces the parent's.

```yaml
name: python-ml-gpu
description: python-ml with GPU support
inherits: python-ml

packages:
  - cuda-toolkit-12
  - cudnn9

build_steps:
  - name: Install GPU PyTorch
    script: pip3 install torch --index-url https://download.pytorch.org/whl/cu121
```

Merging rules:
- `packages`, `users`, `services`: **append** (child adds to parent)
- `ports`: **deduplicate**
- `package_manager`: **override** (child replaces parent)
- `build_steps`: **append** (parent steps run first)

---

## Creating a custom template

1. Create a YAML file in `templates/`:
   ```bash
   cp templates/python-ml.yaml templates/my-template.yaml
   ```

2. Edit the file with your changes.

3. Validate the template:
   ```bash
   prism templates validate my-template
   ```

4. Launch a test workspace:
   ```bash
   prism workspace launch my-template test-build
   ```

---

## Tips

- Start from an existing template rather than from scratch — inheritance saves a lot of work.
- Only include packages you actually need; smaller templates launch faster.
- Add validation steps for the tools your users will rely on most.
- Set `timeout` on long build steps (R package installs, compiling from source).

---

# Advanced: full YAML reference


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