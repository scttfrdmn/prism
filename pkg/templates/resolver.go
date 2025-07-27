package templates

import (
	"fmt"
	"time"
)

// NewTemplateResolver creates a new template resolver
func NewTemplateResolver() *TemplateResolver {
	return &TemplateResolver{
		Parser:    NewTemplateParser(),
		ScriptGen: NewScriptGenerator(),
		AMIRegistry: getDefaultAMIRegistry(),
	}
}

// ResolveTemplate converts a unified template to a runtime template
func (r *TemplateResolver) ResolveTemplate(template *Template, region, architecture string) (*RuntimeTemplate, error) {
	// Select package manager
	packageManager := r.Parser.Strategy.SelectPackageManager(template)
	
	// Generate installation script
	userDataScript, err := r.ScriptGen.GenerateScript(template, packageManager)
	if err != nil {
		return nil, fmt.Errorf("failed to generate installation script: %w", err)
	}
	
	// Get AMI mapping for this template
	amiMapping, err := r.getAMIMapping(template, region, architecture)
	if err != nil {
		return nil, fmt.Errorf("failed to get AMI mapping: %w", err)
	}
	
	// Get instance type mapping
	instanceTypeMapping := r.getInstanceTypeMapping(template, architecture)
	
	// Get port mapping
	ports := r.getPortMapping(template)
	
	// Get cost estimates
	costMapping := r.getCostMapping(template, architecture)
	
	// Create runtime template
	runtimeTemplate := &RuntimeTemplate{
		Name:         template.Name,
		Description:  template.Description,
		AMI:          amiMapping,
		InstanceType: instanceTypeMapping,
		UserData:     userDataScript,
		Ports:        ports,
		EstimatedCostPerHour: costMapping,
		Source:       template,
		Generated:    time.Now(),
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
	// Check if we have pre-built AMIs for this template
	if templateAMIs, exists := r.AMIRegistry[template.Name]; exists {
		return templateAMIs, nil
	}
	
	// Fall back to base AMI mapping
	baseAMIs := r.Parser.BaseAMIs[template.Base]
	if baseAMIs == nil {
		return nil, fmt.Errorf("no base AMI found for OS: %s", template.Base)
	}
	
	return baseAMIs, nil
}

// getInstanceTypeMapping generates instance type mapping based on template requirements
func (r *TemplateResolver) getInstanceTypeMapping(template *Template, architecture string) map[string]string {
	// If template specifies instance type, use it
	if template.InstanceDefaults.Type != "" {
		return map[string]string{
			"x86_64": template.InstanceDefaults.Type,
			"arm64":  template.InstanceDefaults.Type,
		}
	}
	
	// Smart defaults based on template characteristics
	instanceTypes := r.selectOptimalInstanceTypes(template)
	
	return instanceTypes
}

// selectOptimalInstanceTypes selects optimal instance types based on template characteristics
func (r *TemplateResolver) selectOptimalInstanceTypes(template *Template) map[string]string {
	// Analyze template to determine resource requirements
	requiresGPU := r.templateRequiresGPU(template)
	requiresHighMemory := r.templateRequiresHighMemory(template)
	requiresHighCPU := r.templateRequiresHighCPU(template)
	
	if requiresGPU {
		return map[string]string{
			"x86_64": "g4dn.xlarge",  // NVIDIA T4 GPU
			"arm64":  "g5g.xlarge",  // ARM GPU instance
		}
	}
	
	if requiresHighMemory {
		return map[string]string{
			"x86_64": "r5.large",    // Memory optimized
			"arm64":  "r6g.large",  // ARM memory optimized
		}
	}
	
	if requiresHighCPU {
		return map[string]string{
			"x86_64": "c5.large",    // Compute optimized
			"arm64":  "c6g.large",  // ARM compute optimized
		}
	}
	
	// Default: general purpose
	return map[string]string{
		"x86_64": "t3.medium",   // General purpose
		"arm64":  "t4g.medium", // ARM general purpose (cheaper)
	}
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
		"gcc", "gfortran", "openmpi", "mpich", "openmp",
		"fftw", "blas", "lapack", "compiler", "build-essential",
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
	
	// Generate estimates based on selected instance types
	instanceTypes := r.selectOptimalInstanceTypes(template)
	
	costs := make(map[string]float64)
	
	// Cost estimates for common instance types (approximate)
	costTable := map[string]float64{
		"t3.micro":    0.0116,
		"t3.small":    0.0232,
		"t3.medium":   0.0464,
		"t3.large":    0.0928,
		"t4g.micro":   0.0092,
		"t4g.small":   0.0184,
		"t4g.medium":  0.0368,
		"t4g.large":   0.0736,
		"c5.large":    0.096,
		"c6g.large":   0.0768,
		"r5.large":    0.144,
		"r6g.large":   0.1152,
		"g4dn.xlarge": 0.71,
		"g5g.xlarge":  0.85,
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