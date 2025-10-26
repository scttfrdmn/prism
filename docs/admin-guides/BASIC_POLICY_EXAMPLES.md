# Basic Policy Framework Examples

This document demonstrates the basic policy framework included in the open source version of Prism, showing practical examples for educational, research, and small organizational use cases.

## Educational Use Cases

### Computer Science Class Management

**Instructor Setup** (creating restricted invitations for students):

```bash
# Create invitation for CS101 students with specific templates only
prism profiles invitations create "CS101 Introduction to Python" \
  --type read_only \
  --valid-days 120 \
  --template-whitelist "python-basic,ubuntu-basic" \
  --max-instance-types "t2.micro,t2.small" \
  --max-hourly-cost 0.10

# Output:
Policy restrictions applied:
  - Allowed templates: [python-basic ubuntu-basic]
  - Max instance types: [t2.micro t2.small]
  - Max hourly cost: $0.10

Invitation Created Successfully
Name: CS101 Introduction to Python
Type: read_only
Expires: May 15, 2024 (in 120 days)

Share this invitation code with the recipient:
inv-AbCdEfGhIjKlMnOpQrStUvWxYz1234567890

They can accept it with:
prism profiles accept-invitation --encoded 'inv-AbCdEfGhIjKlMnOpQrStUvWxYz1234567890' --name 'CS101'
```

**Student Experience** (accepting and using restricted profile):

```bash
# Student accepts class invitation
prism profiles accept-invitation \
  --encoded 'inv-AbCdEfGhIjKlMnOpQrStUvWxYz1234567890' \
  --name 'CS101'

# Output:
Accepted invitation and created profile 'CS101'

# Student tries to launch an allowed template - SUCCESS
prism launch python-basic my-homework
# Output:
✓ Policy check: Template 'python-basic' is allowed
✓ Instance type 't2.micro' is within limits
✓ Cost $0.0116/hour is within $0.10 limit
Launching instance...

# Student tries to launch a restricted template - BLOCKED
prism launch python-ml advanced-project
# Output:
✗ Policy violation: Template 'python-ml' not in allowed list: [python-basic ubuntu-basic]
Available templates for this profile: python-basic, ubuntu-basic

# Student tries expensive instance - BLOCKED  
prism launch python-basic my-project --size XL
# Output:
✗ Policy violation: Instance type 'c5.4xlarge' not allowed. Maximum allowed: [t2.micro t2.small]
✗ Policy violation: Hourly cost $0.544 exceeds maximum allowed $0.10
```

### Advanced CS Course with More Flexibility

```bash
# CS401 Machine Learning course with GPU access
prism profiles invitations create "CS401 Machine Learning" \
  --type read_write \
  --valid-days 90 \
  --template-whitelist "python-ml,r-research,jupyter-gpu" \
  --max-instance-types "t3.medium,c5.large,p3.2xlarge" \
  --max-hourly-cost 3.06 \
  --max-daily-budget 25.00

# Policy allows GPU instances for ML coursework but with budget controls
```

## Research Lab Management

### PI Managing Lab Members

**Lab Director Setup**:

```bash
# Create invitation for graduate students with research flexibility
prism profiles invitations create "Bioinformatics Lab Access" \
  --type read_write \
  --valid-days 365 \
  --template-blacklist "windows-desktop,gaming-instance" \
  --forbidden-regions "eu-central-1,ap-southeast-1" \
  --max-daily-budget 50.00

# Allows most templates but blocks inappropriate ones
# Prevents launching in expensive regions
# Sets daily spending limit per student
```

**Graduate Student Experience**:

```bash
# Student can launch appropriate research templates
prism launch python-ml genomics-analysis  # ✓ Allowed
prism launch r-research statistical-modeling  # ✓ Allowed 
prism launch jupyter-gpu deep-learning  # ✓ Allowed

# But blocked from inappropriate templates
prism launch windows-desktop my-project
# ✗ Policy violation: Template 'windows-desktop' is blacklisted

# And prevented from expensive regions
prism launch python-ml project --region eu-central-1  
# ✗ Policy violation: Region 'eu-central-1' is forbidden
```

### Multi-Lab Collaboration

```bash
# Shared project between multiple institutions
prism profiles invitations create "Multi-Lab COVID Study" \
  --type read_write \
  --valid-days 180 \
  --template-whitelist "r-research,python-bio,jupyter-collaborative" \
  --max-instance-types "m5.large,m5.xlarge,r5.large,r5.xlarge" \
  --forbidden-regions "us-gov-west-1,us-gov-east-1" \
  --max-hourly-cost 0.50

# Ensures collaborators use consistent environments
# Prevents government cloud usage (compliance)
# Controls costs across institutions
```

## Small Organization Use Cases

### Startup Development Team

```bash
# Development team with cost controls
prism profiles invitations create "Dev Team Environment" \
  --type read_write \
  --valid-days 90 \
  --template-whitelist "ubuntu-dev,python-web,node-js,docker-compose" \
  --max-instance-types "t3.medium,c5.large,m5.large" \
  --max-daily-budget 20.00

# Standardizes development environments
# Prevents expensive instance launches
# Controls team cloud spending
```

### Consulting Firm Client Projects

```bash
# Client-specific environment with restrictions
prism profiles invitations create "ACME Corp Analytics Project" \
  --type read_only \
  --valid-days 60 \
  --template-whitelist "r-research,python-data-analysis" \
  --forbidden-regions "eu-west-1,ap-northeast-1" \
  --max-instance-types "t3.large,m5.large" \
  --max-hourly-cost 0.25

# Client can only access project-appropriate tools
# Regional compliance (data sovereignty)
# Cost control for client billing
```

## Budget Management Examples

### Grant-Funded Research

```bash
# NSF grant with specific budget limits
prism profiles invitations create "NSF Grant XYZ Computing" \
  --type read_write \
  --valid-days 1095 \  # 3 years
  --template-whitelist "python-scientific,r-hpc,matlab-compute" \
  --max-daily-budget 100.00 \
  --max-hourly-cost 5.00

# Long-term grant with appropriate daily limits
# Ensures spending aligns with NSF requirements
# Blocks inappropriate template usage
```

### Department Budget Controls

```bash
# Chemistry department semester budget
prism profiles invitations create "Chem Dept Fall 2024" \
  --type read_write \
  --valid-days 120 \
  --template-blacklist "gaming-instance,desktop-heavy,video-editing" \
  --max-instance-types "t3.medium,c5.large,m5.large,r5.large" \
  --max-daily-budget 75.00

# Prevents non-academic usage
# Reasonable instance size limits
# Department-wide spending control
```

## Policy Inheritance and Management

### Checking Profile Restrictions

```bash
# View current profile policy restrictions
prism profiles current

# Output:
Current profile: CS101 (Invitation)
Name: CS101 Introduction to Python
Region: us-west-2
Owner Account: prof-smith-account

Policy Restrictions:
  - Template whitelist: python-basic, ubuntu-basic
  - Max instance types: t2.micro, t2.small  
  - Max hourly cost: $0.10
```

### Template Validation

```bash
# Check which templates are available for current profile
prism templates list --profile-filtered

# Output:
Available templates for profile 'CS101':
  
✓ python-basic          Simple Python environment for learning
✓ ubuntu-basic          Basic Ubuntu server with development tools

Restricted templates (policy violations):
✗ python-ml            Template not in whitelist
✗ r-research          Template not in whitelist  
✗ jupyter-gpu         Template not in whitelist
```

### Policy Override (Admin Only)

```bash
# Profile owner can temporarily override restrictions (admin profiles only)
prism launch python-ml emergency-analysis --override-policy --confirm

# Requires confirmation and logs policy override for audit
```

## Advanced Policy Scenarios

### Hierarchical Course Structure

```bash
# Department-level base restrictions
prism profiles invitations create "Computer Science Department" \
  --type admin \
  --template-blacklist "windows-desktop,gaming-instance" \
  --forbidden-regions "us-gov-west-1" \
  --max-daily-budget 100.00

# Individual instructors inherit and add specific restrictions
# Students inherit all restrictions from instructor + department
```

### Seasonal Budget Adjustments

```bash
# Summer research program with higher limits
prism profiles invitations create "Summer REU Program" \
  --type read_write \
  --valid-days 90 \
  --template-whitelist "python-ml,r-research,jupyter-gpu,matlab-compute" \
  --max-instance-types "t3.xlarge,c5.2xlarge,p3.2xlarge" \
  --max-daily-budget 150.00  # Higher summer research budget

# Regular semester limits are lower for coursework
```

This basic policy framework provides immediate value for educational institutions, research labs, and small organizations without requiring the full enterprise policy engine. The restrictions are inherited through invitations and enforced at launch time, ensuring users stay within defined boundaries while maintaining the flexibility needed for their specific use cases.