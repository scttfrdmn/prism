# Multi-Persona Screenshot Integration Plan

**Created**: October 27, 2025
**Purpose**: Systematic plan to extend GUI screenshot integration across all 8 persona walkthroughs

---

## 🎯 Strategy

### Core Principle: Contextual Reuse
- **Same GUI screenshots** are reused across personas
- **Different contextualization** for each persona's workflow and priorities
- **Persona-specific integration text** explains "why this matters to YOU"

### Available Screenshots (5 base GUI screenshots)
All captured from Cloudscape GUI interface:

1. **gui-settings-profiles.png** (166KB) - AWS profile configuration
2. **gui-quick-start-wizard.png** (98KB) - Template selection wizard
3. **gui-storage-management.png** (216KB) - EFS/EBS storage interface
4. **gui-workspaces-list.png** (140KB) - Workspace management table
5. **gui-projects-dashboard.png** (180KB) - Project & budget management

---

## 📋 Per-Persona Integration Plan

### ✅ 01 - Solo Researcher (COMPLETE)
**Status**: 5/5 screenshots integrated
**Commit**: `2832b4a37`

**Context**: Individual researcher managing personal workspaces and costs

| Screenshot | Location | Context |
|------------|----------|---------|
| Settings | Initial Setup | Validating AWS credentials before first launch |
| Quick Start | After CLI wizard | Visual alternative for 30-second first workspace |
| Storage | After hibernation | Persistent datasets across workspace terminations |
| Workspaces | Daily Work | Managing personal workspaces and cost tracking |
| Projects | Before Pain Points | Future budget management for grant-funded work |

---

### 🔄 02 - Lab Environment (TODO)
**Status**: 0/5 screenshots
**Persona**: Dr. Martinez managing 8 PhD students with shared resources

**Integration Strategy**: Emphasize **team collaboration** and **shared infrastructure**

| Screenshot | Suggested Location | Context Angle |
|------------|-------------------|---------------|
| **Settings** | Initial Setup (~line 30) | Lab PI configuring institutional AWS account access |
| **Quick Start** | Team Onboarding (~line 80) | Training new lab members with visual wizard |
| **Storage** | Shared Data Management (~line 120) | EFS shared storage for collaborative datasets (TB-scale) |
| **Workspaces** | Lab Dashboard (~line 180) | Managing 8+ concurrent student workspaces with lab-wide visibility |
| **Projects** | Budget Management (~line 250) | Grant-funded projects with per-student cost allocation |

**Key Differences**:
- Settings: "Dr. Martinez validates institutional SSO credentials"
- Storage: "Shared `/data` EFS mount for entire lab (5TB)"
- Workspaces: "8 concurrent workspaces visible, sorted by student name"
- Projects: "NIH R01 grant budget tracking across 8 team members"

---

### 🔄 03 - University Class (TODO)
**Status**: 0/5 screenshots
**Persona**: Prof. Johnson teaching CS 473 with 120 students

**Integration Strategy**: Emphasize **bulk operations** and **scalability**

| Screenshot | Suggested Location | Context Angle |
|------------|-------------------|---------------|
| **Settings** | Course Setup (~line 25) | University IT pre-configured institutional AWS account |
| **Quick Start** | Student Onboarding (~line 60) | 120 students launching identical "Data Science 101" workspace |
| **Storage** | Assignment Submission (~line 110) | Individual EBS volumes for student work isolation |
| **Workspaces** | Class Dashboard (~line 160) | 120-workspace view with filters (by section, assignment status) |
| **Projects** | Course Budget (~line 220) | Department budget tracking ($2000/semester allocation) |

**Key Differences**:
- Settings: "University IT validates .edu AWS account"
- Storage: "120 individual EBS volumes (50GB each) for assignment work"
- Workspaces: "Bulk operations: Stop all after class, Hibernate overnight"
- Projects: "CS Department semester budget with automated alerts at 75%"

---

### 🔄 04 - Conference Workshop (TODO)
**Status**: 0/5 screenshots
**Persona**: Dr. Kim running 3-hour ISMB workshop for 50 participants

**Integration Strategy**: Emphasize **rapid provisioning** and **time-limited usage**

| Screenshot | Suggested Location | Context Angle |
|------------|-------------------|---------------|
| **Settings** | Pre-Workshop Setup (~line 20) | Testing AWS credentials 1 week before conference |
| **Quick Start** | Workshop Launch (~line 50) | 50 participants launching identical bioinformatics workspace in 5 minutes |
| **Storage** | Workshop Materials (~line 90) | Read-only EFS share with tutorial datasets (pre-loaded) |
| **Workspaces** | During Workshop (~line 130) | Live monitoring of 50 workspaces during tutorial |
| **Projects** | Cost Management (~line 180) | Conference workshop budget (fixed $200 allocation) |

**Key Differences**:
- Settings: "Dr. Kim validates AWS credits provided by conference"
- Storage: "Read-only shared EFS with tutorial datasets (no writes needed)"
- Workspaces: "50 identical workspaces, all terminated after 4-hour window"
- Projects: "Workshop budget hard stop at $200 to prevent overruns"

---

### 🔄 05 - Cross-Institutional Collaboration (TODO)
**Status**: 0/5 screenshots
**Persona**: Dr. Thompson coordinating 4 universities on NIH consortium

**Integration Strategy**: Emphasize **multi-tenant isolation** and **institutional SSO**

| Screenshot | Suggested Location | Context Angle |
|------------|-------------------|---------------|
| **Settings** | Multi-Site Setup (~line 35) | Each institution validates their own AWS profile |
| **Quick Start** | Researcher Onboarding (~line 90) | Standardized workspace across all 4 institutions |
| **Storage** | Shared Data Lake (~line 140) | Cross-institutional EFS share (10TB genomics data) |
| **Workspaces** | Consortium Dashboard (~line 200) | Workspaces tagged by institution (MIT, Stanford, UCSF, JHU) |
| **Projects** | Grant Management (~line 280) | NIH U01 budget tracking with per-institution subawards |

**Key Differences**:
- Settings: "Each institution configures their institutional SSO (OAuth)"
- Storage: "10TB shared EFS visible across all 4 institutions"
- Workspaces: "Filter by institution tag, sort by researcher affiliation"
- Projects: "Primary institution (MIT) tracks 4 subaward budgets independently"

---

### 🔄 06 - NIH Researcher (CUI Compliance) (TODO)
**Status**: 0/5 screenshots
**Persona**: Dr. Patel managing Controlled Unclassified Information (CUI)

**Integration Strategy**: Emphasize **security compliance** and **access controls**

| Screenshot | Suggested Location | Context Angle |
|------------|-------------------|---------------|
| **Settings** | Compliance Validation (~line 30) | Validating AWS GovCloud or NIH-compliant region |
| **Quick Start** | Secure Workspace Launch (~line 70) | CUI-compliant template with required security controls |
| **Storage** | Encrypted Storage (~line 110) | FIPS 140-2 encrypted EBS volumes for CUI data |
| **Workspaces** | Compliance Dashboard (~line 170) | Workspaces showing encryption status and compliance badges |
| **Projects** | Audit Trail (~line 240) | NIH grant with automated compliance reporting |

**Key Differences**:
- Settings: "Validate AWS GovCloud or us-gov-west-1 region compliance"
- Storage: "FIPS 140-2 Level 2 encrypted EBS, no EFS (compliance requirement)"
- Workspaces: "Compliance badges: ✅ Encrypted, ✅ CUI-Approved, ✅ Audited"
- Projects: "Automated quarterly compliance reports for NIH ISSO"

---

### 🔄 07 - NIH Researcher (PHI/HIPAA Compliance) (TODO)
**Status**: 0/5 screenshots
**Persona**: Dr. Lee analyzing Protected Health Information (PHI)

**Integration Strategy**: Emphasize **HIPAA compliance** and **BAA agreements**

| Screenshot | Suggested Location | Context Angle |
|------------|-------------------|---------------|
| **Settings** | HIPAA Validation (~line 35) | Validating BAA-compliant AWS account configuration |
| **Quick Start** | PHI Workspace Launch (~line 80) | HIPAA-compliant template with required safeguards |
| **Storage** | Encrypted PHI Storage (~line 125) | HIPAA-compliant encrypted EFS with access logging |
| **Workspaces** | HIPAA Dashboard (~line 190) | Workspaces showing BAA status and audit logging |
| **Projects** | Clinical Trial Budget (~line 270) | NIH R01 clinical trial with HIPAA compliance tracking |

**Key Differences**:
- Settings: "Validate AWS BAA agreement and HIPAA-eligible services"
- Storage: "HIPAA-compliant EFS with CloudTrail logging and encryption"
- Workspaces: "HIPAA badges: ✅ BAA, ✅ Encrypted, ✅ Audit Logging, ✅ PHI-Safe"
- Projects: "Automated HIPAA compliance attestations for IRB and NIH"

---

### 🔄 08 - Institutional Research IT (TODO)
**Status**: 0/5 screenshots
**Persona**: Alex (IT Admin) managing university research computing

**Integration Strategy**: Emphasize **multi-tenant administration** and **policy enforcement**

| Screenshot | Suggested Location | Context Angle |
|------------|-------------------|---------------|
| **Settings** | Multi-Tenant Setup (~line 40) | IT Admin configuring university-wide AWS Organization |
| **Quick Start** | Faculty Onboarding (~line 90) | Template marketplace with approved institutional templates |
| **Storage** | Storage Quota Management (~line 140) | University-wide storage allocation across 12 departments |
| **Workspaces** | Admin Dashboard (~line 200) | 500+ workspaces across all faculty, sorted by department |
| **Projects** | Chargeback System (~line 280) | Automated monthly billing to department budgets |

**Key Differences**:
- Settings: "IT Admin validates AWS Organizations SCPs and cost allocation tags"
- Storage: "University-wide storage quotas: 100TB total, 10TB per department"
- Workspaces: "Admin view: 500 workspaces, filter by dept/PI/grant/compliance"
- Projects: "Automated monthly chargeback to 12 department financial systems"

---

## 📊 Implementation Progress

**Total Personas**: 8
**Completed**: 1 (12.5%)
**Remaining**: 7 (87.5%)

**Total Screenshot Integrations**: 40 (8 personas × 5 screenshots)
**Completed**: 5 (12.5%)
**Remaining**: 35 (87.5%)

---

## 🔄 Execution Plan

### Phase 1: Copy Screenshots (5 minutes)
```bash
# Copy Solo Researcher screenshots to all other persona directories
for persona in 02-lab-environment 03-university-class 04-conference-workshop \
               05-cross-institutional 06-nih-cui 07-nih-hipaa 08-institutional-it; do
  cp docs/USER_SCENARIOS/images/01-solo-researcher/*.png \
     docs/USER_SCENARIOS/images/$persona/
done
```

### Phase 2: Systematic Integration (40-50 minutes, ~5-7 min/persona)

#### Priority Order (by impact):
1. **Lab Environment** - High-value persona for academic adoption
2. **University Class** - Demonstrates scalability and educational use
3. **Conference Workshop** - Shows rapid provisioning and time-limited usage
4. **Institutional IT** - Appeals to IT decision-makers
5. **Cross-Institutional** - Demonstrates enterprise collaboration
6. **NIH CUI** - Security-conscious research compliance
7. **NIH HIPAA** - Clinical research compliance

### Phase 3: Validation & Testing (10 minutes)
- Verify all markdown images render correctly
- Check for broken image links
- Validate context makes sense for each persona
- Review integration text for persona consistency

---

## 🎨 Integration Template

For each persona, use this template pattern:

```markdown
**[Feature Name]** (with persona-specific context):

![Screenshot Alt Text](images/[persona-dir]/[screenshot-name].png)

*Screenshot shows [interface description]. [Persona-specific context sentence explaining
why this matters to this specific persona and how it fits their workflow].*

**What [Persona Name] [uses/sees/configures]**:
- **Feature 1**: [Persona-specific benefit/usage]
- **Feature 2**: [Persona-specific benefit/usage]
- **Feature 3**: [Persona-specific benefit/usage]
- **Feature 4**: [Persona-specific benefit/usage]
```

---

## 📝 Success Metrics

**Completion Criteria**:
- ✅ All 7 remaining personas have 5 screenshots integrated
- ✅ Each screenshot has persona-appropriate contextual text
- ✅ Integration enhances persona's narrative flow
- ✅ All image links verified working
- ✅ SCREENSHOT_INTEGRATION_GUIDE.md updated with completion status

**Expected Outcome**:
- 40 total screenshot integrations (8 personas × 5 screenshots)
- 60-70% reduction in "am I doing this right?" anxiety across all personas
- Visual documentation completeness for institutional evaluations

---

**Last Updated**: October 27, 2025
**Next Steps**: Execute Phase 1 (copy screenshots), then Phase 2 (systematic integration)
