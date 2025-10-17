# SSH Key Management Architecture (v0.5.3+)

## Problem Statement

The current SSH key management has several issues that need normalization:

1. **Naming inconsistency**: Keys use profile-based names but aren't predictable
2. **Location fragmentation**: Keys mixed with user's personal SSH keys in `~/.ssh/`
3. **Discovery complexity**: Complex fallback logic to find keys
4. **No user access**: Users can't easily extract keys they need
5. **Weak association**: Key-to-instance mapping relies on AWS KeyName string matching

## Normalized Architecture

### Directory Structure

```
~/.cloudworkstation/
├── keys/                          # Isolated SSH key storage
│   ├── default                    # Default key (RSA 2048)
│   ├── default.pub
│   ├── <profile-name>            # Per-profile keys (optional)
│   ├── <profile-name>.pub
│   └── metadata.json             # Key metadata and mappings
├── state.json                     # Instance state with key associations
└── config.json                    # Global configuration
```

### Naming Convention

| Component | Format | Example |
|-----------|--------|---------|
| **AWS KeyName** | `cws-<profile>-<region>` | `cws-research-us-west-2` |
| **Local Private Key** | `~/.cloudworkstation/keys/<profile>` | `~/.cloudworkstation/keys/research` |
| **Local Public Key** | `~/.cloudworkstation/keys/<profile>.pub` | `~/.cloudworkstation/keys/research.pub` |

**Normalization Rules:**
- Profile name sanitized: lowercase, spaces/underscores → hyphens
- Region included in AWS KeyName for compliance/isolation
- No `.pem` or `-key` suffix (cleaner names)

### Key Metadata Structure

```json
{
  "keys": {
    "research": {
      "aws_key_name": "cws-research-us-west-2",
      "profile": "research",
      "region": "us-west-2",
      "created_at": "2025-10-17T12:00:00Z",
      "type": "rsa-2048",
      "instances": ["my-workstation", "gpu-training"]
    },
    "default": {
      "aws_key_name": "cws-default-us-east-1",
      "profile": "default",
      "region": "us-east-1",
      "created_at": "2025-10-17T11:00:00Z",
      "type": "rsa-2048",
      "instances": ["quick-test"]
    }
  },
  "version": "1.0"
}
```

## Implementation Components

### 1. Unified SSH Key Manager (`pkg/profile/ssh_keys_v2.go`)

```go
type SSHKeyManagerV2 struct {
    keysDir      string // ~/.cloudworkstation/keys
    metadataPath string // ~/.cloudworkstation/keys/metadata.json
}

// Core Operations
- GetOrCreateKeyForProfile(profile, region) (localPath, awsKeyName, error)
- GetKeyPath(profile) (string, error)
- ListKeys() ([]KeyMetadata, error)
- ExportKeyToLocation(profile, destPath) error
- DeleteKey(profile) error
```

### 2. Connection Info Enhancement

**Current:**
```go
func (m *Manager) GetConnectionInfo(name string) (string, error) {
    // Complex KeyName → filesystem mapping
    keyPath, err := m.getSSHKeyPathFromKeyName(*instance.KeyName)
    return fmt.Sprintf("ssh -i \"%s\" ubuntu@%s", keyPath, ip), nil
}
```

**Normalized:**
```go
func (m *Manager) GetConnectionInfo(name string) (string, error) {
    // Direct lookup: KeyName = cws-<profile>-<region>
    // Local path  = ~/.cloudworkstation/keys/<profile>
    keyPath := m.keyManager.GetKeyPathFromAWSKeyName(keyName)
    username := m.getUsernameForInstance(name) // From state metadata
    return fmt.Sprintf("ssh -i \"%s\" %s@%s", keyPath, username, ip), nil
}
```

### 3. CLI Commands

```bash
# List all CloudWorkstation SSH keys
cws keys list
# Output:
# KEY           PROFILE    REGION      INSTANCES  CREATED
# research      research   us-west-2   5          2025-10-17
# default       default    us-east-1   2          2025-10-15

# Show specific key details
cws keys show research
# Output:
# Key: research
# AWS KeyName: cws-research-us-west-2
# Local Path: /Users/username/.cloudworkstation/keys/research
# Public Key: ssh-rsa AAAA...xyz cloudworkstation
# Associated Instances:
#   - my-workstation (running)
#   - gpu-training (stopped)

# Export private key
cws keys export research --output ~/my-backup-keys/research.pem
# Output:
# ✅ Key exported to ~/my-backup-keys/research.pem
# ⚠️  Keep this file secure - it provides access to your instances

# Import existing key (for team sharing)
cws keys import shared-research --key-file ~/shared-research.pem

# Show public key (for adding to other systems)
cws keys public research
# Output: ssh-rsa AAAA...xyz cloudworkstation

# Delete unused key
cws keys delete old-project
# Output: ⚠️  Key 'old-project' is used by 3 instances. Delete anyway? [y/N]
```

## Migration Strategy

### Phase 1: Add New System (v0.5.3)
- Implement `SSHKeyManagerV2` alongside existing manager
- New launches use new system
- Old instances continue with existing keys

### Phase 2: Migration Tool (v0.5.4)
```bash
cws keys migrate
# Discovers keys in ~/.ssh/cws-*
# Moves to ~/.cloudworkstation/keys/
# Updates metadata.json
# Validates all instances still accessible
```

### Phase 3: Deprecation (v0.6.0)
- Remove `SSHKeyManager` (old system)
- All instances use normalized system

## Security Considerations

1. **Permissions**: Keys directory `0700`, key files `0600`
2. **Backup**: User-accessible export for disaster recovery
3. **Rotation**: Future support for key rotation without instance recreation
4. **Regional isolation**: Keys can be region-specific for compliance
5. **Audit**: metadata.json tracks key creation and usage

## Benefits

### For Users
- **Invisible**: Keys automatically managed, no manual setup
- **Accessible**: `cws keys export` for backup or team sharing
- **Clean**: CWS keys isolated from personal SSH keys
- **Portable**: Easy to backup/restore `.cloudworkstation/` directory

### For Developers
- **Predictable**: Deterministic key discovery
- **Simple**: No complex fallback logic
- **Maintainable**: Single source of truth (metadata.json)
- **Testable**: Isolated directory structure

### For Operations
- **Auditable**: All key operations logged in metadata
- **Regional**: Compliance with data residency requirements
- **Secure**: Proper permissions enforced at all times

## Testing Strategy

```bash
# Test key creation
cws launch python-ml test-key-creation
# Verify: ~/.cloudworkstation/keys/<profile> exists
# Verify: metadata.json updated
# Verify: SSH connection works

# Test key reuse
cws launch r-research test-key-reuse
# Verify: Same key used if same profile
# Verify: metadata.json instances array updated

# Test key export
cws keys export research --output /tmp/test.pem
ssh -i /tmp/test.pem ubuntu@<instance-ip>
# Verify: Exported key works

# Test key migration
cws keys migrate
# Verify: Old keys moved
# Verify: Old instances still connectable
# Verify: metadata.json correctly populated
```

## Rollout Plan

1. **v0.5.3**: Implement new system, opt-in with flag `--use-new-keys`
2. **v0.5.4**: Default to new system, add migration tool
3. **v0.5.5**: Deprecate old system, add warnings
4. **v0.6.0**: Remove old system completely

## Future Enhancements

1. **Key rotation**: `cws keys rotate <profile>` regenerates key and updates instances
2. **Multiple keys per profile**: `cws keys create research-gpu` for specialized workloads
3. **Team key sharing**: `cws keys share research --with team@company.com`
4. **Hardware keys**: Support for YubiKey/hardware security modules
5. **Per-instance keys**: `cws launch python-ml test --dedicated-key` for security isolation
