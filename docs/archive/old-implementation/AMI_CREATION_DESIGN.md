# AMI Creation System Design

## Overview

Design a system to automatically create Prism AMIs from template definitions, replacing the current hard-coded AMI approach with a dynamic, reproducible pipeline.

## Current Problem

**Hard-coded AMI IDs**: Templates currently use manually created AMIs:
```go
AMI: map[string]map[string]string{
    "us-east-1": {
        "x86_64": "ami-02029c87fa31fb148", // Manual creation
        "arm64":  "ami-050499786ebf55a6a", // Manual maintenance
    },
    // ... 4+ regions × 2 architectures × N templates = lots of manual work
}
```

**Issues with Current Approach:**
- Manual AMI creation across regions/architectures
- No version tracking or reproducibility
- Updates require rebuilding all AMIs manually
- No automated testing of AMI contents
- Security updates require manual propagation

## Target Architecture

### 1. Template Definition System

**Template Structure:**
```yaml
# templates/r-research.yml
name: "R Research Environment"
description: "R + RStudio Server + tidyverse packages"
base_ami:
  source: "ubuntu-22.04-lts"  # Use latest Ubuntu AMI
  architecture: ["x86_64", "arm64"]
regions: ["us-east-1", "us-east-2", "us-west-1", "us-west-2"]

setup_script: |
  #!/bin/bash
  set -euo pipefail
  
  # Update system
  apt-get update -y
  apt-get upgrade -y
  
  # Install R
  apt-get install -y r-base r-base-dev
  
  # Install RStudio Server (architecture-aware)
  ARCH=$(uname -m)
  if [ "$ARCH" = "x86_64" ]; then
      wget https://download2.rstudio.org/server/jammy/amd64/rstudio-server-2023.06.1-524-amd64.deb
      dpkg -i rstudio-server-2023.06.1-524-amd64.deb || true
  elif [ "$ARCH" = "aarch64" ]; then
      wget https://download2.rstudio.org/server/jammy/arm64/rstudio-server-2023.06.1-524-arm64.deb
      dpkg -i rstudio-server-2023.06.1-524-arm64.deb || true
  fi
  
  # Fix any dependency issues
  apt-get install -f -y
  
  # Install R packages
  R -e "install.packages(c('tidyverse','ggplot2','dplyr','readr'), repos='http://cran.rstudio.com/')"
  
  # Configure RStudio
  echo "www-port=8787" >> /etc/rstudio/rserver.conf
  systemctl enable rstudio-server
  systemctl start rstudio-server
  
  # Create default user
  useradd -m -s /bin/bash ubuntu || true
  echo "ubuntu:prism" | chpasswd
  usermod -aG sudo ubuntu
  
  # Cleanup
  apt-get autoremove -y
  apt-get autoclean
  rm -rf /var/lib/apt/lists/*
  rm -f rstudio-server-*.deb
  
  # Mark setup complete
  echo "$(date): R Research Environment setup complete" > /var/log/cws-setup.log

validation_tests:
  - name: "RStudio Server running"
    command: "systemctl is-active rstudio-server"
    expected: "active"
  
  - name: "R packages installed"
    command: "R -e 'packageVersion(\"tidyverse\")'"
    expected_contains: "ℹ"
  
  - name: "User can login"
    command: "id ubuntu"
    expected_contains: "ubuntu"

ports: [22, 8787]
estimated_cost_per_hour:
  x86_64: 0.0464
  arm64: 0.0368
```

### 2. AMI Builder Service

**Core Components:**

```go
// pkg/ami/builder.go
package ami

import (
    "context"
    "fmt"
    
    "github.com/aws/aws-sdk-go-v2/service/ec2"
    "github.com/scttfrdmn/prism/pkg/types"
)

// Builder handles AMI creation from templates
type Builder struct {
    ec2Client *ec2.Client
    region    string
    keyPair   string
    subnet    string
}

// BuildRequest represents an AMI build request
type BuildRequest struct {
    TemplateName string
    Architecture string
    Region       string
    Version      string
    DryRun       bool
}

// BuildResult contains the results of an AMI build
type BuildResult struct {
    AMIID        string
    InstanceID   string
    BuildTime    time.Duration
    Status       string
    LogURL       string
    ValidationResults []ValidationResult
}

// ValidationResult represents a single validation test result
type ValidationResult struct {
    TestName string
    Status   string
    Output   string
    Error    string
}

// AMITemplate represents a parsed template definition
type AMITemplate struct {
    Name         string
    Description  string
    BaseAMI      BaseAMIConfig
    Regions      []string
    SetupScript  string
    Validation   []ValidationTest
    Ports        []int
    CostPerHour  map[string]float64
}

// BuildAMI creates an AMI from a template
func (b *Builder) BuildAMI(ctx context.Context, req BuildRequest) (*BuildResult, error) {
    // 1. Load template definition
    template, err := LoadTemplate(req.TemplateName)
    if err != nil {
        return nil, fmt.Errorf("failed to load template: %w", err)
    }
    
    // 2. Find base AMI for region/architecture
    baseAMI, err := b.findBaseAMI(ctx, template.BaseAMI, req.Region, req.Architecture)
    if err != nil {
        return nil, fmt.Errorf("failed to find base AMI: %w", err)
    }
    
    // 3. Launch temporary instance
    instanceID, err := b.launchInstance(ctx, baseAMI, req.Architecture)
    if err != nil {
        return nil, fmt.Errorf("failed to launch instance: %w", err)
    }
    defer b.cleanupInstance(ctx, instanceID)
    
    // 4. Wait for instance to be ready
    if err := b.waitForInstanceReady(ctx, instanceID); err != nil {
        return nil, fmt.Errorf("instance not ready: %w", err)
    }
    
    // 5. Execute setup script
    if err := b.executeSetupScript(ctx, instanceID, template.SetupScript); err != nil {
        return nil, fmt.Errorf("setup script failed: %w", err)
    }
    
    // 6. Run validation tests
    validationResults, err := b.runValidationTests(ctx, instanceID, template.Validation)
    if err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    // 7. Create AMI
    amiID, err := b.createAMI(ctx, instanceID, req.TemplateName, req.Version)
    if err != nil {
        return nil, fmt.Errorf("AMI creation failed: %w", err)
    }
    
    // 8. Tag AMI with metadata
    if err := b.tagAMI(ctx, amiID, template, req); err != nil {
        return nil, fmt.Errorf("AMI tagging failed: %w", err)
    }
    
    return &BuildResult{
        AMIID:             amiID,
        InstanceID:        instanceID,
        Status:            "success",
        ValidationResults: validationResults,
    }, nil
}
```

### 3. CLI Integration

**New Commands:**
```bash
# Build AMIs
prism ami build r-research --architecture x86_64 --region us-east-1
prism ami build-all r-research  # Build for all regions/architectures
prism ami build-all --templates r-research,python-research

# List built AMIs
prism ami list
prism ami list --template r-research

# Update templates to use latest AMIs
prism ami update-templates

# Validate existing AMIs
prism ami validate r-research --region us-east-1

# Clean up old AMIs
prism ami cleanup --keep-latest 3
```

**Example Build Output:**
```
🏗️ Building AMI for template: r-research
   Architecture: x86_64
   Region: us-east-1
   Version: v1.2.3

⏳ Finding base AMI... ubuntu-22.04-lts/amd64
✅ Base AMI: ami-0abcd1234 (ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-20231201)

⏳ Launching build instance... i-0abcd1234
✅ Instance launched: i-0abcd1234 (t3.medium)

⏳ Waiting for instance ready...
✅ Instance ready (45s)

⏳ Executing setup script...
   📦 Installing R packages...
   📦 Installing RStudio Server...
   🔧 Configuring services...
✅ Setup complete (312s)

⏳ Running validation tests...
   ✅ RStudio Server running
   ✅ R packages installed  
   ✅ User can login
✅ All tests passed

⏳ Creating AMI...
✅ AMI created: ami-0xyz7890

⏳ Tagging AMI...
✅ AMI tagged with metadata

🎉 Build complete!
   AMI ID: ami-0xyz7890
   Total time: 6m 42s
   
Template updated:
   r-research.us-east-1.x86_64 = ami-0xyz7890
```

### 4. Automated Build Pipeline

**GitHub Actions Integration:**
```yaml
# .github/workflows/ami-build.yml
name: Build Prism AMIs

on:
  push:
    paths:
      - 'templates/**'
      - 'scripts/**'
  schedule:
    - cron: '0 2 * * 1'  # Weekly builds for security updates
  workflow_dispatch:
    inputs:
      templates:
        description: 'Templates to build (comma-separated)'
        required: false
        default: 'all'

jobs:
  detect-changes:
    runs-on: ubuntu-latest
    outputs:
      templates: ${{ steps.changes.outputs.templates }}
    steps:
      - uses: actions/checkout@v3
      - name: Detect changed templates
        id: changes
        run: |
          # Detect which templates changed
          if [ "${{ github.event_name }}" = "workflow_dispatch" ]; then
            echo "templates=${{ github.event.inputs.templates }}" >> $GITHUB_OUTPUT
          else
            # Auto-detect from git changes
            echo "templates=r-research,python-research" >> $GITHUB_OUTPUT
          fi

  build-ami:
    needs: detect-changes
    runs-on: ubuntu-latest
    strategy:
      matrix:
        template: ${{ fromJson(needs.detect-changes.outputs.templates) }}
        region: [us-east-1, us-east-2, us-west-1, us-west-2]
        architecture: [x86_64, arm64]
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v3
        with:
          go-version: '1.24'
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: ${{ matrix.region }}
      
      - name: Build Prism
        run: make build
      
      - name: Build AMI
        run: |
          ./bin/cws ami build ${{ matrix.template }} \
            --architecture ${{ matrix.architecture }} \
            --region ${{ matrix.region }} \
            --version $(git rev-parse --short HEAD)
      
      - name: Update template registry
        run: |
          ./bin/cws ami update-registry ${{ matrix.template }} \
            --region ${{ matrix.region }} \
            --architecture ${{ matrix.architecture }}
```

### 5. Template Registry

**Centralized AMI Registry:**
```json
{
  "version": "1.0",
  "last_updated": "2024-06-17T10:30:00Z",
  "templates": {
    "r-research": {
      "version": "v1.2.3",
      "description": "R + RStudio Server + tidyverse packages",
      "amis": {
        "us-east-1": {
          "x86_64": {
            "ami_id": "ami-0abcd1234",
            "created": "2024-06-17T08:30:00Z",
            "validated": true,
            "size_gb": 8
          },
          "arm64": {
            "ami_id": "ami-0efgh5678",
            "created": "2024-06-17T08:45:00Z", 
            "validated": true,
            "size_gb": 8
          }
        },
        "us-east-2": { ... }
      },
      "ports": [22, 8787],
      "estimated_cost_per_hour": {
        "x86_64": 0.0464,
        "arm64": 0.0368
      }
    },
    "python-research": { ... }
  }
}
```

### 6. Benefits of New Architecture

**Automation:**
- ✅ Automatic AMI builds across all regions/architectures
- ✅ Consistent, reproducible environments
- ✅ Automated validation testing
- ✅ Version tracking and rollback capability

**Maintenance:**
- ✅ Easy template updates via YAML files
- ✅ Automated security updates via scheduled builds
- ✅ Centralized AMI registry for version management
- ✅ Cleanup of old AMIs to control costs

**Quality:**
- ✅ Validation tests ensure AMI functionality
- ✅ Consistent environments across regions
- ✅ Traceable build artifacts
- ✅ Standardized setup scripts

**Developer Experience:**
- ✅ Simple template YAML format
- ✅ Local testing with `prism ami build`
- ✅ CI/CD integration for automated builds
- ✅ Clear build logs and error reporting

### 7. Implementation Phases

**Phase 1: Core AMI Builder** (2-3 weeks)
- Template YAML parser
- AMI builder service
- Basic validation framework
- CLI commands for manual builds

**Phase 2: Automation** (1-2 weeks)
- GitHub Actions integration
- Automated template registry updates
- Scheduled security builds
- AMI cleanup automation

**Phase 3: Advanced Features** (2-3 weeks)
- Multi-template dependency management
- Advanced validation tests
- Build artifact caching
- Cost optimization features

### 8. Migration Strategy

**Gradual Migration:**
1. Build new AMI creation system alongside existing hard-coded AMIs
2. Create YAML templates for existing templates
3. Build new AMIs using automated system
4. Update code to use registry-based AMI lookup
5. Retire hard-coded AMI mappings
6. Enable automated builds for ongoing maintenance

**Backward Compatibility:**
- Maintain existing AMI IDs during migration
- Gradual rollout with feature flags
- Fallback to hard-coded AMIs if registry unavailable
- Clear migration timeline and communication

This AMI creation system transforms Prism from a manually-maintained tool into a fully automated, enterprise-grade platform while maintaining its core simplicity for end users.