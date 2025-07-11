# CloudWorkstation 0.3.0 Release Plan

This document outlines the development plan for CloudWorkstation 0.3.0, focusing on three major feature areas:

1. Research Domain Template Expansion
2. Idle Detection System
3. Multi-Repository Support

## 1. Research Domain Template Expansion

**Goal**: Expand template coverage from current basic templates to 24 research domains based on AWS Research Wizard project.

### Implementation Plan

1. **Template Format Extension**
   - Enhance YAML format to include domain-specific metadata
   - Add fields for research type, common use cases, and recommended instance types
   - Document extended format in `docs/TEMPLATE_FORMAT.md`

2. **Domain Categories Implementation**
   - **Life Sciences**
     - Genomics & Bioinformatics
     - Structural Biology
     - Systems Biology
     - Neuroscience & Brain Imaging
     - Drug Discovery
   - **Physical Sciences**
     - Climate Science & Atmospheric Physics
     - Materials Science & Computational Chemistry
     - Physics Simulation
     - Astronomy & Astrophysics
     - Geoscience
   - **Engineering**
     - Computational Fluid Dynamics (CFD)
     - Mechanical Engineering
     - Electrical Engineering
     - Aerospace Engineering
   - **Computer Science & AI**
     - Machine Learning & AI
     - HPC Development
     - Data Science
     - Quantum Computing
   - **Social Sciences & Humanities**
     - Digital Humanities
     - Economics Analysis
     - Social Science Research
   - **Interdisciplinary**
     - Mathematical Modeling
     - Visualization Studio
     - Research Workflow Management

3. **Template Migration Process**
   - Analyze existing AWS Research Wizard templates
   - Convert to CloudWorkstation YAML format
   - Create build scripts and validation tests for each template
   - Implement appropriate resource profiles

4. **Priority Implementation Order**
   - Priority 1: Machine Learning & AI, Data Science, Genomics & Bioinformatics, Visualization Studio
   - Priority 2: Neuroscience, Climate Science, Digital Humanities, Physics Simulation
   - Priority 3: Remaining domains

### Technical Specifications

- Each template will include:
  - Base OS image
  - Software installation scripts
  - Validation tests
  - Resource recommendations by T-shirt size
  - Cost estimates
  - Example workflows

- Directory structure:
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

## 2. Idle Detection System

**Goal**: Implement a CloudSnooze-inspired idle detection system to optimize costs while respecting research workflows.

### Implementation Plan

1. **Core Idle Detection Engine**
   - Create `pkg/idle/` package
   - Implement multi-metric monitoring:
     - CPU usage (threshold: 10%)
     - Memory usage (threshold: 30%)
     - Network traffic (threshold: 50 KBps)
     - Disk I/O (threshold: 100 KBps)
     - GPU usage (threshold: 5%)
   - Separate from budget management features
   - User-configurable thresholds

2. **Domain-Specific Profiles**
   - Create idle profiles for each research domain
   - Define appropriate thresholds based on workload patterns
   - Allow users to select/customize profiles

3. **User Interface**
   - CLI commands for idle detection configuration
   - TUI integration for visual monitoring
   - Notification system for idle warnings

4. **Integration with AWS**
   - Add idle state tracking via instance tags
   - Implement automated stop/hibernate actions
   - Create detailed idle history for analytics

### Technical Specifications

- Idle detection configuration in `~/.cloudworkstation/idle.json`:
```json
{
  "enabled": true,
  "default_profile": "standard",
  "profiles": {
    "standard": {
      "cpu_threshold": 10,
      "memory_threshold": 30,
      "network_threshold": 50,
      "disk_threshold": 100,
      "gpu_threshold": 5,
      "idle_minutes": 30,
      "action": "stop"
    },
    "batch": {
      "cpu_threshold": 5,
      "memory_threshold": 20,
      "idle_minutes": 60,
      "action": "hibernate"
    }
  },
  "domain_mappings": {
    "ml": "standard",
    "genomics": "batch"
  },
  "instance_overrides": {
    "my-gpu-instance": {
      "profile": "gpu",
      "idle_minutes": 15
    }
  }
}
```

- CloudWorkstation daemon will run the idle detection service
- User notification options: email, webhook, desktop notification

## 3. Multi-Repository Support

**Goal**: Support multiple template repositories with override capabilities for organizational customization.

### Implementation Plan

1. **Repository Structure**
   - Create default repository at `github.com/scttfrdmn/cloudworkstation-repository`
   - Define standard repository layout and metadata
   - Implement repository specification format

2. **Multi-Repository Configuration**
   - Add configuration in `~/.cloudworkstation/config.json`:
   ```json
   {
     "repositories": [
       {
         "name": "default",
         "url": "github.com/scttfrdmn/cloudworkstation-repository",
         "priority": 1
       },
       {
         "name": "organizational",
         "url": "github.com/myorg/templates",
         "priority": 2
       }
     ]
   }
   ```
   - Repository priority determines override behavior (higher number = higher priority)

3. **Repository Management Commands**
   - `cws repo add <name> <url> [--priority N]`
   - `cws repo remove <name>`
   - `cws repo list`
   - `cws repo update [name]`
   - `cws repo info <name>`

4. **Template Resolution Logic**
   - Search for template in all repositories by priority
   - Allow explicit repo specification: `cws launch repo:template name`
   - Support template version pinning: `cws launch template@1.2 name`

### Technical Specifications

- Repository metadata in `repository.yaml`:
```yaml
name: "Default Repository"
description: "Official CloudWorkstation template repository"
maintainer: "CloudWorkstation Team"
website: "https://github.com/scttfrdmn/cloudworkstation"
templates:
  - name: "r-research"
    path: "domains/data-science/r-research.yaml"
    versions:
      - version: "1.0.0"
        date: "2025-07-01"
      - version: "1.1.0"
        date: "2025-08-15"
  - name: "python-ml"
    path: "domains/computer-science/python-ml.yaml"
    versions:
      - version: "1.0.0"
        date: "2025-07-01"
```

- Local cache at `~/.cloudworkstation/repositories/`
- Automatic update checking with configurable frequency

## Implementation Timeline

### Week 1-2: Research Domain Templates
- Create extended YAML format documentation
- Implement first 4 priority domains
- Create template conversion workflow
- Update CLI to support domain categories

### Week 3-4: Idle Detection System
- Develop idle detection engine
- Create domain-specific profiles
- Implement CLI and TUI integration
- Add AWS integration for actions

### Week 5-6: Multi-Repository Support
- Create default repository structure
- Implement repository configuration
- Add repository management commands
- Develop template resolution logic

### Week 7-8: Integration and Testing
- Comprehensive testing across all features
- Documentation updates
- Performance optimization
- User experience improvements

## Success Criteria

1. **Template Coverage**
   - 24 research domains implemented
   - All templates validated on multiple instance types
   - Documentation for each domain

2. **Idle Detection**
   - Reduces average monthly costs by 30%+
   - False positives < 1%
   - User satisfaction with notifications > 90%

3. **Multi-Repository**
   - Support for organizational template customization
   - Seamless override behavior
   - Repository update performance < 5 seconds

## Documentation Requirements

1. Update `docs/TEMPLATE_FORMAT.md` with extended format
2. Create `docs/IDLE_DETECTION.md` with complete system documentation
3. Create `docs/REPOSITORIES.md` explaining multi-repository system
4. Update README.md with new features
5. Create domain-specific documentation for each research domain

## Testing Strategy

1. **Automated Tests**
   - Unit tests for all new packages
   - Integration tests for repository resolution
   - Performance tests for idle detection

2. **User Testing**
   - Recruit 5+ researchers from different domains
   - Focused testing of domain-specific templates
   - Idle detection behavior validation

## Release Checklist

1. All features implemented and tested
2. Documentation complete
3. Version updated in code and Makefile
4. CHANGELOG.md updated
5. GitHub release created with release notes
6. Default repository deployed