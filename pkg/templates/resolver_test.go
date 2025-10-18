package templates

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemplateResolver(t *testing.T) {
	resolver := NewTemplateResolver()

	assert.NotNil(t, resolver)
	assert.NotNil(t, resolver.Parser)
	assert.NotNil(t, resolver.ScriptGen)
	assert.NotNil(t, resolver.AMIRegistry)
}

func TestTemplateResolver_ResolveTemplate(t *testing.T) {
	resolver := NewTemplateResolver()

	// Create test template
	template := &Template{
		Name:           "Test Template",
		Description:    "A test template for resolution",
		Base:           "ubuntu-22.04",
		PackageManager: "apt",
		Packages: PackageDefinitions{
			System: []string{"git", "vim", "curl"},
		},
		Users: []UserConfig{
			{Name: "testuser", Groups: []string{"sudo"}},
		},
		Services: []ServiceConfig{
			{Name: "nginx", Port: 80, Enable: true},
			{Name: "ssh", Port: 22, Enable: true},
		},
		InstanceDefaults: InstanceDefaults{
			Type:  "t3.medium",
			Ports: []int{8080},
			EstimatedCostPerHour: map[string]float64{
				"x86_64": 0.05,
				"arm64":  0.04,
			},
		},
	}

	// Resolve template
	runtime, err := resolver.ResolveTemplate(template, "us-east-1", "x86_64")
	require.NoError(t, err)
	require.NotNil(t, runtime)

	// Verify basic fields
	assert.Equal(t, "Test Template", runtime.Name)
	assert.Equal(t, "A test template for resolution", runtime.Description)
	assert.NotEmpty(t, runtime.UserData)
	assert.True(t, runtime.Generated.After(time.Now().Add(-time.Minute)))

	// Verify AMI mapping
	assert.NotNil(t, runtime.AMI)
	assert.Contains(t, runtime.AMI, "us-east-1")
	assert.Contains(t, runtime.AMI["us-east-1"], "x86_64")

	// Verify instance type mapping
	assert.NotNil(t, runtime.InstanceType)
	assert.Contains(t, runtime.InstanceType, "x86_64")

	// Verify ports (should include SSH + service ports + default ports)
	assert.Contains(t, runtime.Ports, 22)   // SSH
	assert.Contains(t, runtime.Ports, 80)   // nginx service
	assert.Contains(t, runtime.Ports, 8080) // explicit port

	// Verify cost estimates
	assert.NotEmpty(t, runtime.EstimatedCostPerHour)
	assert.Equal(t, 0.05, runtime.EstimatedCostPerHour["x86_64"])

	// Verify source reference
	assert.Equal(t, template, runtime.Source)
}

func TestTemplateResolver_ResolveTemplateWithOptions(t *testing.T) {
	resolver := NewTemplateResolver()

	template := &Template{
		Name:           "Override Test",
		Description:    "Template for testing package manager override",
		Base:           "ubuntu-22.04",
		PackageManager: "apt",
		Packages: PackageDefinitions{
			System: []string{"git"},
			Conda:  []string{"numpy"}, // These should be used when overridden to conda
		},
	}

	// Test without override
	runtime1, err := resolver.ResolveTemplateWithOptions(template, "us-east-1", "x86_64", "", "")
	require.NoError(t, err)
	assert.Contains(t, runtime1.UserData, "apt-get") // Should use APT script

	// Test with conda override
	runtime2, err := resolver.ResolveTemplateWithOptions(template, "us-east-1", "x86_64", "conda", "")
	require.NoError(t, err)
	assert.Contains(t, runtime2.UserData, "conda")     // Should use conda script
	assert.Contains(t, runtime2.UserData, "miniforge") // Conda uses miniforge
}

func TestTemplateResolver_ResolveAllTemplates(t *testing.T) {
	resolver := NewTemplateResolver()

	// Create registry with test templates
	registry := NewTemplateRegistry([]string{})
	registry.Templates["Template1"] = &Template{
		Name: "Template1", Description: "First template", Base: "ubuntu-22.04",
		PackageManager: "apt",
	}
	registry.Templates["Template2"] = &Template{
		Name: "Template2", Description: "Second template", Base: "ubuntu-22.04",
		PackageManager: "conda",
	}

	// Resolve all templates
	runtimeTemplates, err := resolver.ResolveAllTemplates(registry, "us-east-1", "x86_64")
	require.NoError(t, err)

	assert.Len(t, runtimeTemplates, 2)
	assert.Contains(t, runtimeTemplates, "Template1")
	assert.Contains(t, runtimeTemplates, "Template2")

	// Verify each runtime template
	assert.Equal(t, "Template1", runtimeTemplates["Template1"].Name)
	assert.Equal(t, "Template2", runtimeTemplates["Template2"].Name)
}

func TestTemplateResolver_getAMIMapping(t *testing.T) {
	resolver := NewTemplateResolver()

	tests := []struct {
		name         string
		template     *Template
		expectError  bool
		checkMapping func(t *testing.T, mapping map[string]map[string]string)
	}{
		{
			name: "AMI-based template with AMI config",
			template: &Template{
				Name:           "AMI Template",
				Base:           "ami-based",
				PackageManager: "ami",
				AMIConfig: AMIConfig{
					AMIs: map[string]map[string]string{
						"us-east-1": {
							"x86_64": "ami-custom123",
							"arm64":  "ami-custom456",
						},
					},
				},
			},
			expectError: false,
			checkMapping: func(t *testing.T, mapping map[string]map[string]string) {
				assert.Equal(t, "ami-custom123", mapping["us-east-1"]["x86_64"])
				assert.Equal(t, "ami-custom456", mapping["us-east-1"]["arm64"])
			},
		},
		{
			name: "template with base OS mapping",
			template: &Template{
				Name:           "Base Template",
				Base:           "ubuntu-22.04",
				PackageManager: "apt",
			},
			expectError: false,
			checkMapping: func(t *testing.T, mapping map[string]map[string]string) {
				assert.NotEmpty(t, mapping["us-east-1"]["x86_64"])
				assert.NotEmpty(t, mapping["us-east-1"]["arm64"])
			},
		},
		{
			name: "template with unsupported base OS",
			template: &Template{
				Name:           "Unsupported Template",
				Base:           "unsupported-os",
				PackageManager: "apt",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping, err := resolver.getAMIMapping(tt.template, "us-east-1", "x86_64")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkMapping != nil {
					tt.checkMapping(t, mapping)
				}
			}
		})
	}
}

func TestTemplateResolver_getInstanceTypeMapping(t *testing.T) {
	resolver := NewTemplateResolver()

	tests := []struct {
		name     string
		template *Template
		arch     string
		expected map[string]string
	}{
		{
			name: "AMI template with instance type config",
			template: &Template{
				PackageManager: "ami",
				AMIConfig: AMIConfig{
					InstanceTypes: map[string]string{
						"x86_64": "m5.xlarge",
						"arm64":  "m6g.xlarge",
					},
				},
			},
			expected: map[string]string{
				"x86_64": "m5.xlarge",
				"arm64":  "m6g.xlarge",
			},
		},
		{
			name: "template with explicit instance type",
			template: &Template{
				InstanceDefaults: InstanceDefaults{
					Type: "c5.large",
				},
			},
			expected: map[string]string{
				"x86_64": "c5.large",
				"arm64":  "c5.large",
			},
		},
		{
			name: "template requiring GPU",
			template: &Template{
				Packages: PackageDefinitions{
					Conda: []string{"tensorflow-gpu", "pytorch"},
				},
			},
			expected: map[string]string{
				"x86_64": "g4dn.2xlarge",
				"arm64":  "g5g.2xlarge",
			},
		},
		{
			name: "template requiring high memory",
			template: &Template{
				Packages: PackageDefinitions{
					System: []string{"r-base", "spark"},
				},
			},
			expected: map[string]string{
				"x86_64": "r5.2xlarge",
				"arm64":  "r6g.2xlarge",
			},
		},
		{
			name: "template requiring high CPU",
			template: &Template{
				Packages: PackageDefinitions{
					System: []string{"gcc", "openmpi", "build-essential"},
				},
			},
			expected: map[string]string{
				"x86_64": "c5.2xlarge",
				"arm64":  "c6g.2xlarge",
			},
		},
		{
			name: "default template",
			template: &Template{
				Packages: PackageDefinitions{
					System: []string{"git", "vim"},
				},
			},
			expected: map[string]string{
				"x86_64": "t3.large",
				"arm64":  "t4g.large",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.getInstanceTypeMapping(tt.template, tt.arch, "M")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTemplateResolver_templateRequiresGPU(t *testing.T) {
	resolver := NewTemplateResolver()

	tests := []struct {
		name     string
		template *Template
		expected bool
	}{
		{
			name: "template with GPU packages",
			template: &Template{
				Packages: PackageDefinitions{
					Conda: []string{"tensorflow-gpu", "pytorch", "numpy"},
				},
			},
			expected: true,
		},
		{
			name: "template with CUDA packages",
			template: &Template{
				Packages: PackageDefinitions{
					System: []string{"nvidia-cuda-toolkit", "git"},
				},
			},
			expected: true,
		},
		{
			name: "template without GPU packages",
			template: &Template{
				Packages: PackageDefinitions{
					System: []string{"git", "vim", "curl"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.templateRequiresGPU(tt.template)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTemplateResolver_templateRequiresHighMemory(t *testing.T) {
	resolver := NewTemplateResolver()

	tests := []struct {
		name     string
		template *Template
		expected bool
	}{
		{
			name: "template with R packages",
			template: &Template{
				Packages: PackageDefinitions{
					System: []string{"r-base", "r-cran-ggplot2"},
				},
			},
			expected: true,
		},
		{
			name: "template with Spark",
			template: &Template{
				Packages: PackageDefinitions{
					System: []string{"spark", "hadoop"},
				},
			},
			expected: true,
		},
		{
			name: "template without memory-intensive packages",
			template: &Template{
				Packages: PackageDefinitions{
					System: []string{"git", "vim"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.templateRequiresHighMemory(tt.template)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTemplateResolver_templateRequiresHighCPU(t *testing.T) {
	resolver := NewTemplateResolver()

	tests := []struct {
		name     string
		template *Template
		expected bool
	}{
		{
			name: "template with compilation packages",
			template: &Template{
				Packages: PackageDefinitions{
					System: []string{"gcc", "gfortran", "build-essential"},
				},
			},
			expected: true,
		},
		{
			name: "template with MPI packages",
			template: &Template{
				Packages: PackageDefinitions{
					Spack: []string{"openmpi", "mpich"},
				},
			},
			expected: true,
		},
		{
			name: "template without CPU-intensive packages",
			template: &Template{
				Packages: PackageDefinitions{
					System: []string{"git", "nodejs"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.templateRequiresHighCPU(tt.template)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTemplateResolver_hasPackageIndicators(t *testing.T) {
	resolver := NewTemplateResolver()

	template := &Template{
		Packages: PackageDefinitions{
			System: []string{"git", "build-essential", "vim"},
			Conda:  []string{"numpy", "tensorflow-gpu", "pandas"},
			Spack:  []string{"openmpi", "fftw"},
			Pip:    []string{"requests", "flask"},
		},
	}

	// Test various indicators
	assert.True(t, resolver.hasPackageIndicators(template, []string{"tensorflow-gpu"}))
	assert.True(t, resolver.hasPackageIndicators(template, []string{"build-essential"}))
	assert.True(t, resolver.hasPackageIndicators(template, []string{"openmpi"}))
	assert.True(t, resolver.hasPackageIndicators(template, []string{"flask"}))

	// Test partial matches (contains logic)
	assert.True(t, resolver.hasPackageIndicators(template, []string{"tensorflow"})) // matches tensorflow-gpu
	assert.True(t, resolver.hasPackageIndicators(template, []string{"build"}))      // matches build-essential

	// Test non-existent indicators
	assert.False(t, resolver.hasPackageIndicators(template, []string{"nonexistent"}))
	assert.False(t, resolver.hasPackageIndicators(template, []string{"cuda"})) // not in packages
}

func TestTemplateResolver_getPortMapping(t *testing.T) {
	resolver := NewTemplateResolver()

	template := &Template{
		Services: []ServiceConfig{
			{Name: "nginx", Port: 80, Enable: true},
			{Name: "mysql", Port: 3306, Enable: true},
			{Name: "redis", Port: 0, Enable: true}, // No port specified
		},
		InstanceDefaults: InstanceDefaults{
			Ports: []int{8080, 9000, 22}, // 22 should be deduplicated
		},
	}

	ports := resolver.getPortMapping(template)

	// Should always include SSH
	assert.Contains(t, ports, 22)

	// Should include service ports (but not port 0)
	assert.Contains(t, ports, 80)
	assert.Contains(t, ports, 3306)
	assert.NotContains(t, ports, 0)

	// Should include explicit ports
	assert.Contains(t, ports, 8080)
	assert.Contains(t, ports, 9000)

	// Should deduplicate port 22
	count22 := 0
	for _, port := range ports {
		if port == 22 {
			count22++
		}
	}
	assert.Equal(t, 1, count22, "Port 22 should appear only once")
}

func TestTemplateResolver_getCostMapping(t *testing.T) {
	resolver := NewTemplateResolver()

	tests := []struct {
		name      string
		template  *Template
		arch      string
		checkFunc func(t *testing.T, costs map[string]float64)
	}{
		{
			name: "template with explicit cost estimates",
			template: &Template{
				InstanceDefaults: InstanceDefaults{
					EstimatedCostPerHour: map[string]float64{
						"x86_64": 0.15,
						"arm64":  0.12,
					},
				},
			},
			arch: "x86_64",
			checkFunc: func(t *testing.T, costs map[string]float64) {
				assert.Equal(t, 0.15, costs["x86_64"])
				assert.Equal(t, 0.12, costs["arm64"])
			},
		},
		{
			name: "GPU template - should be more expensive",
			template: &Template{
				Packages: PackageDefinitions{
					Conda: []string{"tensorflow-gpu"},
				},
			},
			arch: "x86_64",
			checkFunc: func(t *testing.T, costs map[string]float64) {
				assert.True(t, costs["x86_64"] > 0.5, "GPU instances should be expensive")
			},
		},
		{
			name: "basic template - should use default costs",
			template: &Template{
				Packages: PackageDefinitions{
					System: []string{"git", "vim"},
				},
			},
			arch: "x86_64",
			checkFunc: func(t *testing.T, costs map[string]float64) {
				assert.True(t, costs["x86_64"] > 0.0)
				assert.True(t, costs["x86_64"] < 0.2) // Should be relatively cheap
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			costs := resolver.getCostMapping(tt.template, tt.arch)
			assert.NotEmpty(t, costs)
			if tt.checkFunc != nil {
				tt.checkFunc(t, costs)
			}
		})
	}
}

func TestTemplateResolver_UtilityFunctions(t *testing.T) {
	// Test contains function
	assert.True(t, contains("tensorflow-gpu", "tensorflow"))
	assert.True(t, contains("build-essential", "build"))
	assert.True(t, contains("exact-match", "exact-match"))
	assert.False(t, contains("short", "longer"))
	assert.False(t, contains("nomatch", "xyz"))

	// Test removeDuplicatePorts
	ports := []int{22, 80, 22, 443, 80, 8080}
	unique := removeDuplicatePorts(ports)
	assert.Len(t, unique, 4)
	assert.Contains(t, unique, 22)
	assert.Contains(t, unique, 80)
	assert.Contains(t, unique, 443)
	assert.Contains(t, unique, 8080)

	// Verify no duplicates
	seen := make(map[int]bool)
	for _, port := range unique {
		assert.False(t, seen[port], "Port %d should appear only once", port)
		seen[port] = true
	}
}

func TestTemplateResolver_EdgeCases(t *testing.T) {
	resolver := NewTemplateResolver()

	// Test empty template
	emptyTemplate := &Template{
		Name:           "Empty Template",
		Description:    "Template with no packages",
		Base:           "ubuntu-22.04",
		PackageManager: "apt",
	}

	runtime, err := resolver.ResolveTemplate(emptyTemplate, "us-east-1", "x86_64")
	require.NoError(t, err)
	assert.NotNil(t, runtime)
	assert.Equal(t, "Empty Template", runtime.Name)
	assert.Contains(t, runtime.Ports, 22) // Should always have SSH

	// Test template with unknown instance types in cost calculation
	unknownInstanceTemplate := &Template{
		Name:           "Unknown Instance",
		Description:    "Template with unknown instance type",
		Base:           "ubuntu-22.04",
		PackageManager: "apt",
		InstanceDefaults: InstanceDefaults{
			Type: "unknown.instance.type",
		},
	}

	runtime2, err := resolver.ResolveTemplate(unknownInstanceTemplate, "us-east-1", "x86_64")
	require.NoError(t, err)
	assert.NotNil(t, runtime2.EstimatedCostPerHour)
	// Should have default cost for unknown instance type
	cost := runtime2.EstimatedCostPerHour["x86_64"]
	assert.True(t, cost > 0.0, "Should have a positive cost")
	// The actual cost depends on the resolver's cost calculation logic
}

// Version Resolution Tests (v0.5.5 Universal Version System)

func TestVersionResolver_ResolveAMI(t *testing.T) {
	parser := NewTemplateParser()
	resolver := NewVersionResolver(parser)

	tests := []struct {
		name        string
		distro      string
		version     string
		region      string
		arch        string
		expectError bool
		description string
	}{
		// Ubuntu version resolution
		{
			name:        "Ubuntu 24.04 explicit",
			distro:      "ubuntu",
			version:     "24.04",
			region:      "us-east-1",
			arch:        "x86_64",
			expectError: false,
			description: "Should resolve Ubuntu 24.04",
		},
		{
			name:        "Ubuntu 22.04 explicit",
			distro:      "ubuntu",
			version:     "22.04",
			region:      "us-east-1",
			arch:        "x86_64",
			expectError: false,
			description: "Should resolve Ubuntu 22.04",
		},
		{
			name:        "Ubuntu latest alias",
			distro:      "ubuntu",
			version:     "latest",
			region:      "us-east-1",
			arch:        "x86_64",
			expectError: false,
			description: "Should resolve 'latest' to Ubuntu 24.04",
		},
		{
			name:        "Ubuntu lts alias",
			distro:      "ubuntu",
			version:     "lts",
			region:      "us-east-1",
			arch:        "x86_64",
			expectError: false,
			description: "Should resolve 'lts' to Ubuntu 24.04",
		},
		{
			name:        "Ubuntu previous-lts alias",
			distro:      "ubuntu",
			version:     "previous-lts",
			region:      "us-east-1",
			arch:        "x86_64",
			expectError: false,
			description: "Should resolve 'previous-lts' to Ubuntu 22.04",
		},
		{
			name:        "Ubuntu ARM64",
			distro:      "ubuntu",
			version:     "24.04",
			region:      "us-east-1",
			arch:        "arm64",
			expectError: false,
			description: "Should resolve Ubuntu 24.04 for ARM64",
		},

		// Rocky Linux version resolution
		{
			name:        "Rocky 10 explicit",
			distro:      "rocky",
			version:     "10",
			region:      "us-east-1",
			arch:        "x86_64",
			expectError: false,
			description: "Should resolve Rocky 10",
		},
		{
			name:        "Rocky 9 explicit",
			distro:      "rocky",
			version:     "9",
			region:      "us-east-1",
			arch:        "x86_64",
			expectError: false,
			description: "Should resolve Rocky 9",
		},
		{
			name:        "Rocky latest alias",
			distro:      "rocky",
			version:     "latest",
			region:      "us-east-1",
			arch:        "x86_64",
			expectError: false,
			description: "Should resolve 'latest' to Rocky 10",
		},

		// Amazon Linux version resolution
		{
			name:        "Amazon Linux 2023 explicit",
			distro:      "amazonlinux",
			version:     "2023",
			region:      "us-east-1",
			arch:        "x86_64",
			expectError: false,
			description: "Should resolve Amazon Linux 2023",
		},
		{
			name:        "Amazon Linux 2 explicit",
			distro:      "amazonlinux",
			version:     "2",
			region:      "us-east-1",
			arch:        "x86_64",
			expectError: false,
			description: "Should resolve Amazon Linux 2",
		},

		// Alpine version resolution
		{
			name:        "Alpine 3.20 explicit",
			distro:      "alpine",
			version:     "3.20",
			region:      "us-east-1",
			arch:        "x86_64",
			expectError: false,
			description: "Should resolve Alpine 3.20",
		},

		// Error cases
		{
			name:        "Invalid version",
			distro:      "ubuntu",
			version:     "99.99",
			region:      "us-east-1",
			arch:        "x86_64",
			expectError: true,
			description: "Should fail for non-existent version",
		},
		{
			name:        "Invalid distro",
			distro:      "invalid-distro",
			version:     "1.0",
			region:      "us-east-1",
			arch:        "x86_64",
			expectError: true,
			description: "Should fail for unsupported distro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ami, err := resolver.ResolveAMI(tt.distro, tt.version, tt.region, tt.arch)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotEmpty(t, ami, "AMI should not be empty")
			}
		})
	}
}

func TestVersionResolver_VersionAliases(t *testing.T) {
	parser := NewTemplateParser()
	resolver := NewVersionResolver(parser)

	tests := []struct {
		distro          string
		alias           string
		expectedVersion string
	}{
		{"ubuntu", "latest", "24.04"},
		{"ubuntu", "lts", "24.04"},
		{"ubuntu", "previous-lts", "22.04"},
		{"rocky", "latest", "10"},
		{"rocky", "lts", "9"},
		{"amazonlinux", "latest", "2023"},
		{"alpine", "latest", "3.20"},
		{"rhel", "latest", "9"},
	}

	for _, tt := range tests {
		t.Run(tt.distro+"_"+tt.alias, func(t *testing.T) {
			resolved := resolver.resolveVersionAlias(tt.distro, tt.alias)
			assert.Equal(t, tt.expectedVersion, resolved, "Alias %s for %s should resolve to %s", tt.alias, tt.distro, tt.expectedVersion)
		})
	}
}

func TestVersionResolver_DefaultVersions(t *testing.T) {
	parser := NewTemplateParser()
	resolver := NewVersionResolver(parser)

	tests := []struct {
		distro          string
		expectedDefault string
	}{
		{"ubuntu", "24.04"},
		{"rocky", "10"},
		{"amazonlinux", "2023"},
		{"alpine", "3.20"},
		{"rhel", "9"},
	}

	for _, tt := range tests {
		t.Run(tt.distro, func(t *testing.T) {
			defaultVersion := resolver.getDefaultVersion(tt.distro)
			assert.Equal(t, tt.expectedDefault, defaultVersion, "Default version for %s should be %s", tt.distro, tt.expectedDefault)
		})
	}
}

func TestVersionResolver_GetSupportedVersions(t *testing.T) {
	parser := NewTemplateParser()
	resolver := NewVersionResolver(parser)

	tests := []struct {
		distro      string
		minVersions int
		expectError bool
	}{
		{"ubuntu", 3, false},      // 24.04, 22.04, 20.04
		{"rocky", 2, false},       // 10, 9
		{"amazonlinux", 2, false}, // 2023, 2
		{"alpine", 1, false},      // 3.20
		{"invalid", 0, true},      // Should error
	}

	for _, tt := range tests {
		t.Run(tt.distro, func(t *testing.T) {
			versions, err := resolver.GetSupportedVersions(tt.distro)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, len(versions), tt.minVersions, "Should have at least %d versions", tt.minVersions)
			}
		})
	}
}

func TestVersionResolver_ValidateVersion(t *testing.T) {
	parser := NewTemplateParser()
	resolver := NewVersionResolver(parser)

	tests := []struct {
		distro      string
		version     string
		expectError bool
	}{
		// Valid versions
		{"ubuntu", "24.04", false},
		{"ubuntu", "22.04", false},
		{"ubuntu", "20.04", false},
		{"ubuntu", "latest", false},
		{"ubuntu", "lts", false},
		{"rocky", "10", false},
		{"rocky", "9", false},
		{"amazonlinux", "2023", false},
		{"amazonlinux", "2", false},

		// Invalid versions
		{"ubuntu", "99.99", true},
		{"ubuntu", "invalid", true},
		{"rocky", "100", true},
		{"invalid-distro", "1.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.distro+"_"+tt.version, func(t *testing.T) {
			err := resolver.ValidateVersion(tt.distro, tt.version)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
