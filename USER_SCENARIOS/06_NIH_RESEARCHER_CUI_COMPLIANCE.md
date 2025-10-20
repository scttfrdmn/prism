# User Scenario 6: NIH-Funded Researcher with CUI Compliance Requirements

**Persona**: Dr. Maria Rodriguez, Genomics Researcher
**Institution**: Major Research University
**Grant**: NIH R01 - $3.2M over 4 years
**Research Focus**: Cancer genomics using NIH controlled-access data repositories (dbGaP)
**Challenge**: Must comply with NIST 800-171 for controlled-access genomic data per NIH NOT-OD-24-157

---

## ⚠️ COMPLIANCE DISCLAIMER

**This scenario is for educational purposes only and does not constitute legal, regulatory, or compliance advice.**

Use of CloudWorkStation does not, by itself, ensure compliance with NIST 800-171, NIH data use agreements, or any other requirement. Your institution is solely responsible for:
- Determining compliance obligations for NIH controlled-access data
- Conducting NIST 800-171 assessments and attestations
- Implementing organizational policies beyond technical controls
- Consulting with your Research Security Office and Sponsored Projects Office

**Always obtain institutional approval before using CloudWorkStation for NIH dbGaP data or other CUI.**

See [COMPLIANCE_DISCLAIMER.md](../docs/admin-guides/COMPLIANCE_DISCLAIMER.md) for complete legal notice.

---

## Background Context

### The Compliance Landscape (2025)

**NIH NIST 800-171 Requirements (Effective January 25, 2025)**:
Per NIH Notice NOT-OD-24-157, **all NIH funding mechanisms supporting approved users accessing controlled-access human genomic data must now attest compliance with NIST SP 800-171**.[^1] This requirement applies to grants, cooperative agreements, contracts, and Other Transactions involving NIH controlled-access data repositories. Users must attest that their institution and any third-party cloud providers meet NIST 800-171 security standards.

**What is CUI in Biomedical Research?**
Controlled Unclassified Information (CUI) is Federal non-classified information the U.S. Government creates or possesses, or that a non-Federal entity (such as universities) receives, possesses, or creates for, or on behalf of, the U.S Government, that requires information and information system security controls.[^2]

**Common CUI in Federal Research**:
- Technical data under export control (ITAR/EAR) - dual-use research
- Pre-publication research data collected under federal contracts
- Proprietary information shared under federal research agreements
- Materials formulations with national security implications
- Computational models for defense-related applications

**NIST 800-171 Rev. 3 Requirements**:
Released May 2024, NIST SP 800-171 Rev. 3 provides **110 unique security requirements** that apply to university information systems that process, store, or transmit CUI.[^3] The August 2025 small business primer helps under-resourced organizations implement these requirements.[^4]

**Institutional Compliance Requirements**:
- **System Security Plan (SSP)**: Signed letter from authorized IT Director that an SSP is in place
- **CUI Training**: All project team members with CUI access must complete CUI Protections training
- **Annual Assessments**: Annual NIST 800-171 compliance assessment by Cyber Security, resulting in an attestation report signed by the CISO[^5]

**Consequences of Non-Compliance**:
Failure to comply may result in:
- Contract challenges or loss of the award
- Future ineligibility to be awarded government contracts
- Charges of fraud and criminal penalties for inaccurate compliance reporting[^6]

---

## Dr. Rodriguez's Research Context

### The Project

**Grant Title**: "Pan-Cancer Genomic Analysis of Treatment Response Using NIH dbGaP Data"

**Research Activities**:
- Access to controlled-access genomic data from NIH dbGaP (Database of Genotypes and Phenotypes)
- Analysis of tumor sequencing data from TCGA (The Cancer Genome Atlas) - 10,000+ patient cohort
- Integration with de-identified clinical outcomes (survival, treatment response, demographics)
- Machine learning prediction of treatment response biomarkers

**Why This is CUI** (Not PHI):
1. **NIH Controlled-Access Data**: Data from dbGaP requires NIST 800-171 compliance per NOT-OD-24-157 (effective January 25, 2025)
2. **De-identified Genomic Data**: HIPAA identifiers removed, but genomic data remains CUI
3. **Federal Data Use Limitations**: Data use agreement restricts analysis to approved research protocol
4. **No Re-identification**: Cannot attempt to re-identify individuals from genomic data

**CUI vs PHI Distinction**:
- This is **NOT PHI** because all 18 HIPAA identifiers have been removed by NIH
- However, genomic data is still **CUI** requiring NIST 800-171 compliance
- Re-identification is prohibited by data use agreement
- CloudWorkstation NIST 800-171 compliance is sufficient (HIPAA not required)

**The Notification**:
Dr. Rodriguez receives an email from NIH dbGaP Data Access Committee (DAC):

> "Your request for controlled-access genomic data from dbGaP (study phs000178) has been approved by the DAC. Per NIH Notice NOT-OD-24-157 (effective January 25, 2025), you must attest that your institution's IT systems comply with NIST SP 800-171 security standards. Please complete the institutional compliance attestation and provide details on your cloud computing environment (if applicable) within 30 days before data download authorization will be granted."

---

## The Compliance Challenge

### Dr. Rodriguez's Current Workflow (Pre-Compliance)

**Previous Setup**:
```bash
# Dr. Rodriguez's typical research computing (before CUI requirements)
ssh research-server.university.edu
cd /scratch/rodriguez-lab/battery-modeling/

# Run computational chemistry simulations
python run_dft_calculations.py --material LiNiCoAl --cycles 1000

# Train ML models on performance data
python train_battery_predictor.py --dataset proprietary_formulations.csv

# Collaborate with national lab
rsync -avz results/ collaborator@anl.gov:/shared/battery-research/
```

**Problems with This Approach for CUI**:
- ❌ Shared departmental server (not dedicated to CUI projects)
- ❌ No documented System Security Plan (SSP)
- ❌ Unknown compliance status (no NIST 800-171 assessment)
- ❌ Unencrypted data transfer to national lab collaborators
- ❌ No audit logging of CUI data access
- ❌ Mixed CUI and non-CUI research on same system

### University Research IT Requirements

**What Research IT Security Tells Dr. Rodriguez**:

1. **Dedicated CUI Environment Required**:
   - Cannot use shared departmental servers
   - Must use university-approved CUI computing platform
   - $15,000 annual fee for managed CUI infrastructure
   - OR: Self-managed system with annual compliance audit ($8,000)

2. **NIST 800-171 Controls to Address** (sample):
   - **AC.1.001**: Limit system access to authorized users (unique authentication)
   - **AU.2.041**: Create audit records for all CUI access
   - **CM.2.061**: Establish configuration baselines and document changes
   - **IA.2.078**: Use multifactor authentication for all CUI system access
   - **MP.2.120**: Protect and control CUI media during transport
   - **SC.2.179**: Use authenticated encryption for CUI at rest and in transit
   - **SI.2.214**: Monitor system security alerts and advisories

3. **Timeline Pressure**:
   - IRB approval pending (needs security approval first)
   - Post-doc starting in 2 months (needs compute access)
   - Collaborator data sharing agreements require compliance attestation
   - Grant spending period already started (time = money)

4. **Resource Constraints**:
   - Lab budget already allocated (adding $15K/year hurts)
   - No dedicated IT staff in research group
   - Dr. Chen's expertise is genomics, not cybersecurity
   - Can't afford delays while IT builds custom solution

---

## CloudWorkstation Solution

### Discovery & Setup

**Dr. Chen's Path to CloudWorkstation**:

1. **Research IT Recommends CloudWorkstation**:
   - University Research Security Office recently validated CloudWorkstation against NIST 800-171
   - Provides compliance documentation: [NIST_800_171_COMPLIANCE.md](../docs/admin-guides/NIST_800_171_COMPLIANCE.md)
   - Meets all 110 required controls in Rev. 3
   - Researchers can self-service with pre-approved configuration

2. **Institutional Compliance Profile**:
   University provides pre-configured compliance profile:
   ```bash
   # Dr. Chen installs CloudWorkstation
   brew install scttfrdmn/tap/cloudworkstation

   # Import university's CUI compliance profile
   cws profile import university-cui-profile.json
   # Profile includes:
   # - Required security group configurations
   # - Encrypted EBS/EFS settings (KMS key: university-managed)
   # - Audit logging requirements (CloudTrail integration)
   # - Network isolation (private subnets only)
   # - MFA enforcement
   ```

3. **Quick Start Guide from Research IT**:
   ```bash
   # Verify compliance profile is active
   cws profile list
   # Output shows: [university-cui] ✅ NIST 800-171 Rev. 3 Compliant

   # Launch CUI-compliant research environment
   cws launch python-ml lung-cancer-genomics \
     --profile university-cui \
     --project chen-lab-nih-r01 \
     --data-classification CUI \
     --require-mfa

   # Result: Instance launched with all 110 NIST 800-171 controls applied
   ```

---

## Day-to-Day Research with Compliance

### Scenario 1: Initial Data Analysis Setup

**Dr. Chen's Workflow**:

```bash
# Connect to CUI-compliant workstation
cws connect lung-cancer-genomics
# ↑ Prompts for MFA token (IA.2.078 - Multifactor Authentication)
# ↑ Logs connection attempt (AU.2.041 - Audit Records)

# Inside the workstation:
$ whoami
drschen

$ ls /mnt/efs/cui-data/
# Mounted EFS volume (university-managed, encrypted with KMS)
# MP.2.120 - Media Protection
# SC.2.179 - Encrypted Storage

$ df -h /mnt/efs/cui-data/
Filesystem                                Size  Used Avail Use% Mounted on
fs-0abc123.efs.us-east-1.amazonaws.com   8.0E     0  8.0E   0% /mnt/efs/cui-data

# Install research-specific tools
$ conda install -c bioconda bwa samtools gatk4 vcftools
# CM.2.061 - Configuration baseline managed via template

# Set up analysis pipeline
$ git clone https://github.com/chen-lab/genomics-pipeline.git
$ cd genomics-pipeline
$ python setup.py install
```

**What Happens Behind the Scenes** (Transparent to Dr. Chen):

1. **Access Control (AC)**:
   - SSH key-based authentication with MFA (AC.1.001, IA.2.078)
   - User assigned to project-specific security group (AC.1.002)
   - All network traffic through VPC with Security Groups (AC.2.015)

2. **Audit Logging (AU)**:
   - SSH connection logged with timestamp, source IP, user ID (AU.2.041)
   - All commands logged to university's SIEM (AU.2.042)
   - File access to CUI data logged (AU.2.044)

3. **Encryption (SC)**:
   - EBS volumes encrypted with university KMS key (SC.2.179)
   - EFS encrypted at rest (SC.2.179)
   - All data transfer uses TLS 1.3 (SC.2.183)

4. **Configuration Management (CM)**:
   - Workstation launched from approved template (CM.2.061)
   - Software installations logged (CM.2.065)
   - Template configuration matches university SSP (CM.2.062)

---

### Scenario 2: Collaborator Data Sharing (CUI Transfer)

**Challenge**: Dr. Chen needs to share analysis results with collaborator at Partner University.

**Compliant Workflow**:

```bash
# On CloudWorkstation:
$ cd /mnt/efs/cui-data/analysis-results/

# Create encrypted archive for transfer
# (MP.2.120 - Protect CUI during transport)
$ tar czf results-batch1.tar.gz *.vcf *.csv
$ openssl enc -aes-256-cbc -salt -in results-batch1.tar.gz \
    -out results-batch1.tar.gz.enc -pass file:$HOME/.cui-transfer-key

# Upload to university's approved secure file transfer
$ aws s3 cp results-batch1.tar.gz.enc \
    s3://university-cui-transfer/outbound/partner-university/ \
    --sse aws:kms \
    --sse-kms-key-id arn:aws:kms:us-east-1:123456789012:key/university-cui
# ↑ Uses university-managed KMS key for encryption
# ↑ S3 bucket has CUI access controls and logging

# Notify collaborator via secure email
$ echo "Results available in secure transfer portal" | \
    mail -s "[CUI] Batch 1 Results Ready" collaborator@partner.edu
```

**What This Achieves**:
- ✅ **MP.2.120** (Media Protection): Encrypted during transport
- ✅ **SC.2.179** (Encryption): AES-256-CBC + AWS KMS double encryption
- ✅ **AU.2.043** (Audit for Remote Activities): S3 transfer logged in CloudTrail
- ✅ **AC.2.015** (Route Through Managed Access Points): University-approved S3 bucket

**Traditional (Non-Compliant) Approach** Dr. Chen Would Have Used:
```bash
# ❌ This would FAIL compliance:
rsync -avz results/ collaborator@partner.edu:/data/shared/
# Problems:
# - No encryption in transit documentation
# - No audit trail of what was transferred
# - No access control verification
# - No CUI marking/handling
```

---

### Scenario 3: Adding New Team Member

**Dr. Chen Hires Post-Doc** (Dr. James Martinez):

**Compliant Onboarding**:

```bash
# 1. Dr. Chen requests access via university portal
# University Research Security verifies:
# - Background check completed (PS - Personnel Security)
# - CUI training certificate valid (AT - Awareness & Training)
# - NDA signed for NIH contract

# 2. Research IT provisions access
cws project member add chen-lab-nih-r01 \
  jmartinez@university.edu \
  --role member \
  --require-cui-training \
  --require-mfa

# 3. Dr. Martinez receives automated setup email:
# - CWS CLI installation instructions
# - University compliance profile download link
# - MFA enrollment instructions
# - CUI handling training (required within 7 days)

# 4. Dr. Martinez sets up access
brew install scttfrdmn/tap/cloudworkstation
cws profile import university-cui-profile.json
cws connect lung-cancer-genomics  # Prompts for MFA setup on first use

# 5. Access logged for compliance
# AU.2.042 - Account creation logged
# AC.1.001 - Unique user ID assigned (jmartinez)
# IA.1.076 - User identified uniquely in all logs
```

**Compliance Benefits**:
- ✅ **AC.1.001** (Limit Access): Only authorized users after training/verification
- ✅ **IA.1.076** (Unique Identification): Each user has unique CloudWorkstation identity
- ✅ **AT.2.008** (Security Awareness): CUI training required before access
- ✅ **AU.2.042** (Audit Record Generation): All account activities logged

---

### Scenario 4: Annual Compliance Assessment

**University CISO Requests Evidence** for Annual NIST 800-171 Assessment:

**CloudWorkstation Makes This Easy**:

```bash
# Generate compliance evidence package
cws compliance report \
  --framework "NIST 800-171 Rev 3" \
  --project chen-lab-nih-r01 \
  --output chen-lab-compliance-evidence.pdf

# Report automatically includes:
# ✅ Access Control Evidence:
#    - List of authorized users with MFA status
#    - Security group configurations
#    - Network access logs (90-day retention)
#
# ✅ Audit & Accountability Evidence:
#    - CloudTrail logs for all API calls
#    - SSH connection logs
#    - Data access logs (EFS/EBS)
#
# ✅ Configuration Management Evidence:
#    - Template configuration (version-controlled)
#    - Software inventory (conda list, apt list)
#    - Configuration change history
#
# ✅ Encryption Evidence:
#    - EBS encryption status (KMS key ID)
#    - EFS encryption status (KMS key ID)
#    - TLS configuration for data in transit
#
# ✅ Incident Response Evidence:
#    - Security alert history
#    - Incident response timeline (if any)
#
# ✅ System & Information Integrity Evidence:
#    - Vulnerability scan results (AWS Inspector)
#    - Patch management status
#    - Security baseline compliance
```

**Assessment Results**:
```
╔══════════════════════════════════════════════════════════════════╗
║        NIST 800-171 Rev. 3 Compliance Assessment Results        ║
║                    Chen Lab NIH R01 Project                      ║
╠══════════════════════════════════════════════════════════════════╣
║                                                                  ║
║  Control Families:                                               ║
║    ✅ Access Control (AC)              22/22 controls   100%     ║
║    ✅ Awareness & Training (AT)         4/4 controls    100%     ║
║    ✅ Audit & Accountability (AU)      12/12 controls   100%     ║
║    ✅ Configuration Mgmt (CM)          11/11 controls   100%     ║
║    ✅ Identification & Auth (IA)       11/11 controls   100%     ║
║    ✅ Incident Response (IR)            6/6 controls    100%     ║
║    ✅ Maintenance (MA)                  6/6 controls    100%     ║
║    ✅ Media Protection (MP)             8/8 controls    100%     ║
║    ✅ Personnel Security (PS)           2/2 controls    100%     ║
║    ✅ Physical Protection (PE)          6/6 controls    100%*    ║
║    ✅ Risk Assessment (RA)              5/5 controls    100%     ║
║    ✅ Security Assessment (CA)          9/9 controls    100%     ║
║    ✅ System & Comm Protection (SC)    23/23 controls   100%     ║
║    ✅ System & Info Integrity (SI)     16/16 controls   100%     ║
║                                                                  ║
║  Overall Compliance:          110/110 controls (100%)            ║
║  Assessment Date:             October 19, 2025                   ║
║  Assessor:                    University CISO                    ║
║  Next Assessment Due:         October 19, 2026                   ║
║                                                                  ║
║  *AWS data center physical security (inherited controls)        ║
╚══════════════════════════════════════════════════════════════════╝
```

**Dr. Chen's Reaction**:
"Wait, that's it? I thought this would take weeks of meetings and documentation. CloudWorkstation just... did all of that automatically?"

**University CISO's Reaction**:
"This is the first research project that passed 100% on first assessment. Usually we spend months remediating findings. Your use of CloudWorkstation with our compliance profile made this trivial."

---

## Cost & Time Comparison

### Traditional Approach (University-Managed CUI Infrastructure)

**Costs**:
- Setup: $8,000 (IT assessment, custom configuration)
- Annual: $15,000/year (managed infrastructure, compliance audits)
- Time: 8-12 weeks from request to usable environment

**Dr. Chen's Share**:
- Grant must cover $15,000/year × 5 years = $75,000
- Impacts ability to hire additional research staff
- Delays project timeline by 3 months

### CloudWorkstation Approach

**Costs**:
- Setup: $0 (self-service with university profile)
- Annual Compliance: $0 (automated evidence generation)
- AWS Compute: ~$200/month for medium-usage workstation
  - AWS charges for EC2, EBS, EFS (pay-as-you-go)
  - Hibernation reduces costs during non-use
  - Total: ~$2,400/year

**Dr. Chen's Share**:
- $2,400/year × 5 years = $12,000 vs. $75,000
- **Savings: $63,000** (can fund additional research assistant)
- **Time to Production**: 1 day (launch compliance-ready workstation immediately)

### Return on Investment

**Quantified Benefits**:

| Metric | Traditional | CloudWorkstation | Improvement |
|--------|-------------|------------------|-------------|
| Setup Cost | $8,000 | $0 | $8,000 saved |
| Annual Cost | $15,000 | $2,400 | $12,600 saved/year |
| 5-Year Total | $83,000 | $12,000 | $71,000 saved |
| Time to Production | 8-12 weeks | 1 day | ~10 weeks faster |
| Compliance Assessment | Manual (weeks) | Automated (minutes) | ~160 hours saved/year |
| Team Onboarding | IT ticket + 2 weeks | Self-service (1 hour) | ~80 hours saved/person |

**Qualitative Benefits**:
- ✅ **Researcher Control**: Dr. Chen manages her own environment
- ✅ **Reproducibility**: Template-based approach ensures consistent configurations
- ✅ **Collaboration**: Easy to share compliant workspaces with partners
- ✅ **Flexibility**: Can launch multiple workstations for different analysis phases
- ✅ **Peace of Mind**: Built-in compliance, not an afterthought

---

## Key Compliance Mappings

### How CloudWorkstation Addresses NIST 800-171 Rev. 3 for Dr. Chen

**Access Control (AC) - 22 Controls**:
- **AC.1.001** (Limit Access): Project membership + MFA enforcement
- **AC.1.002** (Limit Transactions): API-level access control via profiles
- **AC.2.007** (Least Privilege): Non-root user access, IAM role-based AWS access
- **AC.2.013** (Remote Access): SSH with key-based auth + MFA
- **AC.2.015** (Managed Access Points): VPC security groups, private subnets

**Audit & Accountability (AU) - 12 Controls**:
- **AU.2.041** (Audit Records): CloudTrail (AWS API), SSH logs, application logs
- **AU.2.042** (Audit Capability): Real-time logging to university SIEM
- **AU.2.043** (Remote Maintenance Audit): All SSH sessions logged
- **AU.2.044** (Review Audit Records): Compliance dashboard with query capability

**Configuration Management (CM) - 11 Controls**:
- **CM.2.061** (Baselines): Template-based launch ensures consistent baseline
- **CM.2.062** (Change Control): Template version control + audit logging
- **CM.2.064** (Least Functionality): Only required packages installed
- **CM.2.065** (Track Changes): Audit log of all configuration modifications

**Identification & Authentication (IA) - 11 Controls**:
- **IA.1.076** (Unique Users): Each researcher has unique CWS identity
- **IA.2.078** (MFA): Required for all CUI workstation access
- **IA.2.079** (Network MFA): MFA required for SSH connections
- **IA.2.081** (Replay Resistance): Cryptographic session tokens

**Media Protection (MP) - 8 Controls**:
- **MP.2.120** (Media Transport): Encrypted S3 transfer + KMS encryption
- **MP.2.122** (Media Disposal): Secure volume deletion (NIST SP 800-88 compliant)
- **MP.3.124** (Media Marking): CUI classification tags on resources

**System & Communications Protection (SC) - 23 Controls**:
- **SC.2.179** (Encrypted CUI): EBS/EFS encrypted with KMS (AES-256)
- **SC.2.181** (Session Authenticators): SSH key + time-limited session tokens
- **SC.2.183** (Network Encryption): TLS 1.3 for all data in transit

**System & Information Integrity (SI) - 16 Controls**:
- **SI.2.214** (Security Alerts): Integration with AWS GuardDuty, Security Hub
- **SI.2.216** (Monitor for Attacks): VPC Flow Logs, CloudWatch alarms
- **SI.2.217** (Unauthorized Use): Behavioral analysis via CloudWatch Insights

---

## Lessons Learned & Best Practices

### What Worked Well

1. **University Pre-Configuration**:
   - Research IT created "university-cui" profile with all institutional requirements
   - Researchers import and use without needing security expertise
   - Ensures consistent compliance across all NIH projects

2. **Template-Based Compliance**:
   - CloudWorkstation's template system maps directly to NIST CM.2.061 (Configuration Baselines)
   - Version-controlled templates provide audit trail of changes
   - Easy to update all workstations when university policy changes

3. **Automated Evidence Collection**:
   - Annual compliance assessments go from weeks to hours
   - No manual log collection or screenshot taking
   - Compliance reports generated on-demand for auditors

4. **Self-Service with Guardrails**:
   - Researchers launch compliant environments without IT tickets
   - University profile enforces security requirements automatically
   - Faster research, lower IT burden, maintained compliance

### Challenges & Solutions

**Challenge 1: Initial Learning Curve**
- **Problem**: Researchers unfamiliar with CLI tools
- **Solution**: University created "CUI Quick Start" video tutorial (15 min)
- **Result**: Most researchers productive within 1 hour

**Challenge 2: Collaborator Access Across Institutions**
- **Problem**: Partner universities have different CUI compliance approaches
- **Solution**: CloudWorkstation's portable profiles - share configuration across institutions
- **Result**: Collaborators adopt same compliant workflow

**Challenge 3: Legacy Data Migration**
- **Problem**: Existing data on non-compliant systems needs migration to CUI environment
- **Solution**: University IT provides secure migration service using CloudWorkstation's encrypted transfer
- **Result**: Data migrated with full audit trail, maintaining compliance

---

## Scaling Across University

### Institutional Adoption

**After Dr. Chen's Success**, University Research IT promotes CloudWorkstation:

**For Researchers**:
```bash
# Every NIH grant with CUI requirements now uses:
cws launch <research-template> <project-name> \
  --profile university-cui \
  --data-classification CUI

# University provides templates for common research types:
# - genomics-workstation (Dr. Chen's use case)
# - clinical-data-analysis (epidemiology studies)
# - imaging-analysis (neuroimaging with patient data)
# - synthetic-biology (export-controlled research)
```

**University-Wide Benefits**:
- **300+ NIH grants** now using CloudWorkstation for CUI compliance
- **$4.2M saved annually** vs. traditional managed infrastructure approach
- **95% reduction** in compliance assessment time
- **Zero compliance findings** in past 18 months since adoption
- **Faster grant proposal process** - compliance no longer a blocker

**Research Security Office Dashboard**:
```bash
cws admin compliance summary --institution
# Output:
# ✅ 312 CUI projects actively monitored
# ✅ 1,247 researchers with compliant workstations
# ✅ 100% pass rate on NIST 800-171 assessments
# ✅ 0 security incidents involving CUI in 18 months
# ✅ Average assessment time: 2.3 hours (vs. 40 hours previously)
```

---

## References & Citations

[^1]: NIH Office of the Director. (September 6, 2024). "NOT-OD-24-157: Implementation Update for Data Management and Access Practices Under the Genomic Data Sharing Policy." https://grants.nih.gov/grants/guide/notice-files/NOT-OD-24-157.html - Effective January 25, 2025, all NIH funding mechanisms supporting approved users accessing controlled-access human genomic data require attestation of NIST SP 800-171 compliance.

[^2]: University of Washington, Research Evaluation & Integrity. (2025). "Controlled Unclassified Information - CUI." https://research-eval.ui.oris.washington.edu/research/myresearch-lifecycle/setup/compliance-requirements-non-financial/information-privacy-and-security/controlled-unclassified-information-cui/ - Definition of CUI in federal research context.

[^3]: NIST Special Publication 800-171 Rev. 3. (May 2024). "Protecting Controlled Unclassified Information in Nonfederal Systems and Organizations." https://csrc.nist.gov/pubs/sp/800/171/r3/final - Current standard for CUI protection with 110 security requirements.

[^4]: NIST. (August 18, 2025). "Small Business Primer for NIST SP 800-171 Rev. 3." Supplement to help smaller organizations implement CUI security requirements.

[^5]: University of Michigan, Research Ethics & Compliance. (2025). "Controlled Unclassified Information (CUI)." https://research-compliance.umich.edu/research-information-security/controlled-unclassified-information-cui - Institutional CUI compliance requirements including SSP, training, and annual assessments.

[^6]: University of Connecticut, Office of the Vice President for Research. (2025). "Controlled Unclassified Information." https://ovpr.uconn.edu/services/research-security/controlled-unclassified-info/ - Consequences of non-compliance with CUI requirements.

---

## Related Documentation

- **[NIST 800-171 Compliance Guide](../docs/admin-guides/NIST_800_171_COMPLIANCE.md)** - Detailed control-by-control compliance mapping
- **[Security & Compliance Roadmap](../docs/admin-guides/SECURITY_COMPLIANCE_ROADMAP.md)** - Comprehensive compliance framework
- **[Compliance Matrix](../docs/admin-guides/COMPLIANCE_MATRIX.md)** - Quick reference for all frameworks

---

**Scenario Created**: October 19, 2025
**NIST 800-171 Version**: Revision 3 (May 2024)
**Based On**: Real NIH research compliance requirements and university CUI programs
