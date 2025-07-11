package ami

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

// ParseEnhancedTemplate parses a template file with support for 0.3.0 format.
func (p *Parser) ParseEnhancedTemplate(templatePath string) (*Template, error) {
	// Check if file exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template file %q does not exist", templatePath)
	}

	// Read file
	data, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	// Parse YAML
	var template Template
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to parse template YAML: %w", err)
	}

	// Validate template against schema
	if err := p.validateEnhancedTemplate(&template); err != nil {
		return nil, fmt.Errorf("template validation failed: %w", err)
	}

	return &template, nil
}

// validateEnhancedTemplate validates a template against the enhanced schema.
func (p *Parser) validateEnhancedTemplate(template *Template) error {
	// Basic validation
	if template.Name == "" {
		return fmt.Errorf("template name is required")
	}

	if template.Base == "" {
		return fmt.Errorf("template base is required")
	}

	if len(template.BuildSteps) == 0 {
		return fmt.Errorf("at least one build step is required")
	}

	// Validate Domain field if present
	if template.Domain != nil {
		if err := p.validateDomain(template.Domain); err != nil {
			return fmt.Errorf("domain validation failed: %w", err)
		}
	}

	// Validate Resources field if present
	if template.Resources != nil {
		if err := p.validateResources(template.Resources); err != nil {
			return fmt.Errorf("resources validation failed: %w", err)
		}
	}

	// Validate Cost field if present
	if template.Cost != nil {
		if err := p.validateCost(template.Cost); err != nil {
			return fmt.Errorf("cost validation failed: %w", err)
		}
	}

	// Validate IdleDetection field if present
	if template.IdleDetection != nil {
		if err := p.validateIdleDetection(template.IdleDetection); err != nil {
			return fmt.Errorf("idle detection validation failed: %w", err)
		}
	}

	// Validate Repository field if present
	if template.Repository != nil {
		if err := p.validateRepository(template.Repository); err != nil {
			return fmt.Errorf("repository validation failed: %w", err)
		}
	}

	// Validate Dependencies field if present
	if len(template.Dependencies) > 0 {
		if err := p.validateDependencies(template.Dependencies); err != nil {
			return fmt.Errorf("dependencies validation failed: %w", err)
		}
	}

	// Validate Documentation field if present
	if template.Documentation != nil {
		if err := p.validateDocumentation(template.Documentation); err != nil {
			return fmt.Errorf("documentation validation failed: %w", err)
		}
	}

	// Validate build steps
	for i, step := range template.BuildSteps {
		if step.Name == "" {
			return fmt.Errorf("build step %d is missing a name", i+1)
		}
		if step.Script == "" {
			return fmt.Errorf("build step %q is missing a script", step.Name)
		}
	}

	// Validate validation tests
	for i, test := range template.ValidationTests {
		if test.Name == "" {
			return fmt.Errorf("validation test %d is missing a name", i+1)
		}
		if test.Command == "" && test.Script == "" {
			return fmt.Errorf("validation test %q is missing a command or script", test.Name)
		}
	}

	return nil
}

// validateDomain validates the Domain field.
func (p *Parser) validateDomain(domain *Domain) error {
	// Validate required fields
	if domain.Category == "" {
		return fmt.Errorf("domain category is required")
	}

	if domain.Subcategory == "" {
		return fmt.Errorf("domain subcategory is required")
	}

	if domain.WorkloadType == "" {
		return fmt.Errorf("domain workload type is required")
	}

	// Validate category values
	validCategories := []string{
		"life-sciences",
		"physical-sciences",
		"engineering",
		"computer-science",
		"social-sciences",
		"interdisciplinary",
	}

	if !containsString(validCategories, domain.Category) {
		return fmt.Errorf("invalid domain category %q, must be one of %v", domain.Category, validCategories)
	}

	// Validate workload type values
	validWorkloadTypes := []string{
		"interactive",
		"batch-processing",
		"gpu-intensive",
		"memory-intensive",
		"storage-intensive",
		"network-intensive",
	}

	if !containsString(validWorkloadTypes, domain.WorkloadType) {
		return fmt.Errorf("invalid workload type %q, must be one of %v", domain.WorkloadType, validWorkloadTypes)
	}

	return nil
}

// validateResources validates the Resources field.
func (p *Parser) validateResources(resources *Resources) error {
	// Validate default size
	if resources.DefaultSize == "" {
		return fmt.Errorf("default size is required")
	}

	// Check that default size exists in sizes
	if resources.Sizes != nil && len(resources.Sizes) > 0 {
		if _, ok := resources.Sizes[resources.DefaultSize]; !ok {
			return fmt.Errorf("default size %q not found in sizes", resources.DefaultSize)
		}
	} else {
		return fmt.Errorf("sizes must contain at least one size")
	}

	// Validate each size
	for name, size := range resources.Sizes {
		if size.InstanceType == "" {
			return fmt.Errorf("size %q is missing instance type", name)
		}

		if size.Architecture == "" {
			return fmt.Errorf("size %q is missing architecture", name)
		}

		if size.Architecture != "x86_64" && size.Architecture != "arm64" {
			return fmt.Errorf("size %q has invalid architecture %q, must be x86_64 or arm64", name, size.Architecture)
		}
	}

	return nil
}

// validateCost validates the Cost field.
func (p *Parser) validateCost(cost *Cost) error {
	// Only base daily cost is required
	if cost.BaseDailyUSD <= 0 {
		return fmt.Errorf("base daily cost must be positive")
	}

	return nil
}

// validateIdleDetection validates the IdleDetection field.
func (p *Parser) validateIdleDetection(idleDetection *IdleDetection) error {
	// Validate profile if present
	if idleDetection.Profile == "" {
		return fmt.Errorf("idle detection profile is required")
	}

	// Validate action if present
	if idleDetection.Action != "" {
		validActions := []string{
			"stop",
			"hibernate",
			"notify",
		}

		if !containsString(validActions, idleDetection.Action) {
			return fmt.Errorf("invalid idle detection action %q, must be one of %v", idleDetection.Action, validActions)
		}
	}

	// Validate thresholds
	if idleDetection.CPUThreshold < 0 || idleDetection.CPUThreshold > 100 {
		return fmt.Errorf("CPU threshold must be between 0 and 100")
	}

	if idleDetection.MemoryThreshold < 0 || idleDetection.MemoryThreshold > 100 {
		return fmt.Errorf("memory threshold must be between 0 and 100")
	}

	if idleDetection.NetworkThreshold < 0 {
		return fmt.Errorf("network threshold must be positive")
	}

	if idleDetection.DiskThreshold < 0 {
		return fmt.Errorf("disk threshold must be positive")
	}

	if idleDetection.GPUThreshold < 0 || idleDetection.GPUThreshold > 100 {
		return fmt.Errorf("GPU threshold must be between 0 and 100")
	}

	return nil
}

// validateRepository validates the Repository field.
func (p *Parser) validateRepository(repository *Repository) error {
	// Validate required fields
	if repository.Name == "" {
		return fmt.Errorf("repository name is required")
	}

	if repository.URL == "" {
		return fmt.Errorf("repository URL is required")
	}

	return nil
}

// validateDependencies validates the Dependencies field.
func (p *Parser) validateDependencies(dependencies []Dependency) error {
	// Validate each dependency
	for i, dep := range dependencies {
		if dep.Repository == "" {
			return fmt.Errorf("dependency %d is missing repository", i+1)
		}

		if dep.Template == "" {
			return fmt.Errorf("dependency %d is missing template", i+1)
		}

		if dep.Version == "" {
			return fmt.Errorf("dependency %d is missing version", i+1)
		}
	}

	return nil
}

// validateDocumentation validates the Documentation field.
func (p *Parser) validateDocumentation(documentation *Documentation) error {
	// No specific validation for now, all fields are optional
	return nil
}

// containsString checks if a string is in a slice of strings.
func containsString(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// ValidateTemplateFile validates a template file against the enhanced schema.
func (p *Parser) ValidateTemplateFile(templatePath string) error {
	// Parse template
	template, err := p.ParseEnhancedTemplate(templatePath)
	if err != nil {
		return err
	}

	// Additional validation to check that template can be built
	if template.Base != "" {
		// Check if base image is known
		if _, ok := p.BaseAMIs[template.Base]; !ok {
			// Not a fatal error, just warn
			fmt.Printf("Warning: Base image %q not found in known base AMIs\n", template.Base)
		}
	}

	fmt.Printf("Template %q validation successful\n", template.Name)
	return nil
}

// LoadEnhancedTemplateFromFile loads a template from a file with support for 0.3.0 format.
func (p *Parser) LoadEnhancedTemplateFromFile(templatePath string) (*Template, error) {
	// Get absolute path
	absPath, err := filepath.Abs(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Parse template
	template, err := p.ParseEnhancedTemplate(absPath)
	if err != nil {
		return nil, err
	}

	return template, nil
}