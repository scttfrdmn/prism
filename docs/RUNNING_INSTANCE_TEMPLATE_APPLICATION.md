# Running Instance Template Application - Roadmap Analysis

## User Request
> "Can I run a template on an already defined and running cloudworkstation?"

## Current State Analysis

### What CloudWorkstation Currently Supports
✅ **Template-based Launches**: Templates define complete environments from scratch
✅ **Template Inheritance**: Stack templates to build complex environments
✅ **Package Manager Overrides**: Choose different package managers at launch time
✅ **Instance Management**: Start, stop, connect to running instances

### What's Missing
❌ **Live Template Application**: Apply templates to running instances
❌ **Incremental Updates**: Add packages/services to existing environments
❌ **Configuration Drift Detection**: Compare running state vs template specification
❌ **Rollback Capability**: Undo template applications

## Use Cases & Value Proposition

### Research Workflow Scenarios

**1. Iterative Environment Building**
```bash
# Current workflow (requires new instance)
cws launch base-ubuntu my-workspace
# ... work in environment, realize need ML tools
cws terminate my-workspace
cws launch python-ml my-workspace-v2

# Desired workflow (apply to running instance)
cws launch base-ubuntu my-workspace
# ... work in environment, realize need ML tools  
cws apply python-ml-stack my-workspace  # Apply template to running instance
```

**2. Environment Evolution**
```bash
# Start with basic R environment
cws launch r-research data-analysis

# Later add GIS capabilities
cws apply gis-stack data-analysis

# Later add GPU support for visualization
cws apply gpu-viz-stack data-analysis
```

**3. Collaborative Environment Setup**
```bash
# Researcher A sets up base environment
cws launch basic-python collaboration-env

# Researcher B adds their specialized tools
cws apply bioinformatics-stack collaboration-env

# Researcher C adds visualization tools
cws apply scivis-stack collaboration-env
```

## Technical Architecture Design

### 1. Template Application Engine

**Core Components**:
```go
// New template application system
type TemplateApplicationEngine struct {
    StateInspector   *InstanceStateInspector
    DiffCalculator   *TemplateDiffCalculator  
    ApplyEngine      *IncrementalApplyEngine
    RollbackManager  *TemplateRollbackManager
}

// Inspect current instance state
type InstanceStateInspector struct {
    PackageManager   PackageManagerInspector
    ServiceInspector ServiceStateInspector
    UserInspector    UserStateInspector
    ConfigInspector  ConfigurationInspector
}

// Calculate differences between current state and desired template
type TemplateDiffCalculator struct {
    PackageDiffs  []PackageDiff
    ServiceDiffs  []ServiceDiff
    UserDiffs     []UserDiff
    ConfigDiffs   []ConfigDiff
}
```

### 2. CLI Commands

**New Commands**:
```bash
# Apply template to running instance
cws apply <template> <instance-name> [options]

# Preview what would be applied (dry-run)
cws apply <template> <instance-name> --dry-run

# Show difference between current state and template
cws diff <template> <instance-name>

# List applied templates/layers on instance
cws layers <instance-name>

# Rollback to previous state
cws rollback <instance-name> [--to-layer=<layer-id>]
```

### 3. State Management & Tracking

**Instance State Tracking**:
```json
{
  "instances": {
    "my-workspace": {
      "id": "i-1234567890abcdef0",
      "name": "my-workspace",
      "base_template": "ubuntu-22.04",
      "applied_templates": [
        {
          "template": "basic-python",
          "applied_at": "2024-01-15T10:30:00Z",
          "package_manager": "conda",
          "packages_installed": ["python=3.11", "jupyter"],
          "services_configured": ["jupyter"],
          "rollback_checkpoint": "checkpoint-001"
        },
        {
          "template": "ml-stack", 
          "applied_at": "2024-01-15T14:20:00Z",
          "package_manager": "conda",
          "packages_installed": ["tensorflow", "pytorch", "scikit-learn"],
          "rollback_checkpoint": "checkpoint-002"
        }
      ]
    }
  }
}
```

## Implementation Phases

### Phase 1: State Inspection & Diff System
**Goal**: Understand what's currently installed vs what template wants

**Implementation**:
```go
// Inspect current instance state via SSH/Systems Manager
func (e *TemplateApplicationEngine) InspectInstanceState(instanceName string) (*InstanceState, error) {
    // Connect to instance
    conn, err := e.connectToInstance(instanceName)
    if err != nil {
        return nil, err
    }
    
    // Inspect installed packages
    packages, err := e.inspectInstalledPackages(conn)
    if err != nil {
        return nil, err
    }
    
    // Inspect running services
    services, err := e.inspectRunningServices(conn)
    if err != nil {
        return nil, err
    }
    
    // Inspect user accounts
    users, err := e.inspectUsers(conn)
    if err != nil {
        return nil, err
    }
    
    return &InstanceState{
        Packages: packages,
        Services: services,
        Users:    users,
    }, nil
}

// Calculate what needs to be changed
func (e *TemplateApplicationEngine) CalculateDiff(currentState *InstanceState, template *Template) (*TemplateDiff, error) {
    diff := &TemplateDiff{}
    
    // Package differences
    for _, pkg := range template.Packages.System {
        if !currentState.HasPackage(pkg) {
            diff.PackagesToInstall = append(diff.PackagesToInstall, pkg)
        }
    }
    
    // Service differences  
    for _, svc := range template.Services {
        if !currentState.HasService(svc.Name) {
            diff.ServicesToConfigure = append(diff.ServicesToConfigure, svc)
        }
    }
    
    return diff, nil
}
```

### Phase 2: Incremental Application Engine
**Goal**: Apply template differences to running instance

**Implementation**:
```go
func (e *TemplateApplicationEngine) ApplyTemplate(instanceName string, template *Template, options ApplyOptions) error {
    // 1. Create rollback checkpoint
    checkpoint, err := e.createRollbackCheckpoint(instanceName)
    if err != nil {
        return fmt.Errorf("failed to create rollback checkpoint: %w", err)
    }
    
    // 2. Calculate what needs to be applied
    currentState, err := e.InspectInstanceState(instanceName)
    if err != nil {
        return err
    }
    
    diff, err := e.CalculateDiff(currentState, template)
    if err != nil {
        return err
    }
    
    // 3. Generate application script
    script, err := e.generateApplicationScript(diff, template.PackageManager)
    if err != nil {
        return err
    }
    
    // 4. Apply changes to instance
    if err := e.executeOnInstance(instanceName, script); err != nil {
        // Rollback on failure
        e.rollbackToCheckpoint(instanceName, checkpoint)
        return fmt.Errorf("template application failed: %w", err)
    }
    
    // 5. Update instance state tracking
    return e.recordTemplateApplication(instanceName, template, checkpoint)
}
```

### Phase 3: Rollback & Layer Management
**Goal**: Manage template layers and provide rollback capability

**Commands**:
```bash
# Show applied template layers
$ cws layers my-workspace
LAYER  TEMPLATE           APPLIED              PACKAGES    SERVICES
1      base-ubuntu        2024-01-15 10:00    15          2
2      python-research    2024-01-15 11:30    8           1  
3      ml-stack          2024-01-15 14:20    12          0

# Rollback to specific layer
$ cws rollback my-workspace --to-layer=2
Rolling back to layer 2 (python-research)...
Removing ml-stack packages: tensorflow, pytorch, scikit-learn
Instance rolled back successfully.
```

## Technical Challenges & Solutions

### 1. Package Manager Conflicts
**Challenge**: Different templates might use different package managers
**Solution**: 
- Detect existing package manager on instance
- Use same package manager for consistency
- Warn/error if template requires different package manager
- Support mixed package managers with clear isolation

### 2. Service Conflicts
**Challenge**: Templates might define conflicting services (e.g., different web servers on same port)
**Solution**:
- Port conflict detection before application
- Service dependency resolution
- Graceful service restart/reconfiguration
- Clear error messages for irreconcilable conflicts

### 3. User Account Conflicts  
**Challenge**: Templates might try to create existing users
**Solution**:
- User existence checking before creation
- Merge user group memberships intelligently
- Skip existing users with warning
- Option to force user reconfiguration

### 4. Configuration Drift
**Challenge**: Manual changes vs template-managed configuration
**Solution**:
- Configuration file backup before template application
- Merge strategies for configuration conflicts
- Manual change preservation options
- Clear reporting of what was overwritten

## Implementation Strategy

### CLI Integration
```bash
# Add to existing CLI app structure
func (a *App) Apply(args []string) error {
    if len(args) < 2 {
        return fmt.Errorf("usage: cws apply <template> <instance-name> [options]")
    }
    
    template := args[0]
    instanceName := args[1]
    
    // Parse options
    req := types.ApplyRequest{
        Template:     template,
        InstanceName: instanceName,
        DryRun:       false, // Parse from --dry-run flag
    }
    
    // Apply via daemon API
    response, err := a.apiClient.ApplyTemplate(a.ctx, req)
    if err != nil {
        return fmt.Errorf("failed to apply template: %w", err) 
    }
    
    fmt.Printf("✅ Applied template '%s' to instance '%s'\n", template, instanceName)
    fmt.Printf("📊 Changes: %d packages, %d services, %d users\n", 
        response.PackagesInstalled, response.ServicesConfigured, response.UsersCreated)
    
    return nil
}
```

### API Extensions
```go
// Add to daemon API
type ApplyRequest struct {
    Template     string `json:"template"`
    InstanceName string `json:"instance_name"`
    DryRun       bool   `json:"dry_run"`
    Force        bool   `json:"force"`        // Override conflicts
    PackageManager string `json:"package_manager,omitempty"` // Override template default
}

type ApplyResponse struct {
    Success            bool   `json:"success"`
    Message            string `json:"message"`
    PackagesInstalled  int    `json:"packages_installed"`
    ServicesConfigured int    `json:"services_configured"`
    UsersCreated       int    `json:"users_created"`
    RollbackCheckpoint string `json:"rollback_checkpoint"`
    Warnings           []string `json:"warnings"`
}
```

## Benefits & Impact

### For Researchers
✅ **Environment Evolution**: Grow environments incrementally without starting over
✅ **Experimentation**: Try adding new tools without losing current work
✅ **Collaboration**: Team members can add their tools to shared environments
✅ **Time Savings**: No need to recreate entire environments for minor additions

### For System Administrators
✅ **Standardization**: Apply organizational templates to existing instances
✅ **Compliance**: Ensure running instances meet security/policy requirements
✅ **Maintenance**: Update environments with patches and new tools
✅ **Rollback Capability**: Quick recovery from problematic changes

### For CloudWorkstation Platform
✅ **Competitive Advantage**: Unique capability not available in basic VM platforms
✅ **User Retention**: Reduces friction in environment management
✅ **Template Adoption**: Increases value of template library
✅ **Advanced Use Cases**: Enables sophisticated research workflows

## Roadmap Priority

**Priority**: **High** - This would be a significant differentiator for CloudWorkstation

**Effort Estimate**: **3-4 development cycles**
- Phase 1: State inspection & diff (1 cycle)
- Phase 2: Application engine (2 cycles) 
- Phase 3: Rollback & layer management (1 cycle)

**Dependencies**: 
- ✅ Template inheritance system (completed)
- ✅ Template validation system (completed)
- ⚠️ Remote execution system (needs implementation)
- ⚠️ Instance state management (needs enhancement)

## Next Steps

1. **Prototype State Inspection**: Build SSH-based package/service inspection
2. **Template Diff Engine**: Implement difference calculation system
3. **Remote Execution Framework**: Secure script execution on running instances
4. **Rollback System Design**: Checkpoint and restoration mechanisms
5. **User Testing**: Validate workflow with researcher feedback

This capability would transform CloudWorkstation from a "launch and manage" platform into a true "infrastructure as code" research environment system, where environments can evolve dynamically while maintaining reproducibility and rollback capabilities.