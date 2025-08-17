package templates

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemplateParser(t *testing.T) {
	parser := NewTemplateParser()

	assert.NotNil(t, parser)
	assert.NotNil(t, parser.BaseAMIs)
	assert.NotNil(t, parser.Strategy)

	// Verify default base AMIs are loaded
	assert.Contains(t, parser.BaseAMIs, "ubuntu-22.04")
	assert.Contains(t, parser.BaseAMIs["ubuntu-22.04"], "us-east-1")
	assert.Contains(t, parser.BaseAMIs["ubuntu-22.04"]["us-east-1"], "x86_64")
}

func TestTemplateParser_ParseTemplate(t *testing.T) {
	parser := NewTemplateParser()

	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		checkFunc   func(t *testing.T, template *Template)
	}{
		{
			name: "valid basic template",
			yamlContent: `
name: "Test Template"
description: "A test template"
base: "ubuntu-22.04"
package_manager: "apt"
packages:
  system: ["git", "curl"]
users:
  - name: "testuser"
    password: "auto-generated"
    groups: ["sudo"]
services:
  - name: "nginx"
    port: 80
    enable: true
`,
			wantErr: false,
			checkFunc: func(t *testing.T, template *Template) {
				assert.Equal(t, "Test Template", template.Name)
				assert.Equal(t, "A test template", template.Description)
				assert.Equal(t, "ubuntu-22.04", template.Base)
				assert.Equal(t, "apt", template.PackageManager)
				assert.Equal(t, []string{"git", "curl"}, template.Packages.System)
				assert.Len(t, template.Users, 1)
				assert.Equal(t, "testuser", template.Users[0].Name)
				assert.Equal(t, "/bin/bash", template.Users[0].Shell) // Default shell
				assert.Equal(t, []string{"sudo"}, template.Users[0].Groups)
				assert.Len(t, template.Services, 1)
				assert.Equal(t, "nginx", template.Services[0].Name)
				assert.Equal(t, 80, template.Services[0].Port)
				assert.True(t, template.Services[0].Enable)
			},
		},
		{
			name: "template with inheritance",
			yamlContent: `
name: "Child Template"
description: "A child template"
base: "ubuntu-22.04"
inherits: ["parent-template"]
package_manager: "conda"
packages:
  conda: ["numpy", "pandas"]
`,
			wantErr: false,
			checkFunc: func(t *testing.T, template *Template) {
				assert.Equal(t, "Child Template", template.Name)
				assert.Equal(t, []string{"parent-template"}, template.Inherits)
				assert.Equal(t, "conda", template.PackageManager)
				assert.Equal(t, []string{"numpy", "pandas"}, template.Packages.Conda)
			},
		},
		{
			name: "AMI-based template",
			yamlContent: `
name: "AMI Template"
description: "An AMI-based template"
base: "ami-based"
package_manager: "ami"
ami_config:
  amis:
    us-east-1:
      x86_64: "ami-12345678"
      arm64: "ami-87654321"
  ssh_user: "ubuntu"
`,
			wantErr: false,
			checkFunc: func(t *testing.T, template *Template) {
				assert.Equal(t, "AMI Template", template.Name)
				assert.Equal(t, "ami-based", template.Base)
				assert.Equal(t, "ami", template.PackageManager)
				assert.NotNil(t, template.AMIConfig.AMIs)
				assert.Equal(t, "ami-12345678", template.AMIConfig.AMIs["us-east-1"]["x86_64"])
				assert.Equal(t, "ubuntu", template.AMIConfig.SSHUser)
			},
		},
		{
			name: "invalid YAML",
			yamlContent: `
invalid: yaml: content:
  - malformed
`,
			wantErr: true,
		},
		{
			name: "missing required name",
			yamlContent: `
description: "Missing name"
base: "ubuntu-22.04"
package_manager: "apt"
`,
			wantErr: true,
		},
		{
			name: "missing required description",
			yamlContent: `
name: "Missing Description"
base: "ubuntu-22.04"
package_manager: "apt"
`,
			wantErr: true,
		},
		{
			name: "missing required base",
			yamlContent: `
name: "Missing Base"
description: "Template missing base OS"
package_manager: "apt"
`,
			wantErr: true,
		},
		{
			name: "invalid package manager",
			yamlContent: `
name: "Invalid PM"
description: "Template with invalid package manager"
base: "ubuntu-22.04"
package_manager: "invalid-pm"
`,
			wantErr: true,
		},
		{
			name: "invalid service port",
			yamlContent: `
name: "Invalid Port"
description: "Template with invalid port"
base: "ubuntu-22.04"
package_manager: "apt"
services:
  - name: "badservice"
    port: 99999
`,
			wantErr: true,
		},
		{
			name: "invalid user name",
			yamlContent: `
name: "Invalid User"
description: "Template with invalid user name"
base: "ubuntu-22.04"
package_manager: "apt"
users:
  - name: "bad user:name"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := parser.ParseTemplate([]byte(tt.yamlContent))

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, template)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, template)
				if tt.checkFunc != nil {
					tt.checkFunc(t, template)
				}
			}
		})
	}
}

func TestTemplateParser_ParseTemplateFile(t *testing.T) {
	parser := NewTemplateParser()

	// Create temporary template file
	tempDir, err := os.MkdirTemp("", "template-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	templateContent := `
name: "File Template"
description: "Template loaded from file"
base: "ubuntu-22.04"
package_manager: "apt"
packages:
  system: ["vim", "git"]
`

	templateFile := filepath.Join(tempDir, "test-template.yaml")
	err = os.WriteFile(templateFile, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Test successful parsing
	template, err := parser.ParseTemplateFile(templateFile)
	assert.NoError(t, err)
	assert.NotNil(t, template)
	assert.Equal(t, "File Template", template.Name)
	assert.Equal(t, []string{"vim", "git"}, template.Packages.System)

	// Test with valid template having explicit name
	templateValid := `
name: "Valid File Template"
description: "Template with explicit name"
base: "ubuntu-22.04"
package_manager: "apt"
packages:
  system: ["git", "curl"]
`
	templateFile2 := filepath.Join(tempDir, "valid-template.yml")
	err = os.WriteFile(templateFile2, []byte(templateValid), 0644)
	require.NoError(t, err)

	template2, err := parser.ParseTemplateFile(templateFile2)
	assert.NoError(t, err)
	assert.Equal(t, "Valid File Template", template2.Name)
	assert.Equal(t, []string{"git", "curl"}, template2.Packages.System)

	// Test non-existent file
	_, err = parser.ParseTemplateFile("/nonexistent/file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read template file")
}

func TestTemplateParser_ValidateTemplate(t *testing.T) {
	parser := NewTemplateParser()

	validTemplate := &Template{
		Name:           "Valid Template",
		Description:    "A valid template",
		Base:           "ubuntu-22.04",
		PackageManager: "apt",
		Packages: PackageDefinitions{
			System: []string{"git", "curl"},
		},
		Users: []UserConfig{
			{Name: "testuser", Groups: []string{"sudo"}},
		},
		Services: []ServiceConfig{
			{Name: "nginx", Port: 80},
		},
		InstanceDefaults: InstanceDefaults{
			Ports: []int{22, 80},
		},
	}

	// Test valid template
	err := parser.ValidateTemplate(validTemplate)
	assert.NoError(t, err)

	// Test validation errors
	tests := []struct {
		name          string
		modifyFunc    func(*Template)
		expectedError string
	}{
		{
			name: "empty name",
			modifyFunc: func(t *Template) {
				t.Name = ""
			},
			expectedError: "template name is required",
		},
		{
			name: "empty description",
			modifyFunc: func(t *Template) {
				t.Description = ""
			},
			expectedError: "template description is required",
		},
		{
			name: "empty base",
			modifyFunc: func(t *Template) {
				t.Base = ""
			},
			expectedError: "base OS is required",
		},
		{
			name: "unsupported base OS",
			modifyFunc: func(t *Template) {
				t.Base = "unsupported-os"
			},
			expectedError: "unsupported base OS",
		},
		{
			name: "invalid package manager",
			modifyFunc: func(t *Template) {
				t.PackageManager = "invalid"
			},
			expectedError: "unsupported package manager",
		},
		{
			name: "service without name",
			modifyFunc: func(t *Template) {
				t.Services = []ServiceConfig{{Port: 80}}
			},
			expectedError: "service name is required",
		},
		{
			name: "invalid service port",
			modifyFunc: func(t *Template) {
				t.Services = []ServiceConfig{{Name: "test", Port: 70000}}
			},
			expectedError: "service port must be between 0 and 65535",
		},
		{
			name: "user without name",
			modifyFunc: func(t *Template) {
				t.Users = []UserConfig{{Groups: []string{"sudo"}}}
			},
			expectedError: "user name is required",
		},
		{
			name: "invalid user name with space",
			modifyFunc: func(t *Template) {
				t.Users = []UserConfig{{Name: "bad user"}}
			},
			expectedError: "user name cannot contain spaces or colons",
		},
		{
			name: "invalid user name with colon",
			modifyFunc: func(t *Template) {
				t.Users = []UserConfig{{Name: "bad:user"}}
			},
			expectedError: "user name cannot contain spaces or colons",
		},
		{
			name: "invalid port range",
			modifyFunc: func(t *Template) {
				t.InstanceDefaults.Ports = []int{0}
			},
			expectedError: "port must be between 1 and 65535",
		},
		{
			name: "self-inheritance",
			modifyFunc: func(t *Template) {
				t.Inherits = []string{t.Name}
			},
			expectedError: "template cannot inherit from itself",
		},
		{
			name: "empty parent name",
			modifyFunc: func(t *Template) {
				t.Inherits = []string{""}
			},
			expectedError: "parent template name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the valid template
			testTemplate := *validTemplate
			// Apply modification
			tt.modifyFunc(&testTemplate)

			err := parser.ValidateTemplate(&testTemplate)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestTemplateParser_ValidatePackageConsistency(t *testing.T) {
	parser := NewTemplateParser()

	tests := []struct {
		name     string
		template *Template
		wantErr  bool
		errMsg   string
	}{
		{
			name: "apt with system packages - valid",
			template: &Template{
				Name: "APT Template", Description: "Test", Base: "ubuntu-22.04",
				PackageManager: "apt",
				Packages:       PackageDefinitions{System: []string{"git"}},
			},
			wantErr: false,
		},
		{
			name: "apt with conda packages - invalid",
			template: &Template{
				Name: "APT Template", Description: "Test", Base: "ubuntu-22.04",
				PackageManager: "apt",
				Packages: PackageDefinitions{
					System: []string{"git"},
					Conda:  []string{"numpy"},
				},
			},
			wantErr: true,
			errMsg:  "APT package manager but has conda/spack packages",
		},
		{
			name: "conda with mixed packages - valid",
			template: &Template{
				Name: "Conda Template", Description: "Test", Base: "ubuntu-22.04",
				PackageManager: "conda",
				Packages: PackageDefinitions{
					System: []string{"build-essential"}, // OK for conda
					Conda:  []string{"numpy"},
				},
			},
			wantErr: false,
		},
		{
			name: "ami with packages - invalid",
			template: &Template{
				Name: "AMI Template", Description: "Test", Base: "ami-based",
				PackageManager: "ami",
				Packages:       PackageDefinitions{System: []string{"git"}},
			},
			wantErr: true,
			errMsg:  "AMI-based templates should not define packages",
		},
		{
			name: "no package manager specified - valid",
			template: &Template{
				Name: "No PM Template", Description: "Test", Base: "ubuntu-22.04",
				Packages: PackageDefinitions{System: []string{"git"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.validatePackageConsistency(tt.template)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPackageManagerStrategy_SelectPackageManager(t *testing.T) {
	strategy := NewPackageManagerStrategy()

	tests := []struct {
		name            string
		template        *Template
		expectedManager PackageManagerType
	}{
		{
			name:            "explicit apt",
			template:        &Template{PackageManager: "apt"},
			expectedManager: PackageManagerApt,
		},
		{
			name:            "explicit conda",
			template:        &Template{PackageManager: "conda"},
			expectedManager: PackageManagerConda,
		},
		{
			name:            "explicit spack",
			template:        &Template{PackageManager: "spack"},
			expectedManager: PackageManagerSpack,
		},
		{
			name:            "explicit ami",
			template:        &Template{PackageManager: "ami"},
			expectedManager: PackageManagerAMI,
		},
		{
			name:            "explicit dnf",
			template:        &Template{PackageManager: "dnf"},
			expectedManager: PackageManagerDnf,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.SelectPackageManager(tt.template)
			assert.Equal(t, tt.expectedManager, result)
		})
	}
}

func TestTemplateValidationError(t *testing.T) {
	err := &TemplateValidationError{
		Field:   "test_field",
		Message: "test message",
	}

	expected := "template validation error in test_field: test message"
	assert.Equal(t, expected, err.Error())
}

func TestNewTemplateRegistry(t *testing.T) {
	dirs := []string{"/tmp/templates", "/opt/templates"}
	registry := NewTemplateRegistry(dirs)

	assert.NotNil(t, registry)
	assert.Equal(t, dirs, registry.TemplateDirs)
	assert.NotNil(t, registry.Templates)
	assert.Empty(t, registry.Templates)
}

func TestTemplateRegistry_ScanTemplates(t *testing.T) {
	// Create temporary template directory
	tempDir, err := os.MkdirTemp("", "template-registry-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create test templates
	template1 := `
name: "Template 1"
description: "First test template"
base: "ubuntu-22.04"
package_manager: "apt"
packages:
  system: ["git", "vim"]
`
	template2 := `
name: "Template 2"
description: "Second test template"
base: "ubuntu-22.04"
package_manager: "conda"
packages:
  conda: ["numpy", "pandas"]
`
	childTemplate := `
name: "Child Template"
description: "Template with inheritance"
base: "ubuntu-22.04"
inherits: ["Template 1"]
package_manager: "apt"
packages:
  system: ["curl"]
`

	// Write template files
	err = os.WriteFile(filepath.Join(tempDir, "template1.yaml"), []byte(template1), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "template2.yml"), []byte(template2), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "child.yaml"), []byte(childTemplate), 0644)
	require.NoError(t, err)

	// Create non-template file (should be ignored)
	err = os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("not a template"), 0644)
	require.NoError(t, err)

	// Create subdirectory with template
	subDir := filepath.Join(tempDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	template3 := `
name: "Subdirectory Template"
description: "Template in subdirectory"
base: "ubuntu-22.04"
package_manager: "apt"
packages:
  system: ["htop"]
`
	err = os.WriteFile(filepath.Join(subDir, "template3.yaml"), []byte(template3), 0644)
	require.NoError(t, err)

	// Test registry scanning
	registry := NewTemplateRegistry([]string{tempDir})
	err = registry.ScanTemplates()
	assert.NoError(t, err)

	// Verify templates were loaded
	assert.Len(t, registry.Templates, 4)
	assert.Contains(t, registry.Templates, "Template 1")
	assert.Contains(t, registry.Templates, "Template 2")
	assert.Contains(t, registry.Templates, "Child Template")
	assert.Contains(t, registry.Templates, "Subdirectory Template")

	// Verify inheritance was resolved
	childTemp, exists := registry.Templates["Child Template"]
	assert.True(t, exists)
	// Child should have both parent and own packages
	expectedPackages := []string{"git", "vim", "curl"}
	assert.Equal(t, expectedPackages, childTemp.Packages.System)

	// Verify last scan time was set
	assert.True(t, registry.LastScan.After(time.Now().Add(-time.Minute)))

	// Test with non-existent directory
	registry2 := NewTemplateRegistry([]string{"/nonexistent/dir"})
	err = registry2.ScanTemplates()
	assert.NoError(t, err) // Should not error, just skip
	assert.Empty(t, registry2.Templates)
}

func TestTemplateRegistry_GetTemplate(t *testing.T) {
	registry := NewTemplateRegistry([]string{})

	// Add test template
	testTemplate := &Template{
		Name:        "Test Template",
		Description: "A test template",
		Base:        "ubuntu-22.04",
	}
	registry.Templates["Test Template"] = testTemplate

	// Test successful retrieval
	template, err := registry.GetTemplate("Test Template")
	assert.NoError(t, err)
	assert.NotNil(t, template)
	assert.Equal(t, "Test Template", template.Name)

	// Test non-existent template
	_, err = registry.GetTemplate("Nonexistent Template")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
}

func TestTemplateRegistry_ListTemplates(t *testing.T) {
	registry := NewTemplateRegistry([]string{})

	// Add test templates
	template1 := &Template{Name: "Template 1"}
	template2 := &Template{Name: "Template 2"}
	registry.Templates["Template 1"] = template1
	registry.Templates["Template 2"] = template2

	templates := registry.ListTemplates()
	assert.Len(t, templates, 2)
	assert.Contains(t, templates, "Template 1")
	assert.Contains(t, templates, "Template 2")
}

func TestTemplateRegistry_ResolveInheritance(t *testing.T) {
	registry := NewTemplateRegistry([]string{})

	// Create parent template
	parent := &Template{
		Name:           "Parent",
		Description:    "Parent template",
		Base:           "ubuntu-22.04",
		PackageManager: "apt",
		Packages: PackageDefinitions{
			System: []string{"git", "vim"},
		},
		Users: []UserConfig{
			{Name: "parent-user", Groups: []string{"sudo"}},
		},
		Services: []ServiceConfig{
			{Name: "ssh", Port: 22, Enable: true},
		},
		InstanceDefaults: InstanceDefaults{
			Ports:                []int{22},
			EstimatedCostPerHour: map[string]float64{"x86_64": 0.10},
		},
		Tags: map[string]string{"type": "parent"},
	}

	// Create child template
	child := &Template{
		Name:           "Child",
		Description:    "Child template",
		Base:           "ubuntu-22.04",
		Inherits:       []string{"Parent"},
		PackageManager: "conda", // Override parent
		Packages: PackageDefinitions{
			Conda: []string{"numpy"}, // Add conda packages
		},
		Users: []UserConfig{
			{Name: "child-user", Groups: []string{"users"}},
		},
		Services: []ServiceConfig{
			{Name: "jupyter", Port: 8888, Enable: true},
		},
		InstanceDefaults: InstanceDefaults{
			Ports:                []int{8888}, // Will be merged with parent's ports
			EstimatedCostPerHour: map[string]float64{"arm64": 0.08},
		},
		Tags: map[string]string{"type": "child", "env": "test"}, // Override + add
	}

	registry.Templates["Parent"] = parent
	registry.Templates["Child"] = child

	err := registry.ResolveInheritance()
	assert.NoError(t, err)

	// Verify child template was merged correctly
	resolvedChild := registry.Templates["Child"]

	// Package manager should be overridden
	assert.Equal(t, "conda", resolvedChild.PackageManager)

	// Packages should be merged
	assert.Equal(t, []string{"git", "vim"}, resolvedChild.Packages.System)
	assert.Equal(t, []string{"numpy"}, resolvedChild.Packages.Conda)

	// Users should be merged (both parent and child users)
	assert.Len(t, resolvedChild.Users, 2)
	userNames := make([]string, len(resolvedChild.Users))
	for i, u := range resolvedChild.Users {
		userNames[i] = u.Name
	}
	assert.Contains(t, userNames, "parent-user")
	assert.Contains(t, userNames, "child-user")

	// Services should be merged
	assert.Len(t, resolvedChild.Services, 2)
	serviceNames := make([]string, len(resolvedChild.Services))
	for i, s := range resolvedChild.Services {
		serviceNames[i] = s.Name
	}
	assert.Contains(t, serviceNames, "ssh")
	assert.Contains(t, serviceNames, "jupyter")

	// Ports should be merged and deduplicated
	assert.Contains(t, resolvedChild.InstanceDefaults.Ports, 22)
	assert.Contains(t, resolvedChild.InstanceDefaults.Ports, 8888)

	// Cost estimates should be merged
	assert.Equal(t, 0.10, resolvedChild.InstanceDefaults.EstimatedCostPerHour["x86_64"])
	assert.Equal(t, 0.08, resolvedChild.InstanceDefaults.EstimatedCostPerHour["arm64"])

	// Tags should be merged (child overrides + adds)
	assert.Equal(t, "child", resolvedChild.Tags["type"])
	assert.Equal(t, "test", resolvedChild.Tags["env"])
}

func TestTemplateRegistry_ResolveInheritance_Errors(t *testing.T) {
	registry := NewTemplateRegistry([]string{})

	// Test missing parent template
	child := &Template{
		Name:        "Child",
		Description: "Child template",
		Base:        "ubuntu-22.04",
		Inherits:    []string{"NonexistentParent"},
	}
	registry.Templates["Child"] = child

	err := registry.ResolveInheritance()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parent template not found: NonexistentParent")
}
