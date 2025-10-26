# Multi-User Prism Planning (v0.5.0)

## Overview
Planning document for comprehensive multi-user support in Prism v0.5.0, enabling proper user identity management and consistent UID/GID mapping across instances for seamless EFS collaboration.

## Current State (v0.4.1)
- Single default user per template (ubuntu, rocky, ec2-user)
- Basic EFS mount/unmount functionality
- No centralized identity management
- UID/GID conflicts possible between different templates

## Scenario 2: Multi-User Architecture

### Core Components

#### 1. Centralized User Registry
```json
{
  "version": "1.0",
  "users": {
    "alice": {
      "uid": 2000,
      "gid": 2000,
      "primary_group": "alice",
      "groups": ["researchers", "prism-users"],
      "shell": "/bin/bash",
      "ssh_keys": ["ssh-rsa AAAAB3..."],
      "created": "2025-01-15T10:30:00Z"
    },
    "bob": {
      "uid": 2001,
      "gid": 2001,
      "primary_group": "bob", 
      "groups": ["researchers", "admins"],
      "shell": "/bin/zsh",
      "ssh_keys": ["ssh-ed25519 AAAAC3..."],
      "created": "2025-01-15T11:00:00Z"
    }
  },
  "groups": {
    "researchers": {"gid": 4000, "description": "Research team members"},
    "admins": {"gid": 4001, "description": "Administrative users"},
    "prism-users": {"gid": 3000, "description": "All CWS users"}
  }
}
```

#### 2. CLI Commands
```bash
# User management
prism user add <username> [--uid <uid>] [--groups <group1,group2>]
prism user remove <username> [--preserve-data]
prism user list [--instance <name>]
prism user info <username>
prism user modify <username> --add-group <group> | --remove-group <group>

# User provisioning  
prism user provision <username> <instance1,instance2,...>
prism user deprovision <username> <instance1,instance2,...>
prism user sync [--instance <name>]  # Sync registry to instance(s)

# SSH key management
prism user add-key <username> <public-key-file>
prism user remove-key <username> <key-fingerprint>
prism user list-keys <username>

# Group management
prism group add <groupname> [--gid <gid>]
prism group remove <groupname>
prism group list
prism group add-member <groupname> <username>
prism group remove-member <groupname> <username>
```

#### 3. Template Enhancements
```yaml
# Enhanced template structure
name: "Python Machine Learning (Multi-User)"
base_image: "ubuntu:22.04"
default_user: "ubuntu"  # Preserved for backwards compatibility

# Multi-user settings
multi_user:
  enabled: true
  default_groups: ["prism-users", "researchers"]
  home_base: "/home"
  efs_permissions: "group_shared"  # group_shared | user_private | posix_acl

packages:
  - acl                    # POSIX ACL support
  - quota                  # Disk quotas
  - amazon-efs-utils       # EFS mounting

groups:
  - name: prism-users
    gid: 3000
  - name: researchers
    gid: 4000

# EFS mount configuration for multi-user
efs_config:
  mount_options: "tls,_netdev,gid=3000"
  directory_structure: "user_subdirs"  # user_subdirs | shared | hybrid
  permissions: "0775"
  sticky_bit: true
```

#### 4. EFS Permission Strategies

**A) User Subdirectories (Default)**
```
/mnt/shared/
├── users/
│   ├── alice/           (2000:2000, 755)
│   ├── bob/             (2001:2001, 755)  
│   └── shared/          (root:researchers, 2775)
├── projects/
│   ├── project-alpha/   (varies by project owner)
│   └── project-beta/
└── scratch/             (root:prism-users, 1777) # temp space
```

**B) Group Shared**
```bash
# All files owned by researchers group
sudo mount -t efs fs-xxx:/ /mnt/shared -o tls,_netdev,gid=4000
sudo chmod g+rws /mnt/shared
umask 002  # Group writable by default
```

**C) POSIX ACLs (Advanced)**
```bash
# Fine-grained permissions
sudo setfacl -R -m g:researchers:rwx /mnt/shared
sudo setfacl -R -d -m g:researchers:rwx /mnt/shared
sudo setfacl -m u:alice:rwx /mnt/shared/projects/alice-project
```

#### 5. State Management
```json
// Enhanced Prism state
{
  "instances": {...},
  "volumes": {...},
  "users": {
    "registry_version": "1.0",
    "users": {...},
    "groups": {...},
    "instance_mappings": {
      "instance-1": ["alice", "bob"],
      "instance-2": ["alice", "shared-service"]
    }
  }
}
```

#### 6. API Endpoints
```
POST   /api/v1/users                    # Create user
GET    /api/v1/users                    # List users  
GET    /api/v1/users/{username}         # Get user info
PUT    /api/v1/users/{username}         # Update user
DELETE /api/v1/users/{username}         # Delete user

POST   /api/v1/users/{username}/provision/{instance}    # Provision user
DELETE /api/v1/users/{username}/provision/{instance}    # Deprovision user
POST   /api/v1/users/sync/{instance}                    # Sync users to instance

POST   /api/v1/groups                   # Create group
GET    /api/v1/groups                   # List groups
PUT    /api/v1/groups/{groupname}/members/{username}    # Add member
DELETE /api/v1/groups/{groupname}/members/{username}    # Remove member
```

### Implementation Phases

#### Phase 1: Core Infrastructure (v0.5.0-alpha)
- [ ] User registry system in state management
- [ ] Basic user/group CLI commands
- [ ] User provisioning via SSM
- [ ] Template multi-user flag support

#### Phase 2: EFS Integration (v0.5.0-beta)  
- [ ] Enhanced EFS mounting with user-aware permissions
- [ ] User subdirectory creation
- [ ] Group-based permission strategies
- [ ] EFS volume multi-user testing

#### Phase 3: Advanced Features (v0.5.0-rc)
- [ ] SSH key management
- [ ] POSIX ACL support
- [ ] User migration tools
- [ ] Comprehensive documentation

#### Phase 4: Production Ready (v0.5.0)
- [ ] GUI integration for user management
- [ ] Audit logging for user operations
- [ ] Backup/restore user registry
- [ ] Performance optimization

### Migration Strategy
1. **Backwards Compatibility**: Existing single-user instances continue working unchanged
2. **Opt-in Multi-User**: Templates explicitly enable multi-user with `multi_user.enabled: true`
3. **Gradual Adoption**: Users can migrate existing instances via `prism user sync`
4. **Data Preservation**: User removal preserves data by default with `--preserve-data` flag

### Security Considerations
- UID range reservation (2000-9999 for Prism users)
- SSH key validation and deduplication
- Audit trail for all user management operations
- Group membership verification before EFS access
- Secure user provisioning via encrypted SSM commands

### Testing Strategy
- Unit tests for user registry operations
- Integration tests for multi-instance user provisioning  
- EFS permission testing across different user scenarios
- Performance testing with 10+ users on multiple instances
- Security testing for privilege escalation attempts

## Success Metrics
- Multiple users can collaborate on shared EFS volumes seamlessly
- Consistent file permissions across all instances
- Zero UID/GID conflicts between templates
- Easy user onboarding/offboarding workflows
- Comprehensive audit trail for compliance

## Timeline Estimate
- **Research Phase**: 2 weeks (user requirements, security review)
- **Phase 1**: 4 weeks (core infrastructure)
- **Phase 2**: 3 weeks (EFS integration) 
- **Phase 3**: 3 weeks (advanced features)
- **Phase 4**: 2 weeks (production polish)
- **Total**: ~14 weeks for v0.5.0 release

## Dependencies
- Enhanced SSM integration (command batching)
- Template system extensions for user management
- State management schema migration
- API versioning for backwards compatibility

---

**Note**: This represents a significant architectural enhancement to Prism, transforming it from a single-user research tool into a collaborative multi-user platform while maintaining the core principle of "Default to Success" for simple use cases.