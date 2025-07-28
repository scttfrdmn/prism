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
	
	// Set defaults
	if template.PackageManager == "" {
		template.PackageManager = "auto"
	}
	
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
	validPMs := []string{"auto", "apt", "dnf", "conda", "spack", "ami"}
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

// NewPackageManagerStrategy creates a new package manager strategy with default rules
func NewPackageManagerStrategy() *PackageManagerStrategy {
	return &PackageManagerStrategy{
		Rules: PackageManagerRules{
			HPCIndicators: []string{
				// Scientific computing packages typically available in Spack
				"openmpi", "mpich", "fftw", "blas", "lapack", "scalapack",
				"petsc", "trilinos", "boost", "eigen", "armadillo",
				"paraview", "visit", "vtk", "hdf5", "netcdf",
			},
			PythonDataScienceIndicators: []string{
				// Python data science packages better in conda
				"numpy", "scipy", "pandas", "matplotlib", "seaborn",
				"scikit-learn", "tensorflow", "pytorch", "jupyter",
				"ipython", "bokeh", "plotly", "dask", "xarray",
			},
			RIndicators: []string{
				// R packages better in conda
				"r-base", "rstudio", "tidyverse", "ggplot2",
				"dplyr", "shiny", "rmarkdown", "knitr",
			},
			DefaultManager: PackageManagerApt,
		},
	}
}

// SelectPackageManager determines the best package manager for a template
func (s *PackageManagerStrategy) SelectPackageManager(template *Template) PackageManagerType {
	if template.PackageManager != "auto" {
		return PackageManagerType(template.PackageManager)
	}
	
	// Collect all package names from template
	allPackages := make([]string, 0)
	allPackages = append(allPackages, template.Packages.System...)
	allPackages = append(allPackages, template.Packages.Conda...)
	allPackages = append(allPackages, template.Packages.Spack...)
	allPackages = append(allPackages, template.Packages.Pip...)
	
	// Convert to lowercase for matching
	packageSet := make(map[string]bool)
	for _, pkg := range allPackages {
		packageSet[strings.ToLower(pkg)] = true
	}
	
	// Check for HPC indicators (highest priority)
	for _, indicator := range s.Rules.HPCIndicators {
		if packageSet[strings.ToLower(indicator)] {
			return PackageManagerSpack
		}
	}
	
	// Check for Python data science indicators
	for _, indicator := range s.Rules.PythonDataScienceIndicators {
		if packageSet[strings.ToLower(indicator)] {
			return PackageManagerConda
		}
	}
	
	// Check for R indicators
	for _, indicator := range s.Rules.RIndicators {
		if packageSet[strings.ToLower(indicator)] {
			return PackageManagerConda
		}
	}
	
	// Default to system package manager
	return s.Rules.DefaultManager
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