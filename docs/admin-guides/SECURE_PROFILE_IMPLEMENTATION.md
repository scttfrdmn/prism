# Secure Profile Implementation in Prism

This document describes the implementation of the secure profile management and device binding system in Prism v0.4.3.

## Overview

The secure profile system provides robust security for invitation-based profiles through device binding, a central registry of authorized devices, and multi-level permissions. This implementation ensures that profiles cannot be casually shared while maintaining a smooth user experience for legitimate multi-device scenarios.

## Architecture

The secure profile system is implemented across multiple layers of the application:

### 1. Core Security Components

- **Device Binding**: Profiles can be restricted to specific devices using unique device identifiers
- **Keychain Integration**: Secure storage of binding material using platform-native keystores
- **S3 Registry**: Central tracking of authorized devices for each invitation
- **Multi-Level Permissions**: Hierarchical permission model for invitation delegation

### 2. Security Implementation

The security system is implemented in the following packages:

- `pkg/profile/secure_invitation.go`: Enhanced invitation management with security features
- `pkg/profile/security/binding.go`: Device binding creation and validation
- `pkg/profile/security/keychain.go`: Cross-platform secure storage abstraction
- `pkg/profile/security/registry.go`: S3-based registry for tracking authorized devices

## Key Features

### 1. Device Binding

Device binding restricts invitation profiles to specific devices, preventing unauthorized access from other computers:

```go
// Create device binding
binding, err := security.CreateDeviceBinding(profileID, invitationToken)
if err != nil {
    return fmt.Errorf("failed to create device binding: %w", err)
}

// Store binding in keychain
bindingRef, err := security.StoreDeviceBinding(binding, profileName)
if err != nil {
    return fmt.Errorf("failed to store device binding: %w", err)
}

// Set binding reference in profile
profile.BindingRef = bindingRef
```

### 2. Secure Storage

The security system uses platform-native secure storage mechanisms:

- **macOS**: Apple Keychain
- **Windows**: Windows Credential Manager
- **Linux**: Secret Service API
- **Fallback**: Encrypted file storage for unsupported platforms

### 3. Central Registry

The S3-based registry provides a centralized authority for tracking and validating devices:

```go
// Register device with registry
err = registry.RegisterDevice(invitationToken, binding.DeviceID)
if err != nil {
    // Non-fatal error, log but continue
}

// Validate device with registry
valid, err := registry.ValidateDevice(invitationToken, deviceID)
if !valid {
    // Handle unauthorized device
}
```

### 4. Multi-Level Permissions

The invitation system implements a hierarchical permission model:

- `can_invite`: Controls who can create sub-invitations
- `transferable`: Controls whether profiles can be exported or shared
- `device_bound`: Controls whether profiles are restricted to specific devices
- `max_devices`: Controls how many devices a user can register (1-5)

## User Interface Components

### 1. CLI Commands

The CLI provides comprehensive commands for secure invitation management:

```bash
# Create secure invitation
prism profiles invitations create-secure lab-access --type admin --can-invite=true --max-devices=3

# List devices for an invitation
prism profiles invitations devices inv-abc123def456

# Revoke specific device
prism profiles invitations revoke-device inv-abc123def456 device-xyz789

# Revoke all devices for an invitation
prism profiles invitations revoke-all inv-abc123def456
```

### 2. TUI Components

The TUI provides a terminal-based interface for secure profile management:

- Profile list with security indicators
- Detailed profile view showing security attributes
- Device binding validation
- Security status in the navigation sidebar

### 3. GUI Components

The GUI provides a user-friendly interface for secure profile management:

- **Add Invitation Dialog**: Allows creating device-bound profiles with a simple checkbox
- **Profile List**: Visual indicators for secure vs. unsecure profiles
- **Device Management Dialog**: Comprehensive interface for viewing and managing devices
- **Profile Validation**: Testing both profile validity and device binding status
- **Security Monitoring**: Background validation of device bindings

## Implementation Details

### 1. Device Identification

Devices are identified using multiple factors:

- Random unique identifier
- Device name (hostname)
- MAC addresses (when available)
- Username

### 2. Invitation Flow

The secure invitation flow works as follows:

1. Admin creates an invitation with security parameters
2. User accepts invitation and opts for device binding
3. System creates and stores binding in keychain
4. Device registers with central registry
5. Profile is created with binding reference

### 3. Validation Process

The validation process for secure profiles includes:

1. Local keychain validation of binding material
2. Registry check for device authorization
3. API validation using the profile's credentials
4. Background monitoring to detect revocation

### 4. Multi-Device Support

The system supports legitimate multi-device usage through:

1. Explicit max_devices parameter in invitations
2. Registry tracking to ensure limits are enforced
3. Device management UI for users to manage their devices

## Security Considerations

### Strengths

- **Keychain Protection**: Leverages OS security for credential protection
- **Two-Factor by Design**: Requires both profile config and device binding
- **Central Authority**: Registry provides revocation and monitoring capability
- **Hierarchical Control**: Administrators maintain control over delegation chain

### Limitations

- **Client-Side Security**: Cannot fully protect against determined local attackers
- **Platform Variations**: Different security levels across operating systems
- **Recovery Complexity**: Recovery procedures add complexity

### Mitigations

- **Background Validation**: Periodic checks against registry
- **Revocation Capability**: Quick revocation of compromised invitations
- **Clear Security Indicators**: UI clearly shows security status of profiles

## User Experience

The implementation follows Prism's design principles:

- **Default to Success**: Device binding enabled by default for security
- **Progressive Disclosure**: Security details shown only when relevant
- **Transparent Feedback**: Clear indicators of security status
- **Helpful Warnings**: Explanations of security implications
- **Zero Surprises**: Explicit confirmation for security-impacting decisions

## Conclusion

The secure profile management implementation in Prism v0.4.3 provides a robust security model that prevents casual sharing of profiles while maintaining excellent user experience for legitimate users. The system leverages platform-native security features and a centralized registry to provide strong security guarantees without excessive friction.