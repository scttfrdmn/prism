# CloudWorkstation Multi-Profile Guide

<p align="center">
  <img src="images/cloudworkstation.png" alt="CloudWorkstation Logo" width="200">
</p>

> **Current v0.4.3**: Enhanced profile management with context support, secure credential storage, and full API integration.

## What are Profiles?

Profiles let you use CloudWorkstation with different AWS accounts. It's like having multiple backpacks - each one carries different things for different activities!

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
cws profiles list

# Switch to a different profile
cws profiles use biology-class

# Add a personal profile
cws profiles add personal my-aws --aws-profile default --region us-west-2

# Add an invitation profile
cws profiles add invitation biology-class --token ABC123 --owner "Professor Smith" --region us-east-1

# See which profile you're using now
cws profiles current
```

## Seeing What You Can Use

Different profiles let you do different things:

- **Personal profiles**: You can use any template you want
- **Invitation profiles**: You can only use templates the owner allows

When using an invitation profile, CloudWorkstation will automatically show you only the templates you're allowed to use.

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

CloudWorkstation v0.4.3 includes a comprehensive API for multi-profile management that can be used by developers building extensions or integrating with the platform.

### Core Components

- **profile.ManagerEnhanced**: Manages profile operations, switching, and validation
- **profile.ProfileAwareStateManager**: Isolates state between different profiles
- **api.ProfileAwareClient**: Provides API access with profile switching capabilities
- **SecureCredentialProvider**: Platform-specific secure storage for credentials

### Using the Profile API

```go
import (
    "github.com/scttfrdmn/cloudworkstation/pkg/profile"
    "github.com/scttfrdmn/cloudworkstation/pkg/api"
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