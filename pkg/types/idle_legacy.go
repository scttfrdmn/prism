package types

// Legacy IdleStatus type alias for compatibility
type IdleStatus = InstanceIdleStatus

// IdleStatusResponse represents the response from GetIdleStatus
type IdleStatusResponse struct {
	Enabled        bool                   `json:"enabled"`
	DefaultProfile string                 `json:"default_profile"`
	Profiles       map[string]IdleProfile `json:"profiles"`
	DomainMappings map[string]string      `json:"domain_mappings"`
}

// IdleProfile represents an idle detection profile
type IdleProfile struct {
	Name             string  `json:"name"`
	CPUThreshold     float64 `json:"cpu_threshold"`
	MemoryThreshold  float64 `json:"memory_threshold"`
	NetworkThreshold float64 `json:"network_threshold"`
	DiskThreshold    float64 `json:"disk_threshold"`
	GPUThreshold     float64 `json:"gpu_threshold"`
	IdleMinutes      int     `json:"idle_minutes"`
	Action           string  `json:"action"`
	Notification     bool    `json:"notification"`
}
