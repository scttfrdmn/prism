# Prism Template System

## Overview

Prism's template system has been completely redesigned to be simple, deterministic, and maintainable. The new system leverages existing package managers (apt, conda, spack) instead of custom bash scripts, following the core principle of "leverage existing tools instead of reinventing the wheel."

## Architecture

### Core Components

```
pkg/templates/
├── types.go           # Core template data structures
├── parser.go          # YAML template parsing and validation
├── script_generator.go # Installation script generation
├── resolver.go        # Template resolution logic
├── compatibility.go   # Backward compatibility layer
└── templates.go       # Main API functions
```

### Template Flow

```
YAML Template → Parser → Package Manager Selection → Script Generation → Runtime Template
```

## Template Format

Templates are now simple, declarative YAML files:

```yaml
name: "R Research Environment"
description: "R + RStudio Server + tidyverse packages"
base: "ubuntu-22.04"

# Package manager strategy (optional - auto-selected if omitted)
package_manager: "auto"  # or "apt", "conda", "spack"

# Simple package lists - package managers handle complexity
packages:
  system:
    - build-essential
    - curl
  conda:
    - r-base=4.3.0
    - rstudio
    - r-tidyverse

# Services to configure
services:
  - name: "rstudio-server"
    port: 8787
    enable: true
    config:
      - "www-port=8787"

# User setup with auto-generated secure passwords
users:
  - name: "researcher"
    password: "auto-generated"
    groups: ["sudo"]

# Instance defaults (auto-optimized per architecture)
instance_defaults:
  type: "t3.medium"
  ports: [22, 8787]
```

## Package Manager Selection

The system automatically selects the best package manager based on template content:

### Selection Rules

1. **HPC/Scientific Computing Packages** → **Spack**
   - `openmpi`, `mpich`, `fftw`, `petsc`, `paraview`, etc.
   - Optimized for scientific computing workflows

2. **Python Data Science Packages** → **Conda**  
   - `numpy`, `pandas`, `tensorflow`, `pytorch`, `jupyter`, etc.
   - Better package management for data science

3. **R Statistical Packages** → **Conda**
   - `r-base`, `rstudio`, `tidyverse`, `ggplot2`, etc.
   - Superior R ecosystem management

4. **System/General Packages** → **Apt**
   - `build-essential`, `curl`, `git`, etc.
   - Standard system package management

### Manual Override

Users can force a specific package manager:

```yaml
package_manager: "conda"  # Force conda even for system packages
```

## Benefits

### 1. Massive Simplification
- **Before**: 50+ line bash scripts with custom installation logic
- **After**: Declarative package lists, let package managers handle complexity

### 2. Deterministic Results
- Package managers provide reproducible environments
- No custom script variations or environment drift
- Version pinning handled by package manager

### 3. Maintainability
- Templates become simple configuration files
- Package manager expertise leveraged
- Easier to update and test

### 4. Smart Defaults
- Prism picks optimal package manager automatically
- Per-architecture instance types selected automatically
- Secure password generation

## Backward Compatibility

The new system maintains **100% backward compatibility** with existing code:

```go
// Existing code continues to work unchanged
templates := aws.getTemplates()
template := templates["r-research"]
```

The compatibility layer automatically:
1. Scans for new YAML templates
2. Converts them to legacy `RuntimeTemplate` format
3. Falls back to hardcoded templates if needed
4. Maintains all existing APIs

## API Usage

### Basic Usage

```go
// Get all templates (backward compatible)
templates, err := templates.GetTemplatesForRegion("us-east-1", "x86_64")

// Get single template
template, err := templates.GetTemplate("r-research", "us-east-1", "x86_64")
```

### Advanced Usage

```go
// Direct template management
registry := templates.NewTemplateRegistry([]string{"templates"})
registry.ScanTemplates()

parser := templates.NewTemplateParser()
template, err := parser.ParseTemplateFile("templates/my-template.yml")

resolver := templates.NewTemplateResolver()
runtimeTemplate, err := resolver.ResolveTemplate(template, "us-east-1", "x86_64")
```

## Script Generation

The system generates optimized installation scripts for each package manager:

### Apt Script
```bash
#!/bin/bash
apt-get update -y
apt-get install -y build-essential curl r-base
# User and service configuration
```

### Conda Script  
```bash
#!/bin/bash
# Install Miniforge
wget https://github.com/conda-forge/miniforge/releases/latest/download/Miniforge3-Linux-x86_64.sh
bash Miniforge3-Linux-x86_64.sh -b -p /opt/miniforge
/opt/miniforge/bin/conda install -y r-base rstudio tidyverse
```

### Spack Script
```bash
#!/bin/bash
# Install Spack
git clone https://github.com/spack/spack.git /opt/spack
source /opt/spack/share/spack/setup-env.sh
spack install openmpi fftw petsc
```

## Template Examples

### R Research Environment
```yaml
name: "R Research Environment"
base: "ubuntu-22.04"
package_manager: "auto"  # Will select conda

packages:
  conda:
    - r-base=4.3.0
    - rstudio
    - r-tidyverse
    - r-ggplot2

services:
  - name: "rstudio-server"
    port: 8787

users:
  - name: "researcher"
    password: "auto-generated"
    groups: ["sudo"]
```

### Python Machine Learning
```yaml
name: "Python Machine Learning"
base: "ubuntu-22.04" 
package_manager: "auto"  # Will select conda

packages:
  conda:
    - python=3.11
    - jupyter
    - numpy
    - pandas
    - scikit-learn
    - tensorflow

services:
  - name: "jupyter"
    port: 8888

# Prism detects ML packages and recommends GPU instance
```

### HPC Scientific Computing
```yaml
name: "HPC Environment"
base: "ubuntu-22.04"
package_manager: "auto"  # Will select spack

packages:
  spack:
    - openmpi@4.1.0
    - fftw@3.3.10
    - petsc@3.18.0
    - paraview@5.11.0

users:
  - name: "researcher"
    password: "auto-generated"
```

## Migration Path

### From Hardcoded Templates

Existing hardcoded templates can be automatically migrated:

```go
templates.MigrateFromLegacy("output-dir")
```

This converts hardcoded `RuntimeTemplate` structs to new YAML format.

### Template Validation  

```bash
# Validate template syntax
prism validate-template templates/my-template.yml

# List available templates
prism templates list

# Get template information
prism templates info r-research
```

## Implementation Status

### ✅ Completed
- [x] Unified template data structures
- [x] YAML template parser with validation  
- [x] Package manager selection logic
- [x] Script generation for apt/conda/spack
- [x] Template resolution system
- [x] Backward compatibility layer
- [x] Integration with existing AWS manager
- [x] Example templates created

### 🔄 Future Enhancements
- [ ] YAML marshaling for template creation tools
- [ ] Template repository system
- [ ] Template versioning and dependencies
- [ ] Advanced validation rules
- [ ] Template sharing and marketplace

## Performance

The new system is significantly more efficient:

- **Template Loading**: O(1) lookup vs O(n) hardcoded map search
- **Script Generation**: Template-based vs string concatenation
- **Caching**: Built-in template registry caching
- **Memory Usage**: YAML parsing only when needed

## Security

- **Auto-generated Passwords**: Cryptographically secure random passwords
- **Validation**: Comprehensive template validation prevents injection
- **Package Verification**: Package managers handle signature verification
- **Principle of Least Privilege**: Users created with minimal required permissions

## Testing

The template system includes comprehensive testing:

```go
// Template parsing tests
template, err := parser.ParseTemplateFile("test-template.yml")

// Package manager selection tests  
pm := strategy.SelectPackageManager(template)

// Script generation tests
script, err := generator.GenerateScript(template, pm)

// Integration tests with backward compatibility
legacyTemplates, err := GetTemplatesForRegion("us-east-1", "x86_64")
```

## Conclusion

The new template system represents a fundamental improvement in Prism's architecture:

1. **Simplicity**: Templates went from complex bash scripts to simple YAML configuration
2. **Reliability**: Package managers provide deterministic, reproducible builds  
3. **Maintainability**: Declarative templates are easier to understand and modify
4. **Extensibility**: New package managers and template features easily added
5. **Compatibility**: Existing code continues to work without changes

This change eliminates a major source of complexity and technical debt while providing a foundation for future template system enhancements.