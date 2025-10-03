package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComprehensiveValidator tests the complete template validation system
func TestComprehensiveValidatorFunctional(t *testing.T) {
	// Create a test registry
	registry := NewTemplateRegistry([]string{})
	validator := NewComprehensiveValidator(registry)

	tests := []struct {
		name        string
		template    *Template
		expectError bool
		errorMsg    string
		description string
	}{
		{
			name: "valid_complete_template",
			template: &Template{
				Name:           "valid-ml-template",
				Description:    "A valid machine learning template",
				Base:           "ubuntu-22.04",
				PackageManager: "conda",
				Packages: PackageDefinitions{
					Conda: []string{"numpy", "pandas", "scikit-learn"},
				},
				Services: []ServiceConfig{
					{Name: "jupyter", Port: 8888},
				},
				Users: []UserConfig{
					{Name: "mluser", Groups: []string{"sudo"}},
				},
			},
			expectError: false,
			description: "Complete valid template should pass all validation",
		},
		{
			name: "invalid_empty_name",
			template: &Template{
				Name:           "",
				Description:    "Template with empty name",
				Base:           "ubuntu-22.04",
				PackageManager: "conda",
			},
			expectError: true,
			errorMsg:    "name",
			description: "Template with empty name should fail validation",
		},
		{
			name: "invalid_empty_description",
			template: &Template{
				Name:           "no-description-template",
				Description:    "",
				Base:           "ubuntu-22.04",
				PackageManager: "conda",
			},
			expectError: true,
			errorMsg:    "description",
			description: "Template with empty description should fail validation",
		},
		{
			name: "invalid_empty_base",
			template: &Template{
				Name:           "no-base-template",
				Description:    "Template without base OS",
				Base:           "",
				PackageManager: "conda",
			},
			expectError: true,
			errorMsg:    "base",
			description: "Template without base OS should fail validation",
		},
		{
			name: "invalid_username_with_spaces",
			template: &Template{
				Name:           "invalid-user-template",
				Description:    "Template with invalid username",
				Base:           "ubuntu-22.04",
				PackageManager: "apt",
				Users: []UserConfig{
					{Name: "invalid user", Groups: []string{"sudo"}},
				},
			},
			expectError: true,
			errorMsg:    "username",
			description: "Template with invalid username (spaces) should fail validation",
		},
		{
			name: "duplicate_port_conflict",
			template: &Template{
				Name:           "invalid-port-template",
				Description:    "Template with port conflicts",
				Base:           "ubuntu-22.04",
				PackageManager: "apt",
				Services: []ServiceConfig{
					{Name: "jupyter", Port: 8888},
					{Name: "tensorboard", Port: 8888}, // Duplicate port
				},
			},
			expectError: true,
			errorMsg:    "conflict",
			description: "Template with duplicate ports should fail validation",
		},
		{
			name: "circular_inheritance",
			template: &Template{
				Name:           "circular-template",
				Description:    "Template with circular inheritance",
				Base:           "ubuntu-22.04",
				Inherits:       []string{"circular-template"}, // Self-reference
				PackageManager: "apt",
			},
			expectError: true,
			errorMsg:    "circular",
			description: "Template with circular inheritance should fail validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := validator.ValidateTemplate(tt.template)

			if tt.expectError {
				assert.False(t, report.Valid, "Expected validation failure for: %s", tt.description)
				assert.Greater(t, report.ErrorCount, 0, "Expected validation errors for: %s", tt.description)
				if tt.errorMsg != "" {
					// Check if any validation result contains the expected error message
					found := false
					for _, result := range report.Results {
						if strings.Contains(result.Message, tt.errorMsg) || strings.Contains(result.Field, tt.errorMsg) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error message to contain '%s' for: %s", tt.errorMsg, tt.description)
				}
				t.Logf("Template validation error (expected): %d errors, %d warnings", report.ErrorCount, report.WarningCount)
			} else {
				assert.True(t, report.Valid, "Expected no validation error for: %s", tt.description)
				assert.Equal(t, 0, report.ErrorCount, "Expected no validation errors for: %s", tt.description)
			}
		})
	}
}

// TestValidateAllTemplates tests bulk template validation functionality
func TestValidateAllTemplatesFunctional(t *testing.T) {
	// Create a test registry with test templates
	registry := NewTemplateRegistry([]string{})
	validator := NewComprehensiveValidator(registry)

	templates := []*Template{
		{
			Name:           "valid-python-template",
			Description:    "Valid Python template",
			Base:           "ubuntu-22.04",
			PackageManager: "conda",
			Packages: PackageDefinitions{
				Conda: []string{"python", "pip"},
			},
		},
		{
			Name:           "invalid-template", // Missing description
			Base:           "ubuntu-22.04",
			PackageManager: "apt",
		},
		{
			Name:           "another-valid-template",
			Description:    "Another valid template",
			Base:           "rocky-linux-9",
			PackageManager: "dnf",
		},
	}

	// Add templates to registry
	for _, template := range templates {
		registry.Templates[template.Name] = template
	}

	reports := validator.ValidateAll()
	require.NotEmpty(t, reports, "ValidateAll should return validation reports")

	// Check that at least one template failed validation
	invalidFound := false
	for _, report := range reports {
		if !report.Valid {
			invalidFound = true
			break
		}
	}
	assert.True(t, invalidFound, "At least one template should fail validation")

	t.Logf("Bulk validation completed: %d templates validated", len(reports))
}

// TestTemplateValidationRealWorldScenarios tests scenarios users actually encounter
func TestTemplateValidationRealWorldScenarios(t *testing.T) {
	// Create a test registry
	registry := NewTemplateRegistry([]string{})
	validator := NewComprehensiveValidator(registry)

	t.Run("researcher_copies_template_with_typos", func(t *testing.T) {
		// Simulate: Researcher copies template but introduces typos
		template := &Template{
			Name:           "my-ml-workstation",
			Description:    "My ML workstation", // Too short description
			Base:           "ubuntu-22.04",
			PackageManager: "conda",
			Users: []UserConfig{
				{Name: "ML-User", Groups: []string{"sudo"}}, // Invalid uppercase username
			},
			Services: []ServiceConfig{
				{Name: "jupyter", Port: 8888},
				{Name: "tensorboard", Port: 8888}, // Duplicate port
			},
		}

		report := validator.ValidateTemplate(template)
		assert.False(t, report.Valid, "Should catch multiple validation issues")
		assert.Greater(t, report.ErrorCount, 0, "Should have validation errors")

		// Should catch username issue
		usernameErrorFound := false
		for _, result := range report.Results {
			if strings.Contains(result.Field, "users") || strings.Contains(result.Message, "username") {
				usernameErrorFound = true
				break
			}
		}
		assert.True(t, usernameErrorFound, "Should detect invalid username")

		t.Logf("User template validation errors: %d errors, %d warnings", report.ErrorCount, report.WarningCount)
	})

	t.Run("lab_creates_gpu_template_with_invalid_config", func(t *testing.T) {
		// Simulate: Computer lab creates GPU template with configuration issues
		template := &Template{
			Name:           "gpu-deep-learning",
			Description:    "GPU deep learning environment for computer lab",
			Base:           "ubuntu-22.04",
			PackageManager: "conda",
			Packages: PackageDefinitions{
				Conda: []string{}, // Empty package list
			},
			Users: []UserConfig{
				{Name: "student1", Groups: []string{"sudo"}},
				{Name: "student1", Groups: []string{"docker"}}, // Duplicate user
			},
		}

		report := validator.ValidateTemplate(template)
		assert.False(t, report.Valid, "Should catch configuration issues")
		assert.Greater(t, report.ErrorCount, 0, "Should have validation errors")

		t.Logf("Lab template validation errors: %d errors, %d warnings", report.ErrorCount, report.WarningCount)
	})

	t.Run("professor_inheritance_chain_validation", func(t *testing.T) {
		// Simulate: Professor creates complex inheritance chain
		baseTemplate := &Template{
			Name:           "course-base",
			Description:    "Base template for course",
			Base:           "ubuntu-22.04",
			PackageManager: "apt",
		}

		advancedTemplate := &Template{
			Name:           "advanced-course",
			Description:    "Advanced course template",
			Base:           "ubuntu-22.04",
			Inherits:       []string{"course-base"},
			PackageManager: "conda", // Override parent package manager
			Packages: PackageDefinitions{
				Conda: []string{"advanced-packages"},
			},
		}

		// Add templates to registry for inheritance validation
		registry.Templates[baseTemplate.Name] = baseTemplate
		registry.Templates[advancedTemplate.Name] = advancedTemplate

		// Validate base template first
		report := validator.ValidateTemplate(baseTemplate)
		assert.True(t, report.Valid, "Base template should be valid")

		// Validate advanced template (inheritance validation)
		report = validator.ValidateTemplate(advancedTemplate)
		if !report.Valid {
			t.Logf("Inheritance validation result: %d errors, %d warnings", report.ErrorCount, report.WarningCount)
		}
	})
}

// TestTemplateValidatorComponents tests individual validation components
func TestTemplateValidatorComponentsFunctional(t *testing.T) {
	// Create a test registry
	registry := NewTemplateRegistry([]string{})
	validator := NewComprehensiveValidator(registry)

	t.Run("name_validation", func(t *testing.T) {
		tests := []struct {
			name     string
			template *Template
			valid    bool
		}{
			{"empty_name", &Template{Name: ""}, false},
			{"valid_name", &Template{Name: "python-ml"}, true},
			{"name_with_spaces", &Template{Name: "python ml"}, true}, // Spaces allowed in name
			{"very_long_name", &Template{Name: "this-is-a-very-long-template-name-that-might-cause-issues-in-some-systems"}, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Add minimal required fields
				tt.template.Description = "Test description"
				tt.template.Base = "ubuntu-22.04"
				tt.template.PackageManager = "apt"

				report := validator.ValidateTemplate(tt.template)
				if tt.valid {
					assert.True(t, report.Valid, "Template should be valid")
					assert.Equal(t, 0, report.ErrorCount, "Template should have no errors")
				} else {
					assert.False(t, report.Valid, "Template should be invalid")
					assert.Greater(t, report.ErrorCount, 0, "Template should have errors")
				}
			})
		}
	})

	t.Run("service_port_validation", func(t *testing.T) {
		tests := []struct {
			port  int
			valid bool
		}{
			{22, true},    // SSH (privileged port - warning, but still valid)
			{80, true},    // HTTP (privileged port - warning, but still valid)
			{443, true},   // HTTPS (privileged port - warning, but still valid)
			{8888, true},  // Jupyter (normal port)
			{0, true},     // Port 0 is ignored by validator
			{65536, true}, // High port - not currently validated
		}

		for _, tt := range tests {
			template := &Template{
				Name:           "port-test-template",
				Description:    "Template for testing port validation",
				Base:           "ubuntu-22.04",
				PackageManager: "apt",
				Services: []ServiceConfig{
					{Name: "test-service", Port: tt.port},
				},
			}

			report := validator.ValidateTemplate(template)
			assert.True(t, report.Valid, "Port %d should be valid (warnings don't fail validation)", tt.port)
			assert.Equal(t, 0, report.ErrorCount, "Port %d should have no errors", tt.port)

			// Check for privileged port warnings
			if tt.port < 1024 && tt.port > 0 {
				assert.Greater(t, report.WarningCount, 0, "Port %d should have warnings", tt.port)
			}
		}
	})
}
