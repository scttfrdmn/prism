# Prism Multi-Profile Guide

<p align="center">
  <img src="images/prism.png" alt="Prism Logo" width="200">
</p>

Profile management provides context support, secure credential storage, and full API integration.

## What are Profiles?

Profiles let you use Prism with different AWS accounts. It's like having multiple backpacks - each one carries different things for different activities!

With profiles, you can:
- Use your own AWS account (Personal Profile)
- Use someone else's AWS account they shared with you (Invitation Profile)
- Switch between accounts without logging in and out

## When to Use Multiple Profiles

You might need multiple profiles when:

- You have your own AWS account AND your teacher invited you to use their class account
- You're working on personal projects AND team projects that use different accounts
- You want to keep work and learning projects separate

## Types of Profiles

### Personal Profiles
- Connected to your own AWS account
- You pay for anything you create
- You have full control (based on your AWS permissions)

### Invitation Profiles
- Connected to someone else's AWS account
- They pay for what you create
- You can only use what they allow you to use
- Perfect for classes, workshops, and team projects

## How to Use Profiles

### In the GUI Application

1. **See Your Current Profile**
   - Look in the sidebar under "AWS Profile"
   - It shows which profile you're currently using

2. **Switch Profiles**
   - Click the "Switch Profile" button in the sidebar
   - Or go to Settings → Profile Management
   - Choose the profile you want to use

3. **Add a Personal Profile**
   - Go to Settings → Profile Management
   - Click "Add Personal Profile"
   - Give it a name
   - Choose which AWS profile to use (from your computer)
   - Pick a region (or leave empty to use your default)

4. **Add an Invitation**
   - Go to Settings → Profile Management
   - Click "Add Invitation" 
   - Enter the invitation details your teacher or team leader gave you
   - Give it a name that helps you remember what it's for (like "Biology Class")

### In the Terminal (Command Line)

Use these commands to manage your profiles:

```bash
# List all your profiles
prism profiles list

# Switch to a different profile
prism profiles use biology-class

# Add a personal profile
prism profiles add personal my-aws --aws-profile default --region us-west-2

# Add an invitation profile
prism profiles add invitation biology-class --token ABC123 --owner "Professor Smith" --region us-east-1

# See which profile you're using now
prism profiles current
```

## Seeing What You Can Use

Different profiles let you do different things:

- **Personal profiles**: You can use any template you want
- **Invitation profiles**: You can only use templates the owner allows

When using an invitation profile, Prism will automatically show you only the templates you're allowed to use.

## Costs and Billing

- **Personal profiles**: You pay for everything you create
- **Invitation profiles**: The account owner pays

Always check which profile you're using before launching new cloud computers!

## Need Help?

If you're having trouble with profiles:

1. Make sure you entered the invitation information correctly
2. Check that you're using the right AWS region
3. Ask the person who invited you for help if needed

Remember: The profile name shown in the sidebar tells you which account you're currently using.

## Technical Reference for Developers

Prism includes a comprehensive API for multi-profile management that can be used by developers building extensions or integrating with the platform.

### Core Components

- **profile.ManagerEnhanced**: Manages profile operations, switching, and validation
- **profile.ProfileAwareStateManager**: Isolates state between different profiles
- **api.ProfileAwareClient**: Provides API access with profile switching capabilities
- **SecureCredentialProvider**: Platform-specific secure storage for credentials

### Using the Profile API

```go
import (
    "github.com/scttfrdmn/prism/pkg/profile"
    "github.com/scttfrdmn/prism/pkg/api"
)

// Create profile manager
profileManager, err := profile.NewManagerEnhanced(configPath)
if err != nil {
    // Handle error
}

// Create state manager with profile awareness
stateManager := profile.NewProfileAwareStateManager(profileManager)

// Create API client with profile support
client, err := api.NewProfileAwareClient("http://localhost:8080", profileManager, stateManager)
if err != nil {
    // Handle error
}

// Switch profiles
err = client.SwitchProfile("work-research")
if err != nil {
    // Handle error
}

// Use the client with current profile
instances, err := client.ListInstances(ctx)
```

### Context Integration

The profile API integrates with Go's context package:

```go
// Create context with current profile
ctx := context.Background()
ctxWithProfile, err := client.WithProfileContext(ctx)
if err != nil {
    // Handle error
}

// Use context-aware API methods
instances, err := client.ListInstances(ctxWithProfile)
```

### Creating Temporary Clients

Sometimes you need a client for a specific profile without changing the current one:

```go
// Get client for specific profile without switching
tempClient, err := client.WithProfile("collaborator")
if err != nil {
    // Handle error
}

// Use temporary client
instances, err := tempClient.ListInstances(ctx)
```

### Profile Data Structure

```go
type Profile struct {
    Type            ProfileType `json:"type"`
    Name            string      `json:"name"`
    AWSProfile      string      `json:"aws_profile,omitempty"`
    Region          string      `json:"region"`
    InvitationToken string      `json:"invitation_token,omitempty"`
    OwnerAccount    string      `json:"owner_account,omitempty"`
    S3ConfigPath    string      `json:"s3_config_path,omitempty"`
    LastUsed        time.Time   `json:"last_used"`
}
```

### Performance Considerations

- Profile switching is designed to be lightweight (<1ms operation)
- Credential loading is lazy (only happens when needed)
- State files are isolated to prevent cross-profile contamination
- API clients maintain connection pools per profile

See the API documentation for more details on using profiles in your code.
---


Prism's profile export/import lets you:

1. Back up your Prism profiles
2. Share profile configurations between machines
3. Transfer profiles to team members

Exports are ZIP (default) or JSON, optionally password-encrypted, and can include
AWS credentials or be limited to specific profiles. This document explains how to
use these features effectively.

## Profile Export

You can export your profiles to a file using the command line interface:

```bash
prism profiles export my-profiles.zip
```

This creates a ZIP file containing your profile configurations.

### Export Options

Several options are available for customizing your exports:

| Option | Description |
|--------|-------------|
| `--include-credentials` | Include AWS credentials (use with caution) |
| `--include-invitations` | Include invitation profiles (default: true) |
| `--profiles profile1,profile2` | Export only specific profiles |
| `--format zip|json` | Export format (default: zip) |
| `--password password` | Password protect the export (zip format only) |

### Example Uses

**Export all profiles without credentials (safest option):**
```bash
prism profiles export my-profiles.zip
```

**Export specific profiles:**
```bash
prism profiles export personal-profiles.zip --profiles personal,work
```

**Export with credentials (only for personal backups):**
```bash
prism profiles export full-backup.zip --include-credentials
```

**Export in JSON format:**
```bash
prism profiles export profiles.json --format json
```

## Profile Import

You can import profiles from a previously exported file:

```bash
prism profiles import my-profiles.zip
```

### Import Options

Several options control how imports are handled:

| Option | Description |
|--------|-------------|
| `--mode skip\|overwrite\|rename` | How to handle conflicts (default: rename) |
| `--profiles profile1,profile2` | Import only specific profiles |
| `--import-credentials` | Import credentials if available |
| `--password password` | Password for encrypted imports |

### Handling Profile Conflicts

When importing profiles, conflicts can occur if profiles with the same ID already exist. Three resolution modes are available:

1. **rename** (default): Rename imported profiles to avoid conflicts
2. **skip**: Skip importing profiles that would conflict
3. **overwrite**: Replace existing profiles with imported ones

### Example Uses

**Import all profiles, renaming any conflicts:**
```bash
prism profiles import my-profiles.zip
```

**Import only specific profiles:**
```bash
prism profiles import team-profiles.zip --profiles team-project,shared
```

**Import and skip any profiles that already exist:**
```bash
prism profiles import my-profiles.zip --mode skip
```

**Import with credentials:**
```bash
prism profiles import my-profiles.zip --import-credentials
```

## Security Considerations

### Credential Handling

By default, credentials are **not** included in exports for security reasons. This prevents accidental sharing of AWS access keys.

**For personal backups only**, you can include credentials with the `--include-credentials` flag. However, this should be used with caution:

- Always store export files with credentials securely
- Consider using password protection (`--password`)
- Never share exports containing credentials with others

### Password Protection

For sensitive exports, particularly those including credentials, you can add password protection:

```bash
prism profiles export secure-backup.zip --include-credentials --password "my-secure-password"
```

When importing, provide the same password:

```bash
prism profiles import secure-backup.zip --password "my-secure-password"
```

## Sharing with Teams

The export/import functionality is particularly useful for teams who need to share common Prism configurations.

**Best practice for sharing with teams:**

1. Create profiles without credentials
2. Export without credentials
3. Share the export file with team members
4. Team members import the profiles
5. Each team member configures their own credentials

```bash
# Team lead:
prism profiles export team-profiles.zip --profiles team-project,shared

# Team members:
prism profiles import team-profiles.zip
```

## Working with Invitation Profiles

When exporting invitation profiles:

- The invitation token is exported
- The invitation's expiration status is preserved
- The recipient's ability to use the invitation depends on whether the invitation is still valid

To exclude invitation profiles from export:

```bash
prism profiles export personal-only.zip --include-invitations=false
```

## File Formats

### ZIP Format

The default export format is ZIP, which includes:

- `profiles.json` - Profile configurations
- `credentials/` directory (if credentials included)
- Metadata files

### JSON Format

For simpler integration with other tools, you can export in plain JSON format:

```bash
prism profiles export profiles.json --format json
```

Note that JSON exports cannot include credentials.

## Troubleshooting

### Common Issues

1. **Import fails with "invalid profiles format"**: The import file may be corrupted or created with an incompatible version.

2. **Credentials not imported**: Credentials are only imported if:
   - They were included in the export (`--include-credentials`)
   - The import was run with `--import-credentials`

3. **Profiles missing after import**: Check if:
   - The profiles were filtered out during export/import
   - There were naming conflicts and the chosen mode (`--mode`) skipped the profiles

### Getting Help

For additional assistance with profile export/import:

```bash
prism profiles export --help
prism profiles import --help
```

## Version Compatibility

Profile export/import is available in Prism v0.4.2 and later. Exports created with newer versions may not be compatible with older versions of Prism.