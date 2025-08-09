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
		Parser:    NewTemplateParser(),
		ScriptGen: NewScriptGenerator(),
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

	// Create runtime template
	runtimeTemplate := &RuntimeTemplate{
		Name:         template.Name,
		Description:  template.Description,
		AMI:          amiMapping,
		InstanceType: instanceTypeMapping,
		UserData:     userDataScript,
		Ports:        ports,
		EstimatedCostPerHour: costMapping,
		IdleDetection: idleDetectionConfig,
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
	
	// Fall back to base AMI mapping
	baseAMIs := r.Parser.BaseAMIs[template.Base]
	if baseAMIs == nil {
		return nil, fmt.Errorf("no base AMI found for OS: %s", template.Base)
	}
	
	return baseAMIs, nil
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

// Size-based instance selection functions

// selectGeneralPurposeInstancesBySize selects general purpose instances based on size
func (r *TemplateResolver) selectGeneralPurposeInstancesBySize(size string) map[string]string {
	switch size {
	case "XS", "xs":
		return map[string]string{
			"x86_64": "t3.small",   // 1 vCPU, 2GB RAM
			"arm64":  "t4g.small",  // 1 vCPU, 2GB RAM (ARM, cheaper)
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
			"x86_64": "t3.large",   // 2 vCPU, 8GB RAM (balanced default)
			"arm64":  "t4g.large",  // 2 vCPU, 8GB RAM (ARM, cheaper)
		}
	}
}

// selectComputeInstancesBySize selects compute-optimized instances based on size
func (r *TemplateResolver) selectComputeInstancesBySize(size string) map[string]string {
	switch size {
	case "XS", "xs":
		return map[string]string{
			"x86_64": "c5.large",   // 2 vCPU, 4GB RAM
			"arm64":  "c6g.large",  // 2 vCPU, 4GB RAM (ARM)
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
			"x86_64": "r5.large",   // 2 vCPU, 16GB RAM
			"arm64":  "r6g.large",  // 2 vCPU, 16GB RAM (ARM)
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
			"x86_64": "g4dn.large",   // 2 vCPU, 8GB RAM, 1x T4 GPU
			"arm64":  "g5g.large",    // 2 vCPU, 8GB RAM, 1x ARM GPU (if available)
		}
	case "S", "s":
		return map[string]string{
			"x86_64": "g4dn.xlarge",  // 4 vCPU, 16GB RAM, 1x T4 GPU
			"arm64":  "g5g.xlarge",   // 4 vCPU, 16GB RAM, 1x ARM GPU
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
			Enabled:                  template.IdleDetection.Enabled,
			IdleThresholdMinutes:     template.IdleDetection.IdleThresholdMinutes,
			HibernateThresholdMinutes: template.IdleDetection.HibernateThresholdMinutes,
			CheckIntervalMinutes:     template.IdleDetection.CheckIntervalMinutes,
		}
	}
	
	// Default idle detection configuration for all instances - DISABLED by default
	// Scripts are installed but do nothing unless explicitly enabled
	return &IdleDetectionConfig{
		Enabled:                  false, // DISABLED by default - user must explicitly enable
		IdleThresholdMinutes:     999999, // Effectively infinite - no automatic idle detection
		HibernateThresholdMinutes: 999999, // Effectively infinite - no automatic hibernation
		CheckIntervalMinutes:     60,     // Check once per hour (minimal overhead when disabled)
	}
}

// ensureIdleDetection ensures idle detection script is present in UserData
func (r *TemplateResolver) ensureIdleDetection(userDataScript string, template *Template, packageManager PackageManagerType) string {
	// Check if idle detection script is already present
	if strings.Contains(userDataScript, "cloudworkstation-idle-check.sh") {
		return userDataScript // Already has idle detection
	}
	
	// Get idle detection configuration (with proper defaults)
	idleConfig := r.ensureIdleDetectionConfig(template)
	
	// Inject universal idle detection script based on package manager
	idleScript := r.getIdleDetectionScript(packageManager)
	
	// Replace template placeholders with actual values
	enabledStr := "false"
	if idleConfig.Enabled {
		enabledStr = "true"
	}
	idleScript = strings.ReplaceAll(idleScript, "{{ENABLED}}", enabledStr)
	idleScript = strings.ReplaceAll(idleScript, "{{IDLE_THRESHOLD_MINUTES}}", fmt.Sprintf("%d", idleConfig.IdleThresholdMinutes))
	idleScript = strings.ReplaceAll(idleScript, "{{HIBERNATE_THRESHOLD_MINUTES}}", fmt.Sprintf("%d", idleConfig.HibernateThresholdMinutes))
	idleScript = strings.ReplaceAll(idleScript, "{{CHECK_INTERVAL_MINUTES}}", fmt.Sprintf("%d", idleConfig.CheckIntervalMinutes))
	
	// Append idle detection to existing script
	return userDataScript + "\n\n# Universal CloudWorkstation Idle Detection\n" + idleScript
}

// getIdleDetectionScript returns the appropriate idle detection script for the package manager
func (r *TemplateResolver) getIdleDetectionScript(packageManager PackageManagerType) string {
	baseScript := `
# Install CloudWorkstation Idle Detection Agent
cat > /usr/local/bin/cloudworkstation-idle-check.sh << 'EOF'
#!/bin/bash
# CloudWorkstation Idle Detection Agent
# Version: 1.1.0 (Universal)
# Description: Autonomous idle detection with hibernation/stop capabilities

set -euo pipefail

# Configuration - read from config file or use defaults
CONFIG_FILE="/etc/cloudworkstation/idle-config"
LOG_FILE="/var/log/cloudworkstation-idle.log"

# Read configuration with defaults
if [[ -f "$CONFIG_FILE" ]]; then
    source "$CONFIG_FILE"
fi

# Default values (used if config file doesn't exist or values not set)
ENABLED=${ENABLED:-{{ENABLED}}}
IDLE_THRESHOLD_MINUTES=${IDLE_THRESHOLD_MINUTES:-{{IDLE_THRESHOLD_MINUTES}}}
HIBERNATE_THRESHOLD_MINUTES=${HIBERNATE_THRESHOLD_MINUTES:-{{HIBERNATE_THRESHOLD_MINUTES}}}
CHECK_INTERVAL_MINUTES=${CHECK_INTERVAL_MINUTES:-{{CHECK_INTERVAL_MINUTES}}}

# Logging function
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') [IDLE-AGENT v1.1.0] $*" | tee -a "$LOG_FILE"
}

# Get instance metadata using IMDSv2
get_instance_metadata() {
    # Get IMDSv2 token
    TOKEN=$(curl -s --max-time 5 -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")
    
    if [[ -z "$TOKEN" ]]; then
        log "ERROR: Failed to get IMDSv2 token"
        return 1
    fi
    
    INSTANCE_ID=$(curl -s --max-time 5 -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/instance-id)
    REGION=$(curl -s --max-time 5 -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/placement/region 2>/dev/null || \
             curl -s --max-time 5 -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/placement/availability-zone | sed 's/.$//')
    
    if [[ -z "$REGION" ]]; then
        REGION="us-west-2"
        log "Warning: Could not detect region, defaulting to us-west-2"
    fi
    
    if [[ -z "$INSTANCE_ID" ]]; then
        log "ERROR: Could not get instance ID"
        return 1
    fi
    
    log "Instance ID: $INSTANCE_ID, Region: $REGION"
}

# Collect detailed usage analytics for rightsizing
collect_usage_analytics() {
    local analytics_file="/var/log/cloudworkstation-analytics.json"
    local timestamp=$(date -Iseconds)
    
    # CPU metrics
    CPU_LOAD_1MIN=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | tr -d ',' | xargs)
    CPU_LOAD_5MIN=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $2}' | tr -d ',' | xargs)
    CPU_LOAD_15MIN=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $3}' | tr -d ',' | xargs)
    CPU_COUNT=$(nproc)
    
    # Memory metrics (in MB)
    MEMORY_TOTAL=$(free -m | grep '^Mem:' | awk '{print $2}')
    MEMORY_USED=$(free -m | grep '^Mem:' | awk '{print $3}')
    MEMORY_FREE=$(free -m | grep '^Mem:' | awk '{print $4}')
    MEMORY_AVAILABLE=$(free -m | grep '^Mem:' | awk '{print $7}')
    MEMORY_PERCENT=$(( (MEMORY_USED * 100) / MEMORY_TOTAL ))
    
    # Disk metrics
    DISK_TOTAL=$(df -BG / | tail -1 | awk '{print $2}' | tr -d 'G')
    DISK_USED=$(df -BG / | tail -1 | awk '{print $3}' | tr -d 'G')
    DISK_AVAILABLE=$(df -BG / | tail -1 | awk '{print $4}' | tr -d 'G')
    DISK_PERCENT=$(df / | tail -1 | awk '{print $5}' | tr -d '%')
    
    # Network metrics (bytes per second - approximate)
    NETWORK_RX_BYTES=$(cat /proc/net/dev | grep -E "eth0|ens" | head -1 | awk '{print $2}' || echo "0")
    NETWORK_TX_BYTES=$(cat /proc/net/dev | grep -E "eth0|ens" | head -1 | awk '{print $10}' || echo "0")
    
    # Process count
    PROCESS_COUNT=$(ps aux | wc -l)
    
    # Active users
    USERS_LOGGED_IN=$(who | grep -v '^root' | wc -l)
    
    # GPU metrics (if available)
    GPU_USAGE="0"
    GPU_MEMORY_USED="0"
    GPU_MEMORY_TOTAL="0"
    GPU_TEMPERATURE="0"
    GPU_POWER="0"
    GPU_COUNT="0"
    
    if command -v nvidia-smi &> /dev/null; then
        GPU_COUNT=$(nvidia-smi --list-gpus | wc -l 2>/dev/null || echo "0")
        if [[ "$GPU_COUNT" -gt 0 ]]; then
            GPU_USAGE=$(nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits 2>/dev/null | head -1 || echo "0")
            GPU_MEMORY_USED=$(nvidia-smi --query-gpu=memory.used --format=csv,noheader,nounits 2>/dev/null | head -1 || echo "0")
            GPU_MEMORY_TOTAL=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits 2>/dev/null | head -1 || echo "0")
            GPU_TEMPERATURE=$(nvidia-smi --query-gpu=temperature.gpu --format=csv,noheader,nounits 2>/dev/null | head -1 || echo "0")
            GPU_POWER=$(nvidia-smi --query-gpu=power.draw --format=csv,noheader,nounits 2>/dev/null | head -1 | cut -d'.' -f1 || echo "0")
        fi
    fi
    
    # Create analytics JSON entry
    cat >> "$analytics_file" << EOF
{
  "timestamp": "$timestamp",
  "cpu": {
    "load_1min": $CPU_LOAD_1MIN,
    "load_5min": $CPU_LOAD_5MIN,
    "load_15min": $CPU_LOAD_15MIN,
    "core_count": $CPU_COUNT,
    "utilization_percent": $(echo "scale=2; ($CPU_LOAD_1MIN / $CPU_COUNT) * 100" | bc -l 2>/dev/null || echo "0")
  },
  "memory": {
    "total_mb": $MEMORY_TOTAL,
    "used_mb": $MEMORY_USED,
    "free_mb": $MEMORY_FREE,
    "available_mb": $MEMORY_AVAILABLE,
    "utilization_percent": $MEMORY_PERCENT
  },
  "disk": {
    "total_gb": $DISK_TOTAL,
    "used_gb": $DISK_USED,
    "available_gb": $DISK_AVAILABLE,
    "utilization_percent": $DISK_PERCENT
  },
  "network": {
    "rx_bytes": $NETWORK_RX_BYTES,
    "tx_bytes": $NETWORK_TX_BYTES
  },
  "gpu": {
    "count": $GPU_COUNT,
    "utilization_percent": $GPU_USAGE,
    "memory_used_mb": $GPU_MEMORY_USED,
    "memory_total_mb": $GPU_MEMORY_TOTAL,
    "temperature_celsius": $GPU_TEMPERATURE,
    "power_draw_watts": $GPU_POWER
  },
  "system": {
    "process_count": $PROCESS_COUNT,
    "users_logged_in": $USERS_LOGGED_IN,
    "uptime_seconds": $(cat /proc/uptime | cut -d' ' -f1 | cut -d'.' -f1)
  }
}
EOF
    
    # Keep only last 1000 analytics entries to prevent log bloat
    if [[ -f "$analytics_file" ]]; then
        tail -1000 "$analytics_file" > "$analytics_file.tmp" && mv "$analytics_file.tmp" "$analytics_file"
        chown ubuntu:ubuntu "$analytics_file" 2>/dev/null || true
    fi
}

# Analyze usage patterns and provide rightsizing recommendations
analyze_usage_patterns() {
    local analytics_file="/var/log/cloudworkstation-analytics.json"
    local recommendations_file="/var/log/cloudworkstation-rightsizing.json"
    
    # Skip analysis if no analytics data
    if [[ ! -f "$analytics_file" ]] || [[ ! -s "$analytics_file" ]]; then
        return 0
    fi
    
    # Analyze last 24 hours of data (assuming 2-minute intervals = ~720 samples)
    local sample_count=$(wc -l < "$analytics_file" | head -720)
    
    # Calculate averages and peaks using simple bash/awk
    local avg_cpu=$(grep '"utilization_percent"' "$analytics_file" | tail -720 | awk -F: '{sum+=$2} END {printf "%.1f", sum/NR}' 2>/dev/null || echo "0")
    local max_cpu=$(grep '"utilization_percent"' "$analytics_file" | tail -720 | awk -F: '{if($2>max) max=$2} END {printf "%.1f", max}' 2>/dev/null || echo "0")
    
    local avg_memory=$(grep '"utilization_percent":' "$analytics_file" | tail -720 | awk -F: '{sum+=$2} END {printf "%.1f", sum/NR}' 2>/dev/null || echo "0")
    local max_memory=$(grep '"utilization_percent":' "$analytics_file" | tail -720 | awk -F: '{if($2>max) max=$2} END {printf "%.1f", max}' 2>/dev/null || echo "0")
    
    local avg_gpu=$(grep '"utilization_percent":' "$analytics_file" | grep -A1 '"gpu":' | grep '"utilization_percent"' | tail -720 | awk -F: '{sum+=$2} END {printf "%.1f", sum/NR}' 2>/dev/null || echo "0")
    local max_gpu=$(grep '"utilization_percent":' "$analytics_file" | grep -A1 '"gpu":' | grep '"utilization_percent"' | tail -720 | awk -F: '{if($2>max) max=$2} END {printf "%.1f", max}' 2>/dev/null || echo "0")
    
    # Generate rightsizing recommendations
    local recommendation="optimal"
    local reason=""
    local suggested_size=""
    
    # CPU-based recommendations
    if command -v bc >/dev/null 2>&1; then
        if (( $(echo "$avg_cpu < 10" | bc -l) )); then
            recommendation="downsize"
            reason="Low average CPU utilization ($avg_cpu%)"
            suggested_size="S or XS"
        elif (( $(echo "$avg_cpu > 80" | bc -l) )); then
            recommendation="upsize"
            reason="High average CPU utilization ($avg_cpu%)"
            suggested_size="L or XL"
        elif (( $(echo "$max_cpu > 95" | bc -l) )); then
            recommendation="upsize"
            reason="CPU peaks at $max_cpu% indicating occasional bottlenecks"
            suggested_size="L"
        fi
        
        # Memory-based recommendations (override CPU if memory is the constraint)
        if (( $(echo "$avg_memory > 85" | bc -l) )); then
            recommendation="upsize_memory"
            reason="High average memory utilization ($avg_memory%)"
            suggested_size="memory-optimized (r5/r6g)"
        elif (( $(echo "$max_memory > 95" | bc -l) )); then
            recommendation="upsize_memory"
            reason="Memory peaks at $max_memory% indicating memory pressure"
            suggested_size="memory-optimized (r5/r6g)"
        fi
        
        # GPU-based recommendations
        if (( $(echo "$avg_gpu > 50" | bc -l) )); then
            recommendation="gpu_intensive"
            reason="High average GPU utilization ($avg_gpu%)"
            suggested_size="GPU-optimized (g4dn/g5g)"
        fi
    fi
    
    # Write recommendations
    cat > "$recommendations_file" << EOF
{
  "timestamp": "$(date -Iseconds)",
  "analysis_period_samples": $sample_count,
  "metrics": {
    "cpu": {
      "average_utilization": $avg_cpu,
      "peak_utilization": $max_cpu
    },
    "memory": {
      "average_utilization": $avg_memory,
      "peak_utilization": $max_memory
    },
    "gpu": {
      "average_utilization": $avg_gpu,
      "peak_utilization": $max_gpu
    }
  },
  "recommendation": {
    "action": "$recommendation",
    "reason": "$reason",
    "suggested_size": "$suggested_size",
    "confidence": "medium"
  }
}
EOF
    
    # Log the recommendation
    log "RIGHTSIZING: $recommendation - $reason (suggested: $suggested_size)"
    chown ubuntu:ubuntu "$recommendations_file" 2>/dev/null || true
}

# Check system activity (enhanced with analytics)
check_system_activity() {
    # Collect usage analytics for rightsizing (always collect when active)
    collect_usage_analytics
    
    # CPU load (1-minute average)
    CPU_LOAD=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | tr -d ',')
    log "CPU load: $CPU_LOAD"
    
    # Active users (excluding system users)
    USERS_LOGGED_IN=$(who | grep -v '^root' | wc -l)
    log "Users logged in: $USERS_LOGGED_IN"
    
    # GPU usage (if available)
    GPU_USAGE="0"
    if command -v nvidia-smi &> /dev/null; then
        GPU_USAGE=$(nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits 2>/dev/null | head -1 || echo "0")
        log "GPU usage: ${GPU_USAGE}%"
    fi
    
    # Run rightsizing analysis every hour (when check count is divisible by 30, assuming 2-min intervals)
    local check_count=$(get_instance_tag "CloudWorkstation:CheckCount" || echo "0")
    check_count=$((check_count + 1))
    set_instance_tag "CloudWorkstation:CheckCount" "$check_count"
    
    if (( check_count % 30 == 0 )); then
        log "Running periodic rightsizing analysis..."
        analyze_usage_patterns
    fi
    
    # Check if system is busy
    if command -v bc >/dev/null 2>&1; then
        if (( $(echo "$CPU_LOAD > 0.5" | bc -l) )) || [[ "$USERS_LOGGED_IN" -gt 0 ]] || [[ "$GPU_USAGE" -gt 10 ]]; then
            return 1  # System is busy
        else
            return 0  # System is idle
        fi
    else
        # Fallback without bc
        if [[ "$USERS_LOGGED_IN" -gt 0 ]]; then
            return 1  # System is busy
        else
            return 0  # System is idle
        fi
    fi
}

# AWS instance tag operations
set_instance_tag() {
    local key="$1"
    local value="$2"
    
    log "Setting tag $key=$value"
    aws ec2 create-tags --region "$REGION" --resources "$INSTANCE_ID" --tags "Key=$key,Value=$value" || {
        log "ERROR: Failed to set tag $key=$value"
        return 1
    }
}

get_instance_tag() {
    local key="$1"
    
    aws ec2 describe-tags --region "$REGION" --filters "Name=resource-id,Values=$INSTANCE_ID" "Name=key,Values=$key" \
        --query 'Tags[0].Value' --output text 2>/dev/null || echo ""
}

# Calculate idle duration in minutes
get_idle_duration() {
    local idle_since_tag=$(get_instance_tag "CloudWorkstation:IdleSince")
    
    if [[ -z "$idle_since_tag" || "$idle_since_tag" == "None" ]]; then
        echo "0"
        return
    fi
    
    local idle_since_epoch=$(date -d "$idle_since_tag" +%s 2>/dev/null || echo "0")
    local current_epoch=$(date +%s)
    local duration_seconds=$((current_epoch - idle_since_epoch))
    local duration_minutes=$((duration_seconds / 60))
    
    echo "$duration_minutes"
}

# Check hibernation support
check_hibernation_support() {
    local hibernation_enabled=$(aws ec2 describe-instances --region "$REGION" --instance-ids "$INSTANCE_ID" \
        --query 'Reservations[0].Instances[0].HibernationOptions.Configured' --output text 2>/dev/null || echo "false")
    
    [[ "$hibernation_enabled" == "true" ]]
}

# Hibernate or stop instance
hibernate_instance() {
    log "🛌 HIBERNATING instance after prolonged idle period"
    
    set_instance_tag "CloudWorkstation:IdleAction" "hibernating"
    
    if aws ec2 stop-instances --region "$REGION" --instance-ids "$INSTANCE_ID" --hibernate 2>/dev/null; then
        log "✅ Hibernation initiated successfully"
        set_instance_tag "CloudWorkstation:IdleAction" "hibernated"
    else
        log "❌ Hibernation failed, falling back to regular stop"
        if aws ec2 stop-instances --region "$REGION" --instance-ids "$INSTANCE_ID" 2>/dev/null; then
            log "✅ Stop initiated successfully"
            set_instance_tag "CloudWorkstation:IdleAction" "stopped"
        else
            log "❌ Failed to stop instance"
            set_instance_tag "CloudWorkstation:IdleAction" "stop_failed"
        fi
    fi
}

# Main function
main() {
    log "=== Starting idle check ==="
    
    # Check if idle detection is enabled
    if [[ "$ENABLED" != "true" ]]; then
        log "Idle detection is disabled (ENABLED=$ENABLED). Exiting."
        exit 0
    fi
    
    # Get metadata
    get_instance_metadata
    
    # Configure AWS CLI to use instance role
    export AWS_DEFAULT_REGION="$REGION"
    
    # Check system activity
    if check_system_activity; then
        log "System is IDLE"
        
        # Check if this is the first time we're detecting idle
        current_idle_status=$(get_instance_tag "CloudWorkstation:IdleStatus")
        if [[ "$current_idle_status" != "idle" ]]; then
            # First time idle - set timestamp
            set_instance_tag "CloudWorkstation:IdleStatus" "idle"
            set_instance_tag "CloudWorkstation:IdleSince" "$(date -Iseconds)"
            log "Tagged instance as idle (first detection)"
        else
            # Already idle - check duration and take action if needed
            idle_duration=$(get_idle_duration)
            log "Instance has been idle for $idle_duration minutes"
            
            if [[ $idle_duration -ge $HIBERNATE_THRESHOLD_MINUTES ]]; then
                log "Idle duration ($idle_duration min) exceeds hibernation threshold ($HIBERNATE_THRESHOLD_MINUTES min)"
                
                # Check hibernation support and take appropriate action
                if check_hibernation_support; then
                    log "Instance supports hibernation - hibernating now"
                    hibernate_instance
                else
                    log "Instance does not support hibernation - stopping instead"
                    hibernate_instance  # Function handles fallback to stop
                fi
            elif [[ $idle_duration -ge $IDLE_THRESHOLD_MINUTES ]]; then
                log "Idle duration ($idle_duration min) exceeds idle threshold ($IDLE_THRESHOLD_MINUTES min) but not hibernation threshold"
                log "Continuing to monitor..."
            fi
        fi
    else
        log "System is ACTIVE"
        
        # Clear idle status
        set_instance_tag "CloudWorkstation:IdleStatus" "active"
        set_instance_tag "CloudWorkstation:IdleSince" ""
        set_instance_tag "CloudWorkstation:IdleAction" ""
    fi
    
    log "=== Check complete ==="
}

# Run main function
main "$@"
EOF

# Make script executable
chmod +x /usr/local/bin/cloudworkstation-idle-check.sh

# Create config directory and initial config file
mkdir -p /etc/cloudworkstation
cat > /etc/cloudworkstation/idle-config << EOF
# CloudWorkstation Idle Detection Configuration
# This file can be modified to change idle detection behavior at runtime
ENABLED={{ENABLED}}
IDLE_THRESHOLD_MINUTES={{IDLE_THRESHOLD_MINUTES}}
HIBERNATE_THRESHOLD_MINUTES={{HIBERNATE_THRESHOLD_MINUTES}}
CHECK_INTERVAL_MINUTES={{CHECK_INTERVAL_MINUTES}}
EOF

# Create log directory
touch /var/log/cloudworkstation-idle.log
chown -f ubuntu:ubuntu /var/log/cloudworkstation-idle.log 2>/dev/null || true

# Create script to update cron job when interval changes
cat > /usr/local/bin/cloudworkstation-update-cron.sh << 'EOF'
#!/bin/bash
# Update cron job based on current configuration
source /etc/cloudworkstation/idle-config
cat > /etc/cron.d/cloudworkstation-idle << CRONEOF
# CloudWorkstation Idle Detection - runs every $CHECK_INTERVAL_MINUTES minutes
*/$CHECK_INTERVAL_MINUTES * * * * root /usr/local/bin/cloudworkstation-idle-check.sh >> /var/log/cloudworkstation-idle.log 2>&1
CRONEOF
echo "Updated cron job to run every $CHECK_INTERVAL_MINUTES minutes"
EOF

chmod +x /usr/local/bin/cloudworkstation-update-cron.sh

# Install initial cron job
/usr/local/bin/cloudworkstation-update-cron.sh`

	// Add package manager specific dependencies
	switch packageManager {
	case PackageManagerApt:
		return baseScript + `

# Install dependencies for idle detection
apt-get update && apt-get install -y bc awscli curl

# Initial run after 2 minutes to let system settle
(sleep 120 && /usr/local/bin/cloudworkstation-idle-check.sh) &

echo "Universal CloudWorkstation idle detection installed successfully" >> /var/log/cws-setup.log`
	
	case PackageManagerDnf:
		return baseScript + `

# Install dependencies for idle detection  
dnf install -y bc awscli curl

# Initial run after 2 minutes to let system settle
(sleep 120 && /usr/local/bin/cloudworkstation-idle-check.sh) &

echo "Universal CloudWorkstation idle detection installed successfully" >> /var/log/cws-setup.log`
	
	case PackageManagerConda:
		return baseScript + `

# Install dependencies for idle detection (conda environment should have AWS CLI)
apt-get update && apt-get install -y bc curl
pip install awscli

# Initial run after 2 minutes to let system settle
(sleep 120 && /usr/local/bin/cloudworkstation-idle-check.sh) &

echo "Universal CloudWorkstation idle detection installed successfully" >> /var/log/cws-setup.log`
	
	default:
		return baseScript + `

# Install dependencies for idle detection (fallback)
if command -v apt-get >/dev/null 2>&1; then
    apt-get update && apt-get install -y bc awscli curl
elif command -v dnf >/dev/null 2>&1; then
    dnf install -y bc awscli curl
fi

# Initial run after 2 minutes to let system settle  
(sleep 120 && /usr/local/bin/cloudworkstation-idle-check.sh) &

echo "Universal CloudWorkstation idle detection installed successfully" >> /var/log/cws-setup.log`
	}
}

// UpdateAMIRegistry queries the AMI registry and updates the resolver's AMI registry
// This enables automatic discovery of pre-built AMIs for templates
func (r *TemplateResolver) UpdateAMIRegistry(ctx context.Context, ssmClient interface{}) error {
	// Skip if no SSM client provided
	if ssmClient == nil {
		return nil
	}
	
	// For now, implement a mock discovery system that could be expanded
	// In a full implementation, this would integrate with pkg/ami.Registry
	
	// Clear existing AMI registry
	r.AMIRegistry = make(map[string]map[string]map[string]string)
	
	// Mock AMI discovery - in practice this would query SSM Parameter Store
	// and discover AMIs built with the AMI build system
	mockAMIRegistry := map[string]map[string]map[string]string{
		// Example: if "python-ml" template has pre-built AMIs
		"python-ml": {
			"us-east-1": {
				"x86_64": "ami-example-python-ml-x86",
				"arm64":  "ami-example-python-ml-arm64",
			},
			"us-west-2": {
				"x86_64": "ami-example-python-ml-x86-west",
				"arm64":  "ami-example-python-ml-arm64-west", 
			},
		},
		// Add more templates as they get AMIs built
	}
	
	r.AMIRegistry = mockAMIRegistry
	
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