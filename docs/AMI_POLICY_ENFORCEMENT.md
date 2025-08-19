# AMI Policy Enforcement: Compiled Template Architecture

## Overview

This document outlines the architecture for treating **AMIs as "compiled" forms of templates**, enabling unified policy enforcement across both YAML templates and their pre-built AMI variants. This approach provides performance benefits (faster launches) while maintaining governance and traceability.

## Core Concept: Templates → AMIs → Policy Enforcement

```
YAML Template (Source)  →  Compilation Process  →  AMI (Compiled)  →  Policy Enforcement
     ↓                          ↓                      ↓                    ↓
python-ml.yml           →   Template Build      →  ami-abc123def    →  Same policies apply
- packages: numpy       →   Install & Configure →  (numpy installed) →  Template whitelist
- policy_metadata       →   Embed metadata      →  + template metadata → Cost limits
- cost_limits           →   Cost calculation    →  + cost estimates   → Regional restrictions
```

## Technical Architecture

### **Template Compilation System**

```go
// TemplateCompiler converts YAML templates to AMIs with embedded metadata
type TemplateCompiler struct {
    ec2Client     *ec2.Client
    ssmClient     *ssm.Client
    imageBuilder  *imagebuilder.Client
    
    // Compilation configuration
    buildRegions     []string  // Regions to build AMIs
    buildArchs       []string  // Architectures to support
    builderInstance  string    // Instance type for building
}

// CompilationRequest represents a template compilation job
type CompilationRequest struct {
    Template        *Template         `json:"template"`
    OutputName      string           `json:"output_name"`
    Regions         []string         `json:"regions"`
    Architectures   []string         `json:"architectures"`
    PolicyMetadata  *PolicyMetadata  `json:"policy_metadata"`
}

// CompilationResult contains the compiled AMI information
type CompilationResult struct {
    SourceTemplate   string                         `json:"source_template"`
    CompilationID    string                         `json:"compilation_id"`
    AMIs            map[string]map[string]string   `json:"amis"`  // region -> arch -> AMI ID
    PolicyMetadata  *PolicyMetadata                `json:"policy_metadata"`
    CompiledAt      time.Time                      `json:"compiled_at"`
    BuildDuration   time.Duration                  `json:"build_duration"`
}
```

### **AMI Metadata Embedding**

```go
// PolicyMetadata embedded in AMI tags and user data
type PolicyMetadata struct {
    // Source traceability
    SourceTemplate     string    `json:"source_template"`     // Original template name
    TemplateVersion    string    `json:"template_version"`    // Template version/hash
    CompilationID      string    `json:"compilation_id"`      // Unique build ID
    CompiledAt         time.Time `json:"compiled_at"`         // Build timestamp
    
    // Policy enforcement data
    PolicySignature    string              `json:"policy_signature,omitempty"`    // Enterprise feature
    AllowedProfiles    []string            `json:"allowed_profiles,omitempty"`    // Profile restrictions
    CostEstimates      map[string]float64  `json:"cost_estimates"`                // arch -> hourly cost
    ResourceLimits     *ResourceLimits     `json:"resource_limits,omitempty"`     // Instance constraints
    
    // Classification and governance
    DataClassification string   `json:"data_classification,omitempty"` // public, internal, confidential
    ComplianceFrameworks []string `json:"compliance_frameworks,omitempty"` // HIPAA, SOX, etc.
    ApprovalRequired   bool     `json:"approval_required,omitempty"`    // Requires explicit approval
}

// ResourceLimits defines constraints embedded in AMI
type ResourceLimits struct {
    MaxInstanceTypes   []string `json:"max_instance_types,omitempty"`   // Allowed instance types
    ForbiddenRegions   []string `json:"forbidden_regions,omitempty"`    // Regional restrictions  
    MaxHourlyCost      float64  `json:"max_hourly_cost,omitempty"`      // Cost ceiling
    RequiredTags       []string `json:"required_tags,omitempty"`        // Must-have tags
}
```

### **AMI Policy Validation Engine**

```go
// AMIPolicyValidator validates AMI launches against embedded and profile policies
type AMIPolicyValidator struct {
    ec2Client      *ec2.Client
    profileManager *profile.ManagerEnhanced
}

// ValidateAMILaunch checks if AMI launch is permitted by policies
func (v *AMIPolicyValidator) ValidateAMILaunch(amiID, region string, launchConfig *LaunchConfig) (*ValidationResult, error) {
    // 1. Retrieve AMI metadata from tags
    amiMetadata, err := v.getAMIMetadata(amiID, region)
    if err != nil {
        return nil, fmt.Errorf("failed to retrieve AMI metadata: %w", err)
    }
    
    // 2. Get current profile restrictions
    profile, err := v.profileManager.GetCurrentProfile()
    if err != nil {
        return nil, fmt.Errorf("failed to get current profile: %w", err)
    }
    
    // 3. Validate AMI against profile policy
    violations := []string{}
    
    // Check if AMI source template is allowed
    if profile.PolicyRestrictions != nil {
        if !profile.PolicyRestrictions.IsTemplateAllowed(amiMetadata.SourceTemplate) {
            violations = append(violations, 
                fmt.Sprintf("AMI source template '%s' not allowed by profile policy", 
                    amiMetadata.SourceTemplate))
        }
    }
    
    // Check embedded resource limits
    if amiMetadata.ResourceLimits != nil {
        if !amiMetadata.ResourceLimits.IsInstanceTypeAllowed(launchConfig.InstanceType) {
            violations = append(violations, 
                fmt.Sprintf("Instance type '%s' not allowed by AMI policy", 
                    launchConfig.InstanceType))
        }
        
        if !amiMetadata.ResourceLimits.IsRegionAllowed(region) {
            violations = append(violations, 
                fmt.Sprintf("Region '%s' not allowed by AMI policy", region))
        }
    }
    
    // Check cost limits
    estimatedCost := v.calculateInstanceCost(launchConfig.InstanceType, region)
    if amiMetadata.ResourceLimits != nil && amiMetadata.ResourceLimits.MaxHourlyCost > 0 {
        if estimatedCost > amiMetadata.ResourceLimits.MaxHourlyCost {
            violations = append(violations, 
                fmt.Sprintf("Estimated cost $%.2f exceeds AMI limit $%.2f", 
                    estimatedCost, amiMetadata.ResourceLimits.MaxHourlyCost))
        }
    }
    
    return &ValidationResult{
        Allowed:       len(violations) == 0,
        Violations:    violations,
        AMIMetadata:   amiMetadata,
        EstimatedCost: estimatedCost,
    }, nil
}
```

## Template System Integration

### **Enhanced Template Types**

```go
// Enhanced Template struct with compilation support
type Template struct {
    // ... existing fields ...
    
    // Compilation configuration
    CompileToAMI    *CompilationConfig `yaml:"compile_to_ami,omitempty" json:"compile_to_ami,omitempty"`
    
    // Policy enforcement metadata
    PolicyMetadata  *PolicyMetadata    `yaml:"policy_metadata,omitempty" json:"policy_metadata,omitempty"`
    
    // AMI references (for pre-compiled templates)
    PrecompiledAMIs map[string]map[string]string `yaml:"precompiled_amis,omitempty" json:"precompiled_amis,omitempty"`
}

type CompilationConfig struct {
    Enabled       bool     `yaml:"enabled" json:"enabled"`
    Regions       []string `yaml:"regions" json:"regions"`           // Target regions
    Architectures []string `yaml:"architectures" json:"architectures"` // x86_64, arm64
    BuildInstance string   `yaml:"build_instance,omitempty" json:"build_instance,omitempty"` // Builder instance type
    
    // Advanced options
    CustomScript  string            `yaml:"custom_script,omitempty" json:"custom_script,omitempty"`
    BuildTags     map[string]string `yaml:"build_tags,omitempty" json:"build_tags,omitempty"`
    Encrypted     bool              `yaml:"encrypted,omitempty" json:"encrypted,omitempty"`
}
```

### **Template Example with AMI Compilation**

```yaml
# Source template with compilation configuration
name: "Python Machine Learning (Compiled)"
description: "Pre-built ML environment for faster launch times"
base: "ubuntu-22.04"

# Traditional package installation (for source builds)
packages:
  system: ["python3", "python3-pip", "git", "htop"]
  conda: ["numpy", "pandas", "scikit-learn", "jupyter", "matplotlib"]

# Compilation configuration
compile_to_ami:
  enabled: true
  regions: ["us-west-2", "us-east-1", "eu-west-1"]
  architectures: ["x86_64", "arm64"]
  build_instance: "t3.large"  # Instance type for building
  encrypted: true

# Pre-compiled AMIs (populated after compilation)
precompiled_amis:
  us-west-2:
    x86_64: "ami-0abc123def456789a"
    arm64:  "ami-0def456abc789012b"
  us-east-1:
    x86_64: "ami-0456789abc012def3"
    arm64:  "ami-0789012def345abc4"

# Policy metadata embedded in AMI
policy_metadata:
  data_classification: "internal"
  compliance_frameworks: ["university_policy"]
  resource_limits:
    max_instance_types: ["t3.medium", "t3.large", "c5.large", "m5.large"]
    max_hourly_cost: 0.20
    forbidden_regions: ["us-gov-west-1"]

# Instance defaults
instance_defaults:
  type: "t3.medium"
  ports: [22, 8888]  # SSH + Jupyter
  estimated_cost_per_hour:
    x86_64: 0.0464
    arm64:  0.0371  # ARM instances cheaper
```

## Policy Enforcement Examples

### **Basic Policy Framework (Open Source)**

```bash
# Template whitelist applies to both YAML templates and their compiled AMIs
cws profiles invitations create "CS101 Class" \
  --template-whitelist "python-basic,python-ml-compiled" \
  --max-hourly-cost 0.15

# Student launch - AMI inherits template restrictions
cws launch python-ml-compiled my-homework
# → Policy check: Source template 'python-ml' is in whitelist ✓
# → Policy check: AMI embedded cost $0.0464 < limit $0.15 ✓  
# → Policy check: Using pre-compiled AMI for faster launch ✓
# → Launch approved (ami-0abc123def456789a)

# Student tries expensive instance - blocked by AMI embedded limits
cws launch python-ml-compiled my-project --instance-type c5.4xlarge
# → Policy check: Instance type not in AMI resource limits ✗
# → Error: AMI 'python-ml-compiled' restricts instance types to: [t3.medium, t3.large, c5.large, m5.large]
```

### **Enterprise Policy Framework (Proprietary)**

```bash
# Enterprise deployment with AMI signature verification
cws launch institutional-python-ml research-project
# → Policy check: AMI signature verified ✓ (institutional key)
# → Policy check: Compliance frameworks match ✓ (HIPAA approved)
# → Policy check: User security clearance sufficient ✓ (internal data)
# → Policy check: Template source approved ✓ (IT-signed template)
# → Launch approved with audit log entry

# Unauthorized AMI - blocked by signature verification
cws launch external-ami-12345 test-project
# → Policy check: AMI signature missing or invalid ✗
# → Error: AMI not approved by institutional policy
# → Contact IT for AMI approval process
```

## CLI Integration

### **Enhanced Template Commands**

```bash
# List templates shows both source and compiled variants
cws templates list
# TEMPLATE                    TYPE      STATUS      LAUNCH TIME
# python-basic               source    ready       ~3-5 minutes
# python-ml                  source    ready       ~5-8 minutes  
# python-ml-compiled         compiled  ready       ~30 seconds
# r-research                 source    ready       ~4-6 minutes
# r-research-compiled        compiled  building    ETA: 15 minutes

# Template info shows compilation status and AMI details
cws templates info python-ml-compiled
# Template: Python Machine Learning (Compiled)
# Type: Compiled (AMI-based)
# Source Template: python-ml-v2.1
# Compilation Status: Complete
# 
# Available AMIs:
#   us-west-2: ami-0abc123def456789a (x86_64), ami-0def456abc789012b (arm64)
#   us-east-1: ami-0456789abc012def3 (x86_64), ami-0789012def345abc4 (arm64)
# 
# Embedded Policy Restrictions:
#   - Max instance types: t3.medium, t3.large, c5.large, m5.large
#   - Max hourly cost: $0.20
#   - Forbidden regions: us-gov-west-1
# 
# Launch Performance: ~30 seconds (vs ~5-8 minutes for source template)

# Compile templates on-demand
cws templates compile python-ml --regions us-west-2,eu-west-1 --architectures x86_64,arm64
# → Initiating template compilation...
# → Building AMI in us-west-2 (x86_64): ami-build-0abc123
# → Building AMI in us-west-2 (arm64): ami-build-0def456  
# → Building AMI in eu-west-1 (x86_64): ami-build-0789abc
# → Building AMI in eu-west-1 (arm64): ami-build-0abc789
# → Estimated completion: 20-25 minutes

# Check compilation status
cws templates compile status python-ml
# Compilation Status: In Progress
# Started: 2024-01-15 10:30:00 UTC
# Progress:
#   ✓ us-west-2 (x86_64): ami-0abc123def456789a - Complete
#   ⏳ us-west-2 (arm64): 85% - Installing conda packages
#   ⏳ eu-west-1 (x86_64): 45% - Installing system packages  
#   ⏳ eu-west-1 (arm64): 30% - Configuring base system
```

## Benefits and Use Cases

### **Performance Benefits**
- **Faster Launches**: AMI-based launches in 30 seconds vs 5-8 minutes for package installation
- **Reliability**: Pre-tested, known-good configurations
- **Consistency**: Identical environments across regions and architectures

### **Governance Benefits**  
- **Traceability**: Every AMI links back to source template and build process
- **Policy Enforcement**: Same restrictions apply to templates and compiled AMIs
- **Audit Trail**: Complete provenance from source template to running instance
- **Institutional Control**: AMI approval workflows parallel template approval

### **Educational Use Cases**
```bash
# CS department pre-compiles class templates for faster student access
cws templates compile python-basic --batch-compile class-templates
# → Students get 30-second launch times instead of 5-minute waits
# → Same policy restrictions apply (cost limits, instance types)
# → Consistent environment across all student instances

# Research lab compiles specialized templates for GPU workloads
cws templates compile deep-learning-gpu --regions us-west-2 --instance-types p3.2xlarge
# → Lab members get immediate access to complex ML environments
# → Pre-installed CUDA, PyTorch, TensorFlow, research-specific libraries
# → Policy-enforced cost and instance type restrictions maintained
```

### **Enterprise Use Cases**
```bash  
# IT department maintains approved AMI catalog
cws templates compile institutional-python --sign-with university-it-key
# → Creates digitally signed AMIs for institutional deployment
# → Embedded compliance metadata (HIPAA, SOX, university policies)
# → Automatic security patching and vulnerability scanning

# Department budgets control AMI usage
cws profiles create chemistry-dept --ami-whitelist "chem-analysis-v2.1,molecular-modeling-v1.3"
# → Department members restricted to approved AMIs only
# → Cost controls and budget tracking apply to AMI-based launches
# → Template governance extends to compiled AMI governance
```

This AMI-as-compiled-template architecture provides the performance benefits of pre-built images while maintaining the governance, traceability, and policy enforcement that makes CloudWorkstation valuable for institutional deployments. The unified policy framework ensures consistent controls whether users launch from source templates or compiled AMIs.