package types

import "time"

// StorageType represents user-friendly storage types
type StorageType string

const (
	StorageTypeWorkspace StorageType = "workspace" // Workspace Storage (EBS)
	StorageTypeShared    StorageType = "shared"    // Shared Storage (EFS)
	StorageTypeCloud     StorageType = "cloud"     // Cloud Storage (S3)
)

// AWSService represents the underlying AWS service (technical detail)
type AWSService string

const (
	AWSServiceEBS AWSService = "ebs" // Elastic Block Store
	AWSServiceEFS AWSService = "efs" // Elastic File System
	AWSServiceS3  AWSService = "s3"  // Simple Storage Service
)

// StorageVolume represents a unified storage volume (any type)
type StorageVolume struct {
	Name         string      `json:"name"`          // User-friendly name
	Type         StorageType `json:"type"`          // User-facing type (local/shared/cloud)
	AWSService   AWSService  `json:"aws_service"`   // Technical AWS service
	Region       string      `json:"region"`        // AWS region
	State        string      `json:"state"`         // available, creating, in-use, deleting
	CreationTime time.Time   `json:"creation_time"` // When created

	// Size/capacity (varies by type)
	SizeGB    *int32 `json:"size_gb,omitempty"`    // Size in GB (EBS)
	SizeBytes *int64 `json:"size_bytes,omitempty"` // Size in bytes (EFS)

	// EBS-specific fields
	VolumeID   string `json:"volume_id,omitempty"`   // AWS EBS volume ID
	VolumeType string `json:"volume_type,omitempty"` // gp3, io2, etc.
	IOPS       *int32 `json:"iops,omitempty"`        // Provisioned IOPS
	Throughput *int32 `json:"throughput,omitempty"`  // MB/s throughput
	AttachedTo string `json:"attached_to,omitempty"` // Instance name if attached

	// EFS-specific fields
	FileSystemID    string   `json:"filesystem_id,omitempty"`    // AWS EFS ID
	MountTargets    []string `json:"mount_targets,omitempty"`    // Mount target IDs
	PerformanceMode string   `json:"performance_mode,omitempty"` // generalPurpose, maxIO
	ThroughputMode  string   `json:"throughput_mode,omitempty"`  // bursting, provisioned

	// S3-specific fields
	BucketName string `json:"bucket_name,omitempty"` // S3 bucket name

	// Cost estimation
	EstimatedCostGB float64 `json:"estimated_cost_gb"` // $/GB/month
}

// Helper methods for StorageVolume

// IsWorkspace returns true if this is workspace storage (EBS)
func (sv *StorageVolume) IsWorkspace() bool {
	return sv.Type == StorageTypeWorkspace || sv.AWSService == AWSServiceEBS
}

// IsShared returns true if this is shared storage (EFS)
func (sv *StorageVolume) IsShared() bool {
	return sv.Type == StorageTypeShared || sv.AWSService == AWSServiceEFS
}

// IsCloud returns true if this is cloud storage (S3)
func (sv *StorageVolume) IsCloud() bool {
	return sv.Type == StorageTypeCloud || sv.AWSService == AWSServiceS3
}

// GetDisplayType returns the user-friendly type name
func (sv *StorageVolume) GetDisplayType() string {
	switch sv.Type {
	case StorageTypeWorkspace:
		return "Workspace Storage"
	case StorageTypeShared:
		return "Shared Storage"
	case StorageTypeCloud:
		return "Cloud Storage"
	default:
		return string(sv.Type)
	}
}

// GetTechnicalType returns the AWS service name (for verbose mode)
func (sv *StorageVolume) GetTechnicalType() string {
	switch sv.AWSService {
	case AWSServiceEBS:
		if sv.VolumeType != "" {
			return "EBS " + sv.VolumeType
		}
		return "EBS"
	case AWSServiceEFS:
		return "EFS"
	case AWSServiceS3:
		return "S3"
	default:
		return string(sv.AWSService)
	}
}

// EFSVolume represents a persistent EFS file system
type EFSVolume struct {
	Name            string    `json:"name"`          // User-friendly name
	FileSystemId    string    `json:"filesystem_id"` // AWS EFS ID
	Region          string    `json:"region"`
	CreationTime    time.Time `json:"creation_time"`
	MountTargets    []string  `json:"mount_targets"`     // Mount target IDs
	State           string    `json:"state"`             // available, creating, deleting
	PerformanceMode string    `json:"performance_mode"`  // generalPurpose, maxIO
	ThroughputMode  string    `json:"throughput_mode"`   // bursting, provisioned
	EstimatedCostGB float64   `json:"estimated_cost_gb"` // $/GB/month
	SizeBytes       int64     `json:"size_bytes"`        // Current size
}

// EBSVolume represents a secondary EBS volume for high-performance storage
type EBSVolume struct {
	Name            string    `json:"name"`      // User-friendly name
	VolumeID        string    `json:"volume_id"` // AWS EBS volume ID
	Region          string    `json:"region"`
	CreationTime    time.Time `json:"creation_time"`
	State           string    `json:"state"`             // available, creating, in-use, deleting
	VolumeType      string    `json:"volume_type"`       // gp3, io2, etc.
	SizeGB          int32     `json:"size_gb"`           // Volume size in GB
	IOPS            int32     `json:"iops"`              // Provisioned IOPS (for io2)
	Throughput      int32     `json:"throughput"`        // Throughput in MB/s (for gp3)
	EstimatedCostGB float64   `json:"estimated_cost_gb"` // $/GB/month
	AttachedTo      string    `json:"attached_to"`       // Instance name if attached
}

// VolumeCreateRequest represents a request to create an EFS volume
type VolumeCreateRequest struct {
	Name            string `json:"name"`
	PerformanceMode string `json:"performance_mode,omitempty"` // generalPurpose, maxIO
	ThroughputMode  string `json:"throughput_mode,omitempty"`  // bursting, provisioned
	Region          string `json:"region,omitempty"`
}

// StorageCreateRequest represents a request to create an EBS volume
type StorageCreateRequest struct {
	Name       string `json:"name"`
	Size       string `json:"size"`        // XS, S, M, L, XL or specific GB
	VolumeType string `json:"volume_type"` // gp3, io2
	Region     string `json:"region,omitempty"`
}

// Conversion functions for backward compatibility

// EFSVolumeToStorageVolume converts legacy EFSVolume to unified StorageVolume
func EFSVolumeToStorageVolume(efs *EFSVolume) *StorageVolume {
	if efs == nil {
		return nil
	}

	return &StorageVolume{
		Name:            efs.Name,
		Type:            StorageTypeShared,
		AWSService:      AWSServiceEFS,
		Region:          efs.Region,
		State:           efs.State,
		CreationTime:    efs.CreationTime,
		SizeBytes:       &efs.SizeBytes,
		FileSystemID:    efs.FileSystemId,
		MountTargets:    efs.MountTargets,
		PerformanceMode: efs.PerformanceMode,
		ThroughputMode:  efs.ThroughputMode,
		EstimatedCostGB: efs.EstimatedCostGB,
	}
}

// EBSVolumeToStorageVolume converts legacy EBSVolume to unified StorageVolume
func EBSVolumeToStorageVolume(ebs *EBSVolume) *StorageVolume {
	if ebs == nil {
		return nil
	}

	sizeGB := ebs.SizeGB
	iops := ebs.IOPS
	throughput := ebs.Throughput

	return &StorageVolume{
		Name:            ebs.Name,
		Type:            StorageTypeWorkspace,
		AWSService:      AWSServiceEBS,
		Region:          ebs.Region,
		State:           ebs.State,
		CreationTime:    ebs.CreationTime,
		SizeGB:          &sizeGB,
		VolumeID:        ebs.VolumeID,
		VolumeType:      ebs.VolumeType,
		IOPS:            &iops,
		Throughput:      &throughput,
		AttachedTo:      ebs.AttachedTo,
		EstimatedCostGB: ebs.EstimatedCostGB,
	}
}

// StorageVolumeToEFSVolume converts unified StorageVolume to legacy EFSVolume
func StorageVolumeToEFSVolume(sv *StorageVolume) *EFSVolume {
	if sv == nil || !sv.IsShared() {
		return nil
	}

	var sizeBytes int64
	if sv.SizeBytes != nil {
		sizeBytes = *sv.SizeBytes
	}

	return &EFSVolume{
		Name:            sv.Name,
		FileSystemId:    sv.FileSystemID,
		Region:          sv.Region,
		CreationTime:    sv.CreationTime,
		MountTargets:    sv.MountTargets,
		State:           sv.State,
		PerformanceMode: sv.PerformanceMode,
		ThroughputMode:  sv.ThroughputMode,
		EstimatedCostGB: sv.EstimatedCostGB,
		SizeBytes:       sizeBytes,
	}
}

// StorageVolumeToEBSVolume converts unified StorageVolume to legacy EBSVolume
func StorageVolumeToEBSVolume(sv *StorageVolume) *EBSVolume {
	if sv == nil || !sv.IsWorkspace() {
		return nil
	}

	var sizeGB, iops, throughput int32
	if sv.SizeGB != nil {
		sizeGB = *sv.SizeGB
	}
	if sv.IOPS != nil {
		iops = *sv.IOPS
	}
	if sv.Throughput != nil {
		throughput = *sv.Throughput
	}

	return &EBSVolume{
		Name:            sv.Name,
		VolumeID:        sv.VolumeID,
		Region:          sv.Region,
		CreationTime:    sv.CreationTime,
		State:           sv.State,
		VolumeType:      sv.VolumeType,
		SizeGB:          sizeGB,
		IOPS:            iops,
		Throughput:      throughput,
		EstimatedCostGB: sv.EstimatedCostGB,
		AttachedTo:      sv.AttachedTo,
	}
}
