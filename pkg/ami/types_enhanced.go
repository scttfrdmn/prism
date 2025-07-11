package ami

import (
	"time"
)

// BuildRequest represents a request to build an AMI.
type BuildRequest struct {
	// Template is the name of the template to build
	Template string `json:"template"`

	// Region is the AWS region to build in
	Region string `json:"region"`

	// SubnetID is the subnet ID to use for the builder instance
	SubnetID string `json:"subnet_id,omitempty"`

	// SecurityGroupID is the security group ID to use for the builder instance
	SecurityGroupID string `json:"security_group_id,omitempty"`

	// InstanceType is the EC2 instance type to use for the builder instance
	InstanceType string `json:"instance_type,omitempty"`

	// Architecture is the CPU architecture to build for (x86_64 or arm64)
	Architecture string `json:"architecture,omitempty"`

	// DryRun indicates whether this is a dry run (no resources created)
	DryRun bool `json:"dry_run,omitempty"`

	// AMIName is the name to give the built AMI
	AMIName string `json:"ami_name,omitempty"`

	// CopyRegions are regions to copy the AMI to after building
	CopyRegions []string `json:"copy_regions,omitempty"`

	// Public indicates whether the AMI should be made public
	Public bool `json:"public,omitempty"`

	// Tags are tags to apply to the AMI
	Tags map[string]string `json:"tags,omitempty"`
}

// BuildResult represents the result of an AMI build.
type BuildResult struct {
	// RequestID is a unique identifier for the build request
	RequestID string `json:"request_id"`

	// Template is the name of the template that was built
	Template string `json:"template"`

	// Region is the AWS region the build was performed in
	Region string `json:"region"`

	// AMIID is the ID of the built AMI
	AMIID string `json:"ami_id"`

	// TemplatePath is the path to the template file
	TemplatePath string `json:"template_path"`

	// Architecture is the CPU architecture of the AMI
	Architecture string `json:"architecture"`

	// BuildTime is the time it took to build the AMI
	BuildTime time.Duration `json:"build_time"`

	// StartTime is the time the build started
	StartTime time.Time `json:"start_time"`

	// EndTime is the time the build completed
	EndTime time.Time `json:"end_time"`

	// Status is the status of the build
	Status string `json:"status"`

	// Error is any error that occurred during the build
	Error string `json:"error,omitempty"`

	// CopiedAMIs are the AMI IDs in other regions
	CopiedAMIs map[string]string `json:"copied_amis,omitempty"`
}

// Template represents a CloudWorkstation template.
type Template struct {
	// Name is the template name
	Name string `yaml:"name" json:"name"`

	// Description is the template description
	Description string `yaml:"description" json:"description"`

	// Base is the base OS image to use
	Base string `yaml:"base" json:"base"`

	// Architecture is the CPU architecture to build for (x86_64 or arm64)
	Architecture string `yaml:"architecture" json:"architecture"`

	// Version is the template version
	Version string `yaml:"version" json:"version"`

	// Domain contains research domain metadata (new in 0.3.0)
	Domain *Domain `yaml:"domain,omitempty" json:"domain,omitempty"`

	// Resources contains resource recommendations
	Resources *Resources `yaml:"resources,omitempty" json:"resources,omitempty"`

	// Cost contains cost estimates
	Cost *Cost `yaml:"cost,omitempty" json:"cost,omitempty"`

	// BuildSteps are the steps to build the AMI
	BuildSteps []BuildStep `yaml:"build_steps" json:"build_steps"`

	// ValidationTests are the tests to validate the AMI
	ValidationTests []ValidationTest `yaml:"validation,omitempty" json:"validation,omitempty"`

	// UserData is the user data script to run at instance launch
	UserData string `yaml:"user_data,omitempty" json:"user_data,omitempty"`

	// IdleDetection contains idle detection configuration (new in 0.3.0)
	IdleDetection *IdleDetection `yaml:"idle_detection,omitempty" json:"idle_detection,omitempty"`

	// Repository contains repository information (new in 0.3.0)
	Repository *Repository `yaml:"repository,omitempty" json:"repository,omitempty"`

	// Dependencies are the template's dependencies (new in 0.3.0)
	Dependencies []Dependency `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`

	// Documentation contains template documentation (new in 0.3.0)
	Documentation *Documentation `yaml:"docs,omitempty" json:"docs,omitempty"`
}

// BuildStep represents a step in building an AMI.
type BuildStep struct {
	// Name is the step name
	Name string `yaml:"name" json:"name"`

	// Description is the step description
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Script is the shell script to execute
	Script string `yaml:"script" json:"script"`

	// TimeoutSeconds is the maximum execution time in seconds
	TimeoutSeconds int `yaml:"timeout_seconds,omitempty" json:"timeout_seconds,omitempty"`
}

// ValidationTest represents a validation test for an AMI.
type ValidationTest struct {
	// Name is the test name
	Name string `yaml:"name" json:"name"`

	// Command is the command to execute
	Command string `yaml:"command,omitempty" json:"command,omitempty"`

	// Script is the shell script to execute
	Script string `yaml:"script,omitempty" json:"script,omitempty"`

	// ExpectedOutput is the expected output of the command (substring match)
	ExpectedOutput string `yaml:"expected_output,omitempty" json:"expected_output,omitempty"`

	// ExpectedExitCode is the expected exit code of the command
	ExpectedExitCode int `yaml:"expected_exit_code,omitempty" json:"expected_exit_code,omitempty"`
}

// Resources contains resource recommendations.
type Resources struct {
	// Sizes maps T-shirt sizes to instance types
	Sizes map[string]Size `yaml:"sizes" json:"sizes"`

	// DefaultSize is the default T-shirt size
	DefaultSize string `yaml:"default_size" json:"default_size"`

	// MemoryRequired is the minimum memory required in GB
	MemoryRequired int `yaml:"memory_required,omitempty" json:"memory_required,omitempty"`

	// CPURequired is the minimum CPU cores required
	CPURequired int `yaml:"cpu_required,omitempty" json:"cpu_required,omitempty"`

	// GPURecommended indicates whether a GPU is recommended
	GPURecommended bool `yaml:"gpu_recommended,omitempty" json:"gpu_recommended,omitempty"`
}

// Size represents an instance size.
type Size struct {
	// InstanceType is the AWS instance type
	InstanceType string `yaml:"instance_type" json:"instance_type"`

	// Architecture is the CPU architecture
	Architecture string `yaml:"architecture" json:"architecture"`
}

// Cost contains cost estimates.
type Cost struct {
	// BaseDailyUSD is the base daily cost estimate in USD for the default size
	BaseDailyUSD float64 `yaml:"base_daily" json:"base_daily"`

	// XSDailyUSD is the daily cost estimate in USD for the XS size
	XSDailyUSD float64 `yaml:"xs_daily,omitempty" json:"xs_daily,omitempty"`

	// SDailyUSD is the daily cost estimate in USD for the S size
	SDailyUSD float64 `yaml:"s_daily,omitempty" json:"s_daily,omitempty"`

	// MDailyUSD is the daily cost estimate in USD for the M size
	MDailyUSD float64 `yaml:"m_daily,omitempty" json:"m_daily,omitempty"`

	// LDailyUSD is the daily cost estimate in USD for the L size
	LDailyUSD float64 `yaml:"l_daily,omitempty" json:"l_daily,omitempty"`

	// XLDailyUSD is the daily cost estimate in USD for the XL size
	XLDailyUSD float64 `yaml:"xl_daily,omitempty" json:"xl_daily,omitempty"`

	// GPUSDailyUSD is the daily cost estimate in USD for the GPU-S size
	GPUSDailyUSD float64 `yaml:"gpu_s_daily,omitempty" json:"gpu_s_daily,omitempty"`

	// GPUMDailyUSD is the daily cost estimate in USD for the GPU-M size
	GPUMDailyUSD float64 `yaml:"gpu_m_daily,omitempty" json:"gpu_m_daily,omitempty"`

	// GPULDailyUSD is the daily cost estimate in USD for the GPU-L size
	GPULDailyUSD float64 `yaml:"gpu_l_daily,omitempty" json:"gpu_l_daily,omitempty"`
}

// Domain contains research domain metadata (new in 0.3.0).
type Domain struct {
	// Category is the top-level research category
	Category string `yaml:"category" json:"category"`

	// Subcategory is the specific research domain
	Subcategory string `yaml:"subcategory" json:"subcategory"`

	// WorkloadType is the computational profile
	WorkloadType string `yaml:"workload_type" json:"workload_type"`

	// AnalysisType is the type of analysis
	AnalysisType string `yaml:"analysis_type,omitempty" json:"analysis_type,omitempty"`

	// DataScale is the expected data scale
	DataScale string `yaml:"data_scale,omitempty" json:"data_scale,omitempty"`

	// CommonTools is a list of common tools included
	CommonTools []string `yaml:"common_tools,omitempty" json:"common_tools,omitempty"`

	// RecommendedStorage is the recommended storage in GB
	RecommendedStorage int `yaml:"recommended_storage,omitempty" json:"recommended_storage,omitempty"`

	// IdleProfile is the default idle detection profile
	IdleProfile string `yaml:"idle_profile,omitempty" json:"idle_profile,omitempty"`
}

// IdleDetection contains idle detection configuration (new in 0.3.0).
type IdleDetection struct {
	// Profile is the idle detection profile to use
	Profile string `yaml:"profile" json:"profile"`

	// CPUThreshold is the CPU usage threshold percentage
	CPUThreshold float64 `yaml:"cpu_threshold,omitempty" json:"cpu_threshold,omitempty"`

	// MemoryThreshold is the memory usage threshold percentage
	MemoryThreshold float64 `yaml:"memory_threshold,omitempty" json:"memory_threshold,omitempty"`

	// NetworkThreshold is the network activity threshold in KBps
	NetworkThreshold float64 `yaml:"network_threshold,omitempty" json:"network_threshold,omitempty"`

	// DiskThreshold is the disk I/O threshold in KBps
	DiskThreshold float64 `yaml:"disk_threshold,omitempty" json:"disk_threshold,omitempty"`

	// GPUThreshold is the GPU usage threshold percentage
	GPUThreshold float64 `yaml:"gpu_threshold,omitempty" json:"gpu_threshold,omitempty"`

	// IdleMinutes is the minutes before an action is taken
	IdleMinutes int `yaml:"idle_minutes,omitempty" json:"idle_minutes,omitempty"`

	// Action is the action to take when idle
	Action string `yaml:"action,omitempty" json:"action,omitempty"`

	// Notification indicates whether to send a notification
	Notification bool `yaml:"notification,omitempty" json:"notification,omitempty"`
}

// Repository contains repository information (new in 0.3.0).
type Repository struct {
	// Name is the repository name
	Name string `yaml:"name" json:"name"`

	// URL is the repository URL
	URL string `yaml:"url" json:"url"`

	// Maintainer is the repository maintainer
	Maintainer string `yaml:"maintainer,omitempty" json:"maintainer,omitempty"`

	// License is the repository license
	License string `yaml:"license,omitempty" json:"license,omitempty"`
}

// Dependency represents a template dependency (new in 0.3.0).
type Dependency struct {
	// Repository is the repository name
	Repository string `yaml:"repository" json:"repository"`

	// Template is the template name
	Template string `yaml:"template" json:"template"`

	// Version is the template version
	Version string `yaml:"version" json:"version"`
}

// Documentation contains template documentation (new in 0.3.0).
type Documentation struct {
	// UsageExamples are examples of using the template
	UsageExamples []UsageExample `yaml:"usage_examples,omitempty" json:"usage_examples,omitempty"`

	// CommonWorkflows are common workflows for the template
	CommonWorkflows []Workflow `yaml:"common_workflows,omitempty" json:"common_workflows,omitempty"`

	// Troubleshooting contains troubleshooting information
	Troubleshooting []TroubleshootingItem `yaml:"troubleshooting,omitempty" json:"troubleshooting,omitempty"`
}

// UsageExample represents a usage example for a template.
type UsageExample struct {
	// Description is the example description
	Description string `yaml:"description" json:"description"`

	// Command is the example command
	Command string `yaml:"command" json:"command"`
}

// Workflow represents a common workflow for a template.
type Workflow struct {
	// Name is the workflow name
	Name string `yaml:"name" json:"name"`

	// Description is the workflow description
	Description string `yaml:"description" json:"description"`

	// Steps are the workflow steps
	Steps []string `yaml:"steps" json:"steps"`
}

// TroubleshootingItem represents a troubleshooting item for a template.
type TroubleshootingItem struct {
	// Problem is the problem description
	Problem string `yaml:"problem" json:"problem"`

	// Solution is the solution to the problem
	Solution string `yaml:"solution" json:"solution"`
}