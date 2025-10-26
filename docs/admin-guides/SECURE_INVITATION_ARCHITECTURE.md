# Prism Secure Invitation Architecture

This document outlines the architecture for the enhanced secure invitation system in Prism v0.4.3. The system provides a multi-level permissions model with device binding capabilities, similar to modern passkey systems.

## Goals

1. **Prevent Casual Sharing**: Block unauthorized redistribution of invitation profiles
2. **Support Multi-Device**: Allow legitimate users to use profiles on their multiple devices
3. **Hierarchical Permissions**: Enable delegation of invitation authority with constraints
4. **Low Friction**: Maintain excellent user experience for legitimate users
5. **Administrator Visibility**: Provide tools to track and manage invitation usage

## Security Model

### Enhanced Permission System

Prism implements a hierarchical permission model for invitation-based profiles:

| Permission | Description |
|------------|-------------|
| `can_invite` | Whether this profile can create sub-invitations |
| `transferable` | Whether this profile can be exported/shared |
| `device_bound` | Whether the profile is restricted to specific devices |
| `max_devices` | Maximum number of devices allowed per user (1-5) |

These permissions follow a hierarchical model:
- A user cannot grant permissions they don't have
- Sub-invitations inherit restrictions from their parent
- Administrators can see the full invitation chain

### Keychain-Based Security

For device binding, Prism uses the system's native secure storage:

- **macOS**: Apple Keychain
- **Windows**: Windows Credential Manager
- **Linux**: Secret Service API (GNOME Keyring/KWallet)

This approach provides:
1. Secure storage of binding material
2. Native multi-device sync (on platforms that support it)
3. Protection against casual profile sharing

### S3-Based Registry

A lightweight registry in S3 tracks invitation usage:

```
s3://prism-invitations/
  ├── registry/
  │   └── registry.json       # Master registry of all invitations
  ├── invitations/
  │   ├── inv-abc123/
  │   │   ├── metadata.json   # Invitation details
  │   │   └── devices.json    # Authorized devices
  │   └── inv-def456/
  │       ├── metadata.json
  │       └── devices.json
  └── audit/
      └── access-log.ndjson   # Activity logs
```

The registry enables:
- Validation of authorized devices
- Usage tracking for administrators
- Revocation of compromised invitations
- Audit logging of invitation activities

## User Flows

### Creating Invitations with Security Settings

```bash
# Professor creates TA invitation with invitation abilities
prism profiles invitations create ta-access --type admin \
  --can-invite=true --transferable=false --device-bound=true --max-devices=3

# TA creates student invitation with restricted permissions
prism profiles invitations create student-access --type read_only \
  --can-invite=false --transferable=false --device-bound=true --max-devices=2
```

### Accepting an Invitation

When a user accepts an invitation, the system:

1. Validates the invitation token
2. Creates a device binding in the system keychain
3. Registers the device in the S3 registry
4. Creates a profile with a reference to the keychain item

For seamless multi-device use:
- On Apple platforms, iCloud Keychain can automatically sync the binding
- On other platforms, an enrollment code is generated for additional devices

### Using Multiple Devices

The system supports multiple approaches for multi-device usage:

1. **Native Keychain Sync**: For platforms with built-in keychain sync (Apple)
2. **Enrollment Flow**: For other platforms
   ```bash
   # On secondary device
   prism profiles enroll ENROLLMENT_CODE
   ```
3. **Device Management**:
   ```bash
   # List authorized devices
   prism profiles devices list
   
   # Remove a device
   prism profiles devices remove DEVICE_ID
   ```

## Technical Implementation

### Enhanced Data Models

```go
// Enhanced InvitationToken
type InvitationToken struct {
    // Basic invitation data
    Token        string          `json:"token"`
    OwnerProfile string          `json:"owner_profile"`
    Name         string          `json:"name"`
    Type         InvitationType  `json:"type"`
    Created      time.Time       `json:"created"`
    Expires      time.Time       `json:"expires"`
    
    // Security attributes
    CanInvite    bool            `json:"can_invite"`
    Transferable bool            `json:"transferable"`
    DeviceBound  bool            `json:"device_bound"`
    MaxDevices   int             `json:"max_devices"`
    
    // Parentage tracking
    ParentToken  string          `json:"parent_token,omitempty"`
}

// Enhanced Profile
type Profile struct {
    // Basic profile data
    Type            string      `json:"type"`
    Name            string      `json:"name"`
    AWSProfile      string      `json:"aws_profile"`
    
    // Security attributes
    CanInvite       bool        `json:"can_invite"`
    Transferable    bool        `json:"transferable"`
    DeviceBound     bool        `json:"device_bound"`
    BindingRef      string      `json:"binding_ref,omitempty"`
}
```

### Keychain Integration

Prism uses a cross-platform abstraction for secure storage:

```go
// Cross-platform keychain interface
type KeychainProvider interface {
    Store(key string, data []byte) error
    Retrieve(key string) ([]byte, error)
    Exists(key string) bool
    Delete(key string) error
}

// Platform implementations
func NewKeychainProvider() KeychainProvider {
    switch runtime.GOOS {
    case "darwin":
        return &MacOSKeychain{}
    case "windows":
        return &WindowsCredentialManager{}
    default:
        return &LinuxSecretService{}
    }
}
```

### Device Binding Process

1. **Create Binding**: When accepting an invitation
   ```go
   func createDeviceBinding(profile *Profile, invitation *InvitationToken) error {
       // Generate device identifier
       deviceID := generateDeviceID()
       
       // Create binding material
       binding := BindingMaterial{
           DeviceID:    deviceID,
           ProfileID:   profile.AWSProfile,
           InvitationToken: invitation.Token,
           Created:     time.Now(),
       }
       
       // Store in keychain
       bindingData, _ := json.Marshal(binding)
       bindingRef := fmt.Sprintf("com.prism.profile.%s", profile.AWSProfile)
       
       keychain := NewKeychainProvider()
       if err := keychain.Store(bindingRef, bindingData); err != nil {
           return fmt.Errorf("failed to create binding: %w", err)
       }
       
       // Save reference in profile
       profile.BindingRef = bindingRef
       
       // Register with S3 registry (background)
       go registerWithS3Registry(invitation.Token, deviceID)
       
       return nil
   }
   ```

2. **Validate Binding**: When using a profile
   ```go
   func validateBinding(profile *Profile) error {
       if !profile.DeviceBound {
           return nil // No validation needed
       }
       
       keychain := NewKeychainProvider()
       bindingData, err := keychain.Retrieve(profile.BindingRef)
       
       if err != nil || bindingData == nil {
           return errors.New("profile not authorized for this device")
       }
       
       // Binding exists, allow usage
       return nil
   }
   ```

## Administrator Tools

Administrators can monitor and manage invitation usage:

```bash
# View usage statistics
prism profiles invitations usage INVITATION_TOKEN

# Revoke specific device
prism profiles invitations revoke-device INVITATION_TOKEN DEVICE_ID

# Revoke entire invitation
prism profiles invitations revoke INVITATION_TOKEN
```

The GUI will provide visual dashboards showing:
- Active invitations and their status
- User and device counts
- Usage patterns and anomalies

## Security Considerations

### Security Strengths

1. **Keychain Protection**: Leverages OS security for credential protection
2. **Multi-Factor by Design**: Requires both profile config and keychain binding
3. **Hierarchical Control**: Administrators maintain control over delegation chain
4. **Visibility**: All invitation usage is trackable

### Security Limitations

1. **Client-Side Security**: Can't fully protect against determined local attackers
2. **Platform Variations**: Different security levels across operating systems
3. **Recovery Complexity**: Recovery procedures add complexity

### Mitigations

1. **Background Validation**: Periodic checks against S3 registry
2. **Usage Analytics**: Detecting unusual patterns
3. **Revocation Capability**: Quick revocation of compromised invitations
4. **Audit Logging**: All security events are logged

## Implementation Timeline

The secure invitation system will be implemented in phases:

1. **Phase 1 (v0.4.3)**: Enhanced data models and keychain integration
2. **Phase 2 (v0.4.4)**: S3 registry and basic validation
3. **Phase 3 (v0.4.5)**: Administrator tools and advanced monitoring
4. **Phase 4 (v0.5.0)**: GUI integration and analytics dashboard

## Integration with Existing Features

This feature enhances and integrates with:

1. **Profile Management**: Adds security attributes to profiles
2. **Invitation System**: Extends the invitation model with security constraints
3. **Export/Import**: Enforces transferability restrictions
4. **CLI/GUI**: Adds security-related commands and interfaces

## Conclusion

The secure invitation architecture provides a robust solution for controlling profile sharing while maintaining an excellent user experience. By leveraging OS-native security features and a lightweight server component, it achieves the security goals without adding significant friction for legitimate users.

This approach is particularly well-suited for educational and research environments where preventing casual sharing is important, but extremely high security against sophisticated attacks is not required.