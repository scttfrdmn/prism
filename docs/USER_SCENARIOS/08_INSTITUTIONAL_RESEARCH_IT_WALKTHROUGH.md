# Persona 6: Institutional Research IT / Research Enablement

**Last Updated**: October 2025
**Version**: 0.5.5

---

## 👤 Persona Overview

**Name**: Dr. Maria Chen
**Role**: Director of Research Computing Services
**Institution**: State University Research Computing Center
**Team**: 8 FTEs supporting 5,000 researchers across 12 colleges

### Background

Maria leads the central research computing team at a large public university. Her team provides computing infrastructure, consulting, and training for faculty, postdocs, and graduate students across diverse research domains—from genomics to machine learning to computational social science. The team manages on-premise HPC clusters, provides cloud computing support, and helps researchers navigate the complex landscape of research computing resources.

### Key Responsibilities

- **Infrastructure Planning**: Strategic planning for institutional research computing investments
- **Resource Management**: Allocation and cost management for shared computing resources
- **Policy & Compliance**: Enforce data security, export control, and grant compliance requirements
- **User Support**: Training, consulting, and troubleshooting for 5,000+ researchers
- **Vendor Relationships**: Negotiate institutional agreements with cloud providers
- **Budget Oversight**: $2M annual operating budget plus grant-funded resources

### Current Pain Points

1. **Fragmented Cloud Usage**: Researchers using personal AWS accounts create:
   - Security risks (no institutional oversight)
   - Cost inefficiency (no bulk discounts)
   - Support burden (team can't help with personal accounts)
   - Compliance issues (data leaving institutional control)

2. **Onboarding Complexity**: New researchers face weeks of setup:
   - Account provisioning across multiple systems
   - Training on institutional policies
   - Software installation and configuration
   - HPC cluster access and job submission

3. **Template Proliferation**: Each lab maintains their own software stacks:
   - Duplicated effort across research groups
   - Inconsistent quality and security patching
   - Knowledge locked in individual labs
   - No institutional best practices

4. **Cost Visibility Gaps**: Limited ability to track and optimize spending:
   - Individual projects can't see their costs
   - No way to set project-specific budgets
   - Surprise bills from unmonitored resources
   - Difficulty justifying investments to administration

5. **Compliance Burden**: Manual processes for security and compliance:
   - Per-project security reviews
   - Manual tracking of data classifications
   - Periodic access audits
   - Export control verification

---

## 🎯 CloudWorkstation Solution for Research IT

### Strategic Value Proposition

CloudWorkstation positions Research IT as a **value-adding service provider** rather than a gatekeeper, enabling researchers while maintaining institutional oversight.

**For Researchers**:
- Self-service access to approved environments
- Faster time to research productivity
- Pre-configured, institutionally-supported tools

**For Research IT**:
- Centralized visibility and control
- Automated policy enforcement
- Reduced support burden through standardization
- Clear cost tracking and chargeback capabilities

---

## 📋 Institutional Deployment Workflow

### Phase 1: Pilot Program (Weeks 1-4)

**Week 1: Internal Setup**
```bash
# Maria's team installs CloudWorkstation for institutional deployment
brew tap scttfrdmn/cloudworkstation
brew install cloudworkstation

# Configure institutional AWS account
cws profile create university-research \
  --aws-profile state-university-research \
  --region us-west-2 \
  --description "State University Research Computing"

# Set up institutional template repository
cws admin policy set template-registry \
  --registry https://github.com/state-university/cws-templates \
  --require-approval true
```

**Maria's Thought Process**:
> "We need to maintain control over what computing environments researchers can use. Our institutional template repository lets us curate approved software stacks that meet security requirements, include properly licensed software, and follow best practices. Researchers get self-service convenience, we maintain oversight."

**Week 2: Template Curation**
```bash
# Maria's team creates institutional templates
# templates/bioinformatics-approved.yml
name: "Bioinformatics Research (Approved)"
display_name: "State University - Bioinformatics"
category: "bioinformatics"
approved_by: "research-computing@university.edu"
security_reviewed: "2025-10-01"
description: |
  Institutional bioinformatics environment with licensed tools
  Security-reviewed, HIPAA-compliant configuration

base_image:
  os: "ubuntu"
  version: "22.04"

packages:
  system:
    - blast
    - bowtie2
    - samtools

  licensed:
    - name: "matlab-runtime"
      license_server: "license.university.edu:27000"
    - name: "geneious"
      floating_license: true

security:
  data_classification: ["public", "internal", "restricted"]
  export_control_cleared: true
  hipaa_compliant: true

compliance:
  audit_logging: true
  encryption_at_rest: true
  encryption_in_transit: true
```

**Security Integration**:
```bash
# Enforce institutional security policies
cws admin policy set security-baseline \
  --require-encryption true \
  --require-mfa true \
  --allowed-regions us-west-1,us-west-2 \
  --prohibited-regions cn-*,us-gov-* \
  --require-tagging true

# Set up cost controls
cws admin policy set cost-limits \
  --default-project-budget 500 \
  --max-instance-cost 5.00 \
  --require-budget-approval-over 1000
```

**Week 3: Pilot Group Selection**
```yaml
# pilot-program.yml
pilot_groups:
  - name: "Genomics Lab (Dr. Sarah Johnson)"
    size: 12 researchers
    rationale: "Heavy cloud users, familiar with AWS, vocal advocates"

  - name: "ML Research Group (Prof. David Lee)"
    size: 8 researchers
    rationale: "GPU workloads, cost-sensitive, good documentation habits"

  - name: "Social Science Data Lab (Prof. Emily Rodriguez)"
    size: 6 researchers
    rationale: "New to cloud, need simplicity, diverse skill levels"

success_criteria:
  - "80% of pilot users prefer CloudWorkstation over manual AWS"
  - "50% reduction in support tickets compared to manual cloud setup"
  - "Template reuse across at least 2 research groups"
  - "Zero security incidents during pilot"
  - "Cost tracking accurate within 5%"
```

**Week 4: Pilot Kickoff**
```bash
# Create pilot project with budget
cws project create genomics-pilot \
  --budget 2000 \
  --pi "sarah.johnson@university.edu" \
  --department "Biology" \
  --grant-number "NIH-R01-12345"

# Add pilot users
cws project member add genomics-pilot researcher1@university.edu --role member
cws project member add genomics-pilot researcher2@university.edu --role member

# Pilot users can now self-serve
# (as pilot user)
cws launch bioinformatics-approved my-genomics-analysis \
  --project genomics-pilot

# Maria's team monitors usage
cws admin usage report --project genomics-pilot
cws admin cost report --start-date 2025-10-01
```

---

### Phase 2: Institutional Rollout (Months 2-4)

**Month 2: Expand to 10 Research Groups**

**Template Governance**:
```bash
# Establish template approval workflow
# .github/workflows/template-approval.yml
name: Template Review
on:
  pull_request:
    paths:
      - 'templates/**'

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Security Scan
        run: cws templates validate --security-scan
      - name: License Check
        run: cws templates check-licenses
      - name: Cost Estimate
        run: cws templates estimate-cost
      # → Requires approval from research-computing team
```

**Maria's Team Workflow**:
> "Researchers can propose new templates via pull requests. Our automated checks verify security, licenses, and cost implications. A team member reviews and approves. This scales our expertise across the institution while maintaining quality control."

**Chargeback Model**:
```bash
# Configure departmental chargeback
cws admin billing configure \
  --model departmental-chargeback \
  --billing-contact finance@university.edu \
  --monthly-invoicing

# Each project ties to a department/grant
cws project create physics-simulation \
  --department "Physics" \
  --chartstring "12-3456-789-0000" \
  --budget 5000

# Monthly reports for finance
cws admin billing report --month 2025-10 \
  --format csv \
  --group-by department \
  --output /shared/finance/cloudworkstation-2025-10.csv
```

**Month 3: Training & Documentation**

**Maria creates institutional documentation**:
```markdown
# State University CloudWorkstation Guide

## For Researchers

### Quick Start
1. Visit: https://cloudworkstation.research.university.edu
2. Log in with your university credentials
3. Choose an approved template for your research domain
4. Launch your environment - ready in under 60 seconds

### Available Templates
- **Bioinformatics**: BLAST, bowtie2, samtools, IGV
- **Machine Learning**: PyTorch, TensorFlow, Jupyter, GPU-enabled
- **Social Science**: R, RStudio, SPSS alternatives, survey tools
- **General Computing**: Python, Julia, MATLAB runtime

### Cost Management
- Your PI sets project budgets
- You can see your project's spending in real-time
- Automated hibernation saves 90% on idle resources
- Email alerts at 50%, 75%, 90% of budget

## For PIs

### Creating a Project
Contact research-computing@university.edu with:
- Project title and description
- Grant or chartstring for billing
- Initial budget request
- List of researchers who need access

### Monitoring Costs
```bash
cws project costs genomics-research --detailed
cws project members list genomics-research
```

## For Departmental Administrators

### Monthly Billing Reports
Available at: https://cloudworkstation.research.university.edu/billing
- CSV export for integration with financial systems
- Breakdown by project, researcher, resource type
- Cost allocation by chartstring
```

**Month 4: Policy Refinement**

**Based on pilot feedback, Maria implements governance policies**:

```yaml
# institutional-policies.yml
templates:
  approval_required: true
  security_scan: mandatory
  license_compliance: true

access:
  authentication: "university-sso"
  mfa_required: true
  session_timeout: "8 hours"

data_classification:
  public:
    - storage: "any"
    - export: "allowed"

  internal:
    - storage: "encrypted-ebs,efs"
    - export: "approval-required"

  restricted:
    - storage: "encrypted-ebs-only"
    - export: "prohibited"
    - audit_logging: "comprehensive"

  hipaa:
    - storage: "hipaa-compliant-only"
    - regions: ["us-west-2"]
    - encryption: "mandatory"
    - audit_logging: "comprehensive"
    - access_control: "role-based"

cost_controls:
  default_project_budget: 500
  budget_alerts: [50, 75, 90]
  auto_hibernate_at_budget_limit: true
  require_pi_approval_over: 2000

compliance:
  export_control_check: true
  prohibited_regions: ["cn-*", "ru-*"]
  require_project_metadata: true
  quarterly_access_audit: true
```

---

## 🎓 Use Cases Across Institution

### Use Case 1: Grant-Funded Research Project

**Scenario**: Prof. Johnson receives NIH R01 grant for genomics research

```bash
# Maria's team creates project from grant award
cws project create johnson-nih-r01-genomics \
  --pi "sarah.johnson@university.edu" \
  --grant-number "5R01HG012345" \
  --budget 15000 \
  --duration "2025-01-01 to 2029-12-31" \
  --data-classification restricted \
  --compliance hipaa

# Add researchers
cws project member add johnson-nih-r01-genomics postdoc1@university.edu --role admin
cws project member add johnson-nih-r01-genomics gradstudent1@university.edu --role member
cws project member add johnson-nih-r01-genomics gradstudent2@university.edu --role member

# Set compliance requirements
cws admin policy apply johnson-nih-r01-genomics \
  --template hipaa-compliant \
  --audit-logging comprehensive \
  --require-training hipaa-basics
```

**Researcher Experience**:
```bash
# Postdoc launches environment (compliant by default)
cws launch bioinformatics-hipaa my-patient-genomics \
  --project johnson-nih-r01-genomics

# Instance automatically has:
# ✅ Encrypted storage
# ✅ HIPAA-compliant configuration
# ✅ Audit logging enabled
# ✅ Restricted to approved regions
# ✅ Costs tracked to grant
```

### Use Case 2: Course Computing Environment

**Scenario**: CS 401 - Introduction to Machine Learning (150 students)

```bash
# Instructor requests course support
cws project create cs401-fall2025 \
  --instructor "david.lee@university.edu" \
  --course-number "CS-401" \
  --semester "Fall 2025" \
  --budget 3000 \
  --max-members 160

# Batch add students from roster
cws project member batch-add cs401-fall2025 \
  --csv course-roster.csv \
  --role member \
  --auto-provision

# Create assignment-specific template
cws templates create cs401-assignment1 \
  --base python-ml \
  --add-package scikit-learn==1.3.0 \
  --add-package matplotlib==3.8.0 \
  --include-notebook assignment1.ipynb \
  --set-max-cost 0.50

# Students launch identical environments
# (as student)
cws launch cs401-assignment1 homework1 --project cs401-fall2025
```

**Instructor Monitoring**:
```bash
# Track student progress
cws admin report cs401-fall2025 \
  --metric instances-launched \
  --metric compute-hours \
  --metric costs \
  --format dashboard
```

### Use Case 3: Multi-Institutional Collaboration

**Scenario**: NSF-funded collaboration with 3 universities

```bash
# Each institution maintains own CloudWorkstation
# But shares templates via template marketplace

# State University publishes template
cws templates publish bioinformatics-pipeline \
  --registry community \
  --license MIT \
  --tested-regions us-west-1,us-west-2,us-east-1

# Partner University discovers and installs
cws templates install bioinformatics-pipeline \
  --from community \
  --author state-university

# Data sharing via EFS cross-account access
cws storage create collaboration-data \
  --type efs \
  --share-with arn:aws:iam::partner-account:root \
  --read-only

# Each institution tracks costs separately
# But uses consistent, validated research environment
```

---

## 📊 Institutional Benefits & Metrics

### Cost Efficiency

**Before CloudWorkstation**:
- Researchers using 47 different personal AWS accounts
- No bulk discounting
- Average monthly spend: $48,000
- 23% waste from forgotten resources
- No cost visibility by project

**After CloudWorkstation (6 months)**:
- Consolidated to institutional AWS Organizations account
- 15% EDU bulk discount negotiated
- Average monthly spend: $39,000 (18% reduction)
- 4% waste (improved monitoring and auto-hibernation)
- Full cost visibility with project-level tracking
- **Annual savings: $108,000**

### Support Efficiency

**Before CloudWorkstation**:
- 340 support tickets/month for cloud computing
- Average resolution time: 4.2 hours
- 62% of tickets were "how do I set up X"

**After CloudWorkstation (6 months)**:
- 115 support tickets/month (66% reduction)
- Average resolution time: 1.8 hours
- 82% of new researchers self-serve successfully
- Team capacity freed for advanced consulting

### Compliance & Security

**Automated Compliance**:
```bash
# Quarterly audit report
cws admin audit report --quarter Q3-2025

Audit Summary:
✅ 342 projects reviewed
✅ 1,847 instances scanned
✅ 0 security violations
✅ 0 export control issues
✅ 100% encryption compliance
✅ 0 unauthorized data transfers

Risk Findings:
⚠️  12 projects approaching budget limits (notifications sent)
⚠️  3 instances running >30 days (hibernation recommended)
✅ All findings within acceptable parameters
```

### Template Reuse

**Community Building**:
```
Institutional Template Library (6 months):
├── 23 approved templates
├── 847 total launches
├── 14 templates contributed by researchers
├── 9 templates shared with partner institutions
└── 3.2 average launches per template per week

Most Popular:
1. Python ML (243 launches)
2. Bioinformatics Pipeline (156 launches)
3. R Data Science (134 launches)
4. GPU Deep Learning (98 launches)
5. Social Science Stats (87 launches)
```

---

## 🔐 Institutional Policy Framework

### Security & Compliance Policies

```yaml
# File: /institutional/security-policies.yml

authentication:
  provider: "university-sso"
  mfa_required: true
  session_timeout: 28800  # 8 hours
  idle_timeout: 3600      # 1 hour

authorization:
  default_role: "member"
  pi_approval_required_for: ["admin", "owner"]
  automatic_offboarding: "30-days-after-separation"

network_security:
  allowed_regions:
    - "us-west-1"
    - "us-west-2"
    - "us-east-1"
    - "us-east-2"

  prohibited_regions:
    - "cn-*"      # Export control
    - "ru-*"      # Sanctions compliance
    - "us-gov-*"  # Government-only

  vpc_configuration:
    use_institutional_vpc: true
    require_vpn_for_admin_access: true

data_classification:
  public:
    encryption: "recommended"
    storage_locations: "any-allowed-region"
    sharing: "allowed"

  internal:
    encryption: "required"
    storage_locations: "us-only"
    sharing: "within-institution"

  restricted:
    encryption: "required-fips-140-2"
    storage_locations: "us-west-2-only"
    sharing: "explicit-approval"
    audit_logging: "comprehensive"

  hipaa:
    encryption: "required-fips-140-2"
    storage_locations: "us-west-2-only"
    sharing: "prohibited"
    audit_logging: "comprehensive"
    access_controls: "role-based"
    minimum_training: "hipaa-basics"

export_control:
  enabled: true
  check_user_clearance: true
  prohibited_countries: ["CN", "RU", "IR", "KP", "SY"]
  technical_data_review: "automatic"
```

### Cost Management Policies

```yaml
# File: /institutional/cost-policies.yml

budgets:
  default_project_budget: 500
  require_pi_approval_over: 2000
  require_admin_approval_over: 10000

  alerts:
    thresholds: [50, 75, 90, 100]
    recipients: ["pi", "project-admins", "research-computing"]

  actions:
    at_75_percent: "send-warning"
    at_90_percent: "send-urgent-warning"
    at_100_percent: "auto-hibernate-instances"

instance_limits:
  max_hourly_cost_without_approval: 5.00
  max_simultaneous_instances: 10
  max_gpu_instances: 2

cost_optimization:
  auto_hibernate_after_idle: "60-minutes"
  recommend_rightsizing: true
  prefer_spot_instances: "when-appropriate"

chargeback:
  model: "departmental"
  billing_cycle: "monthly"
  require_chartstring: true
  finance_integration: "banner-finance"
```

### Template Governance

```yaml
# File: /institutional/template-policies.yml

template_approval:
  required: true
  reviewers: ["research-computing-team"]
  review_criteria:
    - security_scan_passed
    - license_compliance_verified
    - cost_estimate_acceptable
    - documentation_complete

template_sources:
  institutional_repository:
    url: "https://github.com/state-university/cws-templates"
    trust_level: "trusted"
    auto_approve: false

  community_repository:
    url: "https://github.com/cloudworkstation/community-templates"
    trust_level: "review-required"
    security_scan: "mandatory"

  researcher_contributions:
    enabled: true
    process: "pull-request"
    review_required: true

template_maintenance:
  security_patch_sla: "7-days"
  quarterly_review: true
  deprecation_notice: "90-days"
  automated_testing: true
```

---

## 🎯 Success Criteria

### Adoption Metrics (Year 1)

- ✅ **50% of cloud-using researchers** migrated to CloudWorkstation
- ✅ **15 departments** actively using the platform
- ✅ **30+ approved templates** in institutional repository
- ✅ **20% cost reduction** vs unmanaged cloud usage
- ✅ **60% reduction** in cloud-related support tickets

### Quality Metrics

- ✅ **99.5% uptime** for CloudWorkstation daemon
- ✅ **<2% security incidents** (vs 8% industry average)
- ✅ **95% user satisfaction** rating
- ✅ **80% self-service** success rate
- ✅ **100% compliance** with institutional policies

### Strategic Goals

- ✅ **Become preferred platform** for grant-funded cloud computing
- ✅ **Enable new research** previously blocked by cloud complexity
- ✅ **Reduce researcher friction** while maintaining oversight
- ✅ **Build institutional expertise** in cloud research computing
- ✅ **Establish best practices** shared across peer institutions

---

## 🚀 Future Roadmap: Research IT Enablement

### Phase 3: Advanced Governance (Months 6-12)

**Automated Compliance Reporting**:
```bash
# NSF grant compliance report
cws admin compliance report \
  --grant "NSF-2112345" \
  --period "2025-Q3" \
  --format nsf-standard

# Generated report includes:
# - All compute resources used
# - Cost allocation by task
# - Data storage locations
# - Security compliance status
# - Export control verification
```

**Template Marketplace Curation**:
```bash
# Institutional template ratings and quality metrics
cws templates leaderboard --institution state-university

Top Templates (by usage):
1. ⭐⭐⭐⭐⭐ Python ML (243 launches, 4.8/5.0)
2. ⭐⭐⭐⭐⭐ Bioinformatics (156 launches, 4.9/5.0)
3. ⭐⭐⭐⭐☆ R Data Science (134 launches, 4.2/5.0)

# Contribute to community
cws templates publish bioinformatics-pipeline \
  --registry community \
  --quality-assured \
  --peer-reviewed
```

### Phase 4: Multi-Cloud Strategy (Year 2)

**Azure/GCP Support**:
```yaml
# Enable multi-cloud for researchers with specific needs
cws admin cloud-providers enable azure \
  --institutional-account state-university-azure \
  --cost-tracking-integration banner-finance

# Researchers can now choose cloud provider
cws launch python-ml my-project \
  --cloud azure \
  --reason "Microsoft-specific ML services required"
```

**Hybrid HPC Integration**:
```bash
# Connect CloudWorkstation to on-prem HPC
cws cluster connect university-hpc \
  --scheduler slurm \
  --hybrid-workflows enabled

# Researchers can burst to cloud when cluster is full
cws job submit analysis.sh \
  --prefer on-prem \
  --burst-to-cloud-when-busy
```

---

## 💡 Key Insights for Research IT Leaders

### 1. Start with Pilot Programs
Don't attempt institution-wide rollout immediately. Select enthusiastic early adopters who can become champions.

### 2. Template Curation is Strategic
Invest time in creating high-quality, approved templates. They become institutional knowledge assets.

### 3. Cost Transparency Builds Trust
Researchers appreciate clear project budgets and real-time cost visibility. It reduces friction and builds responsible usage patterns.

### 4. Automate Policy Enforcement
Manual compliance reviews don't scale. Build policies into templates and automate checks.

### 5. Leverage Researchers' Expertise
Allow researcher-contributed templates with appropriate review. It builds community and reduces your team's burden.

### 6. Measure and Communicate Value
Regular reports on cost savings, support reduction, and researcher satisfaction justify continued investment and demonstrate value to administration.

---

## 🤝 Community of Practice

### Peer Institution Collaboration

**Share experiences and templates**:
- Monthly virtual meetups with peer Research IT teams
- Shared template repositories for common research domains
- Collaborative policy development
- Benchmark metrics across institutions

**Join**: research-it-cloudworkstation@groups.university.edu

---

**CloudWorkstation for Research IT**: From infrastructure management burden to strategic research enablement. Empower researchers while maintaining institutional oversight, security, and cost control.
