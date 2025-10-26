// Package storage provides advanced storage integration for Prism.
//
// This package implements high-performance storage solutions including FSx filesystems,
// S3 mount points, and comprehensive storage analytics for research workloads.
package storage

import (
	"time"
)

// StorageType defines the type of storage system
type StorageType string

const (
	StorageTypeEFS StorageType = "efs" // Amazon EFS (existing)
	StorageTypeFSx StorageType = "fsx" // Amazon FSx
	StorageTypeS3  StorageType = "s3"  // S3 mount points
	StorageTypeEBS StorageType = "ebs" // EBS volumes (existing)
)

// FSxFilesystemType defines FSx filesystem types
type FSxFilesystemType string
type FSxType = FSxFilesystemType // Alias for compatibility

const (
	FSxTypeLustre  FSxFilesystemType = "lustre"  // High-performance computing
	FSxTypeWindows FSxFilesystemType = "windows" // Windows File Server
	FSxTypeZFS     FSxFilesystemType = "zfs"     // OpenZFS
	FSxTypeOpenZFS FSxFilesystemType = "openzfs" // OpenZFS (alternate name)
	FSxTypeNetApp  FSxFilesystemType = "netapp"  // NetApp ONTAP

	// Alias for manager.go compatibility
	FSxFilesystemTypeLustre = FSxTypeLustre
)

// S3MountMethod defines S3 mounting mechanisms
type S3MountMethod string

const (
	S3MountMethodS3FS       S3MountMethod = "s3fs"       // s3fs-fuse
	S3MountMethodGoofys     S3MountMethod = "goofys"     // Goofys
	S3MountMethodMountpoint S3MountMethod = "mountpoint" // AWS Mountpoint for Amazon S3
	S3MountMethodRclone     S3MountMethod = "rclone"     // Rclone
)

// WorkloadType defines general workload optimization types
type WorkloadType string

const (
	WorkloadTypeGeneral  WorkloadType = "general"
	WorkloadTypeML       WorkloadType = "ml"
	WorkloadTypeBigData  WorkloadType = "bigdata"
	WorkloadTypeHPC      WorkloadType = "hpc"
	WorkloadTypeArchival WorkloadType = "archival"
)

// S3WorkloadType defines S3-specific workload optimization types
type S3WorkloadType string

const (
	S3WorkloadFrequentAccess S3WorkloadType = "frequent_access"
	S3WorkloadArchival       S3WorkloadType = "archival"
	S3WorkloadBigData        S3WorkloadType = "bigdata"
)

// FSxWorkloadType defines FSx-specific workload optimization types
type FSxWorkloadType string

const (
	FSxWorkloadGeneral FSxWorkloadType = "general"
	FSxWorkloadHPC     FSxWorkloadType = "hpc"
	FSxWorkloadBigData FSxWorkloadType = "bigdata"
)

// StorageRequest represents a generic storage creation request
type StorageRequest struct {
	// Basic configuration
	Name string      `json:"name" yaml:"name"`
	Type StorageType `json:"type" yaml:"type"`
	Size int64       `json:"size,omitempty" yaml:"size,omitempty"` // Size in GB for applicable types

	// Performance configuration
	PerformanceMode string `json:"performance_mode,omitempty" yaml:"performance_mode,omitempty"`
	ThroughputMode  string `json:"throughput_mode,omitempty" yaml:"throughput_mode,omitempty"`

	// Encryption
	Encrypted bool   `json:"encrypted,omitempty" yaml:"encrypted,omitempty"`
	KMSKeyId  string `json:"kms_key_id,omitempty" yaml:"kms_key_id,omitempty"`

	// Tags and metadata
	Tags        map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`

	// Type-specific configuration
	FSxConfig *FSxConfiguration `json:"fsx_config,omitempty" yaml:"fsx_config,omitempty"`
	S3Config  *S3Configuration  `json:"s3_config,omitempty" yaml:"s3_config,omitempty"`
	EFSConfig *EFSConfiguration `json:"efs_config,omitempty" yaml:"efs_config,omitempty"`
	EBSConfig *EBSConfiguration `json:"ebs_config,omitempty" yaml:"ebs_config,omitempty"`
}

// FSxConfiguration contains FSx-specific settings
type FSxConfiguration struct {
	// Filesystem type
	FilesystemType FSxFilesystemType `json:"filesystem_type" yaml:"filesystem_type"`

	// Capacity and performance
	StorageCapacity     int32 `json:"storage_capacity" yaml:"storage_capacity"`                           // GB
	ThroughputCapacity  int32 `json:"throughput_capacity,omitempty" yaml:"throughput_capacity,omitempty"` // MB/s
	PerSecondThroughput int32 `json:"per_unit_throughput,omitempty" yaml:"per_unit_throughput,omitempty"`

	// Networking
	SubnetIds         []string `json:"subnet_ids" yaml:"subnet_ids"`
	SecurityGroupIds  []string `json:"security_group_ids,omitempty" yaml:"security_group_ids,omitempty"`
	PreferredSubnetId string   `json:"preferred_subnet_id,omitempty" yaml:"preferred_subnet_id,omitempty"`

	// Backup configuration
	AutomaticBackupRetention int32  `json:"automatic_backup_retention,omitempty" yaml:"automatic_backup_retention,omitempty"` // Days
	CopyTagsToBackups        bool   `json:"copy_tags_to_backups,omitempty" yaml:"copy_tags_to_backups,omitempty"`
	DailyBackupTime          string `json:"daily_backup_time,omitempty" yaml:"daily_backup_time,omitempty"` // HH:MM format

	// Lustre-specific configuration
	LustreConfig *LustreConfiguration `json:"lustre_config,omitempty" yaml:"lustre_config,omitempty"`

	// OpenZFS-specific configuration
	ZFSConfig *ZFSConfiguration `json:"zfs_config,omitempty" yaml:"zfs_config,omitempty"`

	// Windows-specific configuration
	WindowsConfig *WindowsConfiguration `json:"windows_config,omitempty" yaml:"windows_config,omitempty"`

	// NetApp-specific configuration
	NetAppConfig *NetAppConfiguration `json:"netapp_config,omitempty" yaml:"netapp_config,omitempty"`
}

// LustreConfiguration contains Lustre-specific settings
type LustreConfiguration struct {
	// Data repository configuration
	ImportPath            string `json:"import_path,omitempty" yaml:"import_path,omitempty"`                           // S3 bucket path
	ExportPath            string `json:"export_path,omitempty" yaml:"export_path,omitempty"`                           // S3 export path
	ImportedFileChunkSize int32  `json:"imported_file_chunk_size,omitempty" yaml:"imported_file_chunk_size,omitempty"` // MiB

	// Deployment type
	DeploymentType string `json:"deployment_type,omitempty" yaml:"deployment_type,omitempty"` // SCRATCH_1, SCRATCH_2, PERSISTENT_1, PERSISTENT_2

	// Drive cache type for persistent filesystems
	DriveCacheType string `json:"drive_cache_type,omitempty" yaml:"drive_cache_type,omitempty"` // NONE, READ

	// Data compression
	DataCompression string `json:"data_compression,omitempty" yaml:"data_compression,omitempty"` // LZ4

	// Log configuration
	LogConfiguration *LustreLogConfiguration `json:"log_configuration,omitempty" yaml:"log_configuration,omitempty"`
}

// LustreLogConfiguration contains Lustre logging settings
type LustreLogConfiguration struct {
	Level       string `json:"level" yaml:"level"`                                 // DISABLED, WARN_ONLY, ERROR_ONLY
	Destination string `json:"destination,omitempty" yaml:"destination,omitempty"` // CloudWatch log group
}

// ZFSConfiguration contains OpenZFS-specific settings
type ZFSConfiguration struct {
	// Root volume configuration
	RootVolumeConfiguration *ZFSRootVolumeConfiguration `json:"root_volume_configuration" yaml:"root_volume_configuration"`

	// Deployment type
	DeploymentType string `json:"deployment_type,omitempty" yaml:"deployment_type,omitempty"` // SINGLE_AZ_1, SINGLE_AZ_2

	// Disk IOPS configuration
	DiskIopsConfiguration *ZFSDiskIopsConfiguration `json:"disk_iops_configuration,omitempty" yaml:"disk_iops_configuration,omitempty"`

	// Weekly maintenance start time
	WeeklyMaintenanceStartTime string `json:"weekly_maintenance_start_time,omitempty" yaml:"weekly_maintenance_start_time,omitempty"`
}

// ZFSRootVolumeConfiguration contains ZFS root volume settings
type ZFSRootVolumeConfiguration struct {
	// Data compression
	DataCompression string `json:"data_compression,omitempty" yaml:"data_compression,omitempty"` // NONE, ZSTD, LZ4

	// NFS exports
	NfsExports []ZFSNfsExport `json:"nfs_exports,omitempty" yaml:"nfs_exports,omitempty"`

	// User and group quotas
	UserAndGroupQuotas []ZFSUserGroupQuota `json:"user_and_group_quotas,omitempty" yaml:"user_and_group_quotas,omitempty"`

	// Copy tags to snapshots
	CopyTagsToSnapshots bool `json:"copy_tags_to_snapshots,omitempty" yaml:"copy_tags_to_snapshots,omitempty"`

	// Read only
	ReadOnly bool `json:"read_only,omitempty" yaml:"read_only,omitempty"`

	// Record size
	RecordSizeKiB int32 `json:"record_size_kib,omitempty" yaml:"record_size_kib,omitempty"`
}

// ZFSNfsExport defines NFS export configuration
type ZFSNfsExport struct {
	ClientConfigurations []ZFSClientConfiguration `json:"client_configurations" yaml:"client_configurations"`
	Path                 string                   `json:"path" yaml:"path"`
}

// ZFSClientConfiguration defines NFS client settings
type ZFSClientConfiguration struct {
	Clients string   `json:"clients" yaml:"clients"` // IP range or hostname
	Options []string `json:"options" yaml:"options"` // NFS options
}

// ZFSUserGroupQuota defines user/group quotas
type ZFSUserGroupQuota struct {
	Id                      int32  `json:"id" yaml:"id"` // User/Group ID
	StorageCapacityQuotaGiB int32  `json:"storage_capacity_quota_gib" yaml:"storage_capacity_quota_gib"`
	Type                    string `json:"type" yaml:"type"` // USER, GROUP
}

// ZFSDiskIopsConfiguration defines disk IOPS settings
type ZFSDiskIopsConfiguration struct {
	Iops int32  `json:"iops,omitempty" yaml:"iops,omitempty"`
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty"` // AUTOMATIC, USER_PROVISIONED
}

// WindowsConfiguration contains Windows File Server settings
type WindowsConfiguration struct {
	// Active Directory configuration
	ActiveDirectoryId          string                      `json:"active_directory_id,omitempty" yaml:"active_directory_id,omitempty"`
	SelfManagedActiveDirectory *SelfManagedActiveDirectory `json:"self_managed_active_directory,omitempty" yaml:"self_managed_active_directory,omitempty"`

	// Aliases
	Aliases []string `json:"aliases,omitempty" yaml:"aliases,omitempty"`

	// Audit log configuration
	AuditLogConfiguration *WindowsAuditLogConfiguration `json:"audit_log_configuration,omitempty" yaml:"audit_log_configuration,omitempty"`

	// Deployment type
	DeploymentType string `json:"deployment_type,omitempty" yaml:"deployment_type,omitempty"` // MULTI_AZ_1, SINGLE_AZ_1, SINGLE_AZ_2

	// Preferred subnet (for Multi-AZ)
	PreferredSubnetId string `json:"preferred_subnet_id,omitempty" yaml:"preferred_subnet_id,omitempty"`

	// Weekly maintenance
	WeeklyMaintenanceStartTime string `json:"weekly_maintenance_start_time,omitempty" yaml:"weekly_maintenance_start_time,omitempty"`
}

// SelfManagedActiveDirectory contains self-managed AD settings
type SelfManagedActiveDirectory struct {
	DnsIps                   []string `json:"dns_ips" yaml:"dns_ips"`
	DomainName               string   `json:"domain_name" yaml:"domain_name"`
	FileSystemAdministrators []string `json:"file_system_administrators,omitempty" yaml:"file_system_administrators,omitempty"`
	OrganizationalUnit       string   `json:"organizational_unit,omitempty" yaml:"organizational_unit,omitempty"`
	Password                 string   `json:"password" yaml:"password"`
	Username                 string   `json:"username" yaml:"username"`
}

// WindowsAuditLogConfiguration contains Windows audit logging settings
type WindowsAuditLogConfiguration struct {
	FileAccessAuditLogLevel      string `json:"file_access_audit_log_level" yaml:"file_access_audit_log_level"` // DISABLED, SUCCESS_ONLY, FAILURE_ONLY, SUCCESS_AND_FAILURE
	FileShareAccessAuditLogLevel string `json:"file_share_access_audit_log_level" yaml:"file_share_access_audit_log_level"`
	AuditLogDestination          string `json:"audit_log_destination,omitempty" yaml:"audit_log_destination,omitempty"` // CloudWatch log group
}

// NetAppConfiguration contains NetApp ONTAP settings
type NetAppConfiguration struct {
	// Deployment type
	DeploymentType string `json:"deployment_type,omitempty" yaml:"deployment_type,omitempty"` // MULTI_AZ_1, SINGLE_AZ_1

	// Endpoint IP address range
	EndpointIpAddressRange string `json:"endpoint_ip_address_range,omitempty" yaml:"endpoint_ip_address_range,omitempty"`

	// FsxAdmin password
	FsxAdminPassword string `json:"fsx_admin_password,omitempty" yaml:"fsx_admin_password,omitempty"`

	// Preferred subnet
	PreferredSubnetId string `json:"preferred_subnet_id,omitempty" yaml:"preferred_subnet_id,omitempty"`

	// Route table IDs
	RouteTableIds []string `json:"route_table_ids,omitempty" yaml:"route_table_ids,omitempty"`

	// Weekly maintenance
	WeeklyMaintenanceStartTime string `json:"weekly_maintenance_start_time,omitempty" yaml:"weekly_maintenance_start_time,omitempty"`

	// Disk IOPS configuration
	DiskIopsConfiguration *NetAppDiskIopsConfiguration `json:"disk_iops_configuration,omitempty" yaml:"disk_iops_configuration,omitempty"`
}

// NetAppDiskIopsConfiguration defines NetApp disk IOPS settings
type NetAppDiskIopsConfiguration struct {
	Iops int32  `json:"iops,omitempty" yaml:"iops,omitempty"`
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty"` // AUTOMATIC, USER_PROVISIONED
}

// S3Configuration contains S3 mount point settings
type S3Configuration struct {
	// S3 bucket configuration
	BucketName   string `json:"bucket_name" yaml:"bucket_name"`
	BucketRegion string `json:"bucket_region,omitempty" yaml:"bucket_region,omitempty"`
	Prefix       string `json:"prefix,omitempty" yaml:"prefix,omitempty"` // S3 key prefix

	// Mount configuration
	MountMethod  S3MountMethod `json:"mount_method" yaml:"mount_method"`
	MountOptions []string      `json:"mount_options,omitempty" yaml:"mount_options,omitempty"`

	// Access configuration
	AccessMode        string `json:"access_mode,omitempty" yaml:"access_mode,omitempty"` // ro, rw
	RequesterPays     bool   `json:"requester_pays,omitempty" yaml:"requester_pays,omitempty"`
	UseVirtualHosting bool   `json:"use_virtual_hosting,omitempty" yaml:"use_virtual_hosting,omitempty"`

	// Performance tuning
	CacheSize        int64  `json:"cache_size,omitempty" yaml:"cache_size,omitempty"` // Local cache size in MB
	CacheDirectory   string `json:"cache_directory,omitempty" yaml:"cache_directory,omitempty"`
	ParallelRequests int32  `json:"parallel_requests,omitempty" yaml:"parallel_requests,omitempty"`
	MultipartSize    int64  `json:"multipart_size,omitempty" yaml:"multipart_size,omitempty"` // MB

	// Security
	UseIAMRole      bool   `json:"use_iam_role,omitempty" yaml:"use_iam_role,omitempty"`
	AccessKeyId     string `json:"access_key_id,omitempty" yaml:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty" yaml:"secret_access_key,omitempty"`
	SessionToken    string `json:"session_token,omitempty" yaml:"session_token,omitempty"`
}

// EFSConfiguration contains EFS-specific settings
type EFSConfiguration struct {
	// Performance configuration
	PerformanceMode       string  `json:"performance_mode,omitempty" yaml:"performance_mode,omitempty"`             // generalPurpose, maxIO
	ThroughputMode        string  `json:"throughput_mode,omitempty" yaml:"throughput_mode,omitempty"`               // bursting, provisioned
	ProvisionedThroughput float64 `json:"provisioned_throughput,omitempty" yaml:"provisioned_throughput,omitempty"` // MiB/s

	// Availability and durability
	AvailabilityZone string `json:"availability_zone,omitempty" yaml:"availability_zone,omitempty"`

	// Access points
	AccessPoints []AccessPointConfig `json:"access_points,omitempty" yaml:"access_points,omitempty"`

	// Backup configuration
	BackupPolicy bool `json:"backup_policy,omitempty" yaml:"backup_policy,omitempty"`
}

// AccessPointConfig defines EFS access point configuration
type AccessPointConfig struct {
	Path      string     `json:"path" yaml:"path"`
	PosixUser *PosixUser `json:"posix_user,omitempty" yaml:"posix_user,omitempty"`
}

// EBSConfiguration contains EBS-specific settings
type EBSConfiguration struct {
	// Volume configuration
	VolumeType       string `json:"volume_type,omitempty" yaml:"volume_type,omitempty"` // gp2, gp3, io1, io2, st1, sc1
	IOPS             int32  `json:"iops,omitempty" yaml:"iops,omitempty"`               // Provisioned IOPS
	Throughput       int32  `json:"throughput,omitempty" yaml:"throughput,omitempty"`   // MB/s (for gp3, st1, sc1)
	AvailabilityZone string `json:"availability_zone,omitempty" yaml:"availability_zone,omitempty"`

	// Snapshot configuration
	SnapshotID string `json:"snapshot_id,omitempty" yaml:"snapshot_id,omitempty"`

	// Encryption
	Encrypted bool   `json:"encrypted,omitempty" yaml:"encrypted,omitempty"`
	KmsKeyId  string `json:"kms_key_id,omitempty" yaml:"kms_key_id,omitempty"`

	// Filesystem
	Filesystem string `json:"filesystem,omitempty" yaml:"filesystem,omitempty"` // ext4, xfs, etc.

	// Multi-attach (io1/io2 only)
	MultiAttachEnabled bool `json:"multi_attach_enabled,omitempty" yaml:"multi_attach_enabled,omitempty"`
}

// StorageInfo represents information about a created storage resource
type StorageInfo struct {
	// Basic information
	Name  string      `json:"name" yaml:"name"`
	Type  StorageType `json:"type" yaml:"type"`
	Id    string      `json:"id" yaml:"id"`       // AWS resource ID
	State string      `json:"state" yaml:"state"` // Current state
	Size  int64       `json:"size,omitempty" yaml:"size,omitempty"`

	// Network information
	DNSName      string            `json:"dns_name,omitempty" yaml:"dns_name,omitempty"`
	IPAddress    string            `json:"ip_address,omitempty" yaml:"ip_address,omitempty"`
	MountTargets []MountTargetInfo `json:"mount_targets,omitempty" yaml:"mount_targets,omitempty"`

	// Performance metrics
	ThroughputCapacity int32  `json:"throughput_capacity,omitempty" yaml:"throughput_capacity,omitempty"`
	IOPS               int32  `json:"iops,omitempty" yaml:"iops,omitempty"`
	PerformanceMode    string `json:"performance_mode,omitempty" yaml:"performance_mode,omitempty"`

	// Cost information
	EstimatedMonthlyCost float64 `json:"estimated_monthly_cost" yaml:"estimated_monthly_cost"`
	CostPerGB            float64 `json:"cost_per_gb,omitempty" yaml:"cost_per_gb,omitempty"`
	CostPerThroughput    float64 `json:"cost_per_throughput,omitempty" yaml:"cost_per_throughput,omitempty"`

	// Timestamps
	CreatedAt    time.Time `json:"created_at" yaml:"created_at"`
	LastModified time.Time `json:"last_modified,omitempty" yaml:"last_modified,omitempty"`

	// Tags and metadata
	Tags        map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`

	// Type-specific information
	FSxInfo   *FSxInfo          `json:"fsx_info,omitempty" yaml:"fsx_info,omitempty"`
	S3Info    *S3Info           `json:"s3_info,omitempty" yaml:"s3_info,omitempty"`
	EFSConfig *EFSConfiguration `json:"efs_config,omitempty" yaml:"efs_config,omitempty"`
	EBSConfig *EBSConfiguration `json:"ebs_config,omitempty" yaml:"ebs_config,omitempty"`
	S3Config  *S3Configuration  `json:"s3_config,omitempty" yaml:"s3_config,omitempty"`
	FSxConfig *FSxConfiguration `json:"fsx_config,omitempty" yaml:"fsx_config,omitempty"`

	// Legacy fields for backward compatibility
	FilesystemID string    `json:"filesystem_id,omitempty" yaml:"filesystem_id,omitempty"`
	VolumeID     string    `json:"volume_id,omitempty" yaml:"volume_id,omitempty"`
	BucketName   string    `json:"bucket_name,omitempty" yaml:"bucket_name,omitempty"`
	Region       string    `json:"region,omitempty" yaml:"region,omitempty"`
	CreationTime time.Time `json:"creation_time,omitempty" yaml:"creation_time,omitempty"`
}

// MountTargetInfo contains mount target details
type MountTargetInfo struct {
	Id               string `json:"id" yaml:"id"`
	SubnetId         string `json:"subnet_id" yaml:"subnet_id"`
	IPAddress        string `json:"ip_address" yaml:"ip_address"`
	AvailabilityZone string `json:"availability_zone" yaml:"availability_zone"`
	State            string `json:"state" yaml:"state"`
}

// FSxInfo contains FSx-specific information
type FSxInfo struct {
	FilesystemType     FSxFilesystemType `json:"filesystem_type" yaml:"filesystem_type"`
	StorageCapacity    int32             `json:"storage_capacity" yaml:"storage_capacity"`
	ThroughputCapacity int32             `json:"throughput_capacity,omitempty" yaml:"throughput_capacity,omitempty"`

	// Lustre-specific info
	LustreInfo *LustreInfo `json:"lustre_info,omitempty" yaml:"lustre_info,omitempty"`

	// ZFS-specific info
	ZFSInfo *ZFSInfo `json:"zfs_info,omitempty" yaml:"zfs_info,omitempty"`
}

// LustreInfo contains Lustre filesystem information
type LustreInfo struct {
	DeploymentType     string              `json:"deployment_type" yaml:"deployment_type"`
	MountName          string              `json:"mount_name,omitempty" yaml:"mount_name,omitempty"`
	DataRepositoryInfo *DataRepositoryInfo `json:"data_repository_info,omitempty" yaml:"data_repository_info,omitempty"`
}

// DataRepositoryInfo contains data repository information
type DataRepositoryInfo struct {
	ImportPath            string `json:"import_path,omitempty" yaml:"import_path,omitempty"`
	ExportPath            string `json:"export_path,omitempty" yaml:"export_path,omitempty"`
	ImportedFileChunkSize int32  `json:"imported_file_chunk_size,omitempty" yaml:"imported_file_chunk_size,omitempty"`
}

// ZFSInfo contains OpenZFS filesystem information
type ZFSInfo struct {
	DeploymentType string          `json:"deployment_type" yaml:"deployment_type"`
	RootVolumeId   string          `json:"root_volume_id,omitempty" yaml:"root_volume_id,omitempty"`
	Volumes        []ZFSVolumeInfo `json:"volumes,omitempty" yaml:"volumes,omitempty"`
}

// ZFSVolumeInfo contains ZFS volume information
type ZFSVolumeInfo struct {
	VolumeId     string `json:"volume_id" yaml:"volume_id"`
	VolumePath   string `json:"volume_path" yaml:"volume_path"`
	StorageQuota int64  `json:"storage_quota,omitempty" yaml:"storage_quota,omitempty"`
}

// S3Info contains S3 mount point information
type S3Info struct {
	BucketName   string        `json:"bucket_name" yaml:"bucket_name"`
	BucketRegion string        `json:"bucket_region" yaml:"bucket_region"`
	MountMethod  S3MountMethod `json:"mount_method" yaml:"mount_method"`
	AccessMode   string        `json:"access_mode" yaml:"access_mode"`
	CacheSize    int64         `json:"cache_size,omitempty" yaml:"cache_size,omitempty"`
}

// MountRequest represents a storage mount request
type MountRequest struct {
	// Target configuration
	StorageName  string `json:"storage_name" yaml:"storage_name"`
	InstanceName string `json:"instance_name" yaml:"instance_name"`
	MountPoint   string `json:"mount_point" yaml:"mount_point"`

	// Mount options
	MountOptions []string `json:"mount_options,omitempty" yaml:"mount_options,omitempty"`
	ReadOnly     bool     `json:"read_only,omitempty" yaml:"read_only,omitempty"`

	// User and group configuration
	UserId      int32  `json:"user_id,omitempty" yaml:"user_id,omitempty"`
	GroupId     int32  `json:"group_id,omitempty" yaml:"group_id,omitempty"`
	Permissions string `json:"permissions,omitempty" yaml:"permissions,omitempty"` // e.g., "755"

	// Type-specific mount configuration
	S3MountConfig *S3MountConfig `json:"s3_mount_config,omitempty" yaml:"s3_mount_config,omitempty"`
}

// S3MountConfig contains S3-specific mount configuration
type S3MountConfig struct {
	// Cache configuration
	UseCache       bool   `json:"use_cache,omitempty" yaml:"use_cache,omitempty"`
	CacheDirectory string `json:"cache_directory,omitempty" yaml:"cache_directory,omitempty"`
	CacheTTL       int32  `json:"cache_ttl,omitempty" yaml:"cache_ttl,omitempty"` // seconds

	// Performance configuration
	ParallelRequests int32 `json:"parallel_requests,omitempty" yaml:"parallel_requests,omitempty"`
	MultipartSize    int64 `json:"multipart_size,omitempty" yaml:"multipart_size,omitempty"`
	StatCacheSize    int32 `json:"stat_cache_size,omitempty" yaml:"stat_cache_size,omitempty"`

	// Debug and logging
	LogLevel  string `json:"log_level,omitempty" yaml:"log_level,omitempty"` // crit, err, warn, info, debug
	DebugS3   bool   `json:"debug_s3,omitempty" yaml:"debug_s3,omitempty"`
	DebugFuse bool   `json:"debug_fuse,omitempty" yaml:"debug_fuse,omitempty"`
}

// StorageAnalytics contains comprehensive storage usage and performance analytics
type StorageAnalytics struct {
	// Basic information
	StorageName string      `json:"storage_name" yaml:"storage_name"`
	Type        StorageType `json:"type" yaml:"type"`

	// Time range for analytics
	StartTime time.Time `json:"start_time" yaml:"start_time"`
	EndTime   time.Time `json:"end_time" yaml:"end_time"`

	// Usage statistics
	Usage StorageUsageStats `json:"usage" yaml:"usage"`

	// Performance metrics
	Performance StoragePerformanceStats `json:"performance" yaml:"performance"`

	// Cost analytics
	Cost StorageCostStats `json:"cost" yaml:"cost"`

	// Access patterns
	AccessPatterns StorageAccessPatterns `json:"access_patterns" yaml:"access_patterns"`
}

// StorageUsageStats contains storage usage statistics
type StorageUsageStats struct {
	// Capacity usage
	TotalCapacity int64   `json:"total_capacity" yaml:"total_capacity"` // bytes
	UsedCapacity  int64   `json:"used_capacity" yaml:"used_capacity"`   // bytes
	UsagePercent  float64 `json:"usage_percent" yaml:"usage_percent"`

	// File statistics
	TotalFiles       int64 `json:"total_files" yaml:"total_files"`
	TotalDirectories int64 `json:"total_directories" yaml:"total_directories"`

	// Growth statistics
	DailyGrowthRate   float64 `json:"daily_growth_rate" yaml:"daily_growth_rate"`     // bytes per day
	WeeklyGrowthRate  float64 `json:"weekly_growth_rate" yaml:"weekly_growth_rate"`   // bytes per week
	MonthlyGrowthRate float64 `json:"monthly_growth_rate" yaml:"monthly_growth_rate"` // bytes per month

	// Top consumers
	TopDirectories []DirectoryUsage `json:"top_directories,omitempty" yaml:"top_directories,omitempty"`
	TopFileTypes   []FileTypeUsage  `json:"top_file_types,omitempty" yaml:"top_file_types,omitempty"`
}

// StoragePerformanceStats contains storage performance metrics
type StoragePerformanceStats struct {
	// Throughput metrics
	ReadThroughputMBps  float64 `json:"read_throughput_mbps" yaml:"read_throughput_mbps"`
	WriteThroughputMBps float64 `json:"write_throughput_mbps" yaml:"write_throughput_mbps"`

	// IOPS metrics
	ReadIOPS  float64 `json:"read_iops" yaml:"read_iops"`
	WriteIOPS float64 `json:"write_iops" yaml:"write_iops"`

	// Latency metrics (milliseconds)
	AverageReadLatency  float64 `json:"average_read_latency" yaml:"average_read_latency"`
	AverageWriteLatency float64 `json:"average_write_latency" yaml:"average_write_latency"`
	P95ReadLatency      float64 `json:"p95_read_latency" yaml:"p95_read_latency"`
	P95WriteLatency     float64 `json:"p95_write_latency" yaml:"p95_write_latency"`

	// Queue depth and concurrency
	AverageQueueDepth float64 `json:"average_queue_depth" yaml:"average_queue_depth"`
	MaxConcurrentOps  int32   `json:"max_concurrent_ops" yaml:"max_concurrent_ops"`

	// Cache performance (for S3 mounts)
	CacheHitRate  float64 `json:"cache_hit_rate,omitempty" yaml:"cache_hit_rate,omitempty"`
	CacheMissRate float64 `json:"cache_miss_rate,omitempty" yaml:"cache_miss_rate,omitempty"`
}

// StorageCostStats contains storage cost analytics
type StorageCostStats struct {
	// Base storage costs
	StorageCost      float64 `json:"storage_cost" yaml:"storage_cost"`                           // Total storage cost
	ThroughputCost   float64 `json:"throughput_cost,omitempty" yaml:"throughput_cost,omitempty"` // Throughput cost (FSx)
	RequestCost      float64 `json:"request_cost,omitempty" yaml:"request_cost,omitempty"`       // Request cost (S3)
	DataTransferCost float64 `json:"data_transfer_cost,omitempty" yaml:"data_transfer_cost,omitempty"`

	// Total costs
	TotalCost            float64 `json:"total_cost" yaml:"total_cost"`
	DailyCost            float64 `json:"daily_cost" yaml:"daily_cost"`
	ProjectedMonthlyCost float64 `json:"projected_monthly_cost" yaml:"projected_monthly_cost"`

	// Cost optimization metrics
	CostPerGB         float64 `json:"cost_per_gb" yaml:"cost_per_gb"`
	CostPerIOPS       float64 `json:"cost_per_iops,omitempty" yaml:"cost_per_iops,omitempty"`
	CostPerThroughput float64 `json:"cost_per_throughput,omitempty" yaml:"cost_per_throughput,omitempty"`

	// Savings opportunities
	PotentialSavings []CostOptimizationSuggestion `json:"potential_savings,omitempty" yaml:"potential_savings,omitempty"`
}

// StorageAccessPatterns contains storage access pattern analytics
type StorageAccessPatterns struct {
	// Access frequency
	HotDataPercent  float64 `json:"hot_data_percent" yaml:"hot_data_percent"`   // Frequently accessed
	WarmDataPercent float64 `json:"warm_data_percent" yaml:"warm_data_percent"` // Occasionally accessed
	ColdDataPercent float64 `json:"cold_data_percent" yaml:"cold_data_percent"` // Rarely accessed

	// Access timing
	PeakAccessHours     []int   `json:"peak_access_hours" yaml:"peak_access_hours"`         // Hours of day (0-23)
	BusinessHoursAccess float64 `json:"business_hours_access" yaml:"business_hours_access"` // Percentage

	// User access patterns
	TopUsers []UserAccessPattern `json:"top_users,omitempty" yaml:"top_users,omitempty"`

	// Application access patterns
	ReadWriteRatio          float64 `json:"read_write_ratio" yaml:"read_write_ratio"` // Read operations / Write operations
	SequentialAccessPercent float64 `json:"sequential_access_percent" yaml:"sequential_access_percent"`
	RandomAccessPercent     float64 `json:"random_access_percent" yaml:"random_access_percent"`
}

// Supporting types for analytics

// DirectoryUsage contains directory usage information
type DirectoryUsage struct {
	Path       string  `json:"path" yaml:"path"`
	SizeBytes  int64   `json:"size_bytes" yaml:"size_bytes"`
	FileCount  int64   `json:"file_count" yaml:"file_count"`
	Percentage float64 `json:"percentage" yaml:"percentage"`
}

// FileTypeUsage contains file type usage information
type FileTypeUsage struct {
	Extension  string  `json:"extension" yaml:"extension"`
	SizeBytes  int64   `json:"size_bytes" yaml:"size_bytes"`
	FileCount  int64   `json:"file_count" yaml:"file_count"`
	Percentage float64 `json:"percentage" yaml:"percentage"`
}

// CostOptimizationSuggestion contains cost optimization recommendations
type CostOptimizationSuggestion struct {
	Type             string  `json:"type" yaml:"type"` // storage_class, throughput, cleanup
	Description      string  `json:"description" yaml:"description"`
	PotentialSavings float64 `json:"potential_savings" yaml:"potential_savings"` // Monthly savings in USD
	Effort           string  `json:"effort" yaml:"effort"`                       // low, medium, high
	Impact           string  `json:"impact" yaml:"impact"`                       // low, medium, high
	Action           string  `json:"action" yaml:"action"`                       // Specific action to take
}

// UserAccessPattern contains user access pattern information
type UserAccessPattern struct {
	UserId        string    `json:"user_id" yaml:"user_id"`
	Username      string    `json:"username,omitempty" yaml:"username,omitempty"`
	ReadBytes     int64     `json:"read_bytes" yaml:"read_bytes"`
	WriteBytes    int64     `json:"write_bytes" yaml:"write_bytes"`
	Operations    int64     `json:"operations" yaml:"operations"`
	LastAccess    time.Time `json:"last_access" yaml:"last_access"`
	AccessPercent float64   `json:"access_percent" yaml:"access_percent"`
}

// Analytics types for storage analysis
type AnalyticsRequest struct {
	Resources []StorageResource `json:"resources"`
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time"`
	Period    AnalyticsPeriod   `json:"period"`
}

type AnalyticsResult struct {
	Period          AnalyticsPeriod              `json:"period"`
	StartTime       time.Time                    `json:"start_time"`
	EndTime         time.Time                    `json:"end_time"`
	Metrics         map[string]StorageMetrics    `json:"metrics"`
	CostAnalysis    CostAnalysis                 `json:"cost_analysis"`
	Recommendations []OptimizationRecommendation `json:"recommendations"`
}

type StorageResource struct {
	Name       string      `json:"name"`
	Type       StorageType `json:"type"`
	ResourceID string      `json:"resource_id"`
}

type StorageMetrics struct {
	ResourceName   string      `json:"resource_name"`
	ResourceType   StorageType `json:"resource_type"`
	TotalSize      int64       `json:"total_size"`
	AverageSize    int64       `json:"average_size"`
	ObjectCount    int64       `json:"object_count"`
	IOPS           float64     `json:"iops"`
	Throughput     float64     `json:"throughput"`
	PeakThroughput float64     `json:"peak_throughput"`
	LastUpdated    time.Time   `json:"last_updated"`
}

type OptimizationRecommendation struct {
	Type              OptimizationType     `json:"type"`
	Priority          OptimizationPriority `json:"priority"`
	Resource          string               `json:"resource"`
	Title             string               `json:"title"`
	Description       string               `json:"description"`
	PotentialSavings  float64              `json:"potential_savings"`
	ImplementationURL string               `json:"implementation_url"`
}

type OptimizationType string

const (
	OptimizationTypeCost        OptimizationType = "cost"
	OptimizationTypePerformance OptimizationType = "performance"
	OptimizationTypeSecurity    OptimizationType = "security"
)

type OptimizationPriority string

const (
	OptimizationPriorityLow    OptimizationPriority = "low"
	OptimizationPriorityMedium OptimizationPriority = "medium"
	OptimizationPriorityHigh   OptimizationPriority = "high"
)

type AnalyticsPeriod string

const (
	AnalyticsPeriodDaily   AnalyticsPeriod = "daily"
	AnalyticsPeriodWeekly  AnalyticsPeriod = "weekly"
	AnalyticsPeriodMonthly AnalyticsPeriod = "monthly"
	AnalyticsPeriodYearly  AnalyticsPeriod = "yearly"
)

// Usage pattern analysis types
type UsagePatternAnalysis struct {
	AnalysisPeriod         string                          `json:"analysis_period"`
	ResourcePatterns       map[string]ResourceUsagePattern `json:"resource_patterns"`
	PatternRecommendations []PatternRecommendation         `json:"pattern_recommendations"`
}

type ResourceUsagePattern struct {
	ResourceName     string           `json:"resource_name"`
	ResourceType     StorageType      `json:"resource_type"`
	DataPoints       []UsageDataPoint `json:"data_points"`
	PeakUsageHours   []int            `json:"peak_usage_hours"`
	UsageVariability float64          `json:"usage_variability"`
	TrendDirection   string           `json:"trend_direction"`
}

type UsageDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type PatternRecommendation struct {
	Resource       string  `json:"resource"`
	Pattern        string  `json:"pattern"`
	Recommendation string  `json:"recommendation"`
	Confidence     float64 `json:"confidence"`
}

// S3-specific types
type S3Metrics struct {
	BucketName   string    `json:"bucket_name"`
	ObjectCount  int64     `json:"object_count"`
	TotalSize    int64     `json:"total_size"`
	StorageClass string    `json:"storage_class"`
	CostPerMonth float64   `json:"cost_per_month"`
	LastUpdated  time.Time `json:"last_updated"`
}

// Analytics types
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type ServiceCost struct {
	Service string  `json:"service"`
	Cost    float64 `json:"cost"`
	Usage   string  `json:"usage"`
}

type CostAnalysis struct {
	TimeRange       TimeRange     `json:"time_range"`
	TotalCost       float64       `json:"total_cost"`
	Services        []ServiceCost `json:"services"`
	Recommendations []string      `json:"recommendations"`
	LastUpdated     time.Time     `json:"last_updated"`
}

type UsagePattern struct {
	Resource    string  `json:"resource"`
	Pattern     string  `json:"pattern"`
	Confidence  float64 `json:"confidence"`
	Description string  `json:"description"`
}

type UsageAnalysis struct {
	TimeRange       TimeRange               `json:"time_range"`
	Patterns        []UsagePattern          `json:"patterns"`
	Recommendations []PatternRecommendation `json:"recommendations"`
	LastUpdated     time.Time               `json:"last_updated"`
}

type MetricData struct {
	Average float64 `json:"average"`
	Maximum float64 `json:"maximum"`
	Minimum float64 `json:"minimum"`
	Unit    string  `json:"unit"`
}

type PerformanceMetrics struct {
	TimeRange   TimeRange  `json:"time_range"`
	IOPS        MetricData `json:"iops"`
	Throughput  MetricData `json:"throughput"`
	Latency     MetricData `json:"latency"`
	Utilization MetricData `json:"utilization"`
	LastUpdated time.Time  `json:"last_updated"`
}

type Recommendation struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Savings     float64 `json:"savings"`
}

type OptimizationResult struct {
	StorageType     StorageType      `json:"storage_type"`
	CurrentCost     float64          `json:"current_cost"`
	Recommendations []Recommendation `json:"recommendations"`
	OptimalConfig   interface{}      `json:"optimal_config"`
	LastUpdated     time.Time        `json:"last_updated"`
}

type CostEstimate struct {
	Monthly float64 `json:"monthly"`
	Annual  float64 `json:"annual"`
}

type TierInfo struct {
	StorageType   StorageType  `json:"storage_type"`
	Rationale     string       `json:"rationale"`
	EstimatedCost CostEstimate `json:"estimated_cost"`
}

type TierRecommendation struct {
	HotTier      TierInfo     `json:"hot_tier"`
	WarmTier     TierInfo     `json:"warm_tier"`
	ColdTier     TierInfo     `json:"cold_tier"`
	TotalSavings CostEstimate `json:"total_savings"`
	LastUpdated  time.Time    `json:"last_updated"`
}

type ResourceHealth struct {
	ResourceId  string             `json:"resource_id"`
	Type        StorageType        `json:"type"`
	Status      string             `json:"status"`
	Metrics     map[string]float64 `json:"metrics"`
	LastChecked time.Time          `json:"last_checked"`
}

type HealthStatus struct {
	OverallStatus string           `json:"overall_status"`
	Resources     []ResourceHealth `json:"resources"`
	Alerts        []string         `json:"alerts"`
	LastUpdated   time.Time        `json:"last_updated"`
}

// EFS-specific types
type EFSMetrics struct {
	FilesystemId          string    `json:"filesystem_id"`
	SizeInBytes           int64     `json:"size_in_bytes"`
	PerformanceMode       string    `json:"performance_mode"`
	ThroughputMode        string    `json:"throughput_mode"`
	ProvisionedThroughput float64   `json:"provisioned_throughput"`
	LastUpdated           time.Time `json:"last_updated"`
}

type AccessPointInfo struct {
	AccessPointId  string     `json:"access_point_id"`
	FilesystemId   string     `json:"filesystem_id"`
	Path           string     `json:"path"`
	PosixUser      *PosixUser `json:"posix_user,omitempty"`
	CreationTime   time.Time  `json:"creation_time"`
	LifeCycleState string     `json:"lifecycle_state"`
}

type PosixUser struct {
	Uid           uint32   `json:"uid"`
	Gid           uint32   `json:"gid"`
	SecondaryGids []uint32 `json:"secondary_gids,omitempty"`
}

// EBS-specific types
type EBSMetrics struct {
	VolumeId    string    `json:"volume_id"`
	VolumeType  string    `json:"volume_type"`
	Size        int64     `json:"size"`
	State       string    `json:"state"`
	IOPS        int32     `json:"iops"`
	Throughput  int32     `json:"throughput"`
	Encrypted   bool      `json:"encrypted"`
	LastUpdated time.Time `json:"last_updated"`
}

type SnapshotInfo struct {
	SnapshotId  string    `json:"snapshot_id"`
	VolumeId    string    `json:"volume_id"`
	State       string    `json:"state"`
	Progress    string    `json:"progress"`
	StartTime   time.Time `json:"start_time"`
	Description string    `json:"description"`
	VolumeSize  int32     `json:"volume_size"`
}

// Multi-tier storage types
type MultiTierStorageConfig struct {
	HotTier  *StorageRequest `json:"hot_tier,omitempty"`
	WarmTier *StorageRequest `json:"warm_tier,omitempty"`
	ColdTier *StorageRequest `json:"cold_tier,omitempty"`
}

type MultiTierStorageInfo struct {
	Name         string                 `json:"name"`
	CreationTime time.Time              `json:"creation_time"`
	Tiers        map[string]StorageInfo `json:"tiers"`
}

// Storage health monitoring types
type StorageHealthReport struct {
	Timestamp     time.Time            `json:"timestamp"`
	OverallHealth string               `json:"overall_health"`
	ServiceHealth map[string]string    `json:"service_health"`
	Issues        []StorageHealthIssue `json:"issues"`
}

type StorageHealthIssue struct {
	Service   string    `json:"service"`
	Resource  string    `json:"resource,omitempty"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
