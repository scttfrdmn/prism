# Template Application System

## Overview

The CloudWorkstation Template Application System enables applying software templates to already running instances without requiring instance recreation. This revolutionary capability transforms CloudWorkstation from a simple instance launcher into a comprehensive research environment manager.

## Key Benefits

### For Researchers
- **No Downtime**: Apply new software without stopping work
- **Incremental Enhancement**: Layer additional tools on existing environments
- **Safe Experimentation**: Rollback capabilities for risk-free testing
- **Cost Efficiency**: Avoid launch costs and data transfer for environment changes

### For Research Teams  
- **Environment Standardization**: Apply consistent tool stacks across team instances
- **Progressive Configuration**: Start basic, add complexity as projects evolve
- **Collaborative Development**: Share template layers for reproducible environments
- **Version Control**: Track environment changes with rollback history

## Core Capabilities

### 1. Template Application
Apply any CloudWorkstation template to a running instance:

```bash
# Apply machine learning stack to existing Python environment
cws template apply cuda-ml my-workspace

# Add bioinformatics tools to existing instance  
cws template apply bioinformatics data-analysis --package-manager conda

# Preview changes before applying
cws template apply r-research stats-box --dry-run
```

### 2. Environment Difference Analysis
Calculate exactly what will change before applying templates:

```bash
# See what packages/services will be installed
cws template diff neuroimaging brain-analysis

# Example output:
# Packages to install (conda):
#   - fsl 6.0.4
#   - ants 2.3.1
#   - mrtrix3 3.0.2
# Services to configure:
#   - fsl-viewer (port 8080)
# Users to create:
#   - neuroimager (groups: fsl, ants)
```

### 3. Layer Management
Track the history of applied templates with full transparency:

```bash
# View all applied template layers
cws template layers my-workspace

# Example output:
# Applied Templates:
# 1. base-python (2024-01-15 10:30) - checkpoint-001
# 2. jupyter-lab (2024-01-15 14:20) - checkpoint-002  
# 3. ml-stack (2024-01-16 09:15) - checkpoint-003
```

### 4. Safe Rollback
Undo template applications safely with checkpoint-based recovery:

```bash
# Rollback to before ML stack was applied
cws template rollback my-workspace checkpoint-002

# Rollback removes:
# - Packages installed in later templates
# - Configuration changes made after checkpoint
# - Services started after checkpoint
```

## Technical Architecture

### Multi-Package Manager Support

The system intelligently works with multiple package managers:

- **System Packages**: `apt`, `dnf`, `yum` for base system tools
- **Python Environments**: `conda`, `pip` for data science stacks
- **Scientific Computing**: `spack` for HPC and specialized research software
- **Containerized Tools**: `docker`, `apptainer` for complex applications

### Remote Execution Framework

**Intelligent Connection Management**:
- **SSH Execution**: For instances with public IPs (fastest, most reliable)
- **AWS Systems Manager**: For private instances in secure networks
- **Automatic Fallback**: Seamless switching based on connectivity

**Security Features**:
- **Credential Management**: Automatic SSH key resolution and IAM role usage
- **Execution Validation**: All commands validated before execution
- **Audit Logging**: Complete history of all remote operations

### State Management

**Instance State Inspection**:
- **Package Discovery**: Detect installed packages across all managers
- **Service Analysis**: Identify running services and configurations  
- **User Management**: Track created users and group memberships
- **Port Mapping**: Discover exposed services and network configuration

**Rollback Checkpoints**:
- **Automatic Creation**: Checkpoint before every template application
- **Configuration Backup**: Save system configs, service states, user data
- **Incremental Storage**: Efficient storage of only changed files
- **Cross-Session Persistence**: Checkpoints survive instance stops/starts

## Implementation Details

### Core Components

#### 1. Template Application Engine (`pkg/templates/application.go`)
Central orchestrator that coordinates the entire application workflow:

```go
type TemplateApplicationEngine struct {
    executor     RemoteExecutor
    inspector    *InstanceStateInspector  
    diffCalc     *TemplateDiffCalculator
    incremental  *IncrementalApplyEngine
    rollback     *TemplateRollbackManager
}
```

**Key Features**:
- **Unified Workflow**: Single entry point for all template operations
- **Error Recovery**: Automatic rollback on application failures
- **Progress Tracking**: Real-time feedback during long operations
- **Dry-run Support**: Complete simulation without system changes

#### 2. Instance State Inspector (`pkg/templates/inspector.go`)
Comprehensive analysis of current instance configuration:

**Package Detection**:
- Scans all package managers for installed software
- Identifies versions, dependencies, and installation sources
- Handles virtual environments and conda environments
- Detects manually compiled software

**Service Discovery**:
- Enumerates systemd services and their states
- Identifies custom services and startup scripts  
- Maps service dependencies and configurations
- Detects web services and exposed ports

**User Analysis**:
- Lists all system users and their properties
- Maps group memberships and permissions
- Identifies SSH keys and authentication methods
- Analyzes home directory configurations

#### 3. Template Difference Calculator (`pkg/templates/diff.go`) 
Intelligent comparison between current state and desired template:

**Conflict Detection**:
- **Version Conflicts**: Identifies incompatible package versions
- **Service Conflicts**: Detects port and resource conflicts
- **Permission Conflicts**: Finds user/group permission issues
- **Dependency Conflicts**: Analyzes package dependency chains

**Change Calculation**:
- **Package Changes**: Install, upgrade, downgrade, remove operations
- **Service Changes**: Start, stop, enable, disable, reconfigure
- **User Changes**: Create, modify, delete users and groups
- **Configuration Changes**: File modifications and additions

#### 4. Incremental Apply Engine (`pkg/templates/incremental.go`)
Executes template changes with surgical precision:

**Package Management**:
- **Manager Selection**: Choose optimal package manager for each package
- **Batch Operations**: Group related packages for efficient installation
- **Dependency Resolution**: Handle complex dependency chains correctly
- **Error Recovery**: Rollback partial installations on failures

**Service Configuration**:
- **Templated Configs**: Generate configuration files from templates
- **Service Management**: Start/stop/restart services as needed
- **Health Checking**: Verify services start correctly after configuration
- **Port Management**: Configure firewalls and security groups

#### 5. Rollback Manager (`pkg/templates/rollback.go`)
Comprehensive checkpoint and recovery system:

**Checkpoint Creation**:
- **File Snapshots**: Backup critical configuration files
- **Package Lists**: Record installed packages and versions
- **Service States**: Save service configurations and states
- **User Data**: Backup user accounts and home directories

**Recovery Operations**:
- **Package Restoration**: Remove packages added after checkpoint
- **File Restoration**: Restore original configuration files
- **Service Restoration**: Reset services to checkpoint state
- **User Restoration**: Restore user accounts and permissions

#### 6. Remote Executor (`pkg/templates/executor.go`)
Abstracted remote execution with multiple backends:

**SSH Executor**:
- **Connection Pooling**: Reuse connections for multiple operations
- **Key Management**: Automatic SSH key discovery and usage
- **Error Handling**: Robust handling of network interruptions
- **File Transfer**: Efficient copying of files and scripts

**Systems Manager Executor**:
- **IAM Integration**: Use instance roles for authentication
- **Session Management**: Handle long-running operations
- **Output Streaming**: Real-time command output streaming
- **Cross-Region Support**: Connect to instances in any AWS region

### API Integration

#### REST Endpoints

**Template Application**:
```http
POST /api/v1/templates/apply
Content-Type: application/json

{
  "instance_name": "my-workspace",
  "template": { /* template definition */ },
  "package_manager": "conda",
  "dry_run": false,
  "force": false
}
```

**Template Differences**:
```http  
POST /api/v1/templates/diff
Content-Type: application/json

{
  "instance_name": "my-workspace", 
  "template": { /* template definition */ }
}
```

**Layer Management**:
```http
GET /api/v1/instances/my-workspace/layers
```

**Rollback Operations**:
```http
POST /api/v1/instances/my-workspace/rollback
Content-Type: application/json

{
  "checkpoint_id": "checkpoint-1640995200"
}
```

#### Request Validation

**Security Checks**:
- **Instance Verification**: Ensure instance exists and is accessible
- **Template Validation**: Validate template structure and content
- **Permission Verification**: Check user permissions for instance operations
- **Content Sanitization**: Prevent code injection through template content

**Error Handling**:
- **400 Bad Request**: Invalid request structure or missing fields
- **404 Not Found**: Instance not found or not running
- **500 Internal Server Error**: Template application failures
- **503 Service Unavailable**: Remote executor connection failures

### CLI Integration

#### Command Structure

**Apply Command**:
```bash
cws template apply <template-name> <instance-name> [options]

Options:
  --package-manager    Preferred package manager (conda, pip, spack, apt)
  --dry-run           Preview changes without applying
  --force             Apply even if conflicts detected
  --timeout           Execution timeout in minutes
```

**Diff Command**:
```bash
cws template diff <template-name> <instance-name>

# Shows detailed differences:
# - Packages to install/upgrade/remove
# - Services to configure/start/stop  
# - Users to create/modify/remove
# - Conflicts and warnings
```

**Layers Command**:
```bash
cws template layers <instance-name>

# Shows chronological history:
# - Template name and application time
# - Package manager used
# - Packages/services/users added
# - Rollback checkpoint ID
```

**Rollback Command**:
```bash
cws template rollback <instance-name> <checkpoint-id>

# Safe recovery:
# - Validates checkpoint exists
# - Shows what will be removed/restored
# - Confirms operation before proceeding
# - Updates instance template history
```

## Usage Examples

### Machine Learning Researcher Workflow

**Starting Point**: Basic Python instance
```bash
# Launch basic instance
cws launch basic-python ml-project

# Add Jupyter for interactive development
cws template apply jupyter-lab ml-project

# Add machine learning stack when ready
cws template apply cuda-ml ml-project --package-manager conda

# Add specialized computer vision tools
cws template apply cv-research ml-project
```

**Layer History**:
1. `basic-python` (base) - Python 3.9, basic tools
2. `jupyter-lab` (applied) - JupyterLab, extensions, themes  
3. `cuda-ml` (applied) - PyTorch, TensorFlow, CUDA toolkit
4. `cv-research` (applied) - OpenCV, scikit-image, matplotlib

### Bioinformatics Team Environment

**Standardization Across Team**:
```bash
# Team lead creates base environment
cws launch desktop-research bio-base
cws template apply bioinformatics bio-base
cws template apply visualization bio-base

# Team members clone the environment
cws template layers bio-base  # View applied layers
cws launch desktop-research member1-workspace
cws template apply bioinformatics member1-workspace
cws template apply visualization member1-workspace

# Specialized member adds genomics tools
cws template apply genomics member1-workspace --package-manager spack
```

### Experimental Development

**Safe Experimentation**:
```bash
# Working environment with important analysis
cws template layers analysis-server
# Output: base-r, tidyverse, stats-packages (checkpoint-123)

# Try experimental package that might break things
cws template diff experimental-ml analysis-server  # Preview changes
cws template apply experimental-ml analysis-server # Apply carefully

# If something breaks, rollback safely
cws template rollback analysis-server checkpoint-123

# Environment restored to working state
```

## Performance Characteristics

### Application Times

**Package Installation**:
- **Small templates** (5-10 packages): 2-5 minutes
- **Medium templates** (20-50 packages): 5-15 minutes  
- **Large templates** (100+ packages): 15-45 minutes
- **Scientific software** (Spack builds): 30-120 minutes

**State Inspection**:
- **Basic inspection**: 10-30 seconds
- **Comprehensive analysis**: 1-2 minutes
- **Large environments**: 2-5 minutes

**Difference Calculation**:
- **Simple diffs**: 5-15 seconds
- **Complex dependency analysis**: 30-60 seconds
- **Multi-manager environments**: 1-2 minutes

### Resource Usage

**Network Bandwidth**:
- **Package downloads**: Varies by template size (100MB-10GB)
- **SSH operations**: Minimal (< 1MB for most operations)
- **Systems Manager**: Low overhead (< 100KB typical)

**Instance Resources**:
- **CPU Usage**: High during package compilation, low otherwise
- **Memory Usage**: Varies by package manager (conda: 1-4GB, spack: 2-8GB)
- **Disk Usage**: Package caches and checkpoints (1-10GB typical)

## Troubleshooting

### Common Issues

**Connection Problems**:
```bash
# SSH connection failed
Error: Failed to connect via SSH: connection timeout
Solution: Check security groups, verify instance is running

# Systems Manager unavailable  
Error: SSM agent not installed or not running
Solution: Use SSH executor or install SSM agent
```

**Package Manager Issues**:
```bash
# Conda environment conflicts
Error: Package conflicts detected in conda environment
Solution: Use --force flag or resolve conflicts manually

# Spack build failures
Error: Spack package build failed: compiler not found
Solution: Apply compiler template first or use different package manager
```

**Permission Problems**:
```bash
# Insufficient permissions
Error: Permission denied: cannot install system packages
Solution: Ensure user has sudo access or use user-space package managers
```

### Recovery Procedures

**Failed Applications**:
1. **Automatic Rollback**: System automatically rolls back on critical failures
2. **Manual Recovery**: Use `cws template rollback` to restore previous state
3. **Checkpoint Verification**: Verify checkpoints exist before attempting recovery
4. **State Inspection**: Use `cws template layers` to understand current state

**Corrupted Checkpoints**:
1. **Multiple Checkpoints**: System maintains multiple recovery points
2. **Partial Recovery**: Restore what's possible, document what's lost
3. **Manual Reconstruction**: Recreate environment manually if needed
4. **Prevention**: Regular checkpoint creation and validation

## Future Enhancements

### Advanced Features (Roadmap)

**Template Scheduling**:
- Apply templates at specified times
- Batch operations across multiple instances
- Integration with job schedulers and workflow systems

**Dependency Management**:
- Automatic prerequisite template application
- Template dependency graphs and validation
- Smart ordering of template applications

**Cost Optimization**:
- Cost estimation for template applications
- Resource usage prediction and optimization
- Budget tracking for template-applied resources

**Collaboration Features**:
- Shared template libraries and version control
- Team-wide template application policies
- Collaborative environment management

### Integration Opportunities

**CI/CD Integration**:
- Automated template application in development pipelines
- Environment consistency testing and validation
- Template change management and approval workflows

**Monitoring Integration**:
- Application success rate and performance metrics
- Resource usage tracking and optimization
- Alert integration for failed applications

**Research Workflow Integration**:
- Integration with research computing platforms
- Workflow management system compatibility
- Data pipeline and processing tool integration

## Security Considerations

### Access Control
- **Authentication**: All API endpoints protected by authentication middleware
- **Authorization**: Instance-level permissions enforced for template operations
- **Audit Logging**: Complete history of template applications with user attribution

### Execution Security
- **Command Validation**: All remote commands validated before execution
- **Content Sanitization**: Template content sanitized to prevent injection attacks
- **Privilege Management**: Operations run with minimal required privileges

### Data Protection
- **Checkpoint Encryption**: Sensitive checkpoint data encrypted at rest
- **Network Security**: All remote communications encrypted (SSH/TLS)
- **Key Management**: Automatic SSH key rotation and secure storage

This template application system represents a significant advancement in CloudWorkstation's capabilities, transforming it from a simple instance launcher into a comprehensive research environment management platform. The system maintains CloudWorkstation's core principles of simplicity and reliability while providing powerful new capabilities for dynamic environment management.