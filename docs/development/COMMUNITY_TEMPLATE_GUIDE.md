# Community Template Contribution Guide

## Welcome Contributors! 🎉

Thank you for your interest in contributing templates to Prism! Community-contributed templates are essential to building a diverse ecosystem of research environments.

This guide will walk you through the entire process of creating, testing, and submitting a template.

---

## Quick Start (5 Minutes)

1. **Fork** the [prism repository](https://github.com/scttfrdmn/prism)
2. **Create** your template in `templates/community/your-template-name.yml`
3. **Test** locally with `prism templates validate`
4. **Submit** a pull request using our template submission form

---

## Template Requirements

### File Location

All community templates must be placed in:

```
templates/community/your-template-name.yml
```

**Naming convention**: Use lowercase with hyphens (e.g., `python-ml-tensorflow.yml`, `r-tidyverse-stack.yml`)

### Required Fields

Every template must include these fields:

```yaml
name: "Your Template Name"
description: "Clear description of what this environment provides"
base_os: ubuntu-24.04        # or ubuntu-22.04, amazonlinux-2023
architecture: x86_64          # or arm64, both
package_manager: apt          # apt, yum, conda, spack

# Packages organized by type
packages:
  system:
    - build-essential
    - git

# User configuration
users:
  - name: researcher
    groups: ["sudo"]

# Ports to open
ports:
  - 22    # SSH
  - 8787  # RStudio (if applicable)

# Version tracking
version: "1.0.0"

# Categorization
tags:
  category: "Machine Learning"  # See categories below
  type: "research"
  level: "beginner"  # beginner, intermediate, advanced
```

###Template Categories

Choose the most appropriate category for your template:

- **Machine Learning**: ML frameworks, GPU computing, neural networks
- **Data Science**: Statistical computing, data analysis, visualization
- **Bioinformatics**: Genomics, proteomics, sequence analysis
- **Geospatial**: GIS, remote sensing, spatial analysis
- **Web Development**: Web frameworks, databases, containers
- **Research Tools**: General-purpose research computing
- **Desktop Applications**: GUI applications (via Nice DCV)
- **System Administration**: Infrastructure, monitoring, DevOps

### Optional But Recommended Fields

```yaml
# Author information (for recognition)
author: "Your Name <email@example.com>"

# Template inheritance (if building on existing template)
inherits: ["Basic Ubuntu (APT)"]

# Service configuration
services:
  - name: jupyter
    port: 8888
    type: web
    description: "Jupyter Lab"
    start_command: "jupyter lab --ip=0.0.0.0"

# Post-install script
post_install: |
  # Your custom setup commands
  echo "Setup complete!"

# Idle detection (for cost optimization)
idle_detection:
  enabled: true
  idle_threshold_minutes: 15
```

---

## Creating Your Template

### Step 1: Fork and Clone

```bash
# Fork on GitHub, then:
git clone https://github.com/YOUR_USERNAME/prism.git
cd prism
git checkout -b add-template-my-awesome-template
```

### Step 2: Create Template File

Create `templates/community/my-awesome-template.yml`:

```yaml
name: "My Awesome Research Environment"
description: "Python 3.11 with NumPy, Pandas, and Matplotlib for data analysis"
base_os: ubuntu-24.04
architecture: x86_64
package_manager: apt

packages:
  system:
    - python3
    - python3-pip
    - python3-venv
  python:
    - numpy
    - pandas
    - matplotlib
    - jupyter

users:
  - name: researcher
    groups: ["sudo"]

services:
  - name: jupyter
    port: 8888
    type: web
    description: "Jupyter Lab"

ports:
  - 22
  - 8888

version: "1.0.0"
tags:
  category: "Data Science"
  type: "research"
  level: "beginner"
```

### Step 3: Add Documentation

Create `docs/user-guides/MY_AWESOME_TEMPLATE.md`:

```markdown
# My Awesome Research Environment

## Overview
Brief description of what this template provides.

## Quick Start
\`\`\`bash
prism workspace launch my-awesome-template my-analysis
prism workspace connect my-analysis
\`\`\`

## What's Included
- Python 3.11
- NumPy, Pandas, Matplotlib
- Jupyter Lab on port 8888

## Usage
How to use key features...

## Troubleshooting
Common issues and solutions...
```

---

## Testing Your Template

### Validation

First, ensure your template syntax is valid:

```bash
# Validate template file
prism templates validate templates/community/my-awesome-template.yml

# Should output:
# ✅ Template 'My Awesome Research Environment' is valid
```

### Local Test Launch

Test the template by launching an instance:

```bash
# Dry-run to see what would be created
prism workspace launch --dry-run my-awesome-template test-instance

# Full test launch (costs apply!)
prism workspace launch my-awesome-template test-instance

# Connect and verify
prism workspace connect test-instance

# Test all features:
# - Check packages are installed
# - Test services are running
# - Verify ports are accessible

# Clean up when done
prism workspace delete test-instance
```

### Multi-Region Testing (Recommended)

Test in at least 2 regions to ensure compatibility:

```bash
# Test in us-east-1
AWS_REGION=us-east-1 prism workspace launch my-awesome-template test-us-east

# Test in us-west-2
AWS_REGION=us-west-2 prism workspace launch my-awesome-template test-us-west

# Clean up both
prism workspace delete test-us-east test-us-west
```

### Architecture Testing

If your template supports both x86_64 and ARM:

```bash
# Test x86_64 (default)
prism workspace launch my-awesome-template test-x86

# Test ARM64
prism workspace launch --architecture arm64 my-awesome-template test-arm

# Clean up
prism workspace delete test-x86 test-arm
```

---

## Submission Checklist

Before submitting your pull request, ensure:

- [ ] **Template validates** without errors
- [ ] **Tested on x86_64** instance (minimum requirement)
- [ ] **Tested on arm64** instance (if claiming support)
- [ ] **Documentation included** in `docs/user-guides/`
- [ ] **Author information** complete
- [ ] **Category assigned** (choose from list above)
- [ ] **Version number** set (use semantic versioning: 1.0.0)
- [ ] **All required fields** present
- [ ] **No hardcoded secrets** or credentials
- [ ] **Security best practices** followed
- [ ] **Package versions** documented (if pinning specific versions)
- [ ] **Multi-region tested** (at least 2 regions)

---

## Submitting Your Pull Request

### Step 1: Commit Your Changes

```bash
git add templates/community/my-awesome-template.yml
git add docs/user-guides/MY_AWESOME_TEMPLATE.md
git commit -m "feat(templates): Add My Awesome Research Environment template

- Python 3.11 with NumPy, Pandas, Matplotlib
- Jupyter Lab on port 8888
- Tested on x86_64 in us-east-1 and us-west-2
- Beginner-friendly data science environment"
```

### Step 2: Push to Your Fork

```bash
git push origin add-template-my-awesome-template
```

### Step 3: Create Pull Request

1. Go to your fork on GitHub
2. Click "Pull Request"
3. Choose "Template Submission" template
4. Fill in all sections
5. Submit!

**Use our template submission PR template** (creates automatically when you select it)

---

## Review Process

Once you submit your PR, here's what happens:

1. **Automated Validation** (< 5 minutes)
   - CI runs `prism templates validate`
   - Checks for required fields
   - Verifies documentation exists

2. **Maintainer Review** (1-3 days)
   - Code review for security and best practices
   - Documentation review for clarity
   - Feedback or approval

3. **Test Launch** (during review)
   - Maintainer launches in staging environment
   - Verifies all features work as described
   - Tests on multiple architectures if applicable

4. **Merge** (after approval)
   - Template merged to main branch
   - Published to community template registry

5. **Recognition** 🎉
   - Author credited in template metadata
   - Listed in contributors documentation
   - Template available to all Prism users!

---

## Best Practices

### Versioning

Use [Semantic Versioning](https://semver.org/):
- `1.0.0` - Initial release
- `1.1.0` - Add new packages or features
- `1.0.1` - Bug fixes, dependency updates
- `2.0.0` - Breaking changes

Update version number for ANY changes.

### Naming

**Good names**:
- `python-ml-tensorflow` (specific, searchable)
- `r-tidyverse-bioconductor` (technologies clearly listed)
- `julia-scientific-computing` (purpose clear)

**Avoid**:
- `my-template` (not descriptive)
- `best-ml-ever` (subjective)
- `research` (too generic)

### Package Versions

**Option 1**: Latest stable (recommended for most)
```yaml
packages:
  python:
    - numpy
    - pandas
```

**Option 2**: Pinned versions (for reproducibility)
```yaml
packages:
  python:
    - "numpy==1.24.3"
    - "pandas==2.0.2"
```

Document your choice in the description!

### Security

**❌ Never include**:
- API keys or secrets
- Passwords or tokens
- Personal information
- Proprietary software licenses

**✅ Always**:
- Use principle of least privilege
- Document any security considerations
- Follow OS security best practices
- Keep software up to date

### Documentation

**Keep it simple**:
- 5-minute quick start
- What's included (list)
- How to use key features
- Common troubleshooting

**Don't**:
- Write a book
- Assume expert knowledge
- Skip troubleshooting section
- Forget code examples

### Scope

**Do**: Keep templates focused
- "Python ML with TensorFlow" ✅
- "R for genomics" ✅

**Don't**: Mix unrelated tools
- "Python ML + Ruby + Java + databases" ❌

**Consider**: Using template inheritance for complex stacks

---

## Template Inheritance

Prism supports building on existing templates:

```yaml
name: "My Advanced Template"
inherits: ["Basic Ubuntu (APT)"]

# Your additions build on top of the base
packages:
  python:
    - tensorflow
    - pytorch
```

**Benefits**:
- Less duplication
- Inherit security updates
- Clearer relationships

**When to use**:
- Building specialized version of existing template
- Adding specific packages to general-purpose base
- Creating template family (basic → intermediate → advanced)

See Template Inheritance Documentation for details.

---

## Advanced Topics

### Post-Install Scripts

Use `post_install` for custom setup:

```yaml
post_install: |
  #!/bin/bash
  set -e  # Exit on error

  # Download datasets
  wget https://example.com/dataset.tar.gz -O /tmp/data.tar.gz
  tar -xzf /tmp/data.tar.gz -C /home/researcher/

  # Configure services
  systemctl enable myservice
  systemctl start myservice

  echo "✅ Custom setup complete"
```

**Best practices**:
- Use `set -e` for error handling
- Make scripts idempotent (can run multiple times)
- Add progress messages
- Clean up temporary files

### Service Configuration

For web services (Jupyter, RStudio, etc.):

```yaml
services:
  - name: jupyter
    port: 8888
    type: web
    description: "Jupyter Lab"
    start_command: "jupyter lab --ip=0.0.0.0 --no-browser"
    check_command: "curl -f http://localhost:8888 || exit 1"
```

### Custom Components

For tools not in base templates:

```yaml
packages:
  system:
    - custom-package-name

post_install: |
  # Install from source
  wget https://github.com/project/releases/download/v1.0.0/binary.tar.gz
  tar -xzf binary.tar.gz
  cp binary /usr/local/bin/
  chmod +x /usr/local/bin/binary
```

---

## Getting Help

### Community

- **GitHub Discussions**: Ask questions, share tips
- **GitHub Issues**: Report bugs, request features
- **Documentation**: Browse existing guides

### Examples

Check existing community templates for inspiration:
```bash
ls templates/community/
```

Good starting points:
- `base-ubuntu-apt.yml` - Minimal Ubuntu template
- `python-ml-configurable.yml` - Parameterized template
- `ultimate-research-workstation.yml` - Complex multi-tool template

---

## Recognition & Rewards

### Contributor Recognition

When your template is merged:
- ✅ **Author field** in template metadata (your name!)
- ✅ **Contributors page** in documentation
- ✅ **GitHub contributors** list
- ✅ **Community showcase** (featured templates)

### Building Your Reputation

Contributing templates helps:
- Build your open-source portfolio
- Share your research tooling expertise
- Help the research community
- Learn infrastructure-as-code best practices

---

## FAQ

**Q: Can I submit multiple templates?**
A: Yes! Submit each as a separate PR.

**Q: Can I update my template after it's merged?**
A: Yes! Submit a new PR with version bump.

**Q: What if my template needs proprietary software?**
A: Document installation steps in the guide. Users must provide their own licenses.

**Q: How long does review take?**
A: Usually 1-3 days. Complex templates may take longer.

**Q: Can I test templates without launching instances?**
A: Validation (`prism templates validate`) checks syntax. Full testing requires instance launch.

**Q: What if my template fails CI?**
A: CI will show the error. Fix and push again. We're here to help!

**Q: Can I contribute templates for other clouds (Azure, GCP)?**
A: Currently AWS only. Other clouds are planned for future phases.

---

## Next Steps

Ready to contribute? Great!

1. **Browse existing templates** for inspiration
2. **Plan your template** (what tools, what purpose?)
3. **Create and test** following this guide
4. **Submit your PR** using our template
5. **Engage with reviewers** for feedback

**Thank you for contributing to Prism!** 🚀

---

## Related Documentation

- Template README - Template structure and inheritance
- Template Review Checklist - Maintainer review guide
- [AWS Setup Guide](../user-guides/AWS_SETUP_GUIDE.md) - Setting up AWS access

---

**Last Updated**: January 2026 | **Version**: 1.0
