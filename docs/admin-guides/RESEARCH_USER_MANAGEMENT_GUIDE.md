# Research User Management Guide

**Prism v0.5.0 Administrator and Power User Guide**

This guide provides comprehensive information for managing research users in Prism environments, including setup, administration, troubleshooting, and best practices for individual, team, and institutional deployments.

## Table of Contents

1. [Quick Reference](#quick-reference)
2. [Setup and Configuration](#setup-and-configuration)
3. [User Management Operations](#user-management-operations)
4. [SSH Key Management](#ssh-key-management)
5. [EFS Integration](#efs-integration)
6. [Instance Provisioning](#instance-provisioning)
7. [Multi-User Collaboration](#multi-user-collaboration)
8. [Monitoring and Analytics](#monitoring-and-analytics)
9. [Troubleshooting](#troubleshooting)
10. [Security Best Practices](#security-best-practices)
11. [Institutional Deployment](#institutional-deployment)
12. [Advanced Configuration](#advanced-configuration)

## Quick Reference

### Essential Commands (v0.5.0)

```bash
# Research User Management
prism research-user create <username>                    # Create research user
prism research-user list                                 # List all research users
prism research-user status <username>                   # Check user status
prism research-user delete <username>                   # Remove research user

# SSH Key Management
prism research-user ssh-key generate <username> ed25519 # Generate SSH key pair
prism research-user ssh-key import <username> <pubkey>  # Import existing key
prism research-user ssh-key list <username>             # List user's keys
prism research-user ssh-key delete <username> <key-id>  # Remove SSH key

# Instance Integration
prism launch <template> <instance> --research-user <username>  # Launch with research user
prism research-user provision <username> --instance <name>     # Provision user on instance
prism research-user status <username> --instance <name>        # Check user on instance

# EFS and Storage
prism volumes create <name>                             # Create EFS volume
prism volumes mount <volume> <instance>                 # Mount EFS volume
prism volumes list                                      # List EFS volumes
```

### Key File Locations

```
~/.prism/
├── research-users/           # Research user configurations
│   └── <profile-id>/
│       └── <username>.json   # User config
├── ssh-keys/                 # SSH key storage
│   └── <profile-id>/
│       └── <username>/
│           ├── <key-id>.pub  # Public key
│           └── <key-id>.json # Key metadata
└── uid-allocations.json      # UID/GID allocation cache
```

### UID/GID Ranges

- **System Users**: 1000-4999 (templates and system accounts)
- **Research Users**: 5000-5999 (persistent research identities)
- **Research Groups**: 5000-5099 (research, efs-users, etc.)

## Setup and Configuration

### Initial Setup

1. **Verify Prism Installation**
   ```bash
   prism --version
   # Should show v0.5.0 or later for research user support
   ```

2. **Check Profile Configuration**
   ```bash
   prism profiles list
   prism profiles show current
   ```

3. **Initialize Research User System**
   ```bash
   # Create base directories (automatic on first use)
   mkdir -p ~/.prism/research-users
   mkdir -p ~/.prism/ssh-keys
   ```

### Configuration Files

#### Research User Configuration

**Location**: `~/.prism/research-users/<profile-id>/<username>.json`

```json
{
  "username": "alice",
  "uid": 5001,
  "gid": 5000,
  "full_name": "Alice Smith",
  "email": "alice@example.com",
  "home_directory": "/efs/home/alice",
  "efs_volume_id": "fs-1234567890abcdef0",
  "efs_mount_point": "/efs",
  "shell": "/bin/bash",
  "create_home_dir": true,
  "ssh_public_keys": [
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... alice@prism"
  ],
  "secondary_groups": ["research", "efs-users", "sudo", "docker"],
  "sudo_access": true,
  "docker_access": true,
  "created_at": "2025-09-28T10:30:00Z",
  "profile_owner": "personal-research"
}
```

#### SSH Key Configuration

**Location**: `~/.prism/ssh-keys/<profile-id>/<username>/<key-id>.json`

```json
{
  "key_id": "alice-ed25519-1727519400",
  "fingerprint": "SHA256:abc123def456...",
  "public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... alice@prism",
  "comment": "alice@prism-personal",
  "created_at": "2025-09-28T10:30:00Z",
  "from_profile": "personal-research",
  "auto_generated": true
}
```

## User Management Operations

### Creating Research Users

#### Basic User Creation

```bash
# Create research user with automatic UID assignment
prism research-user create alice

# Create with custom full name and email
prism research-user create alice --full-name "Alice Smith" --email "alice@university.edu"

# Create with SSH key generation
prism research-user create alice --generate-ssh-key
```

#### Advanced User Creation

```bash
# Create with specific shell
prism research-user create alice --shell /bin/zsh

# Create with custom groups
prism research-user create alice --groups research,docker,jupyter-users

# Create with EFS configuration
prism research-user create alice --efs-volume fs-1234567890abcdef0 --efs-mount /efs
```

### Modifying Research Users

```bash
# Update user information
prism research-user update alice --full-name "Dr. Alice Smith"
prism research-user update alice --email "alice.smith@university.edu"

# Add/remove groups
prism research-user update alice --add-groups jupyter-users
prism research-user update alice --remove-groups docker

# Change shell
prism research-user update alice --shell /bin/zsh
```

### Listing and Inspecting Users

```bash
# List all research users in current profile
prism research-user list

# List with detailed information
prism research-user list --detailed

# Show specific user information
prism research-user show alice

# Show user with SSH keys and instance history
prism research-user show alice --include-keys --include-instances
```

#### Example Output

```
Research Users (Profile: personal-research)
┌──────────┬─────┬─────┬────────────┬─────────────────┬────────────┐
│ Username │ UID │ GID │ SSH Keys   │ Last Used       │ Instances  │
├──────────┼─────┼─────┼────────────┼─────────────────┼────────────┤
│ alice    │5001 │5000 │ 2          │ 2h ago          │ 3          │
│ bob      │5023 │5000 │ 1          │ 1d ago          │ 1          │
│ carol    │5067 │5000 │ 3          │ 5d ago          │ 0          │
└──────────┴─────┴─────┴────────────┴─────────────────┴────────────┘
```

### Deleting Research Users

```bash
# Delete user (with confirmation)
prism research-user delete alice

# Force delete without confirmation
prism research-user delete alice --force

# Delete user and clean up SSH keys
prism research-user delete alice --cleanup-keys

# Delete user but preserve EFS home directory
prism research-user delete alice --preserve-home
```

## SSH Key Management

### Key Generation

```bash
# Generate Ed25519 key (recommended)
prism research-user ssh-key generate alice ed25519

# Generate RSA key for compatibility
prism research-user ssh-key generate alice rsa

# Generate with custom comment
prism research-user ssh-key generate alice ed25519 --comment "alice-laptop-2025"
```

### Key Import and Export

```bash
# Import existing public key
prism research-user ssh-key import alice ~/.ssh/id_rsa.pub

# Import with custom comment
prism research-user ssh-key import alice ~/.ssh/id_rsa.pub --comment "Personal laptop key"

# Export all keys for backup
prism research-user ssh-key export alice --output alice-keys-backup.tar.gz

# Export single key
prism research-user ssh-key export alice --key-id <key-id> --output alice-key.pub
```

### Key Management

```bash
# List all SSH keys for user
prism research-user ssh-key list alice

# Show detailed key information
prism research-user ssh-key show alice <key-id>

# Delete specific key
prism research-user ssh-key delete alice <key-id>

# Rotate keys (generate new, deactivate old)
prism research-user ssh-key rotate alice ed25519
```

#### SSH Key Listing Example

```
SSH Keys for alice (Profile: personal-research)
┌────────────────────────┬─────────────────────────────┬─────────────┬───────────────┐
│ Key ID                 │ Fingerprint                 │ Created     │ Last Used     │
├────────────────────────┼─────────────────────────────┼─────────────┼───────────────┤
│ alice-ed25519-12345    │ SHA256:abc123def456...      │ 2025-09-25  │ 2h ago        │
│ alice-rsa-67890        │ SHA256:def456ghi789...      │ 2025-09-20  │ 1d ago        │
│ imported-abc123        │ SHA256:ghi789jkl012...      │ 2025-09-15  │ 1w ago        │
└────────────────────────┴─────────────────────────────┴─────────────┴───────────────┘
```

### Authorized Keys Generation

```bash
# Generate authorized_keys content for user
prism research-user ssh-key authorized-keys alice

# Save to file
prism research-user ssh-key authorized-keys alice > alice_authorized_keys

# Generate for multiple users
prism research-user ssh-key authorized-keys alice,bob,carol > team_authorized_keys
```

## EFS Integration

### EFS Volume Management

```bash
# Create EFS volume for research users
prism volumes create research-home --type efs --performance generalPurpose

# Create high-performance EFS for shared data
prism volumes create shared-datasets --type efs --performance provisioned --throughput 500

# List EFS volumes
prism volumes list --type efs
```

### Home Directory Setup

```bash
# Configure research user with EFS home
prism research-user create alice --efs-volume research-home --efs-mount /efs

# Update existing user with EFS
prism research-user update alice --efs-volume research-home --efs-mount /efs

# Create home directory structure
prism research-user setup-home alice --create-directories projects,scratch,archive
```

### EFS Mounting and Permissions

```bash
# Mount EFS volume to instance
prism volumes mount research-home my-instance --mount-point /efs

# Check mount status
prism volumes status research-home

# Set up permissions for research users
prism research-user setup-efs-permissions alice --volume research-home
```

#### EFS Directory Structure

```
/efs/                                    # EFS mount point
├── home/                                # Research user homes (755 root:efs-users)
│   ├── alice/                           # alice's home (750 alice:research)
│   │   ├── .bashrc                      # Personal shell config
│   │   ├── .ssh/                        # SSH keys (700 alice:research)
│   │   │   └── authorized_keys          # (600 alice:research)
│   │   ├── projects/                    # Research projects
│   │   │   ├── ml-analysis/
│   │   │   └── data-processing/
│   │   ├── scratch/                     # Temporary work
│   │   └── archive/                     # Completed projects
│   └── bob/                             # bob's home (750 bob:research)
└── shared/                              # Shared directories (755 root:research)
    ├── datasets/                        # Shared datasets (775 root:research)
    ├── libraries/                       # Code libraries (775 root:research)
    ├── notebooks/                       # Jupyter notebooks (775 root:research)
    └── team-projects/                   # Collaborative projects (775 root:research)
        ├── project-alpha/               # Specific project (775 root:research)
        └── project-beta/
```

## Instance Provisioning

### Provisioning Research Users

```bash
# Provision research user on existing instance
prism research-user provision alice --instance my-python-instance

# Provision with custom EFS mount
prism research-user provision alice --instance my-instance --efs-volume research-data --mount-point /data

# Provision multiple users
prism research-user provision alice,bob,carol --instance shared-instance

# Provision with specific SSH user
prism research-user provision alice --instance my-instance --ssh-user ubuntu --ssh-key ~/.ssh/my-key
```

### Instance Launch with Research Users

```bash
# Launch instance with research user
prism launch "Python Machine Learning" ml-work --research-user alice

# Launch with multiple research users
prism launch "R Research Environment" shared-analysis --research-users alice,bob

# Launch with EFS volume
prism launch "Python ML" gpu-training --research-user alice --efs-volume shared-datasets
```

### Provisioning Status and Monitoring

```bash
# Check research user status on instance
prism research-user status alice --instance ml-work

# Monitor provisioning progress
prism research-user provision-status <job-id>

# List all research user instances
prism research-user instances alice

# Check what instances a user can access
prism research-user list-access alice
```

#### Status Output Example

```
Research User Status: alice on ml-work
┌─────────────────┬─────────────────────────────────────────┐
│ Property        │ Value                                   │
├─────────────────┼─────────────────────────────────────────┤
│ Username        │ alice                                   │
│ UID/GID         │ 5001/5000                              │
│ Home Directory  │ /efs/home/alice                        │
│ EFS Mounted     │ Yes (/efs)                             │
│ SSH Accessible  │ Yes (2 keys configured)                │
│ Last Login      │ 2025-09-28 10:45:00                   │
│ Active Processes│ 3                                      │
│ Disk Usage      │ 2.3 GB                                 │
│ Instance Uptime │ 5h 23m                                 │
└─────────────────┴─────────────────────────────────────────┘
```

## Multi-User Collaboration

### Team Setup

```bash
# Create team research users
prism research-user create alice --full-name "Alice Smith" --email "alice@lab.edu"
prism research-user create bob --full-name "Bob Johnson" --email "bob@lab.edu"
prism research-user create carol --full-name "Carol Davis" --email "carol@lab.edu"

# Create shared EFS volume
prism volumes create team-research --type efs --performance provisioned --throughput 300

# Setup shared directories
prism research-user setup-collaboration --users alice,bob,carol --volume team-research
```

### Collaborative Workflows

#### Scenario 1: Sequential Collaboration

```bash
# Alice starts analysis
alice@python-instance: cd /efs/shared/project-alpha
alice@python-instance: python data_preprocessing.py
alice@python-instance: git add .; git commit -m "Initial data preprocessing"

# Bob continues with statistical analysis
bob@r-instance: cd /efs/shared/project-alpha
bob@r-instance: R -e "source('statistical_analysis.R')"
bob@r-instance: git add .; git commit -m "Statistical analysis complete"

# Carol creates visualizations
carol@viz-instance: cd /efs/shared/project-alpha
carol@viz-instance: python generate_plots.py
carol@viz-instance: git add .; git commit -m "Added data visualizations"
```

#### Scenario 2: Parallel Collaboration

```bash
# Multiple users working simultaneously
alice@gpu-1: cd /efs/shared/datasets && python preprocess_batch_1.py
bob@gpu-2: cd /efs/shared/datasets && python preprocess_batch_2.py
carol@cpu-1: cd /efs/shared/analysis && R -e "source('summary_stats.R')"

# Files automatically have correct ownership
ls -la /efs/shared/datasets/
-rw-r--r-- 1 alice research batch_1_processed.parquet
-rw-r--r-- 1 bob   research batch_2_processed.parquet

ls -la /efs/shared/analysis/
-rw-r--r-- 1 carol research summary_statistics.csv
```

### Access Control and Permissions

```bash
# Create project-specific groups
prism research-user create-group ml-team --members alice,bob
prism research-user create-group viz-team --members alice,carol
prism research-user create-group stats-team --members bob,carol

# Set directory permissions for groups
prism research-user set-permissions /efs/shared/ml-project --group ml-team --mode 775
prism research-user set-permissions /efs/shared/visualizations --group viz-team --mode 775
prism research-user set-permissions /efs/shared/statistics --group stats-team --mode 775
```

## Monitoring and Analytics

### Usage Monitoring

```bash
# Show research user activity summary
prism research-user analytics --profile personal-research

# Show detailed usage for specific user
prism research-user analytics alice --detailed

# Show team usage summary
prism research-user analytics --users alice,bob,carol --timeframe 30d

# Export usage data
prism research-user analytics alice --export csv --output alice-usage.csv
```

#### Analytics Output Example

```
Research User Analytics (Last 30 days)
┌─────────────────┬─────────────────────────────────────────┐
│ Metric          │ alice                                   │
├─────────────────┼─────────────────────────────────────────┤
│ Instances Used  │ 8                                      │
│ Total Login Time│ 47h 32m                                │
│ Files Created   │ 1,247                                  │
│ Data Stored     │ 15.7 GB                               │
│ SSH Connections │ 156                                    │
│ Peak Concurrent │ 3 instances                           │
│ Most Used Template│ Python ML (62% of time)             │
│ Collaboration   │ 3 users (bob, carol, david)          │
└─────────────────┴─────────────────────────────────────────┘
```

### System Health Monitoring

```bash
# Check UID/GID allocation status
prism research-user system-status --uid-allocations

# Check SSH key health
prism research-user system-status --ssh-keys

# Check EFS integration status
prism research-user system-status --efs-integration

# Full system health check
prism research-user system-status --full
```

### Audit and Security Monitoring

```bash
# Show recent research user activities
prism research-user audit-log --recent 24h

# Show specific user's audit trail
prism research-user audit-log alice --timeframe 7d

# Show SSH key usage patterns
prism research-user audit-log --ssh-keys --timeframe 30d

# Export audit logs
prism research-user audit-log --export json --output audit-2025-09.json
```

## Troubleshooting

### Common Issues and Solutions

#### 1. SSH Access Problems

**Problem**: Cannot SSH to instance as research user

```bash
# Diagnosis
prism research-user status alice --instance my-instance
prism research-user ssh-key list alice

# Solutions
# Check if user is provisioned
prism research-user provision alice --instance my-instance

# Verify SSH keys are installed
ssh ubuntu@my-instance "sudo cat /efs/home/alice/.ssh/authorized_keys"

# Re-provision if needed
prism research-user provision alice --instance my-instance --force
```

#### 2. File Permission Issues

**Problem**: Cannot access files created by other research users

```bash
# Diagnosis
ssh alice@instance "ls -la /efs/shared/problematic-file"
ssh alice@instance "groups"

# Solutions
# Check if user is in research group
prism research-user update alice --add-groups research

# Fix file permissions
ssh alice@instance "sudo chgrp research /efs/shared/problematic-file"
ssh alice@instance "sudo chmod g+r /efs/shared/problematic-file"
```

#### 3. EFS Mount Issues

**Problem**: EFS home directory not accessible

```bash
# Diagnosis
ssh alice@instance "mount | grep efs"
ssh alice@instance "ls -la /efs/"

# Solutions
# Check EFS mount status
prism volumes status research-home

# Remount EFS volume
prism volumes mount research-home my-instance --mount-point /efs

# Fix EFS permissions
prism research-user setup-efs-permissions alice --volume research-home
```

#### 4. UID Conflicts

**Problem**: Research user has different UID on different instances

```bash
# Diagnosis
ssh alice@instance1 "id alice"  # Should be 5001
ssh alice@instance2 "id alice"  # Should also be 5001

# Solutions
# Check UID allocation
prism research-user show alice --include-uid

# Re-provision user with correct UID
prism research-user provision alice --instance instance2 --force

# Clear and regenerate UID cache
prism research-user system-maintenance --clear-uid-cache
```

### Diagnostic Commands

```bash
# Comprehensive system diagnostics
prism research-user diagnose

# Diagnose specific user
prism research-user diagnose alice

# Diagnose specific instance
prism research-user diagnose --instance my-instance

# Generate diagnostic report
prism research-user diagnose --report diagnostic-report.txt
```

### Recovery Procedures

#### Recover Lost Research User

```bash
# Recreate research user from backup
prism research-user restore alice --from-backup alice-backup.json

# Recreate with same UID (if known)
prism research-user create alice --force-uid 5001

# Restore SSH keys
prism research-user ssh-key import-backup alice alice-keys-backup.tar.gz
```

#### Reset Research User System

```bash
# Clear all research user data (DESTRUCTIVE)
prism research-user system-reset --confirm

# Reset specific profile
prism research-user system-reset --profile personal-research --confirm

# Reset UID allocations only
prism research-user system-reset --uid-allocations --confirm
```

## Security Best Practices

### SSH Key Security

1. **Use Ed25519 Keys**: Prefer Ed25519 over RSA for new key generation
   ```bash
   prism research-user ssh-key generate alice ed25519
   ```

2. **Regular Key Rotation**: Rotate SSH keys periodically
   ```bash
   # Monthly key rotation
   prism research-user ssh-key rotate alice ed25519 --deactivate-old-after 30d
   ```

3. **Monitor Key Usage**: Track SSH key usage patterns
   ```bash
   prism research-user audit-log alice --ssh-keys --timeframe 7d
   ```

### Access Control

1. **Principle of Least Privilege**: Only grant necessary permissions
   ```bash
   # Remove docker access if not needed
   prism research-user update alice --remove-groups docker
   ```

2. **Regular Access Reviews**: Review user permissions quarterly
   ```bash
   prism research-user list --detailed --include-permissions
   ```

3. **Group-Based Permissions**: Use groups for shared access
   ```bash
   prism research-user create-group project-alpha --members alice,bob
   ```

### Data Security

1. **EFS Encryption**: Use encrypted EFS volumes
   ```bash
   prism volumes create secure-research --type efs --encrypted
   ```

2. **Home Directory Isolation**: Ensure proper home directory permissions
   ```bash
   # Verify home directory permissions are 750
   ssh alice@instance "ls -ld /efs/home/alice"
   ```

3. **Shared Directory Controls**: Implement proper shared directory permissions
   ```bash
   # Shared directories should be 755 or 775
   ssh alice@instance "ls -ld /efs/shared/*"
   ```

### Monitoring and Compliance

1. **Enable Audit Logging**: Track all research user activities
   ```bash
   # Enable comprehensive audit logging
   prism config set research-user.audit-log.enabled true
   prism config set research-user.audit-log.level detailed
   ```

2. **Regular Security Scans**: Check for security issues
   ```bash
   prism research-user security-scan --profile personal-research
   ```

3. **Compliance Reporting**: Generate compliance reports
   ```bash
   prism research-user compliance-report --format pdf --output compliance-2025-Q3.pdf
   ```

## Institutional Deployment

### Large-Scale Setup

#### University Department (100+ Users)

```bash
# Batch user creation from CSV
prism research-user batch-create --from-csv students-cs501.csv

# Template optimization for education
prism research-user configure-education --class cs501 --template "Python ML" --users-from-csv students.csv

# Automated EFS setup for classes
prism research-user setup-class-storage cs501 --volume-size 1TB --shared-quota 100GB-per-user
```

#### Research Institution (500+ Users)

```bash
# Department-based organization
prism research-user create-department computer-science --quota 10TB
prism research-user create-department biology --quota 25TB
prism research-user create-department physics --quota 15TB

# Automated provisioning pipeline
prism research-user setup-auto-provisioning --ldap-integration --department-quotas
```

### Integration with External Systems

#### LDAP/Active Directory Integration

```bash
# Configure LDAP authentication
prism research-user configure-ldap --server ldap.university.edu --base-dn "ou=users,dc=university,dc=edu"

# Sync users from LDAP
prism research-user ldap-sync --department computer-science

# Map LDAP groups to research user groups
prism research-user map-ldap-groups "cn=CS Students,ou=groups,dc=university,dc=edu" students
```

#### Single Sign-On (SSO) Integration

```bash
# Configure SAML SSO
prism research-user configure-sso --provider saml --metadata-url https://sso.university.edu/metadata

# Configure OAuth integration
prism research-user configure-sso --provider oauth --client-id university-cws --discovery-url https://oauth.university.edu/.well-known
```

### Policy Management

#### Institutional Policies

```bash
# Create policy templates for different user types
prism research-user create-policy undergraduate --max-instances 2 --max-storage 10GB --templates "Python ML,R Research"
prism research-user create-policy graduate --max-instances 5 --max-storage 100GB --templates "*"
prism research-user create-policy faculty --max-instances unlimited --max-storage 1TB --templates "*"

# Apply policies to users
prism research-user apply-policy alice undergraduate
prism research-user apply-policy bob graduate
```

#### Compliance and Governance

```bash
# Enable data residency controls
prism research-user configure-compliance --data-residency US --encryption required

# Set retention policies
prism research-user configure-retention --inactive-users 365d --archive-data 7y

# Configure audit requirements
prism research-user configure-audit --level comprehensive --retention 10y --export-format syslog
```

## Advanced Configuration

### Performance Tuning

#### UID Allocation Optimization

```bash
# Configure UID allocation for large deployments
prism config set research-user.uid-base 10000
prism config set research-user.uid-range 50000

# Enable UID allocation caching
prism config set research-user.uid-cache.enabled true
prism config set research-user.uid-cache.ttl 24h
```

#### EFS Performance Optimization

```bash
# Configure EFS performance mode for research workloads
prism config set research-user.efs.performance-mode provisioned
prism config set research-user.efs.throughput-mode provisioned
prism config set research-user.efs.throughput-mib 500
```

#### SSH Key Management Optimization

```bash
# Enable SSH key caching for faster access
prism config set research-user.ssh-key.cache.enabled true
prism config set research-user.ssh-key.cache.ttl 1h

# Configure key rotation policies
prism config set research-user.ssh-key.rotation.enabled true
prism config set research-user.ssh-key.rotation.interval 90d
```

### Custom Integration

#### API Integration

```go
// Example Go code for custom research user integration
package main

import (
    "github.com/scttfrdmn/prism/pkg/research"
    "github.com/scttfrdmn/prism/pkg/profile"
)

func main() {
    // Create research user service
    profileMgr := profile.NewManager()
    service := research.CreateDefaultResearchUserService(profileMgr)

    // Create research user programmatically
    user, err := service.CreateResearchUser("alice", &research.CreateResearchUserOptions{
        GenerateSSHKey: true,
    })
    if err != nil {
        panic(err)
    }

    // Provision on instance
    req := &research.ProvisionInstanceRequest{
        InstanceID:   "i-1234567890abcdef0",
        InstanceName: "ml-workstation",
        PublicIP:     "54.123.45.67",
        Username:     "alice",
        SSHKeyPath:   "/home/admin/.ssh/id_rsa",
        SSHUser:      "ubuntu",
    }

    response, err := service.ProvisionUserOnInstance(context.Background(), req)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Provisioning successful: %v\n", response.Success)
}
```

#### Plugin System (Future)

```bash
# Install research user plugins
prism plugin install research-user-ldap-sync
prism plugin install research-user-quota-manager
prism plugin install research-user-analytics-dashboard

# Configure plugins
prism plugin configure research-user-ldap-sync --server ldap.university.edu
```

### Backup and Disaster Recovery

#### Backup Procedures

```bash
# Backup all research user configurations
prism research-user backup --profile personal-research --output research-users-backup.tar.gz

# Backup SSH keys
prism research-user backup-ssh-keys --profile personal-research --output ssh-keys-backup.tar.gz

# Backup EFS data
prism volumes snapshot research-home --description "Weekly backup $(date +%Y-%m-%d)"
```

#### Recovery Procedures

```bash
# Restore research users from backup
prism research-user restore --from-backup research-users-backup.tar.gz

# Restore SSH keys
prism research-user restore-ssh-keys --from-backup ssh-keys-backup.tar.gz

# Restore EFS from snapshot
prism volumes restore research-home --from-snapshot snap-1234567890abcdef0
```

## Conclusion

Research User Management in Prism v0.5.0 provides a comprehensive foundation for collaborative research computing. This guide covers:

- ✅ **Complete Setup**: From initial configuration to advanced deployment
- ✅ **User Management**: Creation, modification, and lifecycle management
- ✅ **Security**: SSH keys, access control, and compliance
- ✅ **Collaboration**: Multi-user workflows and team management
- ✅ **Monitoring**: Analytics, troubleshooting, and system health
- ✅ **Scale**: From individual researchers to institutional deployments

The research user system transforms Prism from a single-user tool into a collaborative research platform while maintaining the simplicity and flexibility that makes Prism powerful.

For additional support:
- 📚 **Technical Documentation**: [Phase 5A Research User Architecture](PHASE_5A_RESEARCH_USER_ARCHITECTURE.md)
- 👥 **User Guide**: [Research Users User Guide](USER_GUIDE_RESEARCH_USERS.md)
- 🏗️ **Architecture Guide**: [Dual User Architecture](DUAL_USER_ARCHITECTURE.md)
- 🐛 **Support**: [GitHub Issues](https://github.com/scttfrdmn/prism/issues)

---

**Document Version**: 1.0
**Last Updated**: September 28, 2025
**Prism Version**: v0.5.0+