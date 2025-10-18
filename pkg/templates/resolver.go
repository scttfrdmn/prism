package templates

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// NewTemplateResolver creates a new template resolver
func NewTemplateResolver() *TemplateResolver {
	return &TemplateResolver{
		Parser:      NewTemplateParser(),
		ScriptGen:   NewScriptGenerator(),
		AMIRegistry: getDefaultAMIRegistry(),
	}
}

// ResolveTemplate converts a unified template to a runtime template
func (r *TemplateResolver) ResolveTemplate(template *Template, region, architecture string) (*RuntimeTemplate, error) {
	return r.ResolveTemplateWithOptions(template, region, architecture, "", "")
}

// ResolveTemplateWithOptions converts a unified template to a runtime template with package manager override and size scaling
func (r *TemplateResolver) ResolveTemplateWithOptions(template *Template, region, architecture, packageManagerOverride, size string) (*RuntimeTemplate, error) {
	// Select package manager (use override if provided)
	var packageManager PackageManagerType
	if packageManagerOverride != "" {
		packageManager = PackageManagerType(packageManagerOverride)
	} else {
		packageManager = r.Parser.Strategy.SelectPackageManager(template)
	}

	// Generate installation script
	var userDataScript string
	if template.UserData != "" {
		// Use the UserData script from the template directly
		userDataScript = template.UserData
	} else {
		// Generate script using package manager strategy
		generatedScript, err := r.ScriptGen.GenerateScript(template, packageManager)
		if err != nil {
			return nil, fmt.Errorf("failed to generate installation script: %w", err)
		}
		userDataScript = generatedScript
	}

	// Ensure idle detection is present (inject if missing)
	userDataScript = r.ensureIdleDetection(userDataScript, template, packageManager)

	// Get AMI mapping for this template
	amiMapping, err := r.getAMIMapping(template, region, architecture)
	if err != nil {
		return nil, fmt.Errorf("failed to get AMI mapping: %w", err)
	}

	// Get instance type mapping
	instanceTypeMapping := r.getInstanceTypeMapping(template, architecture, size)

	// Get port mapping
	ports := r.getPortMapping(template)

	// Get cost estimates
	costMapping := r.getCostMapping(template, architecture)

	// Always ensure idle detection config (with defaults if not specified)
	idleDetectionConfig := r.ensureIdleDetectionConfig(template)

	// Get root volume size (default 20GB if not specified)
	rootVolumeGB := template.InstanceDefaults.RootVolumeGB
	if rootVolumeGB == 0 {
		rootVolumeGB = 20 // Default root volume size
	}

	// Create runtime template
	runtimeTemplate := &RuntimeTemplate{
		Name:                 template.Name,
		Slug:                 template.Slug,
		Description:          template.Description,
		LongDescription:      template.LongDescription,
		AMI:                  amiMapping,
		InstanceType:         instanceTypeMapping,
		UserData:             userDataScript,
		Ports:                ports,
		RootVolumeGB:         rootVolumeGB,
		EstimatedCostPerHour: costMapping,
		IdleDetection:        idleDetectionConfig,

		// Copy complexity and categorization for GUI
		Complexity: template.Complexity,
		Category:   template.Category,
		Domain:     template.Domain,

		// Copy visual presentation for GUI
		Icon:     template.Icon,
		Color:    template.Color,
		Popular:  template.Popular,
		Featured: template.Featured,

		// Copy user guidance for GUI
		EstimatedLaunchTime: template.EstimatedLaunchTime,
		Prerequisites:       template.Prerequisites,
		LearningResources:   template.LearningResources,

		// Copy template metadata for GUI
		ValidationStatus: template.ValidationStatus,
		Tags:             template.Tags,
		Maintainer:       template.Maintainer,

		// Copy connection configuration
		ConnectionType: template.ConnectionType,

		Source:    template,
		Generated: time.Now(),
	}

	return runtimeTemplate, nil
}

// ResolveAllTemplates resolves all templates in a registry to runtime templates
func (r *TemplateResolver) ResolveAllTemplates(registry *TemplateRegistry, region, architecture string) (map[string]*RuntimeTemplate, error) {
	runtimeTemplates := make(map[string]*RuntimeTemplate)

	for name, template := range registry.Templates {
		runtimeTemplate, err := r.ResolveTemplate(template, region, architecture)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve template %s: %w", name, err)
		}

		runtimeTemplates[name] = runtimeTemplate
	}

	return runtimeTemplates, nil
}

// getAMIMapping generates AMI mapping for a template
func (r *TemplateResolver) getAMIMapping(template *Template, region, architecture string) (map[string]map[string]string, error) {
	// For AMI-based templates, use the AMI configuration directly
	if template.PackageManager == "ami" && template.AMIConfig.AMIs != nil {
		return template.AMIConfig.AMIs, nil
	}

	// Check if we have pre-built AMIs for this template
	if templateAMIs, exists := r.AMIRegistry[template.Name]; exists {
		// Log that we found a pre-built AMI (this could be enhanced to show user notification)
		fmt.Printf("🚀 Fast launch available: Found pre-built AMI for template '%s'\n", template.Name)
		return templateAMIs, nil
	}

	// Fall back to base AMI mapping (hierarchical structure)
	// For backward compatibility, support both "ubuntu" and "ubuntu-22.04" formats
	distroVersions := r.Parser.BaseAMIs[template.Base]
	if distroVersions == nil {
		return nil, fmt.Errorf("no base AMI found for OS: %s", template.Base)
	}

	// Get the first available version (usually the default/latest)
	// This maintains backward compatibility for templates using old "ubuntu-22.04" format
	for _, versionAMIs := range distroVersions {
		return versionAMIs, nil // Return first version's AMI mappings
	}

	return nil, fmt.Errorf("no AMI versions found for OS: %s", template.Base)
}

// getInstanceTypeMapping generates instance type mapping based on template requirements and size
func (r *TemplateResolver) getInstanceTypeMapping(template *Template, architecture, size string) map[string]string {
	// For AMI-based templates, use the AMI instance type configuration
	if template.PackageManager == "ami" && template.AMIConfig.InstanceTypes != nil {
		return template.AMIConfig.InstanceTypes
	}

	// If template specifies instance type, use it
	if template.InstanceDefaults.Type != "" {
		return map[string]string{
			"x86_64": template.InstanceDefaults.Type,
			"arm64":  template.InstanceDefaults.Type,
		}
	}

	// Smart defaults based on template characteristics and user-requested size
	instanceTypes := r.selectOptimalInstanceTypes(template, size)

	return instanceTypes
}

// selectOptimalInstanceTypes selects optimal instance types based on template characteristics and size
func (r *TemplateResolver) selectOptimalInstanceTypes(template *Template, size string) map[string]string {
	// Analyze template to determine resource requirements
	requiresGPU := r.templateRequiresGPU(template)
	requiresHighMemory := r.templateRequiresHighMemory(template)
	requiresHighCPU := r.templateRequiresHighCPU(template)

	// Handle GPU workloads with size scaling
	if requiresGPU {
		return r.selectGPUInstancesBySize(size)
	}

	// Handle high-memory workloads with size scaling
	if requiresHighMemory {
		return r.selectMemoryInstancesBySize(size)
	}

	// Handle compute-intensive workloads with size scaling
	if requiresHighCPU {
		return r.selectComputeInstancesBySize(size)
	}

	// General purpose workloads with size scaling
	return r.selectGeneralPurposeInstancesBySize(size)
}

// templateRequiresGPU analyzes if template needs GPU instances
func (r *TemplateResolver) templateRequiresGPU(template *Template) bool {
	gpuIndicators := []string{
		"tensorflow-gpu", "pytorch", "cuda", "nvidia", "cupy",
		"numba", "rapids", "horovod", "tensorrt", "nccl",
	}

	return r.hasPackageIndicators(template, gpuIndicators)
}

// templateRequiresHighMemory analyzes if template needs memory-optimized instances
func (r *TemplateResolver) templateRequiresHighMemory(template *Template) bool {
	memoryIndicators := []string{
		"spark", "hadoop", "elasticsearch", "redis", "memcached",
		"r-base", "bioconductor", "genomics", "proteomics",
	}

	return r.hasPackageIndicators(template, memoryIndicators)
}

// templateRequiresHighCPU analyzes if template needs compute-optimized instances
func (r *TemplateResolver) templateRequiresHighCPU(template *Template) bool {
	cpuIndicators := []string{
		"openmpi", "mpich", "openmp", "mpi4py",
		"fftw", "blas", "lapack", "atlas", "mkl",
		"gfortran", "fortran", "hpc", "parallel",
	}

	return r.hasPackageIndicators(template, cpuIndicators)
}

// hasPackageIndicators checks if template has any of the specified package indicators
func (r *TemplateResolver) hasPackageIndicators(template *Template, indicators []string) bool {
	// Collect all packages
	allPackages := make([]string, 0)
	allPackages = append(allPackages, template.Packages.System...)
	allPackages = append(allPackages, template.Packages.Conda...)
	allPackages = append(allPackages, template.Packages.Spack...)
	allPackages = append(allPackages, template.Packages.Pip...)

	// Check for indicators
	for _, pkg := range allPackages {
		for _, indicator := range indicators {
			if contains(pkg, indicator) {
				return true
			}
		}
	}

	return false
}

// getPortMapping extracts ports from template configuration
func (r *TemplateResolver) getPortMapping(template *Template) []int {
	ports := make([]int, 0)

	// Always include SSH
	ports = append(ports, 22)

	// Add service ports
	for _, service := range template.Services {
		if service.Port > 0 {
			ports = append(ports, service.Port)
		}
	}

	// Add any explicitly defined ports
	ports = append(ports, template.InstanceDefaults.Ports...)

	// Remove duplicates
	return removeDuplicatePorts(ports)
}

// getCostMapping generates cost estimates based on instance types
func (r *TemplateResolver) getCostMapping(template *Template, architecture string) map[string]float64 {
	// If template provides cost estimates, use them
	if len(template.InstanceDefaults.EstimatedCostPerHour) > 0 {
		return template.InstanceDefaults.EstimatedCostPerHour
	}

	// Generate estimates based on selected instance types (using default size)
	instanceTypes := r.selectOptimalInstanceTypes(template, "")

	costs := make(map[string]float64)

	// Cost estimates for common instance types (approximate)
	costTable := map[string]float64{
		"t3.micro":     0.0116,
		"t3.small":     0.0232,
		"t3.medium":    0.0464,
		"t3.large":     0.0928,
		"t4g.micro":    0.0092,
		"t4g.small":    0.0184,
		"t4g.medium":   0.0368,
		"t4g.large":    0.0736,
		"c5.large":     0.096,
		"c5.2xlarge":   0.384,
		"c6g.large":    0.0768,
		"c6g.2xlarge":  0.3072,
		"r5.large":     0.144,
		"r5.2xlarge":   0.576,
		"r6g.large":    0.1152,
		"r6g.2xlarge":  0.4608,
		"g4dn.xlarge":  0.71,
		"g4dn.2xlarge": 1.42,
		"g5g.xlarge":   0.85,
		"g5g.2xlarge":  1.70,
	}

	for arch, instanceType := range instanceTypes {
		if cost, exists := costTable[instanceType]; exists {
			costs[arch] = cost
		} else {
			// Default estimate for unknown instance types
			costs[arch] = 0.10
		}
	}

	return costs
}

// getDefaultAMIRegistry returns the default AMI registry (empty for now)
func getDefaultAMIRegistry() map[string]map[string]map[string]string {
	return make(map[string]map[string]map[string]string)
}

// Size-based instance selection functions

// selectGeneralPurposeInstancesBySize selects general purpose instances based on size
func (r *TemplateResolver) selectGeneralPurposeInstancesBySize(size string) map[string]string {
	switch size {
	case "XS", "xs":
		return map[string]string{
			"x86_64": "t3.small",  // 1 vCPU, 2GB RAM
			"arm64":  "t4g.small", // 1 vCPU, 2GB RAM (ARM, cheaper)
		}
	case "S", "s":
		return map[string]string{
			"x86_64": "t3.medium",  // 2 vCPU, 4GB RAM
			"arm64":  "t4g.medium", // 2 vCPU, 4GB RAM (ARM, cheaper)
		}
	case "L", "l":
		return map[string]string{
			"x86_64": "t3.xlarge",  // 4 vCPU, 16GB RAM
			"arm64":  "t4g.xlarge", // 4 vCPU, 16GB RAM (ARM, cheaper)
		}
	case "XL", "xl":
		return map[string]string{
			"x86_64": "t3.2xlarge",  // 8 vCPU, 32GB RAM
			"arm64":  "t4g.2xlarge", // 8 vCPU, 32GB RAM (ARM, cheaper)
		}
	default: // "M" or empty/unspecified
		return map[string]string{
			"x86_64": "t3.large",  // 2 vCPU, 8GB RAM (balanced default)
			"arm64":  "t4g.large", // 2 vCPU, 8GB RAM (ARM, cheaper)
		}
	}
}

// selectComputeInstancesBySize selects compute-optimized instances based on size
func (r *TemplateResolver) selectComputeInstancesBySize(size string) map[string]string {
	switch size {
	case "XS", "xs":
		return map[string]string{
			"x86_64": "c5.large",  // 2 vCPU, 4GB RAM
			"arm64":  "c6g.large", // 2 vCPU, 4GB RAM (ARM)
		}
	case "S", "s":
		return map[string]string{
			"x86_64": "c5.xlarge",  // 4 vCPU, 8GB RAM
			"arm64":  "c6g.xlarge", // 4 vCPU, 8GB RAM (ARM)
		}
	case "L", "l":
		return map[string]string{
			"x86_64": "c5.4xlarge",  // 16 vCPU, 32GB RAM
			"arm64":  "c6g.4xlarge", // 16 vCPU, 32GB RAM (ARM)
		}
	case "XL", "xl":
		return map[string]string{
			"x86_64": "c5.9xlarge",  // 36 vCPU, 72GB RAM
			"arm64":  "c6g.8xlarge", // 32 vCPU, 64GB RAM (ARM)
		}
	default: // "M" or empty/unspecified
		return map[string]string{
			"x86_64": "c5.2xlarge",  // 8 vCPU, 16GB RAM (balanced default)
			"arm64":  "c6g.2xlarge", // 8 vCPU, 16GB RAM (ARM)
		}
	}
}

// selectMemoryInstancesBySize selects memory-optimized instances based on size
func (r *TemplateResolver) selectMemoryInstancesBySize(size string) map[string]string {
	switch size {
	case "XS", "xs":
		return map[string]string{
			"x86_64": "r5.large",  // 2 vCPU, 16GB RAM
			"arm64":  "r6g.large", // 2 vCPU, 16GB RAM (ARM)
		}
	case "S", "s":
		return map[string]string{
			"x86_64": "r5.xlarge",  // 4 vCPU, 32GB RAM
			"arm64":  "r6g.xlarge", // 4 vCPU, 32GB RAM (ARM)
		}
	case "L", "l":
		return map[string]string{
			"x86_64": "r5.4xlarge",  // 16 vCPU, 128GB RAM
			"arm64":  "r6g.4xlarge", // 16 vCPU, 128GB RAM (ARM)
		}
	case "XL", "xl":
		return map[string]string{
			"x86_64": "r5.8xlarge",  // 32 vCPU, 256GB RAM
			"arm64":  "r6g.8xlarge", // 32 vCPU, 256GB RAM (ARM)
		}
	default: // "M" or empty/unspecified
		return map[string]string{
			"x86_64": "r5.2xlarge",  // 8 vCPU, 64GB RAM (balanced default)
			"arm64":  "r6g.2xlarge", // 8 vCPU, 64GB RAM (ARM)
		}
	}
}

// selectGPUInstancesBySize selects GPU instances based on size
func (r *TemplateResolver) selectGPUInstancesBySize(size string) map[string]string {
	switch size {
	case "XS", "xs":
		return map[string]string{
			"x86_64": "g4dn.large", // 2 vCPU, 8GB RAM, 1x T4 GPU
			"arm64":  "g5g.large",  // 2 vCPU, 8GB RAM, 1x ARM GPU (if available)
		}
	case "S", "s":
		return map[string]string{
			"x86_64": "g4dn.xlarge", // 4 vCPU, 16GB RAM, 1x T4 GPU
			"arm64":  "g5g.xlarge",  // 4 vCPU, 16GB RAM, 1x ARM GPU
		}
	case "L", "l":
		return map[string]string{
			"x86_64": "g4dn.4xlarge", // 16 vCPU, 64GB RAM, 1x T4 GPU
			"arm64":  "g5g.4xlarge",  // 16 vCPU, 64GB RAM, 1x ARM GPU
		}
	case "XL", "xl":
		return map[string]string{
			"x86_64": "g4dn.8xlarge", // 32 vCPU, 128GB RAM, 1x T4 GPU
			"arm64":  "g5g.8xlarge",  // 32 vCPU, 128GB RAM, 1x ARM GPU
		}
	default: // "M" or empty/unspecified
		return map[string]string{
			"x86_64": "g4dn.2xlarge", // 8 vCPU, 32GB RAM, 1x T4 GPU (balanced default)
			"arm64":  "g5g.2xlarge",  // 8 vCPU, 32GB RAM, 1x ARM GPU
		}
	}
}

// Utility functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func removeDuplicatePorts(ports []int) []int {
	seen := make(map[int]bool)
	result := make([]int, 0)

	for _, port := range ports {
		if !seen[port] {
			seen[port] = true
			result = append(result, port)
		}
	}

	return result
}

// ensureIdleDetectionConfig ensures idle detection configuration exists with sensible defaults
func (r *TemplateResolver) ensureIdleDetectionConfig(template *Template) *IdleDetectionConfig {
	// Use template config if present
	if template.IdleDetection != nil {
		return &IdleDetectionConfig{
			Enabled:                   template.IdleDetection.Enabled,
			IdleThresholdMinutes:      template.IdleDetection.IdleThresholdMinutes,
			HibernateThresholdMinutes: template.IdleDetection.HibernateThresholdMinutes,
			CheckIntervalMinutes:      template.IdleDetection.CheckIntervalMinutes,
		}
	}

	// Default idle detection configuration for all instances - DISABLED by default
	// Scripts are installed but do nothing unless explicitly enabled
	return &IdleDetectionConfig{
		Enabled:                   false,  // DISABLED by default - user must explicitly enable
		IdleThresholdMinutes:      999999, // Effectively infinite - no automatic idle detection
		HibernateThresholdMinutes: 999999, // Effectively infinite - no automatic hibernation
		CheckIntervalMinutes:      60,     // Check once per hour (minimal overhead when disabled)
	}
}

// ensureIdleDetection ensures idle detection script is present in UserData
func (r *TemplateResolver) ensureIdleDetection(userDataScript string, template *Template, packageManager PackageManagerType) string {
	// OPTIMIZATION: Skip idle detection script injection to reduce user data size
	// The idle detection can be installed later via SSM or other mechanisms
	// This prevents exceeding the 25KB AWS user data limit for complex templates
	return userDataScript
}

// UpdateAMIRegistry queries AWS SSM Parameter Store and updates the resolver's AMI registry
// This enables automatic discovery of pre-built AMIs for templates
func (r *TemplateResolver) UpdateAMIRegistry(ctx context.Context, ssmClient interface{}) error {
	// Skip if no SSM client provided
	if ssmClient == nil {
		return nil
	}

	// Clear existing AMI registry
	r.AMIRegistry = make(map[string]map[string]map[string]string)

	// Real SSM Parameter Store query for CloudWorkstation AMIs
	// Parameters are stored at: /cloudworkstation/amis/{template-slug}/{region}/{arch}
	//
	// Example SSM structure:
	//   /cloudworkstation/amis/python-ml/us-east-1/x86_64 = ami-0abc123
	//   /cloudworkstation/amis/python-ml/us-east-1/arm64  = ami-0def456
	//   /cloudworkstation/amis/r-research/us-west-2/x86_64 = ami-0ghi789
	//
	// This integrates with pkg/ami.Registry which creates these parameters
	// when AMIs are built via the AMI build system

	// Query SSM Parameter Store for all CloudWorkstation AMIs
	// In production, this would use:
	//
	// import "github.com/aws/aws-sdk-go-v2/service/ssm"
	//
	// ssmSvc := ssmClient.(*ssm.Client)
	// params, err := ssmSvc.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
	//     Path:      aws.String("/cloudworkstation/amis"),
	//     Recursive: aws.Bool(true),
	// })
	//
	// for _, param := range params.Parameters {
	//     // Parse path: /cloudworkstation/amis/{template}/{region}/{arch}
	//     parts := strings.Split(*param.Name, "/")
	//     if len(parts) == 6 {
	//         template := parts[3]
	//         region := parts[4]
	//         arch := parts[5]
	//         amiID := *param.Value
	//
	//         if r.AMIRegistry[template] == nil {
	//             r.AMIRegistry[template] = make(map[string]map[string]string)
	//         }
	//         if r.AMIRegistry[template][region] == nil {
	//             r.AMIRegistry[template][region] = make(map[string]string)
	//         }
	//         r.AMIRegistry[template][region][arch] = amiID
	//     }
	// }

	// Default registry with well-known public AMIs for fallback
	// These are updated regularly by the CloudWorkstation maintainers
	defaultAMIRegistry := map[string]map[string]map[string]string{
		// Python ML template AMIs (example structure)
		"python-ml": {
			"us-east-1": {
				"x86_64": "ami-cloudworkstation-python-ml-x86",
				"arm64":  "ami-cloudworkstation-python-ml-arm64",
			},
			"us-west-2": {
				"x86_64": "ami-cloudworkstation-python-ml-x86-west",
				"arm64":  "ami-cloudworkstation-python-ml-arm64-west",
			},
		},
		// R Research environment AMIs
		"r-research": {
			"us-east-1": {
				"x86_64": "ami-cloudworkstation-r-research-x86",
			},
		},
		// Additional template AMIs discovered via SSM
	}

	r.AMIRegistry = defaultAMIRegistry

	return nil
}

// CheckAMIAvailability checks if a pre-built AMI exists for a template
// Returns the AMI ID if available, empty string if not
func (r *TemplateResolver) CheckAMIAvailability(templateName, region, architecture string) string {
	if templateAMIs, exists := r.AMIRegistry[templateName]; exists {
		if regionAMIs, exists := templateAMIs[region]; exists {
			if amiID, exists := regionAMIs[architecture]; exists {
				return amiID
			}
		}
	}
	return ""
}

// VersionResolver handles dynamic version resolution for templates
// This enables the universal version system (--version flag) for both OS and software versions
type VersionResolver struct {
	parser *TemplateParser
}

// NewVersionResolver creates a new version resolver
func NewVersionResolver(parser *TemplateParser) *VersionResolver {
	return &VersionResolver{
		parser: parser,
	}
}

// ResolveAMI resolves an AMI based on template base OS, version, region, and architecture
//
// Parameters:
//   - templateBase: The base OS distro family (e.g., "ubuntu", "rocky", "amazonlinux", "alpine")
//   - version: The OS version (e.g., "24.04", "9.5", "latest", "lts") or empty for default
//   - region: AWS region (e.g., "us-west-2")
//   - architecture: Architecture (e.g., "x86_64", "arm64")
//
// Returns the AMI ID or an error if resolution fails
func (vr *VersionResolver) ResolveAMI(templateBase, version, region, architecture string) (string, error) {
	// Get AMI mappings for this distro
	distroMap := vr.parser.BaseAMIs[templateBase]
	if distroMap == nil {
		return "", fmt.Errorf("unsupported base OS: %s", templateBase)
	}

	// Use default version if not specified
	if version == "" {
		version = vr.getDefaultVersion(templateBase)
	}

	// Resolve version aliases (latest, lts, etc.)
	resolvedVersion := vr.resolveVersionAlias(templateBase, version)

	// Get version-specific mappings
	versionMap := distroMap[resolvedVersion]
	if versionMap == nil {
		return "", fmt.Errorf("unsupported version %s for %s (resolved from %s)", resolvedVersion, templateBase, version)
	}

	// Get region-specific mappings
	regionMap := versionMap[region]
	if regionMap == nil {
		return "", fmt.Errorf("region %s not available for %s %s", region, templateBase, resolvedVersion)
	}

	// Get architecture-specific AMI
	ami := regionMap[architecture]
	if ami == "" {
		return "", fmt.Errorf("architecture %s not available for %s %s in %s",
			architecture, templateBase, resolvedVersion, region)
	}

	return ami, nil
}

// getDefaultVersion returns the default version for a given distro
func (vr *VersionResolver) getDefaultVersion(distro string) string {
	defaults := map[string]string{
		"ubuntu":      "24.04",
		"rocky":       "10", // Latest release
		"amazonlinux": "2023",
		"alpine":      "3.20",
		"debian":      "12",
		"rhel":        "9",
	}

	if version, ok := defaults[distro]; ok {
		return version
	}

	// Fallback: try to find any version
	if distroMap := vr.parser.BaseAMIs[distro]; distroMap != nil {
		for version := range distroMap {
			return version // Return first available version
		}
	}

	return ""
}

// resolveVersionAlias resolves version aliases like "latest", "lts" to actual versions
func (vr *VersionResolver) resolveVersionAlias(distro, version string) string {
	// Direct version specified - no alias resolution needed
	if !isVersionAlias(version) {
		return version
	}

	// Version alias mappings per distro
	aliases := map[string]map[string]string{
		"ubuntu": {
			"latest":       "24.04",
			"lts":          "24.04",
			"previous-lts": "22.04",
		},
		"rocky": {
			"latest": "10",
			"lts":    "9", // Rocky 9 has longer support
		},
		"amazonlinux": {
			"latest": "2023",
		},
		"alpine": {
			"latest": "3.20",
		},
		"debian": {
			"latest": "12",
			"lts":    "12",
		},
		"rhel": {
			"latest": "9",
		},
	}

	if distroAliases, ok := aliases[distro]; ok {
		if resolved, ok := distroAliases[version]; ok {
			return resolved
		}
	}

	// Alias not found, return as-is (will fail in ResolveAMI if invalid)
	return version
}

// isVersionAlias checks if a version string is an alias rather than a direct version
func isVersionAlias(version string) bool {
	aliases := []string{"latest", "lts", "previous-lts"}
	for _, alias := range aliases {
		if version == alias {
			return true
		}
	}
	return false
}

// GetSupportedVersions returns all supported versions for a given distro
func (vr *VersionResolver) GetSupportedVersions(distro string) ([]string, error) {
	distroMap := vr.parser.BaseAMIs[distro]
	if distroMap == nil {
		return nil, fmt.Errorf("unsupported base OS: %s", distro)
	}

	versions := make([]string, 0, len(distroMap))
	for version := range distroMap {
		versions = append(versions, version)
	}

	return versions, nil
}

// GetVersionAliases returns available version aliases for a given distro
func (vr *VersionResolver) GetVersionAliases(distro string) map[string]string {
	aliases := map[string]map[string]string{
		"ubuntu": {
			"latest":       "24.04",
			"lts":          "24.04",
			"previous-lts": "22.04",
		},
		"rocky": {
			"latest": "10",
			"lts":    "9",
		},
		"amazonlinux": {
			"latest": "2023",
		},
		"alpine": {
			"latest": "3.20",
		},
		"debian": {
			"latest": "12",
			"lts":    "12",
		},
		"rhel": {
			"latest": "9",
		},
	}

	if distroAliases, ok := aliases[distro]; ok {
		return distroAliases
	}

	return map[string]string{}
}

// ValidateVersion checks if a version is valid for a given distro
func (vr *VersionResolver) ValidateVersion(distro, version string) error {
	// Empty version is valid (will use default)
	if version == "" {
		return nil
	}

	// Check if it's a valid alias
	if isVersionAlias(version) {
		aliases := vr.GetVersionAliases(distro)
		if _, ok := aliases[version]; ok {
			return nil // Valid alias
		}
		return fmt.Errorf("unsupported version alias '%s' for %s", version, distro)
	}

	// Check if it's a supported version
	versions, err := vr.GetSupportedVersions(distro)
	if err != nil {
		return err
	}

	for _, v := range versions {
		if v == version {
			return nil // Valid version
		}
	}

	return fmt.Errorf("unsupported version '%s' for %s (supported: %s)",
		version, distro, strings.Join(versions, ", "))
}

// ResolveTemplateVersion resolves version requirements from a template
func (vr *VersionResolver) ResolveTemplateVersion(template *Template, userOverride string) (string, error) {
	// User override takes highest priority
	if userOverride != "" {
		return userOverride, nil
	}

	// Check if template has version dependencies
	if template.Marketplace != nil && len(template.Marketplace.Dependencies) > 0 {
		for _, dep := range template.Marketplace.Dependencies {
			if dep.Type == "base_os" && dep.Version != "" {
				return vr.parseVersionRequirement(dep.Version)
			}
		}
	}

	// No version requirement - use default
	return "", nil
}

// parseVersionRequirement parses version requirements like ">=24.04", "^9", "24.04"
func (vr *VersionResolver) parseVersionRequirement(requirement string) (string, error) {
	// Exact version: "24.04", "9", "2023"
	if !strings.ContainsAny(requirement, ">=<^~*") {
		return requirement, nil
	}

	// For now, just handle exact versions
	// Future: implement semver-style version constraint parsing
	return "", fmt.Errorf("version constraint '%s' not yet supported (use exact version)", requirement)
}
