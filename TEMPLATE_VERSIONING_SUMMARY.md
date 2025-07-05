# Template Versioning System

This document summarizes the implementation of the template versioning system for the CloudWorkstation AMI builder.

## Overview

The template versioning system provides the following features:
1. Semantic versioning for templates (major.minor.patch)
2. Version management (get, set, increment)
3. Template dependencies with version constraints
4. Dependency validation and graph generation
5. CLI commands for version management

## Components

### Core Versioning

- **VersionInfo**: Structure representing semantic versions (major.minor.patch)
- **Version Manipulation**: Functions for incrementing major, minor, and patch versions
- **Version Comparison**: Functions for comparing versions

### Template Manager Extensions

- **GetTemplateVersion**: Get the current version of a template
- **SetTemplateVersion**: Set the version of a template
- **IncrementTemplateVersion**: Increment a template's version (major, minor, patch)
- **VersionTemplate**: Create a new versioned copy of a template
- **CreateTemplateVersion**: Create a new version of a template with changes

### Dependency Management

- **TemplateDependency**: Structure representing a dependency on another template
- **ValidateTemplateDependencies**: Validate all dependencies for a template
- **AddDependency**: Add a dependency to a template
- **RemoveDependency**: Remove a dependency from a template
- **GetDependencyGraph**: Build a dependency graph for a template

### CLI Commands

- **version**: Manage template versions
  - **get**: Get the current version of a template
  - **set**: Set the version of a template
  - **increment**: Increment the version of a template
  - **create**: Create a new version of a template
  - **list**: List all versions of a template

- **dependency**: Manage template dependencies
  - **add**: Add a dependency to a template
  - **remove**: Remove a dependency from a template
  - **list**: List dependencies for a template
  - **check**: Validate dependencies for a template
  - **graph**: Show the dependency graph for a template

## Example Usage

### Version Management

```bash
# Get the current version of a template
cws ami template version get python-ml

# Set the version of a template
cws ami template version set python-ml 2.0.0

# Increment the version of a template
cws ami template version increment python-ml minor

# Create a new version of a template
cws ami template version create python-ml minor
```

### Dependency Management

```bash
# Add a dependency to a template
cws ami template dependency add python-ml base-ubuntu --version 1.0.0 --operator ">="

# Remove a dependency from a template
cws ami template dependency remove python-ml base-ubuntu

# List dependencies for a template
cws ami template dependency list python-ml

# Check dependencies for a template
cws ami template dependency check python-ml

# Show the dependency graph for a template
cws ami template dependency graph python-ml
```

## Next Steps

1. Complete integration testing of the versioning system
2. Add support for template versioning in the registry
3. Implement automatic dependency resolution
4. Add support for advanced version constraints (e.g., ^1.2.3, ~1.2.3)
5. Add support for template upgrade automation