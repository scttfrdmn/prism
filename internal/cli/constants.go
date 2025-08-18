// Package cli provides constants for CloudWorkstation's command-line interface.
//
// This file centralizes all hardcoded values used across CLI modules to improve
// maintainability and make configuration changes easier. Constants are organized
// into logical groups with clear documentation.
package cli

import "time"

// =============================================================================
// Network and API Constants
// =============================================================================

const (
	// DefaultDaemonURL is the default URL for the CloudWorkstation daemon
	DefaultDaemonURL = "http://localhost:8947"

	// DefaultDaemonPort is the default port for the CloudWorkstation daemon  
	DefaultDaemonPort = "8947"

	// DaemonURLEnvVar is the environment variable for daemon URL override
	DaemonURLEnvVar = "CWSD_URL"
)

// =============================================================================
// Default Configuration Values
// =============================================================================

const (
	// DefaultVolumeType is the default EBS volume type for storage creation
	DefaultVolumeType = "gp3"

	// DefaultInstanceRetentionMinutes is the default retention period for terminated instances
	DefaultInstanceRetentionMinutes = 5

	// DefaultMountPointPrefix is the default prefix for volume mount points
	DefaultMountPointPrefix = "/mnt/"

	// DefaultConfigDir is the default configuration directory name
	DefaultConfigDir = ".cloudworkstation"

	// DefaultConfigFile is the default daemon configuration file name  
	DefaultConfigFile = "daemon_config.json"
)

// =============================================================================
// Time and Timeout Constants
// =============================================================================

const (
	// DaemonStartupTimeout is the maximum time to wait for daemon startup
	DaemonStartupTimeout = 10 * time.Second

	// DaemonStartupMaxAttempts is the maximum number of daemon ping attempts
	DaemonStartupMaxAttempts = 20

	// DaemonStartupRetryInterval is the interval between daemon ping attempts
	DaemonStartupRetryInterval = 500 * time.Millisecond

	// AMILaunchMonitorTimeout is the maximum time to monitor AMI launch progress (5 minutes)
	AMILaunchMonitorTimeout = 60

	// PackageLaunchMonitorTimeout is the maximum time to monitor package launch progress (20 minutes)
	PackageLaunchMonitorTimeout = 240

	// LaunchProgressInterval is the interval between launch progress checks
	LaunchProgressInterval = 5 * time.Second

	// SetupProgressCheckInterval is the interval for checking setup completion (30 seconds)
	SetupProgressCheckInterval = 30

	// SetupProgressStartDelay is the delay before starting setup progress checks (1 minute)
	SetupProgressStartDelay = 60

	// AnalyticsCollectionInterval is the interval for collecting usage analytics
	AnalyticsCollectionInterval = "2 minutes"
)

// =============================================================================
// User Interface Messages
// =============================================================================

const (
	// DaemonNotRunningMessage is the standard message when daemon is not running
	DaemonNotRunningMessage = "daemon not running. Start with: cws daemon start"

	// UsageLaunchCommand provides the usage string for the launch command
	UsageLaunchCommand = `usage: cws launch <template> <name> [options]
  options: --size XS|S|M|L|XL --volume <name> --storage <size> --project <name> --with conda|apt|dnf|ami --spot --hibernation --dry-run --wait --subnet <subnet-id> --vpc <vpc-id>

  T-shirt sizes (compute + storage):
    XS: 1 vCPU, 2GB RAM + 100GB storage  (t3.small/t4g.small)
    S:  2 vCPU, 4GB RAM + 500GB storage  (t3.medium/t4g.medium)
    M:  2 vCPU, 8GB RAM + 1TB storage    (t3.large/t4g.large) [default]
    L:  4 vCPU, 16GB RAM + 2TB storage   (t3.xlarge/t4g.xlarge)
    XL: 8 vCPU, 32GB RAM + 4TB storage   (t3.2xlarge/t4g.2xlarge)

  GPU workloads automatically scale to GPU instances (g4dn/g5g family)
  Memory-intensive workloads use r5/r6g instances with more RAM
  Compute-intensive workloads use c5/c6g instances for better CPU performance`

	// NoInstancesFoundMessage is displayed when no instances are found
	NoInstancesFoundMessage = "No workstations found. Launch one with: cws launch <template> <name>"

	// NoInstancesFoundProjectMessage is displayed when no instances are found in a project
	NoInstancesFoundProjectMessage = "No workstations found in project '%s'. Launch one with: cws launch <template> <name> --project %s"

	// NoEFSVolumesFoundMessage is displayed when no EFS volumes are found
	NoEFSVolumesFoundMessage = "No EFS volumes found. Create one with: cws volume create <name>"

	// NoEBSVolumesFoundMessage is displayed when no EBS volumes are found
	NoEBSVolumesFoundMessage = "No EBS volumes found. Create one with: cws storage create <name> <size>"
)

// =============================================================================
// File and Path Constants
// =============================================================================

const (
	// AnalyticsLogFile is the path to the analytics log file on instances
	AnalyticsLogFile = "/var/log/cloudworkstation-analytics.json"

	// RightsizingLogFile is the path to the rightsizing recommendations file on instances
	RightsizingLogFile = "/var/log/cloudworkstation-rightsizing.json"

	// DefaultTemplateDir is the default directory for template files
	DefaultTemplateDir = "./templates"

	// TemplateFileExtensionYML is the YAML file extension for templates
	TemplateFileExtensionYML = ".yml"

	// TemplateFileExtensionYAML is the YAML file extension for templates
	TemplateFileExtensionYAML = ".yaml"
)

// =============================================================================
// Numeric Limits and Thresholds
// =============================================================================

const (
	// DefaultFilePermissions is the default file permissions for config files
	DefaultFilePermissions = 0644

	// DefaultDirPermissions is the default directory permissions for config directories
	DefaultDirPermissions = 0755

	// TabWriterMinWidth is the minimum width for tabwriter columns
	TabWriterMinWidth = 0

	// TabWriterTabWidth is the tab width for tabwriter
	TabWriterTabWidth = 8

	// TabWriterPadding is the padding for tabwriter columns
	TabWriterPadding = 2

	// TabWriterPadChar is the padding character for tabwriter
	TabWriterPadChar = ' '

	// TabWriterFlags are the flags for tabwriter
	TabWriterFlags = 0

	// BytesToGB is the conversion factor from bytes to gigabytes
	BytesToGB = 1024 * 1024 * 1024

	// DaysToMonthEstimate is the multiplier for daily cost to monthly estimate
	DaysToMonthEstimate = 30

	// DaysToYearEstimate is the multiplier for daily cost to yearly estimate
	DaysToYearEstimate = 365

	// DefaultAnalyticsSampleCount is the default number of analytics samples to keep
	DefaultAnalyticsSampleCount = 1000

	// DefaultAnalyticsSampleHours is the approximate hours of data in default sample count
	DefaultAnalyticsSampleHours = 33
)

// =============================================================================
// Progress and State Messages
// =============================================================================

const (
	// LaunchProgressAMIMessage is displayed for AMI-based launches
	LaunchProgressAMIMessage = "📦 AMI-based launch - showing instance status..."

	// LaunchProgressPackageMessage is displayed for package-based launches  
	LaunchProgressPackageMessage = "📦 Package-based launch - monitoring setup progress..."

	// LaunchProgressPackageTiming provides timing information for package setups
	LaunchProgressPackageTiming = "💡 Setup time varies: APT/DNF ~2-3 min, conda ~5-10 min"

	// SetupTimeoutMessage is displayed when setup monitoring times out
	SetupTimeoutMessage = "⚠️  Setup monitoring timeout (20 min). Instance may still be setting up."

	// SetupTimeoutHelpMessage provides help when setup times out
	SetupTimeoutHelpMessage = "💡 Check status with: cws list"

	// SetupTimeoutConnectMessage suggests connecting when setup times out
	SetupTimeoutConnectMessage = "💡 Try connecting: cws connect %s"

	// AMITimeoutMessage is displayed when AMI launch monitoring times out
	AMITimeoutMessage = "⚠️  Timeout waiting for instance to start (5 min). Check status with: cws list"
)

// =============================================================================
// Launch Progress State Messages
// =============================================================================

const (
	// StateMessageInitializing is displayed when instance is initializing
	StateMessageInitializing = "⏳ Instance initializing..."

	// StateMessageStarting is displayed when instance is starting
	StateMessageStarting = "🔄 Instance starting... (%ds)"

	// StateMessageRunningReady is displayed when instance is ready
	StateMessageRunningReady = "✅ Instance running! Ready to connect."

	// StateMessageConnectCommand provides the connect command template
	StateMessageConnectCommand = "🔗 Connect: cws connect %s"

	// StateMessageDryRunSuccess is displayed for successful dry runs
	StateMessageDryRunSuccess = "✅ Dry-run validation successful! No actual instance launched."

	// StateMessageUnexpectedStop is displayed when instance stops unexpectedly
	StateMessageUnexpectedStop = "❌ Instance stopped unexpectedly"

	// StateMessageTerminated is displayed when instance is terminated
	StateMessageTerminated = "❌ Instance terminated during launch"

	// StateMessageSetupBegin is displayed when setup begins
	StateMessageSetupBegin = "🔧 Instance running, beginning setup... (%ds)"

	// StateMessageInstallingPackages is displayed during package installation
	StateMessageInstallingPackages = "📥 Installing packages... (%ds)"

	// StateMessageConfiguringServices is displayed during service configuration
	StateMessageConfiguringServices = "⚙️  Configuring services... (%ds)"

	// StateMessageFinalSetup is displayed during final setup steps
	StateMessageFinalSetup = "🔧 Final setup steps... (%ds)"

	// StateMessageSetupComplete is displayed when setup is complete
	StateMessageSetupComplete = "✅ Setup complete! Instance ready."
)

// =============================================================================
// T-Shirt Size Configuration
// =============================================================================

// TSizeSpecs represents the specifications for a t-shirt size
type TSizeSpecs struct {
	CPU     string
	Memory  string
	Storage string
	Cost    float64 // Daily cost estimate
}

// TSizeSpecifications maps t-shirt sizes to their specifications
var TSizeSpecifications = map[string]TSizeSpecs{
	"XS": {"1vCPU", "2GB", "100GB", 0.50},
	"S":  {"2vCPU", "4GB", "500GB", 1.00},
	"M":  {"2vCPU", "8GB", "1TB", 2.00},
	"L":  {"4vCPU", "16GB", "2TB", 4.00},
	"XL": {"8vCPU", "32GB", "4TB", 8.00},
}

// ValidTSizes contains all valid t-shirt sizes
var ValidTSizes = map[string]bool{
	"XS": true,
	"S":  true,
	"M":  true,
	"L":  true,
	"XL": true,
}

// =============================================================================
// Package Manager and Template Constants
// =============================================================================

const (
	// PackageManagerAMI represents AMI-based templates
	PackageManagerAMI = "AMI"

	// PackageManagerAPT represents APT-based templates  
	PackageManagerAPT = "APT"

	// PackageManagerDNF represents DNF-based templates
	PackageManagerDNF = "DNF"

	// PackageManagerConda represents conda-based templates
	PackageManagerConda = "conda"
)

// =============================================================================
// Service Port Mappings
// =============================================================================

// ServicePortMappings maps port numbers to their common services
var ServicePortMappings = map[int]string{
	22:   "SSH",
	80:   "HTTP",
	443:  "HTTPS",
	3306: "MySQL",
	5432: "PostgreSQL",
	6379: "Redis",
	8787: "RStudio Server",
	8888: "Jupyter Notebook",
}

// =============================================================================
// Instance Type Mappings for Scaling
// =============================================================================

// InstanceTypeSizeMapping maps instance types to t-shirt sizes
var InstanceTypeSizeMapping = map[string]string{
	"t3.nano":     "XS",
	"t3.micro":    "XS", 
	"t3.small":    "S",
	"t3.medium":   "M",
	"t3.large":    "L",
	"t3.xlarge":   "XL",
	"t3.2xlarge":  "XL",
	"t3a.nano":    "XS",
	"t3a.micro":   "XS",
	"t3a.small":   "S",
	"t3a.medium":  "M",
	"t3a.large":   "L",
	"t3a.xlarge":  "XL",
	"t3a.2xlarge": "XL",
	"t4g.nano":    "XS",
	"t4g.micro":   "XS",
	"t4g.small":   "S",
	"t4g.medium":  "M",
	"t4g.large":   "L",
	"t4g.xlarge":  "XL",
	"t4g.2xlarge": "XL",
}

// SizeInstanceTypeMapping maps t-shirt sizes to preferred instance types
var SizeInstanceTypeMapping = map[string]string{
	"XS": "t4g.nano",
	"S":  "t4g.small",
	"M":  "t4g.medium", 
	"L":  "t4g.large",
	"XL": "t4g.xlarge",
}

// =============================================================================
// Package Detection Keywords
// =============================================================================

// GPUPackageIndicators contains keywords that indicate GPU requirements
var GPUPackageIndicators = []string{
	"tensorflow-gpu", "pytorch", "cuda", "nvidia", "cupy", "numba", "rapids",
}

// MemoryPackageIndicators contains keywords that indicate high memory requirements
var MemoryPackageIndicators = []string{
	"spark", "hadoop", "r-base", "bioconductor", "genomics",
}

// ComputePackageIndicators contains keywords that indicate high compute requirements
var ComputePackageIndicators = []string{
	"openmpi", "mpich", "openmp", "fftw", "blas", "lapack", "atlas", "mkl",
}

// =============================================================================
// Date and Time Formatting
// =============================================================================

const (
	// StandardDateFormat is the standard date format used throughout the CLI
	StandardDateFormat = "2006-01-02 15:04:05"

	// ShortDateFormat is the short date format for compact displays
	ShortDateFormat = "2006-01-02 15:04"

	// CompactDateFormat is the compact date format for space-constrained displays
	CompactDateFormat = "2006-01-02"

	// ISO8601DateFormat is the ISO8601 date format for API compatibility
	ISO8601DateFormat = "2006-01-02T15:04:05Z"
)

// =============================================================================
// Cost Optimization Constants
// =============================================================================

const (
	// DefaultSavingsEstimate is the default percentage for estimated savings
	DefaultSavingsEstimate = 0.25 // 25%

	// TypicalRightsizingSavingsMin is the minimum typical savings from rightsizing
	TypicalRightsizingSavingsMin = 20 // 20%

	// TypicalRightsizingSavingsMax is the maximum typical savings from rightsizing
	TypicalRightsizingSavingsMax = 40 // 40%

	// OverProvisioningWastePercent is the typical waste from over-provisioning
	OverProvisioningWastePercent = 30 // 30%
)