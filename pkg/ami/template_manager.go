// Package ami provides CloudWorkstation's AMI creation system.
//
// This file implements the template management interface for CloudWorkstation's
// AMI creation system. It provides functionality for importing, exporting,
// creating, modifying, and sharing templates across different sources and formats.
//
// The template management system implements CloudWorkstation's "Progressive Disclosure"
// principle by providing simple interfaces for common operations while enabling
// advanced capabilities for power users.
//
// Key Components:
//   - TemplateManager: Central template management coordinator
//   - Template operations: Import, export, create, modify, validate, and share
//   - Template metadata tracking: Source, validation status, and modification time
//   - Format conversion: YAML, JSON, and other interchange formats
package ami

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// TemplateMetadata contains additional information about a template
type TemplateMetadata struct {
	SourcePath       string    `json:"source_path,omitempty"`
	SourceURL        string    `json:"source_url,omitempty"`
	LastModified     time.Time `json:"last_modified"`
	ValidationStatus string    `json:"validation_status"`
	IsBuiltIn        bool      `json:"is_built_in"`
	Description      string    `json:"description,omitempty"`
	Author           string    `json:"author,omitempty"`
	Version          string    `json:"version,omitempty"`
}

// Clock provides time-related functionality
type Clock interface {
	Now() time.Time
}

// RealClock is the default clock implementation
type RealClock struct{}

// Now returns the current time
func (r *RealClock) Now() time.Time {
	return time.Now()
}

// TemplateManager handles template management operations
type TemplateManager struct {
	Parser            *Parser
	Registry          *Registry
	TemplateDirectory string
	Templates         map[string]*Template
	TemplateMetadata  map[string]TemplateMetadata
	HTTPClient        *http.Client
	SchemaValidator   *SchemaValidator
	PublicDirectory   string // Directory for publicly shared templates
	clock             Clock   // Clock for time operations
}

// TemplateBuilder is a builder for creating and modifying templates
type TemplateBuilder struct {
	manager     *TemplateManager
	template    *Template
	hasModified bool
}

// TemplateManagerImportOptions configures template import behavior for the template manager
type TemplateManagerImportOptions struct {
	Validate      bool
	Force         bool
	OverwriteName string
}

// TemplateExportOptions configures template export behavior
type TemplateExportOptions struct {
	Format      string // yaml, json
	PrettyPrint bool
}

// Template management error types defined in errors.go

// NewTemplateManager creates a new template manager
//
// The template manager serves as a central coordinator for all template operations,
// providing functionality to import, export, create, modify, and share templates.
//
// Parameters:
//   - parser: YAML template parser instance
//   - registry: AMI registry for template sharing
//   - templateDir: Directory to store template files
//
// Returns:
//   - *TemplateManager: Initialized template manager
//
// Example:
//
//	parser := ami.NewParser(baseAMIs)
//	registry := ami.NewRegistry(ssmClient, "/cloudworkstation/ami")
//	manager := ami.NewTemplateManager(parser, registry, "./templates")
func NewTemplateManager(parser *Parser, registry *Registry, templateDir string) *TemplateManager {
	// Create schema validator
	validator, err := NewSchemaValidator()
	if err != nil {
		// Log error but continue without schema validation
		fmt.Printf("Warning: Failed to initialize schema validator: %v\n", err)
	}
	return &TemplateManager{
		Parser:            parser,
		Registry:          registry,
		TemplateDirectory: templateDir,
		Templates:         make(map[string]*Template),
		TemplateMetadata:  make(map[string]TemplateMetadata),
		HTTPClient:        &http.Client{Timeout: 30 * time.Second},
		SchemaValidator:   validator,
		clock:             &RealClock{},
	}
}

// ImportFromFile imports a template from a local file
//
// This method imports a template from a local YAML file, validates it,
// and adds it to the template cache with metadata.
//
// Parameters:
//   - filePath: Path to the template file
//   - options: Import options (can be nil for defaults)
//
// Returns:
//   - *Template: The imported template
//   - error: Any import errors
//
// Example:
//
//	template, err := manager.ImportFromFile("./templates/r-research.yaml", nil)
//	if err != nil {
//	    log.Fatalf("Failed to import template: %v", err)
//	}
func (m *TemplateManager) ImportFromFile(filePath string, options *TemplateManagerImportOptions) (*Template, error) {
	if options == nil {
		options = &TemplateManagerImportOptions{
			Validate: true,
			Force:    false,
		}
	}

	// Parse template file
	template, err := m.Parser.ParseTemplateFile(filePath)
	if err != nil {
		return nil, TemplateImportError("failed to parse template file", err).
			WithContext("file_path", filePath)
	}

	// If overwrite name is specified, update template name
	if options.OverwriteName != "" {
		template.Name = options.OverwriteName
	}

	// Check if template with same name already exists
	if existing, exists := m.Templates[template.Name]; exists && !options.Force {
		return nil, TemplateImportError(
			fmt.Sprintf("template with name '%s' already exists", template.Name),
			nil,
		).WithContext("existing_template", existing.Name)
	}

	// Validate the template if requested
	if options.Validate {
		// First validate with schema if available
		if m.SchemaValidator != nil {
			if err := m.SchemaValidator.Validate(template); err != nil {
				return nil, err
			}
		}

		// Then perform additional validation with parser
		if err := m.Parser.ValidateTemplate(template); err != nil {
			return nil, TemplateImportError("template validation failed", err).
				WithContext("template_name", template.Name)
		}
	}

	// Create metadata
	metadata := TemplateMetadata{
		SourcePath:       filePath,
		LastModified:     time.Now(),
		ValidationStatus: "valid",
		IsBuiltIn:        false,
		Description:      template.Description,
	}

	// Store template and metadata
	m.Templates[template.Name] = template
	m.TemplateMetadata[template.Name] = metadata

	return template, nil
}

// ImportFromURL imports a template from a URL
//
// This method downloads a template from a URL, validates it, and adds it to the template cache.
//
// Parameters:
//   - url: URL to the template file
//   - options: Import options (can be nil for defaults)
//
// Returns:
//   - *Template: The imported template
//   - error: Any import errors
//
// Example:
//
//	template, err := manager.ImportFromURL("https://example.com/templates/python-ml.yaml", nil)
func (m *TemplateManager) ImportFromURL(url string, options *TemplateManagerImportOptions) (*Template, error) {
	if options == nil {
		options = &TemplateManagerImportOptions{
			Validate: true,
			Force:    false,
		}
	}

	// Download template
	resp, err := m.HTTPClient.Get(url)
	if err != nil {
		return nil, TemplateImportError("failed to download template", err).
			WithContext("url", url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, TemplateImportError(
			fmt.Sprintf("failed to download template: HTTP %d", resp.StatusCode),
			nil,
		).WithContext("url", url).WithContext("status_code", fmt.Sprintf("%d", resp.StatusCode))
	}

	// Read response body
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, TemplateImportError("failed to read template content", err).
			WithContext("url", url)
	}

	// Parse template
	template, err := m.Parser.ParseTemplate(string(content))
	if err != nil {
		return nil, TemplateImportError("failed to parse template", err).
			WithContext("url", url)
	}

	// If overwrite name is specified, update template name
	if options.OverwriteName != "" {
		template.Name = options.OverwriteName
	}

	// Check if template with same name already exists
	if _, exists := m.Templates[template.Name]; exists && !options.Force {
		return nil, TemplateImportError(
			fmt.Sprintf("template with name '%s' already exists", template.Name),
			nil,
		).WithContext("template_name", template.Name)
	}

	// Validate the template if requested
	if options.Validate {
		// First validate with schema if available
		if m.SchemaValidator != nil {
			if err := m.SchemaValidator.Validate(template); err != nil {
				return nil, err
			}
		}

		// Then perform additional validation with parser
		if err := m.Parser.ValidateTemplate(template); err != nil {
			return nil, TemplateImportError("template validation failed", err).
				WithContext("template_name", template.Name).
				WithContext("url", url)
		}
	}

	// Create metadata
	metadata := TemplateMetadata{
		SourceURL:        url,
		LastModified:     time.Now(),
		ValidationStatus: "valid",
		IsBuiltIn:        false,
		Description:      template.Description,
	}

	// Store template and metadata
	m.Templates[template.Name] = template
	m.TemplateMetadata[template.Name] = metadata

	return template, nil
}

// ImportFromGitHub imports a template from a GitHub repository
//
// This method downloads a template from a GitHub repository, validates it, and adds
// it to the template cache. It supports both public and private repositories.
//
// Parameters:
//   - repo: GitHub repository (username/repo)
//   - path: Path to template file in the repository
//   - ref: Git reference (branch, tag, commit)
//   - options: Import options (can be nil for defaults)
//
// Returns:
//   - *Template: The imported template
//   - error: Any import errors
//
// Example:
//
//	template, err := manager.ImportFromGitHub(
//	    "cloudworkstation/templates",
//	    "r-research.yaml",
//	    "main",
//	    nil,
//	)
func (m *TemplateManager) ImportFromGitHub(repo, path, ref string, options *TemplateManagerImportOptions) (*Template, error) {
	// Construct raw GitHub URL
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", repo, ref, path)
	return m.ImportFromURL(url, options)
}

// ExportToFile exports a template to a file
//
// This method exports a template to a file in the specified format.
//
// Parameters:
//   - templateName: Name of the template to export
//   - filePath: Path to export the template to
//   - options: Export options (can be nil for defaults)
//
// Returns:
//   - error: Any export errors
//
// Example:
//
//	err := manager.ExportToFile("r-research", "./exported/r-research.yaml", nil)
func (m *TemplateManager) ExportToFile(templateName, filePath string, options *TemplateExportOptions) error {
	if options == nil {
		options = &TemplateExportOptions{
			Format:      "yaml",
			PrettyPrint: true,
		}
	}

	// Get template
	template, err := m.GetTemplate(templateName)
	if err != nil {
		return TemplateExportError(fmt.Sprintf("template '%s' not found", templateName), err).
			WithContext("template_name", templateName)
	}

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return TemplateExportError("failed to create export file", err).
			WithContext("file_path", filePath)
	}
	defer file.Close()

	// Export based on format
	switch options.Format {
	case "json":
		return m.exportJSON(template, file, options.PrettyPrint)
	case "yaml", "yml", "":
		return m.exportYAML(template, file)
	default:
		return TemplateExportError(fmt.Sprintf("unsupported export format: %s", options.Format), nil).
			WithContext("format", options.Format).
			WithContext("supported_formats", "yaml, json")
	}
}

// ExportToJSON exports a template to JSON format
//
// This method serializes a template to JSON format and returns the result.
//
// Parameters:
//   - templateName: Name of the template to export
//   - prettyPrint: Whether to format the JSON with indentation
//
// Returns:
//   - []byte: The JSON data
//   - error: Any export errors
//
// Example:
//
//	data, err := manager.ExportToJSON("python-ml", true)
//	if err != nil {
//	    log.Fatalf("Failed to export template: %v", err)
//	}
func (m *TemplateManager) ExportToJSON(templateName string, prettyPrint bool) ([]byte, error) {
	// Get template
	template, err := m.GetTemplate(templateName)
	if err != nil {
		return nil, TemplateExportError(fmt.Sprintf("template '%s' not found", templateName), err).
			WithContext("template_name", templateName)
	}

	var buf bytes.Buffer
	if err := m.exportJSON(template, &buf, prettyPrint); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// exportJSON is a helper function to export a template as JSON
func (m *TemplateManager) exportJSON(template *Template, writer io.Writer, prettyPrint bool) error {
	var data []byte
	var err error

	if prettyPrint {
		data, err = json.MarshalIndent(template, "", "  ")
	} else {
		data, err = json.Marshal(template)
	}

	if err != nil {
		return TemplateExportError("failed to marshal template to JSON", err).
			WithContext("template_name", template.Name)
	}

	_, err = writer.Write(data)
	if err != nil {
		return TemplateExportError("failed to write JSON data", err).
			WithContext("template_name", template.Name)
	}

	return nil
}

// ExportToYAML exports a template to YAML format
//
// This method serializes a template to YAML format and returns the result.
//
// Parameters:
//   - templateName: Name of the template to export
//
// Returns:
//   - []byte: The YAML data
//   - error: Any export errors
//
// Example:
//
//	data, err := manager.ExportToYAML("bioinformatics")
//	if err != nil {
//	    log.Fatalf("Failed to export template: %v", err)
//	}
func (m *TemplateManager) ExportToYAML(templateName string) ([]byte, error) {
	// Get template
	template, err := m.GetTemplate(templateName)
	if err != nil {
		return nil, TemplateExportError(fmt.Sprintf("template '%s' not found", templateName), err).
			WithContext("template_name", templateName)
	}

	var buf bytes.Buffer
	if err := m.exportYAML(template, &buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// exportYAML is a helper function to export a template as YAML
func (m *TemplateManager) exportYAML(template *Template, writer io.Writer) error {
	if err := m.Parser.WriteTemplate(template, writer); err != nil {
		return TemplateExportError("failed to export template to YAML", err).
			WithContext("template_name", template.Name)
	}
	return nil
}

// CreateTemplate creates a new template with the builder pattern
//
// This method starts the template creation process using a builder pattern,
// allowing for fluent definition of the template properties.
//
// Parameters:
//   - name: Template name
//   - description: Template description
//
// Returns:
//   - *TemplateBuilder: Builder for the template
//
// Example:
//
//	template, err := manager.CreateTemplate("custom-python", "Custom Python environment").
//	    WithBase("ubuntu-22.04").
//	    AddBuildStep("install-python", "apt-get update && apt-get install -y python3").
//	    AddValidation("check-python", "python3 --version", true, "", "").
//	    Build()
func (m *TemplateManager) CreateTemplate(name, description string) *TemplateBuilder {
	template := &Template{
		Name:        name,
		Description: description,
		BuildSteps:  []BuildStep{},
		Validation:  []Validation{},
		Tags:        make(map[string]string),
	}

	return &TemplateBuilder{
		manager:     m,
		template:    template,
		hasModified: true,
	}
}

// ModifyTemplate creates a builder for modifying an existing template
//
// This method starts the template modification process using a builder pattern.
//
// Parameters:
//   - name: Name of the template to modify
//
// Returns:
//   - *TemplateBuilder: Builder for the template
//   - error: Any template retrieval errors
//
// Example:
//
//	builder, err := manager.ModifyTemplate("r-research")
//	if err != nil {
//	    log.Fatalf("Failed to modify template: %v", err)
//	}
//
//	modified := builder.
//	    AddBuildStep("install-tidyverse", "R -e 'install.packages(\"tidyverse\")'").
//	    Build()
func (m *TemplateManager) ModifyTemplate(name string) (*TemplateBuilder, error) {
	template, err := m.GetTemplate(name)
	if err != nil {
		return nil, TemplateManagementError(fmt.Sprintf("template '%s' not found for modification", name), err).
			WithContext("template_name", name)
	}

	// Create a deep copy of the template to avoid modifying the original
	copied := *template

	// Deep copy build steps
	copied.BuildSteps = make([]BuildStep, len(template.BuildSteps))
	copy(copied.BuildSteps, template.BuildSteps)

	// Deep copy validation checks
	copied.Validation = make([]Validation, len(template.Validation))
	copy(copied.Validation, template.Validation)

	// Deep copy tags
	copied.Tags = make(map[string]string)
	for k, v := range template.Tags {
		copied.Tags[k] = v
	}

	return &TemplateBuilder{
		manager:     m,
		template:    &copied,
		hasModified: false,
	}, nil
}

// ValidateTemplate validates a template against schema and requirements
//
// This method performs comprehensive validation of a template, including schema
// validation, base AMI validation, and additional custom checks.
//
// Parameters:
//   - templateName: Name of the template to validate
//
// Returns:
//   - error: Validation error or nil if valid
//
// Example:
//
//	err := manager.ValidateTemplate("r-research")
//	if err != nil {
//	    log.Fatalf("Template validation failed: %v", err)
//	}
func (m *TemplateManager) ValidateTemplate(templateName string) error {
	template, err := m.GetTemplate(templateName)
	if err != nil {
		return ValidationError(fmt.Sprintf("template '%s' not found", templateName), err).
			WithContext("template_name", templateName)
	}

	// First validate with schema if available
	if m.SchemaValidator != nil {
		if err := m.SchemaValidator.Validate(template); err != nil {
			return err
		}
	}

	// Then perform additional validation with parser
	if err := m.Parser.ValidateTemplate(template); err != nil {
		return ValidationError("template validation failed", err).
			WithContext("template_name", templateName)
	}

	// Update template metadata
	if metadata, exists := m.TemplateMetadata[templateName]; exists {
		metadata.ValidationStatus = "valid"
		metadata.LastModified = time.Now()
		m.TemplateMetadata[templateName] = metadata
	}

	return nil
}

// ShareTemplate shares a template via the registry
//
// This method publishes a template to the AMI registry, making it available
// for other CloudWorkstation instances.
//
// Parameters:
//   - templateName: Name of the template to share
//   - ctx: Context for cancellation and timeouts
//
// Returns:
//   - error: Any sharing errors
//
// Example:
//
//	err := manager.ShareTemplate("custom-ml", context.Background())
//	if err != nil {
//	    log.Fatalf("Failed to share template: %v", err)
//	}
func (m *TemplateManager) ShareTemplate(templateName string, ctx context.Context) error {
	if m.Registry == nil {
		return TemplateManagementError("registry not configured for template sharing", nil).
			WithContext("template_name", templateName)
	}

	template, err := m.GetTemplate(templateName)
	if err != nil {
		return TemplateManagementError(fmt.Sprintf("template '%s' not found", templateName), err).
			WithContext("template_name", templateName)
	}

	// Validate template before sharing
	if err := m.Parser.ValidateTemplate(template); err != nil {
		return ValidationError("cannot share invalid template", err).
			WithContext("template_name", templateName)
	}

	// Serialize template to YAML
	var buf bytes.Buffer
	if err := m.Parser.WriteTemplate(template, &buf); err != nil {
		return TemplateManagementError("failed to serialize template", err).
			WithContext("template_name", templateName)
	}

	// Share template using registry
	if m.Registry != nil {
		// Get metadata for the template
		metadata := map[string]string{
			"description":   template.Description,
			"version":      "1.0.0", // Default version
			"publisher":    "CloudWorkstation",
			"architecture": template.Architecture,
		}
		
		// Add tags if available
		for k, v := range template.Tags {
			metadata[k] = v
		}
		
		// Publish template to registry
		err = m.Registry.PublishTemplate(ctx, templateName, buf.String(), "yaml", metadata)
		if err != nil {
			return TemplateManagementError("failed to publish template to registry", err).
				WithContext("template_name", templateName)
		}
	} else {
		return TemplateManagementError("registry not configured for template sharing", nil).
			WithContext("template_name", templateName)
	}

	return nil
}

// GetTemplate retrieves a template by name
//
// This method gets a template from the cache or loads it from disk if not cached.
//
// Parameters:
//   - name: Template name
//
// Returns:
//   - *Template: The template
//   - error: Any retrieval errors
func (m *TemplateManager) GetTemplate(name string) (*Template, error) {
	// Check if template is in cache
	if template, exists := m.Templates[name]; exists {
		return template, nil
	}

	// Template not in cache, try to load from disk
	if m.TemplateDirectory != "" {
		filePath := filepath.Join(m.TemplateDirectory, name+".yaml")
		if _, err := os.Stat(filePath); err == nil {
			// File exists, try to load it
			template, err := m.ImportFromFile(filePath, &TemplateManagerImportOptions{
				Validate: false, // Don't validate during get
				Force:    true,  // Force load even if it already exists
			})
			if err != nil {
				return nil, TemplateManagementError("failed to load template from disk", err).
					WithContext("template_name", name).
					WithContext("file_path", filePath)
			}
			return template, nil
		}
	}

	return nil, TemplateManagementError(fmt.Sprintf("template '%s' not found", name), nil).
		WithContext("template_name", name)
}

// ListTemplates lists all available templates
//
// This method returns all templates in the cache and optionally scans the
// template directory for additional templates.
//
// Parameters:
//   - includeDirectory: Whether to scan the template directory
//
// Returns:
//   - map[string]*Template: Map of template name to template
//   - error: Any listing errors
func (m *TemplateManager) ListTemplates(includeDirectory bool) (map[string]*Template, error) {
	result := make(map[string]*Template)

	// Add templates from cache
	for name, template := range m.Templates {
		result[name] = template
	}

	// Scan directory if requested
	if includeDirectory && m.TemplateDirectory != "" {
		templateNames, err := m.Parser.ListTemplates()
		if err != nil {
			return result, TemplateManagementError("failed to list templates from directory", err).
				WithContext("directory", m.TemplateDirectory)
		}

		// Add templates from directory to result, skipping those already in cache
		for _, templateName := range templateNames {
			if _, exists := result[templateName]; !exists {
				// Parse template from file
				template, err := m.Parser.ParseTemplateFile(filepath.Join(m.TemplateDirectory, templateName+".yaml"))
				if err != nil {
					continue // Skip failed templates
				}
				result[templateName] = template
			}
		}
	}

	return result, nil
}

// DeleteTemplate deletes a template from cache and optionally from disk
//
// This method removes a template from the manager's cache and optionally
// deletes the template file from disk.
//
// Parameters:
//   - name: Template name
//   - deleteFile: Whether to delete the template file from disk
//
// Returns:
//   - error: Any deletion errors
func (m *TemplateManager) DeleteTemplate(name string, deleteFile bool) error {
	// Check if template exists
	_, err := m.GetTemplate(name)
	if err != nil {
		return TemplateManagementError(fmt.Sprintf("template '%s' not found", name), err).
			WithContext("template_name", name)
	}

	// Delete from cache
	delete(m.Templates, name)
	delete(m.TemplateMetadata, name)

	// Delete file if requested
	if deleteFile && m.TemplateDirectory != "" {
		filePath := filepath.Join(m.TemplateDirectory, name+".yaml")
		if _, err := os.Stat(filePath); err == nil {
			// File exists, delete it
			if err := os.Remove(filePath); err != nil {
				return TemplateManagementError("failed to delete template file", err).
					WithContext("template_name", name).
					WithContext("file_path", filePath)
			}
		}
	}

	return nil
}

// WithBase sets the base image for the template
func (b *TemplateBuilder) WithBase(base string) *TemplateBuilder {
	b.template.Base = base
	b.hasModified = true
	return b
}

// WithArchitecture sets the architecture for the template
func (b *TemplateBuilder) WithArchitecture(arch string) *TemplateBuilder {
	b.template.Architecture = arch
	b.hasModified = true
	return b
}

// WithMinDiskSize sets the minimum disk size for the template
func (b *TemplateBuilder) WithMinDiskSize(sizeGB int) *TemplateBuilder {
	b.template.MinDiskSize = sizeGB
	b.hasModified = true
	return b
}

// WithTag adds a tag to the template
func (b *TemplateBuilder) WithTag(key, value string) *TemplateBuilder {
	if b.template.Tags == nil {
		b.template.Tags = make(map[string]string)
	}
	b.template.Tags[key] = value
	b.hasModified = true
	return b
}

// AddBuildStep adds a build step to the template
func (b *TemplateBuilder) AddBuildStep(name, script string) *TemplateBuilder {
	b.template.BuildSteps = append(b.template.BuildSteps, BuildStep{
		Name:   name,
		Script: script,
	})
	b.hasModified = true
	return b
}

// AddBuildStepWithTimeout adds a build step with a custom timeout to the template
func (b *TemplateBuilder) AddBuildStepWithTimeout(name, script string, timeoutSeconds int) *TemplateBuilder {
	b.template.BuildSteps = append(b.template.BuildSteps, BuildStep{
		Name:           name,
		Script:         script,
		TimeoutSeconds: timeoutSeconds,
	})
	b.hasModified = true
	return b
}

// AddValidation adds a validation check to the template
func (b *TemplateBuilder) AddValidation(name, command string, success bool, contains, equals string) *TemplateBuilder {
	b.template.Validation = append(b.template.Validation, Validation{
		Name:     name,
		Command:  command,
		Success:  success,
		Contains: contains,
		Equals:   equals,
	})
	b.hasModified = true
	return b
}

// Build finalizes and validates the template
//
// This method validates and stores the template in the manager.
//
// Returns:
//   - *Template: The built template
//   - error: Any validation errors
func (b *TemplateBuilder) Build() (*Template, error) {
	// First validate with schema if available
	if b.manager.SchemaValidator != nil {
		if err := b.manager.SchemaValidator.Validate(b.template); err != nil {
			return nil, err
		}
	}

	// Then perform additional validation with parser
	if err := b.manager.Parser.ValidateTemplate(b.template); err != nil {
		return nil, ValidationError("template validation failed", err).
			WithContext("template_name", b.template.Name)
	}

	// Check if template with same name already exists and we're creating a new one
	if _, exists := b.manager.Templates[b.template.Name]; exists && !b.hasModified {
		return nil, TemplateManagementError(
			fmt.Sprintf("template with name '%s' already exists", b.template.Name),
			nil,
		).WithContext("template_name", b.template.Name)
	}

	// Create metadata
	metadata := TemplateMetadata{
		LastModified:     time.Now(),
		ValidationStatus: "valid",
		IsBuiltIn:        false,
		Description:      b.template.Description,
	}

	// Store template and metadata
	b.manager.Templates[b.template.Name] = b.template
	b.manager.TemplateMetadata[b.template.Name] = metadata

	return b.template, nil
}