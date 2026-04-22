package types

import "time"

// LaunchRequest represents a request to launch an instance
type LaunchRequest struct {
	Template            string                 `json:"template"`
	Name                string                 `json:"name"`
	Size                string                 `json:"size,omitempty"`            // XS, S, M, L, XL, GPU-S, etc.
	PackageManager      string                 `json:"package_manager,omitempty"` // auto, conda, spack, apt
	Volumes             []string               `json:"volumes,omitempty"`         // EFS volume names to attach
	EBSVolumes          []string               `json:"ebs_volumes,omitempty"`     // EBS volume IDs to attach
	Region              string                 `json:"region,omitempty"`
	SubnetID            string                 `json:"subnet_id,omitempty"`
	VpcID               string                 `json:"vpc_id,omitempty"`
	ProjectID           string                 `json:"project_id,omitempty"`            // Project to associate instance with
	CourseID            string                 `json:"course_id,omitempty"`             // Course to enforce template policy (v0.14.0)
	FundingAllocationID string                 `json:"funding_allocation_id,omitempty"` // Budget allocation to charge (v0.5.10+)
	SSHKeyName          string                 `json:"ssh_key_name,omitempty"`          // AWS key pair name to use
	Spot                bool                   `json:"spot,omitempty"`
	IdlePolicy          bool                   `json:"idle_policy,omitempty"` // Enable idle policy for automatic cost optimization
	DryRun              bool                   `json:"dry_run,omitempty"`
	Wait                bool                   `json:"wait,omitempty"`          // Wait and show launch progress
	Quiet               bool                   `json:"quiet,omitempty"`         // Suppress progress output (for scripting)
	NoProgress          bool                   `json:"no_progress,omitempty"`   // Disable progress monitoring
	AutoYes             bool                   `json:"auto_yes,omitempty"`      // Skip confirmation prompt (--yes)
	Parameters          map[string]interface{} `json:"parameters,omitempty"`    // Template parameters
	ResearchUser        string                 `json:"research_user,omitempty"` // Research user to create and provision (Phase 5A+)

	// Universal Version System (v0.5.5)
	Version string `json:"version,omitempty"` // OS version (e.g., "24.04", "22.04", "9", "10", "latest", "lts")

	// Universal AMI System fields (Phase 5.1)
	AMIStrategy         string               `json:"ami_strategy,omitempty"`          // Override template AMI strategy
	PreferScript        bool                 `json:"prefer_script,omitempty"`         // Prefer script over AMI
	ShowAMIResolution   bool                 `json:"show_ami_resolution,omitempty"`   // Show AMI resolution details
	AMIResolutionResult *AMIResolutionResult `json:"ami_resolution_result,omitempty"` // Internal: resolved AMI info
	CustomAMI           string               `json:"custom_ami,omitempty"`            // Override AMI ID (e.g., for restore from snapshot)

	// Time-boxing (#146): set ExpiresAt directly or derive from Hours
	Hours     int        `json:"hours,omitempty"`      // Auto-stop after N hours (convenience)
	ExpiresAt *time.Time `json:"expires_at,omitempty"` // Explicit expiry timestamp

	// EC2 Capacity Blocks (#63): pin launch to a pre-reserved capacity reservation
	CapacityBlockID string `json:"capacity_block_id,omitempty"`

	// Approval workflow (#495): create an approval request instead of launching
	RequestApproval bool `json:"request_approval,omitempty"`

	// Approval workflow (#495): launch using a pre-approved request ID
	ApprovalID string `json:"approval_id,omitempty"`

	// spored daemon integration (#588)
	DNSName     string `json:"dns_name,omitempty"`     // DNS record name (defaults to sanitized workspace name)
	TTL         string `json:"ttl,omitempty"`          // Time-to-live (e.g., "8h", "24h"); empty = no limit
	IdleTimeout string `json:"idle_timeout,omitempty"` // Idle timeout override (e.g., "30m", "2h")

	// Slack/Teams bot notifications (#607)
	SlackWorkspaceID string `json:"slack_workspace_id,omitempty"` // Enables lifecycle DMs via spore-bot
}

// LaunchResponse represents a successful launch response
type LaunchResponse struct {
	Instance       Instance `json:"instance"`
	Message        string   `json:"message"`
	EstimatedCost  string   `json:"estimated_cost"`
	ConnectionInfo string   `json:"connection_info"`

	// ApprovalPending is set when the daemon returned HTTP 202 (approval required)
	// instead of launching. The approval request ID is in ApprovalRequestID.
	ApprovalPending   bool   `json:"approval_pending,omitempty"`
	ApprovalRequestID string `json:"approval_request_id,omitempty"`
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

// ExecRequest represents a request to execute a command on an instance
type ExecRequest struct {
	Command        string            `json:"command"`                   // Command to execute
	WorkingDir     string            `json:"working_dir,omitempty"`     // Working directory
	User           string            `json:"user,omitempty"`            // User to execute as
	Environment    map[string]string `json:"environment,omitempty"`     // Environment variables
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"` // Command timeout
	Interactive    bool              `json:"interactive,omitempty"`     // Interactive session
}

// ExecResult represents the result of command execution
type ExecResult struct {
	Command       string `json:"command"`
	ExitCode      int    `json:"exit_code"`
	StdOut        string `json:"stdout"`
	StdErr        string `json:"stderr"`
	Status        string `json:"status"`               // success, failed, timeout
	ExecutionTime int    `json:"execution_time"`       // Execution time in milliseconds
	CommandID     string `json:"command_id,omitempty"` // SSM command ID for reference
}

// ResizeRequest represents a request to resize an instance
type ResizeRequest struct {
	InstanceName       string `json:"instance_name"`
	TargetInstanceType string `json:"target_instance_type"`
	Force              bool   `json:"force,omitempty"`
	Wait               bool   `json:"wait,omitempty"`
}

// ResizeResponse represents the response to a resize request
type ResizeResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	InstanceID string `json:"instance_id,omitempty"`
	OldType    string `json:"old_type,omitempty"`
	NewType    string `json:"new_type,omitempty"`
	Status     string `json:"status,omitempty"`
}
