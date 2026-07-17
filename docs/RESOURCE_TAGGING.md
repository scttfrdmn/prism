# Prism Resource Tagging System

**Issue #128: Enhanced Resource Tagging and Zombie Cleanup**

This document describes Prism's comprehensive resource tagging strategy for AWS cost optimization, compliance, and operational excellence.

## Table of Contents

- [Overview](#overview)
- [Tagging Schema](#tagging-schema)
- [Tag Categories](#tag-categories)
- [AWS Cost Explorer Integration](#aws-cost-explorer-integration)
- [Zombie Resource Detection](#zombie-resource-detection)
- [Cleanup Procedures](#cleanup-procedures)
- [Best Practices](#best-practices)

## Overview

Prism automatically applies comprehensive tags to all AWS resources (EC2 instances, EBS volumes, EFS file systems) to enable:

- **Cost Optimization**: Automated zombie resource detection preventing $12K+/year waste
- **Cost Allocation**: Per-project cost tracking via AWS Cost Explorer
- **Audit Trail**: Compliance-ready lifecycle tracking (who launched what, when)
- **Budget Integration**: Direct linking to v0.5.10 multi-budget system
- **Operational Excellence**: Clear resource ownership and management

### Namespacing

Prism uses the **`prism:*`** namespace for all custom tags following AWS best practices:
- Prevents tag conflicts with other tools
- Enables bulk filtering (`prism:*`)
- Clearly identifies Prism-managed resources
- Professional appearance for institutional deployments

## Tagging Schema

### Core Tags (Always Applied)

Every Prism-launched instance receives these tags:

| Tag Key | Example Value | Purpose |
|---------|---------------|---------|
| `Name` | `my-ml-workstation` | User-friendly instance name |
| `prism:managed` | `true` | **Zombie detection key** - identifies Prism resources |
| `prism:version` | `0.5.10-dev` | Prism version for troubleshooting |
| `prism:instance-id` | `my-ml-workstation` | Prism's internal instance identifier |
| `prism:template` | `python-ml` | Template used for launch |
| `prism:package-manager` | `conda` | Package manager configured |
| `prism:primary-user` | `researcher` | Primary Linux username |
| `prism:launched-at` | `2025-11-08T19:16:56Z` | ISO 8601 launch timestamp |
| `prism:launched-by` | `jsmith` | OS username who launched instance |
| `Application` | `Prism` | AWS Cost Explorer application tag |
| `Environment` | `research` | AWS Cost Explorer environment tag |

**Legacy Tags** (maintained for backwards compatibility):
- `Prism=true`
- `LaunchedBy=Prism`
- `Template=python-ml`
- `PackageManager=conda`
- `PrimaryUser=researcher`

### Conditional Tags (Applied When Available)

These tags are added when the corresponding field is specified:

| Tag Key | Example Value | When Applied | Purpose |
|---------|---------------|--------------|---------|
| `prism:project-id` | `proj-abc123` | `--project-id` flag used | Project association |
| `CostCenter` | `proj-abc123` | `--project-id` flag used | AWS Cost Explorer project costs |
| `prism:funding-allocation-id` | `alloc-xyz789` | v0.5.10 budget system | Budget tracking |
| `prism:research-user` | `phd-student` | Phase 5A multi-user | Research user identity |

## Tag Categories

### 1. Identification Tags

**Purpose**: Uniquely identify and classify resources

```
prism:managed = true          # CRITICAL: Zombie detection
prism:version = 0.5.10-dev    # Version tracking
prism:instance-id = my-instance
prism:template = python-ml    # Workload classification
```

**Use Cases**:
- Zombie resource detection (resources without `prism:managed=true`)
- Template usage analytics
- Version-specific troubleshooting

### 2. Lifecycle Tags

**Purpose**: Audit trail and compliance

```
prism:launched-at = 2025-11-08T19:16:56Z  # RFC3339 format
prism:launched-by = jsmith                 # OS username
```

**Use Cases**:
- Compliance audits ("who launched this expensive GPU instance?")
- Usage pattern analysis
- Security incident investigation
- Automated cleanup of old resources

### 3. Cost Allocation Tags

**Purpose**: AWS Cost Explorer integration

```
Application = Prism           # Group all Prism costs
Environment = research        # Distinguish from production
CostCenter = proj-abc123      # Per-project cost tracking
```

**Use Cases**:
- Monthly cost reports per project
- Budget alerts by project
- Institutional cost reporting
- Grant spending verification

### 4. Budget Integration Tags

**Purpose**: v0.5.10 Multi-Budget System integration

```
prism:project-id = proj-abc123
prism:funding-allocation-id = alloc-xyz789
```

**Use Cases**:
- Link instances to project budgets
- Track spending by funding source
- Automated budget exhaustion detection
- Multi-source funding attribution

### 5. Multi-User Tags

**Purpose**: Phase 5A Research User System integration

```
prism:research-user = phd-student
prism:primary-user = researcher
```

**Use Cases**:
- User-specific cost attribution
- Multi-tenant resource management
- Collaboration tracking
- User onboarding/offboarding

## AWS Cost Explorer Integration

### Enabling Cost Allocation Tags

1. **AWS Console → Billing → Cost Allocation Tags**
2. **Activate these tags**:
   - `Application`
   - `Environment`
   - `CostCenter`
   - `prism:project-id`
   - `prism:template`

3. **Wait 24 hours** for tags to appear in Cost Explorer

### Creating Cost Reports

**Example 1: Per-Project Monthly Costs**
```
Filter: CostCenter (contains) proj-
Group by: CostCenter
Time range: Last 12 months
```

**Example 2: Template Usage Costs**
```
Filter: Application = Prism
Group by: prism:template
Time range: Current month
```

**Example 3: Research Environment Total**
```
Filter: Environment = research
Group by: Application
Time range: Year to date
```

### Budget Alerts

Create budget alerts using cost allocation tags:

```bash
aws budgets create-budget \
  --account-id 123456789012 \
  --budget file://project-budget.json \
  --notifications-with-subscribers file://alerts.json
```

**project-budget.json**:
```json
{
  "BudgetName": "Project ABC Monthly Budget",
  "BudgetLimit": {
    "Amount": "1000",
    "Unit": "USD"
  },
  "CostFilters": {
    "TagKeyValue": ["user:CostCenter$proj-abc123"]
  },
  "TimeUnit": "MONTHLY",
  "BudgetType": "COST"
}
```

## Zombie Resource Detection

### What Are Zombie Resources?

**Zombie resources** are AWS instances/volumes that:
- Are running or stopped (consuming money)
- Were NOT launched by Prism (missing `prism:managed=true`)
- Have no clear owner or purpose
- Often forgotten after experiments/testing

**Real Example**: 3 zombie instances discovered:
- Cost: **$1,054.60/month** ($12,655/year)
- Duration: Running for months unnoticed
- Owner: Unknown (no tags)
- Resolution: Manual cleanup took 30+ minutes

### Automated Detection

Prism's cleanup script automatically detects zombies by checking for the **`prism:managed=true`** tag.

**Detection Logic**:
```bash
# New namespaced tag (preferred)
if [ "$(aws ec2 describe-tags --filters Name=key,Values=prism:managed)" = "true" ]; then
    # Prism-managed ✓
fi

# Legacy fallback (backwards compatible)
if [ "$(aws ec2 describe-tags --filters Name=key,Values=Prism)" = "true" ]; then
    # Prism-managed ✓
fi

# Neither tag present → ZOMBIE ✗
```

## Cleanup Procedures

### Manual Cleanup Script

Location: **`scripts/cleanup_untagged_resources.sh`**

#### Basic Usage

**1. Dry Run (Safe - No Deletions)**
```bash
./scripts/cleanup_untagged_resources.sh
```

Output:
```
🔍 Scanning for EC2 instances...
✗ i-0abc123def (running, t3.xlarge) - ZOMBIE: old-test-instance (launched: 2025-10-01T...)
✗ i-0def456abc (stopped, m5.large) - ZOMBIE: forgotten-instance (launched: 2025-09-15T...)

Summary:
  Prism-managed instances: 5
  Zombie instances: 2

💰 Estimated monthly cost of zombie resources: $243.84
💰 Estimated yearly cost: $2,926.08

⚠️ DRY RUN MODE - No resources will be terminated
To actually terminate, run: DRY_RUN=false ./scripts/cleanup_untagged_resources.sh
```

**2. Interactive Cleanup (With Confirmation)**
```bash
DRY_RUN=false ./scripts/cleanup_untagged_resources.sh
```

Prompts:
```
⚠️ WARNING: This will PERMANENTLY DELETE the resources listed above!
Are you sure you want to proceed? (yes/no): yes

Terminating i-0abc123def...
Terminating i-0def456abc...
Deleting volume vol-xyz...

✓ Cleanup complete!
Estimated monthly savings: $243.84
```

**3. Automated Cleanup (For Scheduled Jobs)**
```bash
DRY_RUN=false FORCE=true ./scripts/cleanup_untagged_resources.sh
```

Skips confirmation - useful for scheduled jobs.

#### Advanced Usage

**Specify AWS Profile and Region:**
```bash
AWS_PROFILE=research \
AWS_REGION=us-west-2 \
./scripts/cleanup_untagged_resources.sh
```

**Check Exit Code (for automation):**
```bash
./scripts/cleanup_untagged_resources.sh
EXIT_CODE=$?

if [ $EXIT_CODE -eq 2 ]; then
    echo "Zombies found! Alerting team..."
    # Send Slack notification, create ticket, etc.
fi
```

**Exit Codes**:
- `0`: Success (no zombies found or cleanup successful)
- `1`: Error (AWS CLI issues, permissions, etc.)
- `2`: Zombies found (dry run only)

### Scheduled Cleanup (Institutional Automation)

#### Weekly Scan (Cron Job on Linux/macOS Server)

Add to crontab on your institutional server:
```bash
# Every Monday at 9 AM: Scan for zombies and send email if found
0 9 * * 1 /path/to/prism/scripts/cleanup_untagged_resources.sh > /tmp/zombie-report.txt 2>&1 || mail -s "Zombie Resources Detected" admin@university.edu < /tmp/zombie-report.txt
```

#### Windows Task Scheduler (Windows Server)

Create a scheduled task to run PowerShell wrapper:

**PowerShell Wrapper** (`zombie-scan.ps1`):
```powershell
# Run bash script via WSL or Git Bash
$env:AWS_PROFILE = "research"
$env:AWS_REGION = "us-east-1"

# If using WSL
wsl bash /path/to/prism/scripts/cleanup_untagged_resources.sh

# Or if using Git Bash
# & "C:\Program Files\Git\bin\bash.exe" /path/to/prism/scripts/cleanup_untagged_resources.sh

# Send email if zombies found
if ($LASTEXITCODE -eq 2) {
    Send-MailMessage -To "admin@university.edu" `
        -From "prism@university.edu" `
        -Subject "Zombie Resources Detected" `
        -Body "Zombie AWS resources detected. Check server logs." `
        -SmtpServer "smtp.university.edu"
}
```

**Schedule in Task Scheduler**:
- Trigger: Weekly, Monday 9:00 AM
- Action: Run PowerShell script
- Run as: Service account with AWS credentials

#### Native Prism Daemon Monitoring (Coming Soon - Issue #237)

**Future**: Prism daemon will include native zombie detection:
```bash
# Configure automatic scanning
prism admin zombies config --scan-interval 24h --alert-email admin@university.edu

# Daemon will automatically scan and alert
# Cross-platform (Windows, macOS, Linux)
# No need for cron or Task Scheduler
```

See [Issue #237](https://github.com/scttfrdmn/prism/issues/237) for native zombie detection roadmap.

## Best Practices

### 1. Tag Consistency

**DO**: Always launch instances via Prism CLI/API
```bash
prism workspace launch python-ml my-instance --project-id proj-abc
```
✓ Automatically gets all required tags

**DON'T**: Launch instances manually via AWS Console
```
AWS Console → Launch Instance → ...
```
✗ Missing Prism tags → detected as zombie

### 2. Project Association

**Always specify `--project-id` when available**:
```bash
prism workspace launch r-research analysis-01 --project-id nsf-grant-2024
```

Benefits:
- AWS Cost Explorer per-project costs
- Budget integration
- Clear ownership
- Easy cost attribution

### 3. Regular Zombie Scans

**Recommended frequency**:
- **Weekly**: Dry-run scans with email alerts
- **Monthly**: Interactive cleanup with review
- **Quarterly**: Full audit of all resources

**Set up alerts**:
```bash
# Add to ~/.bashrc or company scripts
alias zombie-scan='cd ~/prism && ./scripts/cleanup_untagged_resources.sh'
```

### 4. Cost Allocation Tag Activation

**Enable in AWS Billing Console**:
1. Billing → Cost Allocation Tags
2. Activate:
   - `Application`
   - `Environment`
   - `CostCenter`
   - `prism:project-id`
   - `prism:template`

**Wait 24 hours** before creating reports.

### 5. Legacy Migration

**For existing instances** (launched before v0.5.10):

Option A: Add tags manually
```bash
aws ec2 create-tags \
  --resources i-0abc123def \
  --tags Key=prism:managed,Value=true \
         Key=prism:version,Value=0.5.9 \
         Key=prism:template,Value=python-ml
```

Option B: Let cleanup script identify and migrate
```bash
# Dry run shows old instances
./scripts/cleanup_untagged_resources.sh

# Review and decide: tag or terminate
```

### 6. Institutional Policies

**For university/enterprise deployments**:

Create policy document:
```markdown
# Prism Resource Tagging Policy

## Requirements
- All instances MUST have prism:managed=true
- Project ID MUST be specified for grant-funded work
- Zombie scans run weekly via automated job
- Untagged resources are flagged after 7 days
- Untagged resources are terminated after 30 days (with notification)

## Exceptions
Contact cloudops@university.edu for:
- Long-running experiments (>90 days)
- Non-Prism AWS resources (justify business need)
- Shared infrastructure resources
```

## Troubleshooting

### Tags Not Appearing in Cost Explorer

**Problem**: Launched instance with tags but not in Cost Explorer

**Solution**:
1. **Verify tag activation** in Billing → Cost Allocation Tags
2. **Wait 24 hours** - AWS tag propagation delay
3. **Check time range** - Current month may be incomplete

### Cleanup Script Shows False Positives

**Problem**: Script detects Prism instances as zombies

**Cause**: Instance has legacy `Prism=true` but missing `prism:managed=true`

**Solution**: Script checks BOTH tags (new + legacy). If false positives persist:
```bash
# Check instance tags
aws ec2 describe-tags --filters "Name=resource-id,Values=i-YOUR-INSTANCE-ID"

# Add missing tag if needed
aws ec2 create-tags --resources i-YOUR-INSTANCE-ID \
  --tags Key=prism:managed,Value=true
```

### Permission Errors

**Problem**: `AccessDenied` when running cleanup script

**Required IAM Permissions**:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances",
        "ec2:DescribeVolumes",
        "ec2:DescribeTags",
        "ec2:TerminateInstances",
        "ec2:DeleteVolume"
      ],
      "Resource": "*"
    }
  ]
}
```

## ROI Analysis

### Cost Savings Example

**Scenario**: Medium-sized research lab

**Before Prism Tagging**:
- 3 zombie instances discovered
- Average runtime: 6 months unnoticed
- Monthly cost: $1,054.60
- **Total waste: $6,327.60** (6 months)

**After Prism Tagging**:
- Weekly automated scans
- Email alerts on detection
- Resources caught within 1 week
- **Waste prevented: $6,083.97** (97% reduction)

**Annual ROI**:
- Implementation time: 2 hours
- Annual zombie prevention: **$12,655/year**
- **ROI: 6,327x** (assuming $100/hr engineering time)

### Institutional Impact

For university with 50 research groups:
- Average: 1 zombie/year per group (conservative)
- Average cost: $250/month/zombie
- **Total annual savings: $150,000/year**

Plus:
- **Compliance**: Audit-ready lifecycle tracking
- **Grant Management**: Per-project cost attribution
- **Budget Control**: Real-time spending visibility
- **Operational Excellence**: Clear resource ownership

## References

- Issue #128: Enhanced Resource Tagging and Zombie Cleanup
- [AWS Cost Allocation Tags](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/cost-alloc-tags.html)
- [AWS Cost Explorer](https://docs.aws.amazon.com/cost-management/latest/userguide/ce-what-is.html)
- [AWS Tagging Best Practices](https://docs.aws.amazon.com/whitepapers/latest/tagging-best-practices/welcome.html)

## Related Documentation

- Multi-Budget System (Issue #236 - planned)
- Project Management
- AWS Setup Guide
