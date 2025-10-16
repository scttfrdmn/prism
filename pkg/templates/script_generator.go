package templates

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/scttfrdmn/cloudworkstation/pkg/security"
)

// NewScriptGenerator creates a new script generator
func NewScriptGenerator() *ScriptGenerator {
	return &ScriptGenerator{
		AptTemplate:   aptScriptTemplate,
		DnfTemplate:   dnfScriptTemplate,
		CondaTemplate: condaScriptTemplate,
		SpackTemplate: spackScriptTemplate,
		AMITemplate:   amiScriptTemplate,
		PipTemplate:   pipScriptTemplate,
	}
}

// GenerateScript generates an installation script for a template
func (sg *ScriptGenerator) GenerateScript(tmpl *Template, packageManager PackageManagerType) (string, error) {
	// Prepare script data
	scriptData := &ScriptData{
		Template:           tmpl,
		PackageManager:     string(packageManager),
		Packages:           sg.selectPackagesForManager(tmpl, packageManager),
		Users:              sg.prepareUsers(tmpl.Users),
		Services:           tmpl.Services,
		WebInterfaceBindIP: security.GetWebInterfaceBindIP(),
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
	case PackageManagerPip:
		scriptTemplate = sg.PipTemplate
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
	Template           *Template
	PackageManager     string
	Packages           []string
	Users              []UserData
	Services           []ServiceConfig
	WebInterfaceBindIP string // Dynamic IP binding for web interfaces (0.0.0.0 or 127.0.0.1)
}

// UserData contains processed user data for script generation
type UserData struct {
	Name   string
	Groups []string
	Shell  string
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
	case PackageManagerPip:
		return tmpl.Packages.Pip
	default:
		return tmpl.Packages.System
	}
}

// prepareUsers processes user configurations and generates passwords
func (sg *ScriptGenerator) prepareUsers(users []UserConfig) []UserData {
	userData := make([]UserData, len(users))

	for i, user := range users {
		userData[i] = UserData(user)

		// SSH key authentication only - no passwords needed
	}

	return userData
}

// Password generation removed - using SSH key authentication only

// Script templates for different package managers

const dnfScriptTemplate = `#!/bin/bash
set -euo pipefail

# CloudWorkstation Template: {{.Template.Name}}
# Generated script using dnf package manager (RHEL/Rocky/Fedora)
# Generated at: $(date)

echo "=== CloudWorkstation Setup: {{.Template.Name}} ==="
echo "Using package manager: {{.PackageManager}} (DNF for RHEL-based systems)"

# Update system using DNF
echo "Updating system packages with DNF..."
dnf check-update -y || true  # Non-zero exit if updates available
dnf upgrade -y

# Enable EPEL repository for additional packages (RHEL/Rocky)
echo "Enabling EPEL repository..."
dnf install -y epel-release || true

# Install base requirements for RHEL-based systems
echo "Installing base requirements..."
dnf install -y curl wget ca-certificates gcc gcc-c++ make git

# Install development tools group
dnf groupinstall -y "Development Tools" || true

{{if .Packages}}
# Install template packages
echo "Installing template packages with DNF..."
dnf install -y{{range .Packages}} {{.}}{{end}}
{{end}}

{{range .Users}}
# Create user: {{.Name}}
echo "Creating user: {{.Name}}"
{{if .Shell}}useradd -m -s {{.Shell}} {{.Name}} || true{{else}}useradd -m -s /bin/bash {{.Name}} || true{{end}}
# SSH key authentication configured - no password needed
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
echo "User created - Name: {{.Name}} (SSH key authentication)"
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
# SSH key authentication configured - no password needed
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
echo "User created - Name: {{.Name}} (SSH key authentication)"
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

# CloudWorkstation Progress Monitoring
# This script logs progress markers that can be monitored via SSH
PROGRESS_LOG="/var/log/cws-setup.log"
touch "$PROGRESS_LOG"
chmod 644 "$PROGRESS_LOG"

# Progress marker function
progress() {
    echo "[CWS-PROGRESS] $1" | tee -a "$PROGRESS_LOG"
    logger -t cws-setup "$1"
}

progress "STAGE:init:START"

# System initialization
apt-get update -y && apt-get install -y curl wget bzip2 ca-certificates

progress "STAGE:init:COMPLETE"
progress "STAGE:system-packages:START"

# Install miniforge
ARCH=$(uname -m)
MINIFORGE_URL="https://github.com/conda-forge/miniforge/releases/latest/download/Miniforge3-Linux-${ARCH}.sh"
wget -O /tmp/mf.sh "$MINIFORGE_URL" && bash /tmp/mf.sh -b -p /opt/miniforge && rm /tmp/mf.sh
export PATH="/opt/miniforge/bin:$PATH"
echo 'export PATH="/opt/miniforge/bin:$PATH"' >> /etc/environment
/opt/miniforge/bin/conda init bash

progress "STAGE:system-packages:COMPLETE"
progress "STAGE:conda-packages:START"

{{if .Packages}}/opt/miniforge/bin/conda install -y{{range .Packages}} {{.}}{{end}}{{end}}

progress "STAGE:conda-packages:COMPLETE"
progress "STAGE:pip-packages:START"

{{if .Template.Packages.Pip}}/opt/miniforge/bin/pip install{{range .Template.Packages.Pip}} {{.}}{{end}}{{end}}

progress "STAGE:pip-packages:COMPLETE"
progress "STAGE:service-config:START"

{{range .Users}}useradd -m -s /bin/bash {{.Name}} || true
{{if .Groups}}{{$user := .}}{{range .Groups}}usermod -aG {{.}} {{$user.Name}}{{end}}{{end}}
sudo -u {{.Name}} /opt/miniforge/bin/conda init bash
echo 'export PATH="/opt/miniforge/bin:$PATH"' >> /home/{{.Name}}/.bashrc
chown -R {{.Name}}:{{.Name}} /home/{{.Name}}
{{end}}

{{range .Services}}{{if eq .Name "jupyter"}}# Generate Jupyter config for researcher user
sudo -u {{range $.Users}}{{if eq .Name "researcher"}}{{.Name}}{{end}}{{end}} /opt/miniforge/bin/jupyter lab --generate-config -y

# Configure Jupyter for no-token access (safe for SSH tunnel usage)
JUPYTER_CONFIG="/home/{{range $.Users}}{{if eq .Name "researcher"}}{{.Name}}{{end}}{{end}}/.jupyter/jupyter_lab_config.py"
cat >> "$JUPYTER_CONFIG" << 'JUPYTEREOF'

# CloudWorkstation: Disable token for SSH tunnel access
c.ServerApp.token = ''
c.ServerApp.password = ''
c.ServerApp.disable_check_xsrf = False
JUPYTEREOF
chown {{range $.Users}}{{if eq .Name "researcher"}}{{.Name}}:{{.Name}}{{end}}{{end}} "$JUPYTER_CONFIG"

# Create Jupyter systemd service
cat > /etc/systemd/system/jupyter.service << 'EOF'
[Unit]
Description=Jupyter Lab
After=network.target
[Service]
Type=simple
User={{range $.Users}}{{if eq .Name "researcher"}}{{.Name}}{{end}}{{end}}
Environment=PATH=/opt/miniforge/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
WorkingDirectory={{range $.Users}}{{if eq .Name "researcher"}}/home/{{.Name}}{{end}}{{end}}
ExecStart=/opt/miniforge/bin/jupyter lab --ip=127.0.0.1 --port={{.Port}} --no-browser
Restart=always
[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
{{if .Enable}}systemctl enable jupyter && systemctl start jupyter{{end}}{{end}}{{if eq .Name "rstudio-server"}}# Install RStudio Server
wget -q https://download2.rstudio.org/server/jammy/amd64/rstudio-server-2024.12.0-467-amd64.deb -O /tmp/rstudio-server.deb
apt-get install -y gdebi-core
gdebi -n /tmp/rstudio-server.deb
rm /tmp/rstudio-server.deb

# Configure RStudio Server
mkdir -p /etc/rstudio
cat > /etc/rstudio/rserver.conf << 'EOF'
# RStudio Server Configuration
www-port={{.Port}}
www-address=127.0.0.1
rsession-which-r=/opt/miniforge/bin/R
rsession-ld-library-path=/opt/miniforge/lib
EOF

# Configure R session
cat > /etc/rstudio/rsession.conf << 'EOF'
# R Session Configuration
r-libs-user=~/R/library
session-timeout-minutes=0
EOF

# Create rstudio-users group and add R user
groupadd -f rstudio-users
{{range $.Users}}{{if or (eq .Name "rstats") (eq .Name "researcher")}}usermod -aG rstudio-users {{.Name}}
{{end}}{{end}}
# Restart RStudio Server
systemctl daemon-reload
{{if .Enable}}systemctl enable rstudio-server && systemctl restart rstudio-server{{end}}
{{end}}{{if eq .Name "shiny-server"}}
# Install Shiny Server
wget -q https://download3.rstudio.org/ubuntu-18.04/x86_64/shiny-server-1.5.22.1017-amd64.deb -O /tmp/shiny-server.deb
gdebi -n /tmp/shiny-server.deb
rm /tmp/shiny-server.deb

# Install Shiny R package via conda
/opt/miniforge/bin/R -e "install.packages('shiny', repos='https://cloud.r-project.org')"

# Configure Shiny Server
cat > /etc/shiny-server/shiny-server.conf << 'EOF'
# Shiny Server Configuration
run_as shiny;
server {
  listen {{.Port}} 127.0.0.1;
  location / {
    site_dir /srv/shiny-server;
    log_dir /var/log/shiny-server;
    directory_index on;
  }
}
EOF

# Create shared Shiny apps directory accessible by R users
mkdir -p /srv/shiny-server
chmod 755 /srv/shiny-server
{{range $.Users}}{{if or (eq .Name "rstats") (eq .Name "researcher")}}chown -R {{.Name}}:shiny /srv/shiny-server
{{end}}{{end}}
# Restart Shiny Server
systemctl daemon-reload
{{if .Enable}}systemctl enable shiny-server && systemctl restart shiny-server{{end}}{{end}}{{end}}

progress "STAGE:service-config:COMPLETE"
progress "STAGE:ready:START"

# Cleanup
/opt/miniforge/bin/conda clean -a -y && apt-get autoremove -y && apt-get autoclean

progress "STAGE:ready:COMPLETE"
progress "SETUP:COMPLETE:All setup tasks finished successfully"

# Final completion marker
echo "CloudWorkstation setup completed at $(date)" >> "$PROGRESS_LOG"
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
# SSH key authentication configured - no password needed
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
echo "User created - Name: {{.Name}} (SSH key authentication)"
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
# SSH key authentication configured - no password needed
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
echo "Additional user created - Name: {{.Name}} (SSH key authentication)"
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

const pipScriptTemplate = `#!/bin/bash
set -euo pipefail

# CloudWorkstation Template: {{.Template.Name}}
# Generated script using pip package manager
# Generated at: $(date)

echo "=== CloudWorkstation Setup: {{.Template.Name}} ==="
echo "Using package manager: {{.PackageManager}}"

# System update (assumes Ubuntu/Debian for system packages)
echo "Updating system packages..."
apt-get update -y
apt-get upgrade -y

# Install base requirements including Python and pip
echo "Installing base requirements..."
apt-get install -y curl wget software-properties-common build-essential python3 python3-pip python3-venv

# Install system packages if defined
{{if .Template.Packages.System}}
echo "Installing system packages..."
apt-get install -y{{range .Template.Packages.System}} {{.}}{{end}}
{{end}}

{{if .Packages}}
# Install pip packages
echo "Installing pip packages..."
pip3 install{{range .Packages}} {{.}}{{end}}
{{end}}

{{range .Users}}
# Create user: {{.Name}}
echo "Creating user: {{.Name}}"
{{if .Shell}}useradd -m -s {{.Shell}} {{.Name}} || true{{else}}useradd -m -s /bin/bash {{.Name}} || true{{end}}
# SSH key authentication configured - no password needed
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

# Final cleanup
apt-get autoremove -y
apt-get autoclean

echo "=== CloudWorkstation Setup Complete ==="
echo "Template: {{.Template.Name}}"
echo "Description: {{.Template.Description}}"
echo "Pip packages installed"
{{range .Users}}
echo "User created - Name: {{.Name}} (SSH key authentication)"
{{end}}
{{range .Services}}
{{if .Port}}
echo "Service available - {{.Name}} on port {{.Port}}"
{{end}}
{{end}}
echo "Setup log: /var/log/cws-setup.log"

# Write completion marker
date > /var/log/cws-setup.log
echo "CloudWorkstation pip setup completed successfully" >> /var/log/cws-setup.log
`
