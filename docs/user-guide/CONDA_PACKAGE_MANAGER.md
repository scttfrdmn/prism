# Conda Package Manager Guide

**CloudWorkstation** uses **conda** as the primary package manager for research environments, providing reliable, cross-platform package management for Python, R, and scientific computing.

## 🎯 Why Conda for Research?

### ✅ **Research-Optimized**
- **Scientific Packages**: Comprehensive ecosystem (conda-forge, bioconda)
- **Reproducibility**: Environment isolation and dependency management
- **Cross-Platform**: Consistent across Windows, macOS, Linux, ARM64
- **GPU Support**: Native CUDA, PyTorch, TensorFlow integration

### ✅ **CloudWorkstation Integration**
- **Smart Defaults**: Automatically selected for Python/R templates
- **Manual Override**: `--with conda` for explicit control  
- **Optimized Installation**: Miniforge for fast, reliable setup
- **Multi-Architecture**: Native ARM64 and x86_64 support

## 🚀 Usage Examples

### Basic Usage (Automatic)
```bash
# Conda automatically selected for Python/R templates
cws launch python-research my-analysis
cws launch r-research stats-project

# Templates detect scientific packages and choose conda
cws launch neuroimaging brain-study
```

### Explicit Conda Selection
```bash
# Force conda package manager
cws launch python-research my-project --with conda

# Combine with other options
cws launch python-research gpu-training --with conda --size GPU-L --volume shared-data
```

### Advanced Usage
```bash
# Dry run to see conda installation script
cws launch python-research test --with conda --dry-run

# Launch with specific conda environment
cws launch r-research stats-work --with conda --storage L
```

## 📦 Supported Package Types

### Python Packages
```yaml
packages:
  conda:
    - python=3.11
    - jupyter
    - numpy=1.24.3
    - pandas=2.0.3
    - matplotlib=3.7.1
    - scikit-learn=1.3.0
    - pytorch=2.0.1
    - tensorflow=2.13.0
```

### R Packages  
```yaml
packages:
  conda:
    - r-base=4.3.0
    - rstudio
    - r-tidyverse
    - r-ggplot2
    - r-dplyr
    - r-shiny
```

### Scientific Computing
```yaml
packages:
  conda:
    - numpy
    - scipy  
    - matplotlib
    - jupyter
    - pandas
    - seaborn
    - plotly
```

## 🔧 How Conda Integration Works

### 1. **Template Detection**
CloudWorkstation automatically selects conda when templates contain:
- Python data science packages (`numpy`, `pandas`, `jupyter`)  
- R packages (`r-base`, `tidyverse`, `rstudio`)
- Scientific computing libraries (`scipy`, `matplotlib`)

### 2. **Installation Process**
```bash
# 1. Download and install Miniforge  
wget -O /tmp/miniforge.sh "$MINIFORGE_URL"
bash /tmp/miniforge.sh -b -p /opt/miniforge

# 2. Install packages via conda
/opt/miniforge/bin/conda install -y python=3.11 jupyter numpy pandas

# 3. Configure environment for users
echo 'export PATH="/opt/miniforge/bin:$PATH"' >> ~/.bashrc
/opt/miniforge/bin/conda init bash
```

### 3. **Service Integration**
- **Jupyter**: Automatically configured with conda environment
- **RStudio**: R packages available through conda integration
- **Custom Services**: Access to conda-installed packages

## 🎛️ Environment Configuration

### Multi-User Setup
```bash
# Each user gets conda access
sudo -u researcher /opt/miniforge/bin/conda init bash
echo 'export PATH="/opt/miniforge/bin:$PATH"' >> /home/researcher/.bashrc

# Shared conda installation at /opt/miniforge
# User-specific environments in ~/.conda/envs/
```

### Package Management
```bash
# Install additional packages
conda install package-name

# Create custom environments  
conda create -n myproject python=3.11 pandas numpy
conda activate myproject

# Export environment for reproducibility
conda env export > environment.yml
```

## 📊 Performance Benefits

### ✅ **Optimized for Research**
- **Fast Solving**: Miniforge uses libmamba for faster dependency resolution
- **Pre-compiled**: Binary packages avoid compilation time
- **GPU Acceleration**: Native CUDA toolkit integration
- **ARM64 Native**: Apple Silicon optimization

### ✅ **CloudWorkstation Optimizations**
- **Multi-Architecture**: Smart ARM64/x86_64 detection
- **Package Caching**: Reduced installation time for common packages
- **Environment Reuse**: Efficient environment setup across instances

## 🛠️ Troubleshooting

### Common Issues

#### Package Installation Fails
```bash
# Update conda first
conda update -n base -c defaults conda

# Clear cache if needed
conda clean --all

# Use conda-forge channel
conda install -c conda-forge package-name
```

#### Environment Issues
```bash
# Reinitialize conda
conda init bash
source ~/.bashrc

# Fix PATH issues
export PATH="/opt/miniforge/bin:$PATH"
```

#### GPU Package Issues
```bash
# Install GPU packages explicitly
conda install pytorch torchvision torchaudio pytorch-cuda=11.8 -c pytorch -c nvidia

# Verify GPU access
python -c "import torch; print(torch.cuda.is_available())"  
```

## 🔮 Future Enhancements

### Planned Improvements
- **Mamba Integration**: Even faster package solving
- **Environment Templates**: Pre-configured research environments
- **Package Caching**: Instance-level package cache optimization
- **GPU Optimization**: Enhanced CUDA/PyTorch conda integration

### Specialized Conda Support
- **Bioconda**: Bioinformatics package ecosystem
- **Conda-Forge**: Community-maintained packages
- **PyPI Integration**: Seamless pip package fallback
- **R Integration**: Enhanced R + conda workflow

## 📚 Resources

### Conda Documentation
- [Conda User Guide](https://docs.conda.io/projects/conda/en/latest/user-guide/)
- [Conda-Forge Community](https://conda-forge.org/)
- [Miniforge Project](https://github.com/conda-forge/miniforge)

### CloudWorkstation Resources
- Template examples with conda integration
- Best practices for research environments
- Multi-user conda configuration guides

---

**Summary**: Conda provides CloudWorkstation users with world-class package management for research computing, combining reliability, performance, and comprehensive scientific package ecosystems in a research-optimized platform.