# Template Signing & Verification Roadmap (Cosign)

**Status**: Planned
**Related**: Template Security, Supply Chain Security, Trust Framework
**Dependencies**: Sigstore Cosign v2.x, Rekor, Fulcio

## Executive Summary

Implement comprehensive template signing and verification using **Sigstore's Cosign** - the industry-standard tool for signing, verifying, and protecting software artifacts. This provides researchers with confidence that templates are legitimate, tested, and safe to use.

**Goal**: Establish chain of trust from template creation → signing → testing → distribution → verification → installation

**Technology**: Cosign (not GPG) - the cloud-native standard used by Kubernetes, Docker, Helm, and other CNCF projects

---

## Why Cosign over GPG?

| Feature | Cosign | GPG |
|---------|--------|-----|
| **Key Management** | Keyless (OIDC) or keys | Keys only |
| **Certificate Authority** | Fulcio (automatic) | Manual/self-signed |
| **Transparency Log** | Rekor (public audit) | None |
| **Cloud Native** | ✅ Designed for it | ❌ Pre-cloud era |
| **Short-lived Certs** | ✅ Yes (10 min default) | ❌ No |
| **Supply Chain** | ✅ In-toto attestations | ❌ Limited |
| **Industry Adoption** | Kubernetes, Docker, Helm | Legacy systems |
| **Learning Curve** | Low | High |

**Decision**: Use Cosign for modern, cloud-native security with minimal key management burden.

---

## Phase 1: Cosign Integration (v0.6.1)

**Estimated Effort**: 2-3 weeks
**Business Value**: CRITICAL (trust & security foundation)
**Target Release**: v0.6.1

### Core Features

#### 1. Keyless Signing (Default - Recommended)

Uses OIDC authentication (GitHub, Google, etc.) - no keys to manage!

**Signing Flow**:
```bash
$ cws templates sign ubuntu-24.04-server.yml

🔐 Signing template with Cosign...
🌐 Authenticating with GitHub (OIDC)...
✅ Signed by: team@cloudworkstation.dev (GitHub)
📝 Certificate stored in Rekor transparency log
🔗 Rekor entry: https://rekor.sigstore.dev/api/v1/log/entries/abc123...

Signature: templates/.signatures/ubuntu-24-04-server.yml.sig
Certificate: templates/.signatures/ubuntu-24-04-server.yml.cert
```

**Verification Flow**:
```bash
$ cws templates verify ubuntu-24.04-server.yml

🔍 Verifying template signature...
✅ Valid signature from team@cloudworkstation.dev
✅ Certificate verified via Fulcio CA
✅ Timestamp verified via Rekor transparency log
📅 Signed: 2025-10-18 01:23:45 UTC
🔗 Rekor entry: https://rekor.sigstore.dev/...

Template is verified and safe to use.
```

**Benefits**:
- No key management required
- Identity verified via OIDC provider
- Automatic certificate issuance
- Public audit trail in Rekor
- Expires in 10 minutes (prevents long-term key compromise)

#### 2. Key-Based Signing (Enterprise/Institutional)

For organizations that want key custody:

```bash
# One-time setup: Generate key pair
$ cosign generate-key-pair

Enter password for private key:
Private key written to cosign.key
Public key written to cosign.pub

# Sign template
$ cws templates sign ubuntu-24.04-server.yml --key stanford.key

Enter password for private key:
✅ Signed with key: stanford.key
🔒 Public key: stanford.pub (distribute to users)

# Verify with public key
$ cws templates verify ubuntu-24.04-server.yml --key stanford.pub

✅ Valid signature from Stanford University
✅ Template verified
```

**Use Cases**:
- Institutional policies requiring key custody
- Air-gapped environments (no internet for OIDC)
- Long-term signatures (years)
- Regulatory compliance

### Implementation Details

**Dependencies**:
```go
// go.mod
require (
    github.com/sigstore/cosign/v2 v2.2.3
    github.com/sigstore/rekor v1.3.4
    github.com/in-toto/in-toto-golang v0.9.0
    github.com/slsa-framework/slsa-verifier/v2 v2.5.1
)
```

**Code Structure**:
```
pkg/templates/
├── signing/
│   ├── cosign.go           # Cosign wrapper
│   ├── keyless.go          # OIDC keyless signing
│   ├── keybased.go         # Key-based signing
│   ├── verifier.go         # Signature verification
│   └── rekor.go            # Rekor transparency log
└── policy/
    ├── enforcement.go      # Policy engine
    └── trust.go            # Trust hierarchy
```

**Signature Storage**:
```
templates/
├── ubuntu-24.04-server.yml          # Template
└── .signatures/
    ├── ubuntu-24.04-server.yml.sig  # Cosign signature
    └── ubuntu-24.04-server.yml.cert # Certificate (keyless)
```

**CLI Commands**:
```bash
# Sign (keyless)
cws templates sign <template.yml>

# Sign (with key)
cws templates sign <template.yml> --key <private-key>

# Verify (keyless)
cws templates verify <template.yml>

# Verify (with key)
cws templates verify <template.yml> --key <public-key>

# Verify with identity constraint
cws templates verify <template.yml> \
    --certificate-identity team@cloudworkstation.dev \
    --certificate-oidc-issuer https://github.com/login/oauth
```

**Daemon API**:
```go
// POST /api/v1/templates/sign
type SignTemplateRequest struct {
    TemplatePath string `json:"template_path"`
    KeyPath      string `json:"key_path,omitempty"`  // Optional for key-based
    Keyless      bool   `json:"keyless"`             // Default: true
}

// POST /api/v1/templates/verify
type VerifyTemplateRequest struct {
    TemplatePath       string `json:"template_path"`
    KeyPath            string `json:"key_path,omitempty"`
    CertificateIdentity string `json:"certificate_identity,omitempty"`
    OIDCIssuer         string `json:"oidc_issuer,omitempty"`
}

type VerifyTemplateResponse struct {
    Valid            bool      `json:"valid"`
    Signer           string    `json:"signer"`
    SignedAt         time.Time `json:"signed_at"`
    RekorEntry       string    `json:"rekor_entry"`
    CertificateChain []string  `json:"certificate_chain"`
}
```

---

## Phase 2: In-Toto Attestations (v0.6.2)

**Estimated Effort**: 2-3 weeks
**Business Value**: HIGH (supply chain security)
**Target Release**: v0.6.2

### Supply Chain Security with SLSA

**In-toto** attestations provide signed statements about the build/test process.

**SLSA Provenance Attestation**:
```json
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "subject": [{
    "name": "ubuntu-24-04-server.yml",
    "digest": {"sha256": "abc123..."}
  }],
  "predicateType": "https://slsa.dev/provenance/v0.2",
  "predicate": {
    "builder": {
      "id": "https://github.com/scttfrdmn/cloudworkstation"
    },
    "buildType": "https://cloudworkstation.dev/template-build/v1",
    "invocation": {
      "configSource": {
        "uri": "git+https://github.com/scttfrdmn/cloudworkstation",
        "digest": {"sha1": "c5f84ed5"},
        "entryPoint": "templates/ubuntu-24-04-server.yml"
      }
    },
    "metadata": {
      "buildStartedOn": "2025-10-18T00:00:00Z",
      "buildFinishedOn": "2025-10-18T00:10:00Z",
      "completeness": {
        "parameters": true,
        "environment": true,
        "materials": true
      },
      "reproducible": true
    },
    "materials": [{
      "uri": "git+https://github.com/canonical/ubuntu-ami",
      "digest": {"sha256": "ami-abc123"}
    }]
  }
}
```

**Test Results Attestation**:
```json
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "subject": [{
    "name": "ubuntu-24-04-server.yml",
    "digest": {"sha256": "abc123..."}
  }],
  "predicateType": "https://cloudworkstation.dev/test-results/v1",
  "predicate": {
    "test_framework": "cloudworkstation-validator",
    "test_run": {
      "started_at": "2025-10-18T00:00:00Z",
      "finished_at": "2025-10-18T00:10:00Z",
      "duration_seconds": 600
    },
    "results": {
      "total": 47,
      "passed": 47,
      "failed": 0,
      "skipped": 0
    },
    "tests": [
      {
        "name": "Package installation",
        "result": "passed",
        "duration_ms": 5000
      },
      {
        "name": "User creation",
        "result": "passed",
        "duration_ms": 1000
      }
    ]
  }
}
```

**Security Scan Attestation**:
```json
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "subject": [{
    "name": "ubuntu-24-04-server.yml",
    "digest": {"sha256": "abc123..."}
  }],
  "predicateType": "https://cloudworkstation.dev/security-scan/v1",
  "predicate": {
    "scanner": {
      "name": "trivy",
      "version": "0.50.0",
      "uri": "https://github.com/aquasecurity/trivy"
    },
    "scan_time": "2025-10-18T00:00:00Z",
    "vulnerabilities": {
      "critical": 0,
      "high": 0,
      "medium": 2,
      "low": 5,
      "total": 7
    },
    "scan_result": "passed",
    "report_url": "https://scans.cloudworkstation.dev/..."
  }
}
```

**CLI Commands**:
```bash
# Create SLSA provenance attestation
$ cws templates attest ubuntu-24-04-server.yml \
    --type slsa-provenance \
    --predicate provenance.json

✅ SLSA provenance attestation signed and stored

# Create test results attestation
$ cws templates attest ubuntu-24-04-server.yml \
    --type test-results \
    --predicate test-results.json

✅ Test results attestation signed and stored

# Create security scan attestation
$ cws templates attest ubuntu-24-04-server.yml \
    --type security-scan \
    --predicate security-scan.json

✅ Security scan attestation signed and stored

# Verify all attestations
$ cws templates verify-attestation ubuntu-24-04-server.yml

✅ SLSA Provenance verified
   Built by: GitHub Actions
   Source: github.com/scttfrdmn/cloudworkstation@c5f84ed5
   Build time: 10m 23s
   Reproducible: Yes

✅ Test Results verified
   Tests passed: 47/47
   Duration: 10m
   Framework: cloudworkstation-validator

✅ Security Scan verified
   Scanner: Trivy v0.50.0
   Vulnerabilities: 0 critical, 0 high, 2 medium, 5 low
   Result: PASSED
```

**Badge System**:

Templates earn badges based on attestations:

- ✅ **Signed**: Valid Cosign signature
- 🧪 **Tested**: Test results attestation
- 🔒 **Scanned**: Security scan attestation
- 🏆 **SLSA L3**: Highest supply chain security level
- 🏢 **Institution Verified**: Signed by verified institution

**Display in CLI**:
```bash
$ cws templates

🏗️  Ubuntu 24.04 Server [✅🧪🔒🏆]
    Slug: ubuntu-24-04-server
    Signed by: CloudWorkstation Team
    SLSA Level: 3 (highest)
    Tests: 47/47 passed
    Security: 0 critical issues
    Last tested: 2 hours ago
```

---

## Phase 3: Policy Enforcement (v0.6.3)

**Estimated Effort**: 2 weeks
**Business Value**: CRITICAL (institutional compliance)
**Target Release**: v0.6.3

### Admission Control for Templates

**Policy Configuration** (`~/.cloudworkstation/policies/signing.yml`):
```yaml
signature_policy:
  # Enforcement level
  enforcement: strict  # strict | warn | permissive

  # Keyless signatures (OIDC-based)
  keyless:
    enabled: true
    allowed_identities:
      - "team@cloudworkstation.dev"
      - "*@stanford.edu"          # Any Stanford email
      - "*@mit.edu"
      - "*@berkeley.edu"
    required_oidc_issuer: "https://github.com/login/oauth"

  # Key-based signatures
  key_based:
    enabled: true
    trusted_keys:
      - path: "~/.cloudworkstation/keys/cloudworkstation-team.pub"
        name: "CloudWorkstation Team"
      - path: "~/.cloudworkstation/keys/stanford.pub"
        name: "Stanford Research Computing"
      - path: "~/.cloudworkstation/keys/mit.pub"
        name: "MIT CSAIL"

  # Attestation requirements
  attestations:
    require_slsa_provenance: true
    minimum_slsa_level: 2          # 0-3
    require_test_results: true
    require_security_scan: true
    max_vulnerabilities:
      critical: 0
      high: 0
      medium: 5
      low: 10

  # Rekor transparency
  rekor:
    require_rekor_entry: true
    max_age_days: 90               # Reject signatures older than 90 days
    verify_checkpoint: true

  # Fallback behavior
  fallback:
    unsigned_templates: reject     # reject | warn | allow
    expired_signatures: reject
    failed_attestations: reject
```

**Policy Presets**:
```yaml
# Research institution preset (strict)
presets:
  research-strict:
    enforcement: strict
    require_slsa_provenance: true
    minimum_slsa_level: 3
    require_test_results: true
    require_security_scan: true
    unsigned_templates: reject

  # Development preset (permissive)
  development:
    enforcement: warn
    require_slsa_provenance: false
    unsigned_templates: allow

  # Production preset (balanced)
  production:
    enforcement: strict
    require_slsa_provenance: true
    minimum_slsa_level: 2
    unsigned_templates: reject
    max_age_days: 30
```

**CLI Commands**:
```bash
# Set policy preset
$ cws admin policy set template-signing research-strict

✅ Policy updated: Research-strict mode
   - SLSA Level 3 required
   - Test results required
   - Security scan required
   - Unsigned templates rejected

# Custom policy
$ cws admin policy set template-signing strict \
    --require-attestations \
    --min-slsa-level 2 \
    --max-signature-age 90

✅ Policy updated

# Check policy compliance
$ cws templates install python-ml-workstation

🔍 Checking signature policy...
✅ Template signed by team@cloudworkstation.dev
✅ SLSA provenance verified (Level 3)
✅ Test results verified (47/47 passed)
✅ Security scan verified (0 critical issues)
💾 Installing template...

# Policy violation example
$ cws templates install community-experimental

🔍 Checking signature policy...
❌ Template is not signed
❌ Policy requires signed templates (strict mode)
💡 Override with: --allow-unsigned (not recommended)

Error: Template rejected by policy
```

**Trust Hierarchy**:
```
CloudWorkstation Team (Root Trust)
├─ Stanford University
│  ├─ Research Computing
│  └─ Computer Science Dept
├─ MIT
│  ├─ CSAIL
│  └─ Media Lab
├─ UC Berkeley
│  └─ EECS
└─ Community Verified Publishers
   ├─ Individual (100+ templates, verified)
   └─ Research Labs Inc (50+ templates)
```

**Policy API**:
```go
// GET /api/v1/policies/signing
type SigningPolicy struct {
    Enforcement    string   `json:"enforcement"`
    KeylessConfig  KeylessConfig  `json:"keyless"`
    KeyBasedConfig KeyBasedConfig `json:"key_based"`
    Attestations   AttestationPolicy `json:"attestations"`
    Rekor          RekorPolicy `json:"rekor"`
}

// PUT /api/v1/policies/signing
// POST /api/v1/templates/{name}/verify
type VerifyRequest struct {
    EnforcePolicy bool `json:"enforce_policy"`
}
```

---

## Phase 4: CI/CD Integration (v0.6.4)

**Estimated Effort**: 1-2 weeks
**Business Value**: HIGH (automation)
**Target Release**: v0.6.4

### GitHub Actions Workflow

**Automated Template Signing**:
```yaml
# .github/workflows/sign-templates.yml
name: Sign Templates

on:
  push:
    branches: [main]
    paths:
      - 'templates/**/*.yml'

permissions:
  id-token: write  # Required for OIDC
  contents: write
  packages: write

jobs:
  sign-templates:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Cosign
        uses: sigstore/cosign-installer@v3

      - name: Sign Templates (Keyless OIDC)
        run: |
          for template in templates/*.yml; do
            echo "🔐 Signing $template..."
            cosign sign-blob $template \
              --bundle ${template}.bundle \
              --yes  # Auto-approve keyless signing
          done

      - name: Generate SLSA Provenance
        uses: slsa-framework/slsa-github-generator@v1
        with:
          artifact-path: templates/

      - name: Run Template Tests
        run: |
          make build
          ./bin/cws templates validate > test-results.json

      - name: Attest Test Results
        run: |
          for template in templates/*.yml; do
            cosign attest-blob $template \
              --predicate test-results.json \
              --type https://cloudworkstation.dev/test-results/v1
          done

      - name: Security Scan
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: 'templates/'
          format: 'json'
          output: 'security-scan.json'

      - name: Attest Security Scan
        run: |
          for template in templates/*.yml; do
            cosign attest-blob $template \
              --predicate security-scan.json \
              --type https://cloudworkstation.dev/security-scan/v1
          done

      - name: Commit Signatures
        run: |
          git config user.name "CloudWorkstation Bot"
          git config user.email "bot@cloudworkstation.dev"
          git add templates/.signatures/
          git commit -m "chore: sign templates [skip ci]"
          git push
```

**Benefits**:
- Automatic signing on every commit
- SLSA provenance generation
- Test result attestations
- Security scan attestations
- All stored in Rekor transparency log

---

## Phase 5: OCI Registry Distribution (v0.6.5)

**Estimated Effort**: 2-3 weeks
**Business Value**: HIGH (industry standard)
**Target Release**: v0.6.5

### Template Registry with Cosign

Templates stored in OCI registries (Docker Hub, GitHub Container Registry, etc.):

**Push Template to OCI Registry**:
```bash
$ cws templates push ubuntu-24.04-server.yml \
    ghcr.io/cloudworkstation/templates/ubuntu-24-04-server:latest

📦 Pushing to ghcr.io...
✅ Pushed: ghcr.io/cloudworkstation/templates/ubuntu-24-04-server:latest
```

**Sign OCI Artifact**:
```bash
$ cosign sign ghcr.io/cloudworkstation/templates/ubuntu-24-04-server:latest

🔐 Signing OCI image...
🌐 Authenticating with GitHub (OIDC)...
✅ Signed by: team@cloudworkstation.dev
📝 Signature stored in OCI registry
```

**Pull and Verify**:
```bash
$ cws templates pull ghcr.io/cloudworkstation/templates/ubuntu-24-04-server:latest

🔍 Verifying signature...
✅ Valid signature from team@cloudworkstation.dev
✅ SLSA provenance verified
📥 Pulling template...
✅ Installed: ubuntu-24-04-server
```

**Benefits**:
- Industry-standard distribution (OCI registries)
- Automatic signature verification on pull
- Immutable artifacts (content-addressable)
- Built-in versioning and tagging
- Bandwidth optimization (layers, caching)

---

## User Experience

### Progressive Trust Model

**Level 0 - Unsigned** (Community templates):
```
⚠️  Python Experimental Template
    Not signed - use at your own risk
    Created by: community-user-123
    [Install anyway] [Cancel]
```

**Level 1 - Signed** (Keyless OIDC):
```
🔒 Ubuntu 24.04 Server
    ✅ Signed by team@cloudworkstation.dev
    📅 Signed: 2 days ago
    [Install] [Cancel]
```

**Level 2 - Signed + Tests**:
```
🔒 Ubuntu 24.04 Server
    ✅ Signed by team@cloudworkstation.dev
    🧪 Tests passed: 47/47
    📅 Signed: 2 days ago
    [Install] [Cancel]
```

**Level 3 - SLSA L2+**:
```
🔒 Ubuntu 24.04 Server
    ✅ Signed by team@cloudworkstation.dev
    🏆 SLSA Level 3 (highest)
    🧪 Tests passed: 47/47
    🔍 Security scan: 0 critical issues
    📅 Built: 2 days ago by GitHub Actions
    [Install] [View Provenance] [Cancel]
```

**Level 4 - Institution Verified**:
```
🔒🏢 Stanford Python ML Template
    ✅ Signed by research-computing@stanford.edu
    🏢 Verified by Stanford University
    🏆 SLSA Level 3
    🧪 Tests passed: 52/52
    🔍 Security scan: 0 issues
    📊 Used by 1,247 researchers
    📅 Last updated: 1 week ago
    [Install] [View Provenance] [Cancel]
```

---

## Security Considerations

### Key Storage

- **macOS**: Keychain
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Windows**: Credential Manager
- **Environment Variable**: `COSIGN_PASSWORD` (CI/CD only)

### Key Rotation

Support key expiration and rotation:
```bash
$ cws admin keys rotate --old-key old.key --new-key new.key

🔑 Rotating signing key...
✅ Re-signing all templates with new key
✅ 47 templates re-signed
⚠️  Old key should be revoked
💡 Run: cws admin keys revoke --key old.key
```

### Revocation

Certificate Revocation List (CRL) for compromised keys:
```bash
$ cws admin keys revoke --key compromised.key --reason "Key leaked"

⚠️  Revoking key: compromised.key
✅ Key revoked in Rekor
✅ All templates signed with this key marked as invalid
💡 Users will be warned when encountering templates signed with this key
```

### Offline Verification

Cache signatures for offline use:
```yaml
cache:
  enabled: true
  directory: "~/.cloudworkstation/signature-cache"
  ttl_days: 7
  max_size_mb: 100
```

### Reproducible Builds

Template signatures include build environment hash for reproducibility.

---

## Implementation Checklist

### Phase 1 (v0.6.1)
- [ ] Integrate Cosign Go SDK
- [ ] Implement keyless signing (OIDC)
- [ ] Implement key-based signing
- [ ] Implement signature verification
- [ ] Add CLI commands (sign, verify)
- [ ] Add daemon API endpoints
- [ ] Create signature storage (.signatures/)
- [ ] Rekor integration
- [ ] Documentation

### Phase 2 (v0.6.2)
- [ ] In-toto attestation framework
- [ ] SLSA provenance attestations
- [ ] Test result attestations
- [ ] Security scan attestations
- [ ] Attestation verification
- [ ] Badge system
- [ ] CLI attestation commands
- [ ] Documentation

### Phase 3 (v0.6.3)
- [ ] Policy engine implementation
- [ ] Policy configuration (YAML)
- [ ] Policy presets
- [ ] Trust hierarchy
- [ ] CLI policy commands
- [ ] Policy API endpoints
- [ ] Admission control
- [ ] Documentation

### Phase 4 (v0.6.4)
- [ ] GitHub Actions workflow
- [ ] Automated signing
- [ ] SLSA provenance generation
- [ ] Automated attestations
- [ ] CI/CD documentation
- [ ] Example workflows

### Phase 5 (v0.6.5)
- [ ] OCI registry integration
- [ ] Template push/pull
- [ ] OCI signature verification
- [ ] Registry authentication
- [ ] Multi-registry support
- [ ] Documentation

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Templates Signed | 100% core | Automated in CI/CD |
| SLSA Level | L3 for core | GitHub Actions provenance |
| Verification Time | < 2 seconds | Performance benchmark |
| Policy Compliance | 100% institutional | Policy engine |
| Community Adoption | 50%+ signed | Marketplace stats |
| Signature Cache Hit | > 90% | Cache metrics |

---

## Related Documents

- [Sigstore Documentation](https://docs.sigstore.dev/)
- [Cosign Documentation](https://docs.sigstore.dev/cosign/overview/)
- [SLSA Framework](https://slsa.dev/)
- [In-Toto Attestations](https://in-toto.io/)
- [OS Version Expansion Roadmap](./OS_VERSION_EXPANSION_ROADMAP.md)

---

**Last Updated**: 2025-10-18
**Status**: Planning Phase
**Next Milestone**: Cosign integration (v0.6.1)
