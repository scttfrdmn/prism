package api

import (
	"time"
)

// InstanceResponse is the response for instance operations
type InstanceResponse struct {
	ID                string      `json:"id"`
	Name              string      `json:"name"`
	Template          string      `json:"template"`
	State             string      `json:"state"`
	LaunchTime        time.Time   `json:"launch_time"`
	PublicIP          string      `json:"public_ip"`
	PrivateIP         string      `json:"private_ip"`
	EstimatedDailyCost float64    `json:"estimated_daily_cost"`
	Username          string      `json:"username"`
	WebPort           int         `json:"web_port"`
	HasWebInterface   bool        `json:"has_web_interface"`
	Ports             []int       `json:"ports"`
	IdleDetection     *IdleStatus `json:"idle_detection"`
	InstanceType      string      `json:"instance_type"`
}

// IdleStatus contains idle detection information for an instance
type IdleStatus struct {
	Enabled       bool      `json:"enabled"`
	Policy        string    `json:"policy"`
	Threshold     int       `json:"threshold"`
	IdleTime      int       `json:"idle_time"`
	ActionSchedule time.Time `json:"action_schedule"`
	ActionPending bool      `json:"action_pending"`
}

// ListInstancesResponse is the response for listing instances
type ListInstancesResponse struct {
	Instances []InstanceResponse `json:"instances"`
	TotalCost float64           `json:"total_cost"`
}

// TemplateResponse is the response for template operations
type TemplateResponse struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	BaseAMI       string                 `json:"base_ami"`
	InstanceType  string                 `json:"instance_type"`
	Tags          map[string]string      `json:"tags"`
	UserData      string                 `json:"user_data"`
	Requirements  map[string]interface{} `json:"requirements"`
	Costs         map[string]float64     `json:"costs"`
	Capabilities  []string               `json:"capabilities"`
}

// ListTemplatesResponse is the response for listing templates
type ListTemplatesResponse struct {
	Templates map[string]TemplateResponse `json:"templates"`
}

// LaunchInstanceResponse is the response for launching an instance
type LaunchInstanceResponse struct {
	Instance InstanceResponse `json:"instance"`
	Message  string          `json:"message"`
}

// VolumeResponse is the response for volume operations
type VolumeResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Size        int       `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	AttachedTo  string    `json:"attached_to"`
	Status      string    `json:"status"`
	Type        string    `json:"type"`
	MountPoint  string    `json:"mount_point"`
	DailyCost   float64   `json:"daily_cost"`
}

// ListVolumesResponse is the response for listing volumes
type ListVolumesResponse struct {
	Volumes   []VolumeResponse `json:"volumes"`
	TotalCost float64         `json:"total_cost"`
}

// StorageResponse is the response for storage operations
type StorageResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Size        int       `json:"size"`
	VolumeType  string    `json:"volume_type"`
	CreatedAt   time.Time `json:"created_at"`
	AttachedTo  string    `json:"attached_to"`
	Status      string    `json:"status"`
	MountPoint  string    `json:"mount_point"`
	DailyCost   float64   `json:"daily_cost"`
}

// ListStorageResponse is the response for listing storage
type ListStorageResponse struct {
	Volumes   []StorageResponse `json:"volumes"`
	TotalCost float64          `json:"total_cost"`
}

// RegistryStatusResponse is the response for registry status
type RegistryStatusResponse struct {
	Enabled     bool      `json:"enabled"`
	LastSync    time.Time `json:"last_sync"`
	SyncStatus  string    `json:"sync_status"`
	ItemCount   int       `json:"item_count"`
}

// AMIReferenceResponse is the response for AMI lookups
type AMIReferenceResponse struct {
	AMI         string    `json:"ami"`
	Region      string    `json:"region"`
	Architecture string   `json:"architecture"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
}