package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// NewTemplateParser creates a new template parser
func NewTemplateParser() *TemplateParser {
	return &TemplateParser{
		BaseAMIs: getDefaultBaseAMIs(),
		Strategy: NewPackageManagerStrategy(),
	}
}

// ParseTemplate parses a template from YAML content
func (p *TemplateParser) ParseTemplate(content []byte) (*Template, error) {
	var template Template
	if err := yaml.Unmarshal(content, &template); err != nil {
		return nil, fmt.Errorf("failed to parse template YAML: %w", err)
	}
	
	// Templates must specify their package manager explicitly
	
	// Set default service enable state
	for i := range template.Services {
		if template.Services[i].Enable == false && template.Services[i].Port > 0 {
			template.Services[i].Enable = true // Default to enabled if port specified
		}
	}
	
	// Set default user shell
	for i := range template.Users {
		if template.Users[i].Shell == "" {
			template.Users[i].Shell = "/bin/bash"
		}
	}
	
	// Validate template
	if err := p.ValidateTemplate(&template); err != nil {
		return nil, err
	}
	
	return &template, nil
}

// ParseTemplateFile parses a template from a YAML file
func (p *TemplateParser) ParseTemplateFile(filename string) (*Template, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file %s: %w", filename, err)
	}
	
	template, err := p.ParseTemplate(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template file %s: %w", filename, err)
	}
	
	// Set template name from filename if not specified
	if template.Name == "" {
		baseName := filepath.Base(filename)
		template.Name = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	}
	
	return template, nil
}

// ValidateTemplate validates a template for correctness
func (p *TemplateParser) ValidateTemplate(template *Template) error {
	// Required fields
	if template.Name == "" {
		return &TemplateValidationError{Field: "name", Message: "template name is required"}
	}
	
	if template.Description == "" {
		return &TemplateValidationError{Field: "description", Message: "template description is required"}
	}
	
	if template.Base == "" {
		return &TemplateValidationError{Field: "base", Message: "base OS is required"}
	}
	
	// Validate base OS is supported (skip for AMI-based templates)
	if template.Base != "ami-based" {
		if _, exists := p.BaseAMIs[template.Base]; !exists {
			return &TemplateValidationError{
				Field:   "base",
				Message: fmt.Sprintf("unsupported base OS: %s", template.Base),
			}
		}
	}
	
	// Validate package manager
	validPMs := []string{"apt", "dnf", "conda", "spack", "ami"}
	if template.PackageManager != "" {
		valid := false
		for _, pm := range validPMs {
			if template.PackageManager == pm {
				valid = true
				break
			}
		}
		if !valid {
			return &TemplateValidationError{
				Field:   "package_manager",
				Message: fmt.Sprintf("unsupported package manager: %s (valid: %v)", template.PackageManager, validPMs),
			}
		}
	}
	
	// Validate services
	for i, service := range template.Services {
		if service.Name == "" {
			return &TemplateValidationError{
				Field:   fmt.Sprintf("services[%d].name", i),
				Message: "service name is required",
			}
		}
		
		if service.Port < 0 || service.Port > 65535 {
			return &TemplateValidationError{
				Field:   fmt.Sprintf("services[%d].port", i),
				Message: "service port must be between 0 and 65535",
			}
		}
	}
	
	// Validate users
	for i, user := range template.Users {
		if user.Name == "" {
			return &TemplateValidationError{
				Field:   fmt.Sprintf("users[%d].name", i),
				Message: "user name is required",
			}
		}
		
		// Validate user name format (basic check)
		if strings.Contains(user.Name, " ") || strings.Contains(user.Name, ":") {
			return &TemplateValidationError{
				Field:   fmt.Sprintf("users[%d].name", i),
				Message: "user name cannot contain spaces or colons",
			}
		}
	}
	
	// Validate ports
	for i, port := range template.InstanceDefaults.Ports {
		if port < 1 || port > 65535 {
			return &TemplateValidationError{
				Field:   fmt.Sprintf("instance_defaults.ports[%d]", i),
				Message: "port must be between 1 and 65535",
			}
		}
	}
	
	return nil
}

// NewPackageManagerStrategy creates a new package manager strategy
func NewPackageManagerStrategy() *PackageManagerStrategy {
	return &PackageManagerStrategy{}
}

// SelectPackageManager returns the template's specified package manager
func (s *PackageManagerStrategy) SelectPackageManager(template *Template) PackageManagerType {
	// Templates must specify their package manager explicitly
	return PackageManagerType(template.PackageManager)
}

// getDefaultBaseAMIs returns the default base AMI mappings
func getDefaultBaseAMIs() map[string]map[string]map[string]string {
	return map[string]map[string]map[string]string{
		"ubuntu-22.04": {
			"us-east-1": {
				"x86_64": "ami-02029c87fa31fb148",
				"arm64":  "ami-050499786ebf55a6a",
			},
			"us-east-2": {
				"x86_64": "ami-0b05d988257befbbe",
				"arm64":  "ami-010755a3881216bba",
			},
			"us-west-1": {
				"x86_64": "ami-043b59f1d11f8f189",
				"arm64":  "ami-0d3e8bea392f79ebb",
			},
			"us-west-2": {
				"x86_64": "ami-016d360a89daa11ba",
				"arm64":  "ami-09f6c9efbf93542be",
			},
		},
		"ubuntu-22.04-server-lts": {
			"us-east-1": {
				"x86_64": "ami-02029c87fa31fb148",
				"arm64":  "ami-050499786ebf55a6a",
			},
			"us-east-2": {
				"x86_64": "ami-0b05d988257befbbe",
				"arm64":  "ami-010755a3881216bba",
			},
			"us-west-1": {
				"x86_64": "ami-043b59f1d11f8f189",
				"arm64":  "ami-0d3e8bea392f79ebb",
			},
			"us-west-2": {
				"x86_64": "ami-016d360a89daa11ba",
				"arm64":  "ami-09f6c9efbf93542be",
			},
		},
		"ubuntu-20.04": {
			"us-east-1": {
				"x86_64": "ami-0d70546e43a941d70",
				"arm64":  "ami-0c5a8b0c5d3c6d5a1",
			},
			// Add more regions as needed
		},
	}
}

// TemplateRegistry implementation
func NewTemplateRegistry(templateDirs []string) *TemplateRegistry {
	return &TemplateRegistry{
		TemplateDirs: templateDirs,
		Templates:    make(map[string]*Template),
	}
}

// ScanTemplates scans template directories and loads templates
func (r *TemplateRegistry) ScanTemplates() error {
	parser := NewTemplateParser()
	r.Templates = make(map[string]*Template)
	
	for _, dir := range r.TemplateDirs {
		// Skip directories that don't exist
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			// Skip directories
			if info.IsDir() {
				return nil
			}
			
			// Only process YAML files
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".yml" && ext != ".yaml" {
				return nil
			}
			
			// Parse template
			template, err := parser.ParseTemplateFile(path)
			if err != nil {
				return fmt.Errorf("failed to parse template %s: %w", path, err)
			}
			
			// Store template by name
			r.Templates[template.Name] = template
			
			return nil
		})
		
		if err != nil {
			return fmt.Errorf("failed to scan template directory %s: %w", dir, err)
		}
	}
	
	// After loading all templates, resolve inheritance
	if err := r.ResolveInheritance(); err != nil {
		return fmt.Errorf("failed to resolve template inheritance: %w", err)
	}
	
	r.LastScan = time.Now()
	return nil
}

// GetTemplate retrieves a template by name
func (r *TemplateRegistry) GetTemplate(name string) (*Template, error) {
	template, exists := r.Templates[name]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", name)
	}
	
	return template, nil
}

// ListTemplates returns all available templates
func (r *TemplateRegistry) ListTemplates() map[string]*Template {
	return r.Templates
}
// ResolveInheritance resolves template inheritance by merging parent templates
func (r *TemplateRegistry) ResolveInheritance() error {
	// First pass: collect all templates that need inheritance resolution
	templatesWithInheritance := make([]*Template, 0)
	for _, template := range r.Templates {
		if len(template.Inherits) > 0 {
			templatesWithInheritance = append(templatesWithInheritance, template)
		}
	}
	
	// Second pass: resolve inheritance for each template
	for _, template := range templatesWithInheritance {
		resolved, err := r.resolveTemplateInheritance(template)
		if err != nil {
			return fmt.Errorf("failed to resolve inheritance for template %s: %w", template.Name, err)
		}
		
		// Replace original template with resolved one
		r.Templates[template.Name] = resolved
	}
	
	return nil
}

// resolveTemplateInheritance resolves inheritance for a single template
func (r *TemplateRegistry) resolveTemplateInheritance(template *Template) (*Template, error) {
	// Create a new template to merge into
	merged := &Template{
		Name:           template.Name,
		Description:    template.Description,
		Base:          template.Base,
		Version:        template.Version,
		Maintainer:     template.Maintainer,
		LastUpdated:    template.LastUpdated,
		Tags:           make(map[string]string),
		Packages:       PackageDefinitions{},
		Users:          []UserConfig{},
		Services:       []ServiceConfig{},
		InstanceDefaults: InstanceDefaults{
			Ports: []int{},
			EstimatedCostPerHour: make(map[string]float64),
		},
	}
	
	// Process inheritance chain (parents first, then child)
	for _, parentName := range template.Inherits {
		parent, exists := r.Templates[parentName]
		if !exists {
			return nil, fmt.Errorf("parent template not found: %s", parentName)
		}
		
		// Recursively resolve parent if it has inheritance
		if len(parent.Inherits) > 0 {
			resolvedParent, err := r.resolveTemplateInheritance(parent)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve parent template %s: %w", parentName, err)
			}
			parent = resolvedParent
		}
		
		// Merge parent into merged template
		r.mergeTemplate(merged, parent)
	}
	
	// Finally merge the current template (child overrides parent)
	r.mergeTemplate(merged, template)
	
	return merged, nil
}

// mergeTemplate merges source template into target template
func (r *TemplateRegistry) mergeTemplate(target, source *Template) {
	// Merge package manager (child overrides parent)
	if source.PackageManager != "" {
		target.PackageManager = source.PackageManager
	}
	
	// Merge packages (append, don't override)
	target.Packages.System = append(target.Packages.System, source.Packages.System...)
	target.Packages.Conda = append(target.Packages.Conda, source.Packages.Conda...)
	target.Packages.Spack = append(target.Packages.Spack, source.Packages.Spack...)
	target.Packages.Pip = append(target.Packages.Pip, source.Packages.Pip...)
	
	// Merge users (append)
	target.Users = append(target.Users, source.Users...)
	
	// Merge services (append)
	target.Services = append(target.Services, source.Services...)
	
	// Merge tags (child overrides parent)
	if target.Tags == nil {
		target.Tags = make(map[string]string)
	}
	for k, v := range source.Tags {
		target.Tags[k] = v
	}
	
	// Merge AMI config (child overrides parent)
	if source.AMIConfig.AMIs != nil {
		target.AMIConfig = source.AMIConfig
	}
	
	// Merge post-install script (append)
	if source.PostInstall != "" {
		if target.PostInstall != "" {
			target.PostInstall += "\n\n# --- From parent template ---\n" + source.PostInstall
		} else {
			target.PostInstall = source.PostInstall
		}
	}
	
	// Merge instance defaults
	if source.InstanceDefaults.Type != "" {
		target.InstanceDefaults.Type = source.InstanceDefaults.Type
	}
	
	// Merge ports (append and deduplicate)
	portMap := make(map[int]bool)
	for _, port := range target.InstanceDefaults.Ports {
		portMap[port] = true
	}
	for _, port := range source.InstanceDefaults.Ports {
		if !portMap[port] {
			target.InstanceDefaults.Ports = append(target.InstanceDefaults.Ports, port)
			portMap[port] = true
		}
	}
	
	// Merge cost estimates (child overrides parent)
	if target.InstanceDefaults.EstimatedCostPerHour == nil {
		target.InstanceDefaults.EstimatedCostPerHour = make(map[string]float64)
	}
	for k, v := range source.InstanceDefaults.EstimatedCostPerHour {
		target.InstanceDefaults.EstimatedCostPerHour[k] = v
	}
}
