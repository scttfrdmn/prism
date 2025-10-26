# Template Dependency Resolution System

Prism's template dependency system enables the creation of modular, composable templates with clear relationships between them. This document explains how the dependency resolution system works and how to use it effectively.

## Key Concepts

### Semantic Versioning

All templates follow [Semantic Versioning](https://semver.org/) (SemVer) in the format `MAJOR.MINOR.PATCH`:

- **MAJOR**: Incremented for incompatible API changes
- **MINOR**: Incremented for new functionality in a backward compatible manner
- **PATCH**: Incremented for backward compatible bug fixes

### Dependencies

Templates can depend on other templates using the `dependencies` field:

```yaml
name: python-ml
description: Machine learning template with Python
dependencies:
  - name: python-base
    version: 2.0.0
    version_operator: ">="
  - name: cuda-drivers
    version: 1.5.0
    version_operator: ">="
    optional: true
```

### Version Constraints

Templates can specify version constraints for dependencies:

| Operator | Description | Example |
|----------|-------------|---------|
| `=` or `==` | Exact version | `=1.2.3` (exactly version 1.2.3) |
| `>=` | Greater than or equal to | `>=2.0.0` (version 2.0.0 or higher) |
| `>` | Greater than | `>1.0.0` (any version above 1.0.0) |
| `<=` | Less than or equal to | `<=3.0.0` (version 3.0.0 or lower) |
| `<` | Less than | `<2.0.0` (any version below 2.0.0) |
| `~>` | Compatible version | `~>1.2.0` (any 1.2.x version but not 1.3.0) |

If no operator is specified, `>=` is used by default.

### Optional Dependencies

Dependencies can be marked as optional using the `optional: true` flag. These are not required for the template to function but provide additional capabilities when available.

## Dependency Resolution Process

The dependency resolver performs these steps:

1. **Validation**: Checks all dependencies exist and are accessible
2. **Version Constraint Checking**: Validates all versions against their constraints
3. **Conflict Resolution**: Handles competing version requirements for the same template
4. **Build Order Generation**: Creates a deterministic build sequence
5. **Fetching**: Optionally retrieves missing dependencies from the registry

### Build Order Algorithm

The dependency graph is generated using a modified topological sort:

1. Start with the target template
2. Recursively add all dependencies
3. Sort the graph so dependencies are built before templates that depend on them
4. Detect and prevent circular dependencies

## Using the CLI Tools

### Analyzing Template Dependencies

```bash
# View dependencies for a template
prism ami template dependency list my-template

# Check if dependencies are satisfied
prism ami template dependency check my-template

# Analyze dependencies in detail
prism ami template dependency analyze my-template
```

### Resolving Dependencies

```bash
# Resolve dependencies (without fetching)
prism ami template dependency resolve my-template

# Resolve and fetch missing dependencies
prism ami template dependency resolve my-template --fetch

# View dependency graph
prism ami template dependency graph my-template
```

### Working with Template Versions

```bash
# List all versions of a template
prism ami template version list my-template

# Search for versions matching criteria
prism ami template version search my-template --min-version 2.0.0

# Compare two versions
prism ami template version compare 1.2.3 2.0.0
```

## Managing Dependencies

### Adding Dependencies

```bash
# Add a dependency with version constraint
prism ami template dependency add my-template dependency-name --version 1.0.0 --operator ">="

# Add an optional dependency
prism ami template dependency add my-template optional-dep --optional
```

### Removing Dependencies

```bash
# Remove a dependency
prism ami template dependency remove my-template dependency-name
```

### Incrementing Template Versions

```bash
# Increment major version (breaking changes)
prism ami template version increment my-template major

# Increment minor version (new features)
prism ami template version increment my-template minor

# Increment patch version (bug fixes)
prism ami template version increment my-template patch
```

## Best Practices

1. **Be Specific**: Always specify version constraints for dependencies
2. **Minimize Dependencies**: Only include necessary dependencies
3. **Proper Versioning**: Follow SemVer guidelines for version increments
4. **Compatible Constraints**: Use `~>` for compatible version requirements
5. **Document Relationships**: Clearly document why each dependency is needed
6. **Test Dependency Graph**: Validate the full dependency chain before publishing
7. **Avoid Circular Dependencies**: Ensure dependencies form a directed acyclic graph (DAG)
8. **Mark Optional Dependencies**: Use the optional flag appropriately

## Advanced Topics

### Conflict Resolution Strategy

When different templates require conflicting versions of the same dependency, the resolver uses these rules:

1. Exact versions (`=`) take precedence
2. For lower bounds (`>=`, `>`), the highest version is used
3. For upper bounds (`<=`, `<`), the lowest version is used
4. If a version satisfies all constraints, it's selected
5. If no version satisfies all constraints, an error is reported

### Registry Integration

The dependency resolver can automatically fetch missing dependencies from the Prism template registry when the `--fetch` option is used.

### Dependency Analysis

Use the `analyze` command to get insights about:
- Total dependencies
- Missing dependencies (both required and optional)
- Version mismatches
- Whether the template is buildable

## Error Handling

Common error messages and their solutions:

- **Circular dependency detected**: Restructure your templates to remove the cycle
- **No version satisfies all constraints**: Update constraints to be compatible
- **Missing required dependency**: Add the dependency or mark it as optional
- **Version mismatch**: Update your template to be compatible with available versions
- **Template not found**: Check if the template name is correct or needs to be fetched from the registry

---

The dependency resolution system ensures Prism templates can be composed reliably while maintaining compatibility between different components. This aligns with Prism's core design principles of Default to Success and Transparent Fallbacks.