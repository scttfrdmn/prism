# Directory Sync Planning

## Executive Summary

This document outlines the design for bidirectional directory synchronization between local systems and Prism instances, providing seamless file access similar to Google Drive, Dropbox, or OneDrive, but optimized for research workflows.

## Problem Statement

Researchers need seamless file access between their local development environments and cloud instances. Current solutions require manual file transfers or complex mounting procedures. The ideal solution should:

- **Bidirectional Sync**: Changes propagate both ways automatically
- **Real-Time Updates**: Near-instant synchronization of changes
- **Conflict Resolution**: Handle simultaneous edits gracefully
- **Selective Sync**: Control which files sync to optimize bandwidth
- **Research-Optimized**: Handle large datasets, code, and notebooks efficiently
- **Cross-Platform**: Work consistently across macOS, Linux, and Windows

## Architecture Overview

### 1. Sync Architecture Models

**Option A: Agent-Based Sync**
```
Local System                    Prism Instance
┌─────────────────┐           ┌─────────────────────────┐
│   cws-sync      │  ◄────►   │    cws-sync-agent      │
│   (daemon)      │   HTTPS   │    (daemon)            │
│                 │           │                        │
│ ~/research/     │           │ ~/research-sync/       │
│ ├── project1/   │           │ ├── project1/          │
│ ├── project2/   │           │ ├── project2/          │
│ └── datasets/   │           │ └── datasets/          │
└─────────────────┘           └─────────────────────────┘
```

**Option B: EFS-Backed Sync (Recommended)**
```
Local System                    EFS Volume                Prism Instance
┌─────────────────┐           ┌─────────────┐           ┌─────────────────────────┐
│   cws-sync      │  ◄────►   │    EFS      │  ◄────►   │    EFS Mount            │
│   (daemon)      │   API     │   Storage   │   NFS     │    /mnt/research-sync/  │
│                 │           │             │           │                        │
│ ~/research/     │           │ Versioned   │           │ ~/research-sync/       │
│ ├── project1/   │           │ Conflict    │           │ ├── project1/          │
│ ├── project2/   │           │ Resolution  │           │ ├── project2/          │
│ └── datasets/   │           │ Metadata    │           │ └── datasets/          │
└─────────────────┘           └─────────────┘           └─────────────────────────┘
```

### 2. EFS-Backed Sync Implementation (Recommended)

**Technical Benefits**:
- **Native AWS Integration**: Leverage EFS versioning and backup
- **Multi-Instance Access**: Multiple instances can access same sync folder
- **Cost Effective**: EFS storage costs lower than custom infrastructure
- **Reliability**: AWS-managed durability and availability
- **Conflict Resolution**: EFS versioning handles file conflicts
- **Security**: IAM-based access controls

**Architecture Components**:
```go
// pkg/sync/manager.go
type DirectorySyncManager struct {
    localWatcher    *fsnotify.Watcher
    efsClient      EFSClientInterface
    s3Client       S3ClientInterface  // For metadata and conflict resolution
    syncRules      *SyncRuleEngine
    conflictResolver *ConflictResolver
}

type SyncConfig struct {
    LocalPath      string              `yaml:"local_path"`
    EFSVolumeID    string              `yaml:"efs_volume_id"`
    SyncMode       SyncMode            `yaml:"sync_mode"`
    ExcludePatterns []string           `yaml:"exclude_patterns"`
    ConflictPolicy ConflictPolicy      `yaml:"conflict_policy"`
    Instances      []string            `yaml:"instances"`
}

type SyncMode string
const (
    SyncModeBidirectional SyncMode = "bidirectional"
    SyncModeUploadOnly    SyncMode = "upload_only"
    SyncModeDownloadOnly  SyncMode = "download_only"
)
```

### 3. Sync Rule Engine

**Intelligent File Filtering**:
```yaml
# ~/.prism/sync-rules.yml
default_rules:
  include_patterns:
    - "*.py"
    - "*.R"
    - "*.ipynb"
    - "*.md"
    - "*.txt"
    - "*.csv"
    - "*.json"
    - "*.yml"
    - "*.yaml"

  exclude_patterns:
    - ".git/"
    - "__pycache__/"
    - "*.pyc"
    - ".DS_Store"
    - "Thumbs.db"
    - "*.tmp"
    - "*.log"
    - "node_modules/"
    - ".venv/"
    - ".conda/"

  size_limits:
    max_file_size: "100MB"
    warn_file_size: "10MB"

research_rules:
  datasets:
    include_patterns:
      - "*.csv"
      - "*.parquet"
      - "*.h5"
      - "*.hdf5"
    max_file_size: "1GB"
    sync_mode: "upload_only"  # Datasets rarely change

  code:
    include_patterns:
      - "*.py"
      - "*.R"
      - "*.ipynb"
    sync_mode: "bidirectional"
    real_time: true

  results:
    include_patterns:
      - "*.png"
      - "*.pdf"
      - "*.html"
    sync_mode: "download_only"  # Results come from cloud
```

### 4. Command Interface

**Setup and Configuration**:
```bash
# Initialize sync for a directory
prism sync init ~/research/project1
✅ Created sync configuration
📂 Sync directory: ~/research/project1
🔗 EFS Volume: fs-1234567890abcdef0
⚙️  Sync mode: bidirectional

# Add Prism instances to sync
prism sync add-instance project1-sync my-ml-instance
prism sync add-instance project1-sync my-analysis-instance

# Start sync daemon
prism sync start project1-sync
🔄 Starting directory sync...
📡 Monitoring local changes: ~/research/project1
🔗 Connected to EFS: fs-1234567890abcdef0
✅ Sync active - 2 instances connected

# Monitor sync status
prism sync status project1-sync
📊 Sync Status: project1-sync
Local: ~/research/project1 (1,247 files, 2.3GB)
Remote: fs-1234567890abcdef0 (1,247 files, 2.3GB)
✅ In sync - Last update: 2 seconds ago

Instances:
  my-ml-instance: ✅ Connected (~/research-sync/project1)
  my-analysis-instance: ✅ Connected (~/research-sync/project1)

Recent Activity:
  📄 analysis.py - updated 2 seconds ago
  📄 results.csv - uploaded 1 minute ago
  📄 model.pkl - downloaded 3 minutes ago
```

**Advanced Sync Management**:
```bash
# Pause/resume sync
prism sync pause project1-sync
prism sync resume project1-sync

# Force sync (resolve conflicts)
prism sync force-sync project1-sync --direction up
prism sync force-sync project1-sync --direction down

# Show sync conflicts
prism sync conflicts project1-sync
⚠️  3 conflicts detected:
  📄 analysis.py (modified locally and remotely)
  📄 config.yml (modified locally and remotely)
  📄 data.csv (deleted locally, modified remotely)

# Resolve conflicts
prism sync resolve project1-sync analysis.py --keep local
prism sync resolve project1-sync config.yml --keep remote
prism sync resolve project1-sync data.csv --keep remote
```

### 5. Real-Time Sync Implementation

**File System Watching**:
```go
// pkg/sync/watcher.go
type LocalWatcher struct {
    watcher     *fsnotify.Watcher
    syncManager *DirectorySyncManager
    debouncer   *Debouncer
}

func (w *LocalWatcher) Start() error {
    go func() {
        for {
            select {
            case event := <-w.watcher.Events:
                // Debounce rapid changes
                w.debouncer.Add(event.Name, func() {
                    w.handleFileChange(event)
                })

            case err := <-w.watcher.Errors:
                w.handleError(err)
            }
        }
    }()
    return nil
}

func (w *LocalWatcher) handleFileChange(event fsnotify.Event) {
    if w.shouldSync(event.Name) {
        switch event.Op {
        case fsnotify.Write:
            w.syncManager.UploadFile(event.Name)
        case fsnotify.Remove:
            w.syncManager.DeleteFile(event.Name)
        case fsnotify.Rename:
            w.syncManager.RenameFile(event.Name)
        }
    }
}
```

**EFS Change Detection**:
```go
// pkg/sync/efs_monitor.go
type EFSChangeMonitor struct {
    efsClient    EFSClientInterface
    syncManager  *DirectorySyncManager
    pollInterval time.Duration
}

func (m *EFSChangeMonitor) Start() error {
    ticker := time.NewTicker(m.pollInterval)
    go func() {
        for range ticker.C {
            changes, err := m.detectChanges()
            if err == nil {
                m.processRemoteChanges(changes)
            }
        }
    }()
    return nil
}
```

### 6. Conflict Resolution System

**Conflict Detection**:
```go
// pkg/sync/conflicts.go
type ConflictResolver struct {
    policy ConflictPolicy
    s3Client S3ClientInterface  // For storing conflict metadata
}

type Conflict struct {
    FilePath     string
    LocalMTime   time.Time
    RemoteMTime  time.Time
    LocalSize    int64
    RemoteSize   int64
    LocalHash    string
    RemoteHash   string
    ConflictType ConflictType
}

type ConflictType string
const (
    ConflictModifiedBoth ConflictType = "modified_both"
    ConflictDeletedLocal ConflictType = "deleted_local"
    ConflictDeletedRemote ConflictType = "deleted_remote"
)
```

**Automatic Conflict Resolution**:
```yaml
conflict_resolution:
  policy: "user_prompt"  # Options: user_prompt, keep_local, keep_remote, keep_both

  automatic_rules:
    - pattern: "*.tmp"
      action: "keep_local"
    - pattern: "*.log"
      action: "keep_remote"
    - pattern: "*.ipynb"
      action: "keep_both"  # Create backup versions

  backup_strategy:
    enabled: true
    location: ".cws-sync-backups/"
    retention: "30d"
```

### 7. Performance Optimization

**Bandwidth Optimization**:
- **Delta Sync**: Only upload changed file portions
- **Compression**: Compress data before transfer
- **Batch Operations**: Group small file operations
- **Smart Scheduling**: Sync large files during off-peak hours

**Storage Optimization**:
- **Deduplication**: Avoid storing duplicate files
- **Intelligent Caching**: Cache frequently accessed files locally
- **Lazy Loading**: Download files on-demand when accessed
- **Cleanup**: Automatically remove old versions and temporary files

### 8. Integration with Prism

**Instance Launch Integration**:
```bash
# Launch instance with auto-sync
prism launch python-ml my-research --sync ~/research/current-project
🚀 Launching instance...
📂 Setting up directory sync...
   EFS Volume: fs-1234567890abcdef0 created
   Sync Config: bidirectional mode enabled
   Local: ~/research/current-project
   Remote: /mnt/research-sync/current-project
✅ Instance ready with synchronized directory
```

**Template Integration**:
```yaml
# templates/python-ml-sync.yml
name: "Python ML (with Sync)"
inherits: ["Python Machine Learning (Simplified)"]

sync_config:
  auto_setup: true
  mount_point: "/mnt/research-sync"
  default_rules: "research"
  conflict_policy: "user_prompt"

user_data_additions: |
  # Mount EFS sync volume
  mkdir -p /mnt/research-sync
  mount -t efs ${EFS_SYNC_ID}:/ /mnt/research-sync

  # Set up sync agent
  systemctl enable cws-sync-agent
  systemctl start cws-sync-agent
```

### 9. Implementation Phases

**Phase 1: Basic Directory Sync (v0.5.4)**
- Local file watching and basic upload
- EFS integration for storage
- Simple conflict detection
- CLI setup and configuration commands

**Phase 2: Bidirectional Sync (v0.5.5)**
- Remote change detection
- Automatic conflict resolution
- Real-time synchronization
- Multi-instance support

**Phase 3: Advanced Features (v0.5.6)**
- Intelligent sync rules
- Bandwidth optimization
- Backup and versioning
- Performance monitoring and tuning

**Phase 4: Enterprise Integration (v0.5.7)**
- Template-based sync configuration
- Institutional policy enforcement
- Audit logging and compliance
- Advanced conflict resolution strategies

### 10. Security Considerations

**Data Encryption**:
- EFS encryption at rest
- TLS encryption in transit
- Local cache encryption
- Secure credential management

**Access Control**:
- IAM-based EFS permissions
- Per-directory access controls
- Instance-level sync permissions
- Audit logging for compliance

**Privacy Protection**:
- Configurable data retention policies
- Secure deletion of synced data
- GDPR compliance features
- Institutional data governance integration

## User Experience Goals

**Seamless Integration**:
- Setup in under 2 minutes
- Zero configuration for common use cases
- Visual feedback for sync status
- Clear conflict resolution workflows

**Performance Targets**:
- File changes sync within 10 seconds
- Large file transfers optimize for available bandwidth
- Minimal impact on local system performance
- Efficient bandwidth usage for limited connections

**Reliability Standards**:
- 99.9% sync success rate
- Automatic recovery from network issues
- Data integrity verification
- Comprehensive backup and recovery

This directory sync system will provide researchers with the seamless file access they expect from modern cloud storage while being optimized for research workflows and integrated with Prism's existing infrastructure.