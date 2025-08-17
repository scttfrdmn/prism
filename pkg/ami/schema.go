// Package ami provides CloudWorkstation's AMI creation system.
package ami

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// templateSchema defines the JSON schema for template validation
const templateSchema = `
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["name", "base", "description", "build_steps"],
  "properties": {
    "name": {
      "type": "string",
      "minLength": 1,
      "description": "Template name"
    },
    "base": {
      "type": "string",
      "minLength": 1,
      "description": "Base image identifier"
    },
    "description": {
      "type": "string",
      "minLength": 1,
      "description": "Template description"
    },
    "build_steps": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["name", "script"],
        "properties": {
          "name": {
            "type": "string",
            "minLength": 1,
            "description": "Build step name"
          },
          "script": {
            "type": "string",
            "minLength": 1,
            "description": "Build script to execute"
          },
          "timeout_seconds": {
            "type": "integer",
            "minimum": 1,
            "default": 600,
            "description": "Timeout in seconds"
          }
        }
      }
    },
    "validation": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name", "command"],
        "properties": {
          "name": {
            "type": "string",
            "minLength": 1,
            "description": "Validation check name"
          },
          "command": {
            "type": "string",
            "minLength": 1,
            "description": "Validation command to execute"
          },
          "success": {
            "type": "boolean",
            "default": true,
            "description": "Whether command should exit successfully"
          },
          "contains": {
            "type": "string",
            "description": "String that output should contain"
          },
          "equals": {
            "type": "string",
            "description": "String that output should exactly match"
          }
        },
        "oneOf": [
          { "required": ["success"] },
          { "required": ["contains"] },
          { "required": ["equals"] }
        ]
      }
    },
    "tags": {
      "type": "object",
      "additionalProperties": {
        "type": "string"
      },
      "description": "Template tags"
    },
    "min_disk_size": {
      "type": "integer",
      "minimum": 1,
      "description": "Minimum disk size in GB"
    },
    "architecture": {
      "type": "string",
      "enum": ["x86_64", "arm64", ""],
      "default": "",
      "description": "Target architecture"
    }
  }
}
`

// SchemaValidator validates templates against the JSON schema
type SchemaValidator struct {
	schema *gojsonschema.Schema
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() (*SchemaValidator, error) {
	// Load schema
	schemaLoader := gojsonschema.NewStringLoader(templateSchema)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return nil, ValidationError("failed to load template schema", err)
	}

	return &SchemaValidator{
		schema: schema,
	}, nil
}

// Validate validates a template against the JSON schema
//
// This method converts the template to JSON and validates it against the schema.
//
// Parameters:
//   - template: The template to validate
//
// Returns:
//   - error: Validation error or nil if valid
func (v *SchemaValidator) Validate(template *Template) error {
	// Convert template to JSON
	data, err := json.Marshal(template)
	if err != nil {
		return ValidationError("failed to marshal template to JSON", err).
			WithContext("template_name", template.Name)
	}

	// Validate against schema
	documentLoader := gojsonschema.NewBytesLoader(data)
	result, err := v.schema.Validate(documentLoader)
	if err != nil {
		return ValidationError("schema validation failed", err).
			WithContext("template_name", template.Name)
	}

	// Check validation result
	if !result.Valid() {
		// Collect validation errors
		var errMsgs []string
		for _, err := range result.Errors() {
			errMsgs = append(errMsgs, fmt.Sprintf("- %s", err.String()))
		}

		return ValidationError(
			fmt.Sprintf("template schema validation failed with %d errors", len(result.Errors())),
			fmt.Errorf("%s", strings.Join(errMsgs, "\n")),
		).WithContext("template_name", template.Name)
	}

	return nil
}

// GetSchema returns the JSON schema document
func (v *SchemaValidator) GetSchema() ([]byte, error) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, []byte(templateSchema), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format schema: %w", err)
	}
	return prettyJSON.Bytes(), nil
}

// ValidateWithSchema enhances the Parser's ValidateTemplate method with schema validation
func (p *Parser) ValidateWithSchema(template *Template, validator *SchemaValidator) error {
	// First validate with the schema
	if err := validator.Validate(template); err != nil {
		return err
	}

	// Then perform additional validation
	return p.ValidateTemplate(template)
}
