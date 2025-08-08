package types

// LaunchRequest represents a request to launch an instance
type LaunchRequest struct {
	Template       string   `json:"template"`
	Name           string   `json:"name"`
	Size           string   `json:"size,omitempty"`           // XS, S, M, L, XL, GPU-S, etc.
	PackageManager string   `json:"package_manager,omitempty"` // auto, conda, spack, apt
	Volumes        []string `json:"volumes,omitempty"`        // EFS volume names to attach
	EBSVolumes     []string `json:"ebs_volumes,omitempty"`    // EBS volume IDs to attach
	Region         string   `json:"region,omitempty"`
	SubnetID       string   `json:"subnet_id,omitempty"`
	VpcID          string   `json:"vpc_id,omitempty"`
	ProjectID      string   `json:"project_id,omitempty"`     // Project to associate instance with
	SSHKeyName     string   `json:"ssh_key_name,omitempty"`   // AWS key pair name to use
	Spot           bool     `json:"spot,omitempty"`
	Hibernation    bool     `json:"hibernation,omitempty"`    // Enable hibernation support
	DryRun         bool     `json:"dry_run,omitempty"`
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