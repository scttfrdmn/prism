package types

import "time"

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
