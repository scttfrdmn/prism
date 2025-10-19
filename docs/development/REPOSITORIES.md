# CloudWorkstation Template Repositories

This document describes the multi-repository support introduced in CloudWorkstation 0.3.0, allowing users to use templates from multiple sources.

## Overview

CloudWorkstation 0.3.0 introduces a powerful multi-repository system that enables:

- Using templates from different sources
- Organizational customization of templates
- Override capabilities based on repository priority
- Template version management
- Dependency resolution across repositories

## Repository Structure

A CloudWorkstation template repository follows a standardized structure:

```
repository/
├── repository.yaml           # Repository metadata
├── domains/                  # Domain-specific templates
│   ├── life-sciences/
│   │   ├── genomics.yaml
│   │   ├── neuroscience.yaml
│   │   └── ...
│   ├── physical-sciences/
│   │   ├── climate.yaml
│   │   └── ...
│   └── ...
├── base/                     # Base templates
│   ├── ubuntu-desktop.yaml
│   └── ...
└── stacks/                   # Reusable template stacks
    ├── python-ml.yaml
    └── ...
```

### Repository Metadata

Each repository includes a `repository.yaml` file with metadata:

```yaml
name: "Default Repository"
description: "Official CloudWorkstation template repository"
maintainer: "CloudWorkstation Team"
website: "https://github.com/scttfrdmn/cloudworkstation"
contact_email: "contact@example.com"
version: "1.0.0"
last_updated: "2025-07-11"
compatibility:
  min_version: "0.3.0"
  max_version: "0.3.99"
templates:
  - name: "r-research"
    path: "domains/data-science/r-research.yaml"
    versions:
      - version: "1.0.0"
        date: "2025-07-01"
      - version: "1.1.0"
        date: "2025-08-15"
  - name: "python-ml"
    path: "domains/computer-science/python-ml.yaml"
    versions:
      - version: "1.0.0"
        date: "2025-07-01"
```

## Configuration

Multiple repositories are configured in `~/.cloudworkstation/config.json`:

```json
{
  "repositories": [
    {
      "name": "default",
      "url": "github.com/scttfrdmn/cloudworkstation-repository",
      "priority": 1
    },
    {
      "name": "organizational",
      "url": "github.com/myorg/templates",
      "priority": 2
    },
    {
      "name": "personal",
      "url": "github.com/username/my-templates",
      "priority": 3
    }
  ]
}
```

The `priority` field determines the override order, with higher numbers taking precedence over lower numbers.

## Repository Types

CloudWorkstation supports multiple repository types:

### GitHub Repository

```json
{
  "name": "default",
  "type": "github",
  "url": "github.com/scttfrdmn/cloudworkstation-repository",
  "branch": "main",
  "priority": 1
}
```

### Local Directory

```json
{
  "name": "local-dev",
  "type": "local",
  "path": "/path/to/local/templates",
  "priority": 3
}
```

### S3 Bucket

```json
{
  "name": "org-templates",
  "type": "s3",
  "bucket": "my-org-templates",
  "prefix": "cloudworkstation/",
  "region": "us-west-2",
  "priority": 2
}
```

## Default Repository

CloudWorkstation includes a default repository at `github.com/scttfrdmn/cloudworkstation-repository` that provides:

1. Base templates for common operating systems
2. Stack templates for popular research environments
3. Domain-specific templates for 24 research domains
4. Example templates for customization

## Command Line Interface

### Repository Management

```bash
# List configured repositories
cws repo list

# Add a repository
cws repo add myorg github.com/myorg/templates --priority 2

# Remove a repository
cws repo remove myorg

# Update repositories
cws repo update

# Get repository information
cws repo info myorg
```

### Template Management

```bash
# List templates from all repositories
cws repo templates

# List templates from a specific repository
cws repo templates --repo myorg

# Search for templates
cws repo search machine-learning

# View template details
cws repo template info python-ml
cws repo template info myorg:python-ml@1.2.0
```

### Template Transfer

```bash
# Pull template from repository
cws repo pull python-ml
cws repo pull myorg:custom-ml

# Push template to repository (with write access)
cws repo push my-template.yaml --repo myorg
```

## Template Resolution

When a template name is specified, CloudWorkstation resolves it using the following process:

1. Parse template specification: [repo:]template[@version]
2. If repo is specified, look only in that repository
3. If no repo is specified, search repositories in priority order (highest to lowest)
4. If version is specified, use that specific version
5. If no version is specified, use the latest version

### Examples

- `python-ml` - Latest version of python-ml from highest priority repository
- `myorg:python-ml` - Latest version of python-ml from myorg repository
- `python-ml@1.2.0` - Specific version 1.2.0 of python-ml
- `myorg:python-ml@1.2.0` - Specific version 1.2.0 of python-ml from myorg repository

## Template Dependencies

Templates can depend on other templates, potentially from different repositories:

```yaml
dependencies:
  - repository: "default"
    template: "base/ubuntu-desktop"
    version: "1.0.0"
  - repository: "myorg"
    template: "stacks/python-ml"
    version: "1.1.0"
```

The dependency resolution process:

1. Resolve each dependency using the template resolution process
2. Check for circular dependencies
3. Build dependency tree with correct order
4. Apply templates in dependency order

## Local Cache

CloudWorkstation maintains a local cache of repositories to improve performance:

```
~/.cloudworkstation/repositories/
├── default/
│   └── ... (repository contents)
├── myorg/
│   └── ... (repository contents)
└── cache.json
```

The cache is automatically updated:
- When explicitly requested with `cws repo update`
- When a template is not found in the cache
- When the cache is older than the configured TTL (default: 24 hours)

## Creating Your Own Repository

To create your own template repository:

1. Create a new GitHub repository
2. Add a `repository.yaml` file with metadata
3. Create the directory structure (domains, base, stacks)
4. Add your templates
5. Add the repository to CloudWorkstation with `cws repo add`

### Example repository.yaml

```yaml
name: "My Organization Templates"
description: "Custom CloudWorkstation templates for my organization"
maintainer: "Your Name"
website: "https://github.com/myorg/templates"
contact_email: "you@example.com"
version: "1.0.0"
last_updated: "2025-07-11"
compatibility:
  min_version: "0.3.0"
  max_version: "0.3.99"
templates:
  - name: "custom-ml"
    path: "domains/computer-science/custom-ml.yaml"
    versions:
      - version: "1.0.0"
        date: "2025-07-01"
```

## Best Practices

1. **Namespace Organization**: Use clear naming conventions to avoid conflicts
2. **Repository Specificity**: Create repositories for specific purposes (e.g., organization, research group, personal)
3. **Prioritization**: Assign priorities based on specificity (personal > organizational > default)
4. **Version Management**: Use semantic versioning for templates
5. **Documentation**: Include comprehensive documentation in repository.yaml and individual templates
6. **Dependencies**: Explicitly specify dependencies with version constraints
7. **Cache Management**: Update repositories regularly with `cws repo update`

## Security Considerations

1. Templates can execute arbitrary code during AMI building
2. Only add repositories from trusted sources
3. Review template code before building AMIs
4. Use template validation with `cws ami validate` before building
5. Consider using checksums for template verification