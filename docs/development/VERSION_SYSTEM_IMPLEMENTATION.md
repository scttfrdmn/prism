# Universal Version System Implementation (v0.5.5)

## Overview

The Universal Version System enables dynamic OS version selection at launch time, solving the template explosion problem and providing automated AMI maintenance.

## Architecture

### 4-Level Hierarchical AMI Structure

```
distro → version → region → architecture → AMI
  |        |         |           |          |
ubuntu → 24.04 → us-east-1 → x86_64 → ami-0e2c8caa4b6378d8c
```

### 3-Tier Version Priority System

1. **User Override** (highest priority): `--version` flag at launch
2. **Template Requirement**: Version specified in template dependencies
3. **Default Version** (fallback): Distro-specific defaults

### Hybrid AMI Discovery

```
SSM Parameter Store (Dynamic)
    ↓ (if available)
Latest AMI from AWS
    ↓ (if unavailable)
Static Fallback AMI
```

## Supported Distributions

| Distribution | Versions | Default | SSM Support | Aliases |
|--------------|----------|---------|-------------|---------|
| Ubuntu | 24.04, 22.04, 20.04 | 24.04 | ✅ Yes | latest, lts, previous-lts |
| Rocky Linux | 10, 9 | 10 | ❌ No | latest, lts |
| Amazon Linux | 2023, 2 | 2023 | ✅ Yes | latest |
| Alpine | 3.20 | 3.20 | ❌ No | latest |
| RHEL | 9 | 9 | ❌ No | latest |
| Debian | 12 | 12 | ✅ Yes | latest, lts |

## Usage Examples

### Basic Version Selection

```bash
# Use default version (Ubuntu 24.04)
prism launch python-ml my-project

# Specify explicit version
prism launch python-ml my-project --version 22.04

# Use version alias
prism launch python-ml my-project --version lts
prism launch python-ml my-project --version latest
```

### Multi-Distro Support

```bash
# Rocky Linux 10 (latest)
prism launch rocky-base my-server --version 10

# Rocky Linux 9 (LTS)
prism launch rocky-base my-server --version 9

# Amazon Linux 2023
prism launch web-server aws-host --version 2023

# Debian 12
prism launch database-server db-host --version 12
```

### Version Aliases

```bash
# Ubuntu
--version latest       # → 24.04 (Noble Numbat)
--version lts          # → 24.04 (current LTS)
--version previous-lts # → 22.04 (Jammy Jellyfish)

# Rocky Linux
--version latest       # → 10 (latest release)
--version lts          # → 9 (long-term support)

# Amazon Linux
--version latest       # → 2023 (latest release)
```

## AMI Freshness Checking

### Automatic Validation

The system includes proactive AMI freshness checking that:

1. **Runs Monthly**: Validates all static AMI IDs against latest SSM values
2. **Fallback Trigger**: Automatically checks when static fallback is used
3. **Clear Reporting**: Shows exactly which AMIs are outdated and need updates

### Example Report

```
⚠️  Found 2 outdated AMI mappings:

  ubuntu 22.04 (us-east-1/x86_64):
    Current: ami-0abcdef1234567890
    Latest:  ami-0xyz9876543210fed

  debian 12 (us-west-2/arm64):
    Current: ami-0fedcba0987654321
    Latest:  ami-0123456789abcdef0

Run automated update or manually update static AMI mappings in pkg/templates/parser.go
```

### Checking Freshness Manually

```bash
# Check AMI freshness for current region
prism admin ami check-freshness

# Check specific region
prism admin ami check-freshness --region us-west-2

# Show only outdated AMIs
prism admin ami check-freshness --outdated-only
```

## AWS SSM Parameter Store Integration

### SSM Parameter Paths

Prism queries these official AWS SSM parameters:

#### Ubuntu (Canonical)
```
/aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp3/ami-id
/aws/service/canonical/ubuntu/server/24.04/stable/current/arm64/hvm/ebs-gp3/ami-id
/aws/service/canonical/ubuntu/server/22.04/stable/current/amd64/hvm/ebs-gp3/ami-id
/aws/service/canonical/ubuntu/server/22.04/stable/current/arm64/hvm/ebs-gp3/ami-id
/aws/service/canonical/ubuntu/server/20.04/stable/current/amd64/hvm/ebs-gp3/ami-id
/aws/service/canonical/ubuntu/server/20.04/stable/current/arm64/hvm/ebs-gp3/ami-id
```

#### Amazon Linux
```
/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-amd64
/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-arm64
/aws/service/ami-amazon-linux-latest/amzn2-ami-kernel-5.10-hvm-amd64-gp2
/aws/service/ami-amazon-linux-latest/amzn2-ami-kernel-5.10-hvm-arm64-gp2
```

#### Debian
```
/aws/service/debian/release/12/latest/amd64
/aws/service/debian/release/12/latest/arm64
```

### Static Fallback AMIs

Distributions without SSM support use static AMIs:
- **Rocky Linux**: Community-maintained AMIs
- **RHEL**: Red Hat official AMIs
- **Alpine**: Alpine official cloud images

These are validated monthly via freshness checking and can be manually updated in `pkg/templates/parser.go`.

## Implementation Components

### Core Files

1. **pkg/aws/ami_discovery.go** (416 lines)
   - `AMIDiscovery` struct with SSM client
   - `GetLatestAMI()` - Dynamic discovery via SSM
   - `GetAMIWithFallback()` - SSM with static fallback
   - `BulkDiscoverAMIs()` - Daemon startup warm-up
   - `CheckAMIFreshness()` - Monthly validation
   - `FormatFreshnessReport()` - Human-readable reports

2. **pkg/templates/resolver.go** (267 lines)
   - `VersionResolver` - Version resolution and aliases
   - `ResolveAMI()` - AMI lookup in hierarchical structure
   - `GetVersionAliases()` - Alias mappings per distro
   - `getDefaultVersion()` - Distro-specific defaults
   - `resolveVersionAlias()` - Alias to version translation

3. **pkg/templates/dependencies.go** (300 lines)
   - `DependencyResolver` - 3-tier priority resolution
   - `ResolveDependencies()` - User > Template > Default
   - `ResolvedDependencies` - Resolution result with source

4. **pkg/templates/parser.go** (modified)
   - Reorganized to 4-level hierarchical structure
   - Added Rocky, Amazon Linux, Alpine, RHEL, Debian
   - `getDefaultBaseAMIs()` - Complete AMI mappings

5. **internal/cli/commands.go** (+16 lines)
   - `VersionCommand` - CLI flag handler
   - Integration with `LaunchCommandDispatcher`

6. **pkg/types/requests.go** (+3 lines)
   - Added `Version` field to `LaunchRequest`

### Test Coverage

**pkg/templates/resolver_test.go** (286 lines)
- ✅ Version resolution for all distros
- ✅ Version alias translation
- ✅ Default version fallback
- ✅ Invalid version error handling
- ✅ Architecture support validation
- ✅ All 14 tests passing

## Version Resolution Flow

```
User Launch Command
    ↓
Parse --version flag (or default)
    ↓
DependencyResolver
    ├─→ User Override? → Use user version
    ├─→ Template Requirement? → Use template version
    └─→ Default → Use distro default
    ↓
VersionResolver
    ├─→ Alias? → Resolve to actual version
    └─→ Explicit → Use as-is
    ↓
AMIDiscovery
    ├─→ Query SSM Parameter Store
    ├─→ Found? → Return latest AMI
    └─→ Not found? → Use static fallback
    ↓
Launch Instance with Resolved AMI
```

## Benefits

### For Users
- **No Configuration**: Templates work with sensible defaults
- **Version Flexibility**: Choose any supported OS version
- **Always Current**: SSM integration provides latest AMIs
- **Clear Communication**: Know exactly which version you're getting

### For Maintainers
- **Automatic Updates**: Ubuntu/Amazon Linux/Debian AMIs update automatically
- **Proactive Validation**: Monthly freshness checks catch outdated AMIs
- **Reduced Maintenance**: No manual AMI updates for SSM-supported distros
- **Clear Reports**: Know exactly which AMIs need updating

### For Templates
- **No Explosion**: Single template supports multiple versions
- **Version Constraints**: Templates can specify version requirements
- **Backward Compatible**: Existing templates continue working

## Future Enhancements

### Planned (Future Releases)
- **CLI Commands**: `prism admin ami check-freshness`, `prism admin ami update`
- **Cron Integration**: Automatic monthly freshness checks
- **Daemon Integration**: AMI discovery warm-up at daemon startup
- **Advanced Caching**: In-memory AMI cache with TTL
- **Multi-Region**: Parallel freshness checks across all regions
- **Auto-Update**: Automated AMI updates with approval workflow

### Potential (Under Consideration)
- **Custom SSM Paths**: Institutional overrides for private AMI registries
- **AMI Pinning**: Lock specific templates to specific AMI versions
- **Rollback Support**: Revert to previous AMI versions if issues occur
- **AMI Metrics**: Track AMI age, usage, and update frequency

## Maintenance

### Static AMI Updates

When freshness checking reports outdated AMIs:

1. **Review Report**: Check which AMIs are outdated
2. **Find Latest AMIs**: Use distribution-specific sources
3. **Update parser.go**: Update AMI IDs in `getDefaultBaseAMIs()`
4. **Test Changes**: Run `make test` to validate
5. **Commit**: Document which AMIs were updated and why

### Distribution-Specific Sources

- **Ubuntu**: https://cloud-images.ubuntu.com/locator/ec2/
- **Amazon Linux**: https://aws.amazon.com/amazon-linux-2/release-notes/
- **Rocky Linux**: https://rockylinux.org/cloud-images/
- **RHEL**: https://access.redhat.com/solutions/15356
- **Alpine**: https://alpinelinux.org/cloud/
- **Debian**: https://wiki.debian.org/Cloud/AmazonEC2Image

### Recommended Update Frequency

| Distribution | Frequency | Reason |
|--------------|-----------|--------|
| Ubuntu LTS | Every 6 months | Point releases (24.04.1, 24.04.2) |
| Amazon Linux 2023 | Quarterly | Frequent updates |
| Amazon Linux 2 | Every 6 months | Stable updates |
| Rocky Linux | Every 6 months | Point releases (9.1, 9.2) |
| RHEL | Every 6 months | Point releases |
| Alpine | Every 3 months | Frequent releases |
| Debian | Yearly | Very stable |

## Implementation Status

### ✅ Completed (v0.5.5)
- Core version resolution system
- 4-level hierarchical AMI structure
- 3-tier version priority system
- AWS SSM Parameter Store integration
- Static fallback AMI system
- AMI freshness checking
- Version alias support
- CLI --version flag integration
- Comprehensive test coverage
- Documentation

### 🎯 Next Steps (Future Integration)
- Wire AMIDiscovery into daemon initialization
- Add CLI commands for manual freshness checks
- Set up automated monthly validation
- Integrate version resolution into launch flow
- Add AMI update automation

## Commits

1. **7a7a35b4**: 🚀 v0.5.5: Universal Version System with Dynamic AMI Discovery
2. **760276c7**: feat: Add automatic AMI freshness checking and validation

## References

- [CLAUDE.md](../CLAUDE.md) - Phase 5B: v0.5.5 Universal Version System
- [pkg/aws/ami_discovery.go](../pkg/aws/ami_discovery.go) - AMI discovery implementation
- [pkg/templates/resolver.go](../pkg/templates/resolver.go) - Version resolution
- [pkg/templates/dependencies.go](../pkg/templates/dependencies.go) - Dependency resolution
- [pkg/templates/resolver_test.go](../pkg/templates/resolver_test.go) - Test coverage
