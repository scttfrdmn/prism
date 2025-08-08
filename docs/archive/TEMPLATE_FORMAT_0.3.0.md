# CloudWorkstation Template Format (0.3.0)

This document describes the enhanced YAML template format for CloudWorkstation 0.3.0, including the new research domain extensions.

## Overview

Templates define the steps needed to build an Amazon Machine Image (AMI) for a specific research environment. In version 0.3.0, the template format has been extended to support:

1. Research domain categorization
2. Idle detection configuration
3. Multi-repository support
4. Enhanced cost and resource management

## Basic Template Structure

```yaml
name: "template-name"
description: "A description of the template"
base: "ubuntu-22.04-server-lts"
architecture: "x86_64"  # or arm64
version: "1.0.0"        # Template version (new in 0.3.0)

# Research domain extension (new in 0.3.0)
domain:
  category: "computer-science"   # Main research category
  subcategory: "machine-learning"  # Specific research domain
  workload_type: "gpu-intensive"  # Computational profile

# Resource recommendations
resources:
  sizes:
    XS:
      instance_type: "t4g.medium"
      architecture: "arm64"
    S:
      instance_type: "t4g.large"
      architecture: "arm64"
    M:
      instance_type: "m6g.xlarge"
      architecture: "arm64"
    L:
      instance_type: "m6g.2xlarge"
      architecture: "arm64"
    XL:
      instance_type: "m6g.4xlarge"
      architecture: "arm64"
    GPU-S:
      instance_type: "g4dn.xlarge"
      architecture: "x86_64"
    GPU-M:
      instance_type: "g4dn.2xlarge"
      architecture: "x86_64"
    GPU-L:
      instance_type: "g4dn.4xlarge"
      architecture: "x86_64"
  default_size: "M"             # Default instance size
  memory_required: 8            # Minimum memory in GB
  cpu_required: 2               # Minimum CPU cores
  gpu_recommended: true         # Whether GPU is recommended

# Cost estimates (daily)
cost:
  base_daily: 2.40             # Base daily cost estimate (size M)
  xs_daily: 0.60               # Cost estimates for each size
  s_daily: 1.20
  m_daily: 2.40
  l_daily: 4.80
  xl_daily: 9.60
  gpu_s_daily: 8.40
  gpu_m_daily: 16.80
  gpu_l_daily: 33.60

build_steps:
  - name: "Step name"
    script: |
      # Commands to run
    timeout_seconds: 600  # Optional

validation:
  - name: "Test name"
    script: |
      # Commands to run for validation

# Idle detection configuration (new in 0.3.0)
idle_detection:
  profile: "standard"          # Default profile
  cpu_threshold: 10            # CPU usage percentage
  memory_threshold: 30         # Memory usage percentage
  network_threshold: 50        # Network activity (KBps)
  disk_threshold: 100          # Disk I/O (KBps)
  gpu_threshold: 5             # GPU usage percentage
  idle_minutes: 30             # Minutes before action
  action: "stop"               # Action: stop, hibernate, notify

# Multi-repository support (new in 0.3.0)
repository:
  name: "default"              # Repository name
  url: "github.com/scttfrdmn/cloudworkstation-repository"
  maintainer: "CloudWorkstation Team"
  license: "MIT"

# Dependencies (new in 0.3.0)
dependencies:
  - repository: "default"
    template: "base/ubuntu-desktop"
    version: "1.0.0"
  - repository: "default"
    template: "stacks/python-ml"
    version: "1.1.0"

# Documentation (new in 0.3.0)
docs:
  usage_examples:
    - description: "Launch Jupyter notebook server"
      command: "jupyter notebook --ip=0.0.0.0 --no-browser"
    - description: "Run PyTorch GPU benchmark"
      command: "python3 /opt/benchmarks/pytorch_benchmark.py"
  common_workflows:
    - name: "Data preprocessing pipeline"
      description: "Run standard data preprocessing steps"
      steps:
        - "Upload data to instance"
        - "Run preprocessing script: python3 preprocess.py"
        - "Visualize results: python3 visualize.py"
```

## Research Domain Extensions (0.3.0)

The 0.3.0 release introduces structured research domain metadata to better organize and recommend templates.

### Domain Categories

```yaml
domain:
  category: "life-sciences"        # Top-level research category
  subcategory: "genomics"          # Specific research domain
  workload_type: "batch-processing" # Computational profile
  analysis_type: "sequence-analysis" # Type of analysis
  data_scale: "large"              # Expected data scale
  common_tools:                    # List of common tools included
    - "BWA"
    - "GATK"
    - "SAMtools"
  recommended_storage: 500         # Recommended storage in GB
  idle_profile: "batch"            # Default idle detection profile
```

### Available Categories

1. **life-sciences**
   - genomics
   - structural-biology
   - systems-biology
   - neuroscience
   - drug-discovery

2. **physical-sciences**
   - climate-science
   - materials-science
   - physics-simulation
   - astronomy
   - geoscience

3. **engineering**
   - cfd
   - mechanical
   - electrical
   - aerospace

4. **computer-science**
   - machine-learning
   - hpc
   - data-science
   - quantum-computing

5. **social-sciences**
   - digital-humanities
   - economics
   - social-research

6. **interdisciplinary**
   - mathematical-modeling
   - visualization
   - workflow-management

### Workload Types

- **interactive**: User actively working on instance
- **batch-processing**: Running jobs without continuous interaction
- **gpu-intensive**: Requires GPU for performance
- **memory-intensive**: Requires large memory
- **storage-intensive**: Requires large storage
- **network-intensive**: Requires high network bandwidth

## Idle Detection Configuration

Idle detection configuration allows templates to specify appropriate resource monitoring for cost optimization:

```yaml
idle_detection:
  profile: "standard"          # Default profile
  cpu_threshold: 10            # CPU usage percentage
  memory_threshold: 30         # Memory usage percentage
  network_threshold: 50        # Network activity (KBps)
  disk_threshold: 100          # Disk I/O (KBps)
  gpu_threshold: 5             # GPU usage percentage
  idle_minutes: 30             # Minutes before action
  action: "stop"               # Action: stop, hibernate, notify
  notification: true           # Send notification
```

Available profiles:
- `standard`: Balanced for interactive work (default)
- `batch`: For batch processing jobs
- `gpu`: Optimized for GPU workloads
- `data-intensive`: For data processing workloads
- `custom`: User-defined thresholds

## Multi-Repository Support

Templates can specify their source repository and dependencies:

```yaml
repository:
  name: "default"              # Repository name
  url: "github.com/scttfrdmn/cloudworkstation-repository"
  maintainer: "CloudWorkstation Team"
  license: "MIT"

dependencies:
  - repository: "default"
    template: "base/ubuntu-desktop"
    version: "1.0.0"
  - repository: "default"
    template: "stacks/python-ml"
    version: "1.1.0"
```

## Enhanced Documentation

Templates now include structured documentation for better user experience:

```yaml
docs:
  usage_examples:
    - description: "Launch Jupyter notebook server"
      command: "jupyter notebook --ip=0.0.0.0 --no-browser"
    - description: "Run PyTorch GPU benchmark"
      command: "python3 /opt/benchmarks/pytorch_benchmark.py"
  common_workflows:
    - name: "Data preprocessing pipeline"
      description: "Run standard data preprocessing steps"
      steps:
        - "Upload data to instance"
        - "Run preprocessing script: python3 preprocess.py"
        - "Visualize results: python3 visualize.py"
  troubleshooting:
    - problem: "Jupyter notebook not accessible"
      solution: "Check that the security group allows port 8888"
```

## Example: Machine Learning Template

```yaml
name: "machine-learning"
description: "Python environment with machine learning libraries"
base: "ubuntu-22.04-server-lts"
architecture: "x86_64"
version: "1.0.0"

domain:
  category: "computer-science"
  subcategory: "machine-learning"
  workload_type: "gpu-intensive"
  analysis_type: "deep-learning"
  data_scale: "large"
  common_tools:
    - "PyTorch"
    - "TensorFlow"
    - "Jupyter"
    - "scikit-learn"
  recommended_storage: 100
  idle_profile: "gpu"

resources:
  sizes:
    XS:
      instance_type: "t4g.medium"
      architecture: "arm64"
    S:
      instance_type: "t4g.large"
      architecture: "arm64"
    M:
      instance_type: "m6g.xlarge"
      architecture: "arm64"
    L:
      instance_type: "m6g.2xlarge"
      architecture: "arm64"
    XL:
      instance_type: "m6g.4xlarge"
      architecture: "arm64"
    GPU-S:
      instance_type: "g4dn.xlarge"
      architecture: "x86_64"
    GPU-M:
      instance_type: "g4dn.2xlarge"
      architecture: "x86_64"
    GPU-L:
      instance_type: "g4dn.4xlarge"
      architecture: "x86_64"
  default_size: "GPU-S"
  memory_required: 16
  cpu_required: 4
  gpu_recommended: true

cost:
  base_daily: 8.40
  xs_daily: 0.60
  s_daily: 1.20
  m_daily: 2.40
  l_daily: 4.80
  xl_daily: 9.60
  gpu_s_daily: 8.40
  gpu_m_daily: 16.80
  gpu_l_daily: 33.60

build_steps:
  - name: "Update system packages"
    script: |
      apt-get update
      apt-get upgrade -y
    timeout_seconds: 300
    
  - name: "Install system dependencies"
    script: |
      apt-get install -y build-essential python3-pip git curl
    timeout_seconds: 600
    
  - name: "Install NVIDIA drivers"
    script: |
      apt-get install -y nvidia-driver-525 nvidia-utils-525
    timeout_seconds: 900
    
  - name: "Install CUDA toolkit"
    script: |
      wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/cuda-keyring_1.0-1_all.deb
      dpkg -i cuda-keyring_1.0-1_all.deb
      apt-get update
      apt-get install -y cuda-toolkit-12.0
    timeout_seconds: 1800
    
  - name: "Install Python packages"
    script: |
      pip3 install numpy pandas scikit-learn matplotlib jupyter
      pip3 install torch torchvision torchaudio
      pip3 install tensorflow
    timeout_seconds: 1200

validation:
  - name: "Verify Python installation"
    script: python3 --version
    
  - name: "Verify ML libraries"
    script: |
      python3 -c "import numpy; import pandas; import sklearn; import torch; import tensorflow; print('All libraries loaded')"
      
  - name: "Verify NVIDIA driver"
    script: nvidia-smi
    
  - name: "Verify CUDA"
    script: nvcc --version

idle_detection:
  profile: "gpu"
  cpu_threshold: 5
  memory_threshold: 20
  network_threshold: 50
  disk_threshold: 100
  gpu_threshold: 3
  idle_minutes: 15
  action: "stop"
  notification: true

repository:
  name: "default"
  url: "github.com/scttfrdmn/cloudworkstation-repository"
  maintainer: "CloudWorkstation Team"
  license: "MIT"

docs:
  usage_examples:
    - description: "Launch Jupyter notebook server"
      command: "jupyter notebook --ip=0.0.0.0 --no-browser"
    - description: "Run PyTorch GPU test"
      command: "python3 -c 'import torch; print(torch.cuda.is_available())'"
  common_workflows:
    - name: "Train a simple neural network"
      description: "Train a basic neural network on MNIST dataset"
      steps:
        - "Clone example repository: git clone https://github.com/example/mnist-pytorch.git"
        - "Run training script: cd mnist-pytorch && python3 train.py"
        - "View results: python3 visualize.py"
  troubleshooting:
    - problem: "GPU not detected by PyTorch"
      solution: "Check that NVIDIA drivers are installed with 'nvidia-smi'. If missing, run 'sudo apt-get install -y nvidia-driver-525'"
```

## Template Organization

CloudWorkstation 0.3.0 introduces a standardized repository structure:

```
repository/
├── domains/
│   ├── life-sciences/
│   │   ├── genomics.yaml
│   │   ├── neuroscience.yaml
│   │   └── ...
│   ├── physical-sciences/
│   │   ├── climate.yaml
│   │   └── ...
│   └── ...
├── base/
│   ├── ubuntu-desktop.yaml
│   └── ...
└── stacks/
    ├── python-ml.yaml
    └── ...
```

## Template Development Workflow

1. Start with basic template structure
2. Define research domain metadata
3. Create build steps
4. Test locally with `cws ami build --dry-run`
5. Build AMI with `cws ami build`
6. Validate with `cws launch` and testing
7. Publish to repository with `cws repo push`

## Best Practices

### General Tips

1. **Idempotent Scripts**: Ensure your scripts are idempotent (can be run multiple times safely)
2. **Error Handling**: Include error checking in critical scripts
3. **Timeouts**: Set appropriate timeouts for long-running operations
4. **Clear Names**: Use descriptive names for steps and tests
5. **Comments**: Add comments to explain complex operations
6. **Dependencies**: Install all required dependencies explicitly
7. **Validation**: Include comprehensive validation tests

### Research Domain Best Practices

1. Use consistent categorization for related templates
2. Specify accurate workload types for proper resource allocation
3. Include common tools and example workflows
4. Configure appropriate idle detection settings for the domain
5. Provide domain-specific documentation and examples

### Build Step Recommendations

1. Start with system updates
2. Install system packages before language-specific packages
3. Use non-interactive installation flags where possible (`-y`, `DEBIAN_FRONTEND=noninteractive`, etc.)
4. For large installations, split into multiple build steps
5. Specify versions for critical software components
6. Clean up temporary files to reduce AMI size

### Testing Templates

Test your template before building:

```bash
# Validate the template format
cws ami validate my-template.yaml

# Test with dry run
cws ami build my-template.yaml --dry-run

# Build the AMI
cws ami build my-template.yaml

# Test with multi-repository support
cws repo add myrepo https://github.com/myorg/templates
cws ami validate myrepo:my-template.yaml
```