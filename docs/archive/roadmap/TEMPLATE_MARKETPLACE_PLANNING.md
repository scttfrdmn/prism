# Template Marketplace Planning

## Executive Summary

This document outlines the design for a decentralized template marketplace that enables community-contributed research environments while maintaining security through optional access controls. The marketplace expands beyond the core Prism template repository to support institutional, community, and private template collections.

## Problem Statement

The current Prism template system is limited to a single core repository. Researchers need:

- **Community Templates**: Access to templates created by other researchers
- **Institutional Templates**: Private template repositories for universities/organizations
- **Specialized Templates**: Domain-specific templates not suitable for core repository
- **Template Discovery**: Easy way to find and evaluate available templates
- **Security Controls**: Optional authentication for private/premium templates
- **Version Management**: Track template updates and compatibility

## Architecture Overview

### 1. Decentralized Repository Model

**Repository Types**:
```
Core Repository (prism/templates)
├── Essential research templates
├── Maintained by Prism team
├── High quality standards
└── Always accessible

Community Repositories
├── community/bioinformatics-templates
├── community/economics-research
├── community/machine-learning-advanced
└── community/chemistry-computational

Institutional Repositories
├── university-edu/research-templates
├── national-lab-gov/hpc-templates
├── company-com/proprietary-tools
└── consortium/shared-resources

Private Repositories
├── researcher/personal-templates
├── lab-group/internal-tools
├── startup/commercial-software
└── consultant/premium-templates
```

### 2. Template Registry System

**Registry Architecture**:
```go
// pkg/templates/registry.go
type TemplateRegistry struct {
    repositories map[string]*Repository
    cache        *TemplateCache
    keyManager   *AccessKeyManager
}

type Repository struct {
    Name        string            `json:"name"`
    URL         string            `json:"url"`
    Type        RepositoryType    `json:"type"`
    AuthMethod  AuthMethod        `json:"auth_method"`
    Description string            `json:"description"`
    Verified    bool              `json:"verified"`
    LastUpdated time.Time         `json:"last_updated"`
    Templates   []TemplateMetadata `json:"templates"`
}

type RepositoryType string
const (
    RepositoryTypeCore         RepositoryType = "core"
    RepositoryTypeCommunity    RepositoryType = "community"
    RepositoryTypeInstitutional RepositoryType = "institutional"
    RepositoryTypePrivate      RepositoryType = "private"
)

type AuthMethod string
const (
    AuthMethodNone   AuthMethod = "none"     // Public access
    AuthMethodToken  AuthMethod = "token"    // API token required
    AuthMethodSSH    AuthMethod = "ssh"      // SSH key authentication
    AuthMethodOAuth  AuthMethod = "oauth"    # OAuth flow
)
```

### 3. Template Discovery and Search

**Search Command Interface**:
```bash
# Search all available templates
prism templates search machine-learning
🔍 Searching templates for: machine-learning

Core Repository (prism/templates):
  ✅ Python Machine Learning (Simplified) - Basic ML environment
  ✅ Python ML (GPU) - GPU-accelerated ML with CUDA

Community Repository (community/ml-advanced):
  🌟 PyTorch Research Environment - Advanced PyTorch setup
  🌟 TensorFlow Distributed - Multi-GPU training setup
  🌟 MLOps Pipeline - Full MLOps with MLflow and Airflow

Institutional Repository (university-edu/research):
  🏛️  ML Cluster Integration - University HPC integration
  🔒 Requires authentication: university-edu access key

Private Repository (premium-ml/templates):
  💎 Commercial ML Suite - MATLAB ML + Simulink
  🔒 Requires authentication: premium subscription

# Search specific repository
prism templates search --repo community/ml-advanced pytorch
prism templates search --repo university-edu gpu --auth-key ~/.cws/university.key

# Browse repository contents
prism templates browse community/bioinformatics-templates
📂 Repository: community/bioinformatics-templates
🌟 72 templates available

Categories:
  🧬 Genomics (18 templates)
  🔬 Proteomics (12 templates)
  📊 Phylogenetics (8 templates)
  🔬 Structural Biology (15 templates)
  📈 Biostatistics (19 templates)
```

### 4. Repository Management

**Repository Registration**:
```bash
# Add public community repository
prism templates repo add community/ml-advanced https://github.com/ml-community/cws-templates.git
✅ Added repository: community/ml-advanced
📥 Downloading template metadata...
🔍 Found 24 templates

# Add private institutional repository
prism templates repo add university-edu https://git.university.edu/cws/templates.git --auth ssh
🔐 SSH key authentication required
🔑 Using SSH key: ~/.ssh/id_rsa
✅ Added repository: university-edu
📥 Downloaded 45 private templates

# Add premium repository with token
prism templates repo add premium-ml https://api.premium-templates.com/v1/templates --auth token --key premium-123abc
✅ Added repository: premium-ml
💎 Access granted to 15 premium templates

# List registered repositories
prism templates repo list
📋 Registered Template Repositories:

Core:
  ✅ prism/templates (58 templates) - Always available

Community:
  🌟 community/ml-advanced (24 templates) - Public
  🌟 community/bioinformatics (72 templates) - Public

Institutional:
  🏛️  university-edu (45 templates) - SSH authenticated

Private:
  💎 premium-ml (15 templates) - Token authenticated
  🔒 lab-internal (8 templates) - Token authenticated
```

### 5. Template Metadata and Verification

**Enhanced Template Metadata**:
```yaml
# Templates from external repos include source information
name: "PyTorch Research Environment"
category: "machine-learning"
version: "2.1.0"
author: "ml-research-group"
repository: "community/ml-advanced"
source_url: "https://github.com/ml-community/cws-templates/pytorch-research.yml"

verification:
  signature: "sha256:a1b2c3d4..."
  signed_by: "ml-research-group@university.edu"
  verified: true
  trust_level: "community"

compatibility:
  prism_version: ">=0.5.0"
  required_features: ["gpu", "large-instance"]
  tested_regions: ["us-east-1", "us-west-2", "eu-west-1"]

metrics:
  downloads: 1247
  rating: 4.8
  reviews: 23
  last_tested: "2024-01-15T10:30:00Z"

dependencies:
  external_repos:
    - "community/cuda-base"
  software_licenses:
    - "PyTorch BSD License"
    - "CUDA Toolkit License"
```

### 6. Security and Access Control

**Authentication Methods**:

**Public Repositories (No Auth)**:
```bash
# Public community templates - no authentication required
prism templates search --repo community/open-science
prism launch community/open-science/jupyter-basic my-project
```

**SSH Key Authentication**:
```bash
# SSH-based authentication for institutional repos
prism templates repo add university-edu git@git.university.edu:cws/templates.git --auth ssh
# Uses existing SSH keys from ~/.ssh/

prism launch university-edu/hpc-cluster-access my-research
```

**Token-Based Authentication**:
```bash
# API token for premium/private repositories
prism templates auth set-token premium-ml "premium-api-key-abc123"
prism templates auth set-token lab-internal "lab-token-xyz789"

# Token stored securely in profile keychain
prism launch premium-ml/matlab-optimized my-project
🔐 Authenticating with premium-ml...
✅ Premium license validated
```

**OAuth Flow (Future)**:
```bash
# OAuth for enterprise integrations
prism templates repo add enterprise-corp https://templates.corp.com --auth oauth
🌐 Opening browser for authentication...
✅ Enterprise SSO authentication successful
```

### 7. Template Contribution Workflow

**Publishing Templates**:
```bash
# Contribute to community repository
prism templates publish my-custom-template community/ml-advanced
📤 Preparing template for publication...
🔍 Validating template syntax and dependencies...
📋 Template validation successful
📤 Submitting to community/ml-advanced...
✅ Template published! Pull request created: #123

# Publish to private repository
prism templates publish lab-specific-tool lab-internal
🔐 Authenticating with lab-internal...
📤 Publishing private template...
✅ Template published to private repository
```

**Template Development**:
```bash
# Create new template from existing instance
prism templates create-from-instance my-running-instance custom-r-setup
📸 Capturing instance configuration...
📝 Generating template YAML...
🔍 Template created: templates/custom-r-setup.yml

# Test template before publishing
prism templates test custom-r-setup --dry-run
prism templates test custom-r-setup --launch-test
🧪 Testing template launch...
✅ Template launches successfully
💰 Estimated cost: $0.12/hour
```

### 8. Repository Implementation

**Git-Based Repositories**:
```yaml
# Repository configuration: .cws-repository.yml
repository:
  name: "ML Research Templates"
  type: "community"
  description: "Advanced machine learning research environments"
  maintainer: "ML Research Community"

  access:
    public: true
    auth_methods: ["none"]

  quality:
    review_required: true
    testing_required: true

  categories:
    - "machine-learning"
    - "deep-learning"
    - "computer-vision"
    - "nlp"

templates:
  directory: "templates/"
  schema_version: "v1"
  validation: "strict"
```

**API-Based Repositories**:
```go
// pkg/templates/api_repository.go
type APIRepository struct {
    baseURL    string
    authToken  string
    client     *http.Client
}

func (r *APIRepository) ListTemplates() ([]TemplateMetadata, error) {
    resp, err := r.client.Get(r.baseURL + "/templates")
    if err != nil {
        return nil, err
    }

    var templates []TemplateMetadata
    return templates, json.NewDecoder(resp.Body).Decode(&templates)
}

func (r *APIRepository) GetTemplate(name string) (*Template, error) {
    url := fmt.Sprintf("%s/templates/%s", r.baseURL, name)
    resp, err := r.client.Get(url)
    if err != nil {
        return nil, err
    }

    var template Template
    return &template, json.NewDecoder(resp.Body).Decode(&template)
}
```

### 9. Template Caching and Performance

**Local Template Cache**:
```bash
# Cache management commands
prism templates cache status
📊 Template Cache Status:
Size: 245MB (1,247 templates cached)
Last Update: 2 hours ago
Repositories: 5 active, 2 need updates

prism templates cache update
🔄 Updating template cache...
📥 Downloaded 15 new templates
✅ Cache updated successfully

prism templates cache clean
🧹 Cleaning template cache...
🗑️  Removed 23 old template versions
💾 Freed 45MB of storage
```

**Smart Caching Strategy**:
- Cache frequently used templates locally
- Lazy-load template content on demand
- Automatic cache updates on repository changes
- Configurable cache size limits

### 10. Integration with Core Prism

**Template Launch Integration**:
```bash
# Launch templates from any repository
prism launch pytorch-research my-ml-project  # Searches all repos
prism launch community/ml-advanced/pytorch-research my-ml-project  # Specific repo
prism launch university-edu/hpc-pytorch my-ml-project  # Private repo

# Template info from marketplace
prism templates info community/ml-advanced/pytorch-research
📋 Template: PyTorch Research Environment
Repository: community/ml-advanced ✅ Verified
Author: ML Research Group
Rating: ⭐⭐⭐⭐⭐ (4.8/5, 23 reviews)
Downloads: 1,247 times

Description:
Advanced PyTorch environment with distributed training support,
pre-installed research libraries, and optimized CUDA configuration.

Verification:
✅ Digitally signed by ml-research-group@university.edu
✅ Template tested in 3 AWS regions
✅ Compatible with Prism v0.5.0+

Dependencies:
📦 External: community/cuda-base
💿 Software: PyTorch 2.1, CUDA 12.0
🔧 Features: GPU required, Large instance recommended
```

### 11. Implementation Phases

**Phase 1: Core Marketplace (v0.5.3)**
- Repository registration and management
- Basic template search and discovery
- Git-based repository support
- Public template access (no authentication)

**Phase 2: Authentication (v0.5.4)**
- SSH key authentication for private repos
- Token-based authentication system
- Secure credential storage
- Institutional repository support

**Phase 3: Advanced Features (v0.5.5)**
- Template ratings and reviews
- Advanced search and filtering
- Template contribution workflows
- API-based repository support

**Phase 4: Enterprise Integration (v0.5.6)**
- OAuth authentication flows
- Enterprise policy enforcement
- Template verification and signing
- Advanced security controls

## Success Metrics

**Adoption Metrics**:
- 500+ community templates within 6 months
- 50+ active template contributors
- 10+ institutional repositories
- 80% user adoption of marketplace features

**Quality Metrics**:
- 95% template launch success rate
- Average template rating >4.0/5
- <2 minute template discovery time
- Zero security incidents

**Community Growth**:
- Monthly template contributions >20
- Template download growth >50% quarterly
- Active contributor retention >70%
- Positive community feedback scores

This marketplace architecture provides a scalable foundation for community-driven template development while maintaining the security and reliability standards expected in research environments.