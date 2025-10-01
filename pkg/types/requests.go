package types

// LaunchRequest represents a request to launch an instance
type LaunchRequest struct {
	Template       string                 `json:"template"`
	Name           string                 `json:"name"`
	Size           string                 `json:"size,omitempty"`            // XS, S, M, L, XL, GPU-S, etc.
	PackageManager string                 `json:"package_manager,omitempty"` // auto, conda, spack, apt
	Volumes        []string               `json:"volumes,omitempty"`         // EFS volume names to attach
	EBSVolumes     []string               `json:"ebs_volumes,omitempty"`     // EBS volume IDs to attach
	Region         string                 `json:"region,omitempty"`
	SubnetID       string                 `json:"subnet_id,omitempty"`
	VpcID          string                 `json:"vpc_id,omitempty"`
	ProjectID      string                 `json:"project_id,omitempty"`   // Project to associate instance with
	SSHKeyName     string                 `json:"ssh_key_name,omitempty"` // AWS key pair name to use
	Spot           bool                   `json:"spot,omitempty"`
	IdlePolicy     bool                   `json:"idle_policy,omitempty"` // Enable idle policy for automatic cost optimization
	DryRun         bool                   `json:"dry_run,omitempty"`
	Wait           bool                   `json:"wait,omitempty"`          // Wait and show launch progress
	Parameters     map[string]interface{} `json:"parameters,omitempty"`    // Template parameters
	ResearchUser   string                 `json:"research_user,omitempty"` // Research user to create and provision (Phase 5A+)

	// Universal AMI System fields (Phase 5.1)
	AMIStrategy         string               `json:"ami_strategy,omitempty"`          // Override template AMI strategy
	PreferScript        bool                 `json:"prefer_script,omitempty"`         // Prefer script over AMI
	ShowAMIResolution   bool                 `json:"show_ami_resolution,omitempty"`   // Show AMI resolution details
	AMIResolutionResult *AMIResolutionResult `json:"ami_resolution_result,omitempty"` // Internal: resolved AMI info
}

// LaunchResponse represents a successful launch response
type LaunchResponse struct {
	Instance       Instance `json:"instance"`
	Message        string   `json:"message"`
	EstimatedCost  string   `json:"estimated_cost"`
	ConnectionInfo string   `json:"connection_info"`
}

// ListResponse represents a list of instances
type ListResponse struct {
	Instances []Instance `json:"instances"`
	TotalCost float64    `json:"total_cost"`
}

// RollbackRequest represents a request to rollback an instance
type RollbackRequest struct {
	InstanceName string `json:"instance_name"`
	CheckpointID string `json:"checkpoint_id"`
}
