#!/bin/bash
set -e

REPO="scttfrdmn/cloudworkstation"

echo "🚀 Creating GitHub issues for all remaining phases..."
echo ""

# Phase 5.2 (already complete per CLAUDE.md)
echo "📦 Phase 5.2: Template Marketplace Foundation (COMPLETED)"
echo "Skipping - already implemented in v0.5.2"
echo ""

# Phase 5.3: Advanced Storage Integration
echo "📦 Creating Phase 5.3: Advanced Storage Integration issues..."

gh issue create \
  --repo "$REPO" \
  --title "[Storage] FSx Integration for High-Performance Workloads" \
  --body "## Summary
Add Amazon FSx filesystem support for research workloads requiring high-performance storage (Lustre, OpenZFS).

## Motivation
Researchers working with large datasets, HPC, or ML training need high-throughput, low-latency storage beyond EFS capabilities.

## Implementation Tasks
- [ ] FSx Lustre integration for HPC workloads
- [ ] FSx OpenZFS integration for general high-performance needs
- [ ] Template schema extension for FSx configuration
- [ ] CLI commands: \`cws storage create --type fsx-lustre\`
- [ ] GUI interface for FSx filesystem management
- [ ] Cost estimation and comparison (FSx vs EFS vs EBS)

## Persona Impact
- **Solo Researcher**: High-performance storage for large datasets
- **Lab Environment**: Shared high-performance filesystem for team
- **University Class**: Fast storage for student compute workloads

## Success Metrics
- FSx filesystems can be created and attached to instances
- Performance testing shows expected throughput improvements
- Clear cost comparison helps researchers choose right storage

## Related
- Phase 5.3: Advanced Storage Integration
- v0.5.3 milestone" \
  --milestone "Phase 5.3: Advanced Storage" \
  --label "enhancement,area: storage,priority: medium,phase: 5.3-storage"

gh issue create \
  --repo "$REPO" \
  --title "[Storage] S3 Mount Points for Direct Data Access" \
  --body "## Summary
Enable direct S3 bucket mounting to CloudWorkstation instances for seamless data access.

## Motivation
Many research datasets live in S3. Researchers should access them directly without manual downloads.

## Implementation Tasks
- [ ] S3 FUSE integration (s3fs or goofys)
- [ ] Template schema for S3 mount configuration
- [ ] IAM role automation for S3 access
- [ ] CLI commands: \`cws storage mount s3://bucket/path\`
- [ ] GUI interface for S3 bucket browsing and mounting
- [ ] Read-only vs read-write mount options

## Persona Impact
- **Solo Researcher**: Direct access to personal S3 datasets
- **Cross-Institutional**: Shared S3 buckets for multi-institution data
- **Conference Workshop**: Pre-populated S3 datasets for attendees

## Success Metrics
- S3 buckets mount successfully with proper permissions
- Read/write performance meets expectations
- Clear IAM policy generation for secure access

## Related
- Phase 5.3: Advanced Storage Integration
- v0.5.3 milestone" \
  --milestone "Phase 5.3: Advanced Storage" \
  --label "enhancement,area: storage,area: aws,priority: medium,phase: 5.3-storage"

gh issue create \
  --repo "$REPO" \
  --title "[Storage] Storage Analytics and Cost Optimization" \
  --body "## Summary
Provide detailed analytics on storage usage patterns and cost optimization recommendations.

## Motivation
Storage costs can grow unexpectedly. Researchers need visibility and optimization guidance.

## Implementation Tasks
- [ ] Storage usage tracking (EFS, EBS, FSx, S3)
- [ ] Cost breakdown by storage type and instance
- [ ] Usage pattern analysis (hot vs cold data)
- [ ] Automated recommendations (EFS → S3 Glacier for cold data)
- [ ] CLI: \`cws storage analyze\` with cost reports
- [ ] GUI dashboard with storage cost visualizations

## Persona Impact
- **Lab Environment**: Track team storage spending
- **University Class**: Identify student storage waste
- **Solo Researcher**: Optimize personal storage costs

## Success Metrics
- Clear visibility into storage costs by type
- Actionable recommendations save 20%+ on storage
- Automated alerts for unexpected storage growth

## Related
- Phase 5.3: Advanced Storage Integration
- v0.5.3 milestone" \
  --milestone "Phase 5.3: Advanced Storage" \
  --label "enhancement,area: storage,priority: medium,phase: 5.3-storage"

echo "✅ Phase 5.3 issues created"
echo ""

# Phase 5.4: Policy Framework Enhancement
echo "📦 Creating Phase 5.4: Policy Framework Enhancement issues..."

gh issue create \
  --repo "$REPO" \
  --title "[Policy] Advanced Template Access Policies" \
  --body "## Summary
Institutional control over which templates users can access and launch.

## Motivation
Universities need to restrict expensive GPU templates, limit commercial software, or enforce approved template lists.

## Implementation Tasks
- [ ] Template whitelist/blacklist policies
- [ ] Cost-based template restrictions
- [ ] Department/group-based template access
- [ ] Policy inheritance (institution → department → user)
- [ ] CLI: \`cws admin policy template --allow python-ml --deny gpu-workstation\`
- [ ] GUI policy editor with template preview

## Persona Impact
- **University Class**: Instructors limit students to course-appropriate templates
- **Lab Environment**: PIs control expensive resource access
- **Cross-Institutional**: Different policies per institution

## Success Metrics
- Policies prevent unauthorized template launches
- Clear error messages guide users to approved templates
- Policy violations logged for audit

## Related
- Phase 5.4: Policy Framework Enhancement
- v0.5.4 milestone" \
  --milestone "Phase 5.4: Policy Framework" \
  --label "enhancement,area: policy,priority: high,phase: 5.4-policy"

gh issue create \
  --repo "$REPO" \
  --title "[Policy] Resource Limit Policies" \
  --body "## Summary
Enforce limits on instance types, storage sizes, and concurrent resources per user/group.

## Motivation
Prevent accidental cost overruns and ensure fair resource allocation in shared environments.

## Implementation Tasks
- [ ] Max instance type policies (no instances > c5.4xlarge)
- [ ] Max concurrent instances per user
- [ ] Max storage size policies (EFS, EBS)
- [ ] Max project budget enforcement
- [ ] Grace period before hard limit enforcement
- [ ] CLI: \`cws admin policy limits --max-instances 5 --max-instance-type c5.2xlarge\`

## Persona Impact
- **University Class**: Prevent students from launching expensive instances
- **Lab Environment**: Fair resource allocation among team members
- **Solo Researcher**: Personal budget guardrails

## Success Metrics
- Resource launches blocked when limits exceeded
- Clear guidance on current usage vs limits
- Admins can adjust limits per user/group

## Related
- Phase 5.4: Policy Framework Enhancement
- v0.5.4 milestone" \
  --milestone "Phase 5.4: Policy Framework" \
  --label "enhancement,area: policy,priority: high,phase: 5.4-policy"

gh issue create \
  --repo "$REPO" \
  --title "[Policy] Compliance and Audit Logging" \
  --body "## Summary
Comprehensive audit logging for institutional compliance (NIST 800-171, SOC 2, HIPAA).

## Motivation
Institutions need detailed audit trails for compliance, security investigations, and cost attribution.

## Implementation Tasks
- [ ] Detailed action logging (who, what, when, where, why)
- [ ] Immutable audit log storage (S3 with versioning)
- [ ] Compliance report generation (NIST 800-171, SOC 2)
- [ ] User activity dashboards
- [ ] Anomaly detection (unusual launch patterns)
- [ ] CLI: \`cws admin audit --user alice --date-range 2025-10-01..2025-10-31\`

## Persona Impact
- **University Class**: Track student resource usage
- **Lab Environment**: Attribute costs to grants/projects
- **Cross-Institutional**: Multi-institution audit trails

## Success Metrics
- All user actions logged with full context
- Compliance reports meet institutional requirements
- Audit logs support security investigations

## Related
- Phase 5.4: Policy Framework Enhancement
- v0.5.4 milestone" \
  --milestone "Phase 5.4: Policy Framework" \
  --label "enhancement,area: policy,security,priority: high,phase: 5.4-policy"

echo "✅ Phase 5.4 issues created"
echo ""

# Phase 5.5: AWS Research Services Integration
echo "📦 Creating Phase 5.5: AWS Research Services Integration issues..."

gh issue create \
  --repo "$REPO" \
  --title "[AWS] EMR Studio Integration for Big Data Analytics" \
  --body "## Summary
Integrate Amazon EMR Studio for Spark-based big data research and analytics.

## Motivation
Researchers analyzing large datasets need distributed computing beyond single EC2 instances.

## Implementation Tasks
- [ ] EMR cluster provisioning from templates
- [ ] EMR Studio workspace integration
- [ ] Jupyter notebook synchronization
- [ ] S3 data lake integration
- [ ] CLI: \`cws launch emr-spark big-data-analysis\`
- [ ] GUI EMR cluster management interface

## Persona Impact
- **Solo Researcher**: Big data analysis without infrastructure management
- **Lab Environment**: Shared Spark clusters for team
- **Cross-Institutional**: Distributed analysis on shared datasets

## Success Metrics
- EMR clusters launch and connect to CloudWorkstation
- Jupyter notebooks access Spark seamlessly
- Cost-effective cluster auto-scaling

## Related
- Phase 5.5: AWS Research Services Integration
- v0.5.5 milestone" \
  --milestone "Phase 5.5: AWS Research Services" \
  --label "enhancement,area: aws,priority: medium,phase: 5.5-aws-services"

gh issue create \
  --repo "$REPO" \
  --title "[AWS] Amazon Braket Integration for Quantum Computing" \
  --body "## Summary
Enable quantum computing research via Amazon Braket integration.

## Motivation
Quantum computing is increasingly important in academic research. Braket provides simulator and hardware access.

## Implementation Tasks
- [ ] Braket notebook environment template
- [ ] Quantum algorithm development workflow
- [ ] Simulator and QPU access management
- [ ] Cost tracking for quantum executions
- [ ] CLI: \`cws launch braket-quantum quantum-research\`
- [ ] GUI quantum job monitoring

## Persona Impact
- **Solo Researcher**: Quantum algorithm development and testing
- **University Class**: Quantum computing education
- **Conference Workshop**: Hands-on quantum computing tutorials

## Success Metrics
- Braket notebooks launch with simulator access
- Clear cost attribution for quantum executions
- Example quantum algorithms included in templates

## Related
- Phase 5.5: AWS Research Services Integration
- v0.5.5 milestone" \
  --milestone "Phase 5.5: AWS Research Services" \
  --label "enhancement,area: aws,priority: low,phase: 5.5-aws-services"

gh issue create \
  --repo "$REPO" \
  --title "[AWS] SageMaker Studio Lab Integration (Pending Partnership)" \
  --body "## Summary
Investigate SageMaker Studio Lab integration for educational ML use cases.

## Motivation
SageMaker Studio Lab offers free ML notebooks for education. Potential integration with CloudWorkstation.

## Research Tasks
- [ ] AWS partnership feasibility assessment
- [ ] SageMaker Studio Lab vs CloudWorkstation comparison
- [ ] Integration architecture design
- [ ] Cost-benefit analysis for institutional deployments
- [ ] Pilot program design

## Risk Assessment
⚠️ **STRATEGIC**: Full integration depends on AWS partnership. May not be feasible.

## Persona Impact
- **University Class**: Free ML notebooks for students
- **Conference Workshop**: Zero-setup ML tutorials

## Success Metrics
- Partnership feasibility determined
- Clear integration roadmap (if feasible)
- Alternative approaches identified (if not feasible)

## Related
- Phase 5.5: AWS Research Services Integration
- v0.5.5 milestone
- Strategic partnership evaluation" \
  --milestone "Phase 5.5: AWS Research Services" \
  --label "enhancement,area: aws,priority: low,phase: 5.5-aws-services"

echo "✅ Phase 5.5 issues created"
echo ""

# Phase 5.6: Template Provisioning Enhancements
echo "📦 Creating Phase 5.6: Template Provisioning Enhancements issues..."

gh issue create \
  --repo "$REPO" \
  --title "[Templates] SSM File Operations for Large File Transfer" \
  --body "## Summary
Use AWS Systems Manager with S3-backed file operations for transferring large files during template provisioning.

## Motivation
Current provisioning scripts struggle with multi-GB files. SSM with S3 backend enables reliable large file transfer.

## Implementation Tasks
- [ ] S3-backed SSM file operations
- [ ] Progress reporting for large file transfers
- [ ] Resume support for interrupted transfers
- [ ] Parallel multi-file transfer
- [ ] Template schema: \`large_files\` section with S3 URIs
- [ ] CLI progress display during provisioning

## Persona Impact
- **Solo Researcher**: Large dataset downloads during instance setup
- **Lab Environment**: Pre-populated reference genomes, models
- **Conference Workshop**: Pre-loaded datasets for attendees

## Success Metrics
- Multi-GB file transfers complete reliably
- Clear progress indication during provisioning
- Failed transfers can resume automatically

## Related
- Phase 5.6: Template Provisioning Enhancements
- v0.5.6 milestone" \
  --milestone "Phase 5.6: Template Provisioning" \
  --label "enhancement,area: templates,priority: medium,phase: 5.6-provisioning"

gh issue create \
  --repo "$REPO" \
  --title "[Templates] Template Asset Management System" \
  --body "## Summary
Centralized system for managing template assets (binaries, configs, datasets) with versioning and distribution.

## Motivation
Template creators need an easy way to package and distribute large assets without embedding them in YAML.

## Implementation Tasks
- [ ] Asset repository (S3-backed with CDN)
- [ ] Asset versioning and lifecycle management
- [ ] Template asset references: \`assets://asset-name/version\`
- [ ] Asset publishing workflow for template authors
- [ ] Asset caching on instances
- [ ] CLI: \`cws template asset publish my-dataset.tar.gz\`

## Persona Impact
- **Template Authors**: Easy asset distribution
- **Solo Researcher**: Fast template provisioning with cached assets
- **University Class**: Shared asset repository for course templates

## Success Metrics
- Assets referenced cleanly in template YAML
- Fast asset downloads with CDN distribution
- Clear versioning prevents template breakage

## Related
- Phase 5.6: Template Provisioning Enhancements
- v0.5.6 milestone" \
  --milestone "Phase 5.6: Template Provisioning" \
  --label "enhancement,area: templates,priority: medium,phase: 5.6-provisioning"

echo "✅ Phase 5.6 issues created"
echo ""

# Phase 6.0: Enterprise Authentication & Security
echo "📦 Creating Phase 6.0: Enterprise Authentication & Security issues..."

gh issue create \
  --repo "$REPO" \
  --title "[Auth] OAuth/OIDC Integration for Institutional SSO" \
  --body "## Summary
Support OAuth 2.0 and OpenID Connect for institutional single sign-on (Google, Microsoft, Okta).

## Motivation
Universities require SSO integration with existing identity providers for seamless authentication.

## Implementation Tasks
- [ ] OAuth 2.0 / OIDC authentication flow
- [ ] Provider configuration (Google, Microsoft, Okta, generic OIDC)
- [ ] Token validation and refresh
- [ ] User attribute mapping (email, groups, roles)
- [ ] CLI login flow with browser redirect
- [ ] GUI login with institutional provider selection

## Persona Impact
- **University Class**: Students log in with university credentials
- **Lab Environment**: Team uses institutional SSO
- **Cross-Institutional**: Multiple SSO providers supported

## Success Metrics
- Users authenticate with institutional credentials
- No separate CloudWorkstation password needed
- Token refresh maintains session seamlessly

## Related
- Phase 6.0: Enterprise Authentication & Security
- v0.6.0 milestone" \
  --milestone "Phase 6.0: Authentication" \
  --label "enhancement,area: auth,security,priority: high,phase: 6.0-auth"

gh issue create \
  --repo "$REPO" \
  --title "[Auth] LDAP/Active Directory Integration" \
  --body "## Summary
Support enterprise LDAP and Active Directory authentication for on-premise environments.

## Motivation
Many institutions use LDAP/AD for centralized authentication and need CloudWorkstation integration.

## Implementation Tasks
- [ ] LDAP authentication support
- [ ] Active Directory integration
- [ ] Group membership synchronization
- [ ] Nested group support
- [ ] TLS/SSL for secure LDAP connections
- [ ] Configuration: \`cws admin auth configure --type ldap --server ldap.university.edu\`

## Persona Impact
- **University Class**: Campus-wide authentication
- **Lab Environment**: Department LDAP integration
- **Cross-Institutional**: Multi-institution LDAP support

## Success Metrics
- LDAP users authenticate successfully
- Group memberships sync for access control
- Secure connections protect credentials

## Related
- Phase 6.0: Enterprise Authentication & Security
- v0.6.0 milestone" \
  --milestone "Phase 6.0: Authentication" \
  --label "enhancement,area: auth,security,priority: high,phase: 6.0-auth"

gh issue create \
  --repo "$REPO" \
  --title "[Auth] SAML Support for Federated SSO" \
  --body "## Summary
SAML 2.0 support for enterprise federated single sign-on.

## Motivation
Enterprise and government institutions commonly use SAML for federated authentication.

## Implementation Tasks
- [ ] SAML 2.0 authentication flow
- [ ] IdP metadata configuration
- [ ] Assertion validation and parsing
- [ ] Attribute mapping (NameID, groups, roles)
- [ ] SP-initiated and IdP-initiated flows
- [ ] GUI SAML provider configuration interface

## Persona Impact
- **Cross-Institutional**: Federated authentication across institutions
- **University Class**: Enterprise SSO for large universities
- **Lab Environment**: Government lab SAML integration

## Success Metrics
- SAML authentication completes successfully
- Multiple IdPs supported simultaneously
- Clear error messages for configuration issues

## Related
- Phase 6.0: Enterprise Authentication & Security
- v0.6.0 milestone" \
  --milestone "Phase 6.0: Authentication" \
  --label "enhancement,area: auth,security,priority: medium,phase: 6.0-auth"

gh issue create \
  --repo "$REPO" \
  --title "[Security] IAM Profile Validation Pre-Launch" \
  --body "## Summary
Validate IAM instance profiles before launch to catch permission issues early.

## Motivation
Instance launch failures due to IAM issues waste time and create frustration. Early validation prevents this.

## Implementation Tasks
- [ ] IAM profile validation checks
- [ ] Required permission enumeration per template
- [ ] Pre-launch permission verification
- [ ] Clear error messages with fix suggestions
- [ ] CLI: \`cws launch python-ml test --validate-iam\`
- [ ] GUI validation indicator before launch

## Persona Impact
- **Solo Researcher**: Catch IAM issues before launch
- **University Class**: Clear guidance for students on IAM setup
- **Lab Environment**: Validate institutional IAM policies

## Success Metrics
- IAM issues detected before instance launch
- Clear remediation steps provided
- 50% reduction in IAM-related launch failures

## Related
- Phase 6.0: Enterprise Authentication & Security
- v0.6.0 milestone" \
  --milestone "Phase 6.0: Authentication" \
  --label "enhancement,area: aws,security,priority: high,phase: 6.0-auth"

gh issue create \
  --repo "$REPO" \
  --title "[Security] Role-Based Access Control (RBAC) System" \
  --body "## Summary
Comprehensive RBAC system for multi-tenant deployments with fine-grained permissions.

## Motivation
Institutions need granular control over who can do what in CloudWorkstation.

## Implementation Tasks
- [ ] Role definition system (admin, user, viewer, etc.)
- [ ] Permission matrix (launch, stop, delete, view costs, etc.)
- [ ] User-role assignment
- [ ] Resource-level permissions (project-specific access)
- [ ] Role inheritance and composition
- [ ] CLI: \`cws admin rbac role create researcher --allow launch,stop,connect\`

## Persona Impact
- **University Class**: Instructors = admin, students = limited user
- **Lab Environment**: PI = admin, postdocs = user, undergrads = viewer
- **Cross-Institutional**: Institution-specific role hierarchies

## Success Metrics
- Roles prevent unauthorized actions
- Clear error messages guide users
- Audit logs show who did what

## Related
- Phase 6.0: Enterprise Authentication & Security
- v0.6.0 milestone" \
  --milestone "Phase 6.0: Authentication" \
  --label "enhancement,area: auth,security,priority: high,phase: 6.0-auth"

echo "✅ Phase 6.0 issues created"
echo ""

# Phase 6.1: TUI Feature Completeness
echo "📦 Creating Phase 6.1: TUI Feature Completeness issues..."

gh issue create \
  --repo "$REPO" \
  --title "[TUI] Project Member Management Interface" \
  --body "## Summary
Full project member management in TUI with paginated lists, add/remove dialogs, and role editing.

## Motivation
TUI users need complete project management capabilities without switching to CLI/GUI.

## Implementation Tasks
- [ ] Paginated member list view
- [ ] Add member dialog with email/role selection
- [ ] Remove member confirmation dialog
- [ ] Role change interface
- [ ] Member search/filter
- [ ] Keyboard shortcuts for common operations

## Persona Impact
- **Lab Environment**: Terminal-based project management
- **Solo Researcher**: Quick member additions from terminal

## Success Metrics
- Complete project member CRUD via TUI
- Smooth keyboard-driven navigation
- Feature parity with GUI

## Related
- Phase 6.1: TUI Feature Completeness
- v0.6.1 milestone" \
  --milestone "Phase 6.1: TUI Completeness" \
  --label "enhancement,area: tui,priority: medium,phase: 6.1-tui"

gh issue create \
  --repo "$REPO" \
  --title "[TUI] Project Instance Filtering and Management" \
  --body "## Summary
Filter instance lists by project and perform project-specific instance actions in TUI.

## Motivation
Users working within projects need focused views of project instances.

## Implementation Tasks
- [ ] Project filter dropdown in instance list
- [ ] Project-specific instance actions
- [ ] Visual indication of instance project membership
- [ ] Multi-instance operations within project
- [ ] Project cost summary in instance view

## Persona Impact
- **Lab Environment**: View only team project instances
- **University Class**: Instructors filter student instances by course project

## Success Metrics
- Clean project-filtered instance views
- Obvious project context in TUI
- Fast project switching

## Related
- Phase 6.1: TUI Feature Completeness
- v0.6.1 milestone" \
  --milestone "Phase 6.1: TUI Completeness" \
  --label "enhancement,area: tui,priority: medium,phase: 6.1-tui"

gh issue create \
  --repo "$REPO" \
  --title "[TUI] Cost Breakdown Visualization" \
  --body "## Summary
Terminal-based cost charts and breakdowns showing service-level spending.

## Motivation
TUI users need cost visibility without switching to GUI.

## Implementation Tasks
- [ ] ASCII/Unicode chart rendering
- [ ] Service-level cost breakdown (EC2, EBS, EFS, etc.)
- [ ] Project cost comparison charts
- [ ] Time-series cost trends
- [ ] Interactive drill-down into cost categories

## Persona Impact
- **Lab Environment**: Team cost monitoring from terminal
- **Solo Researcher**: Quick cost checks during research

## Success Metrics
- Clear cost visualizations in terminal
- Interactive exploration of cost breakdowns
- Comparable clarity to GUI charts

## Related
- Phase 6.1: TUI Feature Completeness
- v0.6.1 milestone" \
  --milestone "Phase 6.1: TUI Completeness" \
  --label "enhancement,area: tui,priority: medium,phase: 6.1-tui"

gh issue create \
  --repo "$REPO" \
  --title "[TUI] Hibernation Savings Display" \
  --body "## Summary
Display hibernation savings trends and forecasts in TUI.

## Motivation
TUI users need visibility into cost optimization benefits from hibernation.

## Implementation Tasks
- [ ] Hibernation savings summary
- [ ] Savings trend charts (ASCII/Unicode)
- [ ] Projected monthly savings
- [ ] Hibernation policy effectiveness metrics
- [ ] Comparison: with hibernation vs without

## Persona Impact
- **Lab Environment**: Track team hibernation cost savings
- **Solo Researcher**: Validate hibernation effectiveness

## Success Metrics
- Clear savings visualization in terminal
- Motivates hibernation policy adoption
- Accurate savings calculations

## Related
- Phase 6.1: TUI Feature Completeness
- v0.6.1 milestone" \
  --milestone "Phase 6.1: TUI Completeness" \
  --label "enhancement,area: tui,priority: low,phase: 6.1-tui"

echo "✅ Phase 6.1 issues created"
echo ""

echo "🎉 All phase issues created successfully!"
echo ""
echo "📊 Summary:"
echo "  - Phase 5.3: Advanced Storage Integration (3 issues)"
echo "  - Phase 5.4: Policy Framework Enhancement (3 issues)"
echo "  - Phase 5.5: AWS Research Services Integration (3 issues)"
echo "  - Phase 5.6: Template Provisioning Enhancements (2 issues)"
echo "  - Phase 6.0: Enterprise Authentication & Security (5 issues)"
echo "  - Phase 6.1: TUI Feature Completeness (4 issues)"
echo ""
echo "Total: 20 new issues created"
echo ""
echo "View all issues: https://github.com/$REPO/issues"
echo "View project board: https://github.com/users/scttfrdmn/projects/2"
