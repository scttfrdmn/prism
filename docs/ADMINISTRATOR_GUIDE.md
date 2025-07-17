# CloudWorkstation Administrator Guide

This guide provides information for administrators on managing CloudWorkstation profiles and invitations with a focus on security features.

## Profile Security System

CloudWorkstation v0.4.3 introduces a comprehensive security model for invitation-based profiles, allowing administrators to control access to their AWS resources with fine-grained permissions.

### Invitation Security Features

Secure invitations support several key security attributes:

| Security Attribute | Description |
|-------------------|-------------|
| `device_bound`    | Restricts profiles to specific devices |
| `can_invite`      | Controls who can create sub-invitations |
| `transferable`    | Controls whether profiles can be exported |
| `max_devices`     | Limits how many devices a user can register (1-5) |

### Creating Secure Invitations

Administrators can create secure invitations with specific security constraints:

```bash
# Create a secure invitation with device binding and other security features
cws profiles invitations create-secure lab-access \
  --type admin \
  --can-invite=true \
  --transferable=false \
  --device-bound=true \
  --max-devices=3
```

### Managing Devices

Administrators can view and manage the devices registered to use invitations:

```bash
# List all devices for an invitation
cws profiles invitations devices inv-abc123def456

# Revoke a specific device
cws profiles invitations revoke-device inv-abc123def456 device-xyz789

# Revoke all devices for an invitation
cws profiles invitations revoke-all inv-abc123def456
```

### Hierarchical Permissions

CloudWorkstation implements a hierarchical permission model for delegation:

1. **Permission Inheritance**: Sub-invitations cannot have more permissions than their parent invitation
2. **Delegation Control**: Only users with `can_invite=true` can create sub-invitations
3. **Security Constraints**: Parent security settings are enforced on all child invitations

For example, if a parent invitation has `device_bound=true` and `max_devices=3`, then all sub-invitations will also have `device_bound=true` and cannot exceed 3 devices per user.

## Advanced Administration

### Registry Management

The S3-based registry tracks all devices authorized to use invitations. For administrative purposes, you can use the device manager tool:

```bash
# View all registered devices for an invitation
go run scripts/device-manager.go list --token inv-abc123def456

# Output JSON for integration with other tools
go run scripts/device-manager.go list --token inv-abc123def456 --format json

# Revoke a specific device
go run scripts/device-manager.go revoke --token inv-abc123def456 --device device-xyz789

# Revoke all devices (useful for emergency response)
go run scripts/device-manager.go revoke-all --token inv-abc123def456 --force
```

### Registry Configuration

The registry can be configured using environment variables:

```bash
# Set registry bucket name (defaults to cloudworkstation-invitations)
export CWS_REGISTRY_BUCKET=my-organization-invitations

# Set registry region (defaults to us-west-2)
export CWS_REGISTRY_REGION=us-east-1

# Set registry API endpoint for custom deployments
export CWS_REGISTRY_API=https://registry.example.com/api
```

### Security Monitoring

Administrators can monitor invitation usage and device registrations:

1. **CloudWatch Metrics**: Enable CloudWatch metrics for the registry S3 bucket
2. **Access Logs**: Enable S3 access logging for audit purposes
3. **Registry API**: Use the registry API for programmatic access to device data

### Security Scenarios

#### Scenario: Security Breach

If you suspect a security breach:

1. **Revoke all devices** for the affected invitation:
   ```bash
   cws profiles invitations revoke-all inv-abc123def456
   ```

2. **Create new invitation** with stricter security:
   ```bash
   cws profiles invitations create-secure new-access --device-bound=true --max-devices=1
   ```

3. **Notify legitimate users** to register with the new invitation

#### Scenario: User Leaves Organization

When a user leaves your organization:

1. **Identify the user's devices**:
   ```bash
   cws profiles invitations devices inv-abc123def456
   ```

2. **Revoke their specific devices**:
   ```bash
   cws profiles invitations revoke-device inv-abc123def456 device-xyz789
   ```

## Security Best Practices

1. **Always enable device binding** for non-public AWS accounts
2. **Limit max devices** to the minimum needed (typically 1-2 devices per user)
3. **Restrict invitation delegation** by setting `can_invite=false` for most users
4. **Disable transferability** (`transferable=false`) for all security-sensitive accounts
5. **Use appropriate invitation types**:
   - `read_only` for most users
   - `read_write` for trusted contributors
   - `admin` only for administrators
6. **Regularly audit device registrations** using the device manager tool
7. **Revoke unused devices** to maintain tight security controls

## Troubleshooting

### Common Issues

#### Issue: "Device binding failed" error when accepting invitation

This typically occurs when:
- The device already has the maximum allowed bindings
- There's an issue with keychain access
- The registry cannot be reached for verification

Resolution:
1. Check that the user hasn't exceeded their device limit
2. Verify keychain permissions on the user's system
3. Check connectivity to the registry API

#### Issue: "Device binding revoked" messages

This occurs when a device's authorization has been revoked by an administrator.

Resolution:
1. Contact the invitation administrator for a new invitation
2. Register a new device with the new invitation

#### Issue: "Maximum devices reached" error

This occurs when a user tries to register more devices than allowed.

Resolution:
1. Use the device manager to list current devices:
   ```bash
   cws profiles invitations devices inv-abc123def456
   ```
2. Revoke unused devices:
   ```bash
   cws profiles invitations revoke-device inv-abc123def456 device-old
   ```
3. Try registering the new device again

## Technical References

- [Secure Profile Implementation](SECURE_PROFILE_IMPLEMENTATION.md): Detailed technical documentation
- [Profile Export/Import Guide](PROFILE_EXPORT_IMPORT.md): Information on secure profile migration
- [Secure Invitation Architecture](SECURE_INVITATION_ARCHITECTURE.md): Design documentation