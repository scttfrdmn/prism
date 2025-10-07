// Package api provides types for the TUI API client.
package api

import (
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Response types for TUI models - these provide a consistent interface
// for the TUI models to interact with the API client, regardless of
// the actual API implementation.

// System status types

// SystemStatusResponse represents daemon status information
type SystemStatusResponse struct {
	Version           string    `json:"version"`
	Status            string    `json:"status"`
	StartTime         time.Time `json:"start_time"`
	Uptime            string    `json:"uptime,omitempty"`
	ActiveOps         int       `json:"active_ops"`
	TotalRequests     int64     `json:"total_requests"`
	RequestsPerMinute float64   `json:"requests_per_minute,omitempty"`
	AWSRegion         string    `json:"aws_region"`
	AWSProfile        string    `json:"aws_profile,omitempty"`
	CurrentProfile    string    `json:"current_profile,omitempty"`
}

// Idle detection types

// IdlePolicyResponse represents an idle detection policy
type IdlePolicyResponse struct {
	Name      string `json:"name"`
	Threshold int    `json:"threshold"` // Minutes
	Action    string `json:"action"`    // stop, terminate, etc.
}

// ListIdlePoliciesResponse represents a list of idle detection policies
type ListIdlePoliciesResponse struct {
	Policies map[string]IdlePolicyResponse `json:"policies"`
}

// IdlePolicyUpdateRequest represents a request to update an idle policy
type IdlePolicyUpdateRequest struct {
	Name      string `json:"name"`
	Threshold int    `json:"threshold"`
	Action    string `json:"action"`
}

// IdleDetectionResponse represents idle detection status for an instance
type IdleDetectionResponse struct {
	Enabled        bool      `json:"enabled"`
	Policy         string    `json:"policy"`
	IdleTime       int       `json:"idle_time"`       // Minutes
	Threshold      int       `json:"threshold"`       // Minutes
	ActionSchedule time.Time `json:"action_schedule"` // When action will occur
	ActionPending  bool      `json:"action_pending"`  // Whether action is pending
}

// Instance types

// InstanceResponse represents an instance returned from the API
type InstanceResponse struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Template           string    `json:"template"`
	PublicIP           string    `json:"public_ip"`
	PrivateIP          string    `json:"private_ip"`
	State              string    `json:"state"`
	LaunchTime         time.Time `json:"launch_time"`
	HourlyRate         float64   `json:"hourly_rate"`    // AWS list price per hour
	CurrentSpend       float64   `json:"current_spend"`  // Actual accumulated cost since launch
	EffectiveRate      float64   `json:"effective_rate"` // Current spend ÷ hours since launch
	AttachedVolumes    []string  `json:"attached_volumes"`
	AttachedEBSVolumes []string  `json:"attached_ebs_volumes"`
	InstanceLifecycle  string    `json:"instance_lifecycle"` // "spot" or "on-demand"
	Ports              []int     `json:"ports"`
}

// ListInstancesResponse represents a list of instances returned from the API
type ListInstancesResponse struct {
	Instances []InstanceResponse `json:"instances"`
	TotalCost float64            `json:"total_cost"`
}

// LaunchInstanceRequest represents a request to launch an instance
type LaunchInstanceRequest struct {
	Template   string   `json:"template"`
	Name       string   `json:"name"`
	Size       string   `json:"size,omitempty"`
	Volumes    []string `json:"volumes,omitempty"`
	EBSVolumes []string `json:"ebs_volumes,omitempty"`
	Region     string   `json:"region,omitempty"`
	Spot       bool     `json:"spot,omitempty"`
	DryRun     bool     `json:"dry_run,omitempty"`
}

// LaunchInstanceResponse represents a successful launch response
type LaunchInstanceResponse struct {
	Instance       InstanceResponse `json:"instance"`
	Message        string           `json:"message"`
	EstimatedCost  string           `json:"estimated_cost"`
	ConnectionInfo string           `json:"connection_info"`
}

// TemplateResponse represents a template returned from the API
type TemplateResponse struct {
	Name          string             `json:"name"`
	Description   string             `json:"description"`
	AMI           map[string]string  `json:"ami"` // Simplified from original structure
	InstanceType  map[string]string  `json:"instance_type"`
	UserData      string             `json:"user_data"`
	Ports         []int              `json:"ports"`
	EstimatedCost map[string]float64 `json:"estimated_cost"`
}

// ListTemplatesResponse represents a list of templates returned from the API
type ListTemplatesResponse struct {
	Templates map[string]TemplateResponse `json:"templates"`
}

// VolumeResponse represents an EFS volume returned from the API
type VolumeResponse struct {
	Name            string    `json:"name"`
	FileSystemId    string    `json:"filesystem_id"`
	Region          string    `json:"region"`
	CreationTime    time.Time `json:"creation_time"`
	State           string    `json:"state"`
	PerformanceMode string    `json:"performance_mode"`
	ThroughputMode  string    `json:"throughput_mode"`
	EstimatedCostGB float64   `json:"estimated_cost_gb"`
	SizeBytes       int64     `json:"size_bytes"`
}

// ListVolumesResponse represents a list of EFS volumes returned from the API
type ListVolumesResponse struct {
	Volumes map[string]VolumeResponse `json:"volumes"`
}

// StorageResponse represents an EBS volume returned from the API
type StorageResponse struct {
	Name            string    `json:"name"`
	VolumeID        string    `json:"volume_id"`
	Region          string    `json:"region"`
	CreationTime    time.Time `json:"creation_time"`
	State           string    `json:"state"`
	VolumeType      string    `json:"volume_type"`
	SizeGB          int32     `json:"size_gb"`
	IOPS            int32     `json:"iops"`
	Throughput      int32     `json:"throughput"`
	EstimatedCostGB float64   `json:"estimated_cost_gb"`
	AttachedTo      string    `json:"attached_to"`
}

// ListStorageResponse represents a list of EBS volumes returned from the API
type ListStorageResponse struct {
	Storage map[string]StorageResponse `json:"storage"`
}

// Convert from pkg/types to internal/tui/api types

// ToInstanceResponse converts a types.Instance to an InstanceResponse
func ToInstanceResponse(instance types.Instance) InstanceResponse {
	// Look up template info to get ports
	ports := []int{}

	// Real template lookup to get actual port configuration
	template, err := templates.GetTemplateInfo(instance.Template)
	if err == nil && template != nil && template.InstanceDefaults.Ports != nil {
		ports = template.InstanceDefaults.Ports
	} else {
		// Fallback: SSH port is always available
		ports = []int{22}
	}

	return InstanceResponse{
		ID:                 instance.ID,
		Name:               instance.Name,
		Template:           instance.Template,
		PublicIP:           instance.PublicIP,
		PrivateIP:          instance.PrivateIP,
		State:              instance.State,
		LaunchTime:         instance.LaunchTime,
		HourlyRate:         instance.HourlyRate,
		CurrentSpend:       instance.CurrentSpend,
		EffectiveRate:      instance.EffectiveRate,
		AttachedVolumes:    instance.AttachedVolumes,
		AttachedEBSVolumes: instance.AttachedEBSVolumes,
		InstanceLifecycle:  instance.InstanceLifecycle,
		Ports:              ports,
	}
}

// ToListInstancesResponse converts a types.ListResponse to a ListInstancesResponse
func ToListInstancesResponse(resp *types.ListResponse) *ListInstancesResponse {
	result := &ListInstancesResponse{
		TotalCost: resp.TotalCost,
	}

	for _, instance := range resp.Instances {
		result.Instances = append(result.Instances, ToInstanceResponse(instance))
	}

	return result
}

// ToTemplateResponse converts a types.Template to a TemplateResponse
func ToTemplateResponse(name string, template types.Template) TemplateResponse {
	// Simplify AMI mapping for TUI
	amiMap := make(map[string]string)
	for region, archMap := range template.AMI {
		for arch, ami := range archMap {
			key := region + "-" + arch
			amiMap[key] = ami
		}
	}

	return TemplateResponse{
		Name:          name,
		Description:   template.Description,
		AMI:           amiMap,
		InstanceType:  template.InstanceType,
		UserData:      template.UserData,
		Ports:         template.Ports,
		EstimatedCost: template.EstimatedCostPerHour,
	}
}

// ToListTemplatesResponse converts a map of types.Template to a ListTemplatesResponse
func ToListTemplatesResponse(templates map[string]types.Template) *ListTemplatesResponse {
	result := &ListTemplatesResponse{
		Templates: make(map[string]TemplateResponse),
	}

	for name, template := range templates {
		result.Templates[name] = ToTemplateResponse(name, template)
	}

	return result
}

// ToVolumeResponse converts a types.EFSVolume to a VolumeResponse
func ToVolumeResponse(volume types.EFSVolume) VolumeResponse {
	return VolumeResponse{
		Name:            volume.Name,
		FileSystemId:    volume.FileSystemId,
		Region:          volume.Region,
		CreationTime:    volume.CreationTime,
		State:           volume.State,
		PerformanceMode: volume.PerformanceMode,
		ThroughputMode:  volume.ThroughputMode,
		EstimatedCostGB: volume.EstimatedCostGB,
		SizeBytes:       volume.SizeBytes,
	}
}

// ToListVolumesResponse converts a map of types.EFSVolume to a ListVolumesResponse
func ToListVolumesResponse(volumes []types.EFSVolume) *ListVolumesResponse {
	result := &ListVolumesResponse{
		Volumes: make(map[string]VolumeResponse),
	}

	for _, volume := range volumes {
		result.Volumes[volume.Name] = ToVolumeResponse(volume)
	}

	return result
}

// ToStorageResponse converts a types.EBSVolume to a StorageResponse
func ToStorageResponse(storage types.EBSVolume) StorageResponse {
	return StorageResponse{
		Name:            storage.Name,
		VolumeID:        storage.VolumeID,
		Region:          storage.Region,
		CreationTime:    storage.CreationTime,
		State:           storage.State,
		VolumeType:      storage.VolumeType,
		SizeGB:          storage.SizeGB,
		IOPS:            storage.IOPS,
		Throughput:      storage.Throughput,
		EstimatedCostGB: storage.EstimatedCostGB,
		AttachedTo:      storage.AttachedTo,
	}
}

// ToListStorageResponse converts a map of types.EBSVolume to a ListStorageResponse
func ToListStorageResponse(storage []types.EBSVolume) *ListStorageResponse {
	result := &ListStorageResponse{
		Storage: make(map[string]StorageResponse),
	}

	for _, volume := range storage {
		result.Storage[volume.Name] = ToStorageResponse(volume)
	}

	return result
}

// ToSystemStatusResponse converts daemon status to TUI response format
func ToSystemStatusResponse(status *types.DaemonStatus) *SystemStatusResponse {
	return &SystemStatusResponse{
		Version:           status.Version,
		Status:            status.Status,
		StartTime:         status.StartTime,
		Uptime:            status.Uptime,
		ActiveOps:         status.ActiveOps,
		TotalRequests:     status.TotalRequests,
		RequestsPerMinute: status.RequestsPerMinute,
		AWSRegion:         status.AWSRegion,
		AWSProfile:        status.AWSProfile,
		CurrentProfile:    status.CurrentProfile,
	}
}
