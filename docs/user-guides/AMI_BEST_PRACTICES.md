# AMI Best Practices

Tips, optimization strategies, and common pitfalls for managing custom Amazon Machine Images (AMIs) with Prism.

## Table of Contents

- [Naming and Organization](#naming-and-organization)
- [Tagging Strategy](#tagging-strategy)
- [Security Best Practices](#security-best-practices)
- [Performance Optimization](#performance-optimization)
- [Cost Management](#cost-management)
- [Version Control](#version-control)
- [Testing and Validation](#testing-and-validation)
- [Team Collaboration](#team-collaboration)
- [Common Pitfalls](#common-pitfalls)
- [Advanced Tips](#advanced-tips)

## Naming and Organization

### Use Semantic Versioning

Follow semantic versioning (MAJOR.MINOR.PATCH) for AMI names:

```bash
# ✅ Good: Semantic versioning
prism ami create my-env --name "Team R Environment v1.0.0"
prism ami create my-env --name "Team R Environment v1.1.0"  # Feature added
prism ami create my-env --name "Team R Environment v2.0.0"  # Breaking change

# ❌ Bad: No versioning
prism ami create my-env --name "Team R Environment"
prism ami create my-env --name "Team R Environment New"
prism ami create my-env --name "Team R Environment Final"
```

### Include Environment in Name

Distinguish between development, staging, and production AMIs:

```bash
# ✅ Good: Environment in name
--name "R Environment v1.0.0 - Development"
--name "R Environment v1.0.0 - Production"

# ❌ Bad: Ambiguous environment
--name "R Environment v1.0.0"
```

### Use Consistent Naming Pattern

Establish a team naming convention:

```bash
# Pattern: [Team/Project] [Tool] [Version] - [Environment]
--name "Climate Team R v1.0.0 - Production"
--name "Climate Team Python v2.1.0 - Development"
--name "Genomics Lab Bioconductor v3.18 - Staging"
```

### Add Date for Backup AMIs

Include timestamps for backup/snapshot AMIs:

```bash
# ✅ Good: Timestamped backups
--name "R Environment - Backup 2026-01-16"
--name "R Environment - Pre-Migration 2026-01-15"

# Use ISO 8601 format for sortable dates
--name "R Environment - Backup $(date +%Y-%m-%d)"
```

## Tagging Strategy

### Essential Tags

Always include these tags:

```bash
prism ami tag ami-0abc123def456 --tags "\
version=1.0.0,\
environment=production,\
team=research,\
project=climate-analysis,\
created_by=jane@example.com,\
created_date=2026-01-16,\
os=ubuntu-22.04,\
languages=r-4.4.2"
```

### Tag for Cost Tracking

Enable AWS cost allocation with tags:

```bash
--tags "\
cost_center=research_dept,\
budget_code=grant-12345,\
project_id=climate-2026"
```

### Tag for Lifecycle Management

Track AMI lifecycle and dependencies:

```bash
--tags "\
status=active,\            # active, deprecated, archived
tested=true,\              # Passed validation tests
replacement_for=ami-old123,\  # Previous version
replaced_by=ami-new456"    # Newer version (after deprecation)
```

### Automation-Friendly Tags

Use tags for automated cleanup and management:

```bash
--tags "\
auto_delete_after=2026-06-01,\   # Automated cleanup date
backup_retention=90days,\         # Retention policy
compliance=phi_approved"          # Regulatory compliance
```

## Security Best Practices

### Audit Before Sharing

**Always review AMI contents before making it accessible to others:**

```bash
# 1. Launch test instance from AMI
prism workspace launch --ami ami-0abc123def456 security-audit

# 2. Connect and audit
prism workspace connect security-audit

# Inside instance, check for sensitive data:
# - SSH keys: ~/.ssh/*
# - AWS credentials: ~/.aws/credentials
# - Passwords: /etc/shadow, application configs
# - Personal data: browser history, email, documents
# - API tokens: .env files, config files

# 3. Clean sensitive data if found
sudo find /home -name "*.pem" -type f
sudo find /home -name "credentials" -type f
sudo find /home -name ".env" -type f

# 4. Delete test instance
exit
prism workspace delete security-audit --force
```

### Remove Credentials

**Never include AWS credentials or SSH keys in AMIs:**

```bash
# Before creating AMI, connect to instance and clean:
prism workspace connect my-instance

# Remove AWS credentials
rm -rf ~/.aws/credentials
rm -rf ~/.aws/config

# Remove SSH keys
rm -rf ~/.ssh/id_*
rm -rf ~/.ssh/authorized_keys  # Careful! May break access

# Remove bash history
history -c
rm ~/.bash_history

# Remove temporary files
sudo rm -rf /tmp/*
sudo rm -rf /var/tmp/*

exit

# Now create AMI
prism ami create my-instance --name "Clean AMI v1.0"
```

### Use IAM Roles Instead

**Prefer IAM instance roles over embedded credentials:**

```bash
# ✅ Good: Launch with IAM role
prism workspace launch --ami ami-0abc123def456 my-instance \
  --iam-role researcher-role

# Inside instance, AWS SDK uses instance role automatically
# No credentials in AMI needed!

# ❌ Bad: Embedding credentials in AMI
# (credentials in ~/.aws/credentials, baked into AMI)
```

### Encrypt AMIs

**Use EBS encryption for sensitive data:**

```bash
# Create encrypted AMI (if base instance has encrypted volumes)
prism ami create my-instance \
  --name "Encrypted Environment v1.0" \
  --encrypted

# Or copy unencrypted AMI to encrypted
prism ami copy ami-unencrypted \
  --source-region us-west-2 \
  --regions us-west-2 \
  --encrypted \
  --kms-key arn:aws:kms:us-west-2:123456789012:key/abc-123
```

### Limit Sharing Scope

**Share AMIs only with accounts that need them:**

```bash
# ✅ Good: Share with specific accounts
prism ami share ami-0abc123def456 \
  --accounts 123456789012,987654321098

# ⚠️  Caution: Organization-wide sharing
prism ami share ami-0abc123def456 \
  --organization-id o-abc123def

# ❌ Dangerous: Public sharing (only for truly public AMIs)
prism ami share ami-0abc123def456 --public
```

## Performance Optimization

### Minimize AMI Size

**Smaller AMIs = faster creation and launches:**

```bash
# Before creating AMI, clean unnecessary files:
prism workspace connect my-instance

# Clean package manager caches
sudo apt-get clean                # Ubuntu/Debian
sudo yum clean all                # RHEL/CentOS
sudo dnf clean all                # Fedora

# Remove old kernels (keep current)
sudo apt-get autoremove --purge

# Clear logs
sudo rm -rf /var/log/*.log
sudo rm -rf /var/log/*/*.log

# Clear temp files
sudo rm -rf /tmp/*
sudo rm -rf /var/tmp/*

# Clear user caches
rm -rf ~/.cache/*

# Clear R package build artifacts
rm -rf ~/R/x86_64-pc-linux-gnu-library/*/src/*

exit

# Check AMI size after creation
prism ami describe ami-0abc123def456
# Look for BlockDeviceMapping size
```

### Use No-Reboot for Speed

**Trade reliability for speed with --no-reboot:**

```bash
# Normal (slower, more reliable)
prism ami create my-instance --name "My AMI"
# Reboots instance, ensures file system consistency
# Takes: 8-12 minutes

# Fast (faster, slight risk)
prism ami create my-instance --name "My AMI" --no-reboot
# No reboot, live snapshot
# Takes: 5-8 minutes
# Risk: Minor file system inconsistency if heavy disk I/O
```

**When to use --no-reboot:**
- ✅ Development/testing AMIs
- ✅ Instance with minimal disk activity
- ✅ Need fast iteration
- ❌ Production AMIs (use default reboot)
- ❌ Database instances
- ❌ Active file writes

### Stop Instance Before AMI Creation

**Stopping instance before AMI creation is faster and safer:**

```bash
# ✅ Recommended workflow:
prism workspace stop my-instance
# Wait for instance to fully stop
prism workspace status my-instance  # Should show 'stopped'
prism ami create my-instance --name "My AMI"
# Fastest and most reliable

# vs.

# ⚠️  Slower workflow:
prism ami create my-instance --name "My AMI"
# Forces reboot, slower
```

### Pre-warm AMI with Snapshots

**Use EBS snapshots to speed up subsequent AMI operations:**

```bash
# AWS automatically manages EBS snapshots
# But you can verify snapshot status:
prism ami describe ami-0abc123def456
# Check SnapshotId and State

# Snapshots are incremental - subsequent AMIs from same instance are faster
```

## Cost Management

### Regular Cleanup

**Delete unused AMIs to save on EBS snapshot storage:**

```bash
# Monthly cleanup script
#!/bin/bash
# cleanup-old-amis.sh

# List AMIs older than 90 days
OLD_DATE=$(date -d '90 days ago' +%Y-%m-%d)
prism ami list --owner self --json | \
  jq -r ".[] | select(.CreationDate < \"$OLD_DATE\") | .ImageId" | \
  while read ami_id; do
    echo "Deleting old AMI: $ami_id"
    prism ami delete $ami_id --force
  done
```

Run monthly:

```bash
chmod +x cleanup-old-amis.sh
# Add to crontab:
0 0 1 * * /path/to/cleanup-old-amis.sh
```

### Cost-Benefit Analysis

**Calculate whether an AMI is worth the storage cost:**

```
AMI Storage Cost:
- Typical AMI: 12 GB
- Cost: 12 GB × $0.05/GB/month = $0.60/month

Time Saved per Launch:
- Template: 4 minutes
- AMI: 30 seconds
- Savings: 3.5 minutes per launch

Monthly Usage:
- Launches/week: 10
- Launches/month: 40
- Time saved: 40 × 3.5 min = 140 minutes = 2.3 hours

Value of Time:
- Researcher time: $50/hour
- Monthly value: 2.3 hours × $50 = $115

ROI:
- Cost: $0.60/month
- Value: $115/month
- ROI: 19,066%

Conclusion: If you launch more than 1x/month, AMI is worth it!
```

### Archive Old Versions

**Move old AMIs to cheaper storage or delete:**

```bash
# Option 1: Tag for archival (don't delete yet)
prism ami tag ami-old123 --tags "status=archived,delete_after=2026-06-01"

# Option 2: Copy to backup region (cheaper in some regions)
prism ami copy ami-old123 \
  --source-region us-west-2 \
  --regions us-east-2 \  # Cheaper region
  --name "Archive - Old AMI"

# Then delete from expensive region
prism ami delete ami-old123 --region us-west-2
```

### Use Smallest Viable AMI

**Don't include unnecessary packages to save storage:**

```bash
# ❌ Bad: Kitchen sink AMI (20+ GB)
# Includes every possible package "just in case"

# ✅ Good: Focused AMI (8-12 GB)
# Only includes packages your team actively uses
```

Create role-specific AMIs:

```bash
# Instead of one huge AMI:
--name "Everything AMI"  # 25 GB

# Create specialized AMIs:
--name "R Basic"         # 8 GB
--name "R ML"            # 12 GB
--name "R Spatial"       # 10 GB
--name "R Bioconductor"  # 15 GB
```

## Version Control

### Maintain Version History

**Track AMI lineage with tags:**

```bash
# v1.0.0
prism ami create base --name "R Environment v1.0.0"
prism ami tag ami-v1 --tags "version=1.0.0,status=production"

# v1.1.0 (update from v1.0.0)
prism ami create updated --name "R Environment v1.1.0"
prism ami tag ami-v1.1 --tags "\
version=1.1.0,\
status=production,\
previous_version=ami-v1,\
changelog=Added tidymodels"

# v2.0.0 (breaking change)
prism ami create new-base --name "R Environment v2.0.0"
prism ami tag ami-v2 --tags "\
version=2.0.0,\
status=production,\
previous_version=ami-v1.1,\
breaking_changes=true,\
changelog=Upgraded R to 4.4.2"
```

### Document Changes

**Maintain CHANGELOG in AMI description or README:**

```bash
# Include changelog in AMI description
prism ami create my-env --name "R Environment v1.1.0" \
  --description "R 4.4.2 + RStudio + tidymodels

CHANGELOG v1.1.0:
- Added tidymodels ecosystem
- Updated arrow to 15.0.0
- Fixed RStudio Server configuration
- See: /home/researcher/CHANGELOG.md for details"
```

Or include README in the AMI:

```bash
# Before creating AMI:
prism workspace connect my-instance

cat > /home/researcher/README.md <<'EOF'
# R Environment v1.1.0

## What's Included
- R 4.4.2
- RStudio Server 2024.04
- tidyverse 2.0.0
- tidymodels 1.2.0
- arrow 15.0.0

## Changelog

### v1.1.0 (2026-01-16)
- Added tidymodels ecosystem
- Updated arrow to 15.0.0
- Fixed RStudio Server HTTPS configuration

### v1.0.0 (2026-01-01)
- Initial release
EOF

exit
prism ami create my-instance --name "R Environment v1.1.0"
```

### Deprecation Process

**Gracefully sunset old versions:**

```bash
# Step 1: Mark as deprecated (warn users)
prism ami deprecate ami-v1.0 \
  --deprecation-date 2026-03-01 \
  --replacement ami-v1.1

# Step 2: Update tags
prism ami tag ami-v1.0 --tags "\
status=deprecated,\
deprecated_date=2026-01-16,\
end_of_life=2026-03-01,\
replacement=ami-v1.1"

# Step 3: Delete after end-of-life date
# (2026-03-01 or later)
prism ami delete ami-v1.0
```

## Testing and Validation

### Always Test Before Sharing

**Validate AMIs work correctly before sharing with team:**

```bash
#!/bin/bash
# test-ami.sh - AMI validation script

AMI_ID=$1
TEST_INSTANCE="ami-test-$(date +%s)"

echo "Testing AMI: $AMI_ID"

# 1. Launch test instance
prism workspace launch --ami $AMI_ID $TEST_INSTANCE --size S
sleep 60  # Wait for boot

# 2. Check instance is running
if ! prism workspace status $TEST_INSTANCE | grep -q "running"; then
  echo "❌ FAIL: Instance not running"
  prism workspace delete $TEST_INSTANCE --force
  exit 1
fi

# 3. Run validation tests
prism workspace exec $TEST_INSTANCE "R --version"
prism workspace exec $TEST_INSTANCE "Rscript -e 'library(tidyverse)'"
prism workspace exec $TEST_INSTANCE "rstudio-server status"

# 4. Check for sensitive data
prism workspace exec $TEST_INSTANCE "[ ! -f ~/.aws/credentials ]"
prism workspace exec $TEST_INSTANCE "[ ! -f ~/.ssh/id_rsa ]"

# 5. Cleanup
prism workspace delete $TEST_INSTANCE --force

echo "✅ PASS: AMI validation complete"
```

Run before sharing:

```bash
chmod +x test-ami.sh
./test-ami.sh ami-0abc123def456
```

### Automated Testing in CI/CD

**Integrate AMI testing into your build pipeline:**

```yaml
# .github/workflows/ami-test.yml
name: Test AMI
on:
  push:
    paths:
      - 'amis/**'

jobs:
  test-ami:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Launch test instance
        run: |
          prism workspace launch --ami ${{ env.AMI_ID }} test-instance

      - name: Run tests
        run: |
          prism workspace exec test-instance "Rscript tests/validate-environment.R"

      - name: Cleanup
        if: always()
        run: |
          prism workspace delete test-instance --force
```

### Performance Testing

**Measure actual launch time:**

```bash
# Test launch speed
time prism workspace launch --ami ami-0abc123def456 speed-test

# Should complete in 30-60 seconds for AMI
# vs 3-5 minutes for template
```

## Team Collaboration

### Establish Team Conventions

**Document and enforce AMI standards:**

Create `TEAM_AMI_STANDARDS.md`:

```markdown
# Team AMI Standards

## Naming Convention
[Team] [Tool] v[X.Y.Z] - [Environment]

Examples:
- Climate Team R v1.0.0 - Production
- Genomics Lab Python v2.1.0 - Development

## Required Tags
- version
- environment
- team
- created_by
- created_date

## Testing Requirements
- Must pass ami-test.sh validation
- Must launch in < 90 seconds
- Must have README in /home/researcher/

## Security Requirements
- No AWS credentials
- No SSH private keys
- No PHI or sensitive data
- Must pass security-audit.sh
```

### Shared AMI Registry

**Maintain a team registry of available AMIs:**

```bash
# team-amis.json
{
  "amis": [
    {
      "id": "ami-0abc123def456",
      "name": "Climate Team R v1.0.0 - Production",
      "version": "1.0.0",
      "environment": "production",
      "owner": "jane@example.com",
      "created": "2026-01-16",
      "description": "R 4.4.2 + Climate analysis packages",
      "tested": true,
      "regions": ["us-west-2", "us-east-1"]
    }
  ]
}
```

### Communication

**Announce new AMIs to team:**

```bash
# Send notification when new AMI is ready
prism ami create my-env --name "Team R v2.0.0" && \
  echo "New AMI available: Team R v2.0.0 (ami-0abc123def456)" | \
  mail -s "New AMI Release" team@example.com
```

## Common Pitfalls

### Pitfall 1: Not Testing AMIs

**Problem**: Sharing untested AMIs that don't work.

**Solution**: Always launch and test before sharing:

```bash
# ❌ Bad
prism ami create my-env --name "Team AMI"
prism ami share ami-abc123 --accounts 111,222  # Untested!

# ✅ Good
prism ami create my-env --name "Team AMI"
./test-ami.sh ami-abc123  # Test first!
prism ami share ami-abc123 --accounts 111,222
```

### Pitfall 2: Forgetting to Clean Sensitive Data

**Problem**: AMIs contain AWS credentials or SSH keys.

**Solution**: Audit and clean before creating:

```bash
# Always run security audit
prism workspace connect my-instance
sudo find /home -name "*.pem" -type f
sudo find /home -name "credentials" -type f
rm -rf ~/.aws/credentials
rm -rf ~/.ssh/id_*
history -c && rm ~/.bash_history
exit
prism ami create my-instance --name "Clean AMI"
```

### Pitfall 3: No Version Control

**Problem**: Multiple AMIs named "Team Environment" with no way to distinguish.

**Solution**: Use semantic versioning:

```bash
# ❌ Bad
Team Environment
Team Environment New
Team Environment Final
Team Environment Final v2

# ✅ Good
Team Environment v1.0.0
Team Environment v1.1.0
Team Environment v2.0.0
```

### Pitfall 4: Not Cleaning Up Old AMIs

**Problem**: Accumulating dozens of unused AMIs, high storage costs.

**Solution**: Regular cleanup with automated scripts:

```bash
# Monthly cleanup
prism ami list --owner self | grep "v0\."  # Old versions
prism ami delete ami-old1 ami-old2 ami-old3
```

### Pitfall 5: Sharing AMI Without Documentation

**Problem**: Team members don't know what's in the AMI or how to use it.

**Solution**: Include README in AMI:

```bash
prism workspace connect my-instance
cat > /home/researcher/README.md <<'EOF'
# Team R Environment v1.0.0

## What's Inside
- R 4.4.2
- RStudio Server (port 8787)
- Packages: tidyverse, tidymodels, arrow

## Quick Start
1. Connect: prism workspace connect <instance>
2. Open RStudio: http://<instance-ip>:8787
3. Username/password: researcher/researcher

## Support
Contact: jane@example.com
EOF
exit
prism ami create my-instance --name "Documented AMI"
```

### Pitfall 6: Region Lock-in

**Problem**: AMI only in one region, can't launch elsewhere.

**Solution**: Copy to all regions your team uses:

```bash
# Copy to common regions
prism ami copy ami-0abc123def456 \
  --source-region us-west-2 \
  --regions us-east-1,eu-west-1,ap-southeast-1
```

### Pitfall 7: Large AMI Sizes

**Problem**: 30+ GB AMIs that take 20+ minutes to create.

**Solution**: Clean before creating:

```bash
prism workspace connect my-instance
sudo apt-get clean
sudo rm -rf /tmp/* /var/tmp/*
rm -rf ~/.cache/*
exit
prism ami create my-instance --name "Compact AMI"
```

## Advanced Tips

### Automated AMI Creation

**Create AMIs automatically on schedule:**

```bash
#!/bin/bash
# daily-ami-backup.sh

INSTANCE="production-instance"
DATE=$(date +%Y-%m-%d)
AMI_NAME="Production Backup - $DATE"

# Create daily backup AMI
prism ami create $INSTANCE --name "$AMI_NAME" --tags "type=backup,date=$DATE"

# Delete backups older than 7 days
OLD_DATE=$(date -d '7 days ago' +%Y-%m-%d)
prism ami list --owner self --tags "type=backup" --json | \
  jq -r ".[] | select(.CreationDate < \"$OLD_DATE\") | .ImageId" | \
  while read ami_id; do
    prism ami delete $ami_id --force
  done
```

### AMI as Code

**Store AMI configurations in version control:**

```yaml
# team-amis.yaml
amis:
  - name: "Team R Environment"
    version: "1.0.0"
    base_template: "r-research"
    packages:
      - tidyverse
      - tidymodels
      - arrow
    system_packages:
      - libgdal-dev
      - libproj-dev
    scripts:
      - setup-rstudio.sh
      - configure-r-packages.sh
```

Build from config:

```bash
# build-ami-from-config.sh
./scripts/launch-and-configure.sh team-amis.yaml
```

### Multi-Region AMI Strategy

**Optimize for global teams:**

```bash
# Primary region: Full AMI
prism ami create primary --name "Team AMI v1.0" --region us-west-2

# Copy to all team regions
for region in us-east-1 eu-west-1 ap-southeast-1; do
  prism ami copy ami-primary \
    --source-region us-west-2 \
    --regions $region \
    --name "Team AMI v1.0" &
done
wait
```

### AMI Inheritance

**Build AMIs from AMIs for specialized environments:**

```bash
# Base AMI: Core tools
prism workspace launch --ami ami-base-tools core-env
prism workspace connect core-env
# ... install common tools ...
exit
prism ami create core-env --name "Base Tools v1.0"

# Specialized AMI 1: ML Tools (built from base)
prism workspace launch --ami ami-base-tools ml-env
prism workspace connect ml-env
# ... install ML packages ...
exit
prism ami create ml-env --name "ML Tools v1.0"

# Specialized AMI 2: Spatial Tools (built from base)
prism workspace launch --ami ami-base-tools spatial-env
prism workspace connect spatial-env
# ... install spatial packages ...
exit
prism ami create spatial-env --name "Spatial Tools v1.0"
```

### Golden AMI Pipeline

**Automated pipeline for production AMIs:**

```
1. Developer creates feature AMI
2. Automated tests run
3. Security scan
4. Performance validation
5. Team review
6. Promote to "Golden AMI"
7. Copy to all regions
8. Announce to team
```

Implement with GitHub Actions or Jenkins.

## Summary Checklist

Before creating and sharing an AMI:

- [ ] Instance is fully configured and tested
- [ ] Removed all sensitive data (credentials, keys)
- [ ] Cleaned cache and temp files (reduced size)
- [ ] Added README in /home/researcher/
- [ ] Used semantic versioning in name
- [ ] Added comprehensive tags
- [ ] Tested AMI by launching and validating
- [ ] Documented changes in description
- [ ] Copied to all necessary regions
- [ ] Shared with appropriate accounts only
- [ ] Announced to team with documentation
- [ ] Added to team AMI registry

## Next Steps

- **Create Your First AMI**: See [CUSTOM_AMI_WORKFLOW.md](CUSTOM_AMI_WORKFLOW.md)
- **Browse Marketplace**: `prism marketplace browse`
- **Join Community**: Share tips at https://github.com/scttfrdmn/prism/discussions

## Related Documentation

- [Custom AMI Workflow](CUSTOM_AMI_WORKFLOW.md) - Step-by-step guide
- [Multi-User Instance Setup](MULTI_USER_INSTANCE_SETUP.md) - Team collaboration
- Cost Management - Optimizing costs
