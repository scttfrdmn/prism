# Compliance Disclaimer

**Last Updated**: October 19, 2025

---

## ⚠️ Important Legal Notice

**Prism is open source software that provides technical security controls. Use of Prism DOES NOT, by itself, ensure compliance with any regulatory framework, standard, or legal requirement.**

---

## No Legal Advice

The information provided in Prism documentation regarding compliance frameworks (including but not limited to NIST 800-171, HIPAA, FISMA, GDPR, CMMC, FERPA, ISO 27001, SOC 2, and other standards) **does not constitute legal, regulatory, or compliance advice**.

You should consult with qualified legal advisors, compliance officers, and information security professionals for questions regarding regulatory compliance specific to your organization, institution, or use case.

---

## No Warranty of Compliance

Prism provides technical security controls and features that **may assist** organizations in meeting certain technical requirements of various compliance frameworks. However:

1. **Compliance is Multi-Faceted**: Meeting a compliance framework requires more than technical controls. It typically involves:
   - Organizational policies and procedures
   - Administrative safeguards
   - Personnel training and awareness
   - Risk assessments and audits
   - Legal agreements (e.g., Business Associate Agreements for HIPAA)
   - Physical security measures
   - Incident response plans
   - Continuous monitoring and assessment

2. **Institutional Responsibility**: Your institution, organization, or entity remains solely responsible for:
   - Determining which compliance frameworks apply to your activities
   - Assessing your compliance obligations
   - Implementing appropriate controls beyond Prism
   - Conducting compliance audits and assessments
   - Maintaining compliance over time
   - Obtaining necessary certifications or attestations

3. **Configuration Dependent**: The security posture of any Prism deployment depends heavily on:
   - User configuration choices
   - Institutional policies applied via profiles
   - AWS account security configuration
   - IAM roles and permissions
   - Network architecture
   - User behavior and training
   - Integration with institutional security tools

---

## Defer to Your Institution

**Always defer to your institution's guidance** on compliance requirements:

- **Research Security Office**: For federal research security requirements (CUI, export control, ITAR/EAR)
- **Information Security Office**: For technical security controls and cybersecurity standards
- **Privacy Office / HIPAA Privacy Officer**: For HIPAA compliance and protected health information (PHI)
- **Compliance Office**: For institutional compliance policies and risk assessment
- **Office of General Counsel**: For legal interpretation of regulatory requirements
- **Institutional Review Board (IRB)**: For human subjects research and data protection

Your institution may have:
- Pre-approved configurations or profiles for Prism
- Additional requirements beyond the compliance framework baseline
- Specific policies prohibiting or restricting certain cloud services
- Required security tools (endpoint agents, SIEM integration, etc.)
- Mandatory training or certification requirements

**Always obtain institutional approval before using Prism for compliance-sensitive research.**

---

## Third-Party Services

Prism relies on third-party cloud infrastructure providers (primarily Amazon Web Services). While AWS provides:
- HIPAA-eligible services with Business Associate Agreements (BAA)
- FedRAMP-authorized cloud services
- SOC 2 Type II attestations
- ISO/IEC 27001 certifications

**Your responsibility remains to**:
- Execute appropriate agreements with cloud providers (e.g., AWS BAA for HIPAA)
- Configure services according to compliance requirements
- Monitor and audit cloud resource usage
- Ensure data residency and sovereignty requirements are met
- Validate that the cloud provider meets your institutional standards

---

## Framework-Specific Disclaimers

### NIST 800-171 (Controlled Unclassified Information)

Prism provides technical controls that **may align with** NIST SP 800-171 Rev. 3 security requirements. However:

- **Attestation Responsibility**: Your institution must attest to NIST 800-171 compliance, not Prism
- **System Security Plans (SSP)**: Your institution must maintain an SSP documenting how all 110 requirements are met
- **Continuous Monitoring**: NIST 800-171 compliance requires ongoing assessment, not one-time implementation
- **Institutional Controls**: Many NIST 800-171 requirements (e.g., personnel security, physical protection) are outside Prism's scope

**For federal contracts requiring NIST 800-171**: Consult your institution's Research Security Office and Sponsored Projects Office before using Prism for CUI data.

### HIPAA (Protected Health Information)

Prism provides technical controls that **may support** HIPAA Security Rule Technical Safeguards (45 CFR § 164.312). However:

- **Covered Entity Responsibility**: Your institution (as a covered entity) remains responsible for HIPAA compliance
- **Business Associate Agreements**: You must ensure BAAs are in place with AWS and any other service providers
- **Risk Analysis Required**: HIPAA requires a comprehensive risk analysis specific to your use case
- **Administrative & Physical Safeguards**: HIPAA compliance requires safeguards beyond Prism's scope
- **Privacy Rule**: HIPAA Privacy Rule requirements (45 CFR § 164.500) must be addressed separately

**For research involving PHI**: Consult your institution's HIPAA Privacy Officer, IRB, and Information Security Office before using Prism.

### CMMC (Cybersecurity Maturity Model Certification)

Prism's CMMC support is **planned for future releases** (v0.9.0 target, Q1 2027). Currently:

- **DOD Contracts**: CMMC Level 2 (or higher) is required for DOD contracts as of October 1, 2025
- **Third-Party Assessment**: CMMC requires assessment by a certified third-party assessor (C3PAO)
- **Institutional Certification**: Your institution must obtain CMMC certification, not individual tools

**For DOD-funded research**: Do not use Prism for CMMC-required work until institutional certification is obtained and Prism is included in the scope.

### FISMA (Federal Information Security Management Act)

Prism provides controls that **may align with** NIST 800-53 security controls (used for FISMA compliance). However:

- **ATO Required**: FISMA compliance requires an Authority to Operate (ATO) from your agency
- **Continuous Monitoring**: FISMA requires ongoing security assessments and monitoring
- **Agency-Specific Requirements**: Federal agencies have additional security requirements beyond NIST 800-53

**For federal information systems**: Consult your agency's CISO and Information Security Office before using Prism.

### GDPR (General Data Protection Regulation)

Prism provides technical controls that **may support** GDPR Article 32 (Security of Processing). However:

- **Data Controller Responsibility**: Your institution (as data controller) is responsible for GDPR compliance
- **Data Processing Agreements**: Required with AWS and other processors
- **Data Subject Rights**: GDPR grants rights (access, rectification, erasure) that must be supported by your processes
- **Data Protection Impact Assessment**: May be required for high-risk processing activities
- **EU Data Residency**: Ensure AWS regions used comply with data residency requirements

**For EU personal data**: Consult your institution's Data Protection Officer (DPO) and legal counsel before using Prism.

---

## Documentation Purpose

Prism's compliance documentation serves the following purposes **only**:

1. **Educational**: To inform users about relevant compliance frameworks and their requirements
2. **Technical Mapping**: To document which technical controls Prism provides
3. **Gap Analysis**: To identify where additional institutional controls are needed
4. **Best Practices**: To share security configuration recommendations

**This documentation does NOT**:
- ❌ Constitute a compliance certification or attestation
- ❌ Replace institutional compliance assessments
- ❌ Guarantee that your use of Prism will meet any specific compliance requirement
- ❌ Create any warranty, expressed or implied, regarding compliance
- ❌ Establish an attorney-client or advisory relationship

---

## Changes to Compliance Requirements

Compliance frameworks, regulations, and standards change over time. Prism documentation reflects the state of these frameworks at the time of writing, but:

- **Your Responsibility**: Stay informed about changes to applicable compliance requirements
- **No Automatic Updates**: Prism does not automatically update to meet new compliance requirements
- **Version-Specific**: Compliance guidance applies to specific Prism versions and may not apply to older or newer versions

---

## No Liability

**Prism is provided "AS IS" without warranty of any kind**, either express or implied, including but not limited to the implied warranties of merchantability, fitness for a particular purpose, or non-infringement.

In no event shall the Prism project, contributors, or copyright holders be liable for any claim, damages, or other liability arising from:
- Use or inability to use Prism
- Failure to meet compliance requirements
- Security incidents or data breaches
- Regulatory penalties or sanctions
- Loss of funding or contracts due to non-compliance

See the [Apache License 2.0](../../LICENSE) for full terms.

---

## Getting Appropriate Guidance

To ensure compliance with applicable regulations and standards:

1. **Consult Institutional Experts**:
   - Information Security Office
   - HIPAA Privacy Officer (for PHI)
   - Research Security Office (for CUI, export control)
   - Compliance Officer
   - Office of General Counsel
   - Institutional Review Board (for human subjects research)

2. **Engage Qualified Professionals**:
   - Legal counsel specializing in regulatory compliance
   - Certified compliance professionals (e.g., CISSP, CISM, HCISPP)
   - Third-party auditors (e.g., C3PAO for CMMC)

3. **Review Official Sources**:
   - NIST publications (https://csrc.nist.gov/)
   - HHS HIPAA guidance (https://www.hhs.gov/hipaa/)
   - Federal agency notices (e.g., NIH NOT-OD-24-157)
   - Regulatory authority websites

4. **Conduct Risk Assessments**:
   - Perform compliance gap analysis for your specific use case
   - Document security controls and their implementation
   - Obtain institutional approval before processing sensitive data

---

## Contributing to Compliance Documentation

If you identify inaccuracies in Prism's compliance documentation:

1. **File a GitHub Issue**: [Report documentation issues](https://github.com/scttfrdmn/prism/issues)
2. **Submit a Pull Request**: Contribute corrections with authoritative citations
3. **Engage in Discussions**: [Join discussions](https://github.com/scttfrdmn/prism/discussions)

**Note**: Contributions to compliance documentation do not create liability for contributors. All contributions are subject to the Apache License 2.0.

---

## Questions?

For questions about Prism features and security controls:
- **GitHub Issues**: [Technical questions and bug reports](https://github.com/scttfrdmn/prism/issues)
- **GitHub Discussions**: [Community discussions](https://github.com/scttfrdmn/prism/discussions)

For questions about compliance obligations:
- **Consult your institution's compliance offices** (see "Getting Appropriate Guidance" above)
- **Do NOT rely on Prism documentation as legal or compliance advice**

---

**Last Updated**: October 19, 2025
**Effective For**: Prism v0.5.x and later
**Review Cycle**: Annually or upon significant regulatory changes
