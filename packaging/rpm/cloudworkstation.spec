# CloudWorkstation RPM Package Specification
# Professional Enterprise Linux Distribution

%define _topdir %(pwd)/packaging/rpm
%define _builddir %{_topdir}/BUILD
%define _rpmdir %{_topdir}/RPMS
%define _sourcedir %{_topdir}/sources
%define _specdir %{_topdir}
%define _srcrpmdir %{_topdir}/SRPMS
%define _tmppath %{_topdir}/tmp

# Package metadata
Name:           cloudworkstation
Version:        0.5.1
Release:        1%{?dist}
Summary:        Autonomous Research Instance Management Platform
License:        MIT
URL:            https://github.com/scttfrdmn/prism
Source0:        %{name}-%{version}.tar.gz

# Build requirements
BuildRequires:  golang >= 1.20
BuildRequires:  systemd-rpm-macros
BuildRequires:  make

# Runtime requirements
Requires:       systemd
Requires:       awscli2
Requires(pre):  shadow-utils
Requires(post): systemd
Requires(preun): systemd
Requires(postun): systemd

# Package information
Group:          Applications/System
Vendor:         CloudWorkstation Project
Packager:       CloudWorkstation Team

# Description
%description
CloudWorkstation is a command-line tool that allows academic researchers
to launch pre-configured cloud workstations in seconds rather than spending
hours setting up research environments.

Features:
- Pre-configured templates for Python ML, R research, and more
- Automated cost optimization with hibernation support
- Multi-modal access: CLI, TUI, and GUI interfaces
- Project-based budget management
- Enterprise-grade security and compliance
- Template inheritance and customization

%description -l es
CloudWorkstation es una herramienta de línea de comandos que permite a los
investigadores académicos lanzar estaciones de trabajo en la nube preconfiguradas
en segundos en lugar de pasar horas configurando entornos de investigación.

# Architecture-specific package generation
%ifarch x86_64
%define go_arch amd64
%endif
%ifarch aarch64
%define go_arch arm64
%endif

# Preparation phase
%prep
%setup -q

# Build phase
%build
# Set Go environment
export GOOS=linux
export GOARCH=%{go_arch}
export CGO_ENABLED=0

# Build flags with version information
LDFLAGS="-ldflags \"-X github.com/scttfrdmn/prism/pkg/version.Version=%{version} \
                  -X github.com/scttfrdmn/prism/pkg/version.BuildDate=$(date -u '+%%Y-%%m-%%d_%%H:%%M:%%S') \
                  -X github.com/scttfrdmn/prism/pkg/version.GitCommit=rpm-build\""

# Build core binaries
make build-cli build-daemon LDFLAGS="$LDFLAGS"

# Installation phase
%install
# Create directory structure
mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{_unitdir}
mkdir -p %{buildroot}%{_sysconfdir}/cloudworkstation
mkdir -p %{buildroot}%{_sysconfdir}/cloudworkstation/aws
mkdir -p %{buildroot}%{_sharedstatedir}/cloudworkstation
mkdir -p %{buildroot}%{_localstatedir}/log/cloudworkstation
mkdir -p %{buildroot}%{_docdir}/%{name}
mkdir -p %{buildroot}%{_mandir}/man1

# Install binaries
install -m 755 bin/cws %{buildroot}%{_bindir}/cws
install -m 755 bin/cwsd %{buildroot}%{_bindir}/cwsd

# Install systemd service
install -m 644 systemd/cwsd.service %{buildroot}%{_unitdir}/cloudworkstation.service

# Install configuration files
cat > %{buildroot}%{_sysconfdir}/cloudworkstation/daemon.conf << 'EOF'
# CloudWorkstation Daemon Configuration
# This file contains default configuration for the CloudWorkstation daemon

# Daemon settings
listen_address = "127.0.0.1:8947"
log_level = "info"
log_file = "/var/log/cloudworkstation/daemon.log"

# AWS settings (override with environment variables or AWS profiles)
# aws_region = "us-west-2"
# aws_profile = "default"

# Security settings
max_concurrent_operations = 10
operation_timeout = "30m"
health_check_interval = "30s"
EOF

# Create default AWS config template
cat > %{buildroot}%{_sysconfdir}/cloudworkstation/aws/config.template << 'EOF'
# AWS Configuration Template for CloudWorkstation
# Copy this file to 'config' and customize for your environment

[default]
region = us-west-2
output = json

# Example profile for research projects
[profile research]
region = us-west-2
output = json

# Add your AWS profiles here
EOF

cat > %{buildroot}%{_sysconfdir}/cloudworkstation/aws/credentials.template << 'EOF'
# AWS Credentials Template for CloudWorkstation
# Copy this file to 'credentials' and add your AWS access keys

[default]
# aws_access_key_id = YOUR_ACCESS_KEY
# aws_secret_access_key = YOUR_SECRET_KEY

[research]
# aws_access_key_id = RESEARCH_ACCESS_KEY
# aws_secret_access_key = RESEARCH_SECRET_KEY

# Add your AWS credentials here
# Consider using IAM roles or AWS SSO for better security
EOF

# Install documentation
install -m 644 README.md %{buildroot}%{_docdir}/%{name}/README.md
install -m 644 LICENSE %{buildroot}%{_docdir}/%{name}/LICENSE
install -m 644 CHANGELOG.md %{buildroot}%{_docdir}/%{name}/CHANGELOG.md

# Create man pages
mkdir -p %{buildroot}%{_mandir}/man1
cat > %{buildroot}%{_mandir}/man1/cws.1 << 'EOF'
.TH CWS 1 "December 2024" "CloudWorkstation 0.4.2" "User Commands"
.SH NAME
cws \- CloudWorkstation command line interface
.SH SYNOPSIS
.B cws
[\fIGLOBAL_OPTIONS\fR] \fICOMMAND\fR [\fICOMMAND_OPTIONS\fR]
.SH DESCRIPTION
CloudWorkstation allows academic researchers to launch pre-configured cloud workstations in seconds.
.SH COMMANDS
.TP
.B templates
List available research templates
.TP
.B launch
Launch a new research instance
.TP
.B list
List running instances
.TP
.B connect
Connect to an instance via SSH
.TP
.B stop
Stop an instance
.TP
.B terminate
Terminate an instance
.TP
.B hibernate
Hibernate an instance to save costs
.TP
.B resume
Resume a hibernated instance
.SH EXAMPLES
.TP
List available templates:
.B cws templates
.TP
Launch a Python ML environment:
.B cws launch "Python Machine Learning" my-project
.TP
Connect to an instance:
.B cws connect my-project
.SH FILES
.TP
.I ~/.cloudworkstation/
User configuration directory
.TP
.I /etc/cloudworkstation/
System configuration directory
.SH SEE ALSO
.B cwsd(1)
.SH AUTHOR
CloudWorkstation Team
EOF

cat > %{buildroot}%{_mandir}/man1/cwsd.1 << 'EOF'
.TH CWSD 1 "December 2024" "CloudWorkstation 0.4.2" "System Commands"
.SH NAME
cwsd \- CloudWorkstation daemon
.SH SYNOPSIS
.B cwsd
[\fIOPTIONS\fR]
.SH DESCRIPTION
The CloudWorkstation daemon provides the backend API for managing research instances.
.SH OPTIONS
.TP
.B \-autonomous
Run in autonomous mode (suitable for systemd service)
.TP
.B \-config PATH
Path to configuration file (default: /etc/cloudworkstation/daemon.conf)
.TP
.B \-log-level LEVEL
Set log level (debug, info, warn, error)
.SH FILES
.TP
.I /etc/cloudworkstation/daemon.conf
Main configuration file
.TP
.I /var/lib/cloudworkstation/
State and data directory
.TP
.I /var/log/cloudworkstation/
Log files directory
.SH SEE ALSO
.B cws(1), systemctl(1)
.SH AUTHOR
CloudWorkstation Team
EOF

# Pre-install script
%pre
# Create cloudworkstation system user and group
getent group cloudworkstation >/dev/null || groupadd -r cloudworkstation
getent passwd cloudworkstation >/dev/null || \
    useradd -r -g cloudworkstation -d %{_sharedstatedir}/cloudworkstation \
            -s /sbin/nologin -c "CloudWorkstation System User" cloudworkstation

# Create necessary directories with proper ownership
mkdir -p %{_sharedstatedir}/cloudworkstation
mkdir -p %{_localstatedir}/log/cloudworkstation
mkdir -p %{_sysconfdir}/cloudworkstation/aws

# Set ownership
chown cloudworkstation:cloudworkstation %{_sharedstatedir}/cloudworkstation
chown cloudworkstation:cloudworkstation %{_localstatedir}/log/cloudworkstation
chown root:cloudworkstation %{_sysconfdir}/cloudworkstation
chown root:cloudworkstation %{_sysconfdir}/cloudworkstation/aws

# Set permissions
chmod 755 %{_sharedstatedir}/cloudworkstation
chmod 750 %{_localstatedir}/log/cloudworkstation
chmod 750 %{_sysconfdir}/cloudworkstation
chmod 750 %{_sysconfdir}/cloudworkstation/aws

exit 0

# Post-install script
%post
# Reload systemd daemon and enable service
%systemd_post cloudworkstation.service

# Create systemd drop-in directory for local overrides
mkdir -p %{_sysconfdir}/systemd/system/cloudworkstation.service.d

# Set proper ownership on created directories (in case they weren't created in %pre)
chown cloudworkstation:cloudworkstation %{_sharedstatedir}/cloudworkstation %{_localstatedir}/log/cloudworkstation 2>/dev/null || true
chmod 755 %{_sharedstatedir}/cloudworkstation 2>/dev/null || true
chmod 750 %{_localstatedir}/log/cloudworkstation 2>/dev/null || true

# Display installation success message
cat << 'EOF'

╔══════════════════════════════════════════════════════════════════════════════╗
║                   CloudWorkstation Successfully Installed                    ║
╠══════════════════════════════════════════════════════════════════════════════╣
║                                                                              ║
║  🎉 CloudWorkstation v0.4.2 has been installed successfully!                ║
║                                                                              ║
║  📋 What's installed:                                                        ║
║     ✅ Core binaries: cws, cwsd                                              ║
║     ✅ Systemd service: cloudworkstation.service                             ║
║     ✅ Configuration: /etc/cloudworkstation/                                 ║
║     ✅ User account: cloudworkstation (system user)                          ║
║                                                                              ║
║  🚀 Next steps:                                                              ║
║                                                                              ║
║     1. Configure AWS credentials:                                            ║
║        sudo cp /etc/cloudworkstation/aws/credentials.template \             ║
║               /etc/cloudworkstation/aws/credentials                          ║
║        sudo cp /etc/cloudworkstation/aws/config.template \                  ║
║               /etc/cloudworkstation/aws/config                               ║
║        sudo nano /etc/cloudworkstation/aws/credentials                      ║
║                                                                              ║
║     2. Start the service:                                                    ║
║        sudo systemctl enable --now cloudworkstation                         ║
║                                                                              ║
║     3. Test the installation:                                                ║
║        cws --version                                                         ║
║        cws templates                                                         ║
║                                                                              ║
║  📚 Documentation: https://github.com/scttfrdmn/prism            ║
║  🐛 Issues: https://github.com/scttfrdmn/prism/issues            ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝

EOF

exit 0

# Pre-uninstall script
%preun
%systemd_preun cloudworkstation.service

# Post-uninstall script
%postun
%systemd_postun_with_restart cloudworkstation.service

# Only remove user on complete package removal (not upgrade)
if [ $1 -eq 0 ]; then
    # Stop any running daemon processes
    pkill -u cloudworkstation 2>/dev/null || true
    
    # Remove user and group (only if no other packages depend on them)
    if getent passwd cloudworkstation >/dev/null; then
        userdel cloudworkstation 2>/dev/null || true
    fi
    if getent group cloudworkstation >/dev/null; then
        groupdel cloudworkstation 2>/dev/null || true
    fi
    
    # Clean up systemd drop-in directory if empty
    rmdir %{_sysconfdir}/systemd/system/cloudworkstation.service.d 2>/dev/null || true
    
    echo ""
    echo "CloudWorkstation has been completely removed from your system."
    echo "Configuration files in /etc/cloudworkstation/ have been preserved."
    echo "To completely remove all data, run:"
    echo "  sudo rm -rf /etc/cloudworkstation/ /var/lib/cloudworkstation/ /var/log/cloudworkstation/"
    echo ""
fi

exit 0

# File list
%files
# Binaries
%attr(755, root, root) %{_bindir}/cws
%attr(755, root, root) %{_bindir}/cwsd

# Systemd service
%attr(644, root, root) %{_unitdir}/cloudworkstation.service

# Configuration files
%dir %attr(750, root, cloudworkstation) %{_sysconfdir}/cloudworkstation
%config(noreplace) %attr(640, root, cloudworkstation) %{_sysconfdir}/cloudworkstation/daemon.conf
%dir %attr(750, root, cloudworkstation) %{_sysconfdir}/cloudworkstation/aws
%attr(640, root, cloudworkstation) %{_sysconfdir}/cloudworkstation/aws/config.template
%attr(640, root, cloudworkstation) %{_sysconfdir}/cloudworkstation/aws/credentials.template

# State and log directories
%dir %attr(755, cloudworkstation, cloudworkstation) %{_sharedstatedir}/cloudworkstation
%dir %attr(750, cloudworkstation, cloudworkstation) %{_localstatedir}/log/cloudworkstation

# Documentation
%doc %{_docdir}/%{name}/README.md
%doc %{_docdir}/%{name}/CHANGELOG.md
%license %{_docdir}/%{name}/LICENSE

# Man pages
%{_mandir}/man1/cws.1.gz
%{_mandir}/man1/cwsd.1.gz

# Changelog
%changelog
* Mon Dec 16 2024 CloudWorkstation Team <team@cloudworkstation.dev> - 0.4.2-1
- Phase 4 complete: Enterprise Research Platform
- Project-based organization with role-based access control
- Advanced budget management with real-time tracking
- Cost analytics with hibernation savings analysis
- Multi-user collaboration with granular permissions
- Enterprise API for project and budget management
- Budget automation with alerts and automated actions
- Template inheritance system with validation
- Comprehensive hibernation ecosystem
- Cross-platform GUI with system tray integration
- Enhanced TUI interface with professional styling
- Multi-modal access (CLI/TUI/GUI) with feature parity

* Fri Nov 15 2024 CloudWorkstation Team <team@cloudworkstation.dev> - 0.4.1-1
- Template system enhancements with inheritance support
- Improved cost optimization and hibernation features
- Enhanced security hardening and compliance
- Bug fixes and performance improvements

* Mon Oct 28 2024 CloudWorkstation Team <team@cloudworkstation.dev> - 0.4.0-1
- Major release: Multi-modal access implementation
- New GUI interface with Fyne framework
- Enhanced TUI interface with BubbleTea
- Unified daemon architecture with REST API
- Profile system integration across all interfaces
- Advanced hibernation and cost optimization
- Template application engine improvements

* Wed Sep 25 2024 CloudWorkstation Team <team@cloudworkstation.dev> - 0.3.0-1
- Advanced research features implementation
- Multi-package manager support (DNF, APT, Conda)
- Hibernation system with cost optimization
- Enhanced state management and storage
- Template system improvements
- Security hardening enhancements

* Tue Aug 20 2024 CloudWorkstation Team <team@cloudworkstation.dev> - 0.2.0-1
- Distributed architecture with daemon + CLI client
- Enhanced AWS integration and error handling
- Improved template system with validation
- Storage management with EFS and EBS support
- Security enhancements and audit logging

* Mon Jul 15 2024 CloudWorkstation Team <team@cloudworkstation.dev> - 0.1.0-1
- Initial RPM package release
- Core functionality: launch, manage, terminate instances
- Basic template system for research environments
- AWS integration with cost estimation
- CLI interface with comprehensive commands