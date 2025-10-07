// Package types provides CloudWorkstation's core type definitions.
//
// This package is organized into logical modules:
//   - runtime.go: Instance and template runtime definitions
//   - storage.go: EFS and EBS volume types
//   - config.go: Configuration and state management
//   - requests.go: API request/response types
//   - api_version.go: API versioning types
//   - errors.go: Error handling types
//   - idle.go: Idle detection types
//   - instance.go: Instance-specific types
//   - repository.go: Repository management types
//
// For backward compatibility, the main types are also available
// through this file via type aliases.
package types

import "time"

// Backward compatibility aliases
// Template is aliased to RuntimeTemplate to distinguish from AMI build templates
type Template = RuntimeTemplate

// SimpleAPIError represents a simple API error response (legacy)
type SimpleAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e SimpleAPIError) Error() string {
	return e.Message
}

// Log-related types

// LogRequest represents a request for instance logs
type LogRequest struct {
	LogType string `json:"log_type,omitempty"`
	Tail    int    `json:"tail,omitempty"`
	Since   string `json:"since,omitempty"`
	Follow  bool   `json:"follow,omitempty"`
}

// LogResponse represents the response from a logs API call
type LogResponse struct {
	InstanceName string    `json:"instance_name"`
	InstanceID   string    `json:"instance_id"`
	LogType      string    `json:"log_type"`
	Lines        []string  `json:"lines"`
	Timestamp    time.Time `json:"timestamp"`
	Follow       bool      `json:"follow"`
	Tail         int       `json:"tail,omitempty"`
}

// LogTypesResponse represents available log types for an instance
type LogTypesResponse struct {
	InstanceName      string   `json:"instance_name"`
	InstanceID        string   `json:"instance_id"`
	AvailableLogTypes []string `json:"available_log_types"`
	SSMEnabled        bool     `json:"ssm_enabled"`
}

// InstanceLogSummary provides log availability summary for an instance
type InstanceLogSummary struct {
	Name          string `json:"name"`
	ID            string `json:"id"`
	State         string `json:"state"`
	LogsAvailable bool   `json:"logs_available"`
}

// LogSummaryResponse represents the response from listing all instance log availability
type LogSummaryResponse struct {
	Instances         []InstanceLogSummary `json:"instances"`
	AvailableLogTypes []string             `json:"available_log_types"`
}

// ==========================================
// Instance Snapshot Management Types
// ==========================================

// InstanceSnapshotResult represents the result of creating an instance snapshot
type InstanceSnapshotResult struct {
	SnapshotID                 string    `json:"snapshot_id"`
	SnapshotName               string    `json:"snapshot_name"`
	SourceInstance             string    `json:"source_instance"`
	SourceInstanceId           string    `json:"source_instance_id"`
	Description                string    `json:"description"`
	State                      string    `json:"state"`
	EstimatedCompletionMinutes int       `json:"estimated_completion_minutes"`
	StorageCostMonthly         float64   `json:"storage_cost_monthly"`
	CreatedAt                  time.Time `json:"created_at"`
	NoReboot                   bool      `json:"no_reboot"`
}

// InstanceSnapshotInfo represents information about an instance snapshot
type InstanceSnapshotInfo struct {
	SnapshotID          string    `json:"snapshot_id"`
	SnapshotName        string    `json:"snapshot_name"`
	SourceInstance      string    `json:"source_instance"`
	SourceInstanceId    string    `json:"source_instance_id"`
	SourceTemplate      string    `json:"source_template"`
	Description         string    `json:"description"`
	State               string    `json:"state"`
	Architecture        string    `json:"architecture"`
	StorageCostMonthly  float64   `json:"storage_cost_monthly"`
	CreatedAt           time.Time `json:"created_at"`
	AssociatedSnapshots []string  `json:"associated_snapshots,omitempty"`
}

// InstanceRestoreResult represents the result of restoring an instance from snapshot
type InstanceRestoreResult struct {
	NewInstanceName string    `json:"new_instance_name"`
	InstanceID      string    `json:"instance_id"`
	SnapshotName    string    `json:"snapshot_name"`
	SnapshotID      string    `json:"snapshot_id"`
	SourceTemplate  string    `json:"source_template"`
	State           string    `json:"state"`
	Message         string    `json:"message"`
	RestoredAt      time.Time `json:"restored_at"`
}

// InstanceSnapshotDeleteResult represents the result of deleting an instance snapshot
type InstanceSnapshotDeleteResult struct {
	SnapshotName          string    `json:"snapshot_name"`
	SnapshotID            string    `json:"snapshot_id"`
	DeletedSnapshots      []string  `json:"deleted_snapshots"`
	StorageSavingsMonthly float64   `json:"storage_savings_monthly"`
	DeletedAt             time.Time `json:"deleted_at"`
}

// InstanceSnapshotListResponse represents the response from listing snapshots
type InstanceSnapshotListResponse struct {
	Snapshots []InstanceSnapshotInfo `json:"snapshots"`
	Count     int                    `json:"count"`
}

// InstanceSnapshotRequest represents a request to create an instance snapshot
type InstanceSnapshotRequest struct {
	InstanceName string `json:"instance_name" binding:"required"`
	SnapshotName string `json:"snapshot_name" binding:"required"`
	Description  string `json:"description"`
	NoReboot     bool   `json:"no_reboot"`
	Wait         bool   `json:"wait"`
}

// InstanceRestoreRequest represents a request to restore an instance from snapshot
type InstanceRestoreRequest struct {
	SnapshotName    string `json:"snapshot_name" binding:"required"`
	NewInstanceName string `json:"new_instance_name" binding:"required"`
	Wait            bool   `json:"wait"`
}

// ==========================================
// Data Backup Management Types
// ==========================================

// BackupCreateRequest represents a request to create a data backup
type BackupCreateRequest struct {
	InstanceName string   `json:"instance_name" binding:"required"`
	BackupName   string   `json:"backup_name" binding:"required"`
	Description  string   `json:"description"`
	IncludePaths []string `json:"include_paths"`
	ExcludePaths []string `json:"exclude_paths"`
	Full         bool     `json:"full"`
	Incremental  bool     `json:"incremental"`
	StorageType  string   `json:"storage_type"` // "s3", "efs", "ebs"
	Encrypted    bool     `json:"encrypted"`
	Wait         bool     `json:"wait"`
}

// BackupInfo represents information about a data backup
type BackupInfo struct {
	BackupName         string            `json:"backup_name"`
	BackupID           string            `json:"backup_id"`
	SourceInstance     string            `json:"source_instance"`
	SourceInstanceId   string            `json:"source_instance_id"`
	Description        string            `json:"description"`
	BackupType         string            `json:"backup_type"` // "full", "incremental"
	StorageType        string            `json:"storage_type"`
	StorageLocation    string            `json:"storage_location"`
	State              string            `json:"state"` // "creating", "available", "error", "deleting"
	SizeBytes          int64             `json:"size_bytes"`
	CompressedBytes    int64             `json:"compressed_bytes"`
	FileCount          int               `json:"file_count"`
	IncludedPaths      []string          `json:"included_paths"`
	ExcludedPaths      []string          `json:"excluded_paths"`
	Encrypted          bool              `json:"encrypted"`
	StorageCostMonthly float64           `json:"storage_cost_monthly"`
	CreatedAt          time.Time         `json:"created_at"`
	CompletedAt        *time.Time        `json:"completed_at,omitempty"`
	ExpiresAt          *time.Time        `json:"expires_at,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	ParentBackup       string            `json:"parent_backup,omitempty"` // For incremental backups
	ChecksumMD5        string            `json:"checksum_md5,omitempty"`
}

// BackupCreateResult represents the result of creating a data backup
type BackupCreateResult struct {
	BackupName                 string    `json:"backup_name"`
	BackupID                   string    `json:"backup_id"`
	SourceInstance             string    `json:"source_instance"`
	BackupType                 string    `json:"backup_type"`
	StorageType                string    `json:"storage_type"`
	StorageLocation            string    `json:"storage_location"`
	EstimatedCompletionMinutes int       `json:"estimated_completion_minutes"`
	EstimatedSizeBytes         int64     `json:"estimated_size_bytes"`
	StorageCostMonthly         float64   `json:"storage_cost_monthly"`
	CreatedAt                  time.Time `json:"created_at"`
	Encrypted                  bool      `json:"encrypted"`
	Message                    string    `json:"message"`
}

// BackupListResponse represents the response from listing backups
type BackupListResponse struct {
	Backups      []BackupInfo   `json:"backups"`
	Count        int            `json:"count"`
	TotalSize    int64          `json:"total_size_bytes"`
	TotalCost    float64        `json:"total_cost_monthly"`
	StorageTypes map[string]int `json:"storage_types"`
}

// BackupDeleteResult represents the result of deleting a backup
type BackupDeleteResult struct {
	BackupName            string    `json:"backup_name"`
	BackupID              string    `json:"backup_id"`
	StorageType           string    `json:"storage_type"`
	StorageLocation       string    `json:"storage_location"`
	DeletedSizeBytes      int64     `json:"deleted_size_bytes"`
	StorageSavingsMonthly float64   `json:"storage_savings_monthly"`
	DeletedAt             time.Time `json:"deleted_at"`
}

// RestoreRequest represents a request to restore data from backup
type RestoreRequest struct {
	BackupName      string   `json:"backup_name" binding:"required"`
	TargetInstance  string   `json:"target_instance" binding:"required"`
	RestorePath     string   `json:"restore_path"`
	SelectivePaths  []string `json:"selective_paths"`
	Overwrite       bool     `json:"overwrite"`
	Merge           bool     `json:"merge"`
	DryRun          bool     `json:"dry_run"`
	PreservePerms   bool     `json:"preserve_permissions"`
	PreserveOwner   bool     `json:"preserve_owner"`
	VerifyIntegrity bool     `json:"verify_integrity"`
	Wait            bool     `json:"wait"`
}

// RestoreResult represents the result of a restore operation
type RestoreResult struct {
	RestoreID           string                 `json:"restore_id"`
	BackupName          string                 `json:"backup_name"`
	TargetInstance      string                 `json:"target_instance"`
	RestorePath         string                 `json:"restore_path"`
	SelectivePaths      []string               `json:"selective_paths"`
	State               string                 `json:"state"` // "running", "completed", "error"
	RestoredFileCount   int                    `json:"restored_file_count"`
	RestoredBytes       int64                  `json:"restored_bytes"`
	SkippedFileCount    int                    `json:"skipped_file_count"`
	ErrorCount          int                    `json:"error_count"`
	StartedAt           time.Time              `json:"started_at"`
	CompletedAt         *time.Time             `json:"completed_at,omitempty"`
	EstimatedCompletion int                    `json:"estimated_completion_minutes"`
	Message             string                 `json:"message"`
	Errors              []string               `json:"errors,omitempty"`
	Summary             map[string]interface{} `json:"summary,omitempty"`
	IntegrityVerified   bool                   `json:"integrity_verified"`
}

// BackupContentsRequest represents a request to list backup contents
type BackupContentsRequest struct {
	BackupName string `json:"backup_name" binding:"required"`
	Path       string `json:"path"`
	Recursive  bool   `json:"recursive"`
}

// BackupContentsResponse represents the response from listing backup contents
type BackupContentsResponse struct {
	BackupName string           `json:"backup_name"`
	Path       string           `json:"path"`
	Files      []BackupFileInfo `json:"files"`
	Count      int              `json:"count"`
	TotalSize  int64            `json:"total_size"`
}

// BackupFileInfo represents information about a file in a backup
type BackupFileInfo struct {
	Path     string            `json:"path"`
	Size     int64             `json:"size"`
	Mode     string            `json:"mode"`
	Owner    string            `json:"owner"`
	Group    string            `json:"group"`
	ModTime  time.Time         `json:"mod_time"`
	IsDir    bool              `json:"is_dir"`
	Checksum string            `json:"checksum,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// BackupVerifyRequest represents a request to verify backup integrity
type BackupVerifyRequest struct {
	BackupName     string   `json:"backup_name" binding:"required"`
	SelectivePaths []string `json:"selective_paths"`
	QuickCheck     bool     `json:"quick_check"`
}

// BackupVerifyResult represents the result of backup verification
type BackupVerifyResult struct {
	BackupName            string                 `json:"backup_name"`
	VerificationState     string                 `json:"verification_state"` // "valid", "corrupt", "partial"
	CheckedFileCount      int                    `json:"checked_file_count"`
	CorruptFileCount      int                    `json:"corrupt_file_count"`
	MissingFileCount      int                    `json:"missing_file_count"`
	VerifiedBytes         int64                  `json:"verified_bytes"`
	VerificationStarted   time.Time              `json:"verification_started"`
	VerificationCompleted *time.Time             `json:"verification_completed,omitempty"`
	CorruptFiles          []string               `json:"corrupt_files,omitempty"`
	MissingFiles          []string               `json:"missing_files,omitempty"`
	Summary               map[string]interface{} `json:"summary,omitempty"`
}
