# Daemon API Integration for Template Application

## Overview

The CloudWorkstation daemon now includes complete API support for applying templates to running instances. This integration provides server-side endpoints that the CLI and GUI clients can use to perform template operations through the REST API.

## API Endpoints

### 1. Apply Template to Running Instance

**Endpoint**: `POST /api/v1/templates/apply`

**Description**: Applies a template to an already running instance with incremental package installation, service configuration, and user management.

**Request Body**:
```json
{
  "instance_name": "my-workspace",
  "template": {
    "name": "python-ml",
    "description": "Python machine learning environment",
    "packages": {
      "conda": ["tensorflow", "pytorch", "scikit-learn"],
      "pip": ["jupyter", "matplotlib"]
    },
    "services": [
      {
        "name": "jupyter",
        "port": 8888,
        "enable": true
      }
    ],
    "users": [
      {
        "name": "researcher",
        "groups": ["sudo", "docker"]
      }
    ]
  },
  "package_manager": "conda",
  "dry_run": false,
  "force": false
}
```

**Response**:
```json
{
  "success": true,
  "message": "Successfully applied template 'python-ml' to instance 'my-workspace'",
  "packages_installed": 15,
  "services_configured": 1,
  "users_created": 1,
  "rollback_checkpoint": "checkpoint-1640995200",
  "warnings": [],
  "execution_time": "45.2s"
}
```

### 2. Calculate Template Differences

**Endpoint**: `POST /api/v1/templates/diff`

**Description**: Calculates the differences between current instance state and desired template configuration without applying changes.

**Request Body**:
```json
{
  "instance_name": "my-workspace",
  "template": {
    "name": "python-ml",
    "packages": {
      "conda": ["tensorflow", "pytorch"]
    }
  }
}
```

**Response**:
```json
{
  "packages_to_install": [
    {
      "name": "tensorflow",
      "target_version": "2.8.0",
      "action": "install",
      "package_manager": "conda"
    }
  ],
  "services_to_configure": [],
  "users_to_create": [],
  "conflicts_found": []
}
```

### 3. List Applied Template Layers

**Endpoint**: `GET /api/v1/instances/{instance_name}/layers`

**Description**: Returns the history of templates applied to an instance with rollback checkpoints.

**Response**:
```json
[
  {
    "name": "base-python",
    "applied_at": "2024-01-15T10:30:00Z",
    "package_manager": "conda",
    "packages_installed": ["python", "jupyter"],
    "services_configured": ["jupyter"],
    "users_created": [],
    "rollback_checkpoint": "checkpoint-1640991000"
  },
  {
    "name": "ml-stack",
    "applied_at": "2024-01-15T14:20:00Z",
    "package_manager": "conda", 
    "packages_installed": ["tensorflow", "pytorch"],
    "services_configured": [],
    "users_created": [],
    "rollback_checkpoint": "checkpoint-1640995200"
  }
]
```

### 4. Rollback Template Applications

**Endpoint**: `POST /api/v1/instances/{instance_name}/rollback`

**Description**: Rolls back an instance to a previous state by undoing template applications.

**Request Body**:
```json
{
  "instance_name": "my-workspace",
  "checkpoint_id": "checkpoint-1640991000"
}
```

**Response**:
```json
{
  "success": true,
  "message": "Successfully rolled back instance 'my-workspace' to checkpoint 'checkpoint-1640991000'"
}
```

## Architecture Integration

### Server Structure

The daemon API integration consists of several key components:

**Handler Functions** (`template_application_handlers.go`):
- `handleTemplateApply()` - Orchestrates template application workflow
- `handleTemplateDiff()` - Calculates template differences 
- `handleInstanceLayers()` - Returns applied template history
- `handleInstanceRollback()` - Performs checkpoint-based rollback

**Route Registration** (`server.go`):
```go
// Template application operations
mux.HandleFunc("/api/v1/templates/apply", applyMiddleware(s.handleTemplateApply))
mux.HandleFunc("/api/v1/templates/diff", applyMiddleware(s.handleTemplateDiff))

// Instance-specific operations (added to existing handler)
// /api/v1/instances/{name}/layers
// /api/v1/instances/{name}/rollback
```

**Remote Execution Integration**:
- `createRemoteExecutor()` - Creates appropriate executor based on instance connectivity
- SSH executor for instances with public IPs
- Systems Manager executor for private instances
- Automatic key path and username resolution

### State Management

**Instance State Enhancement**:
```go
type Instance struct {
    // ... existing fields
    AppliedTemplates []AppliedTemplateRecord `json:"applied_templates,omitempty"`
}

type AppliedTemplateRecord struct {
    TemplateName       string    `json:"template_name"`
    AppliedAt          time.Time `json:"applied_at"`
    PackageManager     string    `json:"package_manager"`
    PackagesInstalled  []string  `json:"packages_installed"`
    ServicesConfigured []string  `json:"services_configured"`
    UsersCreated       []string  `json:"users_created"`
    RollbackCheckpoint string    `json:"rollback_checkpoint"`
}
```

**Persistent Storage**:
- Template application history stored in CloudWorkstation state file
- Rollback checkpoints maintained on instances at `/opt/cloudworkstation/checkpoints/`
- Automatic state synchronization between daemon and instances

## Request Validation

### Security Checks
- Verify instance exists and is in running state
- Validate template structure and required fields
- Ensure user has permissions for instance operations
- Sanitize template content to prevent code injection

### Error Handling
- **400 Bad Request**: Invalid request structure or missing required fields
- **404 Not Found**: Instance not found or not running
- **500 Internal Server Error**: Template application failures with detailed error messages
- **503 Service Unavailable**: Remote executor connection failures

## Remote Execution Strategy

### Connection Method Selection

**Public Instances**:
```go
if instance.PublicIP != "" {
    keyPath := s.getSSHKeyPath()
    username := s.getSSHUsername(instance)
    return templates.NewSSHRemoteExecutor(keyPath, username), nil
}
```

**Private Instances**:
```go
region := s.getAWSRegion()
return templates.NewSystemsManagerExecutor(region), nil
```

### Configuration Management

**SSH Key Resolution**:
- Check CloudWorkstation configuration for instance-specific keys
- Fall back to default AWS key pairs
- Support per-template SSH user configuration

**Systems Manager Integration**:
- Use IAM roles for authentication
- Support cross-region instance connections
- Handle temporary credential management

## Integration with Template Engine

### Component Initialization
```go
func (s *Server) handleTemplateApply(w http.ResponseWriter, r *http.Request) {
    // Create executor based on instance connectivity
    executor, err := s.createRemoteExecutor(instance)
    
    // Initialize template application engine
    engine := templates.NewTemplateApplicationEngine(executor)
    
    // Apply template with full workflow
    response, err := engine.ApplyTemplate(r.Context(), req)
}
```

### Workflow Integration
1. **Request Validation** - Validate instance state and template structure
2. **Executor Creation** - Choose SSH or Systems Manager based on connectivity
3. **Engine Initialization** - Create template application engine with executor
4. **Template Application** - Run complete workflow with rollback protection
5. **State Recording** - Update daemon state with applied template information

## Error Recovery

### Rollback on Failure
- Automatic rollback checkpoint creation before template application
- Failed applications trigger automatic rollback to previous state
- Detailed error reporting with suggested recovery actions

### State Consistency
- Daemon state updates only on successful template application
- Checkpoint cleanup on successful rollback operations
- Instance state synchronization on connection recovery

## Performance Considerations

### Concurrent Operations
- Multiple template applications can run concurrently on different instances
- Operation tracking prevents conflicting template applications on same instance
- Resource limits prevent system overload during large template applications

### Caching Strategy
- Template resolution cached for repeated applications
- Instance state inspection cached for diff calculations
- Remote executor connection pooling for performance

## Future Enhancements

### Advanced Features (Ready for Implementation)
1. **Template Scheduling** - Apply templates at specified times
2. **Batch Operations** - Apply templates to multiple instances simultaneously
3. **Template Dependencies** - Automatic prerequisite template application
4. **Cost Estimation** - Preview cost impact of template applications

### Monitoring Integration
1. **Operation Metrics** - Track template application success rates and duration
2. **Resource Usage** - Monitor resource consumption during template applications
3. **Alert Integration** - Notifications for failed template applications

## Security Considerations

### Access Control
- API endpoints protected by existing authentication middleware
- Instance-level permissions enforced for template operations
- Template content validation prevents malicious code execution

### Audit Logging
- All template applications logged with user attribution
- Rollback operations tracked for compliance
- Failed operations logged with detailed error information

## Testing Strategy

### Integration Testing
- End-to-end workflow testing with real AWS instances
- Mock executor testing for development environments
- Error condition testing with simulated failures

### Performance Testing
- Large template application performance validation
- Concurrent operation stress testing
- Network failure recovery testing

This daemon API integration provides a production-ready foundation for template application operations, enabling the CLI and GUI clients to perform sophisticated environment management through a secure, well-validated REST API.