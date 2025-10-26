# Prism Dual User Architecture

**The Foundation for Collaborative Research Computing**

## Executive Summary

Prism's **Dual User Architecture** solves the fundamental challenge of providing both **template flexibility** and **research continuity** in cloud computing environments. By separating system users (template-created) from research users (persistent identity), researchers can work seamlessly across different computational environments while maintaining consistent file permissions and access patterns.

## The Challenge

### Before Dual User Architecture

Traditional cloud research environments force researchers to choose between:

**Option A: Template Flexibility**
- Different templates create different users (`ubuntu`, `researcher`, `rstudio`, `rocky`)
- Each instance has different usernames and UIDs
- Files cannot be shared between instances
- SSH keys need separate management per template

**Option B: User Consistency**
- Use the same generic user everywhere
- Lose template-specific optimizations
- Services run as wrong user type
- Complex manual configuration required

### The Problem in Action

```bash
# Monday: Python ML analysis
ssh researcher@ml-instance      # UID 1001
echo "results" > analysis.csv   # File owned by 1001

# Tuesday: R visualization
ssh rstudio@r-instance         # UID 1002 (different!)
ls analysis.csv                # Permission denied! Different UID
```

**Result**: Researchers spend time managing files instead of doing research.

## The Dual User Solution

### Architecture Overview

```
┌─────────────────────────┐    ┌─────────────────────────┐
│      System Users       │    │     Research Users      │
├─────────────────────────┤    ├─────────────────────────┤
│ • Template-created      │    │ • Profile-created       │
│ • Service-focused       │    │ • User-focused          │
│ • Variable UIDs         │    │ • Consistent UIDs       │
│ • Instance-specific     │    │ • Cross-instance        │
│                         │    │                         │
│ ubuntu (1000)          │    │ alice (5001)            │
│ researcher (1001)      │    │ bob (5002)              │
│ rstudio (1002)         │    │ carol (5003)            │
│ rocky (1003)           │    │                         │
└─────────────────────────┘    └─────────────────────────┘
```

### How They Work Together

**Python ML Instance:**
```
Users on Instance:
├── ubuntu (1000)          ← System administration
├── researcher (1001)      ← Runs Jupyter notebook
└── alice (5001)          ← Your research files & SSH access
```

**R Research Instance:**
```
Users on Instance:
├── ubuntu (1000)          ← System administration
├── rstudio (1002)         ← Runs RStudio server
└── alice (5001)          ← Same research user, same files!
```

## Key Benefits

### 1. 🔄 **Cross-Template Compatibility**

Work seamlessly across different research environments:

**Workflow Example:**
```bash
# Day 1: Data preprocessing with Python
ssh alice@python-instance
python preprocess_data.py      # Creates dataset.parquet
```

```bash
# Day 2: Statistical analysis with R
ssh alice@r-instance           # Same username!
R -e "data <- read_parquet('dataset.parquet')"  # Same file!
```

**Benefits:**
- ✅ Same SSH access across all templates
- ✅ Files immediately available on new instances
- ✅ No permission conflicts or file copying
- ✅ Seamless workflow continuation

### 2. 📁 **Persistent File Ownership**

Consistent file permissions enable true collaboration:

**File Ownership Consistency:**
```bash
# Alice creates file on Python instance
alice@python-instance: touch /efs/shared/analysis.py
ls -l /efs/shared/analysis.py
-rw-r--r-- 1 alice research 0 analysis.py  # UID 5001

# File accessible from R instance with same ownership
alice@r-instance: ls -l /efs/shared/analysis.py
-rw-r--r-- 1 alice research 0 analysis.py       # Still UID 5001!
```

**Benefits:**
- ✅ Files owned by consistent user ID across instances
- ✅ EFS volumes work seamlessly between instances
- ✅ No permission denied errors
- ✅ Backup and sync tools work correctly

### 3. 👥 **Multi-User Collaboration**

Multiple researchers can share resources with predictable permissions:

**Team Collaboration Example:**
```bash
# Alice (UID 5001) creates shared project
alice@instance1: mkdir /efs/shared/team-project
alice@instance1: echo "Alice's analysis" > team-project/analysis.py

# Bob (UID 5002) contributes from different instance
bob@instance2: cd /efs/shared/team-project
bob@instance2: echo "Bob's visualization" > visualization.py

# Carol (UID 5003) reviews on third instance
carol@instance3: ls -la /efs/shared/team-project/
-rw-r--r-- 1 alice research analysis.py      # Alice's file
-rw-r--r-- 1 bob   research visualization.py # Bob's file
```

**Benefits:**
- ✅ Clear file ownership for accountability
- ✅ Consistent permissions across all instances
- ✅ Multi-user access to shared directories
- ✅ Backup systems preserve user ownership

### 4. 🎛️ **Service Optimization**

Templates can optimize system users for specific services:

**Service User Specialization:**
```bash
# Python ML Template
researcher (1001)  ← Optimized for Jupyter, conda environments
alice (5001)       ← Your files, SSH access

# R Research Template
rstudio (1002)     ← Optimized for RStudio Server, R packages
alice (5001)       ← Same user, same files

# Rocky Linux Template
rocky (1003)       ← Optimized for RHEL-style administration
alice (5001)       ← Same user, same files
```

**Benefits:**
- ✅ Templates retain full flexibility
- ✅ Services run as appropriate specialized users
- ✅ Research users get consistent experience
- ✅ No compromise on template optimization

## Technical Implementation

### UID/GID Allocation

**Research User Range:** 5000-5999 (1000 users)
**System User Range:** 1000-4999 (templates)

**Deterministic Allocation:**
```go
// Same profile + username = same UID everywhere
func allocateUID(profileID, username string) int {
    hash := sha256.Sum256([]byte(profileID + ":" + username))
    offset := binary.BigEndian.Uint64(hash[:8])
    return 5000 + int(offset % 1000)
}

// Example:
// "personal-research:alice" → UID 5001 (always)
// "lab-shared:alice"       → UID 5102 (different profile)
```

### EFS Home Directory Structure

```
/efs/                          # EFS mount point
├── home/                      # Research user homes
│   ├── alice/                 # alice (5001) home directory
│   │   ├── .bashrc
│   │   ├── .ssh/
│   │   └── projects/
│   ├── bob/                   # bob (5002) home directory
│   │   ├── .bashrc
│   │   └── projects/
│   └── carol/                 # carol (5003) home directory
└── shared/                    # Collaborative directories
    ├── datasets/              # Shared data
    └── team-projects/         # Multi-user projects
```

### SSH Key Management

**Per-Profile Key Storage:**
```
~/.prism/ssh-keys/
├── personal-research/
│   ├── alice/
│   │   ├── key1.pub
│   │   └── key1.json (metadata)
│   └── bob/
└── lab-shared/
    └── alice/                 # Different profile = different keys
```

## Real-World Use Cases

### Use Case 1: Individual Researcher

**Dr. Sarah Chen - Computational Biology**

**Challenge:** Sarah needs to preprocess genomic data with Python, analyze it with R, and visualize results with specialized bioinformatics tools.

**Before Dual User Architecture:**
```bash
# Preprocessing instance (Python)
ssh ubuntu@preprocess-instance
sudo -u researcher python preprocess.py     # Files owned by researcher:researcher

# Analysis instance (R)
ssh ubuntu@analysis-instance
sudo cp /shared/data.csv /home/rstudio/     # Manual file copying
sudo chown rstudio:rstudio /home/rstudio/data.csv
sudo -u rstudio R < analysis.R

# Visualization instance (specialized tools)
ssh ubuntu@viz-instance
# More manual copying and permission fixing...
```

**With Dual User Architecture:**
```bash
# Preprocessing instance
ssh sarah@preprocess-instance
python preprocess.py                        # Files in /efs/home/sarah/

# Analysis instance
ssh sarah@analysis-instance                 # Same user!
R < analysis.R                              # Same files, no copying!

# Visualization instance
ssh sarah@viz-instance                      # Same user!
./visualize_results.py                      # Same files, seamless workflow!
```

### Use Case 2: Research Team

**AI Research Lab - 5 Researchers**

**Challenge:** Team needs to collaborate on large language model training, with different researchers using different tools (Python, R, Julia) and sharing datasets, models, and results.

**Team Setup:**
```bash
# Research users with consistent UIDs
alice (5001)    # Lead researcher - Python/PyTorch
bob (5002)      # Data scientist - R/tidyverse
carol (5003)    # ML engineer - Julia/Flux.jl
david (5004)    # Statistician - R/Stan
eve (5005)      # Visualization - Python/D3
```

**Collaboration Workflow:**
```bash
# Alice preprocesses data on Python instance
alice@gpu-cluster: python prepare_training_data.py
# Creates /efs/shared/datasets/llm_training.jsonl (owned by alice:research)

# Bob analyzes data statistics on R instance
bob@stats-instance: cd /efs/shared/datasets
bob@stats-instance: R -e "data <- jsonlite::read_json('llm_training.jsonl', simplifyVector=TRUE)"

# Carol trains model on GPU instance
carol@gpu-instance: julia train_model.jl /efs/shared/datasets/llm_training.jsonl
# Creates /efs/shared/models/llm_v1.bson (owned by carol:research)

# Eve creates visualizations
eve@viz-instance: python plot_training_curves.py /efs/shared/models/llm_v1.bson
```

**Benefits Realized:**
- ✅ No file permission issues between team members
- ✅ Clear ownership and accountability for datasets/models
- ✅ Each researcher uses their preferred tools
- ✅ Seamless handoffs between workflow stages

### Use Case 3: Educational Institution

**University Research Computing - 200 Students**

**Challenge:** Computer Science department needs to provide consistent research environments for students across different courses (Python ML, R Statistics, Systems Programming).

**Before Dual User Architecture:**
```bash
# Students get confused by different usernames per class
CS501-Python:  ssh student@ml-instance      # Different user each class
CS502-R:       ssh rstudio@stats-instance   # Different SSH keys
CS503-Systems: ssh ubuntu@systems-instance  # Different home directories
```

**With Dual User Architecture:**
```bash
# Each student gets one consistent research identity
alice-student: ssh alice@ml-instance        # CS501 Python ML
alice-student: ssh alice@stats-instance     # CS502 R Statistics
alice-student: ssh alice@systems-instance   # CS503 Systems Programming

# Same files, same SSH keys, same environment
ls /efs/home/alice/courses/
├── cs501-ml/
├── cs502-stats/
└── cs503-systems/
```

**Benefits for Institution:**
- ✅ Simplified user management (one account per student)
- ✅ Consistent backup and monitoring
- ✅ Students focus on learning, not technical setup
- ✅ Cross-course project collaboration enabled

## Migration and Adoption

### Backward Compatibility

**Existing Templates Work Unchanged:**
```yaml
# Existing template (no changes needed)
name: "Python Machine Learning"
users:
  - name: "researcher"
    groups: ["sudo"]
# → Still creates researcher user as before
```

**Enhanced Templates (Optional):**
```yaml
# Enhanced template with research user integration
name: "Python ML + Research User"
users:
  - name: "researcher"
    groups: ["sudo"]

# New: Research user integration
research_user:
  auto_create: true
  primary_user: true
  shared_directories: ["/opt/notebooks"]
```

### Migration Path

**Phase 1: Parallel Operation**
- Existing instances continue unchanged
- New instances can opt-in to research users
- Templates support both modes

**Phase 2: Enhanced Integration**
- CLI/TUI/GUI interfaces add research user management
- Templates enhanced with research user features
- EFS integration becomes automatic

**Phase 3: Default Operation**
- Research users become default for new profiles
- Legacy mode available for existing setups
- Full collaborative features enabled

## Performance Considerations

### UID/GID Allocation Performance

- **Hash-based allocation**: O(1) average case
- **Collision resolution**: O(n) worst case, rare in practice
- **Caching**: Allocations cached for repeated access
- **Scalability**: Supports 1000 research users per installation

### Storage Performance

- **EFS home directories**: Leverages EFS caching and performance modes
- **Local scratch space**: System users can use local storage for temp files
- **Hybrid approach**: Critical files on EFS, temporary files local

### Network Performance

- **SSH connection reuse**: Multiple provisioning operations share connections
- **Parallel provisioning**: Multiple users can be set up simultaneously
- **Optimized scripts**: Generated provisioning scripts minimize remote execution time

## Security Model

### Isolation and Access Control

**User Range Isolation:**
- System users: UIDs 1000-4999
- Research users: UIDs 5000-5999
- Clear separation prevents conflicts

**Profile-Based Security:**
- Research users belong to specific Prism profiles
- SSH keys isolated per profile
- Cross-profile access requires explicit sharing

**EFS Permissions:**
```bash
# Home directory permissions
/efs/home/alice → alice:research (750)  # Private
/efs/shared     → root:research (755)   # Collaborative
```

### SSH Key Security

- **Per-profile key generation**: Keys never shared across profiles
- **Secure storage**: Private keys encrypted and access-controlled
- **Key rotation**: Support for key replacement and deactivation
- **Audit trail**: Key usage and access logging

## Future Enhancements

### Advanced Collaboration

**Multi-Profile Research Users:** Share research users across Prism profiles for inter-institutional collaboration.

**Advanced Access Control:** Fine-grained permissions for shared directories and resources.

**Usage Analytics:** Track research user activity, resource usage, and collaboration patterns.

### Enterprise Features

**Policy Integration:** Institutional controls over research user creation and access.

**Quota Management:** Per-user storage and compute quotas with monitoring.

**Automated Provisioning:** Integration with institutional identity providers (LDAP, Active Directory).

### Performance Optimizations

**Database Storage:** Move from file-based to database storage for large deployments.

**Distributed Caching:** Redis-based caching for multi-node Prism deployments.

**Async Provisioning:** Background user provisioning with progress tracking.

## Conclusion

The Dual User Architecture represents a fundamental advancement in cloud research computing. By separating template flexibility from research continuity, Prism enables:

**For Individual Researchers:**
- Seamless workflow continuation across different computational environments
- Persistent identity and file ownership
- Simplified SSH and access management

**For Research Teams:**
- True collaboration with consistent file permissions
- Multi-user shared resources
- Clear accountability and ownership

**For Institutions:**
- Simplified user management and monitoring
- Consistent backup and recovery procedures
- Cross-course and cross-project collaboration

This architecture positions Prism as the foundation for collaborative research computing, enabling the transition from individual research tools to institutional research platforms while maintaining the simplicity and flexibility that makes Prism powerful.

---

**Implementation Status**: Foundation Complete (Phase 5A)
**Next Steps**: CLI/TUI/GUI Integration
**Future Vision**: Full Collaborative Research Platform