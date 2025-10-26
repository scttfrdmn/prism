# Prism Templates

This directory contains YAML template files for building Prism AMIs.

## Template Format

Templates are defined in YAML and must include the following required sections:

- `name`: Display name of the template
- `base`: Base AMI identifier (e.g., "ubuntu-22.04-server-lts")
- `description`: Detailed description of the template
- `build_steps`: List of build steps to create the AMI
- `validation`: List of tests to validate the AMI build

Optional fields include:

- `tags`: Key-value pairs to add as tags to the AMI
- `min_disk_size`: Minimum disk size in GB
- `architecture`: Target architecture (defaults to both x86_64 and arm64)

## Build Steps

Each build step consists of:

- `name`: Descriptive name of the step
- `script`: Shell script to execute
- `timeout_seconds`: Maximum execution time in seconds (default: 600)

Example:
```yaml
build_steps:
  - name: "Install Python"
    script: |
      apt-get update -y
      apt-get install -y python3 python3-pip
    timeout_seconds: 300
```

## Validation

Validation checks help ensure the AMI was built correctly. Each validation consists of:

- `name`: Descriptive name of the check
- `command`: Command to execute
- One of the following validation criteria:
  - `success`: Command must exit with code 0
  - `contains`: Command output must contain string
  - `equals`: Command output must exactly match

Example:
```yaml
validation:
  - name: "Python installed"
    command: "python3 --version"
    success: true
  
  - name: "RStudio port configured"
    command: "grep www-port /etc/rstudio/rserver.conf"
    contains: "8787"
```

## Available Templates

| Template | Description | Tags |
|----------|-------------|------|
| r-research | R + RStudio Server + tidyverse packages | `research`, `statistical-analysis` |
| python-ml | Python + Jupyter + PyTorch + TensorFlow | `research`, `machine-learning` |
| basic-ubuntu | Ubuntu 22.04 with common development tools | `general`, `development` |
| neuroimaging | FSL + AFNI + ANTs + MRtrix | `research`, `medical-imaging` |
| scientific-computing | R + Python + Julia scientific stack | `research`, `scientific-computing` |
| bioinformatics | BWA + GATK + Samtools + Bioconductor | `research`, `genomics` |

## Template Development Guidelines

1. **Idempotent Scripts**: All build steps should be idempotent (can run multiple times without side effects)
2. **Error Handling**: Include proper error handling and exit codes
3. **User Setup**: Always create a `researcher` user with sudo privileges
4. **Cleanup**: Always include a cleanup step to remove temporary files
5. **Validation**: Add comprehensive validation to verify the build
6. **Documentation**: Include sample code, usage examples, and README files for users

## Adding a New Template

To create a new template:

1. Create a new YAML file in this directory with the `.yml` extension
2. Ensure it includes all required sections
3. Add appropriate validation checks
4. Test the template with `prism ami validate <template-name>`
5. Build the AMI with `prism ami build <template-name>`
6. Update this README with details about your template

## Base AMIs

The following base AMIs are available:

- `ubuntu-22.04-server-lts`: Ubuntu 22.04 LTS (Jammy Jellyfish) minimal server installation
- `ubuntu-20.04-server-lts`: Ubuntu 20.04 LTS (Focal Fossa) minimal server installation
- `amazon-linux-2`: Amazon Linux 2 minimal installation