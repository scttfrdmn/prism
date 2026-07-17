# Prism Plugin System

The Prism plugin system provides **tool-agnostic, reusable components** for templates. Instead of embedding tool-specific installation logic in every template, plugins enable clean composition and consistent behavior across templates.

## Table of Contents

- [Overview](#overview)
- [Using Plugins in Templates](#using-plugins-in-templates)
- [Available Plugins](#available-plugins)
- [Creating New Plugins](#creating-new-plugins)
- [Plugin Architecture](#plugin-architecture)
- [Migration Guide](#migration-guide)

---

## Overview

### What Are Plugins?

Plugins are **YAML manifests** that define:
- **Tool installation** for multiple package managers (apt, dnf, conda, pip, etc.)
- **Configuration parameters** with defaults and validation
- **Dependencies and conflicts** for proper resolution
- **Post-installation hooks** for setup and verification

### Why Use Plugins?

**Before Plugins** (Template-Embedded Logic):
```yaml
# templates/python-ml.yml
packages:
  system:
    - python3
    - python3-pip
post_install: |
  pip3 install jupyter numpy pandas
  # Repeat this in every Python template...
```

**With Plugins** (Composable):
```yaml
# templates/python-ml.yml
plugins:
  enabled:
    - name: "python"
      version: "3.12"
      configuration:
        install_jupyter: true
        install_data_science: true
```

**Benefits**:
- ✅ **DRY Principle**: Define installation once, reuse everywhere
- ✅ **Consistency**: Same installation across all templates
- ✅ **Maintainability**: Update plugin → all templates benefit
- ✅ **Multi-Tool Support**: apt, dnf, conda, pip—one plugin handles all

---

## Using Plugins in Templates

### Basic Plugin Usage

Add plugins to any template's `plugins.enabled` section:

```yaml
name: "My Development Environment"
base: "ubuntu-22.04"
package_manager: "apt"

plugins:
  enabled:
    - name: "git"
      version: "latest"
      configuration:
        enable_lfs: true

    - name: "docker"
      version: "latest"
      configuration:
        add_user_to_group: true
        user: "developer"

users:
  - name: "developer"
    groups: ["sudo"]
```

### Plugin Configuration

Each plugin accepts **configuration parameters**:

```yaml
plugins:
  enabled:
    - name: "python"
      version: "3.12"
      configuration:
        upgrade_pip: true              # Upgrade pip after install
        install_dev_tools: true        # Install black, pylint, mypy
        install_jupyter: false         # Skip Jupyter
        install_data_science: true     # Install numpy, pandas, etc.

    - name: "nodejs"
      version: "20"
      configuration:
        install_yarn: true
        install_typescript: true
        install_dev_tools: true

    - name: "postgresql"
      version: "16"
      configuration:
        install_server: true
        create_default_db: true
        database_name: "myapp_db"
        database_user: "developer"
        database_password: "changeme"
```

### Package Manager Compatibility

Plugins **automatically adapt** to your template's package manager:

```yaml
# Template 1: Ubuntu with APT
base: "ubuntu-22.04"
package_manager: "apt"
plugins:
  enabled:
    - name: "python"  # Installs via apt-get

# Template 2: Rocky Linux with Conda
base: "rocky-9"
package_manager: "conda"
plugins:
  enabled:
    - name: "python"  # Installs via conda
```

---

## Available Plugins

### Development Tools

#### **git** - Git Version Control
```yaml
- name: "git"
  version: "latest"  # or "2.40", "2.39"
  configuration:
    enable_lfs: true  # Install Git LFS
```

**Provides**: git, git-lfs (optional)
**Package Managers**: apt, dnf, conda, system

---

#### **python** - Python Programming Language
```yaml
- name: "python"
  version: "3.12"  # or "3.11", "3.10", "latest"
  configuration:
    upgrade_pip: true
    install_dev_tools: true    # black, pylint, mypy, pytest, ipython
    install_jupyter: false     # Jupyter Lab + Notebook
    install_data_science: true # numpy, pandas, matplotlib, etc.
```

**Provides**: python3, pip, venv
**Package Managers**: apt, dnf, conda, pip

---

#### **nodejs** - Node.js Runtime
```yaml
- name: "nodejs"
  version: "20"  # or "18", "16", "latest"
  configuration:
    install_yarn: true
    install_pnpm: false
    install_typescript: true
    install_dev_tools: true  # eslint, prettier, nodemon
```

**Provides**: node, npm, npx
**Package Managers**: apt, dnf, conda, system

---

### Infrastructure Tools

#### **docker** - Docker Container Platform
```yaml
- name: "docker"
  version: "latest"  # or "24.0.7", "24.0.6", "23.0.6"
  configuration:
    add_user_to_group: true
    user: "researcher"
    enable_buildx: true
```

**Provides**: docker, docker-compose
**Package Managers**: apt, dnf, system

---

### Database Tools

#### **mysql** - MySQL/MariaDB Database
```yaml
- name: "mysql"
  version: "latest"  # or "8.0", "5.7", "10.11", "10.6"
  configuration:
    flavor: "mysql"  # or "mariadb"
    install_server: true
    create_default_db: true
    database_name: "myapp_db"
    database_user: "developer"
    database_password: "changeme"
```

**Provides**: mysql client, mysql-server (optional)
**Package Managers**: apt, dnf, conda (client only), system

---

#### **postgresql** - PostgreSQL Database
```yaml
- name: "postgresql"
  version: "16"  # or "15", "14", "13", "latest"
  configuration:
    install_server: true
    create_default_db: true
    database_name: "myapp_db"
    database_user: "developer"
    database_password: "changeme"
```

**Provides**: psql client, postgresql-server (optional)
**Package Managers**: apt, dnf, conda (client only), system

---

## Creating New Plugins

### Plugin Manifest Structure

Create `plugins/<name>.yml`:

```yaml
apiVersion: v1
kind: Plugin
metadata:
  name: "plugin-name"
  display_name: "Human-Readable Name"
  description: "Brief description"
  category: "development"  # development, database, infrastructure
  version: "1.0.0"
  maintainer: "Prism Team"

spec:
  # What capabilities does this plugin provide?
  provides:
    - name: "tool_name"
      type: "capability"
    - name: "feature_name"
      type: "feature"

  # Dependencies on other plugins
  dependencies:
    required: []
    optional:
      - plugin: "git"
        version: ">=2.0.0"

  # Conflicts with other plugins
  conflicts:
    - plugin: "conflicting-plugin"
      reason: "Both provide same functionality"

  # Alternative plugins
  alternatives:
    - plugin: "alternative-plugin"
      reason: "Similar functionality with different approach"

  # Installation for each package manager
  installers:
    apt:
      package_manager: "apt"
      packages:
        - name: "package-name"
          version_param: "version"
          install_command: |
            apt-get update
            {{if eq .version "latest"}}
            apt-get install -y package-name
            {{else}}
            apt-get install -y package-name={{.version}}
            {{end}}

      post_install: |
        # Verify installation
        package-name --version

        {{if .some_config_option}}
        # Optional configuration
        echo "Configuring..."
        {{end}}

    dnf:
      package_manager: "dnf"
      packages:
        - name: "package-name"
          install_command: |
            dnf install -y package-name

    system:
      package_manager: "system"
      packages:
        - name: "package-name"
          install_command: |
            if command -v apt-get &> /dev/null; then
              apt-get install -y package-name
            elif command -v dnf &> /dev/null; then
              dnf install -y package-name
            fi

  # Configuration parameters
  configuration:
    parameters:
      version:
        type: "choice"
        choices: ["latest", "1.0", "2.0"]
        default: "latest"
        description: "Version to install"

      some_option:
        type: "bool"
        default: true
        description: "Enable some feature"

  # Compatibility requirements
  compatibility:
    tools:
      system:
        min_version: "1.0.0"

  # Security metadata
  security:
    trusted: true
    verified_publisher: true
    cve_scanned: true
    last_security_scan: "2024-01-15T10:00:00Z"
```

### Template Variables

Use Go template syntax in install scripts:

**Version Selection**:
```yaml
{{if eq .version "latest"}}
install-package
{{else if eq .version "1.0"}}
install-package-1.0
{{else}}
install-package={{.version}}
{{end}}
```

**Boolean Flags**:
```yaml
{{if .enable_feature}}
# Install optional feature
install-feature-package
{{end}}
```

**String Parameters**:
```yaml
create-user {{.username}}
set-password '{{.password}}'
```

### Heredoc Syntax in YAML

**⚠️ CRITICAL**: When using heredoc in YAML multiline strings, follow these rules:

1. **Use quoted delimiter** to prevent shell variable expansion:
   ```yaml
   post_install: |
     mysql -u root <<'EOSQL'  # Note the single quotes!
     CREATE DATABASE {{.database_name}};
     EOSQL
   ```

2. **Indent heredoc content** to match YAML block:
   ```yaml
   post_install: |
     cat > /etc/config <<'EOF'
     setting1={{.value1}}
     setting2={{.value2}}
     EOF
   ```

**Why?** YAML requires consistent indentation for multiline strings. Without proper indentation, the YAML parser fails with "could not find expected ':'".

### Testing Plugins

1. **Create test template**:
   ```yaml
   # templates/testing/plugin-test.yml
   name: "Plugin Test"
   base: "ubuntu-22.04"
   package_manager: "apt"
   plugins:
     enabled:
       - name: "your-plugin"
         version: "latest"
   ```

2. **Validate**:
   ```bash
   ./bin/prism templates validate "Plugin Test"
   ```

3. **Test on real instance**:
   ```bash
   ./bin/prism workspace launch "Plugin Test" plugin-test
   ./bin/prism workspace exec plugin-test 'your-command --version'
   ```

---

## Plugin Architecture

### Discovery and Loading

1. **Plugin directories** (searched in order):
   - `PRISM_PLUGIN_DIR` environment variable
   - `<binary-dir>/../plugins` (development)
   - `/opt/homebrew/share/prism/plugins` (Homebrew)
   - `/usr/local/share/prism/plugins` (system)
   - `/usr/share/prism/plugins` (system)

2. **Registry initialization**:
   ```go
   pluginRegistry := templates.NewPluginRegistry()
   pluginRegistry.LoadPluginsFromDirectory("/path/to/plugins")
   ```

3. **Template resolution**:
   - Parser loads template YAML
   - Resolver processes `plugins.enabled`
   - Script generator creates installation script
   - Plugins installed before `post_install`

### Script Generation

Plugins generate scripts for the template's package manager:

```bash
# Generated script (simplified)
#!/bin/bash
set -e

# === PLUGIN INSTALLATION ===
echo "🔌 Installing plugins..."

# Plugin: git
echo "📦 Installing plugin: git"
apt-get update
apt-get install -y git
git lfs install --system

# Plugin: docker
echo "📦 Installing plugin: docker"
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
apt-get update
apt-get install -y docker-ce docker-ce-cli containerd.io
systemctl enable docker
systemctl start docker
usermod -aG docker developer

# === SYSTEM PACKAGES ===
apt-get install -y curl wget vim

# === POST INSTALL ===
echo "Running post-install script..."
# ... template's post_install script ...
```

---

## Migration Guide

### From Embedded Installation to Plugins

**Before** (template with embedded logic):
```yaml
name: "Python ML Environment"
base: "ubuntu-22.04"
package_manager: "apt"

packages:
  system:
    - python3
    - python3-pip
    - python3-venv
    - python3-dev

post_install: |
  # Upgrade pip
  python3 -m pip install --upgrade pip

  # Install development tools
  pip3 install black pylint mypy pytest ipython

  # Install Jupyter
  pip3 install jupyter jupyterlab notebook

  # Install data science packages
  pip3 install numpy pandas matplotlib seaborn scikit-learn

  # Verify
  python3 --version
  jupyter --version
```

**After** (plugin-based):
```yaml
name: "Python ML Environment"
base: "ubuntu-22.04"
package_manager: "apt"

plugins:
  enabled:
    - name: "python"
      version: "3.12"
      configuration:
        upgrade_pip: true
        install_dev_tools: true
        install_jupyter: true
        install_data_science: true

post_install: |
  # Only custom configuration here
  echo "Python ML environment ready!"
```

### Migration Checklist

- [ ] Identify tool installations in `post_install`
- [ ] Check if plugin exists for the tool
- [ ] Move tool configuration to `plugins.enabled`
- [ ] Remove installation logic from `post_install`
- [ ] Keep only template-specific configuration
- [ ] Test template with plugin
- [ ] Verify all functionality works

---

## Best Practices

### Plugin Design

1. **Single Responsibility**: One tool per plugin
2. **Sensible Defaults**: Most users shouldn't need configuration
3. **Multi-Tool Support**: Support apt, dnf, conda where applicable
4. **Verification**: Always verify installation in `post_install`
5. **Idempotency**: Scripts should be safe to run multiple times

### Template Design

1. **Prefer Plugins**: Use plugins instead of manual installation
2. **Minimal post_install**: Keep template-specific logic only
3. **Clear Configuration**: Document plugin parameters
4. **Test Coverage**: Validate on real instances before committing

### Security

1. **Trusted Sources**: Only install from official repositories
2. **No Secrets**: Never embed passwords or API keys
3. **Verify GPG Keys**: Check repository signatures
4. **CVE Scanning**: Keep security metadata updated

---

## Examples

### Full-Stack Development Template

```yaml
name: "Full-Stack Dev Environment"
base: "ubuntu-22.04"
package_manager: "apt"

plugins:
  enabled:
    - name: "git"
      version: "latest"
      configuration:
        enable_lfs: true

    - name: "python"
      version: "3.12"
      configuration:
        upgrade_pip: true
        install_dev_tools: true

    - name: "nodejs"
      version: "20"
      configuration:
        install_yarn: true
        install_typescript: true

    - name: "postgresql"
      version: "16"
      configuration:
        install_server: true
        create_default_db: true
        database_name: "dev_db"
        database_user: "developer"
        database_password: "dev_password"

    - name: "docker"
      version: "latest"
      configuration:
        add_user_to_group: true
        user: "developer"

packages:
  system:
    - curl
    - wget
    - vim
    - tmux

users:
  - name: "developer"
    groups: ["sudo", "docker"]
    shell: "/bin/bash"

ports:
  - 22    # SSH
  - 3000  # Node.js
  - 5000  # Python
  - 5432  # PostgreSQL
  - 8000  # Django/Flask

post_install: |
  # Create project structure
  mkdir -p /home/developer/{projects,data,scripts}
  chown -R developer:developer /home/developer

  echo "✅ Full-stack environment ready!"
```

### Data Science Template

```yaml
name: "Data Science Workstation"
base: "ubuntu-22.04"
package_manager: "apt"

plugins:
  enabled:
    - name: "python"
      version: "3.12"
      configuration:
        upgrade_pip: true
        install_dev_tools: true
        install_jupyter: true
        install_data_science: true

    - name: "git"
      version: "latest"
      configuration:
        enable_lfs: true

users:
  - name: "datascientist"
    groups: ["sudo"]
    shell: "/bin/bash"

ports:
  - 22
  - 8888  # Jupyter

post_install: |
  # Configure Jupyter
  su - datascientist -c "jupyter notebook --generate-config"

  echo "✅ Data science environment ready!"
```

---

## Troubleshooting

### Plugin Not Found

```
Error: plugin "unknown-plugin" not found in registry
```

**Solution**: Check plugin name and ensure plugin file exists in plugin directory.

### YAML Parse Error

```
Warning: failed to load plugin foo.yml: yaml: line 147: could not find expected ':'
```

**Solution**: Check heredoc indentation. Ensure heredoc content is indented to match YAML block.

### Version Not Available

```
Error: package python3.99 not available
```

**Solution**: Check `choices` in plugin `configuration.parameters.version` for valid versions.

### Permission Denied

```
Error: cannot add user to docker group: permission denied
```

**Solution**: Ensure scripts run with `sudo` privileges. Check template uses correct user configuration.

---

## Reference

### Plugin Manifest Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | string | Yes | Must be "v1" |
| `kind` | string | Yes | Must be "Plugin" |
| `metadata.name` | string | Yes | Plugin identifier |
| `metadata.display_name` | string | Yes | Human-readable name |
| `metadata.description` | string | Yes | Brief description |
| `metadata.category` | string | Yes | Plugin category |
| `spec.provides` | array | Yes | Capabilities provided |
| `spec.installers` | object | Yes | Package manager installers |
| `spec.configuration` | object | No | Configuration parameters |
| `spec.dependencies` | object | No | Plugin dependencies |
| `spec.conflicts` | object | No | Conflicting plugins |

### Configuration Parameter Types

| Type | Description | Example |
|------|-------------|---------|
| `choice` | Fixed list of options | `["latest", "1.0", "2.0"]` |
| `bool` | Boolean flag | `true` or `false` |
| `string` | Free-form text | `"my-database"` |

---

## Contributing

Want to add a new plugin?

1. Create plugin manifest in `plugins/<name>.yml`
2. Follow plugin structure guidelines
3. Test on at least one package manager
4. Submit PR with:
   - Plugin YAML file
   - Test template
   - Documentation updates

See CONTRIBUTING.md for details.

---

## Support

- **Documentation**: [docs/plugins.md](plugins.md)
- **Examples**: [templates/testing/](../templates/testing/)
- **Issues**: [GitHub Issues](https://github.com/scttfrdmn/prism/issues)
- **Discussions**: [GitHub Discussions](https://github.com/scttfrdmn/prism/discussions)
