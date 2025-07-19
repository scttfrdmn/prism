# CloudWorkstation 0.3.0 Upgrade Guide

This document provides instructions for upgrading from CloudWorkstation 0.2.x to CloudWorkstation 0.3.0. Please read the entire guide before beginning the upgrade process.

## Before You Begin

1. **Backup Your State**: CloudWorkstation 0.3.0 includes significant architectural changes. Although the upgrade process should preserve your state, it's recommended to back up your state file:
   ```bash
   cp ~/.cloudworkstation/state.json ~/.cloudworkstation/state.json.backup
   ```

2. **Check Current Instances**: List your current instances before upgrading to verify all instances are preserved after the upgrade:
   ```bash
   cws list
   ```

3. **Note Current Templates**: Make note of any custom templates you've created, as the template format has been updated in 0.3.0.

## Upgrade Process

### Step 1: Stop Running Daemon (if applicable)

If you're using an earlier version of the daemon:

```bash
cws daemon stop
```

### Step 2: Install CloudWorkstation 0.3.0

For Go installation:
```bash
go install github.com/scttfrdmn/cloudworkstation/cmd/cws@v0.3.0
go install github.com/scttfrdmn/cloudworkstation/cmd/cwsd@v0.3.0
```

For binary installation:
```bash
# Download the latest binary for your platform
curl -L https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.3.0/cloudworkstation_0.3.0_$(uname -s)_$(uname -m).tar.gz | tar xz
# Move the binaries to your PATH
sudo mv cws cwsd /usr/local/bin/
```

### Step 3: Start the Daemon

Start the CloudWorkstation daemon:

```bash
cws daemon start
```

Verify the daemon is running:

```bash
cws daemon status
```

You should see output indicating the daemon is running with version 0.3.0.

### Step 4: Verify Installation

Verify that the upgrade preserved your state:

```bash
cws list
```

This command should display all instances that existed before the upgrade.

### Step 5: Configure Repositories

CloudWorkstation 0.3.0 introduces the concept of template repositories. Configure the default repository:

```bash
cws repository add default https://github.com/scttfrdmn/cloudworkstation-repository
cws repository update default
```

## New Configuration Options

### Idle Detection

Configure the idle detection system (optional):

```bash
# Enable idle detection
cws idle enable

# Configure idle detection profile
cws idle profile set standard --cpu=10 --memory=15 --idle-minutes=30 --action=stop
```

### Domain-Specific Profiles

Set up domain-specific idle profiles for different research domains:

```bash
# Create a high-performance computing profile
cws idle profile add hpc --cpu=30 --memory=40 --idle-minutes=60 --action=stop

# Map it to a domain
cws idle domain set neuroimaging hpc
```

### Repository Management

Work with multiple template repositories:

```bash
# Add a custom repository
cws repository add myrepo https://github.com/username/my-template-repo

# Set repository priority
cws repository set myrepo --priority 10

# List available repositories
cws repository list
```

## Breaking Changes

### API Context Support

If you've developed custom applications using the CloudWorkstation API, you'll need to update your code to use the context-aware methods. The legacy methods are still available but are deprecated and will be removed in a future release.

Example:
```go
// Before (0.2.x)
client.LaunchInstance(templateName, instanceName, options)

// After (0.3.0)
ctx := context.Background()
client.LaunchInstance(ctx, templateName, instanceName, options)
```

### Template Format

The template format has been updated to include validation steps and additional metadata. If you have custom templates, you'll need to update them to the new format. See the [Template Authoring Guide](https://docs.cloudworkstation.dev/templates/authoring) for details.

## Troubleshooting

### Daemon Fails to Start

If the daemon fails to start, check the logs:

```bash
cat ~/.cloudworkstation/logs/daemon.log
```

Common issues include:
- Port conflicts (the daemon now uses port 8080 by default)
- Permission issues with the state file
- Incompatible state file format

### State Migration Failures

If your state doesn't migrate correctly:

1. Stop the daemon: `cws daemon stop`
2. Restore your backup: `cp ~/.cloudworkstation/state.json.backup ~/.cloudworkstation/state.json`
3. Start the daemon with the debug flag: `CWS_DEBUG=1 cws daemon start`
4. Contact support with the output from the logs

### Template Repository Issues

If you have issues with template repositories:

```bash
cws repository reset
cws repository add default https://github.com/scttfrdmn/cloudworkstation-repository
```

## Getting Help

If you encounter issues during the upgrade process:

- Check the [Troubleshooting Guide](https://docs.cloudworkstation.dev/troubleshooting)
- Visit the [GitHub Issues](https://github.com/scttfrdmn/cloudworkstation/issues)
- Contact support at support@cloudworkstation.dev