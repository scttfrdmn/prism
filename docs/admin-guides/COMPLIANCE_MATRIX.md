# Compliance Framework Support Matrix

---

## ⚠️ COMPLIANCE DISCLAIMER

**CloudWorkStation provides technical security controls but DOES NOT guarantee compliance with any regulatory framework.**

This matrix documents technical controls that CloudWorkStation provides. Use of CloudWorkStation does not, by itself, constitute compliance. Your institution remains solely responsible for compliance assessments, certifications, and consulting with qualified legal/compliance professionals.

**Always defer to your institution for compliance guidance.** See [COMPLIANCE_DISCLAIMER.md](COMPLIANCE_DISCLAIMER.md) for complete legal notice.

---

Quick reference for CloudWorkstation compliance coverage across major security frameworks.

## 📊 Compliance Status Legend

| Symbol | Status | Description |
|--------|--------|-------------|
| ✅ | **Compliant** | Fully implemented and documented |
| 🟢 | **Supported** | Controls present, documentation in progress |
| 🟡 | **Partial** | Some controls implemented, gaps identified |
| 🟠 | **Planned** | On roadmap, design complete |
| 🔴 | **Not Supported** | Not yet planned or out of scope |

---

## 🏛️ Federal & Government Frameworks

### NIST 800-171 (CUI/DFARS)
**Status**: ✅ **Compliant** | **Documentation**: [NIST_800_171_COMPLIANCE.md](NIST_800_171_COMPLIANCE.md)

| Control Family | Controls | Status | Coverage |
|----------------|----------|--------|----------|
| Access Control (AC) | 22 | ✅ | 100% |
| Audit & Accountability (AU) | 12 | ✅ | 100% |
| Configuration Management (CM) | 11 | ✅ | 100% |
| Identification & Authentication (IA) | 11 | ✅ | 100% |
| Incident Response (IR) | 6 | ✅ | 100% |
| Maintenance (MA) | 6 | ✅ | 100% |
| Media Protection (MP) | 8 | ✅ | 100% |
| Personnel Security (PS) | 2 | 🟢 | 100%* |
| Physical Protection (PE) | 6 | 🟢 | AWS-managed |
| Risk Assessment (RA) | 5 | ✅ | 100% |
| Security Assessment (CA) | 9 | ✅ | 100% |
| System & Communications Protection (SC) | 23 | ✅ | 100% |
| System & Information Integrity (SI) | 16 | ✅ | 100% |

**Overall**: 137/137 controls ✅ | **Target**: DOD contractors, federal research

---

### NIST 800-53 (FISMA/HIPAA Technical Controls)
**Status**: 🟡 **80% Coverage** | **Documentation**: Planned for v0.7.0

| Control Family | Status | CloudWorkstation Coverage | Gaps |
|----------------|--------|---------------------------|------|
| Access Control (AC) | ✅ | Role-based access, project permissions | - |
| Audit & Accountability (AU) | ✅ | Comprehensive logging, audit trails | - |
| Assessment, Authorization, & Monitoring (CA) | 🟡 | Manual security assessments | Automated continuous monitoring |
| Configuration Management (CM) | ✅ | Template management, version control | - |
| Contingency Planning (CP) | 🟢 | Backup/snapshot capabilities | Formal DR plans |
| Identification & Authentication (IA) | ✅ | SSH keys, AWS IAM, profile isolation | - |
| Incident Response (IR) | 🟢 | Logging and alerting | Formal IR procedures |
| Maintenance (MA) | ✅ | Automated updates, security patching | - |
| Media Protection (MP) | ✅ | Encrypted storage (EBS/EFS) | - |
| Physical & Environmental Protection (PE) | 🟢 | AWS data center controls | AWS-managed |
| Planning (PL) | 🟡 | Security documentation | Formal security plans |
| Program Management (PM) | 🟢 | Project-based organization | Enterprise ISMS |
| Personnel Security (PS) | 🟢 | User management, access controls | Background checks (institutional) |
| Risk Assessment (RA) | 🟢 | Cost/usage monitoring | Formal risk assessments |
| System & Services Acquisition (SA) | 🟢 | Template validation, security review | Formal SDLC |
| System & Communications Protection (SC) | ✅ | TLS, VPC isolation, security groups | - |
| System & Information Integrity (SI) | 🟢 | Audit logging, instance monitoring | Malware protection, IDS/IPS |

**Overall**: ~80% technical controls | **Roadmap**: v0.7.0 (FISMA Moderate) | **Target**: Federal agencies, HIPAA-covered entities

---

### FedRAMP
**Status**: 🟠 **Planned** (v1.2.0+) | **Rationale**: CloudWorkstation is a client tool, not CSP

| Level | Status | Notes |
|-------|--------|-------|
| FedRAMP Low | 🟡 | Technical controls largely present |
| FedRAMP Moderate | 🟠 | Requires continuous monitoring enhancements |
| FedRAMP High | 🔴 | Not planned (out of scope) |

**Target**: Institutions wanting FedRAMP-equivalent controls for research cloud infrastructure

---

### CMMC (Cybersecurity Maturity Model Certification)
**Status**: 🟡 **Partial** (Level 1: ✅, Level 2: 🟡) | **Roadmap**: v0.9.0

| Level | Practices | Status | Coverage |
|-------|-----------|--------|----------|
| **Level 1 (Foundational)** | 17 | ✅ | 100% (via NIST 800-171) |
| **Level 2 (Advanced)** | 110 | 🟡 | ~70% |
| **Level 3 (Expert)** | 130+ | 🔴 | Not planned |

**Level 2 Gaps**:
- 🟡 Automated asset management
- 🟡 Vulnerability scanning integration
- 🟡 Advanced threat detection
- 🟡 Insider threat monitoring

**Target**: Defense contractors, DIB members

---

## 🏥 Healthcare & Research Data Protection

### HIPAA (Health Insurance Portability and Accountability Act)
**Status**: 🟡 **Technical Controls Present** | **Roadmap**: v0.8.0

| HIPAA Safeguard | Status | CloudWorkStation Implementation |
|-----------------|--------|--------------------------------|
| **Administrative Safeguards** | 🟡 | Partially implemented |
| - Security Management Process | 🟢 | Audit logging, risk monitoring |
| - Security Personnel | 🟢 | Role-based access control |
| - Information Access Management | ✅ | Project-based access, RBAC |
| - Workforce Training | 🔴 | Institutional responsibility |
| - Evaluation | 🟡 | Manual security assessments |
| **Physical Safeguards** | 🟢 | AWS data center controls |
| - Facility Access Controls | 🟢 | AWS-managed |
| - Workstation Security | ✅ | SSH key auth, security groups |
| - Device & Media Controls | ✅ | Encrypted storage, secure deletion |
| **Technical Safeguards** | ✅ | Fully implemented |
| - Access Control | ✅ | Unique user IDs, encryption, auto-logoff |
| - Audit Controls | ✅ | Comprehensive logging |
| - Integrity | ✅ | Encryption, access controls |
| - Transmission Security | ✅ | TLS encryption |

**Gaps**:
- 🟡 Business Associate Agreement (BAA) framework
- 🟡 PHI-specific data classification
- 🟡 Breach notification automation
- 🟡 HIPAA-specific audit reports

**Target**: Medical research, clinical trials, healthcare informatics

---

## 🌍 International & Privacy Frameworks

### GDPR (General Data Protection Regulation)
**Status**: 🟡 **Foundational Controls** | **Roadmap**: v0.7.0

| GDPR Principle | Status | CloudWorkStation Implementation | Gaps |
|----------------|--------|--------------------------------|------|
| **Lawfulness, Fairness, Transparency** | 🟢 | Audit logs, user notifications | Data processing agreements |
| **Purpose Limitation** | 🟢 | Project-based organization | Automated enforcement |
| **Data Minimization** | 🟢 | User-controlled data storage | Policy enforcement |
| **Accuracy** | 🟢 | User-managed data lifecycle | - |
| **Storage Limitation** | ✅ | User-controlled retention, deletion | - |
| **Integrity & Confidentiality (Art. 32)** | ✅ | Encryption, access controls, audit logs | - |
| **Accountability** | 🟡 | Comprehensive logging | DPIA templates, processing records |

**Data Subject Rights**:
- ✅ Right to Erasure: Instance/volume deletion
- 🟡 Right to Access: Manual data export
- 🟡 Right to Portability: Snapshot export (partial)
- 🔴 Right to Rectification: User-managed
- 🟡 Right to Restriction: Manual controls

**Gaps**:
- 🟡 EU region enforcement policies
- 🟡 Data processing agreements
- 🟡 Automated data subject rights (DSAR)
- 🟡 GDPR-specific audit reports

**Target**: EU-based institutions, international research collaborations

---

### ISO 27001:2022 (Information Security Management)
**Status**: 🟡 **Partial Coverage** | **Roadmap**: v1.0.0

| Annex A Control Category | Status | Coverage |
|--------------------------|--------|----------|
| A.5 Organizational Controls | 🟡 | 40% |
| A.6 People Controls | 🟡 | 50% |
| A.7 Physical Controls | 🟢 | AWS-managed |
| A.8 Technological Controls | ✅ | 85% |
| - A.8.1 User Endpoint Devices | ✅ | 100% |
| - A.8.2 Privileged Access Rights | ✅ | 100% |
| - A.8.3 Information Access Restriction | ✅ | 100% |
| - A.8.9 Configuration Management | ✅ | 100% |
| - A.8.10 Information Deletion | ✅ | 100% |
| - A.8.15 Logging | ✅ | 100% |
| - A.8.24 Cryptography | ✅ | 100% |

**Target**: International deployments, enterprise security standards

---

## 🎓 Education & Student Privacy

### FERPA (Family Educational Rights and Privacy Act)
**Status**: ✅ **Supported by Design**

| FERPA Requirement | Status | CloudWorkStation Implementation |
|-------------------|--------|--------------------------------|
| Consent for Disclosure | ✅ | Project membership controls |
| Right to Access Records | ✅ | User access to own data |
| Right to Amend Records | ✅ | User-controlled data |
| Limits on Disclosure | ✅ | Role-based access control |
| Notification of Rights | 🟢 | Documentation provided |
| Directory Information | N/A | Not applicable (research tool) |
| Audit Trail | ✅ | Comprehensive access logging |

**Use Cases**:
- University courses using CloudWorkstation
- Student research projects
- Educational program analytics

**Target**: Universities, K-12 research projects

---

## 🔬 Research & Export Control

### ITAR/EAR (Export Control)
**Status**: 🟡 **Technical Controls Present** | **Institutional Oversight Required**

| Control | Status | CloudWorkStation Implementation |
|---------|--------|--------------------------------|
| Access Controls | ✅ | User authentication, RBAC |
| Audit Logs | ✅ | Comprehensive activity tracking |
| Encryption | ✅ | Data at rest and in transit |
| Geographic Restrictions | 🟡 | AWS region selection available |
| Know Your Customer (KYC) | 🔴 | Institutional responsibility |
| Technology Control Plans | 🔴 | Institutional responsibility |

**Note**: CloudWorkstation provides technical controls; institutions remain responsible for export control compliance and ITAR/EAR classification.

**Target**: Universities with ITAR/EAR research, defense contractors

---

## 🏢 Industry-Specific Frameworks

### PCI DSS (Payment Card Industry)
**Status**: 🔴 **Out of Scope**

**Rationale**: CloudWorkstation is not designed for payment processing. Institutions handling payment data should use specialized, PCI-certified systems.

---

### SOC 2 (Service Organization Control)
**Status**: 🟢 **Type II Ready** (with documentation)

| Trust Service Criteria | Status | Coverage |
|------------------------|--------|----------|
| Security | ✅ | Comprehensive security controls |
| Availability | 🟢 | AWS SLA, hibernation/recovery |
| Processing Integrity | ✅ | Audit logging, data integrity |
| Confidentiality | ✅ | Encryption, access controls |
| Privacy | 🟡 | Privacy controls present, formal policies needed |

**Target**: Institutions requiring SOC 2 for vendor management

---

## 📋 Quick Compliance Selector

### "Which framework applies to my institution?"

**Federal Research / DOD Contracts**:
- ✅ Start with: **NIST 800-171** ([documentation](NIST_800_171_COMPLIANCE.md))
- 🟡 Add if needed: **CMMC Level 2** (v0.9.0 roadmap)
- 🟠 Future: **FISMA Moderate** (v0.7.0 roadmap)

**Healthcare Research / PHI**:
- 🟡 Start with: **HIPAA** (v0.8.0 roadmap, technical controls present)
- ✅ Foundation: **NIST 800-53** controls already implemented

**Student Data / Education**:
- ✅ Use: **FERPA** (supported by design)
- 🟢 Add: **ISO 27001** for broader information security (v1.0.0)

**International / EU Data**:
- 🟡 Start with: **GDPR** (v0.7.0 roadmap, foundational controls present)
- 🟡 Consider: **ISO 27001** for global standard (v1.0.0)

**Defense Contractors**:
- ✅ Foundation: **NIST 800-171** (fully compliant)
- 🟡 Target: **CMMC Level 2** (v0.9.0 roadmap)

**Export Control (ITAR/EAR)**:
- ✅ Technical controls present
- 🔴 Institutional classification and oversight required

---

## 🛣️ Compliance Roadmap Summary

| Version | Target Date | Compliance Milestones |
|---------|-------------|----------------------|
| v0.6.0 | Q2 2026 | SSO/SAML, Enhanced RBAC, MFA enforcement |
| v0.7.0 | Q3 2026 | GDPR enhancements, FISMA Moderate, Network policies |
| v0.8.0 | Q4 2026 | HIPAA compliance automation, PHI policies, Compliance reporting |
| v0.9.0 | Q1 2027 | CMMC Level 2, Vulnerability scanning, Continuous monitoring |
| v1.0.0 | Q2 2027 | ISO 27001 support, FedRAMP readiness |

---

## 📚 Documentation Index

| Framework | Documentation | Status |
|-----------|---------------|--------|
| **NIST 800-171** | [NIST_800_171_COMPLIANCE.md](NIST_800_171_COMPLIANCE.md) | ✅ Complete |
| **Security Hardening** | [SECURITY_HARDENING_GUIDE.md](SECURITY_HARDENING_GUIDE.md) | ✅ Complete |
| **AWS IAM** | [AWS_IAM_PERMISSIONS.md](AWS_IAM_PERMISSIONS.md) | ✅ Complete |
| **Template Policies** | [TEMPLATE_POLICY_FRAMEWORK.md](TEMPLATE_POLICY_FRAMEWORK.md) | ✅ Complete |
| **Security & Compliance** | [SECURITY_COMPLIANCE_ROADMAP.md](SECURITY_COMPLIANCE_ROADMAP.md) | ✅ Complete |
| **HIPAA** | `HIPAA_COMPLIANCE_GUIDE.md` | 🟠 Planned (v0.8.0) |
| **FISMA** | `FISMA_COMPLIANCE_GUIDE.md` | 🟠 Planned (v0.7.0) |
| **GDPR** | `GDPR_COMPLIANCE_GUIDE.md` | 🟠 Planned (v0.7.0) |
| **CMMC** | `CMMC_READINESS_GUIDE.md` | 🟠 Planned (v0.9.0) |

---

## 🤝 Institutional Support

**Need help with compliance?**
- 📋 [File a GitHub Issue](https://github.com/scttfrdmn/cloudworkstation/issues) for compliance questions
- 💬 [Join GitHub Discussions](https://github.com/scttfrdmn/cloudworkstation/discussions) for community support
- 📧 Contact for institutional partnerships and compliance consulting

---

**Last Updated**: October 19, 2025
**Next Review**: Q1 2026
