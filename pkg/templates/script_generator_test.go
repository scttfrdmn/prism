package templates

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScriptGenerator(t *testing.T) {
	generator := NewScriptGenerator()
	
	assert.NotNil(t, generator)
	assert.NotEmpty(t, generator.AptTemplate)
	assert.NotEmpty(t, generator.DnfTemplate)
	assert.NotEmpty(t, generator.CondaTemplate)
	assert.NotEmpty(t, generator.SpackTemplate)
	assert.NotEmpty(t, generator.AMITemplate)
}

func TestScriptGenerator_GenerateScript_APT(t *testing.T) {
	generator := NewScriptGenerator()
	
	template := &Template{
		Name:        "APT Test Template",
		Description: "Template for APT testing",
		Base:        "ubuntu-22.04",
		PackageManager: "apt",
		Packages: PackageDefinitions{
			System: []string{"git", "vim", "curl", "build-essential"},
		},
		Users: []UserConfig{
			{Name: "developer", Password: "auto-generated", Groups: []string{"sudo", "docker"}},
			{Name: "researcher", Password: "custom123", Shell: "/bin/zsh"},
		},
		Services: []ServiceConfig{
			{Name: "nginx", Port: 80, Enable: true, Config: []string{"worker_processes auto;"}},
			{Name: "redis", Port: 6379, Enable: false},
		},
		PostInstall: "echo 'Custom post-install script'",
	}
	
	script, err := generator.GenerateScript(template, PackageManagerApt)
	require.NoError(t, err)
	assert.NotEmpty(t, script)
	
	// Verify script content
	assert.Contains(t, script, "#!/bin/bash")
	assert.Contains(t, script, "APT Test Template")
	assert.Contains(t, script, "apt-get update")
	assert.Contains(t, script, "apt-get install -y git vim curl build-essential")
	
	// Verify user creation
	assert.Contains(t, script, "useradd -m -s /bin/bash developer")
	assert.Contains(t, script, "useradd -m -s /bin/zsh researcher")
	assert.Contains(t, script, "usermod -aG sudo developer")
	assert.Contains(t, script, "usermod -aG docker developer")
	assert.Contains(t, script, "researcher:custom123")
	
	// Verify service configuration
	assert.Contains(t, script, "Configure service: nginx")
	assert.Contains(t, script, "Configure service: redis")
	assert.Contains(t, script, "worker_processes auto;")
	assert.Contains(t, script, "systemctl enable nginx")
	assert.NotContains(t, script, "systemctl enable redis") // redis is disabled
	
	// Verify post-install script
	assert.Contains(t, script, "Custom post-install script")
	
	// Verify completion marker
	assert.Contains(t, script, "CloudWorkstation setup completed successfully")
	assert.Contains(t, script, "/var/log/cws-setup.log")
}

func TestScriptGenerator_GenerateScript_Conda(t *testing.T) {
	generator := NewScriptGenerator()
	
	template := &Template{
		Name:        "Conda Test Template",
		Description: "Template for Conda testing",
		Base:        "ubuntu-22.04",
		PackageManager: "conda",
		Packages: PackageDefinitions{
			Conda: []string{"numpy", "pandas", "matplotlib", "jupyter"},
			Pip:   []string{"seaborn", "plotly"},
		},
		Users: []UserConfig{
			{Name: "datascientist", Password: "auto-generated", Groups: []string{"users"}},
		},
		Services: []ServiceConfig{
			{Name: "jupyter", Port: 8888, Enable: true},
		},
	}
	
	script, err := generator.GenerateScript(template, PackageManagerConda)
	require.NoError(t, err)
	assert.NotEmpty(t, script)
	
	// Verify conda-specific content
	assert.Contains(t, script, "Installing Miniforge")
	assert.Contains(t, script, "miniforge/releases/latest/download")
	assert.Contains(t, script, "/opt/miniforge")
	assert.Contains(t, script, "conda init bash")
	assert.Contains(t, script, "conda install -y numpy pandas matplotlib jupyter")
	assert.Contains(t, script, "pip install \"${PIP_PACKAGES[@]}\"") // pip packages
	
	// Verify user conda setup
	assert.Contains(t, script, "Setup conda for user")
	assert.Contains(t, script, "sudo -u datascientist")
	assert.Contains(t, script, "export PATH=\"/opt/miniforge/bin:$PATH\"")
	
	// Verify cleanup
	assert.Contains(t, script, "conda clean -a -y")
}

func TestScriptGenerator_GenerateScript_Spack(t *testing.T) {
	generator := NewScriptGenerator()
	
	template := &Template{
		Name:        "Spack Test Template",
		Description: "Template for Spack testing",
		Base:        "ubuntu-22.04",
		PackageManager: "spack",
		Packages: PackageDefinitions{
			Spack: []string{"openmpi", "hdf5+mpi", "fftw~mpi"},
		},
		Users: []UserConfig{
			{Name: "hpcuser", Password: "auto-generated", Groups: []string{"hpc", "users"}},
		},
	}
	
	script, err := generator.GenerateScript(template, PackageManagerSpack)
	require.NoError(t, err)
	assert.NotEmpty(t, script)
	
	// Verify spack-specific content
	assert.Contains(t, script, "Installing Spack")
	assert.Contains(t, script, "git clone")
	assert.Contains(t, script, "spack/spack.git /opt/spack")
	assert.Contains(t, script, "releases/v0.21") // stable release
	assert.Contains(t, script, "spack compiler find")
	assert.Contains(t, script, "spack external find")
	
	// Verify package installation
	assert.Contains(t, script, "spack install openmpi")
	assert.Contains(t, script, "spack install hdf5+mpi")
	assert.Contains(t, script, "spack install fftw~mpi")
	
	// Verify environment creation
	assert.Contains(t, script, "spack env create default")
	assert.Contains(t, script, "spack env activate default")
	
	// Verify user spack setup
	assert.Contains(t, script, "Setup Spack for user")
	assert.Contains(t, script, "export SPACK_ROOT=/opt/spack")
	assert.Contains(t, script, ". $SPACK_ROOT/share/spack/setup-env.sh")
}

func TestScriptGenerator_GenerateScript_DNF(t *testing.T) {
	generator := NewScriptGenerator()
	
	template := &Template{
		Name:        "DNF Test Template",
		Description: "Template for DNF testing",
		Base:        "ubuntu-22.04", // Note: DNF template uses APT on Ubuntu
		PackageManager: "dnf",
		Packages: PackageDefinitions{
			System: []string{"gcc", "make", "kernel-devel"},
		},
	}
	
	script, err := generator.GenerateScript(template, PackageManagerDnf)
	require.NoError(t, err)
	assert.NotEmpty(t, script)
	
	// Verify DNF script uses APT-compatible mode
	assert.Contains(t, script, "Using package manager: dnf (APT-compatible mode for Ubuntu)")
	assert.Contains(t, script, "apt-get update") // Falls back to APT
	assert.Contains(t, script, "enterprise-style package management")
	assert.Contains(t, script, "Installing enterprise packages")
}

func TestScriptGenerator_GenerateScript_AMI(t *testing.T) {
	generator := NewScriptGenerator()
	
	template := &Template{
		Name:        "AMI Test Template",
		Description: "Template for AMI testing",
		Base:        "ami-based",
		PackageManager: "ami",
		AMIConfig: AMIConfig{
			UserDataScript: "echo 'Custom AMI setup'",
			SSHUser:       "ubuntu",
		},
		Users: []UserConfig{
			{Name: "amiuser", Password: "auto-generated"},
		},
		Services: []ServiceConfig{
			{Name: "custom-service", Port: 9000, Enable: true},
		},
		PostInstall: "systemctl status custom-service",
	}
	
	script, err := generator.GenerateScript(template, PackageManagerAMI)
	require.NoError(t, err)
	assert.NotEmpty(t, script)
	
	// Verify AMI-specific content
	assert.Contains(t, script, "pre-built AMI - minimal setup required")
	assert.Contains(t, script, "Custom AMI setup") // UserDataScript
	assert.Contains(t, script, "SSH User: ubuntu")
	assert.Contains(t, script, "Additional user created - Name: amiuser")
	assert.Contains(t, script, "systemctl status custom-service") // PostInstall
	assert.Contains(t, script, "Service available - custom-service on port 9000")
}

func TestScriptGenerator_selectPackagesForManager(t *testing.T) {
	generator := NewScriptGenerator()
	
	template := &Template{
		Packages: PackageDefinitions{
			System: []string{"git", "vim"},
			Conda:  []string{"numpy", "pandas"},
			Spack:  []string{"openmpi", "hdf5"},
			Pip:    []string{"requests", "flask"},
		},
	}
	
	tests := []struct {
		name            string
		packageManager  PackageManagerType
		expectedPackages []string
	}{
		{
			name:            "APT manager selects system packages",
			packageManager:  PackageManagerApt,
			expectedPackages: []string{"git", "vim"},
		},
		{
			name:            "DNF manager selects system packages",
			packageManager:  PackageManagerDnf,
			expectedPackages: []string{"git", "vim"},
		},
		{
			name:            "Conda manager selects conda packages only",
			packageManager:  PackageManagerConda,
			expectedPackages: []string{"numpy", "pandas"},
		},
		{
			name:            "Spack manager selects spack packages",
			packageManager:  PackageManagerSpack,
			expectedPackages: []string{"openmpi", "hdf5"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages := generator.selectPackagesForManager(template, tt.packageManager)
			assert.Equal(t, tt.expectedPackages, packages)
		})
	}
}

func TestScriptGenerator_prepareUsers(t *testing.T) {
	generator := NewScriptGenerator()
	
	users := []UserConfig{
		{Name: "user1", Password: "auto-generated", Groups: []string{"sudo"}, Shell: "/bin/bash"},
		{Name: "user2", Password: "custom123", Groups: []string{"users"}, Shell: "/bin/zsh"},
		{Name: "user3", Password: "", Groups: []string{}, Shell: ""}, // Defaults
	}
	
	userData := generator.prepareUsers(users)
	
	assert.Len(t, userData, 3)
	
	// First user - auto-generated password
	assert.Equal(t, "user1", userData[0].Name)
	assert.NotEmpty(t, userData[0].Password)
	assert.NotEqual(t, "auto-generated", userData[0].Password) // Should be replaced
	assert.Equal(t, []string{"sudo"}, userData[0].Groups)
	assert.Equal(t, "/bin/bash", userData[0].Shell)
	
	// Second user - custom password
	assert.Equal(t, "user2", userData[1].Name)
	assert.Equal(t, "custom123", userData[1].Password)
	assert.Equal(t, []string{"users"}, userData[1].Groups)
	assert.Equal(t, "/bin/zsh", userData[1].Shell)
	
	// Third user - empty password should be auto-generated
	assert.Equal(t, "user3", userData[2].Name)
	assert.NotEmpty(t, userData[2].Password)
	assert.NotEqual(t, "", userData[2].Password)
	assert.Empty(t, userData[2].Groups)
	assert.Empty(t, userData[2].Shell)
}

func TestScriptGenerator_generateSecurePassword(t *testing.T) {
	// Generate multiple passwords
	passwords := make([]string, 10)
	for i := 0; i < 10; i++ {
		passwords[i] = generateSecurePassword()
	}
	
	// Verify each password
	for _, password := range passwords {
		assert.Len(t, password, 16, "Password should be 16 characters")
		assert.Regexp(t, regexp.MustCompile("^[A-Za-z0-9_-]+$"), password, "Password should contain only base64 URL-safe characters")
	}
	
	// Verify passwords are unique (very high probability)
	seen := make(map[string]bool)
	for _, password := range passwords {
		assert.False(t, seen[password], "Passwords should be unique")
		seen[password] = true
	}
}

func TestScriptGenerator_GenerateScript_ErrorCases(t *testing.T) {
	generator := NewScriptGenerator()
	
	template := &Template{
		Name:        "Error Test Template",
		Description: "Template for error testing",
		Base:        "ubuntu-22.04",
	}
	
	// Test unsupported package manager
	_, err := generator.GenerateScript(template, PackageManagerType("unsupported"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported package manager: unsupported")
}

func TestScriptGenerator_ScriptValidation(t *testing.T) {
	generator := NewScriptGenerator()
	
	template := &Template{
		Name:        "Validation Template",
		Description: "Template for script validation",
		Base:        "ubuntu-22.04",
		PackageManager: "apt",
		Packages: PackageDefinitions{
			System: []string{"git", "curl"},
		},
		Users: []UserConfig{
			{Name: "testuser", Password: "testpass123", Groups: []string{"sudo"}},
		},
		Services: []ServiceConfig{
			{Name: "nginx", Port: 80, Enable: true},
		},
	}
	
	script, err := generator.GenerateScript(template, PackageManagerApt)
	require.NoError(t, err)
	
	// Verify script structure
	lines := strings.Split(script, "\n")
	assert.True(t, len(lines) > 10, "Script should have multiple lines")
	
	// Should start with shebang
	assert.Equal(t, "#!/bin/bash", lines[0])
	
	// Should have error handling
	assert.Contains(t, script, "set -euo pipefail")
	
	// Should have CloudWorkstation branding
	assert.Contains(t, script, "CloudWorkstation")
	
	// Should have logging
	assert.Contains(t, script, "/var/log/cws-setup.log")
	
	// Should have completion marker
	assert.Contains(t, script, "Setup Complete")
	
	// Should have user information output
	assert.Contains(t, script, "User created - Name: testuser, Password: testpass123")
	
	// Should have service information output
	assert.Contains(t, script, "Service available - nginx on port 80")
}

func TestScriptGenerator_PackageInstallationOrder(t *testing.T) {
	generator := NewScriptGenerator()
	
	template := &Template{
		Name:        "Order Test Template",
		Description: "Template for testing installation order",
		Base:        "ubuntu-22.04",
		PackageManager: "conda",
		Packages: PackageDefinitions{
			Conda: []string{"numpy", "pandas", "matplotlib"},
			Pip:   []string{"seaborn", "plotly"},
		},
	}
	
	script, err := generator.GenerateScript(template, PackageManagerConda)
	require.NoError(t, err)
	
	// Verify installation order
	condaIndex := strings.Index(script, "conda install")
	pipIndex := strings.Index(script, "pip install")
	
	assert.True(t, condaIndex < pipIndex, "Conda packages should be installed before pip packages")
}

func TestScriptGenerator_ServiceConfiguration(t *testing.T) {
	generator := NewScriptGenerator()
	
	template := &Template{
		Name:        "Service Test Template",
		Description: "Template for testing service configuration",
		Base:        "ubuntu-22.04",
		PackageManager: "apt",
		Services: []ServiceConfig{
			{
				Name:   "nginx",
				Port:   80,
				Enable: true,
				Config: []string{
					"worker_processes auto;",
					"worker_connections 1024;",
				},
			},
			{
				Name:   "disabled-service",
				Port:   9999,
				Enable: false,
			},
		},
	}
	
	script, err := generator.GenerateScript(template, PackageManagerApt)
	require.NoError(t, err)
	
	// Verify service configuration
	assert.Contains(t, script, "Configure service: nginx")
	assert.Contains(t, script, "mkdir -p /etc/nginx")
	assert.Contains(t, script, "worker_processes auto;")
	assert.Contains(t, script, "worker_connections 1024;")
	assert.Contains(t, script, "systemctl enable nginx")
	assert.Contains(t, script, "systemctl start nginx")
	
	// Verify disabled service handling
	assert.Contains(t, script, "Configure service: disabled-service")
	assert.NotContains(t, script, "systemctl enable disabled-service")
	assert.NotContains(t, script, "systemctl start disabled-service")
}

func TestScriptGenerator_EdgeCases(t *testing.T) {
	generator := NewScriptGenerator()
	
	// Test template with no packages
	emptyTemplate := &Template{
		Name:        "Empty Template",
		Description: "Template with no packages",
		Base:        "ubuntu-22.04",
		PackageManager: "apt",
	}
	
	script, err := generator.GenerateScript(emptyTemplate, PackageManagerApt)
	require.NoError(t, err)
	assert.NotEmpty(t, script)
	assert.Contains(t, script, "Empty Template")
	assert.NotContains(t, script, "Installing template packages") // No packages section
	
	// Test template with only pip packages for conda
	condaTemplate := &Template{
		Name:        "Pip Only Template",
		Description: "Template with only pip packages",
		Base:        "ubuntu-22.04",
		PackageManager: "conda",
		Packages: PackageDefinitions{
			Pip: []string{"requests", "flask"},
		},
	}
	
	script2, err := generator.GenerateScript(condaTemplate, PackageManagerConda)
	require.NoError(t, err)
	assert.Contains(t, script2, "pip install \"${PIP_PACKAGES[@]}\"")
	// Should not have conda install section since no conda packages
	assert.NotContains(t, script2, "Installing conda packages")
}