# Conda Implementation Documentation

## Overview

Prism supports conda-based package management using **Miniforge**, the community-driven, conda-forge-focused distribution of conda. This document describes our implementation and how it follows standard Miniforge installation practices.

## Why Miniforge?

- **Standard Practice**: Miniforge is the recommended conda distribution for modern Python/data science projects
- **Conda-Forge First**: Uses conda-forge channel by default (largest, most up-to-date package repository)
- **Cross-Platform**: Works on x86_64 and ARM64 architectures
- **Open Source**: Fully open-source, unlike Anaconda which has licensing restrictions for commercial use
- **Lightweight**: Smaller initial download than Anaconda

## Implementation

### Installation Script

Located in: `pkg/templates/script_generator.go` - `condaScriptTemplate`

Our implementation follows the official Miniforge installation guide: https://github.com/conda-forge/miniforge

```bash
# Install Miniforge (standard conda-forge distribution)
# Following official Miniforge installation: https://github.com/conda-forge/miniforge
ARCH=$(uname -m)
MINIFORGE_URL="https://github.com/conda-forge/miniforge/releases/latest/download/Miniforge3-Linux-${ARCH}.sh"
wget -O /tmp/miniforge.sh "$MINIFORGE_URL"
bash /tmp/miniforge.sh -b -p /opt/miniforge
rm /tmp/miniforge.sh

# Initialize conda for bash (standard approach - modifies shell rc files)
/opt/miniforge/bin/conda init bash

# Reload bash environment to make conda available
export PATH="/opt/miniforge/bin:$PATH"
source /root/.bashrc || true
```

### Key Design Decisions

1. **Installation Location**: `/opt/miniforge`
   - System-wide installation accessible to all users
   - Standard practice for multi-user environments
   - Different from default `~/miniforge3` for single-user installs

2. **Non-Interactive Installation**: `-b` flag
   - Required for automated cloud instance provisioning
   - Standard practice for CI/CD and cloud deployments

3. **Standard Initialization**: `conda init bash`
   - Uses conda's built-in initialization (standard approach)
   - Automatically modifies `.bashrc` for proper PATH and environment setup
   - NO manual PATH manipulation - let conda handle it

4. **Per-User Initialization**:
   ```bash
   # For each template user
   sudo -u <username> /opt/miniforge/bin/conda init bash
   ```
   - Each user gets their own conda configuration
   - Follows standard multi-user conda setup

## Package Installation

### Conda Packages

Installed via conda-forge channel:

```bash
/opt/miniforge/bin/conda install -y package1 package2 package3
```

### Pip Packages

Installed via Miniforge's pip (compatible with conda environments):

```bash
/opt/miniforge/bin/pip install package1 package2 package3
```

**Best Practice**: Use conda for most packages, pip only for packages not available in conda-forge.

## Template Configuration

### Example Template (python-ml-workstation.yml)

```yaml
name: "Python ML Workstation"
package_manager: "conda"

packages:
  conda:
    - python=3.11
    - jupyter
    - numpy
    - pandas
    - matplotlib
    - seaborn
    - scikit-learn
    - pytorch

  pip:
    - tensorflow  # Not available on conda-forge for ARM64
    - jupyterlab-git
    - plotly
```

### Template Fields

- **package_manager**: Must be set to `"conda"`
- **packages.conda**: List of conda-forge packages
- **packages.pip**: List of pip packages (optional)

## Architecture Support

Miniforge supports both x86_64 and ARM64 architectures automatically:

```bash
ARCH=$(uname -m)  # Returns "x86_64" or "aarch64"
MINIFORGE_URL="https://github.com/conda-forge/miniforge/releases/latest/download/Miniforge3-Linux-${ARCH}.sh"
```

## Progress Monitoring

Conda installation and package installation are integrated with Prism's progress monitoring system:

```bash
progress "STAGE:system-packages:START"
# Install Miniforge
progress "STAGE:system-packages:COMPLETE"
progress "STAGE:conda-packages:START"
# Install conda packages
progress "STAGE:conda-packages:COMPLETE"
progress "STAGE:pip-packages:START"
# Install pip packages
progress "STAGE:pip-packages:COMPLETE"
```

## Comparison with Other Approaches

### ❌ What We DON'T Do (Common Anti-Patterns)

1. **Manual PATH manipulation in /etc/environment**
   ```bash
   # WRONG - Don't do this
   echo 'export PATH="/opt/miniforge/bin:$PATH"' >> /etc/environment
   ```
   - `/etc/environment` doesn't support variable expansion in appended lines
   - Can corrupt system PATH
   - Not necessary - `conda init` handles this correctly

2. **Manual .bashrc modification**
   ```bash
   # WRONG - Don't do this
   echo 'export PATH="/opt/miniforge/bin:$PATH"' >> ~/.bashrc
   ```
   - `conda init` already does this properly
   - Duplicates initialization code
   - Can cause conflicts with conda's own initialization

3. **Using Miniconda or Anaconda**
   - Miniconda: Older, less actively maintained
   - Anaconda: Commercial licensing restrictions for organizations
   - Miniforge: Modern, open-source, conda-forge focused

### ✅ What We DO (Standard Practices)

1. **Let conda init handle everything**
   - Modifies shell configuration files correctly
   - Sets up conda activation
   - Handles PATH correctly

2. **Use Miniforge**
   - Modern, recommended distribution
   - Conda-forge by default
   - Fully open-source

3. **System-wide installation**
   - Appropriate for cloud workstations
   - Shared across multiple users
   - Reduces duplication

## Testing

### Manual Testing

To test conda template provisioning:

```bash
# Build Prism
make build

# Launch conda-based template
./bin/cws launch python-ml-workstation test-conda

# SSH into instance
./bin/cws connect test-conda

# Verify conda is available
conda --version
python --version
jupyter --version

# Check installed packages
conda list
```

### Automated Testing

Conda template validation is included in pre-push tests:

```bash
# Run smoke tests
./scripts/smoke-test.sh

# Verify template validation
./bin/cws templates validate
```

## Troubleshooting

### Issue: conda command not found

**Cause**: Shell hasn't sourced .bashrc after conda init

**Solution**: Log out and log back in, or:
```bash
source ~/.bashrc
```

### Issue: Package installation fails

**Cause**: Network issues or package not available for architecture

**Solution**:
1. Check conda-forge availability for your architecture
2. Consider using pip for packages not in conda-forge
3. Check network connectivity

### Issue: Permission errors

**Cause**: Incorrect ownership of conda installation

**Solution**:
```bash
# Fix ownership
sudo chown -R $(whoami):$(whoami) /opt/miniforge
```

## References

- **Official Miniforge**: https://github.com/conda-forge/miniforge
- **Conda Documentation**: https://docs.conda.io/
- **Conda-Forge**: https://conda-forge.org/
- **Prism Templates**: `templates/python-ml-workstation.yml`, `templates/r-research-workstation.yml`

## Version History

- **v0.4.5**: Initial conda support with Miniforge
- **v0.4.6**: Fixed PATH corruption issues
- **v0.5.6**: Standardized implementation following Miniforge best practices

## Future Improvements

- [ ] Conda environment isolation per project
- [ ] Mamba support for faster package resolution
- [ ] Conda environment.yml support for reproducible environments
- [ ] Integration with research user home directories on EFS
