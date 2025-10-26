# Configuration Sync Planning

## Executive Summary

This document outlines the design for template-based configuration synchronization between local development environments and Prism instances, enabling researchers to maintain consistent tool configurations across environments.

## Problem Statement

Researchers spend significant time reconfiguring familiar tools (RStudio, Jupyter, VS Code, etc.) on each new Prism instance. This reduces productivity and creates barriers to cloud adoption. Configuration sync should be:

- **Template-Based**: Configurations stored as shareable templates
- **Application-Aware**: Smart sync for different application types
- **Incremental**: Only sync changed configurations
- **Secure**: Handle sensitive credentials appropriately
- **Cross-Platform**: Work across macOS, Linux, and Windows

## Architecture Overview

### 1. Template-Based Configuration System

**Configuration Templates Structure**:
```yaml
# config-templates/rstudio/data-science.yml
name: "RStudio Data Science Setup"
category: "rstudio"
version: "1.0.0"
author: "researcher@university.edu"
description: "Optimized RStudio configuration for data science workflows"

applications:
  rstudio:
    preferences:
      - source: "~/.config/rstudio/rstudio-prefs.json"
        target: "~/.config/rstudio/rstudio-prefs.json"
        merge_strategy: "replace"

    packages:
      cran:
        - tidyverse
        - ggplot2
        - dplyr
        - shiny
      bioconductor:
        - Biobase
        - limma
      github:
        - "hadley/devtools"

    themes:
      - source: "~/.config/rstudio/themes/"
        target: "~/.config/rstudio/themes/"
        merge_strategy: "merge"

  git:
    config:
      user.name: "{{ user_input:git_name }}"
      user.email: "{{ user_input:git_email }}"
    ssh_keys: "reference_only"  # Don't copy, just reference

environment:
  variables:
    R_LIBS_USER: "/opt/R/library"
    RSTUDIO_PANDOC: "/usr/lib/rstudio/bin/pandoc"

security:
  exclude_patterns:
    - "*.key"
    - "*.pem"
    - "*password*"
    - "*secret*"
  sensitive_prompts:
    - "git_name"
    - "git_email"
```

### 2. Configuration Capture System

**Local Configuration Scanning**:
```bash
# Capture current local configuration
prism config capture rstudio-config --applications rstudio,git
# Creates: config-templates/local/rstudio-config.yml

# Share configuration template
prism config publish rstudio-config --repository community
# Uploads to: community/rstudio/rstudio-config.yml

# Browse available configurations
prism config browse --application rstudio
```

**Smart Configuration Detection**:
```go
// pkg/config/scanner.go
type ConfigurationScanner struct {
    applications map[string]ApplicationScanner
}

type ApplicationScanner interface {
    Name() string
    DetectInstallation() bool
    ScanConfiguration() (*ApplicationConfig, error)
    GetConfigPaths() []ConfigPath
    GetPackages() ([]Package, error)
    GetThemes() ([]Theme, error)
}

type RStudioScanner struct{}

func (r *RStudioScanner) ScanConfiguration() (*ApplicationConfig, error) {
    config := &ApplicationConfig{
        Application: "rstudio",
        Version:     r.getVersion(),
    }

    // Scan preferences
    if prefs, err := r.scanPreferences(); err == nil {
        config.Preferences = prefs
    }

    // Scan installed packages
    if packages, err := r.scanPackages(); err == nil {
        config.Packages = packages
    }

    return config, nil
}
```

### 3. Template-Based Sync Engine

**Sync Command Architecture**:
```bash
# Apply configuration template to instance
prism config apply rstudio-config my-instance
prism config apply rstudio-config my-instance --dry-run
prism config apply rstudio-config my-instance --interactive

# Apply from different sources
prism config apply community/rstudio/data-science my-instance
prism config apply ./local-config.yml my-instance
prism config apply github:university/rstudio-configs/bioinformatics my-instance
```

**Template Processing Engine**:
```go
// pkg/config/sync.go
type ConfigSyncEngine struct {
    templateResolver *TemplateResolver
    applicationSyncs map[string]ApplicationSync
}

type ApplicationSync interface {
    Apply(template *ConfigTemplate, instance string) error
    Validate(template *ConfigTemplate) error
    Preview(template *ConfigTemplate) (*SyncPreview, error)
}

type RStudioSync struct {
    sshClient SSHClientInterface
}

func (r *RStudioSync) Apply(template *ConfigTemplate, instance string) error {
    // 1. Install required packages
    if err := r.installPackages(template.Applications.RStudio.Packages); err != nil {
        return err
    }

    // 2. Apply preferences
    if err := r.applyPreferences(template.Applications.RStudio.Preferences); err != nil {
        return err
    }

    // 3. Copy themes and extensions
    if err := r.applyThemes(template.Applications.RStudio.Themes); err != nil {
        return err
    }

    return nil
}
```

### 4. Repository System

**Configuration Repository Structure**:
```
config-templates/
├── community/           # Community-contributed configs
│   ├── rstudio/
│   │   ├── data-science.yml
│   │   ├── bioinformatics.yml
│   │   └── econometrics.yml
│   ├── jupyter/
│   │   ├── ml-research.yml
│   │   └── python-data.yml
│   └── vscode/
│       ├── python-dev.yml
│       └── r-analysis.yml
├── institutional/       # Institution-specific configs
│   └── university-edu/
│       ├── rstudio-standard.yml
│       └── jupyter-classroom.yml
└── personal/           # User's personal configs
    └── my-rstudio-setup.yml
```

**Template Sharing Commands**:
```bash
# Create template repository
prism config repo init my-lab-configs
prism config repo add-remote origin git@github.com:mylab/cws-configs.git

# Publish configuration
prism config publish my-rstudio-setup --repo my-lab-configs
prism config publish my-rstudio-setup --repo community --public

# Install from repository
prism config install community/rstudio/data-science
prism config install github:mylab/cws-configs/rstudio-setup
prism config install https://raw.githubusercontent.com/mylab/configs/main/rstudio.yml
```

### 5. Application-Specific Implementations

**RStudio Configuration Sync**:
```yaml
# config-templates/rstudio/comprehensive.yml
applications:
  rstudio:
    preferences:
      editor_theme: "Textmate (default)"
      font_size: 12
      soft_wrap: true
      syntax_highlight: true
      show_line_numbers: true

    packages:
      install_method: "renv"  # or "packrat", "direct"
      renv_lockfile: "./renv.lock"

    projects:
      default_settings:
        use_packrat: false
        restore_last_project: true

    keybindings:
      - source: "~/.config/rstudio/keybindings/editor_bindings.json"
        target: "~/.config/rstudio/keybindings/"
```

**Jupyter Configuration Sync**:
```yaml
# config-templates/jupyter/ml-research.yml
applications:
  jupyter:
    extensions:
      lab:
        - "@jupyterlab/git"
        - "@jupyterlab/variableinspector"
        - "jupyterlab-plotly"
      notebook:
        - "jupyter_contrib_nbextensions"

    kernels:
      python:
        - name: "ml-env"
          conda_env: "ml-research"
        - name: "data-analysis"
          conda_env: "data-env"

    configuration:
      - source: "~/.jupyter/jupyter_lab_config.py"
        target: "~/.jupyter/"
      - source: "~/.jupyter/custom/custom.css"
        target: "~/.jupyter/custom/"
```

### 6. Security and Privacy Model

**Sensitive Data Handling**:
```go
// pkg/config/security.go
type SecurityManager struct {
    encryptionKey []byte
    excludePatterns []string
}

func (s *SecurityManager) FilterSensitive(config *ApplicationConfig) *ApplicationConfig {
    filtered := &ApplicationConfig{}

    for _, file := range config.Files {
        if s.isSensitive(file.Path) {
            // Replace with template variable
            filtered.Templates = append(filtered.Templates, TemplateVar{
                Name: s.generateVarName(file.Path),
                Description: fmt.Sprintf("Value from %s", file.Path),
                Type: "secret",
            })
        } else {
            filtered.Files = append(filtered.Files, file)
        }
    }

    return filtered
}
```

**User Prompts for Sensitive Data**:
```bash
🔒 Configuration contains sensitive information
The following values need to be provided:

Git Configuration:
  User Name: [Your Full Name]
  User Email: [your.email@university.edu]

RStudio Server:
  Default CRAN Mirror: [https://cloud.r-project.org/]

Continue with configuration? [y/N]: y
```

### 7. Implementation Phases

**Phase 1: Core Sync Framework (v0.5.3)**
- Basic template schema and validation
- RStudio configuration sync (preferences, packages)
- Local configuration capture
- SSH-based file synchronization

**Phase 2: Template Repository (v0.5.4)**
- Template sharing and discovery
- Community repository integration
- Git-based template storage
- Template versioning and updates

**Phase 3: Multi-Application Support (v0.5.5)**
- Jupyter configuration sync
- VS Code settings and extensions
- Git configuration management
- Vim/Neovim configuration sync

**Phase 4: Advanced Features (v0.5.6)**
- Incremental sync optimization
- Conflict resolution strategies
- Configuration drift detection
- Automated sync on instance launch

## User Experience Flow

### Initial Setup:
```bash
# Capture local RStudio configuration
prism config capture rstudio-setup
✅ Scanned RStudio preferences
✅ Found 45 installed packages
✅ Detected custom themes: 2 files
📝 Configuration saved as: config-templates/personal/rstudio-setup.yml

# Launch instance with configuration
prism launch python-ml my-research --config rstudio-setup
🚀 Launching instance...
⚙️  Applying configuration template: rstudio-setup
   📦 Installing 45 R packages...
   🎨 Applying themes and preferences...
   ⚙️  Configuring keybindings...
✅ Instance ready with synchronized configuration
```

### Daily Workflow:
```bash
# Quick sync to existing instance
prism config sync rstudio-setup my-research
⚙️  Checking for configuration changes...
📦 New packages detected: 3 packages
🔄 Syncing updates...
✅ Configuration synchronized

# Share configuration with team
prism config publish rstudio-setup --repo lab-configs --description "Updated with new bioinformatics packages"
```

## Success Metrics

**Technical Success**:
- Configuration sync success rate >95%
- Sync operation completion time <5 minutes
- Template validation accuracy >99%
- Zero data loss during sync operations

**User Adoption**:
- 70% of users using config sync within 3 months
- Average time to configure new instance reduced by 80%
- Community template contributions >50 templates
- Positive user feedback on ease of use

## Cost Optimization

**Sync Efficiency**:
- Incremental sync reduces data transfer costs
- Template-based approach reduces storage overhead
- Parallel sync operations reduce time costs
- Smart package management reduces compute time

**Template Sharing Benefits**:
- Reduced duplicated configuration effort
- Institutional standardization reduces support costs
- Community contributions accelerate ecosystem growth
- Version control reduces configuration errors

This template-based approach provides a scalable, secure, and user-friendly system for maintaining consistent development environments across local and cloud resources.