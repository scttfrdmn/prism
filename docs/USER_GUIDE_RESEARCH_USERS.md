# Research Users: Your Persistent Identity Across Cloud Environments

**CloudWorkstation v0.5.0** introduces **Research Users** - your persistent identity that follows you across all your cloud research environments. No more worrying about different usernames, file permissions, or SSH keys when switching between Python, R, and other research templates.

## Quick Start

### What Are Research Users?

**Research Users** give you a consistent identity across all your CloudWorkstation instances:
- **Same username and user ID** on every instance you launch
- **Persistent home directory** that survives instance shutdowns
- **SSH keys that work everywhere** without reconfiguration
- **File sharing** that just works between different instances

### The Problem Research Users Solve

**Before Research Users:**
```bash
# Python ML instance
ssh researcher@my-python-instance    # Different username each time
ls /home/researcher                   # Files lost when instance stops

# R Research instance
ssh rstudio@my-r-instance            # Different username again!
ls /home/rstudio                      # Can't access Python files
```

**With Research Users:**
```bash
# Python ML instance
ssh alice@my-python-instance         # Your research username
ls /efs/home/alice                    # Your persistent files

# R Research instance
ssh alice@my-r-instance              # Same username!
ls /efs/home/alice                    # Same files available instantly
```

## How It Works: Dual User System

CloudWorkstation uses a **Dual User System** that gives you the best of both worlds:

### System Users (Template-Created)
- **Purpose**: Run services like Jupyter, RStudio, databases
- **Examples**: `ubuntu`, `researcher`, `rstudio`, `rocky`
- **Characteristics**: Created by templates, manage applications

### Research Users (Your Persistent Identity)
- **Purpose**: Your files, SSH access, persistent work
- **Examples**: `alice`, `bob`, `your-name`
- **Characteristics**: Same across all instances, consistent file permissions

### Real-World Example

When you launch a "Python Machine Learning" instance with research user **alice**:

```
Instance Setup:
├── ubuntu (system user)    ← SSH access, system management
├── researcher (system user) ← Runs Jupyter notebook server
└── alice (research user)   ← Your files, persistent home directory
```

**You SSH in as alice**, your files are in `/efs/home/alice`, and Jupyter runs as the `researcher` user but you can access it seamlessly.

## Getting Started

### 1. Create Your Research User

```bash
# Create your research user (coming in v0.5.0)
cws research-user create alice

# Generate SSH keys automatically
cws research-user ssh-key generate alice ed25519
```

### 2. Launch Instances with Research Users

```bash
# Launch with your research user
cws launch "Python Machine Learning" my-analysis --research-user alice

# Later, launch R instance with same user
cws launch "R Research Environment" my-analysis-r --research-user alice
```

### 3. Access Your Persistent Home Directory

```bash
# SSH into any instance
ssh alice@your-instance

# Your files are always in the same place
ls /efs/home/alice/
├── projects/          # Your research projects
├── .bashrc           # Your shell configuration
├── .ssh/             # Your SSH keys
└── research/         # Shared research data
```

## Key Benefits

### 🔄 **Cross-Template Compatibility**

Work seamlessly across different research environments:

```bash
# Day 1: Python analysis
cws launch python-ml analysis --research-user alice
ssh alice@analysis
cd /efs/home/alice/projects
python create_dataset.py  # Creates dataset.csv

# Day 2: R visualization
cws launch r-research visualization --research-user alice
ssh alice@visualization
cd /efs/home/alice/projects
R -e "data <- read.csv('dataset.csv')"  # Same file, no copying!
```

### 📁 **Persistent File Storage**

Your files persist across instance lifecycles:

```bash
# Create files on any instance
echo "Important research data" > /efs/home/alice/results.txt

# Instance shutdown/hibernation → files preserved
# Launch new instance → files immediately available
ls /efs/home/alice/results.txt  ✅ Still there!
```

### 👥 **Collaborative Research**

Multiple researchers can share files with correct permissions:

```bash
# Alice creates shared project
alice@instance: mkdir /efs/shared/team-project
alice@instance: cp analysis.py /efs/shared/team-project/

# Bob accesses same project
bob@another-instance: ls /efs/shared/team-project/
analysis.py  ← Bob can read Alice's files
```

### 🔐 **Unified SSH Access**

One set of SSH keys works everywhere:

```bash
# Generate keys once
cws research-user ssh-key generate alice ed25519

# Use same key for all instances
ssh alice@python-instance    # Works
ssh alice@r-instance         # Works
ssh alice@rocky-instance     # Works
```

## File Organization

### Your Home Directory Structure

```
/efs/home/alice/              # Your persistent home
├── .bashrc                   # Your shell preferences
├── .ssh/                     # Your SSH keys
│   └── authorized_keys
├── projects/                 # Your research projects
│   ├── ml-analysis/
│   │   ├── data.csv
│   │   ├── analysis.py
│   │   └── results.ipynb
│   └── r-visualization/
│       ├── plots.R
│       └── figures/
└── research/                 # Shared research data
    └── datasets/
```

### Shared Directories

```
/efs/shared/                  # Collaborative space
├── datasets/                 # Shared datasets
├── libraries/                # Shared code libraries
├── notebooks/                # Shared Jupyter notebooks
└── team-projects/            # Multi-user projects
```

## SSH Key Management

### Generate New Keys

```bash
# Generate Ed25519 key (recommended)
cws research-user ssh-key generate alice ed25519

# Or generate RSA key for compatibility
cws research-user ssh-key generate alice rsa
```

### Import Existing Keys

```bash
# Import your existing public key
cws research-user ssh-key import alice ~/.ssh/id_rsa.pub "My laptop key"
```

### List and Manage Keys

```bash
# List all SSH keys
cws research-user ssh-key list alice

# Delete a key
cws research-user ssh-key delete alice key-id
```

## Working with Templates

### Template Compatibility

Research users work with **all CloudWorkstation templates**:

- ✅ **Python Machine Learning**: Research user + Jupyter service user
- ✅ **R Research Environment**: Research user + RStudio service user
- ✅ **Rocky Linux Base**: Research user + rocky system user
- ✅ **Ubuntu Desktop**: Research user + ubuntu system user
- ✅ **Custom Templates**: Research user + any template-defined users

### How Templates Integrate (Phase 5A+ Template Integration)

**🎉 NEW: Automatic Research User Creation**

Templates can now automatically create and provision research users during launch! Use the `--research-user` flag with research-enabled templates:

```bash
# Automatic research user creation with new templates
cws launch python-ml-research my-project --research-user alice
# ✅ Launches instance + creates 'alice' research user + provisions SSH keys + sets up EFS home

# Check template capabilities
cws templates info python-ml-research
# Shows research user integration features
```

**Research-Enabled Templates**:

Templates specify research user integration in their configuration:

```yaml
# Example: templates/python-ml-research.yml
name: "Python ML Research (Research User Enabled)"
research_user:
  auto_create: true                       # Create research user automatically
  require_efs: true                       # Ensure EFS home directories
  efs_mount_point: "/efs"                 # EFS mount location
  efs_home_subdirectory: "research"       # Home structure: /efs/research/<username>
  install_ssh_keys: true                 # Generate and install SSH keys
  default_shell: "/bin/bash"              # Default shell for research users
  default_groups: ["research", "efs-users", "docker"]  # Research user groups
  user_integration:
    strategy: "dual_user"                 # System + research user architecture
    primary_user: "research"              # Research user is primary
    collaboration_enabled: true           # Multi-user collaboration
```

**Template Integration Benefits**:
- **One-Step Launch**: Create instance + research user + SSH keys in single command
- **EFS Auto-Setup**: Persistent home directories created automatically
- **Cross-Template Identity**: Same research user works across all templates
- **Professional Display**: Template info shows research user capabilities

## Multi-User Collaboration

### Setting Up Team Research

1. **Each team member creates their research user**:
   ```bash
   # Alice
   cws research-user create alice

   # Bob
   cws research-user create bob

   # Carol
   cws research-user create carol
   ```

2. **Share EFS volumes across instances**:
   ```bash
   # Create shared EFS volume
   cws volumes create team-research-data

   # Mount on all instances
   cws volumes mount team-research-data alice-instance
   cws volumes mount team-research-data bob-instance
   ```

3. **Collaborate with consistent permissions**:
   ```bash
   # Alice creates project (on alice-instance)
   mkdir /efs/shared/project-alpha
   echo "# Project Alpha" > /efs/shared/project-alpha/README.md

   # Bob contributes (on bob-instance)
   cd /efs/shared/project-alpha
   git clone https://github.com/team/analysis.git

   # Carol reviews (on carol-instance)
   cd /efs/shared/project-alpha/analysis
   jupyter notebook review.ipynb
   ```

## Best Practices

### File Organization

```bash
# Recommended structure
/efs/home/alice/
├── projects/          # Individual projects
│   ├── project-a/     # One directory per project
│   └── project-b/
├── scratch/           # Temporary work
├── archive/           # Completed projects
└── shared/            # Symlinks to shared directories
    ├── datasets → /efs/shared/datasets
    └── libraries → /efs/shared/libraries
```

### Security

- **Keep SSH keys secure**: Research user SSH keys protect all your instances
- **Use strong usernames**: Choose usernames that don't conflict with system users
- **Regular key rotation**: Periodically generate new SSH keys
- **Monitor access**: Check which instances your research user is active on

### Performance

- **EFS caching**: Files in `/efs/home` cached for better performance
- **Local scratch space**: Use `/tmp` or `/home/ubuntu` for temporary files
- **Shared directories**: Organize shared files to minimize EFS traffic

### Backup

- **EFS persistence**: Your `/efs/home` directory is automatically persistent
- **Regular snapshots**: Consider EFS snapshots for important research data
- **Version control**: Use git for code, research notebooks, and documentation

## Migration from Existing Setups

### Migrating Files

If you have existing research files on old instances:

```bash
# From old instance (as ubuntu/researcher)
tar -czf ~/my-research-backup.tar.gz ~/my-research-files/

# Copy to new instance
scp my-research-backup.tar.gz alice@new-instance:/efs/home/alice/

# On new instance (as alice)
cd /efs/home/alice
tar -xzf my-research-backup.tar.gz
```

### Migrating SSH Keys

```bash
# Import your existing SSH key
cws research-user ssh-key import alice ~/.ssh/id_rsa.pub "Migrated key"

# Or generate new keys and update GitHub/servers
cws research-user ssh-key generate alice ed25519
cat /efs/home/alice/.ssh/id_ed25519.pub  # Add to GitHub, servers
```

## Troubleshooting

### Common Issues

**Q: I can't SSH into my instance with my research user**
```bash
# Check SSH key is properly configured
cws research-user ssh-key list alice

# Verify user was provisioned
cws research-user status alice --instance my-instance

# Check SSH key permissions on instance
ssh ubuntu@my-instance "ls -la /efs/home/alice/.ssh/"
```

**Q: My files disappeared after launching a new instance**
```bash
# Check EFS mount
ssh alice@my-instance "mount | grep efs"

# Verify home directory
ssh alice@my-instance "ls -la /efs/home/"

# Check if EFS volume is mounted
cws volumes list
```

**Q: File permissions are wrong**
```bash
# Check research user UID consistency
ssh alice@instance1 "id alice"  # Should be same UID everywhere
ssh alice@instance2 "id alice"

# Fix permissions if needed
ssh alice@my-instance "sudo chown -R alice:research /efs/home/alice"
```

### Getting Help

```bash
# Check research user status
cws research-user status alice

# List all research users
cws research-user list

# View detailed instance information
cws instances describe my-instance --show-users
```

## What's Coming Next

**CloudWorkstation v0.5.0** will include:
- **CLI Integration**: `cws research-user` command suite
- **TUI Interface**: Visual research user management in terminal
- **GUI Support**: Point-and-click research user management
- **Template Enhancement**: Templates with built-in research user support
- **Advanced EFS**: Automatic EFS volume creation and management

**Future Enhancements**:
- **Multi-Profile Collaboration**: Share research users across CloudWorkstation profiles
- **Advanced Policies**: Institutional controls and governance
- **Usage Analytics**: Track research user activity and resource usage
- **Globus Integration**: Institutional authentication and data transfer

## Summary

Research Users transform CloudWorkstation from a single-instance tool into a collaborative research platform:

- 🎯 **Consistent Identity**: Same username and UID across all instances
- 💾 **Persistent Storage**: Files survive instance shutdowns and changes
- 🔑 **Unified Access**: One SSH key for all your research environments
- 🤝 **Collaboration**: Share files with teammates using consistent permissions
- 🔧 **Template Flexibility**: Works with any research environment template

Start using Research Users today to streamline your cloud research workflows!

---

**Need Help?**
- 📚 Full documentation: [Research User Architecture](PHASE_5A_RESEARCH_USER_ARCHITECTURE.md)
- 🐛 Report issues: [GitHub Issues](https://github.com/scttfrdmn/cloudworkstation/issues)
- 💬 Discuss: [GitHub Discussions](https://github.com/scttfrdmn/cloudworkstation/discussions)