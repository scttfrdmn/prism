// Package templates provides CloudWorkstation's unified template system.
package templates

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidationLevel represents the severity of a validation issue
type ValidationLevel string

const (
	ValidationError   ValidationLevel = "error"   // Must fix before template can be used
	ValidationWarning ValidationLevel = "warning" // Should fix but not blocking
	ValidationInfo    ValidationLevel = "info"    // Suggestions for improvement
)

// ValidationResult represents a single validation finding
type ValidationResult struct {
	Level   ValidationLevel
	Field   string
	Message string
	Line    int // Optional line number in template file
}

// ValidationReport contains all validation results for a template
type ValidationReport struct {
	TemplateName string
	Valid        bool
	Results      []ValidationResult
	ErrorCount   int
	WarningCount int
	InfoCount    int
}

// ComprehensiveValidator performs comprehensive template validation
type ComprehensiveValidator struct {
	registry *TemplateRegistry
	rules    []ValidationRule
}

// ValidationRule defines a validation check
type ValidationRule interface {
	Name() string
	Validate(template *Template) []ValidationResult
}

// NewComprehensiveValidator creates a new validator with all rules
func NewComprehensiveValidator(registry *TemplateRegistry) *ComprehensiveValidator {
	return &ComprehensiveValidator{
		registry: registry,
		rules: []ValidationRule{
			&RequiredFieldsRule{},
			&PackageManagerRule{},
			&ServicePortRule{},
			&UserConfigRule{},
			&InheritanceRule{registry: registry},
			&ParameterRule{},
			&SecurityRule{},
			&CostOptimizationRule{},
			&PerformanceRule{},
			&BestPracticesRule{},
		},
	}
}

// ValidateTemplate validates a single template
func (v *ComprehensiveValidator) ValidateTemplate(template *Template) *ValidationReport {
	report := &ValidationReport{
		TemplateName: template.Name,
		Valid:        true,
		Results:      []ValidationResult{},
	}
	
	// Run all validation rules
	for _, rule := range v.rules {
		results := rule.Validate(template)
		report.Results = append(report.Results, results...)
	}
	
	// Count results by level
	for _, result := range report.Results {
		switch result.Level {
		case ValidationError:
			report.ErrorCount++
			report.Valid = false
		case ValidationWarning:
			report.WarningCount++
		case ValidationInfo:
			report.InfoCount++
		}
	}
	
	return report
}

// ValidateAll validates all templates in the registry
func (v *ComprehensiveValidator) ValidateAll() map[string]*ValidationReport {
	reports := make(map[string]*ValidationReport)
	
	for name, template := range v.registry.Templates {
		reports[name] = v.ValidateTemplate(template)
	}
	
	return reports
}

// RequiredFieldsRule validates required template fields
type RequiredFieldsRule struct{}

func (r *RequiredFieldsRule) Name() string { return "required_fields" }

func (r *RequiredFieldsRule) Validate(template *Template) []ValidationResult {
	var results []ValidationResult
	
	if template.Name == "" {
		results = append(results, ValidationResult{
			Level:   ValidationError,
			Field:   "name",
			Message: "Template name is required",
		})
	}
	
	if template.Description == "" {
		results = append(results, ValidationResult{
			Level:   ValidationError,
			Field:   "description",
			Message: "Template description is required",
		})
	}
	
	if template.Base == "" && len(template.Inherits) == 0 {
		results = append(results, ValidationResult{
			Level:   ValidationError,
			Field:   "base",
			Message: "Template must have a base OS or inherit from another template",
		})
	}
	
	return results
}

// PackageManagerRule validates package manager configuration
type PackageManagerRule struct{}

func (r *PackageManagerRule) Name() string { return "package_manager" }

func (r *PackageManagerRule) Validate(template *Template) []ValidationResult {
	var results []ValidationResult
	
	// Check package manager is valid
	validManagers := map[string]bool{
		"apt": true, "dnf": true, "conda": true, 
		"spack": true, "ami": true, "pip": true,
	}
	
	if template.PackageManager != "" && !validManagers[template.PackageManager] {
		results = append(results, ValidationResult{
			Level:   ValidationError,
			Field:   "package_manager",
			Message: fmt.Sprintf("Invalid package manager: %s", template.PackageManager),
		})
	}
	
	// Check package lists match package manager
	if template.PackageManager == "conda" && len(template.Packages.System) > 0 {
		results = append(results, ValidationResult{
			Level:   ValidationWarning,
			Field:   "packages.system",
			Message: "System packages specified but package manager is conda",
		})
	}
	
	return results
}

// ServicePortRule validates service configurations
type ServicePortRule struct{}

func (r *ServicePortRule) Name() string { return "service_ports" }

func (r *ServicePortRule) Validate(template *Template) []ValidationResult {
	var results []ValidationResult
	
	portMap := make(map[int]string)
	
	for _, service := range template.Services {
		if service.Port > 0 {
			if existing, ok := portMap[service.Port]; ok {
				results = append(results, ValidationResult{
					Level:   ValidationError,
					Field:   "services",
					Message: fmt.Sprintf("Port %d conflict between services %s and %s", 
						service.Port, existing, service.Name),
				})
			} else {
				portMap[service.Port] = service.Name
			}
			
			// Check for well-known ports
			if service.Port < 1024 {
				results = append(results, ValidationResult{
					Level:   ValidationWarning,
					Field:   "services." + service.Name,
					Message: fmt.Sprintf("Service uses privileged port %d", service.Port),
				})
			}
		}
	}
	
	return results
}

// UserConfigRule validates user configuration
type UserConfigRule struct{}

func (r *UserConfigRule) Name() string { return "user_config" }

func (r *UserConfigRule) Validate(template *Template) []ValidationResult {
	var results []ValidationResult
	
	userMap := make(map[string]bool)
	
	for _, user := range template.Users {
		if user.Name == "" {
			results = append(results, ValidationResult{
				Level:   ValidationError,
				Field:   "users",
				Message: "User name cannot be empty",
			})
		} else if userMap[user.Name] {
			results = append(results, ValidationResult{
				Level:   ValidationError,
				Field:   "users",
				Message: fmt.Sprintf("Duplicate user: %s", user.Name),
			})
		} else {
			userMap[user.Name] = true
			
			// Validate username format (skip if it contains template variables)
			if !strings.Contains(user.Name, "{{") && !isValidUsername(user.Name) {
				results = append(results, ValidationResult{
					Level:   ValidationError,
					Field:   "users." + user.Name,
					Message: "Invalid username format (must be lowercase, start with letter)",
				})
			}
		}
	}
	
	return results
}

// InheritanceRule validates template inheritance
type InheritanceRule struct {
	registry *TemplateRegistry
}

func (r *InheritanceRule) Name() string { return "inheritance" }

func (r *InheritanceRule) Validate(template *Template) []ValidationResult {
	var results []ValidationResult
	
	for _, parent := range template.Inherits {
		if r.registry != nil {
			if _, exists := r.registry.Templates[parent]; !exists {
				results = append(results, ValidationResult{
					Level:   ValidationError,
					Field:   "inherits",
					Message: fmt.Sprintf("Parent template not found: %s", parent),
				})
			}
		}
	}
	
	// Check for circular inheritance
	if r.registry != nil && len(template.Inherits) > 0 {
		if hasCircularInheritance(template.Name, template, r.registry, []string{}) {
			results = append(results, ValidationResult{
				Level:   ValidationError,
				Field:   "inherits",
				Message: "Circular inheritance detected",
			})
		}
	}
	
	return results
}

// ParameterRule validates template parameters
type ParameterRule struct{}

func (r *ParameterRule) Name() string { return "parameters" }

func (r *ParameterRule) Validate(template *Template) []ValidationResult {
	var results []ValidationResult
	
	for name, param := range template.Parameters {
		// Check parameter type
		validTypes := map[string]bool{
			"string": true, "int": true, "bool": true, "choice": true,
		}
		
		if !validTypes[param.Type] {
			results = append(results, ValidationResult{
				Level:   ValidationError,
				Field:   fmt.Sprintf("parameters.%s", name),
				Message: fmt.Sprintf("Invalid parameter type: %s", param.Type),
			})
		}
		
		// Check choice parameters have choices
		if param.Type == "choice" && len(param.Choices) == 0 {
			results = append(results, ValidationResult{
				Level:   ValidationError,
				Field:   fmt.Sprintf("parameters.%s", name),
				Message: "Choice parameter must have choices defined",
			})
		}
		
		// Check default value matches type
		if param.Default != nil {
			if err := validateParameterValue(name, param.Default, param); err != nil {
				results = append(results, ValidationResult{
					Level:   ValidationError,
					Field:   fmt.Sprintf("parameters.%s", name),
					Message: err.Message,
				})
			}
		}
	}
	
	return results
}

// SecurityRule checks for security best practices
type SecurityRule struct{}

func (r *SecurityRule) Name() string { return "security" }

func (r *SecurityRule) Validate(template *Template) []ValidationResult {
	var results []ValidationResult
	
	// Check for hardcoded passwords
	if strings.Contains(template.PostInstall, "password") || 
	   strings.Contains(template.UserData, "password") {
		results = append(results, ValidationResult{
			Level:   ValidationWarning,
			Field:   "scripts",
			Message: "Possible hardcoded password detected",
		})
	}
	
	// Check for open ports
	for _, service := range template.Services {
		if service.Port == 3389 || service.Port == 5900 {
			results = append(results, ValidationResult{
				Level:   ValidationWarning,
				Field:   "services",
				Message: fmt.Sprintf("Remote desktop port %d is exposed", service.Port),
			})
		}
	}
	
	return results
}

// CostOptimizationRule checks for cost optimization opportunities
type CostOptimizationRule struct{}

func (r *CostOptimizationRule) Name() string { return "cost_optimization" }

func (r *CostOptimizationRule) Validate(template *Template) []ValidationResult {
	var results []ValidationResult
	
	// Check idle detection settings
	if template.IdleDetection == nil || !template.IdleDetection.Enabled {
		results = append(results, ValidationResult{
			Level:   ValidationInfo,
			Field:   "idle_detection",
			Message: "Consider enabling idle detection for cost optimization",
		})
	}
	
	// Check instance defaults
	if strings.Contains(template.InstanceDefaults.Type, "xlarge") {
		results = append(results, ValidationResult{
			Level:   ValidationWarning,
			Field:   "instance_defaults.type",
			Message: "Default instance type is expensive, consider smaller defaults",
		})
	}
	
	return results
}

// PerformanceRule checks for performance issues
type PerformanceRule struct{}

func (r *PerformanceRule) Name() string { return "performance" }

func (r *PerformanceRule) Validate(template *Template) []ValidationResult {
	var results []ValidationResult
	
	// Check for too many packages
	totalPackages := len(template.Packages.System) + len(template.Packages.Conda) + 
	                len(template.Packages.Pip) + len(template.Packages.Spack)
	
	if totalPackages > 100 {
		results = append(results, ValidationResult{
			Level:   ValidationWarning,
			Field:   "packages",
			Message: fmt.Sprintf("Large number of packages (%d) may slow launch time", totalPackages),
		})
	}
	
	// Check estimated launch time
	if template.EstimatedLaunchTime > 10 {
		results = append(results, ValidationResult{
			Level:   ValidationInfo,
			Field:   "estimated_launch_time",
			Message: "Consider using AMI-based template for faster launches",
		})
	}
	
	return results
}

// BestPracticesRule checks for general best practices
type BestPracticesRule struct{}

func (r *BestPracticesRule) Name() string { return "best_practices" }

func (r *BestPracticesRule) Validate(template *Template) []ValidationResult {
	var results []ValidationResult
	
	// Check for metadata
	if template.Version == "" {
		results = append(results, ValidationResult{
			Level:   ValidationInfo,
			Field:   "version",
			Message: "Consider adding version field for tracking",
		})
	}
	
	if template.Maintainer == "" {
		results = append(results, ValidationResult{
			Level:   ValidationInfo,
			Field:   "maintainer",
			Message: "Consider adding maintainer field for support",
		})
	}
	
	// Check for documentation
	if template.LongDescription == "" {
		results = append(results, ValidationResult{
			Level:   ValidationInfo,
			Field:   "long_description",
			Message: "Consider adding detailed description for users",
		})
	}
	
	if len(template.LearningResources) == 0 {
		results = append(results, ValidationResult{
			Level:   ValidationInfo,
			Field:   "learning_resources",
			Message: "Consider adding learning resources for users",
		})
	}
	
	return results
}

// Helper functions

func isValidUsername(username string) bool {
	// Username must start with lowercase letter, contain only lowercase letters, digits, hyphens, underscores
	pattern := `^[a-z][a-z0-9_-]*$`
	matched, _ := regexp.MatchString(pattern, username)
	return matched && len(username) <= 32
}

func hasCircularInheritance(target string, current *Template, registry *TemplateRegistry, visited []string) bool {
	// Check if we've seen this template before
	for _, v := range visited {
		if v == current.Name {
			return true
		}
	}
	
	visited = append(visited, current.Name)
	
	for _, parent := range current.Inherits {
		if parent == target {
			return true
		}
		
		if parentTemplate, exists := registry.Templates[parent]; exists {
			if hasCircularInheritance(target, parentTemplate, registry, visited) {
				return true
			}
		}
	}
	
	return false
}