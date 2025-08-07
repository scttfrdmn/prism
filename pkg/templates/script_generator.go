package templates

import (
	"fmt"
	"text/template"
	"bytes"
	"crypto/rand"
	"encoding/base64"
)

// NewScriptGenerator creates a new script generator
func NewScriptGenerator() *ScriptGenerator {
	return &ScriptGenerator{
		AptTemplate:   aptScriptTemplate,
		DnfTemplate:   dnfScriptTemplate,
		CondaTemplate: condaScriptTemplate,
		SpackTemplate: spackScriptTemplate,
		AMITemplate:   amiScriptTemplate,
	}
}

// GenerateScript generates an installation script for a template
func (sg *ScriptGenerator) GenerateScript(tmpl *Template, packageManager PackageManagerType) (string, error) {
	// Prepare script data
	scriptData := &ScriptData{
		Template:       tmpl,
		PackageManager: string(packageManager),
		Packages:       sg.selectPackagesForManager(tmpl, packageManager),
		Users:          sg.prepareUsers(tmpl.Users),
		Services:       tmpl.Services,
	}
	
	// Select appropriate template
	var scriptTemplate string
	switch packageManager {
	case PackageManagerApt:
		scriptTemplate = sg.AptTemplate
	case PackageManagerDnf:
		scriptTemplate = sg.DnfTemplate
	case PackageManagerConda:
		scriptTemplate = sg.CondaTemplate
	case PackageManagerSpack:
		scriptTemplate = sg.SpackTemplate
	case PackageManagerAMI:
		// AMI templates use minimal user data (AMI is already configured)
		scriptTemplate = sg.AMITemplate
	default:
		return "", fmt.Errorf("unsupported package manager: %s", packageManager)
	}
	
	// Parse and execute template
	tmplObj, err := template.New("script").Parse(scriptTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse script template: %w", err)
	}
	
	var buf bytes.Buffer
	if err := tmplObj.Execute(&buf, scriptData); err != nil {
		return "", fmt.Errorf("failed to execute script template: %w", err)
	}
	
	return buf.String(), nil
}

// ScriptData contains data for script template execution
type ScriptData struct {
	Template       *Template
	PackageManager string
	Packages       []string
	Users          []UserData
	Services       []ServiceConfig
}

// UserData contains processed user data for script generation
type UserData struct {
	Name     string
	Password string
	Groups   []string
	Shell    string
}

// selectPackagesForManager selects appropriate packages for the given package manager
func (sg *ScriptGenerator) selectPackagesForManager(tmpl *Template, pm PackageManagerType) []string {
	switch pm {
	case PackageManagerApt:
		return tmpl.Packages.System
	case PackageManagerDnf:
		return tmpl.Packages.System
	case PackageManagerConda:
		// Only return conda packages, pip packages are handled separately in the template
		return tmpl.Packages.Conda
	case PackageManagerSpack:
		return tmpl.Packages.Spack
	default:
		return tmpl.Packages.System
	}
}

// prepareUsers processes user configurations and generates passwords
func (sg *ScriptGenerator) prepareUsers(users []UserConfig) []UserData {
	userData := make([]UserData, len(users))
	
	for i, user := range users {
		userData[i] = UserData{
			Name:   user.Name,
			Groups: user.Groups,
			Shell:  user.Shell,
		}
		
		// Generate secure password if auto-generated
		if user.Password == "auto-generated" || user.Password == "" {
			userData[i].Password = generateSecurePassword()
		} else {
			userData[i].Password = user.Password
		}
	}
	
	return userData
}

// generateSecurePassword generates a secure random password
func generateSecurePassword() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:16] // 16 char password
}

// Script templates for different package managers

const dnfScriptTemplate = `#!/bin/bash
set -euo pipefail

# CloudWorkstation Template: {{.Template.Name}}
# Generated script using dnf package manager (APT-compatible mode)
# Generated at: $(date)

echo "=== CloudWorkstation Setup: {{.Template.Name}} ==="
echo "Using package manager: {{.PackageManager}} (APT-compatible mode for Ubuntu)"

# Note: In production, this would use actual DNF on RHEL/Rocky/Fedora
echo "Setting up enterprise-style package management..."
apt-get update -y
apt-get upgrade -y

# Install base requirements
echo "Installing base requirements..."
apt-get install -y curl wget software-properties-common build-essential

{{if .Packages}}
# Install template packages (enterprise-focused)
echo "Installing enterprise packages..."
apt-get install -y{{range .Packages}} {{.}}{{end}}
{{end}}

{{range .Users}}
# Create user: {{.Name}}
echo "Creating user: {{.Name}}"
{{if .Shell}}useradd -m -s {{.Shell}} {{.Name}} || true{{else}}useradd -m -s /bin/bash {{.Name}} || true{{end}}
echo "{{.Name}}:{{.Password}}" | chpasswd
{{if .Groups}}
{{$user := .}}{{range .Groups}}usermod -aG {{.}} {{$user.Name}}
{{end}}
{{end}}
{{end}}

{{range .Services}}
# Configure service: {{.Name}}
echo "Configuring service: {{.Name}}"
{{if .Config}}
mkdir -p /etc/{{.Name}}
{{$service := .}}{{range .Config}}
echo "{{.}}" >> /etc/{{$service.Name}}/{{$service.Name}}.conf
{{end}}
{{end}}
{{if .Enable}}
systemctl enable {{.Name}} || true
systemctl start {{.Name}} || true
{{end}}
{{end}}

{{if .Template.PostInstall}}
# Post-install script
echo "Running post-install script..."
{{.Template.PostInstall}}
{{end}}

# Cleanup
echo "Cleaning up..."
apt-get autoremove -y
apt-get autoclean

echo "=== Setup Complete ==="
echo "Template: {{.Template.Name}}"
echo "Description: {{.Template.Description}}"
{{range .Users}}
echo "User created - Name: {{.Name}}, Password: {{.Password}}"
{{end}}
{{range .Services}}
{{if .Port}}
echo "Service available - {{.Name}} on port {{.Port}}"
{{end}}
{{end}}
echo "Setup log: /var/log/cws-setup.log"

# Write completion marker
date > /var/log/cws-setup.log
echo "CloudWorkstation setup completed successfully" >> /var/log/cws-setup.log
`

const aptScriptTemplate = `#!/bin/bash
set -euo pipefail

# CloudWorkstation Template: {{.Template.Name}}
# Generated script using apt package manager
# Generated at: $(date)

echo "=== CloudWorkstation Setup: {{.Template.Name}} ==="
echo "Using package manager: {{.PackageManager}}"

# System update
echo "Updating system packages..."
apt-get update -y
apt-get upgrade -y

# Install base requirements
echo "Installing base requirements..."
apt-get install -y curl wget software-properties-common build-essential

{{if .Packages}}
# Install template packages
echo "Installing template packages..."
apt-get install -y{{range .Packages}} {{.}}{{end}}
{{end}}

{{range .Users}}
# Create user: {{.Name}}
echo "Creating user: {{.Name}}"
{{if .Shell}}useradd -m -s {{.Shell}} {{.Name}} || true{{else}}useradd -m -s /bin/bash {{.Name}} || true{{end}}
echo "{{.Name}}:{{.Password}}" | chpasswd
{{if .Groups}}
{{$user := .}}{{range .Groups}}usermod -aG {{.}} {{$user.Name}}
{{end}}
{{end}}
{{end}}

{{range .Services}}
# Configure service: {{.Name}}
echo "Configuring service: {{.Name}}"
{{if .Config}}
mkdir -p /etc/{{.Name}}
{{$service := .}}{{range .Config}}
echo "{{.}}" >> /etc/{{$service.Name}}/{{$service.Name}}.conf
{{end}}
{{end}}
{{if .Enable}}
systemctl enable {{.Name}} || true
systemctl start {{.Name}} || true
{{end}}
{{end}}

{{if .Template.PostInstall}}
# Post-install script
echo "Running post-install script..."
{{.Template.PostInstall}}
{{end}}

# Cleanup
echo "Cleaning up..."
apt-get autoremove -y
apt-get autoclean

echo "=== Setup Complete ==="
echo "Template: {{.Template.Name}}"
echo "Description: {{.Template.Description}}"
{{range .Users}}
echo "User created - Name: {{.Name}}, Password: {{.Password}}"
{{end}}
{{range .Services}}
{{if .Port}}
echo "Service available - {{.Name}} on port {{.Port}}"
{{end}}
{{end}}
echo "Setup log: /var/log/cws-setup.log"

# Write completion marker
date > /var/log/cws-setup.log
echo "CloudWorkstation setup completed successfully" >> /var/log/cws-setup.log
`

const condaScriptTemplate = `#!/bin/bash
set -euo pipefail

# CloudWorkstation Template: {{.Template.Name}}
# Generated script using conda package manager
# Generated at: $(date)

echo "=== CloudWorkstation Setup: {{.Template.Name}} ==="
echo "Using package manager: {{.PackageManager}}"

# System update
echo "Updating system packages..."
apt-get update -y
apt-get upgrade -y

# Install base requirements
echo "Installing base requirements..."
apt-get install -y curl wget bzip2 ca-certificates

# Install Miniforge (conda alternative)
echo "Installing Miniforge..."
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    MINIFORGE_URL="https://github.com/conda-forge/miniforge/releases/latest/download/Miniforge3-Linux-x86_64.sh"
elif [ "$ARCH" = "aarch64" ]; then
    MINIFORGE_URL="https://github.com/conda-forge/miniforge/releases/latest/download/Miniforge3-Linux-aarch64.sh"
else
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

wget -O /tmp/miniforge.sh "$MINIFORGE_URL"
bash /tmp/miniforge.sh -b -p /opt/miniforge
rm /tmp/miniforge.sh

# Add conda to PATH
echo 'export PATH="/opt/miniforge/bin:$PATH"' >> /etc/environment
export PATH="/opt/miniforge/bin:$PATH"

# Initialize conda for all users
/opt/miniforge/bin/conda init bash

{{if .Packages}}
# Install template packages
echo "Installing conda packages..."
/opt/miniforge/bin/conda install -y{{range .Packages}} {{.}}{{end}}
{{end}}

# Install pip packages if any were specified
PIP_PACKAGES=({{range .Template.Packages.Pip}}"{{.}}" {{end}})
if [ ${#PIP_PACKAGES[@]} -gt 0 ]; then
    echo "Installing pip packages..."
    /opt/miniforge/bin/pip install "${PIP_PACKAGES[@]}"
fi

{{range .Users}}
# Create user: {{.Name}}
echo "Creating user: {{.Name}}"
{{if .Shell}}useradd -m -s {{.Shell}} {{.Name}} || true{{else}}useradd -m -s /bin/bash {{.Name}} || true{{end}}
echo "{{.Name}}:{{.Password}}" | chpasswd
{{if .Groups}}
{{$user := .}}{{range .Groups}}usermod -aG {{.}} {{$user.Name}}
{{end}}
{{end}}

# Setup conda for user
sudo -u {{.Name}} /opt/miniforge/bin/conda init bash
echo 'export PATH="/opt/miniforge/bin:$PATH"' >> /home/{{.Name}}/.bashrc
chown -R {{.Name}}:{{.Name}} /home/{{.Name}}
{{end}}

{{range .Services}}
# Configure service: {{.Name}}
echo "Configuring service: {{.Name}}"
{{if .Config}}
mkdir -p /etc/{{.Name}}
{{$service := .}}{{range .Config}}
echo "{{.}}" >> /etc/{{$service.Name}}/{{$service.Name}}.conf
{{end}}
{{end}}
{{if .Enable}}
systemctl enable {{.Name}} || true
systemctl start {{.Name}} || true
{{end}}
{{end}}

# Cleanup
echo "Cleaning up..."
/opt/miniforge/bin/conda clean -a -y
apt-get autoremove -y
apt-get autoclean

echo "=== Setup Complete ==="
echo "Template: {{.Template.Name}}"
echo "Description: {{.Template.Description}}"
echo "Conda environment: /opt/miniforge"
{{range .Users}}
echo "User created - Name: {{.Name}}, Password: {{.Password}}"
{{end}}
{{range .Services}}
{{if .Port}}
echo "Service available - {{.Name}} on port {{.Port}}"
{{end}}
{{end}}
echo "Setup log: /var/log/cws-setup.log"

# Write completion marker
date > /var/log/cws-setup.log
echo "CloudWorkstation setup completed successfully" >> /var/log/cws-setup.log
`

const spackScriptTemplate = `#!/bin/bash
set -euo pipefail

# CloudWorkstation Template: {{.Template.Name}}
# Generated script using spack package manager
# Generated at: $(date)

echo "=== CloudWorkstation Setup: {{.Template.Name}} ==="
echo "Using package manager: {{.PackageManager}}"

# System update
echo "Updating system packages..."
apt-get update -y
apt-get upgrade -y

# Install Spack dependencies
echo "Installing Spack dependencies..."
apt-get install -y build-essential ca-certificates coreutils curl environment-modules gfortran git gpg lsb-release python3 python3-distutils python3-venv unzip zip

# Install Spack
echo "Installing Spack..."
git clone -c feature.manyFiles=true https://github.com/spack/spack.git /opt/spack
cd /opt/spack
git checkout releases/v0.21  # Use stable release

# Setup Spack environment
echo 'export SPACK_ROOT=/opt/spack' >> /etc/environment
echo 'export PATH="$SPACK_ROOT/bin:$PATH"' >> /etc/environment
echo '. $SPACK_ROOT/share/spack/setup-env.sh' >> /etc/bash.bashrc
export SPACK_ROOT=/opt/spack
export PATH="$SPACK_ROOT/bin:$PATH"
source $SPACK_ROOT/share/spack/setup-env.sh

# Configure Spack
echo "Configuring Spack..."
spack compiler find
spack external find

{{if .Packages}}
# Install template packages
echo "Installing Spack packages..."
{{range .Packages}}
echo "Installing: {{.}}"
spack install {{.}}
{{end}}

# Load packages by default
echo "Creating default environment..."
spack env create default
spack env activate default
{{range .Packages}}
spack add {{.}}
{{end}}
spack concretize
spack install
{{end}}

{{range .Users}}
# Create user: {{.Name}}
echo "Creating user: {{.Name}}"
{{if .Shell}}useradd -m -s {{.Shell}} {{.Name}} || true{{else}}useradd -m -s /bin/bash {{.Name}} || true{{end}}
echo "{{.Name}}:{{.Password}}" | chpasswd
{{if .Groups}}
{{$user := .}}{{range .Groups}}usermod -aG {{.}} {{$user.Name}}
{{end}}
{{end}}

# Setup Spack for user
echo 'export SPACK_ROOT=/opt/spack' >> /home/{{.Name}}/.bashrc
echo 'export PATH="$SPACK_ROOT/bin:$PATH"' >> /home/{{.Name}}/.bashrc
echo '. $SPACK_ROOT/share/spack/setup-env.sh' >> /home/{{.Name}}/.bashrc
{{if $.Packages}}
echo 'spack env activate default' >> /home/{{.Name}}/.bashrc
{{end}}
chown -R {{.Name}}:{{.Name}} /home/{{.Name}}
{{end}}

{{range .Services}}
# Configure service: {{.Name}}
echo "Configuring service: {{.Name}}"
{{if .Config}}
mkdir -p /etc/{{.Name}}
{{$service := .}}{{range .Config}}
echo "{{.}}" >> /etc/{{$service.Name}}/{{$service.Name}}.conf
{{end}}
{{end}}
{{if .Enable}}
systemctl enable {{.Name}} || true
systemctl start {{.Name}} || true
{{end}}
{{end}}

# Set proper permissions
chown -R root:root /opt/spack
chmod -R go+rX /opt/spack

echo "=== Setup Complete ==="
echo "Template: {{.Template.Name}}"
echo "Description: {{.Template.Description}}"
echo "Spack root: /opt/spack"
{{if .Packages}}
echo "Default environment: spack env activate default"
{{end}}
{{range .Users}}
echo "User created - Name: {{.Name}}, Password: {{.Password}}"
{{end}}
{{range .Services}}
{{if .Port}}
echo "Service available - {{.Name}} on port {{.Port}}"
{{end}}
{{end}}
echo "Setup log: /var/log/cws-setup.log"

# Write completion marker
date > /var/log/cws-setup.log
echo "CloudWorkstation setup completed successfully" >> /var/log/cws-setup.log
`

const amiScriptTemplate = `#\!/bin/bash
set -euo pipefail

# CloudWorkstation Template: {{.Template.Name}}
# Generated script for AMI-based template (minimal user data)
# Generated at: $(date)

echo "=== CloudWorkstation Setup: {{.Template.Name}} ==="
echo "Using pre-built AMI - minimal setup required"

{{if .Template.AMIConfig.UserDataScript}}
# Custom user data script from AMI config
echo "Running custom AMI user data script..."
{{.Template.AMIConfig.UserDataScript}}
{{end}}

{{range .Users}}
# Create user: {{.Name}}
echo "Creating user: {{.Name}}"
{{if .Shell}}useradd -m -s {{.Shell}} {{.Name}} || true{{else}}useradd -m -s /bin/bash {{.Name}} || true{{end}}
echo "{{.Name}}:{{.Password}}" | chpasswd
{{if .Groups}}
{{$user := .}}{{range .Groups}}usermod -aG {{.}} {{$user.Name}}
{{end}}
{{end}}
{{end}}

{{range .Services}}
# Configure service: {{.Name}}
echo "Configuring service: {{.Name}}"
{{if .Config}}
mkdir -p /etc/{{.Name}}
{{$service := .}}{{range .Config}}
echo "{{.}}" >> /etc/{{$service.Name}}/{{$service.Name}}.conf
{{end}}
{{end}}
{{if .Enable}}
systemctl enable {{.Name}} || true
systemctl start {{.Name}} || true
{{end}}
{{end}}

{{if .Template.PostInstall}}
# Post-install script
echo "Running post-install script..."
{{.Template.PostInstall}}
{{end}}

echo "=== Setup Complete ==="
echo "Template: {{.Template.Name}}"
echo "Description: {{.Template.Description}}"
echo "AMI-based template - most software pre-installed"
{{if .Template.AMIConfig.SSHUser}}
echo "SSH User: {{.Template.AMIConfig.SSHUser}}"
{{end}}
{{range .Users}}
echo "Additional user created - Name: {{.Name}}, Password: {{.Password}}"
{{end}}
{{range .Services}}
{{if .Port}}
echo "Service available - {{.Name}} on port {{.Port}}"
{{end}}
{{end}}
echo "Setup log: /var/log/cws-setup.log"

# Write completion marker
date > /var/log/cws-setup.log
echo "CloudWorkstation AMI setup completed successfully" >> /var/log/cws-setup.log
`