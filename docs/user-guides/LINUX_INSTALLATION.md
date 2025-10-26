# Prism Linux Installation Guide

Professional installation guide for Prism on enterprise Linux distributions using native package management.

## Quick Installation

### Ubuntu/Debian (DEB Package)

```bash
# Download and install DEB package
wget https://github.com/scttfrdmn/prism/releases/latest/download/prism_0.4.2-1_amd64.deb
sudo dpkg -i prism_0.4.2-1_amd64.deb
sudo apt-get install -f  # Fix any dependency issues

# Configure and start
sudo systemctl enable --now prism
```

### RHEL/CentOS/Fedora (RPM Package)

```bash
# Download and install RPM package
wget https://github.com/scttfrdmn/prism/releases/latest/download/prism-0.4.2-1.x86_64.rpm
sudo dnf install prism-0.4.2-1.x86_64.rpm

# Or for older systems
sudo yum install prism-0.4.2-1.x86_64.rpm

# Configure and start
sudo systemctl enable --now prism
```

## Supported Distributions

### RPM-Based Distributions

| Distribution | Version | Package Manager | Status |
|--------------|---------|-----------------|---------|
| Red Hat Enterprise Linux | 8, 9 | `dnf` | ✅ Fully Supported |
| CentOS Stream | 8, 9 | `dnf` | ✅ Fully Supported |
| Fedora | 37, 38, 39 | `dnf` | ✅ Fully Supported |
| Rocky Linux | 8, 9 | `dnf` | ✅ Fully Supported |
| AlmaLinux | 8, 9 | `dnf` | ✅ Fully Supported |
| openSUSE Leap | 15.4, 15.5 | `zypper` | ✅ Fully Supported |

### DEB-Based Distributions

| Distribution | Version | Package Manager | Status |
|--------------|---------|-----------------|---------|
| Ubuntu | 20.04, 22.04, 23.04 | `apt` | ✅ Fully Supported |
| Debian | 11, 12 | `apt` | ✅ Fully Supported |
| Linux Mint | 20, 21 | `apt` | ✅ Fully Supported |

### Architecture Support

- **x86_64 (amd64)**: Full support for Intel and AMD processors
- **ARM64 (aarch64)**: Full support for ARM-based systems including AWS Graviton

## Detailed Installation Instructions

### Pre-Installation Requirements

#### System Requirements
- Linux kernel 4.15+ (for systemd features)
- 512MB RAM minimum, 1GB recommended
- 100MB disk space for installation
- Network connectivity for AWS operations

#### Dependencies
All required dependencies are automatically installed by the package manager:
- `systemd` - Service management
- `awscli` - AWS command line interface
- `curl` - HTTP operations
- `openssh-client` - SSH connectivity (recommended)

### Step-by-Step Installation

#### 1. Download Package

**For DEB systems (Ubuntu/Debian):**
```bash
# Download latest DEB package
wget https://github.com/scttfrdmn/prism/releases/latest/download/prism_0.4.2-1_amd64.deb

# Or for ARM64
wget https://github.com/scttfrdmn/prism/releases/latest/download/prism_0.4.2-1_arm64.deb
```

**For RPM systems (RHEL/CentOS/Fedora):**
```bash
# Download latest RPM package
wget https://github.com/scttfrdmn/prism/releases/latest/download/prism-0.4.2-1.x86_64.rpm

# Or for ARM64
wget https://github.com/scttfrdmn/prism/releases/latest/download/prism-0.4.2-1.aarch64.rpm
```

#### 2. Verify Package Integrity

```bash
# Download checksums
wget https://github.com/scttfrdmn/prism/releases/latest/download/SHA256SUMS

# Verify package
sha256sum -c SHA256SUMS --ignore-missing
```

#### 3. Install Package

**Ubuntu/Debian:**
```bash
# Install with dependency resolution
sudo apt install ./prism_0.4.2-1_amd64.deb

# Or manual installation
sudo dpkg -i prism_0.4.2-1_amd64.deb
sudo apt-get install -f  # Fix dependencies if needed
```

**RHEL/CentOS/Fedora:**
```bash
# Fedora/RHEL 8+/CentOS Stream
sudo dnf install prism-0.4.2-1.x86_64.rpm

# Older RHEL/CentOS
sudo yum install prism-0.4.2-1.x86_64.rpm

# openSUSE
sudo zypper install prism-0.4.2-1.x86_64.rpm
```

#### 4. Post-Installation Configuration

After successful installation, you'll see a welcome message with next steps:

```bash
# 1. Configure AWS credentials
sudo cp /etc/prism/aws/credentials.template \
        /etc/prism/aws/credentials
sudo cp /etc/prism/aws/config.template \
        /etc/prism/aws/config

# 2. Edit AWS credentials (choose your preferred editor)
sudo nano /etc/prism/aws/credentials
sudo vim /etc/prism/aws/credentials

# 3. Start and enable the service
sudo systemctl enable --now prism

# 4. Verify installation
prism --version
prism templates
```

### AWS Configuration

#### Configure AWS Credentials

Edit `/etc/prism/aws/credentials`:
```ini
[default]
aws_access_key_id = YOUR_ACCESS_KEY
aws_secret_access_key = YOUR_SECRET_KEY

[research]
aws_access_key_id = RESEARCH_ACCESS_KEY
aws_secret_access_key = RESEARCH_SECRET_KEY
```

#### Configure AWS Settings

Edit `/etc/prism/aws/config`:
```ini
[default]
region = us-west-2
output = json

[profile research]
region = us-west-2
output = json
```

#### Security Best Practices
- Use IAM roles when possible (especially on EC2 instances)
- Rotate access keys regularly
- Use separate credentials for different environments
- Consider AWS SSO for organization-wide access
- Never commit credentials to version control

### Service Management

Prism runs as a systemd service for reliable operation:

```bash
# Start the service
sudo systemctl start prism

# Enable auto-start on boot
sudo systemctl enable prism

# Check service status
sudo systemctl status prism

# View service logs
sudo journalctl -u prism -f

# Restart the service
sudo systemctl restart prism

# Stop the service
sudo systemctl stop prism
```

### Configuration Files

#### System Configuration
- **Main config**: `/etc/prism/daemon.conf`
- **AWS config**: `/etc/prism/aws/config`
- **AWS credentials**: `/etc/prism/aws/credentials`
- **Service file**: `/lib/systemd/system/prism.service`

#### User Configuration
- **Profile config**: `~/.prism/profiles/`
- **Cache**: `~/.prism/cache/`
- **User templates**: `~/.prism/templates/`

#### System Directories
- **State data**: `/var/lib/prism/`
- **Log files**: `/var/log/prism/`
- **System user**: `prism`

### Verification and Testing

#### Basic Functionality Test
```bash
# Test CLI access
prism --version
prism --help

# Test daemon connectivity  
prism daemon status

# List available templates
prism templates

# Test AWS connectivity (requires configured credentials)
prism profiles current
```

#### Advanced Testing
```bash
# Launch a test instance (will create real AWS resources)
prism launch "Python Machine Learning" test-instance

# Check instance status
prism list

# Connect to instance
prism connect test-instance

# Clean up
prism terminate test-instance
```

## Troubleshooting

### Common Issues

#### Package Installation Fails
```bash
# For DEB: Check dependency issues
sudo apt-get install -f

# For RPM: Check for conflicts
sudo dnf check
sudo dnf clean all
```

#### Service Won't Start
```bash
# Check service status and logs
sudo systemctl status prism
sudo journalctl -u prism --no-pager

# Check configuration
sudo prism --config /etc/prism/daemon.conf --debug
```

#### AWS Authentication Errors
```bash
# Verify AWS credentials
aws configure list
aws sts get-caller-identity

# Check Prism AWS config
prism profiles current
prism daemon status
```

#### Permission Errors
```bash
# Check file permissions
sudo ls -la /etc/prism/
sudo ls -la /var/lib/prism/
sudo ls -la /var/log/prism/

# Fix permissions if needed
sudo chown -R prism:prism /var/lib/prism/
sudo chown -R prism:prism /var/log/prism/
```

### Log Files

- **System logs**: `sudo journalctl -u prism`
- **Daemon logs**: `/var/log/prism/daemon.log`
- **Package manager logs**: 
  - DEB: `/var/log/dpkg.log`
  - RPM: `/var/log/dnf.log` or `/var/log/yum.log`

### Getting Help

1. **Documentation**: https://github.com/scttfrdmn/prism
2. **Issues**: https://github.com/scttfrdmn/prism/issues
3. **Discussions**: https://github.com/scttfrdmn/prism/discussions

## Uninstallation

### Remove Package

**Ubuntu/Debian:**
```bash
# Remove package but keep configuration
sudo apt remove prism

# Remove package and configuration
sudo apt purge prism

# Remove unused dependencies
sudo apt autoremove
```

**RHEL/CentOS/Fedora:**
```bash
# Remove package
sudo dnf remove prism

# Or for older systems
sudo yum remove prism
```

### Complete Cleanup

```bash
# Remove all configuration and data (optional)
sudo rm -rf /etc/prism/
sudo rm -rf /var/lib/prism/
sudo rm -rf /var/log/prism/

# Remove system user (if no other packages depend on it)
sudo userdel prism
sudo groupdel prism
```

## Enterprise Deployment

### Configuration Management

**Ansible Example:**
```yaml
- name: Install Prism
  package:
    name: "{{ prism_package_url }}"
    state: present

- name: Configure AWS credentials
  template:
    src: aws_credentials.j2
    dest: /etc/prism/aws/credentials
    owner: root
    group: prism
    mode: '0640'

- name: Start Prism service
  systemd:
    name: prism
    enabled: yes
    state: started
```

### Repository Integration

For automated updates, consider setting up a local package repository:

**For DEB:**
```bash
# Create repository structure
mkdir -p /var/www/html/repo/deb
dpkg-scanpackages /var/www/html/repo/deb /dev/null > /var/www/html/repo/deb/Packages
gzip -k /var/www/html/repo/deb/Packages
```

**For RPM:**
```bash
# Create repository structure
mkdir -p /var/www/html/repo/rpm
createrepo /var/www/html/repo/rpm
```

## Security Considerations

### Network Security
- Prism daemon binds to `127.0.0.1:8947` by default (localhost only)
- AWS API calls use HTTPS with credential authentication
- SSH connections to instances use key-based authentication

### System Security
- Service runs as dedicated `prism` user with minimal privileges
- Systemd security hardening enabled (ProtectSystem, PrivateTmp, etc.)
- Configuration files have restricted permissions
- Comprehensive audit logging enabled

### Compliance
- Follows enterprise security best practices
- Compatible with security scanning tools
- Supports centralized logging and monitoring
- Audit trail for all operations

## Performance Tuning

### Resource Limits
Edit `/etc/systemd/system/prism.service.d/override.conf`:
```ini
[Service]
LimitNOFILE=131072
LimitNPROC=8192
MemoryLimit=1G
```

### Logging Configuration
Edit `/etc/prism/daemon.conf`:
```ini
log_level = "info"  # debug, info, warn, error
max_log_files = 10
log_file_size = "100MB"
```

Then reload systemd and restart:
```bash
sudo systemctl daemon-reload
sudo systemctl restart prism
```