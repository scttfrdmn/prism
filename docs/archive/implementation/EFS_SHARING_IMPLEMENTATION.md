# EFS Multi-Instance Sharing Implementation

## Overview

Prism now supports seamless EFS volume sharing between multiple instances with different default users through an enhanced mount system with shared group permissions.

## Scenario 1: Basic Multi-Instance Sharing (Implemented)

### Key Features

**🤝 Unified Permission System**
- `prism-shared` group (GID: 3000) for consistent permissions
- Automatic user provisioning to shared group during mount operations
- Group sticky bit and umask configuration for collaborative file creation

**📁 Structured Directory Layout**
```
/mnt/shared-volume/
├── shared/              # Collaborative space (all users)
├── users/
│   ├── ubuntu/          # Ubuntu user personal space
│   ├── rocky/           # Rocky user personal space
│   └── {username}/      # Other user personal spaces
└── (root level files)   # Shared files at mount root
```

**🔧 Automated Setup**
- Automatic `amazon-efs-utils` installation (yum/apt support)
- Persistent mounting via `/etc/fstab` with group ownership
- Shell umask (002) configuration for group-friendly file permissions
- Comprehensive error handling and user feedback

### Usage

```bash
# Create shared EFS volume
prism volume create research-data

# Mount to multiple instances with different default users
prism volume mount research-data ubuntu-instance
prism volume mount research-data rocky-instance

# Both instances now share the same EFS with proper permissions
# Files created by either user are accessible to both via shared group
```

### Technical Implementation

#### Enhanced Mount Script (pkg/aws/manager.go:700-762)

The EFS mount operation executes a comprehensive 30-line script on each instance:

1. **Package Installation**: Installs `amazon-efs-utils` if not present
2. **Group Creation**: Creates `prism-shared` group (GID: 3000)
3. **User Provisioning**: Adds current user to shared group
4. **Directory Structure**: Creates mount point and subdirectories
5. **EFS Mounting**: Mounts with TLS encryption and group ownership
6. **Permission Setup**: Configures group sticky bit (2775) and ownership
7. **Persistence**: Adds to `/etc/fstab` for automatic remounting
8. **Shell Configuration**: Sets umask 002 for group-friendly files

#### Key Code Components

**Mount Script Structure:**
```bash
#!/bin/bash
# Install EFS utils if needed
# Create prism-shared group (gid: 3000)  
# Add current user to shared group
# Mount EFS with group ownership
# Set sticky bit permissions (2775)
# Create /shared and /users/{username} subdirectories
# Configure persistent mounting and umask
```

**API Integration:**
- REST endpoint: `POST /api/v1/volumes/{name}/mount`
- CLI command: `prism volume mount <volume-name> <instance-name>`
- SSM-based remote execution with comprehensive error handling

### Directory Permissions

| Path | Owner | Group | Permissions | Purpose |
|------|-------|-------|-------------|---------|
| `/mnt/shared-volume/` | root | prism-shared | 2775 | Mount root |
| `/mnt/shared-volume/shared/` | root | prism-shared | 2775 | Collaboration |
| `/mnt/shared-volume/users/` | root | prism-shared | 2755 | User container |
| `/mnt/shared-volume/users/{user}/` | user | prism-shared | 755 | Personal space |

### Multi-User Collaboration Examples

**Cross-Instance File Sharing:**
```bash
# On ubuntu-instance (ubuntu user)
echo "Research data" > /mnt/shared-volume/shared/experiment.txt

# On rocky-instance (rocky user) - file is immediately accessible
cat /mnt/shared-volume/shared/experiment.txt
# Output: Research data
```

**Personal vs Shared Spaces:**
```bash
# Personal space (private to user)
/mnt/shared-volume/users/ubuntu/my-private-notes.txt

# Shared space (accessible to all users)
/mnt/shared-volume/shared/team-results.csv

# Root level (accessible to all users)
/mnt/shared-volume/dataset.zip
```

### Benefits

1. **Template Independence**: Works across different templates (Ubuntu, Rocky, etc.)
2. **Automatic Setup**: No manual user management required
3. **Persistent Configuration**: Survives instance restarts
4. **Security**: Maintains user isolation while enabling collaboration
5. **Flexibility**: Supports both shared and private file areas

## Future: Scenario 2 (v0.5.0 Planning)

See `docs/MULTI_USER_PLANNING_v0.5.0.md` for comprehensive multi-user architecture with:
- Centralized user registry with consistent UID/GID mapping
- Advanced CLI commands for user/group management
- SSH key provisioning and POSIX ACL support
- Enterprise features and audit logging

## Implementation Status

✅ **Scenario 1**: Basic multi-instance sharing (COMPLETE)
- Enhanced EFS mount script with shared group system
- Automatic user provisioning and directory structure
- Cross-template compatibility (ubuntu, rocky, etc.)

🎯 **Scenario 2**: Comprehensive multi-user system (Planned v0.5.0)
- Centralized identity management
- Advanced permission strategies
- Enterprise collaboration features

## Testing

The implementation has been tested with:
- Ubuntu instances (default user: ubuntu)
- Rocky Linux instances (default user: rocky)
- EFS volume creation, mounting, and file sharing
- Permission verification across different users

All scenarios work correctly with proper file sharing and permission management between instances with different default users.