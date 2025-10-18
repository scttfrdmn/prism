# OS Version Matrix Expansion Roadmap

**Status**: Planned
**Related**: Template System, Base OS Support
**Dependencies**: Template parser (pkg/templates/parser.go)

## Overview

Following the successful implementation of the Ubuntu 24.04 template matrix (commit: c5f84ed5), this roadmap outlines the expansion of CloudWorkstation's OS support to provide comprehensive coverage of major Linux distributions across multiple versions.

**Current Status** (as of v0.5.3):
- ✅ Ubuntu 22.04 Server & Desktop
- ✅ Ubuntu 24.04 Server & Desktop
- ✅ Ubuntu 20.04 (partial support)
- ✅ Rocky Linux 9.x (server only)
- ✅ Amazon Linux 2023 (server only)

**Goal**: Become the most comprehensive cloud workstation platform for researchers by supporting all major Linux distributions across multiple versions.

---

## Implementation Pattern

Based on the Ubuntu 24.04 implementation, each new OS version requires:

### 1. Add AMI Mappings to Parser

**File**: `pkg/templates/parser.go`

**Location**: `getDefaultBaseAMIs()` function

**Example**:
```go
"rocky-8": {
    "us-east-1": {
        "x86_64": "ami-xxxxx",
        "arm64":  "ami-xxxxx",
    },
    "us-east-2": {
        "x86_64": "ami-xxxxx",
        "arm64":  "ami-xxxxx",
    },
    "us-west-1": {
        "x86_64": "ami-xxxxx",
        "arm64":  "ami-xxxxx",
    },
    "us-west-2": {
        "x86_64": "ami-xxxxx",
        "arm64":  "ami-xxxxx",
    },
},
```

### 2. Verify Tester Support

**File**: `pkg/templates/tester.go`

**Check**: Ensure base OS is in `supportedOS` map (line ~344)

```go
supportedOS := map[string]bool{
    "ubuntu-20.04":     true,
    "ubuntu-22.04":     true,
    "ubuntu-24.04":     true,
    "rocky-8":          true,  // Add new OS here
    "rocky-9":          true,
    "amazonlinux-2023": true,
    "ami-based":        true,
}
```

### 3. Create Template YAML Files

**Naming Convention**: `{os}-{major}-{minor}-{purpose}.yml`

**Server Template** (`templates/{os}-{version}-server.yml`):
```yaml
name: "{OS} {Version} Server"
slug: "{os}-{major}-{minor}-server"
description: "Minimal {OS} {version} server with essential development tools"
base: "{os}-{major}.{minor}"
connection_type: "ssh"
complexity: "simple"
category: "Base Systems"
domain: "base"
package_manager: "dnf"  # or apt, depending on distro
packages:
  system:
    - "build-essential"  # or equivalent
    - "curl"
    - "git"
    # ... distro-specific packages
users:
  - name: "{default-user}"
    groups: ["wheel"]  # or ["sudo"] for Debian-based
```

**Desktop Template** (`templates/{os}-{version}-desktop.yml`):
```yaml
name: "{OS} {Version} Desktop"
slug: "{os}-{major}-{minor}-desktop"
description: "{OS} {version} desktop environment with full GUI"
base: "{os}-{major}.{minor}"
connection_type: "dcv"
complexity: "simple"
category: "Desktop Environment"
domain: "desktop"
package_manager: "dnf"  # or apt
packages:
  system:
    - "gnome-desktop"  # or equivalent
    - "firefox"
    # ... desktop packages
services:
  - name: gdm
    enable: true
```

### 4. Test and Validate

```bash
# Rebuild binaries with new base OS support
make build

# Validate all templates
./bin/cws templates validate

# Verify templates are available
./bin/cws templates | grep "{os}.*{version}"
```

### 5. Commit Changes

```bash
git add pkg/templates/parser.go templates/{os}-*.yml
git commit -m "feat: Add {OS} {version} support"
git push origin main
```

---

## Priority 1: Rocky Linux Expansion

**Target Release**: v0.5.5
**Estimated Effort**: 4-6 hours
**Business Value**: HIGH (enterprise/academic demand)

### Target Matrix

| Template | Status | Priority |
|----------|--------|----------|
| Rocky Linux 8.10 Server | 🎯 Planned | HIGH |
| Rocky Linux 8.10 Desktop | 🎯 Planned | HIGH |
| Rocky Linux 9.6 Server | ✅ Have base | MEDIUM |
| Rocky Linux 9.6 Desktop | 🎯 Planned | MEDIUM |

### Benefits

- **Enterprise RHEL compatibility** without licensing costs
- **Popular in academic/research** institutions
- **Strong community support** and long-term maintenance
- **Extended support**: Rocky 8 until 2029, Rocky 9 until 2032

### Implementation Steps

1. **Find Rocky Linux AMIs**:
```bash
# Rocky 8.10
aws ec2 describe-images \
  --owners 679593333241 \
  --filters "Name=name,Values=Rocky-8.10-EC2-Base*" \
  --query 'Images[*].[ImageId,Name,Architecture]' \
  --region us-east-1

# Rocky 9.6
aws ec2 describe-images \
  --owners 679593333241 \
  --filters "Name=name,Values=Rocky-9*-EC2-Base*" \
  --query 'Images[*].[ImageId,Name,Architecture]' \
  --region us-east-1
```

2. **Add to parser.go**: `rocky-8` and `rocky-9` (update existing)

3. **Create templates**:
   - `templates/rocky-8-10-server.yml`
   - `templates/rocky-8-10-desktop.yml`
   - `templates/rocky-9-6-desktop.yml` (server already exists as `rocky-9`)

4. **Package Manager**: DNF (RHEL-compatible)

5. **Default User**: `rocky` (already standard)

---

## Priority 2: Amazon Linux & Alpine

**Target Release**: v0.5.6
**Estimated Effort**: 6-8 hours
**Business Value**: HIGH (AWS-native & lightweight)

### Target Matrix

| Template | Status | Priority |
|----------|--------|----------|
| Amazon Linux 2023 Server | ✅ Have | DONE |
| Amazon Linux 2 Server | 🎯 Planned | MEDIUM |
| Alpine Linux 3.20 Server | 🎯 Planned | HIGH |
| Alpine Linux 3.19 Server | 🎯 Planned | MEDIUM |

### Amazon Linux Benefits

- **Native AWS integration** and optimization
- **Optimized for EC2** instances
- **Free for AWS users**
- **AL2**: Still widely used, support until 2025
- **AL2023**: Modern, rolling release model

### Alpine Linux Benefits

- **Ultra-lightweight**: 5MB base image
- **Security-focused**: musl libc, minimal attack surface
- **Perfect for containers** and microservices research
- **Cost-effective**: Minimal resource usage
- **APK package manager**: Fast and efficient

### Implementation Steps

1. **Find AMIs**:
```bash
# Amazon Linux 2
aws ec2 describe-images \
  --owners amazon \
  --filters "Name=name,Values=amzn2-ami-hvm-*-x86_64-gp2" \
  --query 'Images[*].[ImageId,Name,CreationDate]' \
  --region us-east-1 | sort -k3 -r | head -5

# Alpine Linux
aws ec2 describe-images \
  --owners 538276064493 \
  --filters "Name=name,Values=alpine-3.20*" \
  --query 'Images[*].[ImageId,Name,Architecture]' \
  --region us-east-1
```

2. **Add to parser.go**: `amazonlinux-2`, `alpine-3.20`, `alpine-3.19`

3. **Create templates**:
   - `templates/amazon-linux-2-server.yml` (AL2)
   - `templates/alpine-3-20-server.yml`
   - `templates/alpine-3-19-server.yml`

4. **Package Managers**:
   - Amazon Linux: YUM (AL2), DNF (AL2023)
   - Alpine: APK

5. **Use Cases**:
   - AL2: Legacy workloads, compatibility
   - Alpine: Containerized research, microservices, security research

---

## Priority 3: Debian

**Target Release**: v0.6.0
**Estimated Effort**: 4-6 hours
**Business Value**: MEDIUM (Ubuntu's upstream)

### Target Matrix

| Template | Status | Priority |
|----------|--------|----------|
| Debian 12 "Bookworm" Server | 🎯 Planned | HIGH |
| Debian 12 Desktop | 🎯 Planned | MEDIUM |
| Debian 11 "Bullseye" Server | 🎯 Planned | LOW |

### Benefits

- **Pure Debian experience** (vs Ubuntu modifications)
- **Very stable** for production research
- **Popular in academic** environments
- **APT package manager** (same as Ubuntu)
- **Long support**: Debian 11 until 2026, Debian 12 until 2028

### Implementation Steps

1. **Find Debian AMIs**:
```bash
aws ec2 describe-images \
  --owners 136693071363 \
  --filters "Name=name,Values=debian-12-amd64-*" \
  --query 'Images[*].[ImageId,Name,Architecture]' \
  --region us-east-1
```

2. **Add to parser.go**: `debian-12`, `debian-11`

3. **Create templates**:
   - `templates/debian-12-server.yml`
   - `templates/debian-12-desktop.yml`
   - `templates/debian-11-server.yml`

4. **Package Manager**: APT (same as Ubuntu)

5. **Default User**: `admin` (Debian convention)

---

## Priority 4: RHEL (Optional)

**Target Release**: v0.6.1
**Estimated Effort**: 8-10 hours (licensing complexity)
**Business Value**: LOW-MEDIUM (licensing barrier)

### Target Matrix

| Template | Status | Priority |
|----------|--------|----------|
| RHEL 9.x Server | 🎯 Planned | LOW |
| RHEL 8.x Server | 🎯 Planned | LOW |

### Benefits

- **Official Red Hat support**
- **Enterprise compliance** requirements
- **Certification programs**

### Challenges

- **Requires Red Hat subscription** or developer account
- **Licensing complexity** for end users
- **Rocky Linux provides free alternative** (same binary compatibility)

### Considerations

- Could use **RHEL Universal Base Images (UBI)** for free tier
- Need to document **subscription requirements** clearly
- **Lower priority** due to free Rocky Linux alternative

---

## AMI Discovery Reference

### Official AMI Owner IDs

| Distribution | Owner ID | Notes |
|--------------|----------|-------|
| Ubuntu | 099720109477 | Canonical official |
| Rocky Linux | 679593333241 | Rocky Enterprise Software Foundation |
| Amazon Linux | amazon | AWS-owned |
| Alpine Linux | 538276064493 | Alpine Linux official |
| Debian | 136693071363 | Debian official |
| RHEL | 309956199498 | Red Hat official |

### AMI Search Template

```bash
# Generic search pattern
aws ec2 describe-images \
  --owners {OWNER_ID} \
  --filters "Name=name,Values={PATTERN}" \
  --query 'Images[*].[ImageId,Name,Architecture,CreationDate]' \
  --region {REGION} \
  --output table | sort -k4 -r
```

### Required Regions

Minimum coverage for initial launch:
- ✅ `us-east-1` (N. Virginia) - Most common
- ✅ `us-east-2` (Ohio)
- ✅ `us-west-1` (N. California)
- ✅ `us-west-2` (Oregon)

Additional regions for global support (future):
- `eu-west-1` (Ireland)
- `eu-central-1` (Frankfurt)
- `ap-southeast-1` (Singapore)
- `ap-northeast-1` (Tokyo)

---

## Validation Checklist

For each new OS version, verify:

- [ ] AMIs found for all 4 primary regions (us-east-1/2, us-west-1/2)
- [ ] Both x86_64 and arm64 architectures available
- [ ] AMI IDs added to `parser.go` `getDefaultBaseAMIs()` map
- [ ] Base OS added to `tester.go` `supportedOS` map (if not present)
- [ ] Server template created with appropriate packages
- [ ] Desktop template created (if GUI available)
- [ ] Slugs follow naming convention: `{os}-{major}-{minor}-{purpose}`
- [ ] Connection types set correctly (ssh/dcv)
- [ ] Package manager specified correctly (apt/dnf/apk/yum)
- [ ] Default user configured per distro convention
- [ ] Templates validate successfully: `cws templates validate`
- [ ] Templates appear in listing: `cws templates | grep {os}`
- [ ] Binaries rebuilt and tested

---

## Documentation Updates

When implementing new OS versions, update:

1. **README.md**: OS support matrix table
2. **templates/README.md**: OS selection guidance
3. **docs/TEMPLATES.md**: Template creation guide
4. **CHANGELOG.md**: New OS version additions
5. **This roadmap**: Update status from 🎯 Planned to ✅ Complete

---

## Success Metrics

**Goal**: Comprehensive OS coverage by v0.6.1

| Metric | Target | Current |
|--------|--------|---------|
| Distributions Supported | 6+ | 3 |
| OS Versions | 15+ | 5 |
| Desktop Environments | 8+ | 2 |
| Architecture Coverage (x86_64 + arm64) | 100% | 100% |
| Regional Coverage (4 primary regions) | 100% | 100% |

---

## Future Considerations

### Phase 2 Expansion (v0.7.0+)

- **Fedora**: Latest stable + previous
- **openSUSE Leap**: Enterprise-focused
- **Arch Linux**: Rolling release for bleeding-edge research
- **CentOS Stream**: RHEL upstream

### Phase 3 Expansion (v0.8.0+)

- **Oracle Linux**: RHEL-compatible, free
- **AlmaLinux**: Another RHEL fork
- **Linux Mint**: Popular Ubuntu-based desktop
- **Pop!_OS**: System76's research-optimized distro

---

## Related Documents

- [Template System Implementation](../TEMPLATE_SYSTEM_IMPLEMENTATION.md)
- [Template Inheritance Guide](../TEMPLATE_INHERITANCE.md)
- [Ubuntu 24.04 Implementation](git commit c5f84ed5)
- [Package Manager Roadmap](./PACKAGE_MANAGER_ROADMAP.md)

---

**Last Updated**: 2025-10-18
**Status**: Planning Phase
**Next Milestone**: Rocky Linux 8.10 (v0.5.5)
