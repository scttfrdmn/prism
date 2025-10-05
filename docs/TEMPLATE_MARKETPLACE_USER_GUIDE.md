# Template Marketplace User Guide

**Version**: v0.5.2
**Last Updated**: October 4, 2025
**Status**: Production Ready

## Overview

The CloudWorkstation Template Marketplace is a comprehensive system for discovering, sharing, and installing research environment templates. It connects researchers with a vast ecosystem of pre-configured environments optimized for different research domains, from machine learning and data science to bioinformatics and high-performance computing.

## Key Features

### 🔍 **Template Discovery**
- **Advanced Search**: Find templates using text queries, categories, domains, and complexity levels
- **Quality Filtering**: Filter by ratings, verification status, and validation results
- **Feature Filtering**: Search by research user support, connection types, and package managers
- **Multi-Registry Support**: Access community, institutional, and private template registries

### 🔒 **Security & Validation**
- **Security Scanning**: Comprehensive vulnerability detection and policy enforcement
- **Quality Analysis**: Automated scoring for documentation, metadata, and complexity
- **Verification System**: Official verification badges for trusted templates
- **Dependency Tracking**: Complete dependency analysis with license compatibility

### 💻 **Easy Installation**
- **One-Command Install**: Simple template installation from marketplace
- **Local Integration**: Installed templates work like built-in templates
- **Version Management**: Support for specific template versions
- **Registry Management**: Configure and manage multiple template sources

## Quick Start

### 1. Search for Templates

Search for machine learning templates:
```bash
cws marketplace search "machine learning"
```

Browse templates by category:
```bash
cws marketplace browse --category "Data Science"
```

Filter by specific features:
```bash
cws marketplace search --research-user --verified --min-rating 4.0
```

### 2. Explore Template Details

Get comprehensive information about a template:
```bash
cws marketplace show pytorch-research
```

This shows:
- Detailed description and usage guidance
- Security scan results and validation status
- Community ratings and download statistics
- Dependencies and compatibility information
- Prerequisites and learning resources

### 3. Install Templates

Install a template from the marketplace:
```bash
cws marketplace install pytorch-research
```

Install a specific version:
```bash
cws marketplace install pytorch-research --version 2.1.0
```

Install from a specific registry:
```bash
cws marketplace install pytorch-research --registry community
```

### 4. Use Installed Templates

Once installed, use templates like built-in ones:
```bash
cws launch pytorch-research my-ml-project
```

## Command Reference

### `cws marketplace search [query]`

Search for templates across all configured registries.

**Usage:**
```bash
cws marketplace search [query] [flags]
```

**Examples:**
```bash
# Basic text search
cws marketplace search "deep learning"

# Category-specific search
cws marketplace search --category "Machine Learning" --category "Data Science"

# Quality filtering
cws marketplace search --verified --min-rating 4.0

# Feature filtering
cws marketplace search --research-user --connection ssh

# Advanced filtering with sorting
cws marketplace search "python" --complexity simple --sort rating --order desc
```

**Flags:**
- `--category`: Filter by categories
- `--domain`: Filter by domains (ml, datascience, bio, web, etc.)
- `--complexity`: Filter by complexity (simple, moderate, advanced, complex)
- `--keywords`: Search specific keywords
- `--min-rating`: Minimum rating filter (0-5)
- `--verified`: Show only verified templates
- `--validated`: Show only validated templates
- `--research-user`: Show only templates with research user support
- `--connection`: Filter by connection types (ssh, web, dcv)
- `--package-manager`: Filter by package managers (apt, conda, spack)
- `--registry`: Search specific registries
- `--registry-type`: Filter by registry types (community, institutional, private, official)
- `--sort`: Sort by (popularity, rating, updated, name)
- `--order`: Sort order (asc, desc)
- `--limit`: Results per page (default: 20)
- `--offset`: Results offset for pagination
- `--format`: Output format (table, json)

### `cws marketplace browse`

Browse templates by categories and discover popular templates.

**Usage:**
```bash
cws marketplace browse [flags]
```

**Examples:**
```bash
# Browse all categories overview
cws marketplace browse

# Browse specific category
cws marketplace browse --category "Bioinformatics"

# Browse with different format
cws marketplace browse --category "Web Development" --format json
```

**Flags:**
- `--category`: Browse specific category
- `--format`: Output format (table, json)

### `cws marketplace show <template-name>`

Display comprehensive information about a specific template.

**Usage:**
```bash
cws marketplace show <template-name> [flags]
```

**Examples:**
```bash
# Show template details
cws marketplace show pytorch-research

# Show specific version
cws marketplace show pytorch-research --version 2.1.0

# Show from specific registry
cws marketplace show pytorch-research --registry community
```

**Information Displayed:**
- **Basic Information**: Name, description, category, complexity
- **Marketplace Metadata**: Ratings, downloads, verification status, badges
- **Security Information**: Security scan results, vulnerability reports
- **Technical Specifications**: Base OS, package manager, connection type, launch time
- **Research User Support**: EFS integration, automatic provisioning
- **Dependencies**: Required templates and compatibility information
- **Prerequisites**: Required knowledge and skills
- **Learning Resources**: Documentation links and tutorials
- **Usage Instructions**: Installation and launch commands

**Flags:**
- `--registry`: Search specific registry
- `--version`: Show specific template version

### `cws marketplace install <template-name>`

Download and install a template from the marketplace for local use.

**Usage:**
```bash
cws marketplace install <template-name> [flags]
```

**Examples:**
```bash
# Install latest version
cws marketplace install pytorch-research

# Install specific version
cws marketplace install pytorch-research --version 2.1.0

# Install from specific registry
cws marketplace install pytorch-research --registry institutional

# Force overwrite existing template
cws marketplace install pytorch-research --force
```

**Installation Process:**
1. **Template Download**: Retrieves template YAML from registry
2. **Dependency Resolution**: Checks and resolves template dependencies
3. **Security Validation**: Performs security scan if enabled
4. **Local Installation**: Installs template to local templates directory
5. **Cache Update**: Updates local template cache for immediate availability

**Flags:**
- `--registry`: Install from specific registry
- `--version`: Install specific template version
- `--force`: Force overwrite existing template

### `cws marketplace registries`

Manage and view configured template registries.

**Usage:**
```bash
cws marketplace registries
```

**Example Output:**
```
Configured Registries
====================
NAME        TYPE            URL                                    STATUS
official    official        https://marketplace.cloudworkstation.dev    ✓ Available
community   community       https://community.cloudworkstation.dev      ✓ Available
university  institutional   https://templates.university.edu            ✓ Available
```

**Registry Types:**
- **Official**: CloudWorkstation official templates
- **Community**: Community-contributed templates
- **Institutional**: Organization-specific templates
- **Private**: Private registry requiring authentication

## Registry Configuration

### Default Registries

CloudWorkstation comes configured with two default registries:

1. **Official Registry** (`official`)
   - URL: `https://marketplace.cloudworkstation.dev`
   - Type: Official CloudWorkstation templates
   - Verification: All templates verified by CloudWorkstation team

2. **Community Registry** (`community`)
   - URL: `https://community.cloudworkstation.dev`
   - Type: Community-contributed templates
   - Verification: Community moderation and automated validation

### Adding Custom Registries

To add institutional or private registries, create a registry configuration file:

**~/.cloudworkstation/registries.yaml:**
```yaml
registries:
  - name: "university"
    type: "institutional"
    url: "https://templates.university.edu"
    credentials:
      type: "token"
      token: "your-access-token"

  - name: "research-group"
    type: "private"
    url: "https://templates.myresearchgroup.org"
    credentials:
      type: "basic"
      username: "researcher"
      password: "secure-password"
```

### Registry Authentication

Private and institutional registries support multiple authentication methods:

**Token Authentication:**
```yaml
credentials:
  type: "token"
  token: "your-bearer-token"
```

**Basic Authentication:**
```yaml
credentials:
  type: "basic"
  username: "your-username"
  password: "your-password"
```

**SSH Key Authentication:** (Future enhancement)
```yaml
credentials:
  type: "ssh_key"
  ssh_key: "/path/to/private/key"
```

## Template Quality Indicators

### Verification Badges

Templates in the marketplace display various quality indicators:

- **✓ Verified**: Officially verified by registry maintainers
- **⭐ Trending**: Popular templates with high recent usage
- **🏆 Editor's Choice**: Curated high-quality templates recommended by experts
- **❤️ Community Favorite**: High-rated templates loved by the community

### Security Indicators

Security scan results are displayed with clear indicators:

- **🟢 Passed**: No security issues found
- **🟡 Warning**: Minor security concerns, safe to use with caution
- **🔴 Failed**: Significant security issues, use not recommended

### Quality Scores

Templates receive quality scores (0-100) based on:

- **Security Score** (40%): Vulnerability scan results and policy compliance
- **Quality Score** (30%): Code quality, documentation, and metadata completeness
- **Documentation Score** (20%): Completeness of description, prerequisites, and learning resources
- **Metadata Score** (10%): Completeness of template metadata and categorization

## Template Categories

The marketplace organizes templates into research-focused categories:

### **Machine Learning & AI**
- Deep learning frameworks (PyTorch, TensorFlow, JAX)
- AutoML and model deployment environments
- GPU-optimized training environments
- MLOps and experiment tracking setups

### **Data Science & Analytics**
- Statistical analysis environments (R, Python, Julia)
- Big data processing (Spark, Dask, distributed computing)
- Visualization and dashboard tools
- Business intelligence and reporting platforms

### **Bioinformatics & Computational Biology**
- Genomics analysis pipelines
- Protein structure analysis tools
- Phylogenetic analysis environments
- Systems biology modeling platforms

### **High Performance Computing**
- Parallel computing environments
- Scientific computing libraries (BLAS, LAPACK, FFTW)
- Simulation and modeling frameworks
- Cluster computing configurations

### **Web Development & Applications**
- Full-stack development environments
- API development and microservices
- Database and backend services
- Frontend development tools

### **Domain-Specific Research**
- Physics simulation environments
- Chemistry and materials science tools
- Social science analysis platforms
- Digital humanities research tools

## Best Practices

### Template Selection

1. **Check Verification Status**: Prefer verified templates for production research
2. **Review Security Scores**: Ensure templates meet your security requirements
3. **Read User Ratings**: Community feedback provides valuable insights
4. **Verify Prerequisites**: Ensure you meet the knowledge requirements
5. **Check Compatibility**: Confirm CloudWorkstation version compatibility

### Template Installation

1. **Review Before Installing**: Always use `cws marketplace show` before installation
2. **Start with Verified Templates**: Begin with verified templates for reliability
3. **Test in Development**: Test templates with non-critical data first
4. **Keep Templates Updated**: Regularly check for template updates
5. **Document Dependencies**: Track template dependencies for reproducibility

### Security Considerations

1. **Enable Security Scanning**: Always review security scan results
2. **Validate Sources**: Only install from trusted registries
3. **Monitor for Updates**: Keep templates updated for security patches
4. **Review Permissions**: Understand what access templates require
5. **Report Issues**: Report security concerns to template maintainers

## Advanced Features

### Research User Integration

Many marketplace templates support automatic research user provisioning:

```bash
# Launch template with research user support
cws launch pytorch-research my-project --research-user johndoe
```

Templates with research user support provide:
- **Persistent Home Directories**: EFS-backed home directories
- **SSH Key Management**: Automatic key generation and distribution
- **Collaboration Support**: Shared workspaces and group permissions
- **Consistent Identity**: Same username/UID across all instances

### Template Customization

After installation, templates can be customized locally:

1. **Edit Template**: Modify the installed template YAML
2. **Add Packages**: Include additional packages or dependencies
3. **Configure Services**: Adjust service configurations
4. **Set Parameters**: Customize template parameters for your workflow

### Dependency Management

The marketplace tracks template dependencies:

- **Inheritance Dependencies**: Templates that inherit from others
- **Runtime Dependencies**: Required templates for operation
- **Build Dependencies**: Templates needed during build process

### Version Management

Templates support semantic versioning:

```bash
# Install latest version
cws marketplace install template-name

# Install specific version
cws marketplace install template-name --version 2.1.0

# Install version range (future enhancement)
cws marketplace install template-name --version ">=2.0.0,<3.0.0"
```

## Troubleshooting

### Common Issues

**Template Not Found:**
```
Error: template not found: template-name
```
**Solutions:**
- Check template name spelling
- Verify registry connectivity: `cws marketplace registries`
- Search for similar templates: `cws marketplace search template-name`

**Registry Authentication Failed:**
```
Error: registry returned error status: 401
```
**Solutions:**
- Verify credentials in registry configuration
- Check token expiration
- Contact registry administrator

**Installation Failed:**
```
Error: failed to install template: validation failed
```
**Solutions:**
- Check security scan results: `cws marketplace show template-name`
- Review template dependencies
- Use `--force` flag to override validation (not recommended)

**Template Launch Failed:**
```
Error: template validation failed during launch
```
**Solutions:**
- Verify template installation: `cws templates list`
- Check template syntax: `cws templates validate template-name`
- Reinstall template: `cws marketplace install template-name --force`

### Getting Help

1. **Check Documentation**: Review template-specific documentation
2. **Community Support**: Engage with template authors and community
3. **Registry Support**: Contact registry administrators for institutional registries
4. **CloudWorkstation Support**: Report issues at https://github.com/anthropics/claude-code/issues

## Examples and Use Cases

### Scenario 1: Machine Learning Researcher

A machine learning researcher needs a PyTorch environment with GPU support:

```bash
# Search for PyTorch templates
cws marketplace search "pytorch" --category "Machine Learning" --verified

# Review template details
cws marketplace show pytorch-gpu-research

# Install the template
cws marketplace install pytorch-gpu-research

# Launch with research user support
cws launch pytorch-gpu-research ml-project --research-user researcher
```

### Scenario 2: Bioinformatics Team

A bioinformatics team needs a collaborative environment for genomics analysis:

```bash
# Search for bioinformatics templates with collaboration support
cws marketplace search --category "Bioinformatics" --research-user

# Browse bioinformatics category
cws marketplace browse --category "Bioinformatics"

# Install collaborative genomics template
cws marketplace install collaborative-genomics

# Launch shared workspace
cws launch collaborative-genomics genomics-project --research-user team-lead
```

### Scenario 3: Institutional Deployment

An institution wants to deploy approved templates:

```bash
# List available registries
cws marketplace registries

# Search institutional registry only
cws marketplace search --registry institutional --verified

# Install institution-approved template
cws marketplace install institutional-python --registry institutional

# Launch with institutional policies
cws launch institutional-python student-project
```

## Future Enhancements

The Template Marketplace Foundation provides the groundwork for future enhancements:

### Planned Features (Phase 5C+)
- **GUI Marketplace Interface**: Professional Cloudscape-based browsing
- **Template Publishing**: Contribute templates to community registries
- **Advanced Analytics**: Usage patterns and performance metrics
- **Template Collections**: Curated template bundles for specific workflows
- **Automated Updates**: Background template updates with compatibility checking
- **Collaborative Features**: Template reviews, ratings, and discussions

### Integration Opportunities
- **CI/CD Integration**: Automated template validation in research workflows
- **Institutional SSO**: Single sign-on integration for institutional registries
- **Cloud Marketplace**: Integration with AWS, Azure, and GCP marketplaces
- **Research Platform Integration**: JupyterHub, RStudio Connect, and specialized platforms

## Conclusion

The CloudWorkstation Template Marketplace transforms research environment setup from hours of configuration to seconds of discovery and installation. By providing access to validated, secure, and well-documented templates, researchers can focus on their research rather than infrastructure management.

The marketplace's comprehensive validation system ensures security and quality, while the rich search and filtering capabilities help researchers find exactly the right environment for their work. With support for multiple registries, institutions can maintain their own curated collections while benefiting from the broader community ecosystem.

Whether you're a individual researcher looking for the perfect machine learning environment, or an institution deploying standardized research platforms, the Template Marketplace provides the tools and confidence needed for successful research computing.