// Package api provides types for the TUI API client.
package api

import (
	"time"

	"github.com/scttfrdmn/prism/pkg/templates"
	"github.com/scttfrdmn/prism/pkg/types"
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

// ToVolumeResponse converts a types.StorageVolume (shared/EFS) to a VolumeResponse
func ToVolumeResponse(volume types.StorageVolume) VolumeResponse {
	sizeBytes := int64(0)
	if volume.SizeBytes != nil {
		sizeBytes = *volume.SizeBytes
	}

	return VolumeResponse{
		Name:            volume.Name,
		FileSystemId:    volume.FileSystemID,
		Region:          volume.Region,
		CreationTime:    volume.CreationTime,
		State:           volume.State,
		PerformanceMode: volume.PerformanceMode,
		ThroughputMode:  volume.ThroughputMode,
		EstimatedCostGB: volume.EstimatedCostGB,
		SizeBytes:       sizeBytes,
	}
}

// ToListVolumesResponse converts a slice of types.StorageVolume to a ListVolumesResponse
func ToListVolumesResponse(volumes []*types.StorageVolume) *ListVolumesResponse {
	result := &ListVolumesResponse{
		Volumes: make(map[string]VolumeResponse),
	}

	for _, volume := range volumes {
		// Only include shared storage (EFS) volumes
		if volume.IsShared() {
			result.Volumes[volume.Name] = ToVolumeResponse(*volume)
		}
	}

	return result
}

// ToStorageResponse converts a types.StorageVolume (workspace/EBS) to a StorageResponse
func ToStorageResponse(storage types.StorageVolume) StorageResponse {
	sizeGB := int32(0)
	if storage.SizeGB != nil {
		sizeGB = *storage.SizeGB
	}

	iops := int32(0)
	if storage.IOPS != nil {
		iops = *storage.IOPS
	}

	throughput := int32(0)
	if storage.Throughput != nil {
		throughput = *storage.Throughput
	}

	return StorageResponse{
		Name:            storage.Name,
		VolumeID:        storage.VolumeID,
		Region:          storage.Region,
		CreationTime:    storage.CreationTime,
		State:           storage.State,
		VolumeType:      storage.VolumeType,
		SizeGB:          sizeGB,
		IOPS:            iops,
		Throughput:      throughput,
		EstimatedCostGB: storage.EstimatedCostGB,
		AttachedTo:      storage.AttachedTo,
	}
}

// ToListStorageResponse converts a slice of types.StorageVolume to a ListStorageResponse
func ToListStorageResponse(storage []*types.StorageVolume) *ListStorageResponse {
	result := &ListStorageResponse{
		Storage: make(map[string]StorageResponse),
	}

	for _, volume := range storage {
		// Only include workspace storage (EBS) volumes
		if volume.IsWorkspace() {
			result.Storage[volume.Name] = ToStorageResponse(*volume)
		}
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

// Project and Budget types (Phase 4 Enterprise)

// BudgetStatus represents budget status for a project
type BudgetStatus struct {
	TotalBudget              float64  `json:"total_budget"`
	SpentAmount              float64  `json:"spent_amount"`
	SpentPercentage          float64  `json:"spent_percentage"`
	ActiveAlerts             []string `json:"active_alerts"`
	ProjectedMonthlySpend    float64  `json:"projected_monthly_spend,omitempty"`
	DaysUntilBudgetExhausted *int     `json:"days_until_budget_exhausted,omitempty"`
}

// ProjectResponse represents a project returned from the API
type ProjectResponse struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Description     string        `json:"description,omitempty"`
	Owner           string        `json:"owner"`
	Status          string        `json:"status"`
	MemberCount     int           `json:"member_count"`
	ActiveInstances int           `json:"active_instances"`
	TotalCost       float64       `json:"total_cost"`
	BudgetStatus    *BudgetStatus `json:"budget_status,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	LastActivity    time.Time     `json:"last_activity"`
}

// ListProjectsResponse represents a list of projects returned from the API
type ListProjectsResponse struct {
	Projects []ProjectResponse `json:"projects"`
}

// ProjectFilter represents filters for listing projects
type ProjectFilter struct {
	Status string `json:"status,omitempty"`
	Owner  string `json:"owner,omitempty"`
}

// Policy Framework types (Phase 5A+)

// PolicyStatusResponse represents policy framework status
type PolicyStatusResponse struct {
	Enabled          bool     `json:"enabled"`
	AssignedPolicies []string `json:"assigned_policies"`
	Message          string   `json:"message,omitempty"`
	StatusIcon       string   `json:"status_icon,omitempty"`
}

// PolicySetResponse represents a policy set
type PolicySetResponse struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	PolicyCount int    `json:"policy_count"`
	Status      string `json:"status"`
}

// ListPolicySetsResponse represents a list of policy sets
type ListPolicySetsResponse struct {
	PolicySets []PolicySetResponse `json:"policy_sets"`
}

// TemplateAccessResponse represents template access check result
type TemplateAccessResponse struct {
	Allowed         bool     `json:"allowed"`
	TemplateName    string   `json:"template_name"`
	Reason          string   `json:"reason,omitempty"`
	MatchedPolicies []string `json:"matched_policies,omitempty"`
	Suggestions     []string `json:"suggestions,omitempty"`
}

// Marketplace types (Phase 5B)

// MarketplaceTemplateResponse represents a template from the marketplace
type MarketplaceTemplateResponse struct {
	Name         string   `json:"name"`
	Publisher    string   `json:"publisher"`
	Category     string   `json:"category"`
	Description  string   `json:"description"`
	Rating       float64  `json:"rating"`
	RatingCount  int      `json:"rating_count"`
	Downloads    int64    `json:"downloads"`
	Verified     bool     `json:"verified"`
	Keywords     []string `json:"keywords"`
	SourceURL    string   `json:"source_url,omitempty"`
	License      string   `json:"license,omitempty"`
	Registry     string   `json:"registry,omitempty"`
	RegistryType string   `json:"registry_type,omitempty"`
}

// ListMarketplaceTemplatesResponse represents a list of marketplace templates
type ListMarketplaceTemplatesResponse struct {
	Templates []MarketplaceTemplateResponse `json:"templates"`
}

// MarketplaceFilter represents filters for listing marketplace templates
type MarketplaceFilter struct {
	Query     string  `json:"query,omitempty"`
	Category  string  `json:"category,omitempty"`
	Registry  string  `json:"registry,omitempty"`
	Verified  bool    `json:"verified,omitempty"`
	MinRating float64 `json:"min_rating,omitempty"`
}

// CategoryResponse represents a template category
type CategoryResponse struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	TemplateCount int    `json:"template_count"`
}

// ListCategoriesResponse represents a list of categories
type ListCategoriesResponse struct {
	Categories []CategoryResponse `json:"categories"`
}

// RegistryResponse represents a template registry
type RegistryResponse struct {
	Name          string `json:"name"`
	Type          string `json:"type"` // community, institutional, private, official
	URL           string `json:"url"`
	TemplateCount int    `json:"template_count"`
	Status        string `json:"status"` // active, inactive, syncing
}

// ListRegistriesResponse represents a list of registries
type ListRegistriesResponse struct {
	Registries []RegistryResponse `json:"registries"`
}

// AMI Management types

// AMIResponse represents an AMI
type AMIResponse struct {
	ID           string    `json:"id"`
	TemplateName string    `json:"template_name"`
	Region       string    `json:"region"`
	State        string    `json:"state"`
	Architecture string    `json:"architecture"`
	SizeGB       float64   `json:"size_gb"`
	Description  string    `json:"description,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ListAMIsResponse represents a list of AMIs
type ListAMIsResponse struct {
	AMIs []AMIResponse `json:"amis"`
}

// AMIBuildResponse represents an AMI build job
type AMIBuildResponse struct {
	ID           string    `json:"id"`
	TemplateName string    `json:"template_name"`
	Status       string    `json:"status"`
	Progress     int       `json:"progress"`
	CurrentStep  string    `json:"current_step,omitempty"`
	Error        string    `json:"error,omitempty"`
	StartedAt    time.Time `json:"started_at"`
}

// ListAMIBuildsResponse represents a list of AMI builds
type ListAMIBuildsResponse struct {
	Builds []AMIBuildResponse `json:"builds"`
}

// AMIRegionResponse represents AMI regional coverage
type AMIRegionResponse struct {
	Name     string `json:"name"`
	AMICount int    `json:"ami_count"`
}

// ListAMIRegionsResponse represents a list of AMI regions
type ListAMIRegionsResponse struct {
	Regions []AMIRegionResponse `json:"regions"`
}

// Rightsizing types

// RightsizingRecommendation represents an instance rightsizing recommendation
type RightsizingRecommendation struct {
	InstanceName      string  `json:"instance_name"`
	CurrentType       string  `json:"current_type"`
	RecommendedType   string  `json:"recommended_type"`
	CPUUtilization    float64 `json:"cpu_utilization"`
	MemoryUtilization float64 `json:"memory_utilization"`
	CurrentCost       float64 `json:"current_cost"`
	RecommendedCost   float64 `json:"recommended_cost"`
	MonthlySavings    float64 `json:"monthly_savings"`
	SavingsPercentage float64 `json:"savings_percentage"`
	Confidence        string  `json:"confidence"` // high, medium, low
	Reason            string  `json:"reason,omitempty"`
}

// GetRightsizingRecommendationsResponse represents a list of rightsizing recommendations
type GetRightsizingRecommendationsResponse struct {
	Recommendations []RightsizingRecommendation `json:"recommendations"`
}

// Logs types

// LogsResponse represents logs from an instance
type LogsResponse struct {
	Lines []string `json:"lines"`
}
