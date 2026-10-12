// Package templates provides template parameterization and variable substitution.
package templates

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// ParameterProcessor handles template parameter substitution
type ParameterProcessor struct {
	template   *Template
	parameters TemplateParameterValues
	variables  map[string]interface{} // Changed from map[string]string to preserve type information
}

// NewParameterProcessor creates a parameter processor for a template
func NewParameterProcessor(tmpl *Template, userParams TemplateParameterValues) *ParameterProcessor {
	processor := &ParameterProcessor{
		template:   tmpl,
		parameters: make(TemplateParameterValues),
		variables:  make(map[string]interface{}),
	}

	// Start with template-level variables
	for k, v := range tmpl.Variables {
		processor.variables[k] = v
	}

	// Apply user parameters (with defaults)
	processor.applyParameters(userParams)

	return processor
}

// applyParameters applies user parameters and defaults to the processor
func (pp *ParameterProcessor) applyParameters(userParams TemplateParameterValues) {
	// First, set defaults for all defined parameters
	for name, param := range pp.template.Parameters {
		if param.Default != nil {
			pp.parameters[name] = param.Default
		}
	}

	// Then override with user-provided values
	for name, value := range userParams {
		if _, exists := pp.template.Parameters[name]; exists {
			pp.parameters[name] = value
		}
	}

	// Convert parameters to variables for substitution
	// Preserve type information for proper handling in Go templates (boolean, int, string)
	for name, value := range pp.parameters {
		// Keep typed values for template engine - this enables {{if .bool_param}} to work correctly
		// Boolean false as string "false" would be truthy, but boolean false is falsy
		pp.variables[name] = value
	}
}

// ProcessTemplate applies parameter substitution to the entire template
func (pp *ParameterProcessor) ProcessTemplate() (*Template, error) {
	// Create a copy of the template
	processedTemplate := *pp.template

	// Process text fields that support variable substitution
	var err error

	processedTemplate.Description, err = pp.processString(processedTemplate.Description)
	if err != nil {
		return nil, fmt.Errorf("processing description: %w", err)
	}

	processedTemplate.LongDescription, err = pp.processString(processedTemplate.LongDescription)
	if err != nil {
		return nil, fmt.Errorf("processing long description: %w", err)
	}

	// Process package lists
	if err := pp.processPackageDefinitions(&processedTemplate.Packages); err != nil {
		return nil, fmt.Errorf("processing packages: %w", err)
	}

	// Process services
	if err := pp.processServices(processedTemplate.Services); err != nil {
		return nil, fmt.Errorf("processing services: %w", err)
	}

	// Process post-install script
	processedTemplate.PostInstall, err = pp.processString(processedTemplate.PostInstall)
	if err != nil {
		return nil, fmt.Errorf("processing post-install script: %w", err)
	}

	return &processedTemplate, nil
}

// processString performs variable substitution on a string
func (pp *ParameterProcessor) processString(input string) (string, error) {
	if input == "" {
		return input, nil
	}

	// Use Go's text/template for variable substitution
	tmpl, err := template.New("template").Funcs(template.FuncMap{
		"eq": func(a, b interface{}) bool {
			return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
		},
	}).Parse(input)
	if err != nil {
		// Issue #448: Template parsing failed - return error instead of silent fallback
		return "", fmt.Errorf("template parsing failed: %w (template preview: %.100s...)", err, input)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, pp.variables)
	if err != nil {
		// Issue #448: Template execution failed - return error with helpful context
		// Common causes: undefined variables, incorrect template syntax with conditionals
		availableVars := strings.Join(pp.getVariableNames(), ", ")

		if strings.Contains(err.Error(), "can't evaluate field") {
			return "", fmt.Errorf("template execution failed - undefined variable: %w\nAvailable variables: [%s]\nTemplate preview (first 200 chars): %.200s", err, availableVars, input)
		}

		return "", fmt.Errorf("template execution failed: %w\nAvailable variables: [%s]\nTemplate preview (first 200 chars): %.200s", err, availableVars, input)
	}

	return buf.String(), nil
}

// getVariableNames returns a list of available variable names for debugging
func (pp *ParameterProcessor) getVariableNames() []string {
	names := make([]string, 0, len(pp.variables))
	for name := range pp.variables {
		names = append(names, name)
	}
	return names
}

// simpleSubstitution performs basic {{variable}} substitution
func (pp *ParameterProcessor) simpleSubstitution(input string) string {
	result := input
	for name, value := range pp.variables {
		// Convert interface{} to string for substitution
		valueStr := fmt.Sprintf("%v", value)

		placeholder := fmt.Sprintf("{{.%s}}", name)
		result = strings.ReplaceAll(result, placeholder, valueStr)

		// Also support {{variable}} format
		simplePlaceholder := fmt.Sprintf("{{%s}}", name)
		result = strings.ReplaceAll(result, simplePlaceholder, valueStr)
	}
	return result
}

// processPackageDefinitions applies parameter substitution to package lists
func (pp *ParameterProcessor) processPackageDefinitions(packages *PackageDefinitions) error {
	var err error

	packages.System, err = pp.processStringSlice(packages.System)
	if err != nil {
		return fmt.Errorf("processing system packages: %w", err)
	}

	packages.Conda, err = pp.processStringSlice(packages.Conda)
	if err != nil {
		return fmt.Errorf("processing conda packages: %w", err)
	}

	packages.Spack, err = pp.processStringSlice(packages.Spack)
	if err != nil {
		return fmt.Errorf("processing spack packages: %w", err)
	}

	packages.Pip, err = pp.processStringSlice(packages.Pip)
	if err != nil {
		return fmt.Errorf("processing pip packages: %w", err)
	}

	return nil
}

// processStringSlice applies parameter substitution to a slice of strings
func (pp *ParameterProcessor) processStringSlice(input []string) ([]string, error) {
	if len(input) == 0 {
		return input, nil
	}

	result := make([]string, len(input))
	for i, str := range input {
		processed, err := pp.processString(str)
		if err != nil {
			return nil, err
		}
		result[i] = processed
	}

	return result, nil
}

// processServices applies parameter substitution to service configurations
func (pp *ParameterProcessor) processServices(services []ServiceConfig) error {
	for i := range services {
		service := &services[i]

		var err error
		service.Name, err = pp.processString(service.Name)
		if err != nil {
			return fmt.Errorf("processing service name: %w", err)
		}

		// Process service configuration
		for j, config := range service.Config {
			service.Config[j], err = pp.processString(config)
			if err != nil {
				return fmt.Errorf("processing service config: %w", err)
			}
		}
	}

	return nil
}

// GetParameterValue returns the processed value for a parameter
func (pp *ParameterProcessor) GetParameterValue(name string) (interface{}, bool) {
	value, exists := pp.parameters[name]
	return value, exists
}

// GetVariable returns the processed variable value (preserving type)
func (pp *ParameterProcessor) GetVariable(name string) (interface{}, bool) {
	value, exists := pp.variables[name]
	return value, exists
}

// ValidateParameters validates that all parameters meet their constraints
func (pp *ParameterProcessor) ValidateParameters() []TemplateValidationError {
	// Debug: Print parameter types
	for name, value := range pp.parameters {
		fmt.Printf("DEBUG: Parameter %s = %v (type: %T)\n", name, value, value)
	}
	return pp.parameters.Validate(pp.template.Parameters)
}

// ParameterHelper provides utilities for working with template parameters
type ParameterHelper struct{}

// NewParameterHelper creates a new parameter helper
func NewParameterHelper() *ParameterHelper {
	return &ParameterHelper{}
}

// ParseParameterFlag parses parameter flags in the format --param name=value
func (ph *ParameterHelper) ParseParameterFlag(flag string) (string, interface{}, error) {
	// Expected format: --param python_version=3.11
	if !strings.Contains(flag, "=") {
		return "", nil, fmt.Errorf("parameter must be in format name=value")
	}

	parts := strings.SplitN(flag, "=", 2)
	name := strings.TrimSpace(parts[0])
	valueStr := strings.TrimSpace(parts[1])

	// Try to parse as different types
	value := ph.parseParameterValue(valueStr)
	return name, value, nil
}

// parseParameterValue attempts to parse a parameter value to appropriate type
func (ph *ParameterHelper) parseParameterValue(valueStr string) interface{} {
	// Try boolean
	if valueStr == "true" {
		return true
	}
	if valueStr == "false" {
		return false
	}

	// Try integer
	if intVal, err := strconv.Atoi(valueStr); err == nil {
		return intVal
	}

	// Try float
	if floatVal, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return floatVal
	}

	// Default to string
	return valueStr
}

// GetParametersFromTemplate extracts parameters from a template for CLI help
func (ph *ParameterHelper) GetParametersFromTemplate(tmpl *Template) []string {
	if len(tmpl.Parameters) == 0 {
		return nil
	}

	var params []string
	for name, param := range tmpl.Parameters {
		description := param.Description
		if param.Default != nil {
			description += fmt.Sprintf(" (default: %v)", param.Default)
		}
		if param.Required {
			description += " [required]"
		}

		params = append(params, fmt.Sprintf("  %s: %s", name, description))
	}

	return params
}

// ExpandTemplateVariables expands variables in any string using the current parameter context
func (pp *ParameterProcessor) ExpandTemplateVariables(input string) string {
	if input == "" {
		return input
	}

	// Use simple substitution for general variable expansion
	return pp.simpleSubstitution(input)
}

// VariablePattern regex for finding template variables
var VariablePattern = regexp.MustCompile(`\{\{\.?(\w+)\}\}`)

// FindVariables finds all variable references in a string
func FindVariables(input string) []string {
	matches := VariablePattern.FindAllStringSubmatch(input, -1)
	variables := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) > 1 {
			variables = append(variables, match[1])
		}
	}

	return variables
}
