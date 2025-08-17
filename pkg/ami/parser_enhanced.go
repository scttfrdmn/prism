package ami

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ParseEnhancedTemplate parses a template file with support for 0.3.0 format.
func (p *Parser) ParseEnhancedTemplate(templatePath string) (*Template, error) {
	// Check if file exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template file %q does not exist", templatePath)
	}

	// Read file
	data, err := os.ReadFile(templatePath)
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

// ValidateTemplate validates a template against the enhanced schema
func (p *Parser) ValidateTemplate(template *Template) error {
	return p.validateEnhancedTemplate(template)
}

// ParseTemplate parses a template from a string
func (p *Parser) ParseTemplate(content string) (*Template, error) {
	var template Template
	if err := yaml.Unmarshal([]byte(content), &template); err != nil {
		return nil, fmt.Errorf("failed to parse template YAML: %w", err)
	}

	if err := p.ValidateTemplate(&template); err != nil {
		return nil, fmt.Errorf("template validation failed: %w", err)
	}

	return &template, nil
}

// ParseTemplateFile parses a template from a file
func (p *Parser) ParseTemplateFile(templatePath string) (*Template, error) {
	return p.ParseEnhancedTemplate(templatePath)
}

// WriteTemplate writes a template to a writer
func (p *Parser) WriteTemplate(template *Template, writer io.Writer) error {
	data, err := yaml.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal template to YAML: %w", err)
	}

	_, err = writer.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	return nil
}

// ListTemplates returns a list of available templates (placeholder implementation)
func (p *Parser) ListTemplates() ([]string, error) {
	// TODO: Implement actual template listing logic
	// For now, return an empty list to fix compilation
	return []string{}, nil
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

	// Domain validation removed - not supported in current version

	// Resources validation removed - not supported in current version

	// Cost validation removed - not supported in current version

	// IdleDetection validation removed - not supported in current version

	// Repository validation removed - not supported in current version

	// Dependencies validation removed - not supported in current version

	// Documentation validation removed - not supported in current version

	// Validate build steps
	for i, step := range template.BuildSteps {
		if step.Name == "" {
			return fmt.Errorf("build step %d is missing a name", i+1)
		}
		if step.Script == "" {
			return fmt.Errorf("build step %q is missing a script", step.Name)
		}
	}

	// Use Validation field instead of ValidationTests
	for i, test := range template.Validation {
		if test.Name == "" {
			return fmt.Errorf("validation test %d is missing a name", i+1)
		}
		if test.Command == "" {
			return fmt.Errorf("validation test %q is missing a command", test.Name)
		}
	}

	return nil
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
