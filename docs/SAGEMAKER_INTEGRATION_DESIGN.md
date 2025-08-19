# SageMaker Studio Integration Design

## Overview

This document outlines the technical architecture for integrating AWS SageMaker Studio into CloudWorkstation as the first web-based research service, serving as the proof of concept for Phase 5B AWS Research Services Integration.

## Architecture Components

### **Service Type Extension**

```go
// Enhanced service types including SageMaker variants
type ServiceType string
const (
    ServiceTypeEC2           ServiceType = "ec2"              // Traditional instances
    ServiceTypeSageMakerLab  ServiceType = "sagemaker_lab"    // Studio Lab (free)
    ServiceTypeSageMaker     ServiceType = "sagemaker_studio" // Studio (managed)
    ServiceTypeSageMakerCanvas ServiceType = "sagemaker_canvas" // Canvas (no-code)
)

// Enhanced connection types for web services
type ConnectionType string
const (
    ConnectionTypeSSH  ConnectionType = "ssh"  // Traditional SSH access
    ConnectionTypeDCV  ConnectionType = "dcv"  // Remote desktop
    ConnectionTypeWeb  ConnectionType = "web"  // Direct web browser access
    ConnectionTypeAPI  ConnectionType = "api"  // API-only access
)
```

### **SageMaker Configuration Model**

```go
// SageMakerConfig represents SageMaker-specific configuration
type SageMakerConfig struct {
    // Studio Lab configuration (free tier)
    StudioLabConfig *StudioLabConfig `json:"studio_lab_config,omitempty"`
    
    // Studio managed configuration
    StudioConfig *StudioConfig `json:"studio_config,omitempty"`
    
    // Canvas configuration
    CanvasConfig *CanvasConfig `json:"canvas_config,omitempty"`
}

type StudioLabConfig struct {
    // Studio Lab is pre-configured, minimal options
    ProjectName string `json:"project_name"`
    Runtime     string `json:"runtime"` // "Python 3", "R", "PyTorch", etc.
}

type StudioConfig struct {
    // Domain configuration
    DomainID       string `json:"domain_id,omitempty"`       // Auto-create if empty
    DomainName     string `json:"domain_name"`
    
    // User profile configuration  
    UserProfileName string `json:"user_profile_name"`
    
    // Instance configuration
    DefaultInstanceType string   `json:"default_instance_type"` // ml.t3.medium
    InstanceTypes       []string `json:"instance_types"`        // Allowed types
    
    // Storage and networking
    EFSIntegration bool   `json:"efs_integration,omitempty"` // Mount CloudWorkstation EFS
    VPCConfig      *VPCConfig `json:"vpc_config,omitempty"`  // Custom VPC
    
    // Cost controls
    MaxHourlyCost   float64 `json:"max_hourly_cost,omitempty"`
    AutoStopMinutes int     `json:"auto_stop_minutes,omitempty"`
}

type CanvasConfig struct {
    // Canvas-specific configuration
    DataSources []string `json:"data_sources,omitempty"` // S3 buckets, databases
    ModelTypes  []string `json:"model_types,omitempty"`  // Allowed model types
}
```

### **Enhanced Instance Model**

```go
// Enhanced Instance struct to support web services
type Instance struct {
    // Existing fields...
    ID       string `json:"id"`
    Name     string `json:"name"`
    Template string `json:"template"`
    State    string `json:"state"`
    
    // Service type extensions
    ServiceType    ServiceType     `json:"service_type"`          // ec2, sagemaker_studio, etc.
    ConnectionType ConnectionType  `json:"connection_type"`       // ssh, web, dcv
    ServiceConfig  interface{}     `json:"service_config,omitempty"` // Service-specific config
    
    // Web service specific fields
    WebURL         string `json:"web_url,omitempty"`         // Direct access URL
    AccessMethods  []string `json:"access_methods"`          // ["web", "api"]
    
    // Cost tracking
    ServiceCosts   *ServiceCosts `json:"service_costs,omitempty"` // Multi-service cost breakdown
}

type ServiceCosts struct {
    ComputeCost    float64 `json:"compute_cost"`     // Instance/compute costs
    StorageCost    float64 `json:"storage_cost"`     // EFS/EBS storage costs  
    ServiceCost    float64 `json:"service_cost"`     // SageMaker service costs
    TotalHourlyCost float64 `json:"total_hourly_cost"` // Combined cost
}
```

## Template Integration

### **SageMaker Studio Template Examples**

#### **SageMaker Studio Lab Template (Free Tier)**
```yaml
name: "SageMaker Studio Lab - Python ML"
description: "Free machine learning environment with Jupyter notebooks"
connection_type: "web"
service_type: "sagemaker_lab"

# No AMI needed for web services - uses AWS managed infrastructure
service_config:
  studio_lab:
    runtime: "Python 3"
    project_name: "ml-research"

# Policy restrictions still apply
instance_defaults:
  estimated_cost_per_hour:
    any: 0.00  # Studio Lab is free

# Template metadata for policy enforcement
policy_metadata:
  requires_free_tier: true
  data_classification: "public"
  suitable_for: ["education", "learning", "prototyping"]
```

#### **SageMaker Studio Managed Template**
```yaml
name: "SageMaker Studio - GPU ML Research"
description: "Managed ML environment with GPU support and custom instances"
connection_type: "web" 
service_type: "sagemaker_studio"

service_config:
  studio:
    domain_name: "research-ml-domain"
    user_profile_name: "from_research_user"  # Links to research user identity
    default_instance_type: "ml.t3.medium"
    instance_types: ["ml.t3.medium", "ml.t3.large", "ml.g4dn.xlarge"]
    efs_integration: true  # Mount CloudWorkstation EFS
    auto_stop_minutes: 30  # Cost optimization
    max_hourly_cost: 5.00  # Budget control

# Policy restrictions
instance_defaults:
  estimated_cost_per_hour:
    ml.t3.medium: 0.05
    ml.t3.large: 0.09  
    ml.g4dn.xlarge: 0.736

policy_metadata:
  requires_gpu: false
  data_classification: "internal"
  suitable_for: ["research", "development", "training"]
```

### **AMI-as-Compiled-Template Integration**

For SageMaker services, the "compilation" process creates **SageMaker Images** instead of AMIs:

```yaml
# Template compilation for SageMaker
name: "Custom PyTorch Research Environment"
base: "sagemaker-pytorch-base"
connection_type: "web"
service_type: "sagemaker_studio"

# Traditional package installation becomes SageMaker Image creation
packages:
  conda: ["transformers", "datasets", "wandb", "tensorboard"]
  pip: ["custom-research-lib==1.2.0"]

# Compilation creates SageMaker Image, not AMI
compile_to_image:
  image_name: "custom-pytorch-research"
  base_image: "763104351884.dkr.ecr.us-west-2.amazonaws.com/pytorch-training:1.13.1-gpu-py39"
  
# Policy enforcement applies to both templates and compiled images
policy_metadata:
  source_template: "custom-pytorch-research-v1.0"
  image_uri: "123456789012.dkr.ecr.us-west-2.amazonaws.com/custom-pytorch-research:latest"
  compiled_at: "2024-01-15T10:30:00Z"
```

## CLI Integration

### **Enhanced Launch Commands**

```bash
# Traditional EC2 launch (unchanged)
cws launch python-ml my-research-project

# SageMaker Studio Lab launch (free)
cws launch sagemaker-studio-lab ml-learning
# → Creates Studio Lab environment
# → Returns web URL for direct access
# → Shows "Free tier - no cost" message

# SageMaker Studio launch (managed)  
cws launch sagemaker-studio-gpu ml-training --instance-type ml.g4dn.xlarge
# → Creates SageMaker domain if needed
# → Provisions user profile with research user identity
# → Returns studio URL + cost estimate

# Canvas no-code ML launch
cws launch sagemaker-canvas business-analysis
# → Creates Canvas workspace
# → Configures data sources
# → Returns Canvas URL
```

### **Enhanced List and Info Commands**

```bash
# Unified listing shows all service types
cws list
# INSTANCE          TYPE           STATUS    ACCESS     COST/HOUR
# ml-learning       sagemaker_lab  running   web        $0.00
# ml-training       sagemaker      running   web        $0.736  
# ubuntu-server     ec2            running   ssh        $0.08
# data-prep         glue           running   web        $0.44

# Enhanced info shows service-specific details
cws info ml-training
# Instance: ml-training
# Service: SageMaker Studio
# Status: Running
# Web URL: https://studio-ml-training.studio.us-west-2.sagemaker.aws
# Current Instance: ml.g4dn.xlarge ($0.736/hour)
# Domain: research-ml-domain
# User Profile: alice_researcher
# EFS Mount: /home/alice_researcher/workspace (shared with EC2 instances)
# Auto-stop: 30 minutes idle
```

### **Web Access Command**

```bash
# New connect command for web services
cws connect ml-training
# → Opens web browser to SageMaker Studio URL
# → Shows connection info and shortcuts

cws connect ml-training --print-url
# → Prints URL without opening browser (for remote/headless usage)
# https://studio-ml-training.studio.us-west-2.sagemaker.aws
```

## Implementation Architecture

### **SageMaker Service Manager**

```go
// SageMakerManager handles SageMaker service lifecycle
type SageMakerManager struct {
    client    sagemakerClient
    iamClient iamClient
    efsClient efsClient
    
    // Configuration
    region        string
    vpcID         string  // CloudWorkstation managed VPC
    subnetIDs     []string
    securityGroups []string
}

// LaunchStudioEnvironment creates SageMaker Studio environment
func (sm *SageMakerManager) LaunchStudioEnvironment(config LaunchConfig) (*StudioEnvironment, error) {
    // 1. Create or get existing domain
    domain, err := sm.ensureDomain(config.DomainName, config.VPCConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create domain: %w", err)
    }
    
    // 2. Create user profile with research user identity
    userProfile, err := sm.createUserProfile(domain.DomainID, config.ResearchUser)
    if err != nil {
        return nil, fmt.Errorf("failed to create user profile: %w", err)
    }
    
    // 3. Configure EFS integration if requested
    if config.EFSIntegration {
        err = sm.attachEFSToUserProfile(userProfile, config.EFSFileSystem)
        if err != nil {
            return nil, fmt.Errorf("failed to attach EFS: %w", err)
        }
    }
    
    // 4. Generate presigned URL for direct access
    studioURL, err := sm.generateStudioURL(domain.DomainID, userProfile.UserProfileName)
    if err != nil {
        return nil, fmt.Errorf("failed to generate studio URL: %w", err)
    }
    
    return &StudioEnvironment{
        DomainID:        domain.DomainID,
        UserProfileName: userProfile.UserProfileName,
        StudioURL:       studioURL,
        InstanceTypes:   config.InstanceTypes,
        EFSMounted:      config.EFSIntegration,
    }, nil
}
```

### **Policy Integration**

```go
// Enhanced policy validation for web services
func (p *BasicPolicyRestrictions) ValidateServiceLaunch(serviceType ServiceType, config interface{}) error {
    switch serviceType {
    case ServiceTypeSageMaker:
        return p.validateSageMakerLaunch(config.(*SageMakerConfig))
    case ServiceTypeEC2:
        return p.validateEC2Launch(config.(*EC2Config))
    default:
        return fmt.Errorf("unsupported service type: %s", serviceType)
    }
}

func (p *BasicPolicyRestrictions) validateSageMakerLaunch(config *SageMakerConfig) error {
    var violations []string
    
    // Check if SageMaker services are allowed
    if !p.IsServiceTypeAllowed("sagemaker") {
        violations = append(violations, "SageMaker services not allowed by policy")
    }
    
    // Check instance type restrictions
    if config.StudioConfig != nil {
        for _, instanceType := range config.StudioConfig.InstanceTypes {
            if !p.IsInstanceTypeAllowed(instanceType) {
                violations = append(violations, 
                    fmt.Sprintf("SageMaker instance type '%s' not allowed", instanceType))
            }
        }
        
        // Check cost limits
        if config.StudioConfig.MaxHourlyCost > p.MaxHourlyCost {
            violations = append(violations, 
                fmt.Sprintf("SageMaker cost limit $%.2f exceeds policy maximum $%.2f", 
                    config.StudioConfig.MaxHourlyCost, p.MaxHourlyCost))
        }
    }
    
    if len(violations) > 0 {
        return fmt.Errorf("policy violations: %v", violations)
    }
    
    return nil
}
```

## Cost Tracking Integration

### **Multi-Service Cost Management**

```go
// Enhanced cost tracking for multiple service types
type CostTracker struct {
    ec2Costs       *EC2CostTracker
    sageMakerCosts *SageMakerCostTracker
    storageCosts   *StorageCostTracker
}

func (ct *CostTracker) GetInstanceCosts(instanceID string) (*ServiceCosts, error) {
    instance, err := ct.getInstance(instanceID)
    if err != nil {
        return nil, err
    }
    
    costs := &ServiceCosts{}
    
    switch instance.ServiceType {
    case ServiceTypeEC2:
        costs.ComputeCost = ct.ec2Costs.GetHourlyCost(instance.InstanceType)
    case ServiceTypeSageMaker:
        costs.ServiceCost = ct.sageMakerCosts.GetStudioCost(instance.ID)
        costs.ComputeCost = ct.sageMakerCosts.GetComputeCost(instance.ID)
    }
    
    // Storage costs apply to all service types
    costs.StorageCost = ct.storageCosts.GetStorageCost(instance.ID)
    
    costs.TotalHourlyCost = costs.ComputeCost + costs.ServiceCost + costs.StorageCost
    
    return costs, nil
}
```

## Success Metrics

### **Technical Success Criteria**
- [ ] SageMaker Studio Lab environments launch in <30 seconds
- [ ] SageMaker Studio environments integrate with research user identity
- [ ] EFS sharing works between EC2 instances and SageMaker environments
- [ ] Policy framework applies consistently to both EC2 and SageMaker services
- [ ] Cost tracking accurately reflects multi-service usage

### **User Experience Success Criteria**  
- [ ] CLI commands work identically for EC2 and SageMaker services
- [ ] Web browser launch provides direct access to SageMaker Studio
- [ ] Unified listing shows all services in consistent format
- [ ] Policy violations provide clear, actionable error messages

This SageMaker integration serves as the foundation for all Phase 5B web service integrations, establishing patterns for service-specific configuration, policy enforcement, cost tracking, and unified user experience across EC2 and managed AWS services.