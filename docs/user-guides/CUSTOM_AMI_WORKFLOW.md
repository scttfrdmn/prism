# Custom AMI Workflow Guide

Complete guide to creating and managing custom Amazon Machine Images (AMIs) with Prism for fast instance provisioning.

## Table of Contents

- [Overview](#overview)
- [Why Use Custom AMIs?](#why-use-custom-amis)
- [Quick Start](#quick-start)
- [Complete Workflow](#complete-workflow)
- [Command Reference](#command-reference)
- [Use Cases](#use-cases)
- [Performance Comparison](#performance-comparison)
- [Troubleshooting](#troubleshooting)

## Overview

Custom AMIs are pre-configured instance snapshots that enable:
- **Fast Launching**: 2-3 minutes vs 5-20 minutes for templates
- **Consistent Environments**: Everyone uses identical configuration
- **Rapid Iteration**: Quick launch → test → terminate cycles
- **Workshop Ready**: Provision dozens of identical instances quickly

Perfect for research teams, teaching, workshops, and production deployments.

## Why Use Custom AMIs?

### The Problem
Templates install packages every time an instance launches:
```bash
# Every template launch repeats the same slow steps:
# 1. Boot base OS (1 min)
# 2. Install R/Python (30 sec)
# 3. Install packages (2-15 min) ← SLOW!
# Total: 5-20 minutes per instance
```

### The Solution
Create an AMI once, launch quickly:
```bash
# One-time setup (configure your instance, then save it):
prism ami save my-configured-env "My Research Environment"
# Takes 5-10 minutes (one time)

# Every subsequent launch:
prism workspace launch --ami "My Research Environment" quick-instance
# Takes 2-3 minutes ← FAST!
```

There are two ways to create AMIs in Prism:

1. **`prism ami save`** — Saves a running or stopped workspace as an AMI. Use this after you've customized an instance.
2. **`prism ami create`** — Builds an AMI from a template definition (pre-baking). Use this for clean template-based AMIs.

## Quick Start

### 5-Minute AMI Creation

```bash
# 1. Launch a base template
prism workspace launch r-rstudio-server my-r-env

# 2. Connect and customize
prism workspace connect my-r-env
# Install your packages, configure settings, etc.
sudo -u researcher Rscript -e 'install.packages(c("tidyverse", "caret", "randomForest"))'
exit

# 3. Save the configured instance as an AMI
prism ami save my-r-env "Team R Environment v1.0"

# 4. Launch from your custom AMI (2-3 minutes!)
prism workspace launch --ami "Team R Environment v1.0" quick-start
```

## Complete Workflow

### Step 1: Launch and Customize Base Template

Start with a Prism template that's close to your needs:

```bash
# Launch base R research environment
prism workspace launch r-rstudio-server my-base-env

# Check status
prism workspace list

# Connect to instance
prism workspace connect my-base-env
```

Inside the instance, customize your environment:

```bash
# Install additional R packages (using Posit Package Manager for speed)
sudo -u researcher Rscript -e '
options(repos = c(CRAN = "https://packagemanager.posit.co/cran/__linux__/noble/latest"))
install.packages(c("tidymodels", "arrow", "targets", "renv"))
'

# Install system dependencies
sudo apt-get update
sudo apt-get install -y libgdal-dev libproj-dev

# Configure RStudio settings for the researcher user
echo 'options(repos = c(CRAN = "https://packagemanager.posit.co/cran/__linux__/noble/latest"))' \
  >> /home/researcher/.Rprofile

# Exit when done
exit
```

### Step 2: Save as Custom AMI

Save your configured instance as an AMI:

```bash
# Save with a descriptive name
prism ami save my-base-env "Team R Environment v1.0"

# Output:
# Saving workspace my-base-env as AMI "Team R Environment v1.0"...
# This takes 5-10 minutes. The instance keeps running.
# Use 'prism ami list' to check progress.
```

**Important**: AMI creation does NOT stop your instance. You can continue working.

Check creation progress:

```bash
# List your AMIs with status
prism ami list

# Output:
# NAME                           AMI ID                STATE      CREATED
# Team R Environment v1.0        ami-0abc123def456      available  2026-03-01 10:30

# Get detailed info on a specific AMI
prism ami status ami-0abc123def456
```

Wait for state to become `available` (5-10 minutes).

### Step 3: Launch Instances from Your AMI

Once the AMI is available, launch instances quickly:

```bash
# Launch by AMI name (if you used a descriptive name)
prism workspace launch --ami "Team R Environment v1.0" researcher-1

# Launch by AMI ID
prism workspace launch --ami ami-0abc123def456 researcher-1

# Launch multiple instances (workshop scenario)
for i in {1..20}; do
  prism workspace launch --ami "Team R Environment v1.0" workshop-instance-$i &
done
wait
# All 20 instances launch in parallel!
```

Launch with custom size and spot pricing:

```bash
# Launch with specific instance size
prism workspace launch --ami "Team R Environment v1.0" analysis-job --size XL --spot
```

Verify your custom environment is intact:

```bash
# Connect and test
prism workspace connect researcher-1

# Check that your packages and scripts are present
Rscript -e 'library(tidymodels); cat("tidymodels available\n")'
```

### Step 4: Version Management

Use clear naming conventions to track AMI versions:

```bash
# Include version and date in the AMI name
prism ami save my-env "Team R Environment v1.0 - 2026-03"
prism ami save my-env-v2 "Team R Environment v1.1 - 2026-06"

# Recommended naming patterns:
# "R 4.4 + Seurat 5 + DESeq2 - 2026-03"         # genomics stack
# "Python ML + PyTorch 2.2 - Stats 510 Fall 2026"  # course environment
# "Shiny + leaflet + DT - Lab Dashboard"           # application base
```

List and manage AMIs:

```bash
# List all your AMIs
prism ami list

# Get details on a specific AMI
prism ami status <ami-id>

# Delete AMIs you no longer need
prism ami delete <ami-id>
```

**Sharing AMIs with team members**: Use the AWS Console (EC2 → AMIs) to share AMIs with other AWS accounts or copy them to other regions. Prism does not yet have CLI commands for cross-account AMI sharing.

## Command Reference

### `prism ami save <workspace-name> <ami-name>`

Save a running or stopped workspace as a custom AMI.

```bash
# Basic save
prism ami save my-instance "My Custom AMI"

# Include version info in name
prism ami save my-instance "Production R Environment v2.0 - 2026-03"
```

**Notes:**
- The instance continues running while the AMI is being created
- AMI creation takes 5-10 minutes
- The AMI name can be used with `prism workspace launch --ami`

### `prism ami create <template-name>`

Build an AMI from a template definition (alias for `prism ami build`). This pre-bakes the entire provisioning script so future launches skip the setup phase.

```bash
# Build an AMI from the r-rstudio-server template
prism ami create r-rstudio-server
```

### `prism ami list`

List available AMIs.

```bash
# List all your AMIs
prism ami list

# Output:
# NAME                        AMI ID                STATE      CREATED
# Team R Environment v1.0     ami-0abc123def456      available  2026-03-01
```

### `prism ami status <ami-id>`

Show detailed information about a specific AMI.

```bash
prism ami status ami-0abc123def456
```

### `prism ami delete <ami-id>`

Delete an AMI and its associated snapshots.

```bash
# Interactive deletion (with confirmation)
prism ami delete ami-0abc123def456

# Force deletion (no prompt, for scripts)
prism ami delete ami-0abc123def456 --force
```

### `prism workspace launch --ami`

Launch an instance from a custom AMI.

```bash
# Launch by AMI name
prism workspace launch --ami "Team R Environment v1.0" my-instance

# Launch by AMI ID
prism workspace launch --ami ami-0abc123def456 my-instance

# With custom size and spot pricing
prism workspace launch --ami "Team R Environment v1.0" analysis-job \
  --size XL --spot

# In specific region
prism workspace launch --ami ami-0abc123def456 eu-instance --region eu-west-1
```

## Use Cases

### 1. Team Consistency

**Problem**: Team members have different package versions, causing "works on my machine" issues.

**Solution**: Everyone launches from the same AMI.

```bash
# Team lead creates standardized environment
prism workspace launch r-rstudio-server team-base
prism workspace connect team-base
# ... install and configure team packages ...
exit
prism ami save team-base "Team Environment 2026.1"

# Share the AMI ID with team (use AWS Console to share cross-account)
prism ami list  # get the AMI ID

# Everyone launches identical environment
prism workspace launch --ami "Team Environment 2026.1" my-work
```

**Result**: Zero configuration drift, consistent results across team.

### 2. Conference Workshops

**Problem**: Need to provision 40 identical instances for workshop attendees.

**Solution**: Pre-create AMI, launch all instances quickly.

```bash
# Before workshop: Create AMI
prism workspace launch r-publishing-stack workshop-template
prism workspace connect workshop-template
# ... set up workshop materials ...
exit
prism ami save workshop-template "Workshop Environment 2026"

# During workshop: Launch for all attendees (parallel)
for i in {1..40}; do
  prism workspace launch --ami "Workshop Environment 2026" workshop-attendee-$i &
done
wait

# Result: 40 instances ready in ~5 minutes
```

**Result**: Happy attendees, smooth workshop, minimal wait time.

### 3. Fast Iteration for Development

**Problem**: Need to test code changes repeatedly with clean environments.

**Solution**: Launch → test → terminate cycle in minutes.

```bash
# Create base testing AMI once
prism workspace launch r-base-ubuntu24 test-base
# ... configure test environment ...
prism ami save test-base "Testing Environment"

# Fast iteration loop
while true; do
  # Launch test instance
  prism workspace launch --ami "Testing Environment" test-run

  # Get IP and run tests via SSH
  prism workspace connect test-run

  # Terminate
  prism workspace delete test-run

  # Repeat!
done
```

**Result**: Much faster development iteration.

### 4. Pre-installed Research Packages

**Problem**: Installing bioinformatics packages takes hours.

**Solution**: Install once in AMI, use forever.

```bash
# One-time: Create AMI with all packages
prism workspace launch r-rstudio-server bioinfo-base
prism workspace connect bioinfo-base
# ... install all Bioconductor packages (2-3 hours) ...
exit

prism ami save bioinfo-base "Bioinformatics Environment 2026"

# Every subsequent launch: 2-3 minutes!
prism workspace launch --ami "Bioinformatics Environment 2026" analysis-1
```

**Result**: Save hours per launch, happier researchers.

### 5. Backup Before Major Changes

**Problem**: Need to save instance state before a risky operation.

**Solution**: Create AMI as snapshot before making changes.

```bash
# Before dangerous operation
prism ami save my-instance "Backup before migration $(date +%Y%m%d)"

# Perform risky operation
prism workspace connect my-instance
# ... make changes ...

# If something breaks, restore from AMI
prism workspace launch --ami "Backup before migration 20260301" my-instance-restored
```

**Result**: Confidence to experiment, easy rollback.

## Performance Comparison

### Time Savings

| Scenario | Template | AMI | Time Saved |
|----------|----------|-----|------------|
| **Single launch** | 5-20 min | 2-3 min | **~15 min** (75-85% faster) |
| **10 instances** | 50-200 min | 20-30 min | **~2.5 hours** |
| **40 instances (workshop)** | 3+ hours | 5-10 min | **~3 hours** |

### Storage Costs

AMIs cost ~$0.05/GB/month for EBS snapshots:

```
Typical AMI sizes:
- R + RStudio: ~15 GB = $0.75/month
- Python ML: ~18 GB = $0.90/month
- Full Research Stack: ~25 GB = $1.25/month
```

**Conclusion**: AMIs pay for themselves in time savings after just a few launches.

## Troubleshooting

### AMI Creation Fails

**Symptom**: AMI save hangs or fails.

**Solutions**:

1. **Verify instance is running or stopped (not in a transitional state)**
   ```bash
   # Check instance state
   prism workspace list
   # Must be 'running' or 'stopped'
   ```

2. **Check AWS EBS snapshot limits in Console**
   - AWS Console → Service Quotas → EBS
   - Delete old unused AMIs if needed: `prism ami list` then `prism ami delete <id>`

### AMI Launch Fails

**Symptom**: Cannot launch instance from AMI.

**Solutions**:

1. **AMI not in correct region** — AMIs are region-specific
   ```bash
   # Check your current region
   prism profiles current

   # To use the AMI in another region, copy it via AWS Console:
   # EC2 → AMIs → select AMI → Actions → Copy AMI
   ```

2. **AMI not found by name** — try using the AMI ID
   ```bash
   prism ami list  # get the AMI ID
   prism workspace launch --ami ami-0abc123def456 my-instance
   ```

### High Storage Costs

**Solution**: Delete unused AMIs

```bash
# List all your AMIs
prism ami list

# Delete AMIs you no longer need
prism ami delete ami-old123def456
```

## Next Steps

- **R Template Guide**: See [R Getting Started Guide](R_GETTING_STARTED.md) for R-specific AMI creation tips
- **Marketplace**: `prism marketplace list` — browse community templates
- **Join Community**: Share your workflows at https://github.com/scttfrdmn/prism/discussions

## Related Documentation

- [Getting Started](QUICK_START.md) - Core workflow overview
- [R Getting Started](R_GETTING_STARTED.md) - R-specific guidance including AMI creation
- Cost Management - Optimizing AMI storage costs
