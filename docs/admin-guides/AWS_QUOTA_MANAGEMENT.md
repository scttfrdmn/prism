# AWS Quota Management & Availability Handling Guide

**Status**: Planned for v0.6.0 (Q2 2026)
**Priority**: High
**GitHub Issue**: [#57](https://github.com/scttfrdmn/cloudworkstation/issues/57)

## Overview

AWS imposes service quotas (formerly called "limits") on resources to protect both AWS infrastructure and customer accounts. Researchers often encounter quota-related launch failures without understanding why or how to resolve them. CloudWorkStation v0.6.0 will provide intelligent quota management and automatic failover capabilities.

## Problem Statement

### Current Pain Points

1. **Opaque Quota Errors**: Generic error messages like "The requested configuration is currently not supported" don't explain the underlying quota issue
2. **No Proactive Awareness**: Users don't know they're approaching quota limits until launches fail
3. **Capacity Failures**: `InsufficientInstanceCapacity` errors provide no automatic retry logic
4. **Regional Outages**: No awareness of AWS Health events affecting launches

### Common Quota Types

| Quota Type | Common Default | What It Limits |
|------------|----------------|----------------|
| **Running On-Demand Standard Instances** | 32 vCPUs | Total vCPUs across A, C, D, H, I, M, R, T, Z instance families |
| **Running On-Demand G and VT Instances** | 8 vCPUs | GPU instances (P, G, Inf, DL, Trn families) |
| **Running On-Demand F Instances** | 8 vCPUs | FPGA instances |
| **Running On-Demand X Instances** | 8 vCPUs | High-memory instances |
| **EBS General Purpose SSD (gp3) storage** | 50 TiB | Total gp3 volume storage per region |
| **EBS Provisioned IOPS SSD (io2) storage** | 50 TiB | Total io2 volume storage per region |

**Example Scenario**: A researcher with 24 vCPUs already running tries to launch a `p3.8xlarge` (32 vCPUs). This would require 56 total vCPUs, exceeding the default 32 vCPU quota → **launch fails**.

---

## Planned Features (v0.6.0)

### 1. Quota Awareness System

**Module**: `pkg/aws/quota_manager.go`

Query and track AWS Service Quotas in real-time.

#### CLI Commands

```bash
# Show current quota status for default region
cws admin quota show

# Show quota status for specific region
cws admin quota show --region us-west-2

# Show quota status across all regions
cws admin quota show --all-regions

# Show quota history and trends
cws admin quota history --days 30
```

#### Example Output

```bash
$ cws admin quota show --region us-west-2

📊 AWS Service Quotas - us-west-2

vCPU Limits:
  Standard (A, C, D, H, I, M, R, T, Z): 24/32 (75% used) ⚠️
  GPU (P, G, Inf, DL, Trn):             0/8 (0% used) ✅
  High Memory (X, U):                   0/8 (0% used) ✅

Instance Type Limits:
  p3.2xlarge:  0/2 instances available ✅
  r5.xlarge:   3/5 instances available ⚠️ (approaching limit)
  t3.medium:   8/20 instances available ✅

Storage Quotas:
  EBS General Purpose (gp3):      3.2 TiB / 50 TiB ✅
  EBS Provisioned IOPS (io2):     0 TiB / 50 TiB ✅
  EFS Storage:                    73 GB (no regional limit) ✅

Recommendations:
  ⚠️  Standard vCPU usage at 75% - consider requesting increase
  ⚠️  r5.xlarge approaching instance limit (3/5 used)
  ✅ GPU quota sufficient for current workload
```

#### Pre-Launch Quota Validation

CloudWorkStation will check quotas **before** attempting launch:

```bash
$ cws launch gpu-ml-workstation protein-folding --size XL

⚠️  Quota Check Failed

    Instance type: p3.8xlarge (32 vCPUs, 4 GPUs)
    Current usage: 24/32 vCPUs in us-west-2
    After launch: 56/32 vCPUs ❌ (24 vCPUs over limit)

    You need to request a vCPU quota increase:
    1. Visit AWS Service Quotas Console:
       https://console.aws.amazon.com/servicequotas/home/services/ec2/quotas/L-1216C47A
    2. Request new limit: 64 vCPUs
       (Allows 2 simultaneous p3.8xlarge instances)
    3. Typical approval time: 24-48 hours

    Alternative Options:
    1. Launch p3.2xlarge instead? (8 vCPUs, 1 GPU) [Y/n]
    2. Stop existing instances to free quota? [y/N]
    3. Cancel launch [y/N]

Choice:
```

---

### 2. Quota Increase Assistance

**Module**: `pkg/aws/quota_requests.go`

Help users navigate the quota increase request process.

#### CLI Commands

```bash
# Request quota increase with guided workflow
cws admin quota request --instance-type p3.2xlarge \
  --reason "ML research for NIH-funded genomics project" \
  --desired-limit 16

# Check status of pending quota requests
cws admin quota requests list

# View quota request history
cws admin quota requests history
```

#### Guided Workflow

```bash
$ cws admin quota request --instance-type p3.8xlarge

🔍 Analyzing current usage...

Current Quota: 32 vCPUs (Standard)
Current Usage: 24 vCPUs
Requested Instance: p3.8xlarge (32 vCPUs, 4 GPUs)

📋 Quota Increase Request Wizard

1. How many p3.8xlarge instances do you need to run simultaneously?
   [ 2 ]

2. What is the use case? (helps AWS approve faster)
   [ ] Production workload
   [x] Research / Education
   [ ] Development / Testing
   [ ] Disaster recovery

3. Brief description (shown to AWS):
   [ Cancer genomics research using deep learning for tumor classification.
     NIH R01-funded project requiring GPU compute for PyTorch model training. ]

4. Duration of need:
   [x] Ongoing (default)
   [ ] Temporary (specify end date)

✅ Request Summary:
   Current Limit: 32 vCPUs
   Requested Limit: 64 vCPUs
   Justification: Research workload, NIH-funded cancer genomics project

   This request will be submitted to AWS Service Quotas.
   Typical approval time: 24-48 hours
   You will receive email notification when approved.

Submit request? [Y/n]: y

✅ Quota increase request submitted!
   Request ID: quota-12345678
   Track status: cws admin quota requests list
```

---

### 3. Intelligent AZ Failover

**Module**: `pkg/aws/availability_manager.go`

Automatic retry in different Availability Zones when capacity is unavailable.

#### How It Works

1. **Detect** `InsufficientInstanceCapacity` error from EC2
2. **Automatically retry** in different AZ within same region
3. **Track** AZ health per instance type (success rate)
4. **Prefer** AZs with recent successful launches

#### User Experience

```bash
$ cws launch bioinformatics-suite genome-analysis

✅ Launching r5.4xlarge in us-west-2a...
⚠️  InsufficientInstanceCapacity in us-west-2a
    AWS reports this instance type is temporarily unavailable in us-west-2a

🔄 Retrying in us-west-2b...
✅ Successfully launched in us-west-2b!
🔗 SSH ready in ~90 seconds...

💡 Note: Future launches will prefer us-west-2b for r5.4xlarge
   (Recent success rate: us-west-2b: 95%, us-west-2a: 60%)
```

#### Configuration

```bash
# Configure AZ failover behavior
cws admin config set az-failover.max-retries 3
cws admin config set az-failover.prefer-successful-azs true

# View AZ health statistics
cws admin availability stats --region us-west-2

# Output:
# 📊 Availability Zone Health - us-west-2
#
# r5.4xlarge:
#   us-west-2a: 12/20 launches successful (60%) ⚠️
#   us-west-2b: 19/20 launches successful (95%) ✅
#   us-west-2c: 18/20 launches successful (90%) ✅
#   us-west-2d: 15/20 launches successful (75%) ⚠️
#
# Recommendation: Prefer us-west-2b or us-west-2c for r5.4xlarge
```

---

### 4. AWS Health Dashboard Integration

**Module**: `pkg/aws/health_monitor.go`

Monitor AWS Health API for service events affecting launches.

#### Features

- Detect regional outages, degraded performance, scheduled maintenance
- Proactive notifications **before** launch attempts
- Block launches to affected regions with clear explanations
- Auto-suggest alternative healthy regions

#### CLI Commands

```bash
# Check AWS health status for all regions
cws admin aws-health

# Check specific region
cws admin aws-health --region us-east-1

# Subscribe to health alerts
cws admin aws-health subscribe --email devops@university.edu
```

#### Pre-Launch Health Check

```bash
$ cws launch python-ml earthquake-prediction --region us-east-1

⚠️  AWS Health Alert: Degraded EC2 Performance in us-east-1

    Event ID: AWS_EC2_INSTANCE_LAUNCH_FAILURE
    Status: Open (AWS engineers investigating)
    Started: 15 minutes ago
    Impact: Elevated instance launch failures
    Affected AZs: us-east-1a, us-east-1b

    Details: Increased error rates for On-Demand instance launches.
    AWS is actively working to resolve this issue.

    Recommendations:
    1. Use us-west-2 (healthy) ✅
    2. Use eu-west-1 (healthy) ✅
    3. Wait ~30 minutes for resolution ⏱️
    4. Launch anyway (may experience delays) ⚠️

Choice [1-4]:
```

#### Important: AWS Health API Requirements

**AWS Health API** requires **Business or Enterprise Support** for full programmatic access.

| Support Tier | Health API Access | Cost |
|--------------|-------------------|------|
| Basic | Console only | Free |
| Developer | Console only | $29/month |
| **Business** | **Full API access** | $100/month |
| Enterprise | Full API access | $15,000/month |

CloudWorkStation will gracefully degrade if Health API is unavailable (Basic/Developer support).

---

### 5. Capacity Planning

**Module**: `pkg/aws/capacity_planner.go`

Analyze historical launch patterns and recommend optimal regions/AZs.

#### Features

- Track launch success rates per region/AZ/instance-type
- Recommend regions with best availability
- Warn about high-demand instance types
- Suggest Spot instances when On-Demand capacity constrained

#### CLI Commands

```bash
# Get capacity recommendations for instance type
cws admin capacity recommend --instance-type p3.8xlarge

# View historical capacity data
cws admin capacity history --instance-type p3.8xlarge --days 30
```

#### Example Output

```bash
$ cws admin capacity recommend --instance-type p3.8xlarge

📊 Capacity Recommendations: p3.8xlarge

Best Regions (Last 30 days):
  1. us-west-2:  98% success rate (287/292 launches) ✅
  2. us-east-1:  94% success rate (245/261 launches) ✅
  3. eu-west-1:  91% success rate (156/171 launches) ✅

High-Demand Instance Type: ⚠️
  - p3.8xlarge is frequently capacity-constrained
  - Success rate varies significantly by AZ
  - Consider using Spot instances (60-80% cost savings)

Alternative Options:
  - p3.16xlarge: 92% success rate (more availability)
  - g5.12xlarge: 97% success rate (newer generation, better availability)

Spot Instance Recommendation: ✅
  - Spot availability: 95% (rarely interrupted)
  - Cost savings: $17.10/hr → $5.13/hr (70% off)
  - Recommended for workloads that can tolerate interruption
```

---

## Integration with Persona Workflows

### Solo Researcher (Persona 01)
**Benefit**: Pre-launch quota validation prevents failed launches and wasted time
- Check quota before launching expensive GPU instance
- Guided quota request for ML workload
- Automatic AZ failover for high-availability

### Lab Environment (Persona 02)
**Benefit**: Multi-user quota management across lab projects
- Lab-wide quota tracking (all team members' usage)
- Proactive alerts when lab approaches quota limits
- Coordinated quota increase requests

### University Class (Persona 03)
**Benefit**: Prevent student launch failures during class
- Pre-class quota validation (50 students launching simultaneously)
- Request quota increase for semester before classes start
- Real-time AZ failover during high-demand periods

### Conference Workshop (Persona 04)
**Benefit**: Ensure 60-participant workshop launches reliably
- Pre-event quota validation and increase requests
- AWS Health monitoring to detect regional issues
- Automatic AZ failover for workshop instances

### Cross-Institutional (Persona 05)
**Benefit**: Multi-region quota management for distributed collaborators
- Quota tracking across all collaborator regions
- Regional health monitoring for optimal placement
- Capacity planning for large-scale multi-institution launches

### NIH CUI/PHI Compliance (Personas 06-07)
**Benefit**: Compliance-aware quota management
- Ensure compliant regions have sufficient quota
- Health monitoring for compliance-critical regions
- Documented quota requests for audit trails

### Institutional IT (Persona 08)
**Benefit**: Institution-wide quota monitoring and management
- Centralized quota dashboard for all researchers
- Automated quota increase requests with institutional justification
- Cost-optimized capacity planning across departments

---

## Administrator Features

### Dashboard View

```bash
$ cws admin quota dashboard

╔══════════════════════════════════════════════════════════════════════╗
║             CloudWorkStation Quota Dashboard - us-west-2            ║
╠══════════════════════════════════════════════════════════════════════╣
║                                                                      ║
║  Overall Health: ✅ Healthy                                          ║
║  Active Researchers: 47                                              ║
║  Running Instances: 123                                              ║
║                                                                      ║
╠══════════════════════════════════════════════════════════════════════╣
║  Quota Status                                                        ║
║                                                                      ║
║  vCPU Quotas:                                                        ║
║    Standard: ████████████████░░░░  1,247/2,048 (61%) ✅              ║
║    GPU:      ███░░░░░░░░░░░░░░░░    32/256 (13%) ✅                 ║
║                                                                      ║
║  At-Risk Researchers:                                                ║
║    dr-johnson: 28/32 vCPUs (88%) ⚠️                                  ║
║    lab-team-3: 62/64 vCPUs (97%) 🚨                                  ║
║                                                                      ║
║  Pending Quota Requests:                                             ║
║    GPU vCPUs: 256 → 512 (requested 2 days ago) ⏳                    ║
║    Standard vCPUs: 2,048 → 4,096 (approved!) ✅                      ║
║                                                                      ║
╠══════════════════════════════════════════════════════════════════════╣
║  Regional Health                                                     ║
║                                                                      ║
║    us-west-2: ✅ Healthy                                             ║
║    us-east-1: ⚠️  Degraded (API_ISSUE, started 45m ago)              ║
║    eu-west-1: ✅ Healthy                                             ║
║                                                                      ║
╚══════════════════════════════════════════════════════════════════════╝

Press 'r' to refresh | Press 'q' to quit
```

---

## Implementation Timeline

| Component | Target | Estimate | Priority |
|-----------|--------|----------|----------|
| Quota Awareness | v0.6.0 Sprint 1 | 4-5 days | High |
| AZ Failover | v0.6.0 Sprint 1 | 3-4 days | High |
| Quota Increase Assistance | v0.6.0 Sprint 2 | 3-4 days | Medium |
| AWS Health Integration | v0.6.0 Sprint 2 | 3-4 days | Medium |
| Capacity Planning | v0.6.0 Sprint 3 | 4-5 days | Low |

**Total Effort**: 2-3 weeks
**Target Release**: v0.6.0 (Q2 2026)

---

## Related Documentation

- **Technical Debt Backlog**: [TECHNICAL_DEBT_BACKLOG.md](../archive/roadmap/TECHNICAL_DEBT_BACKLOG.md) (Item #2)
- **GitHub Issues**: [#57](https://github.com/scttfrdmn/cloudworkstation/issues/57), [#58](https://github.com/scttfrdmn/cloudworkstation/issues/58), [#59](https://github.com/scttfrdmn/cloudworkstation/issues/59), [#60](https://github.com/scttfrdmn/cloudworkstation/issues/60)
- **AWS IAM Permissions**: [AWS_IAM_PERMISSIONS.md](AWS_IAM_PERMISSIONS.md) - Required permissions for quota APIs
- **Administrator Guide**: [ADMINISTRATOR_GUIDE.md](ADMINISTRATOR_GUIDE.md) - General administration

---

## FAQ

**Q: Will this work with AWS Budgets?**
A: Yes! Quota management complements AWS Budgets. Quotas limit *what* you can launch, Budgets limit *how much you spend*. CloudWorkStation integrates both.

**Q: Can I request quota increases automatically?**
A: No - AWS requires human review for quota increases. CloudWorkStation will guide you through the manual request process with pre-filled forms.

**Q: What if I don't have Business Support for Health API?**
A: CloudWorkStation will gracefully degrade. Basic quota management and AZ failover will still work. Health monitoring requires Business/Enterprise Support.

**Q: How often are quotas checked?**
A: Quotas are checked before every launch and cached for 5 minutes. You can force refresh with `cws admin quota show --refresh`.

**Q: Can I set custom quota thresholds for alerts?**
A: Yes! Configure via `cws admin config set quota.warn-threshold 75` (default: 75%, 90%).

---

**Last Updated**: October 20, 2025
**Status**: Planned
**Next Review**: v0.6.0 Implementation Kickoff
