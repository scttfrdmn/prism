# Universal AMI Reference System Planning

## Executive Summary

This document outlines the design for a comprehensive AMI reference system that enables any Prism template to reference pre-built AMIs with intelligent fallback strategies and AMI sharing capabilities, extending far beyond just commercial software to cover any use case where pre-built environments would benefit researchers.

## Problem Statement

The current template system relies exclusively on user_data script provisioning, which has limitations:

- **Launch Time**: 5-8 minutes for complex software installations
- **Reliability**: Script failures can prevent instance launch
- **Network Dependencies**: Package downloads can fail or be slow
- **Cost**: Compute costs during lengthy provisioning
- **Complexity**: Managing dependencies and installation ordering

A universal AMI system should provide:

- **Universal Coverage**: Any template can reference an AMI (not just commercial)
- **Intelligent Fallbacks**: Graceful degradation when AMIs unavailable
- **Fast Launch**: Sub-30 second launch times for pre-built environments
- **AMI Creation**: Generate AMIs from successful template launches
- **AMI Sharing**: Community and institutional AMI distribution
- **Regional Intelligence**: Smart cross-region availability handling

## Architecture Overview

### 1. Universal AMI Reference Architecture

**Template AMI Configuration**:
```yaml
# templates/python-ml-optimized.yml
name: "Python ML (Pre-built)"
category: "machine-learning"
ami_config:
  strategy: "ami_preferred"  # ami_preferred, ami_required, ami_fallback

  # Direct AMI mappings (highest priority)
  ami_mappings:
    us-east-1: "ami-0123456789abcdef0"
    us-west-2: "ami-0fedcba9876543210"
    eu-west-1: "ami-0abcdef123456789a"

  # Dynamic AMI discovery (second priority)
  ami_search:
    owner: "prism-community"  # AWS account ID or alias
    name_pattern: "cws-python-ml-*"
    version_tag: "v2.1.0"
    architecture: ["x86_64", "arm64"]
    min_creation_date: "2024-01-01"

  # Marketplace discovery (third priority)
  marketplace_search:
    product_code: "cws-python-ml-community"
    version_constraint: ">=2.0.0"

  # Fallback behavior when no AMI available
  fallback_strategy: "script_provisioning"  # script_provisioning, error, cross_region
  fallback_timeout: "10m"  # Max time to spend on AMI resolution

  # Cost optimization
  preferred_architecture: "arm64"  # Prefer cheaper ARM when available
  instance_family_preference: ["t4g", "m6i", "c6i"]

# Standard script provisioning as fallback
user_data: |
  #!/bin/bash
  # Fallback installation when AMI unavailable
  yum update -y
  # ... standard installation script
```

### 2. Multi-Tier AMI Resolution Engine

**Resolution Strategy Priority**:
1. **Direct Mapping**: Check ami_mappings for exact region match
2. **Dynamic Search**: Use EC2 DescribeImages with search criteria
3. **Marketplace Lookup**: Query AWS Marketplace for product codes
4. **Cross-Region Discovery**: Search neighboring regions with copy capability
5. **Fallback Execution**: Execute configured fallback strategy

**Implementation**:
```go
// pkg/aws/ami_resolver.go
type UniversalAMIResolver struct {
    ec2Client         EC2ClientInterface
    marketplaceClient MarketplaceClientInterface
    stsClient         STSClientInterface
    regionMapping     map[string][]string  // Region to fallback regions
}

type AMIResolutionResult struct {
    AMI              *AMIInfo
    ResolutionMethod AMIResolutionMethod
    FallbackChain    []string
    Warning          string
    EstimatedCost    float64
    LaunchTime       time.Duration
}

type AMIResolutionMethod string
const (
    ResolutionDirectMapping   AMIResolutionMethod = "direct_mapping"
    ResolutionDynamicSearch   AMIResolutionMethod = "dynamic_search"
    ResolutionMarketplace     AMIResolutionMethod = "marketplace"
    ResolutionCrossRegion     AMIResolutionMethod = "cross_region"
    ResolutionFallbackScript  AMIResolutionMethod = "fallback_script"
    ResolutionFailed          AMIResolutionMethod = "failed"
)

func (r *UniversalAMIResolver) ResolveAMI(template *Template, region string) (*AMIResolutionResult, error) {
    result := &AMIResolutionResult{
        FallbackChain: make([]string, 0),
    }

    // 1. Try direct mapping first (fastest)
    if directAMI := template.AMIConfig.AMIMappings[region]; directAMI != "" {
        if ami, err := r.validateAMI(directAMI, region); err == nil {
            result.AMI = ami
            result.ResolutionMethod = ResolutionDirectMapping
            result.LaunchTime = 30 * time.Second
            return result, nil
        }
        result.FallbackChain = append(result.FallbackChain, "direct_mapping_failed")
    }

    // 2. Try dynamic search
    if template.AMIConfig.AMISearch != nil {
        if ami, err := r.searchAMIByPattern(template.AMIConfig.AMISearch, region); err == nil {
            result.AMI = ami
            result.ResolutionMethod = ResolutionDynamicSearch
            result.LaunchTime = 45 * time.Second
            return result, nil
        }
        result.FallbackChain = append(result.FallbackChain, "dynamic_search_failed")
    }

    // 3. Try marketplace lookup
    if template.AMIConfig.MarketplaceSearch != nil {
        if ami, err := r.lookupMarketplaceAMI(template.AMIConfig.MarketplaceSearch, region); err == nil {
            result.AMI = ami
            result.ResolutionMethod = ResolutionMarketplace
            result.LaunchTime = 60 * time.Second
            result.EstimatedCost = ami.MarketplaceCost
            return result, nil
        }
        result.FallbackChain = append(result.FallbackChain, "marketplace_failed")
    }

    // 4. Try cross-region search with copy
    if template.AMIConfig.FallbackStrategy == "cross_region" {
        if ami, err := r.crossRegionSearch(template, region); err == nil {
            result.AMI = ami
            result.ResolutionMethod = ResolutionCrossRegion
            result.Warning = fmt.Sprintf("AMI copied from %s, additional copy cost applies", ami.SourceRegion)
            result.LaunchTime = 2 * time.Minute  // AMI copy time
            return result, nil
        }
        result.FallbackChain = append(result.FallbackChain, "cross_region_failed")
    }

    // 5. Fallback to script provisioning
    if template.AMIConfig.FallbackStrategy == "script_provisioning" {
        result.ResolutionMethod = ResolutionFallbackScript
        result.Warning = "No pre-built AMI available, using script provisioning (5-8 minutes)"
        result.LaunchTime = 6 * time.Minute  // Script provisioning time
        return result, nil
    }

    // 6. Complete failure
    result.ResolutionMethod = ResolutionFailed
    return result, fmt.Errorf("no AMI resolution strategy succeeded: %v", result.FallbackChain)
}
```

### 3. Cross-Region Intelligence

**Regional Fallback Strategy**:
```go
// pkg/aws/region_mapping.go
var RegionFallbacks = map[string][]string{
    "us-east-1": {"us-east-2", "us-west-2", "us-west-1"},
    "us-west-2": {"us-west-1", "us-east-1", "us-east-2"},
    "eu-west-1": {"eu-west-2", "eu-central-1", "us-east-1"},
    "ap-south-1": {"ap-southeast-1", "ap-northeast-1", "us-east-1"},
}

func (r *UniversalAMIResolver) crossRegionSearch(template *Template, targetRegion string) (*AMIInfo, error) {
    fallbackRegions := RegionFallbacks[targetRegion]

    for _, sourceRegion := range fallbackRegions {
        // Search in source region
        if ami, err := r.searchInRegion(template, sourceRegion); err == nil {
            // Copy AMI to target region
            copiedAMI, err := r.copyAMIToRegion(ami, sourceRegion, targetRegion)
            if err == nil {
                copiedAMI.SourceRegion = sourceRegion
                return copiedAMI, nil
            }
        }
    }

    return nil, fmt.Errorf("no AMI found in fallback regions: %v", fallbackRegions)
}
```

### 4. AMI Creation and Sharing System

**AMI Generation from Templates**:
```bash
# Create AMI from successful template launch
prism ami create python-ml my-instance --name "Python ML v2.1.0" --public
🔧 Creating AMI from instance: my-instance
📸 Creating snapshot of root volume...
🏗️  Building AMI: Python ML v2.1.0
✅ AMI created: ami-0123456789abcdef0

# Share AMI with community
prism ami share ami-0123456789abcdef0 --community prism
✅ AMI shared with prism community

# Publish AMI to marketplace (advanced)
prism ami publish ami-0123456789abcdef0 --marketplace --price 0.05
📤 Submitting AMI to AWS Marketplace...
⏳ Marketplace review process initiated
```

**AMI Management Commands**:
```bash
# List available AMIs for templates
prism ami list --template python-ml
📋 Available AMIs for template: python-ml

Region: us-east-1
  ami-0123456789abcdef0  Python ML v2.1.0   (community)  ⭐ 4.8/5
  ami-0fedcba9876543210  Python ML v2.0.5   (official)   ⭐ 4.6/5

Region: us-west-2
  ami-0abcdef123456789a  Python ML v2.1.0   (community)  ⭐ 4.8/5

# Test AMI availability across regions
prism ami test python-ml --all-regions
🧪 Testing AMI availability for template: python-ml

✅ us-east-1: ami-0123456789abcdef0 (available)
✅ us-west-2: ami-0abcdef123456789a (available)
❌ eu-west-1: No AMI available (fallback: script provisioning)
✅ ap-south-1: ami-0xyz123456789def0 (cross-region copy available)

# Create AMI for multiple regions
prism ami create-multi python-ml my-instance --regions us-east-1,us-west-2,eu-west-1
🌍 Creating AMI in multiple regions...
📸 Creating master AMI in us-east-1...
🔄 Copying to us-west-2... ✅
🔄 Copying to eu-west-1... ✅
✅ Multi-region AMI deployment complete
```

### 5. Community AMI Sharing Architecture

**Community AMI Repository**:
```yaml
# .prism/ami-community.yml
community_amis:
  python-ml:
    v2.1.0:
      creator: "ml-research-group@university.edu"
      description: "Optimized Python ML with CUDA 12.0, PyTorch 2.1"
      regions:
        us-east-1: "ami-0123456789abcdef0"
        us-west-2: "ami-0fedcba9876543210"
      verification:
        tested: true
        security_scan: "passed"
        performance_benchmark: "4.2x faster than script install"
      ratings:
        average: 4.8
        reviews: 23
        downloads: 1247

  r-research:
    v1.5.0:
      creator: "stats-dept@college.edu"
      description: "R 4.3 with tidyverse, RStudio Server pre-configured"
      regions:
        us-east-1: "ami-0abcd1234567890ef"
      verification:
        tested: true
        security_scan: "passed"
```

**AMI Discovery Integration**:
```go
// pkg/ami/community.go
type CommunityAMIRegistry struct {
    registry map[string]map[string]*CommunityAMI
    client   HTTPClient
}

type CommunityAMI struct {
    Version       string            `yaml:"version"`
    Creator       string            `yaml:"creator"`
    Description   string            `yaml:"description"`
    Regions       map[string]string `yaml:"regions"`
    Verification  *AMIVerification  `yaml:"verification"`
    Ratings       *AMIRatings       `yaml:"ratings"`
}

func (r *CommunityAMIRegistry) FindBestAMI(templateName, region string) (*CommunityAMI, error) {
    templates := r.registry[templateName]
    if templates == nil {
        return nil, fmt.Errorf("no community AMIs for template: %s", templateName)
    }

    // Find highest rated, most recent AMI
    var bestAMI *CommunityAMI
    var bestScore float64

    for _, ami := range templates {
        if regionAMI := ami.Regions[region]; regionAMI != "" {
            // Score based on ratings and recency
            score := ami.Ratings.Average * float64(ami.Ratings.Reviews) / 10.0
            if score > bestScore {
                bestScore = score
                bestAMI = ami
            }
        }
    }

    return bestAMI, nil
}
```

### 6. Template Schema Extensions

**Enhanced Template Structure**:
```go
// pkg/templates/types.go
type Template struct {
    Name        string     `yaml:"name" json:"name"`
    Category    string     `yaml:"category" json:"category"`
    AMIConfig   *AMIConfig `yaml:"ami_config,omitempty" json:"ami_config,omitempty"`
    UserData    string     `yaml:"user_data" json:"user_data"`
    // ... existing fields
}

type AMIConfig struct {
    Strategy            AMIStrategy            `yaml:"strategy" json:"strategy"`
    AMIMappings         map[string]string      `yaml:"ami_mappings,omitempty" json:"ami_mappings,omitempty"`
    AMISearch           *AMISearchConfig       `yaml:"ami_search,omitempty" json:"ami_search,omitempty"`
    MarketplaceSearch   *MarketplaceConfig     `yaml:"marketplace_search,omitempty" json:"marketplace_search,omitempty"`
    FallbackStrategy    string                 `yaml:"fallback_strategy" json:"fallback_strategy"`
    FallbackTimeout     string                 `yaml:"fallback_timeout" json:"fallback_timeout"`
    PreferredArch       string                 `yaml:"preferred_architecture" json:"preferred_architecture"`
    InstanceFamilyPref  []string              `yaml:"instance_family_preference" json:"instance_family_preference"`
}

type AMIStrategy string
const (
    AMIStrategyPreferred AMIStrategy = "ami_preferred"    // Try AMI first, fallback to script
    AMIStrategyRequired  AMIStrategy = "ami_required"     // AMI only, fail if unavailable
    AMIStrategyFallback  AMIStrategy = "ami_fallback"     // Script first, AMI if script fails
)
```

### 7. User Experience Flow

**Launch with AMI Intelligence**:
```bash
# Standard launch with automatic AMI resolution
prism launch python-ml my-research
🔍 Resolving AMI for template: python-ml
✅ Found optimized AMI: ami-0123456789abcdef0
📈 Performance: 4.2x faster launch (30s vs 6min)
🚀 Launching with pre-built environment...

# Launch with AMI preference override
prism launch python-ml my-research --prefer-script
⚠️  Script provisioning requested (6 minutes estimated)
🔍 AMI available: ami-0123456789abcdef0 (30 seconds)
Continue with script provisioning? [y/N]: n
✅ Using AMI: ami-0123456789abcdef0

# Launch with regional fallback
prism launch python-ml my-research --region ap-south-1
🔍 Resolving AMI in ap-south-1...
❌ No AMI in ap-south-1
🔄 Searching fallback regions...
✅ Found AMI in ap-southeast-1: ami-0xyz123456789def0
📋 Cross-region copy required (2 minutes + $0.03)
Continue? [y/N]: y

# Show AMI resolution preview
prism launch python-ml my-research --dry-run --show-ami-resolution
🔍 AMI Resolution Preview:

Strategy: ami_preferred
Primary: ami-0123456789abcdef0 (us-east-1) ✅
Fallback Chain:
  1. Direct mapping ✅
  2. Dynamic search (not needed)
  3. Marketplace (not needed)
  4. Script provisioning (not needed)

Estimated Launch Time: 30 seconds
Cost Comparison:
  AMI Launch: $0.45/hour (immediate)
  Script Launch: $0.45/hour + 6min setup ($0.045 setup cost)
```

### 8. Performance Optimization

**AMI Selection Intelligence**:
- **Architecture Preference**: ARM64 over x86_64 for cost optimization
- **Instance Family Matching**: Match AMI optimizations to instance types
- **Regional Cost Awareness**: Consider data transfer costs for cross-region copies
- **Launch Speed Priority**: Favor faster launch times for interactive workloads

**Caching and Precompilation**:
- **Popular Template AMIs**: Auto-create AMIs for frequently used templates
- **Regional Coverage**: Ensure high-usage templates have multi-region AMIs
- **Version Management**: Maintain rolling AMI versions with automatic cleanup
- **Cost Controls**: Balance AMI storage costs with launch speed benefits

### 9. Implementation Phases

**Phase 1: Core AMI Resolution (v0.5.2)**
- Universal AMI reference system for any template
- Multi-tier resolution strategy with intelligent fallbacks
- Cross-region discovery and copy capabilities
- Enhanced template schema with AMI configuration

**Phase 2: AMI Creation and Sharing (v0.5.3)**
- AMI generation from successful template launches
- Community AMI sharing and discovery
- AMI testing and validation across regions
- Performance benchmarking and cost comparison

**Phase 3: Advanced Intelligence (v0.5.4)**
- Community AMI registry with ratings and reviews
- Automated AMI creation for popular templates
- Advanced cost optimization with regional intelligence
- Integration with template marketplace for AMI distribution

**Phase 4: Enterprise Features (v0.5.5)**
- Institutional AMI repositories and sharing policies
- AMI signing and verification for security
- Automated AMI updates and security patching
- Compliance reporting and audit trails

## Benefits for Research Computing

**Performance Benefits**:
- **30-second launches** vs 5-8 minute script provisioning
- **Reliable deployments** - pre-tested environments eliminate script failures
- **Cost optimization** - reduced compute time during provisioning
- **Bandwidth efficiency** - no repeated package downloads

**Community Benefits**:
- **Knowledge sharing** - researchers can share optimized environments
- **Institutional templates** - universities can maintain standard AMIs
- **Version control** - track and roll back to previous environment versions
- **Collaboration** - teams can standardize on shared AMI-based templates

**Operational Benefits**:
- **Regional resilience** - automatic cross-region fallbacks
- **Cost transparency** - clear cost comparison between AMI and script approaches
- **Smart defaults** - intelligent architecture and instance type selection
- **Graceful degradation** - always have a working fallback path

This universal AMI system transforms Prism from a script-provisioning platform into a hybrid system that intelligently chooses the fastest, most reliable deployment method while maintaining backward compatibility and providing graceful fallbacks for any scenario.