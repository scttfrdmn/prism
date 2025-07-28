# Template Application Engine - Implementation

## Overview

The Template Application Engine enables applying CloudWorkstation templates to already running instances, allowing for incremental environment evolution without requiring instance recreation.

## Architecture

The system is built with a modular architecture consisting of four main components:

```
TemplateApplicationEngine
├── InstanceStateInspector    # Analyzes current instance state
├── TemplateDiffCalculator    # Calculates differences between current and desired state
├── IncrementalApplyEngine    # Applies template changes incrementally
└── TemplateRollbackManager   # Manages checkpoints and rollback capabilities
```

## Core Components

### 1. TemplateApplicationEngine

**File**: `pkg/templates/application.go`

The main orchestrator that coordinates template application:

```go
func (e *TemplateApplicationEngine) ApplyTemplate(ctx context.Context, req ApplyRequest) (*ApplyResponse, error)
```

**Workflow**:
1. Validates the apply request
2. Inspects current instance state
3. Calculates template differences
4. Handles dry-run mode if requested
5. Checks for conflicts (unless forced)
6. Creates rollback checkpoint
7. Applies template changes
8. Records successful application

### 2. InstanceStateInspector

**File**: `pkg/templates/inspector.go`

Examines the current state of running instances across multiple dimensions:

```go
func (i *InstanceStateInspector) InspectInstance(ctx context.Context, instanceName string) (*InstanceState, error)
```

**Capabilities**:
- **Package Detection**: Supports apt, dnf, conda, pip package managers
- **Service Inspection**: Uses systemctl to analyze running services
- **User Account Analysis**: Inspects /etc/passwd and user groups
- **Port Detection**: Uses netstat/ss to find listening ports
- **Package Manager Detection**: Automatically identifies primary package manager
- **Template History**: Loads previously applied template records

### 3. TemplateDiffCalculator

**File**: `pkg/templates/diff.go`

Calculates the precise differences between current instance state and desired template configuration:

```go
func (d *TemplateDiffCalculator) CalculateDiff(currentState *InstanceState, template *Template) (*TemplateDiff, error)
```

**Analysis Types**:
- **Package Differences**: Identifies packages to install, upgrade, or remove
- **Service Differences**: Determines services to configure, start, or stop
- **User Differences**: Calculates user accounts to create or modify
- **Port Differences**: Identifies ports that need to be opened
- **Conflict Detection**: Finds potential conflicts requiring resolution

**Conflict Resolution**:
- Package manager conflicts (template vs instance)
- Port conflicts (services using same ports)
- User conflicts (existing users with different configurations)

### 4. IncrementalApplyEngine

**File**: `pkg/templates/incremental.go`

Applies the calculated template differences to running instances:

```go
func (e *IncrementalApplyEngine) ApplyChanges(ctx context.Context, instanceName string, diff *TemplateDiff, template *Template) (*ApplyResult, error)
```

**Application Process**:
1. **Package Installation**: Generates package manager-specific scripts
2. **Service Configuration**: Creates systemctl-based service management scripts
3. **User Management**: Handles user creation and group membership updates
4. **Port Management**: Placeholder for security group integration

**Package Manager Support**:
- **apt**: System packages with version pinning
- **dnf**: Red Hat/Fedora packages
- **conda**: Data science and cross-platform packages
- **pip**: Python packages
- **spack**: HPC and scientific computing packages

### 5. TemplateRollbackManager

**File**: `pkg/templates/rollback.go`

Manages rollback checkpoints for safe template application:

```go
func (r *TemplateRollbackManager) CreateCheckpoint(ctx context.Context, instanceName string) (string, error)
func (r *TemplateRollbackManager) RollbackToCheckpoint(ctx context.Context, instanceName, checkpointID string) error
```

**Checkpoint Components**:
- **Package Snapshots**: Record of all installed packages
- **Service States**: Status and configuration of all services
- **User Accounts**: Complete user and group membership records
- **Configuration Files**: Backup of critical system files
- **Environment Variables**: Important shell environment state

**Rollback Capabilities**:
- Configuration file restoration from backups
- Service state restoration to checkpoint conditions
- Package removal (best-effort for packages added after checkpoint)
- Environment variable restoration

## Remote Execution

### RemoteExecutor Interface

**File**: `pkg/templates/executor.go`

Provides abstraction for executing commands on remote instances:

```go
type RemoteExecutor interface {
    Execute(ctx context.Context, instanceName string, command string) (*ExecutionResult, error)
    ExecuteScript(ctx context.Context, instanceName string, script string) (*ExecutionResult, error)
    CopyFile(ctx context.Context, instanceName string, localPath, remotePath string) error
    GetFile(ctx context.Context, instanceName string, remotePath, localPath string) error
}
```

### Implementations

1. **SSHRemoteExecutor**: Direct SSH connections for instances with public IPs
2. **SystemsManagerExecutor**: AWS Systems Manager for private instances (placeholder)
3. **MockRemoteExecutor**: Testing implementation with predefined responses

## Data Types

### Core State Types

```go
type InstanceState struct {
    Packages          []InstalledPackage    `json:"packages"`
    Services          []RunningService      `json:"services"`
    Users            []ExistingUser        `json:"users"`
    Ports            []int                 `json:"ports"`
    PackageManager   string                `json:"package_manager"`
    AppliedTemplates []AppliedTemplate     `json:"applied_templates"`
    LastInspected    time.Time             `json:"last_inspected"`
}

type TemplateDiff struct {
    PackagesToInstall    []PackageDiff  `json:"packages_to_install"`
    ServicesToConfigure  []ServiceDiff  `json:"services_to_configure"`
    UsersToCreate        []UserDiff     `json:"users_to_create"`
    ConflictsFound       []ConflictDiff `json:"conflicts_found"`
    // ... additional fields
}
```

### Request/Response Types

```go
type ApplyRequest struct {
    InstanceName   string    `json:"instance_name"`
    Template       *Template `json:"template"`
    PackageManager string    `json:"package_manager,omitempty"`
    DryRun         bool      `json:"dry_run"`
    Force          bool      `json:"force"`
}

type ApplyResponse struct {
    Success            bool     `json:"success"`
    Message            string   `json:"message"`
    PackagesInstalled  int      `json:"packages_installed"`
    ServicesConfigured int      `json:"services_configured"`
    UsersCreated       int      `json:"users_created"`
    RollbackCheckpoint string   `json:"rollback_checkpoint"`
    Warnings           []string `json:"warnings"`
    ExecutionTime      time.Duration `json:"execution_time"`
}
```

## Usage Examples

### Basic Template Application

```go
// Create template application engine with SSH executor
keyPath := "/path/to/ssh/key"
executor := NewSSHRemoteExecutor(keyPath, "ubuntu")
engine := NewTemplateApplicationEngine(executor)

// Apply template to running instance
req := ApplyRequest{
    InstanceName: "my-workspace",
    Template:     &myTemplate,
    DryRun:       false,
    Force:        false,
}

response, err := engine.ApplyTemplate(context.Background(), req)
if err != nil {
    log.Fatalf("Template application failed: %v", err)
}

fmt.Printf("✅ Applied template successfully\n")
fmt.Printf("📦 Packages installed: %d\n", response.PackagesInstalled)
fmt.Printf("🔧 Services configured: %d\n", response.ServicesConfigured)
fmt.Printf("👤 Users created: %d\n", response.UsersCreated)
fmt.Printf("⏱️ Execution time: %v\n", response.ExecutionTime)
```

### Dry Run Analysis

```go
req := ApplyRequest{
    InstanceName: "my-workspace",
    Template:     &myTemplate,
    DryRun:       true,  // Preview changes without applying
}

response, err := engine.ApplyTemplate(context.Background(), req)
if err != nil {
    log.Fatalf("Dry run failed: %v", err)
}

fmt.Printf("📋 Dry run results:\n")
fmt.Printf("Would install %d packages\n", response.PackagesInstalled)
fmt.Printf("Would configure %d services\n", response.ServicesConfigured)
fmt.Printf("Would create %d users\n", response.UsersCreated)
```

### Rollback Management

```go
// List available checkpoints
rollbackManager := NewTemplateRollbackManager(executor)
checkpoints, err := rollbackManager.ListCheckpoints(context.Background(), "my-workspace")
if err != nil {
    log.Fatalf("Failed to list checkpoints: %v", err)
}

for _, checkpoint := range checkpoints {
    fmt.Printf("Checkpoint: %s (created: %v)\n", checkpoint.ID, checkpoint.CreatedAt)
}

// Rollback to specific checkpoint
err = rollbackManager.RollbackToCheckpoint(context.Background(), "my-workspace", "checkpoint-1234567890")
if err != nil {
    log.Fatalf("Rollback failed: %v", err)
}
```

## Testing

### Mock Executor for Unit Tests

```go
func TestTemplateApplication(t *testing.T) {
    // Create mock executor
    mockExecutor := NewMockRemoteExecutor()
    
    // Set expected command results
    mockExecutor.SetResult("dpkg -l", &ExecutionResult{
        ExitCode: 0,
        Stdout:   "ii  package1  1.0.0  amd64  Test package",
    })
    
    // Create engine with mock executor
    engine := NewTemplateApplicationEngine(mockExecutor)
    
    // Test template application
    req := ApplyRequest{...}
    response, err := engine.ApplyTemplate(context.Background(), req)
    
    // Verify results
    assert.NoError(t, err)
    assert.True(t, response.Success)
    
    // Verify executed commands
    commands := mockExecutor.GetCommands()
    assert.Contains(t, commands, "dpkg -l")
}
```

## Integration Points

### CLI Integration

The template application engine is designed to integrate with CloudWorkstation's CLI:

```bash
# New CLI commands that would use this engine
cws apply <template> <instance-name> [options]
cws diff <template> <instance-name>
cws layers <instance-name>
cws rollback <instance-name> [--to-checkpoint=<id>]
```

### API Integration

The engine provides request/response types suitable for REST API integration:

```go
// Add to daemon API handlers
func (s *Server) handleApplyTemplate(w http.ResponseWriter, r *http.Request) {
    var req ApplyRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    response, err := s.templateEngine.ApplyTemplate(r.Context(), req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(response)
}
```

### State Management Integration

The engine maintains template application history in the CloudWorkstation state system:

```json
{
  "instances": {
    "my-workspace": {
      "applied_templates": [
        {
          "name": "python-ml",
          "applied_at": "2024-01-15T14:20:00Z",
          "package_manager": "conda",
          "packages_installed": ["tensorflow", "pytorch"],
          "rollback_checkpoint": "checkpoint-1705327200"
        }
      ]
    }
  }
}
```

## Current Status

### ✅ Completed Components

1. **Core Architecture**: Complete modular design with proper separation of concerns
2. **State Inspection**: Comprehensive package, service, user, and port analysis
3. **Diff Calculation**: Template difference computation with conflict detection
4. **Incremental Application**: Package installation and service configuration scripts
5. **Rollback System**: Checkpoint creation and restoration capabilities
6. **Remote Execution**: SSH-based executor with testing support

### 🚧 Implementation Needed

1. **Instance IP Resolution**: Integration with CloudWorkstation state management
2. **Security Group Updates**: Port opening via AWS API
3. **Systems Manager Executor**: AWS Systems Manager implementation for private instances
4. **CLI Commands**: `cws apply`, `cws diff`, `cws layers`, `cws rollback`
5. **API Endpoints**: REST API integration with daemon
6. **Integration Testing**: End-to-end testing with real instances

### 🎯 Next Development Phase

The current prototype provides a solid foundation for template application to running instances. The next phase should focus on:

1. **Integration**: Connect with existing CloudWorkstation state and AWS management
2. **CLI Implementation**: Add the new commands to the existing CLI application
3. **Testing**: Comprehensive testing with real AWS instances
4. **Documentation**: User-facing documentation and examples
5. **Error Handling**: Robust error handling and recovery mechanisms

This template application engine represents a significant advancement in CloudWorkstation's capabilities, transforming it from a "launch and manage" platform into a true "infrastructure as code" research environment system.