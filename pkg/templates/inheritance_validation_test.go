package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInheritanceValidator_FunctionalValidation tests inheritance validation scenarios that would block user launches
func TestInheritanceValidator_FunctionalValidation(t *testing.T) {
	parser := NewTemplateParser()

	tests := []struct {
		name          string
		template      *Template
		expectError   bool
		expectedField string
		description   string
	}{
		{
			name: "self_reference_blocks_launch",
			template: &Template{
				Name:           "python-ml",
				Inherits:       []string{"python-ml"}, // Self-reference
				PackageManager: "conda",
			},
			expectError:   true,
			expectedField: "inherits",
			description:   "User creates template that inherits from itself - should fail validation",
		},
		{
			name: "empty_parent_name_blocks_launch",
			template: &Template{
				Name:           "advanced-ml",
				Inherits:       []string{"base-python", ""}, // Empty parent name
				PackageManager: "conda",
			},
			expectError:   true,
			expectedField: "inherits[1]",
			description:   "User specifies empty parent template name - should fail validation",
		},
		{
			name: "whitespace_only_parent_blocks_launch",
			template: &Template{
				Name:           "data-science",
				Inherits:       []string{"python-base", "   "}, // Whitespace only
				PackageManager: "conda",
			},
			expectError:   true,
			expectedField: "inherits[1]",
			description:   "User specifies whitespace-only parent name - should fail validation",
		},
		{
			name: "multiple_inheritance_with_self_reference",
			template: &Template{
				Name:           "gpu-ml",
				Inherits:       []string{"cuda-base", "gpu-ml", "python-base"}, // Self in middle
				PackageManager: "conda",
			},
			expectError:   true,
			expectedField: "inherits",
			description:   "Complex inheritance chain with self-reference should fail",
		},
		{
			name: "valid_single_inheritance",
			template: &Template{
				Name:           "ml-workstation",
				Inherits:       []string{"python-base"},
				PackageManager: "conda",
			},
			expectError: false,
			description: "Valid single inheritance should pass validation",
		},
		{
			name: "valid_multiple_inheritance",
			template: &Template{
				Name:           "research-env",
				Inherits:       []string{"python-base", "r-base", "cuda-support"},
				PackageManager: "conda",
			},
			expectError: false,
			description: "Valid multiple inheritance should pass validation",
		},
		{
			name: "no_inheritance_should_pass",
			template: &Template{
				Name:           "standalone",
				Inherits:       []string{},
				PackageManager: "apt",
			},
			expectError: false,
			description: "Template with no inheritance should pass validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the specific validator
			validator := &InheritanceValidator{parser: parser}
			err := validator.Validate(tt.template)

			if tt.expectError {
				require.Error(t, err, "Expected validation error for: %s", tt.description)

				// Verify it's a template validation error
				validationErr, ok := err.(*TemplateValidationError)
				require.True(t, ok, "Error should be TemplateValidationError")

				if tt.expectedField != "" {
					assert.Equal(t, tt.expectedField, validationErr.Field,
						"Error should reference correct field")
				}

				// Log the validation message for debugging
				t.Logf("Validation error (expected): %s", validationErr.Error())

			} else {
				assert.NoError(t, err, "Expected no validation error for: %s", tt.description)
			}
		})
	}
}

// TestInheritanceValidator_IntegrationWithOrchestrator tests inheritance validation through the full orchestrator
func TestInheritanceValidator_IntegrationWithOrchestrator(t *testing.T) {
	parser := NewTemplateParser()
	orchestrator := NewTemplateValidationOrchestrator(parser)

	// Test that inheritance validation is part of the full validation pipeline
	invalidTemplate := &Template{
		Name:           "circular-ref",
		Inherits:       []string{"circular-ref"}, // Self-reference
		PackageManager: "conda",
		// Missing required fields to test multiple validation failures
	}

	err := orchestrator.ValidateAll(invalidTemplate)
	require.Error(t, err, "Orchestrator should catch inheritance validation errors")

	// Should be a validation error
	validationErr, ok := err.(*TemplateValidationError)
	require.True(t, ok, "Should return TemplateValidationError")

	t.Logf("Orchestrator validation error: %s", validationErr.Error())
}

// TestInheritanceValidation_RealWorldScenarios tests scenarios users would actually encounter
func TestInheritanceValidation_RealWorldScenarios(t *testing.T) {
	parser := NewTemplateParser()

	t.Run("user_accidentally_creates_circular_reference", func(t *testing.T) {
		// Simulate: User creates "my-ml-template" and accidentally puts its own name in inherits
		userTemplate := &Template{
			Name:           "my-ml-template",
			Inherits:       []string{"python-base", "my-ml-template"}, // Oops, self-reference
			Description:    "My custom ML environment",
			PackageManager: "conda",
			Packages: PackageDefinitions{
				Conda: []string{"numpy", "pandas", "scikit-learn"},
			},
		}

		validator := &InheritanceValidator{parser: parser}
		err := validator.Validate(userTemplate)

		require.Error(t, err, "Should catch user's accidental self-reference")
		assert.Contains(t, err.Error(), "cannot inherit from itself")
		assert.Contains(t, err.Error(), "my-ml-template")
	})

	t.Run("user_copies_template_with_typo", func(t *testing.T) {
		// Simulate: User copies a template config but leaves empty inherit field
		userTemplate := &Template{
			Name:           "research-workstation",
			Inherits:       []string{"ubuntu-base", ""}, // Copy-paste error
			Description:    "Research computing environment",
			PackageManager: "apt",
		}

		validator := &InheritanceValidator{parser: parser}
		err := validator.Validate(userTemplate)

		require.Error(t, err, "Should catch empty parent name")
		assert.Contains(t, err.Error(), "cannot be empty")
		assert.Contains(t, err.Error(), "inherits[1]")
	})

	t.Run("complex_inheritance_chain_works", func(t *testing.T) {
		// Simulate: Advanced user creates complex but valid inheritance
		advancedTemplate := &Template{
			Name:           "bioinformatics-gpu",
			Inherits:       []string{"ubuntu-22.04-base", "python-3.11", "cuda-12", "bioconda"},
			Description:    "GPU-accelerated bioinformatics environment",
			PackageManager: "conda",
			Packages: PackageDefinitions{
				Conda: []string{"biopython", "pytorch", "tensorflow"},
			},
		}

		validator := &InheritanceValidator{parser: parser}
		err := validator.Validate(advancedTemplate)

		assert.NoError(t, err, "Valid complex inheritance should pass")
	})
}

// TestTemplateValidation_EndToEndLaunchScenario tests the complete validation flow users experience
func TestTemplateValidation_EndToEndLaunchScenario(t *testing.T) {
	parser := NewTemplateParser()

	// Simulate what happens when user tries to launch a template with inheritance issues
	t.Run("launch_fails_with_clear_error_message", func(t *testing.T) {
		problematicTemplate := &Template{
			Name:           "gpu-research",
			Description:    "GPU research environment",
			Base:           "ubuntu-22.04",           // Required field
			Inherits:       []string{"gpu-research"}, // Self-reference that would break launch
			PackageManager: "conda",
			Packages: PackageDefinitions{
				Conda: []string{"pytorch", "tensorflow"},
			},
			Services: []ServiceConfig{
				{Name: "jupyter", Port: 8888},
			},
		}

		// This is what would happen during template validation before launch
		err := parser.ValidateTemplate(problematicTemplate)

		require.Error(t, err, "Template launch should be blocked by validation")

		// Error message should be helpful for users
		errorMsg := err.Error()
		assert.Contains(t, errorMsg, "cannot inherit from itself",
			"Error message should clearly explain the issue")
		assert.Contains(t, errorMsg, "gpu-research",
			"Error message should include the problematic template name")

		t.Logf("User would see this clear error: %s", errorMsg)
	})
}
