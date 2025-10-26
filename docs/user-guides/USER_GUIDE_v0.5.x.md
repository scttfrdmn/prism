# Prism User Guide - v0.5.x Series

**Version**: 0.5.x Series (Universal AMI System Era)
**Last Updated**: December 2025
**Target Audience**: Researchers, Students, Data Scientists

## Overview

Prism v0.5.x introduces the **Universal AMI System**, revolutionizing how researchers launch cloud environments. Instead of waiting 5-8 minutes for software installation, you can now launch pre-built environments in **30 seconds** while maintaining full flexibility.

## 🚀 What's New in v0.5.x

### **⚡ Instant Environment Launches**
- **30-second launches** for optimized environments
- **4.2x faster** than script-based provisioning
- **Universal AMI support** for any research template
- **Intelligent fallbacks** when AMIs unavailable

### **🌐 Global Availability**
- **Cross-region intelligence** finds AMIs anywhere
- **Automatic AMI copying** between regions
- **Cost-aware deployment** with transparent pricing
- **Regional optimization** for best performance

### **🤝 Community AMI Sharing**
- **Create AMIs** from your optimized instances
- **Share environments** with research community
- **Discover optimized** research environments
- **Rate and review** community contributions

### **🎉 Interactive Onboarding** (New in v0.5.6)
- **2-minute setup wizard** for first-time users
- **Personalized recommendations** based on research area
- **Automated AWS validation** with helpful guidance
- **Optional budget and hibernation** configuration

## First-Time Setup

If you're new to Prism, start with the interactive onboarding wizard:

```bash
prism init
```

The wizard will guide you through:

### 1. AWS Configuration
- Automatic credential detection from `~/.aws/credentials`
- Profile and region discovery
- Credential validation with helpful error messages

### 2. Research Area Selection
Choose your primary research domain for personalized recommendations:
- **Machine Learning / Data Science** - GPU-optimized policies, Python/R templates
- **Bioinformatics / Genomics** - Batch processing policies, computational biology tools
- **Social Science / Statistics** - Balanced policies, statistical analysis environments
- **Other** - General research computing with flexible configurations

### 3. Budget Configuration (Optional)
Set a monthly budget to receive cost alerts:
- Alert thresholds at 50%, 80%, and 100% of budget
- Non-blocking alerts (workspaces continue running)
- Adjust later with `prism budget` commands

### 4. Hibernation Policy
Automatically pause idle workspaces to save costs:
- **GPU Policy** (15 min idle → stop) - For expensive GPU instances
- **Batch Policy** (60 min idle → hibernate) - For long-running jobs
- **Balanced Policy** (30 min idle → hibernate) - General research workflows

### 5. Template Recommendations
Get personalized template suggestions based on your research area:
- Python Machine Learning (Simplified)
- R Research Environment (Simplified)
- Bioinformatics toolkits
- And more...

### Completion

Once setup is complete, you'll see next steps for launching your first workspace:

```
✅ Setup Complete!

You're all set! Here's how to launch your first workspace:

  prism launch "Python Machine Learning (Simplified)" my-first-project

Other helpful commands:
  • prism templates list - List all available templates
  • prism list - List your workspaces
  • prism connect <name> - Connect to a workspace
  • prism tui - Launch interactive interface
```

## Quick Start Guide

> **Note**: If you've already run `prism init`, jump directly to launching workspaces below.

### 1. Launch with AMI Optimization
```bash
# Automatic AMI resolution (fastest path)
prism launch python-ml my-research
🔍 Resolving AMI for template: python-ml
✅ Found optimized AMI: ami-0123456789abcdef0
📈 Performance: 4.2x faster launch (30s vs 6min)
🚀 Launching with pre-built environment...

# Preview AMI resolution before launch
prism launch python-ml my-research --dry-run --show-ami-resolution
```

### 2. Explore AMI Options
```bash
# List available AMIs for templates
prism ami list --template python-ml
📋 Available AMIs for template: python-ml

Region: us-east-1
  ami-0123456789abcdef0  Python ML v2.1.0   (community)  ⭐ 4.8/5
  ami-0fedcba9876543210  Python ML v2.0.5   (official)   ⭐ 4.6/5

# Test AMI availability across regions
prism ami test python-ml --all-regions
```

### 3. Create and Share AMIs
```bash
# Create AMI from your optimized instance
prism ami create python-ml my-instance --name "My Python ML Setup"
🔧 Creating AMI from instance: my-instance
✅ AMI created: ami-0123456789abcdef0

# Share with community
prism ami share ami-0123456789abcdef0 --community prism
```

## AMI System Deep Dive

### AMI Resolution Strategy

Prism uses intelligent **multi-tier resolution** to find the best deployment method:

1. **Direct Mapping**: Region-specific AMI references (fastest - 30 seconds)
2. **Dynamic Search**: Pattern-based AMI discovery (45 seconds)
3. **Marketplace Integration**: AWS Marketplace AMI lookup (60 seconds)
4. **Cross-Region Intelligence**: Copy AMI from other regions (2 minutes)
5. **Script Fallback**: Traditional installation (5-8 minutes)

### Template AMI Configuration

Templates can now include AMI optimization:

```yaml
# Template with AMI optimization
name: "Python ML (Optimized)"
ami_config:
  strategy: "ami_preferred"  # Try AMI first, fallback to script
  ami_mappings:
    us-east-1: "ami-0123456789abcdef0"
    us-west-2: "ami-0fedcba9876543210"
  fallback_strategy: "script_provisioning"
  preferred_architecture: "arm64"  # Cost optimization
```

### Understanding AMI Strategies

| Strategy | Behavior | Use Case |
|----------|----------|----------|
| `ami_preferred` | Try AMI first, fallback to script | **Recommended**: Balance speed and reliability |
| `ami_required` | AMI only, fail if unavailable | Critical applications requiring exact environments |
| `ami_fallback` | Script first, AMI if script fails | Legacy templates transitioning to AMI |

## Advanced Features

### Cross-Region Deployment

When AMIs aren't available in your region:

```bash
# Automatic cross-region resolution
prism launch python-ml my-research --region ap-south-1
🔍 Resolving AMI in ap-south-1...
❌ No AMI in ap-south-1
🔄 Searching fallback regions...
✅ Found AMI in ap-southeast-1: ami-0xyz123456789def0
📋 Cross-region copy required (2 minutes + $0.03)
Continue? [y/N]: y
```

### Performance Optimization

Prism automatically optimizes for:
- **Architecture**: ARM64 preferred for cost savings
- **Instance Types**: Match AMI optimizations to instance families
- **Regional Costs**: Consider data transfer for cross-region copies
- **Launch Speed**: Prioritize faster deployment for interactive work

### Cost Management

Understanding AMI costs:

```bash
# Compare deployment costs
prism launch python-ml my-research --dry-run --show-costs
💰 Cost Analysis:

AMI Launch:
  Instance: $0.45/hour (immediate availability)
  AMI Storage: $0.01/month (shared across launches)

Script Launch:
  Instance: $0.45/hour + 6min setup cost ($0.045)
  No storage costs

Recommendation: AMI launch saves time and reduces setup costs
```

## Community AMI System

### Discovering Community AMIs

```bash
# Browse community AMIs
prism ami browse --category machine-learning
📂 Community AMIs: Machine Learning

Python ML Environments:
  ⭐ 4.8/5  Python ML v2.1.0    (1,247 downloads)
  ⭐ 4.6/5  PyTorch Research     (892 downloads)
  ⭐ 4.5/5  TensorFlow Optimized (654 downloads)

# Show detailed AMI information
prism ami info ami-0123456789abcdef0
📋 AMI: Python ML v2.1.0
Creator: ml-research-group@university.edu
Description: Optimized Python ML with CUDA 12.0, PyTorch 2.1
Rating: ⭐⭐⭐⭐⭐ (4.8/5, 23 reviews)
Performance: 4.2x faster than script installation
```

### Contributing AMIs

```bash
# Create optimized AMI from your work
prism ami create python-ml my-instance \
  --name "Python ML with Custom Libraries" \
  --description "Includes bioinformatics and visualization tools" \
  --public

# Multi-region deployment
prism ami create-multi python-ml my-instance \
  --regions us-east-1,us-west-2,eu-west-1 \
  --name "Global Python ML Environment"
```

### AMI Best Practices

**Creating High-Quality AMIs**:
1. **Test Thoroughly**: Launch from your AMI multiple times
2. **Document Changes**: Clear description of customizations
3. **Security Review**: Remove sensitive data and credentials
4. **Performance Optimize**: Include only necessary software
5. **Multi-Region**: Deploy to popular research regions

**Using Community AMIs**:
1. **Check Ratings**: Prefer highly-rated, well-reviewed AMIs
2. **Verify Source**: Trust reputable creators and institutions
3. **Test First**: Try AMI in development before production use
4. **Stay Updated**: Monitor for updated versions
5. **Provide Feedback**: Rate and review AMIs you use

## Troubleshooting AMI Issues

### Common Issues and Solutions

**AMI Not Available in Region**:
```bash
# Check cross-region options
prism ami test python-ml --region eu-central-1
❌ No direct AMI in eu-central-1
✅ Available in eu-west-1 (copy cost: $0.02, time: 90s)
⚠️  Fallback to script provisioning available (6 minutes)
```

**Slow AMI Resolution**:
```bash
# Force specific resolution method
prism launch python-ml my-research --ami-strategy direct_mapping
prism launch python-ml my-research --ami-strategy marketplace
prism launch python-ml my-research --prefer-script  # Skip AMI entirely
```

**AMI Creation Failures**:
```bash
# Verify instance state before creating AMI
prism instance status my-instance
prism ami create python-ml my-instance --wait-for-running
```

### Getting Help

**AMI System Support**:
- Check AMI availability: `prism ami test <template>`
- View resolution logs: `prism launch <template> <name> --debug`
- Report AMI issues: Include AMI ID and region in support requests

**Community Support**:
- Rate problematic AMIs to help others
- Report security issues in community AMIs
- Contribute fixes and improvements back to community

## Template Marketplace Integration

### Repository-Based Templates

Coming in v0.5.3, templates can reference AMIs from different repositories:

```bash
# Launch from community repository with AMI
prism launch community/bioinformatics/genomics-pipeline my-project
🔍 Resolving from community repository...
✅ Found optimized AMI: ami-0bio123456789def0
🚀 Launching bioinformatics environment...

# Launch from institutional repository
prism launch university-edu/research-standard my-project
🔐 Authenticating with university-edu...
✅ Found institutional AMI: ami-0uni123456789def0
```

### Configuration Sync Integration

Coming in v0.5.4, AMI launches can include configuration sync:

```bash
# Launch with AMI + configuration sync
prism launch python-ml my-research --config my-rstudio-setup --sync ~/research/data
⚡ Using AMI: ami-0123456789abcdef0 (30s launch)
⚙️  Syncing RStudio configuration...
📁 Setting up directory sync...
✅ Environment ready with your personalized configuration
```

## Migration from Script-Based Templates

### Gradual Migration

Existing templates work unchanged in v0.5.x:

```bash
# Existing script-based template (still works)
prism launch python-research my-old-project
⚙️  Using script provisioning (no AMI configured)
⏳ Installing packages... (6 minutes)
✅ Environment ready

# Same template with AMI optimization
prism launch python-ml my-new-project  # AMI-optimized version
⚡ Using AMI (30 seconds)
✅ Environment ready
```

### Template Conversion

Converting your templates to use AMIs:

1. **Launch existing template**: `prism launch old-template optimization-instance`
2. **Customize environment**: Install additional packages, configure settings
3. **Create AMI**: `prism ami create old-template optimization-instance --name "Optimized Version"`
4. **Update template**: Add AMI config to template YAML
5. **Test new template**: Launch and verify functionality
6. **Share improvements**: Contribute AMI to community

## Performance Benchmarks

### Launch Time Comparisons

| Template Type | Script Time | AMI Time | Improvement |
|---------------|-------------|----------|-------------|
| Python ML     | 6m 30s     | 30s      | **13x faster** |
| R Research    | 8m 15s     | 35s      | **14x faster** |
| Bioinformatics| 12m 45s    | 45s      | **17x faster** |
| GIS Research  | 15m 30s    | 60s      | **15x faster** |

### Cost Impact

| Scenario | Script Cost | AMI Cost | Savings |
|----------|-------------|----------|---------|
| 1-hour session | $0.495 | $0.455 | **8%** |
| 8-hour session | $3.60 | $3.61 | **Break-even** |
| Multiple launches | High setup overhead | **Amortized storage cost** |

**Key Insight**: AMIs provide immediate time savings and cost benefits for short sessions and multiple launches.

## Security Considerations

### AMI Security

**Using Community AMIs**:
- Only use AMIs from trusted sources
- Review AMI creator credentials and ratings
- Monitor for security updates and patches
- Report suspicious or compromised AMIs

**Creating Secure AMIs**:
- Remove all sensitive data before creating AMI
- Use least-privilege access controls
- Include security updates and patches
- Document security configuration in AMI description

**Institutional Policies**:
- Follow institutional AMI usage policies
- Use approved AMI repositories where required
- Maintain audit trails of AMI usage
- Report policy violations promptly

### Access Controls

AMI access is controlled through AWS IAM:
- **Public AMIs**: Available to all Prism users
- **Community AMIs**: Shared within research community
- **Institutional AMIs**: Restricted to organization members
- **Private AMIs**: Only available to creator

## Best Practices Summary

### For Researchers
1. **Use AMI-optimized templates** for fastest launches
2. **Preview resolution** with `--dry-run` for complex deployments
3. **Create AMIs** from your optimized environments
4. **Share improvements** with the research community
5. **Monitor costs** for AMI storage vs. launch frequency

### For Institutions
1. **Standardize on validated AMIs** for consistent environments
2. **Create institutional AMI repositories** for approved software
3. **Train users** on AMI system benefits and usage
4. **Monitor AMI costs** and establish governance policies
5. **Contribute improvements** back to the community

### For Development Teams
1. **Include AMI configs** in new templates
2. **Test AMI availability** across target regions
3. **Maintain AMI updates** with security patches
4. **Document AMI customizations** clearly
5. **Version AMIs** consistently with semantic versioning

---

**Prism v0.5.x** transforms research computing by providing **instant access to optimized environments** while maintaining the flexibility and reliability researchers depend on. The Universal AMI System represents the future of research cloud deployment - **fast, reliable, and community-driven**.