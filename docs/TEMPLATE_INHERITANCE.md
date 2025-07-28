# Template Inheritance System

## Overview

CloudWorkstation now supports template inheritance and stacking, allowing templates to build upon each other for powerful composition. This enables users to create specialized environments by combining base templates with additional functionality.

## How It Works

### Inheritance Declaration

Templates can inherit from parent templates using the `inherits` field:

```yaml
name: "Rocky Linux 9 + Conda Stack"
description: "Rocky Linux 9 base with Conda data science stack"
base: "ubuntu-22.04"

# Inherit from parent template(s)
inherits:
  - "Rocky Linux 9 Base"

# Override or extend parent configuration
package_manager: "conda"  # Override parent's DNF manager

packages:
  conda:  # Add conda packages on top of parent's system packages
    - "python=3.11"
    - "jupyter"
    - "numpy"
    - "pandas"

users:  # Add additional users alongside parent's users
  - name: "datascientist"
    password: "auto-generated"
    groups: ["sudo"]
```

### Merging Rules

When templates inherit from parents, configurations are merged using these intelligent rules:

| Field | Merge Behavior | Example |
|-------|---------------|---------|
| **Packages** | **Append** | Parent: `[git, vim]` + Child: `[python, jupyter]` = `[git, vim, python, jupyter]` |
| **Users** | **Append** | Parent: `[rocky]` + Child: `[datascientist]` = `[rocky, datascientist]` |
| **Services** | **Append** | Parent: `[ssh]` + Child: `[jupyter]` = `[ssh, jupyter]` |
| **Package Manager** | **Override** | Parent: `dnf` + Child: `conda` = `conda` |
| **Ports** | **Deduplicate** | Parent: `[22]` + Child: `[22, 8888]` = `[22, 8888]` |
| **Tags** | **Override** | Child tags override parent tags with same key |
| **Post-install** | **Append** | Parent script + Child script (with separator) |

## Working Example

### Base Template: `Rocky Linux 9 Base`

```yaml
name: "Rocky Linux 9 Base"
description: "Base Rocky Linux 9 environment with essential system tools"
base: "ubuntu-22.04"

package_manager: "dnf"

packages:
  system:
    - "build-essential"
    - "curl"
    - "wget" 
    - "git"
    - "vim"
    - "htop"
    - "tree"
    - "unzip"

users:
  - name: "rocky"
    password: "auto-generated"
    groups: ["wheel", "sudo"]

instance_defaults:
  ports: [22]
```

### Stacked Template: `Rocky Linux 9 + Conda Stack`

```yaml
name: "Rocky Linux 9 + Conda Stack"
description: "Rocky Linux 9 base with Conda data science stack"
base: "ubuntu-22.04"

inherits:
  - "Rocky Linux 9 Base"

# Override package manager to conda
package_manager: "conda"

# Add conda packages on top of base system packages
packages:
  conda:
    - "python=3.11"
    - "jupyter"
    - "numpy"
    - "pandas"
    - "matplotlib"
    - "scikit-learn"
  pip:
    - "plotly"

# Add additional user (combines with rocky user from base)
users:
  - name: "datascientist"
    password: "auto-generated"
    groups: ["sudo"]

# Add services on top of base
services:
  - name: "jupyter"
    port: 8888
    enable: true

# Extend ports from base template
instance_defaults:
  ports: [8888]  # Will be merged with base ports [22] = [22, 8888]
```

### Resolved Result

After inheritance resolution, the stacked template contains:

- **Package Manager**: `conda` (overridden from parent's `dnf`)
- **System Packages**: 8 packages from base template
- **Conda Packages**: 6 packages from child template
- **Pip Packages**: 1 package from child template
- **Users**: 2 users - `rocky` (base) + `datascientist` (child)
- **Services**: 1 service - `jupyter` from child
- **Ports**: 2 ports - `[22, 8888]` (merged and deduplicated)

## Multiple Inheritance

Templates can inherit from multiple parents:

```yaml
inherits:
  - "Base System Template"
  - "GPU Drivers Template"
  - "Python Environment Template"
```

Parents are processed in order, with later parents overriding earlier ones for conflicting fields.

## Recursive Inheritance

Templates can inherit from templates that also inherit from others. The system automatically resolves the full inheritance chain:

```
Specialized Template
    ↓ inherits from
Application Stack Template  
    ↓ inherits from
Base OS Template
```

## Error Handling

The inheritance system provides clear error messages for:

- **Missing Parent Templates**: `parent template not found: Template Name`
- **Circular Dependencies**: Detected and prevented with clear error messages
- **Invalid Inheritance**: Validation ensures parent templates exist before resolution

## Benefits

### 1. **Composition Over Duplication**
Instead of duplicating base configuration across templates, inherit and extend.

### 2. **Maintainable Template Library** 
Updates to base templates automatically propagate to child templates.

### 3. **Flexible Customization**
Override any aspect of parent templates while preserving the rest.

### 4. **Clear Relationships**
Template inheritance makes dependencies and relationships explicit.

## Usage Examples

### Data Science Stack
```bash
# Launch with inherited configuration
cws launch "Rocky Linux 9 + Conda Stack" my-analysis

# Override package manager at launch time
cws launch "Rocky Linux 9 + Conda Stack" my-analysis --with spack
```

### Building Complex Environments
```yaml
# GPU ML Template inheriting from multiple sources
name: "GPU Machine Learning Stack"
inherits:
  - "Rocky Linux 9 Base"      # Base OS and system tools
  - "NVIDIA GPU Drivers"      # GPU drivers and CUDA
  - "Conda ML Stack"          # Python ML packages
```

## Migration from Legacy Templates

Templates using the old `"auto"` package manager have been migrated to explicit package managers:

- ✅ `simple-python-ml.yml`: `"auto"` → `"conda"`
- ✅ `simple-r-research.yml`: `"auto"` → `"conda"`

## Implementation Details

The inheritance system is implemented in `pkg/templates/parser.go`:

- `TemplateRegistry.ResolveInheritance()`: Main resolution method
- `resolveTemplateInheritance()`: Handles single template inheritance
- `mergeTemplate()`: Implements intelligent merging rules

Templates are resolved after all templates are loaded, ensuring all parent references are available.

## Design Philosophy Alignment

Template inheritance aligns with CloudWorkstation's core design principles:

- **✅ Default to Success**: Base templates provide working defaults
- **✅ Progressive Disclosure**: Simple inheritance with advanced options available
- **✅ Zero Surprises**: Clear merging rules with predictable results
- **✅ Transparent Fallbacks**: Explicit parent relationships

This system enables the "Rocky9 linux but install some conda software on it" use case that inspired this feature.

## Template Validation

CloudWorkstation includes comprehensive template validation to catch errors early and ensure templates work correctly.

### Validation Commands

```bash
# Validate all templates
cws templates validate

# Validate specific template by name
cws templates validate "Rocky Linux 9 + Conda Stack"

# Validate template file directly
cws templates validate templates/my-template.yml
```

### Validation Rules

The validation system checks for:

#### **Required Fields**
- `name`: Template name must be specified
- `description`: Template description must be provided
- `base`: Base OS must be specified

#### **Package Manager Validation**
- Only supported package managers: `apt`, `dnf`, `conda`, `spack`, `ami`
- Package consistency: APT/DNF templates shouldn't have conda/spack packages
- AMI templates shouldn't define packages (use pre-built AMI instead)

#### **Service Validation**
- Service names must be specified
- Ports must be between 0 and 65535

#### **User Validation**
- User names must be specified
- User names cannot contain spaces or colons
- Basic format validation for system compatibility

#### **Port Validation**
- All ports must be between 1 and 65535
- Applies to both service ports and instance default ports

#### **Inheritance Validation**
- Templates cannot inherit from themselves (self-reference check)
- Parent template names cannot be empty
- Full inheritance resolution validation (missing parents, circular dependencies)

#### **Base OS Validation**
- Base OS must be supported (unless using AMI-based templates)
- AMI-based templates skip base OS validation

### Validation Examples

**Valid Template:**
```yaml
name: "My Research Environment"
description: "Python research environment with Jupyter"
base: "ubuntu-22.04"
package_manager: "conda"

packages:
  conda:
    - "python=3.11"
    - "jupyter"

users:
  - name: "researcher"
    password: "auto-generated"
    
services:
  - name: "jupyter"
    port: 8888
```

**Invalid Templates:**

```yaml
# ❌ Invalid package manager
package_manager: "invalid-manager"

# ❌ Self-reference in inheritance
inherits:
  - "My Template"  # Same as template name

# ❌ Invalid port
services:
  - name: "web"
    port: 70000  # > 65535

# ❌ Invalid user name
users:
  - name: "invalid user"  # Contains space

# ❌ Package inconsistency
package_manager: "apt"
packages:
  conda:  # APT template with conda packages
    - "python"
```

### Error Messages

The validation system provides clear, actionable error messages:

```
❌ template validation error in package_manager: 
   unsupported package manager: invalid-manager (valid: [apt dnf conda spack ami])

❌ template validation error in inherits: 
   template cannot inherit from itself: My Template

❌ template validation error in services[0].port: 
   service port must be between 0 and 65535

❌ template validation error in users[0].name: 
   user name cannot contain spaces or colons
```

### Integration with Build Process

Template validation is automatically run during:
- Template inheritance resolution
- Template loading and registry scanning
- CLI validation commands
- Template file parsing

This ensures that only valid templates are used in the system, preventing runtime errors during instance launches.