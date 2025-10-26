# Prism Conda Package

This document describes how to set up and maintain the Conda package for Prism.

## Overview

[Conda](https://docs.conda.io/en/latest/) is a package manager widely used in scientific computing that simplifies software installation and environment management. The Prism Conda package makes it easy for researchers to install and use Prism in their scientific computing environments.

## Package Structure

The Prism Conda package is structured as follows:

```
packaging/conda/
├── meta.yaml    # Package metadata and build instructions
├── build.sh     # Unix/macOS build script
└── bld.bat      # Windows build script
```

## Package Elements

### Package Specification (meta.yaml)

The `meta.yaml` file is the heart of the Conda package and contains:

- Package metadata (name, version)
- Source URLs and checksums for each platform/architecture
- Build instructions
- Dependencies
- Test commands
- Package information (description, license, etc.)

### Build Scripts

Two build scripts handle platform-specific installation:

- **build.sh**: For Unix-based systems (Linux, macOS)
- **bld.bat**: For Windows systems

These scripts copy the Prism binaries and auxiliary files (completions, man pages) to the appropriate locations in the Conda environment.

## Updating the Package

The package is updated automatically by the GitHub Actions workflow when a new release is created. The process:

1. Downloads the release artifacts for all platforms
2. Calculates SHA256 checksums
3. Updates the meta.yaml with new version and checksums
4. Builds the Conda package
5. Uploads to the Anaconda Cloud

To manually update the package:

```bash
# From the project root
python scripts/update_conda.py v0.4.2 ./dist/v0.4.2

# Build the package
conda build packaging/conda
```

## Testing the Package Locally

To test the Conda package locally:

```bash
# Build the package
conda build packaging/conda

# Install from local build
conda install --use-local prism

# Test installation
prism --version
cwsd --version

# Uninstall
conda remove prism
```

## Publishing to Anaconda Cloud

Once tested, the package can be uploaded to [Anaconda Cloud](https://anaconda.org/):

```bash
# Login to Anaconda Cloud
anaconda login

# Upload the package
anaconda upload /path/to/conda/build/output/prism-0.4.2-*.tar.bz2
```

## Channel Setup

The Prism Conda package is distributed through a dedicated Conda channel:

```bash
# Add the channel
conda config --add channels scttfrdmn

# Install Prism
conda install prism
```

## Continuous Integration

The GitHub Actions workflow `.github/workflows/conda-update.yml` automates the package update process. The workflow:

1. Triggers when a new release is published
2. Downloads the release artifacts
3. Updates meta.yaml with new version information and checksums
4. Builds the Conda package
5. Uploads it to Anaconda Cloud

### Required Secrets

To enable CI automation, add the following secret to your GitHub repository:

- `ANACONDA_TOKEN`: Your Anaconda Cloud API token for package upload

## Scientific Computing Integration

For scientific users, the Conda package offers several advantages:

- **Environment Isolation**: Prism can be installed in specific Conda environments
- **Dependency Management**: Conda handles dependencies automatically
- **Cross-Platform**: Works consistently across Linux, macOS, and Windows
- **Research Workflow**: Integrates with existing Jupyter, R, and Python environments

## Best Practices

- Always test packages locally before publishing
- Use specific version constraints for dependencies
- Include comprehensive test commands
- Ensure binary compatibility with common research platforms
- Provide clear documentation for scientific users

## Example Usage in Research Environments

```bash
# Create a research environment with Prism
conda create -n research python=3.10 jupyter prism

# Activate the environment
conda activate research

# Launch a Python research environment
prism launch python-research my-analysis

# Use with Jupyter for data analysis
jupyter notebook
```