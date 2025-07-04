// Package ami provides CloudWorkstation's AMI creation system.
package ami

import (
	"fmt"
)

// ImportFromTemplate imports a template from an existing Template struct
//
// This method adds an existing template to the manager's cache with optional validation.
//
// Parameters:
//   - template: The template to import
//   - options: Import options (can be nil for defaults)
//
// Returns:
//   - *Template: The imported template
//   - error: Any import errors
//
// Example:
//
//	template, err := manager.ImportFromTemplate(existingTemplate, nil)
func (m *TemplateManager) ImportFromTemplate(template *Template, options *TemplateImportOptions) (*Template, error) {
	if options == nil {
		options = &TemplateImportOptions{
			Validate: true,
			Force:    false,
		}
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