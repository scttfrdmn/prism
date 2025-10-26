# Prism Phase 5A: Research User Architecture

**Version**: v0.5.0 Implementation Guide
**Status**: Foundation Complete
**Date**: September 28, 2025

## Table of Contents

1. [Overview](#overview)
2. [The Dual User Problem](#the-dual-user-problem)
3. [Architecture Design](#architecture-design)
4. [Implementation Components](#implementation-components)
5. [UID/GID Management](#uidgid-management)
6. [SSH Key System](#ssh-key-system)
7. [Provisioning Pipeline](#provisioning-pipeline)
8. [Template Integration](#template-integration)
9. [EFS Home Directory Integration](#efs-home-directory-integration)
10. [Multi-Modal Interface Support](#multi-modal-interface-support)
11. [Security Considerations](#security-considerations)
12. [Performance and Scalability](#performance-and-scalability)
13. [Migration Strategy](#migration-strategy)
14. [Testing Strategy](#testing-strategy)
15. [Future Enhancements](#future-enhancements)

## Overview

Prism Phase 5A implements a **research user architecture** that provides persistent user identity across instances while maintaining template flexibility. This foundation enables collaborative research environments with consistent file permissions and cross-template compatibility.

### Key Achievements

- ✅ **Dual User System**: Combines template-created system users with persistent research users
- ✅ **Consistent UID/GID Mapping**: Deterministic user IDs across all instances
- ✅ **SSH Key Management**: Automated key generation, storage, and distribution
- ✅ **Provisioning Pipeline**: Remote user creation via secure shell scripts
- ✅ **EFS Integration**: Persistent home directories on EFS volumes
- ✅ **Profile Integration**: Seamless integration with existing profile system

## The Dual User Problem

### Current Challenge

Prism templates create various system users:

```yaml
# Python ML Template
users:
  - name: "researcher"  # UID varies by instance
    groups: ["sudo"]

# R Research Template
users:
  - name: "rstudio"     # Different user, different UID
    groups: ["sudo"]

# Rocky Linux Template
users:
  - name: "rocky"       # Yet another user/UID
    groups: ["wheel", "sudo"]
```

**Problems:**
- **Inconsistent Identity**: Different UIDs across templates
- **File Permission Issues**: Cannot share files between instances
- **SSH Key Management**: Need different keys for different templates
- **Collaboration Barriers**: Multiple users cannot easily share resources

### Phase 5A Solution

**Dual User System** separates concerns:

```
┌─────────────────┐    ┌─────────────────┐
│   System Users  │    │  Research Users │
├─────────────────┤    ├─────────────────┤
│ ubuntu (1000)   │    │ alice (5001)    │
│ researcher(1001)│    │ bob (5002)      │
│ Template-created│    │ Profile-based   │
│ Variable UIDs   │    │ Consistent UIDs │
│ Service-focused │    │ User-focused    │
└─────────────────┘    └─────────────────┘
```

**Benefits:**
- **Template Flexibility**: Each template creates appropriate service users
- **Research Continuity**: Same research user (alice:5001) across all instances
- **File Compatibility**: Consistent permissions enable EFS sharing
- **SSH Continuity**: Same keys work across all templates

## Architecture Design

### Component Architecture

```
pkg/research/
├── types.go          # Core data structures and interfaces
├── manager.go        # Research user lifecycle management
├── uid_mapping.go    # Consistent UID/GID allocation
├── provisioner.go    # Remote user provisioning via SSH
├── ssh_keys.go       # SSH key generation and management
└── integration.go    # High-level service layer
```

### Data Flow

```
Profile Selection → Research User Creation → UID/GID Allocation → SSH Key Setup → Template Launch → User Provisioning → EFS Integration
```

### Key Design Principles

1. **Profile-Centric**: Research users belong to Prism profiles
2. **Deterministic**: Same profile+username = same UID everywhere
3. **Template-Agnostic**: Works with any template system
4. **EFS-Ready**: Home directories designed for EFS persistence
5. **Security-First**: SSH keys managed securely per profile

## Implementation Components

### 1. Research User Configuration

```go
type ResearchUserConfig struct {
    Username          string `json:"username"`
    UID               int    `json:"uid"`              // Consistent across instances
    GID               int    `json:"gid"`
    HomeDirectory     string `json:"home_directory"`   // /efs/home/username
    EFSVolumeID       string `json:"efs_volume_id"`
    EFSMountPoint     string `json:"efs_mount_point"`
    SSHPublicKeys     []string `json:"ssh_public_keys"`
    SecondaryGroups   []string `json:"secondary_groups"` // research, efs-users
    ProfileOwner      string `json:"profile_owner"`
}
```

### 2. Dual User System Configuration

```go
type DualUserSystem struct {
    SystemUsers         []SystemUser `json:"system_users"`    // From template
    ResearchUser        *ResearchUserConfig `json:"research_user"` // Persistent
    PrimaryUser         string `json:"primary_user"`         // "research" or system user
    SharedDirectories   []string `json:"shared_directories"` // Cross-user access
    EnvironmentHandling EnvironmentPolicy `json:"environment_handling"`
}
```

### 3. System User Integration

```go
type SystemUser struct {
    Name            string `json:"name"`            // ubuntu, researcher, rocky
    UID             int    `json:"uid"`             // Variable by instance
    Purpose         string `json:"purpose"`         // system, jupyter, rstudio
    TemplateCreated bool   `json:"template_created"` // Created by template
}
```

## UID/GID Management

### Allocation Strategy

**Research User Range**: 5000-5999 (1000 users)
**System User Range**: 1000-4999 (templates)

### Deterministic Algorithm

```go
func allocateUIDGID(profileID, username string) (uid, gid int, err error) {
    // 1. Generate hash from profile + username
    input := fmt.Sprintf("%s:%s", profileID, username)
    hash := sha256.Sum256([]byte(input))

    // 2. Map to UID range
    offset := binary.BigEndian.Uint64(hash[:8])
    targetUID := ResearchUserBaseUID + int(offset % uidRange)

    // 3. Handle collisions
    uid = findNextAvailableUID(targetUID)
    gid = uid // GID matches UID for simplicity

    return uid, gid, nil
}
```

### Consistency Guarantees

- **Same Input → Same Output**: `alice@research-profile` always gets UID 5001
- **Collision Resolution**: If 5001 taken, increment to 5002, etc.
- **Cross-Instance Sync**: UID allocations cached and shared
- **Profile Isolation**: Different profiles can have same username with different UIDs

### UID Allocation Examples

```bash
# Profile: personal-research
alice → UID 5001 (deterministic hash)
bob   → UID 5023 (deterministic hash)
carol → UID 5067 (deterministic hash)

# Profile: lab-shared
alice → UID 5102 (different profile = different UID)
bob   → UID 5234 (different profile = different UID)
```

## SSH Key System

### Key Management Architecture

```
SSH Key Store
├── profiles/
│   ├── personal-research/
│   │   ├── alice/
│   │   │   ├── key1.pub
│   │   │   ├── key1.json (metadata)
│   │   │   └── key2.pub
│   │   └── bob/
│   │       └── key1.pub
│   └── lab-shared/
│       └── alice/
│           └── key1.pub
```

### Key Generation and Storage

```go
type SSHKeyConfig struct {
    KeyID         string    `json:"key_id"`
    Fingerprint   string    `json:"fingerprint"`
    PublicKey     string    `json:"public_key"`
    Comment       string    `json:"comment"`
    CreatedAt     time.Time `json:"created_at"`
    FromProfile   string    `json:"from_profile"`
    AutoGenerated bool      `json:"auto_generated"`
}
```

### Supported Key Types

- **Ed25519** (recommended): Modern, secure, fast
- **RSA 2048**: Compatibility with older systems
- **Import Support**: Import existing public keys

### Key Distribution

```bash
# Generated authorized_keys content
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... alice@prism-personal
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQc7... alice@external-key
```

## Provisioning Pipeline

### Remote Provisioning Process

```
1. Generate Provisioning Script
   ├── Create research groups (research, efs-users)
   ├── Create research user (consistent UID/GID)
   ├── Setup EFS home directory
   ├── Install SSH keys
   └── Configure environment

2. SSH Connection
   ├── Connect as system user (ubuntu)
   ├── Upload provisioning script
   └── Execute with sudo privileges

3. Verification
   ├── Check user creation
   ├── Verify EFS mount
   ├── Test SSH access
   └── Update usage tracking
```

### Generated Provisioning Script Example

```bash
#!/bin/bash
# Prism Research User Provisioning Script
# User: alice (UID: 5001)

# Create research groups
groupadd -g 5000 research || true
groupadd -g 5002 efs-users || true

# Create research user
useradd -m -u 5001 -g 5000 -G research,efs-users,sudo,docker \
        -s /bin/bash -c 'Alice Smith' alice || true

# Setup EFS home directory
mkdir -p /efs/home/alice
chown alice:research /efs/home/alice
chmod 750 /efs/home/alice

# Install SSH keys
mkdir -p /efs/home/alice/.ssh
echo 'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... alice@prism' >> /efs/home/alice/.ssh/authorized_keys
chmod 600 /efs/home/alice/.ssh/authorized_keys
chown -R alice:research /efs/home/alice/.ssh

echo 'Research user provisioning complete!'
```

### Asynchronous Job Management

```go
type ProvisioningJob struct {
    ID        string                    `json:"id"`
    Status    ProvisioningJobStatus     `json:"status"`
    Request   *UserProvisioningRequest  `json:"request"`
    Response  *UserProvisioningResponse `json:"response"`
    Progress  float64                   `json:"progress"` // 0.0 to 1.0
}
```

## Template Integration

### Template Enhancement Strategy

Templates can specify research user integration without breaking existing functionality:

```yaml
# Python ML Template Enhanced
name: "Python Machine Learning (Advanced)"
description: "Python + Jupyter + ML packages with research user support"

# Existing template configuration
users:
  - name: "researcher"
    groups: ["sudo"]

# New: Research user integration
research_user:
  auto_create: true                    # Auto-create research user
  default_shell: "/bin/bash"
  default_groups: ["research", "jupyter-users"]
  require_efs: true
  efs_mount_point: "/efs"
  user_integration:
    strategy: "research_primary"       # Research user is primary
    shared_directories: ["/opt/notebooks", "/home/shared"]
    service_ownership:
      jupyter: "researcher"            # Jupyter runs as researcher user
```

### Integration Strategies

1. **Research Primary**: Research user is primary, system users run services
2. **Coexist**: Both users coexist with shared access
3. **System First**: System user primary, research user secondary

### Template Migration

**Backward Compatibility**: Existing templates work unchanged
**Gradual Enhancement**: Templates can opt-in to research user features
**Dual Support**: Templates support both modes simultaneously

## EFS Home Directory Integration

### Home Directory Structure

```
/efs/                               # EFS mount point
├── home/                           # Research user homes
│   ├── alice/                      # alice's persistent home
│   │   ├── .bashrc
│   │   ├── .ssh/
│   │   │   └── authorized_keys
│   │   ├── projects/               # Research projects
│   │   │   ├── ml-analysis/
│   │   │   └── data-processing/
│   │   └── research/               # Shared research data
│   └── bob/                        # bob's persistent home
│       ├── .bashrc
│       └── projects/
└── shared/                         # Collaborative spaces
    ├── datasets/                   # Shared datasets
    ├── libraries/                  # Shared code libraries
    └── notebooks/                  # Shared Jupyter notebooks
```

### EFS Permissions

```bash
# EFS root permissions
/efs → root:efs-users (755)

# User home directories
/efs/home/alice → alice:research (750)
/efs/home/bob   → bob:research   (750)

# Shared directories
/efs/shared → root:research (755)
```

### Cross-Instance Persistence

**Scenario**: Alice works on Python ML instance, then launches R Research instance

```
Python Instance (Day 1):
- alice (5001) creates /efs/home/alice/projects/analysis.py
- File ownership: alice:research (5001:5000)

R Instance (Day 2):
- alice (5001) sees same file with correct ownership
- Can immediately access and modify analysis.py
- No permission issues, seamless transition
```

## Multi-Modal Interface Support

### CLI Integration

```bash
# Research user management
prism research-user create alice
prism research-user list
prism research-user delete alice

# SSH key management
prism research-user ssh-key generate alice ed25519
prism research-user ssh-key import alice ~/.ssh/id_rsa.pub
prism research-user ssh-key list alice

# Instance provisioning
prism launch python-ml my-instance --research-user alice
prism research-user provision alice --instance my-instance

# Status and monitoring
prism research-user status alice --instance my-instance
prism research-user list-instances alice
```

### TUI Integration

```
┌─ Research Users ─────────────────────────────────┐
│ Profile: personal-research                       │
├─────────────────────────────────────────────────┤
│ ► alice (UID: 5001)     [2 SSH keys] [Active]   │
│   bob   (UID: 5023)     [1 SSH key]  [Inactive] │
│                                                  │
│ Actions: [C]reate [D]elete [S]SH Keys [P]rovision│
└─────────────────────────────────────────────────┘
```

### GUI Integration

```
Research Users Tab
├── User List (Table)
│   ├── Username | UID  | SSH Keys | Last Used | Actions
│   ├── alice    | 5001 | 2        | 1h ago    | [Edit] [Delete]
│   └── bob      | 5023 | 1        | 3d ago    | [Edit] [Delete]
├── [+ Create User] [Import SSH Key]
└── User Details Panel
    ├── SSH Keys Management
    ├── Instance History
    └── EFS Usage Statistics
```

## Security Considerations

### UID/GID Security

- **Range Isolation**: Research users (5000-5999) isolated from system (0-999)
- **Deterministic but Secure**: Hash-based allocation prevents UID prediction attacks
- **Profile Isolation**: Different profiles cannot access each other's resources
- **Collision Handling**: Secure fallback prevents UID conflicts

### SSH Key Security

- **Per-Profile Storage**: SSH keys isolated by Prism profile
- **Secure Generation**: Ed25519/RSA keys generated with cryptographically secure randomness
- **Fingerprint Validation**: All keys validated and fingerprinted
- **Key Rotation**: Support for key replacement and rotation

### EFS Permissions

- **Home Directory Isolation**: Each user's home (750) prevents cross-user access
- **Group-Based Sharing**: Collaborative access via `research` group
- **Root Ownership**: EFS mount owned by root prevents user modification

### Provisioning Security

- **SSH-Based**: All provisioning via encrypted SSH connections
- **Sudo Required**: User creation requires explicit sudo privileges
- **Script Validation**: Generated scripts validated before execution
- **Connection Verification**: SSH connections verified before provisioning

## Performance and Scalability

### UID Allocation Performance

- **O(1) Average Case**: Hash-based allocation typically single lookup
- **O(n) Worst Case**: Linear scan for collision resolution
- **Caching**: UID allocations cached in memory for fast repeated access
- **Scalability**: Supports 1000 research users per installation

### SSH Key Management

- **File-Based Storage**: Keys stored as individual files for fast access
- **Lazy Loading**: Key configs loaded on demand
- **Indexing**: Fingerprint-based indexing for fast key lookup
- **Cleanup**: Automatic cleanup of unused key metadata

### Provisioning Performance

- **Parallel Provisioning**: Multiple users can be provisioned simultaneously
- **Script Optimization**: Generated scripts optimized for minimal remote execution time
- **Connection Pooling**: SSH connections reused where possible
- **Background Jobs**: Long-running provisioning handled asynchronously

### Memory Usage

- **Lazy Loading**: Components loaded only when needed
- **Memory-Efficient**: Minimal memory footprint for UID/GID tracking
- **Garbage Collection**: Unused allocations cleaned up periodically

## Migration Strategy

### Phase 1: Foundation (Current)

- ✅ Core research user architecture implemented
- ✅ UID/GID allocation system complete
- ✅ SSH key management functional
- ✅ Provisioning pipeline operational

### Phase 2: Integration (Next)

- [ ] CLI command integration (`prism research-user`)
- [ ] TUI interface for research user management
- [ ] GUI screens for visual management
- [ ] Template system integration

### Phase 3: Enhancement

- [ ] Automated EFS volume creation
- [ ] Advanced policy framework
- [ ] Multi-profile collaboration
- [ ] Globus Auth integration (optional)

### Backward Compatibility

- **Existing Templates**: Continue to work unchanged
- **Current Instances**: No impact on running instances
- **Profile System**: Existing profiles work with research users
- **Migration Path**: Gradual adoption without breaking changes

## Testing Strategy

### Unit Testing

```go
// UID/GID allocation testing
func TestUIDGIDAllocation(t *testing.T) {
    allocator := NewUIDGIDAllocator()

    // Test deterministic allocation
    uid1, gid1, _ := allocator.AllocateUIDGID("profile1", "alice")
    uid2, gid2, _ := allocator.AllocateUIDGID("profile1", "alice")

    assert.Equal(t, uid1, uid2) // Same input = same output
    assert.Equal(t, gid1, gid2)
}

// SSH key generation testing
func TestSSHKeyGeneration(t *testing.T) {
    keyMgr := NewSSHKeyManager("/tmp/test-keys")

    config, privateKey, err := keyMgr.GenerateSSHKeyPair("profile1", "alice", "ed25519")
    assert.NoError(t, err)
    assert.NotEmpty(t, config.PublicKey)
    assert.NotEmpty(t, privateKey)
}
```

### Integration Testing

```bash
# Test complete research user lifecycle
./test/integration/research_user_lifecycle_test.sh

# Test cross-instance consistency
./test/integration/cross_instance_consistency_test.sh

# Test EFS integration
./test/integration/efs_home_directory_test.sh
```

### User Acceptance Testing

- **Template Compatibility**: Verify existing templates work unchanged
- **Multi-Instance Workflow**: Test user experience across different instances
- **Collaborative Access**: Test multiple users sharing EFS volumes
- **Performance**: Measure provisioning time and resource usage

## Future Enhancements

### Phase 5B: AWS Research Services Integration

- **SageMaker Studio Integration**: Unified cost tracking across EC2 and SageMaker
- **Web Service Framework**: Common interface for web-based research services
- **QuickSight Integration**: Research data visualization and dashboards

### Phase 5C: Enterprise Features

- **Advanced Policy Engine**: Digital signatures and institutional controls
- **Template Marketplace**: Community template sharing with research user support
- **HPC Integration**: ParallelCluster and Batch scheduling with research users

### Advanced Research User Features

- **Multi-Profile Collaboration**: Research users shared across profiles
- **Automated Backup**: EFS home directory backup and versioning
- **Usage Analytics**: Detailed research user activity tracking
- **Resource Quotas**: Per-user storage and compute quotas

### Performance Optimizations

- **Database Storage**: Move from file-based to database storage for large deployments
- **Caching Layer**: Redis-based caching for frequently accessed data
- **Batch Operations**: Bulk user operations for institutional deployments

## Conclusion

Prism Phase 5A Research User Architecture provides a solid foundation for collaborative research computing. The dual user system successfully separates template flexibility from research user continuity, enabling seamless multi-instance workflows while maintaining backward compatibility.

**Key Success Metrics:**
- ✅ **Consistent Identity**: Same UID/GID across all instances
- ✅ **Template Flexibility**: Works with any template system
- ✅ **EFS Integration**: Persistent home directories ready
- ✅ **SSH Management**: Automated key generation and distribution
- ✅ **Profile Integration**: Seamless existing system integration

This architecture enables Prism's evolution from individual research tool to collaborative research platform, supporting institutional deployment while preserving its core simplicity.

---

**Document Version**: 1.0
**Last Updated**: September 28, 2025
**Implementation Status**: Foundation Complete
**Next Milestone**: CLI/TUI/GUI Integration (Phase 5A.2)