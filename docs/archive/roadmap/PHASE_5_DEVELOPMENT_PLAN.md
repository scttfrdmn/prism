# CloudWorkstation Development Plan: AWS-Native Research Ecosystem & Extensibility

## Overview

This comprehensive development plan transforms CloudWorkstation from an EC2 launcher into a unified **Research Portal for AWS** with full extensibility capabilities, providing seamless access to the complete AWS research ecosystem while maintaining enterprise-grade governance and cost control.

## Release Structure

### **Phase 5A: Multi-User Foundation** (v0.5.0 - Q1 2025)
**Duration**: 6-8 weeks  
**Focus**: Research user identity and basic policy enforcement

### **Phase 5B: AWS Research Services Integration** (v0.5.5 - Q2 2025)  
**Duration**: 8-10 weeks  
**Focus**: SageMaker Studio and web service management

### **Phase 5C: Enterprise Research Ecosystem** (v0.6.0 - Q3 2025)
**Duration**: 10-12 weeks  
**Focus**: Template marketplace and enterprise features

### **Phase 6: Extensibility & Ecosystem** (v0.7.0 - Q4 2025)
**Duration**: 12-14 weeks  
**Focus**: Plugin architecture, auto-AMI system, GUI skinning, and web services integration

---

## Phase 5A: Multi-User Foundation (v0.5.0)

### **Epic 1: Research User Architecture Implementation**

#### **Task 1.1: Research User Data Models**
- [ ] Create `ResearchUser` struct with identity fields
- [ ] Add `GlobusIdentity` struct for optional OAuth integration
- [ ] Extend `Profile` struct with research user fields
- [ ] Update profile serialization/deserialization
- [ ] Add research user validation methods

**Files to modify**: `pkg/profile/types.go`, `pkg/profile/research_user.go` (new)

#### **Task 1.2: Research User Creation Pipeline**  
- [ ] Implement research user creation during invitation acceptance
- [ ] Add UID/GID generation with AWS account ranges (5000-5999, 6000-6999, etc.)
- [ ] Create SSH key pair generation for research users
- [ ] Integrate with existing invitation manager
- [ ] Add research user persistence to profile storage

**Files to modify**: `pkg/profile/invitation_manager.go`, `pkg/profile/research_user_manager.go` (new)

#### **Task 1.3: Globus Auth Integration (Optional)**
- [ ] Create `GlobusAuthClient` with OAuth 2.0 flow
- [ ] Implement browser-based authentication flow
- [ ] Add state parameter validation (CSRF protection)
- [ ] Create user info retrieval from Globus API
- [ ] Add CLI integration for optional Globus verification

**Files to create**: `pkg/auth/globus.go`, `internal/cli/globus.go`

### **Epic 2: Policy Framework Integration**

#### **Task 2.1: Template Launch Policy Enforcement**
- [ ] Add policy validation to template resolution process
- [ ] Integrate policy checking into launch command pipeline
- [ ] Create policy violation error types with clear messages
- [ ] Add policy enforcement to daemon launch handlers
- [ ] Update launch response to include policy validation results

**Files to modify**: `pkg/templates/resolver.go`, `internal/cli/app.go`, `pkg/daemon/instance_handlers.go`

#### **Task 2.2: Enhanced Profile Management**
- [ ] Update `cws profiles current` to display active policy restrictions
- [ ] Add `cws templates list --profile-filtered` command
- [ ] Create policy violation explanations in CLI output
- [ ] Add policy override capability for admin users
- [ ] Update profile validation to check policy consistency

**Files to modify**: `internal/cli/profiles.go`, `internal/cli/templates.go`

#### **Task 2.3: Policy Management Interface**
- [ ] Add `cws profiles policy show` command to display current restrictions
- [ ] Create `cws profiles policy test` command to validate launch parameters
- [ ] Add policy inheritance display for invitation chains
- [ ] Update TUI to show policy restrictions in profile pages
- [ ] Add GUI policy display in profile management tab

**Files to modify**: `internal/cli/profiles.go`, `internal/tui/`, `cmd/cws-gui/`

### **Epic 3: Research User Provisioning**

#### **Task 3.1: Enhanced User Data Generation**
- [ ] Extend user data script generation to create research users
- [ ] Add research user SSH key configuration
- [ ] Implement home directory creation with EFS integration
- [ ] Add research user to appropriate groups (sudo, docker, etc.)
- [ ] Create systemd services that run as research user

**Files to modify**: `pkg/templates/script_generator.go`, `pkg/aws/user_data.go` (new)

#### **Task 3.2: Cross-Template Compatibility**
- [ ] Ensure research user creation works across all templates
- [ ] Add research user support to Ubuntu, Rocky Linux, Amazon Linux templates
- [ ] Update package installation to be accessible to research users
- [ ] Test service configuration (Jupyter, RStudio) for research users
- [ ] Validate SSH access and permissions

**Files to modify**: Template YAML files, user data scripts

#### **Task 3.3: EFS Integration Enhancement**  
- [ ] Update EFS mounting to use research user identity
- [ ] Create research user home directory on EFS mount
- [ ] Add research user to `cloudworkstation-shared` group automatically
- [ ] Update permission structure for research user access
- [ ] Test cross-instance sharing with research users

**Files to modify**: `pkg/aws/manager.go` (EFS mount script)

### **Phase 5A Success Criteria**
- [ ] Research users created automatically during invitation acceptance
- [ ] Policy restrictions enforced at template launch time
- [ ] Optional Globus Auth integration working end-to-end
- [ ] Research user SSH access and home directories functional
- [ ] EFS sharing works with research user identities
- [ ] All existing functionality preserved and tested

---

## Phase 5B: AWS Research Services Integration (v0.5.5)

### **Epic 4: SageMaker Studio Integration** 

#### **Task 4.1: SageMaker Service Architecture**
- [ ] Create `ServiceType` enum with SageMaker variants
- [ ] Design `SageMakerConfig` struct for service-specific configuration
- [ ] Extend `Instance` model to represent SageMaker domains/user profiles
- [ ] Add web URL handling for direct service access
- [ ] Create cost tracking for SageMaker compute instances

**Files to create**: `pkg/services/sagemaker.go`, `pkg/types/services.go`

#### **Task 4.2: SageMaker Domain Management**
- [ ] Implement SageMaker domain creation and management
- [ ] Add VPC integration with CloudWorkstation-managed networking
- [ ] Create execution role management with appropriate permissions
- [ ] Add EFS integration for shared storage access
- [ ] Implement domain cleanup and resource management

**Files to create**: `pkg/aws/sagemaker.go`

#### **Task 4.3: SageMaker Templates**
- [ ] Create SageMaker Studio Lab template (free tier)
- [ ] Create SageMaker Studio template with instance type options
- [ ] Add SageMaker Canvas template for no-code ML
- [ ] Implement template validation for SageMaker services
- [ ] Add cost estimation for SageMaker instance types

**Files to create**: `templates/sagemaker-studio-lab.yml`, `templates/sagemaker-studio.yml`

#### **Task 4.4: CLI Integration**
- [ ] Update `cws launch` command to handle web services
- [ ] Add web URL display in `cws list` and `cws info` commands
- [ ] Create `cws connect` command with web browser launch
- [ ] Add SageMaker-specific status information
- [ ] Update cost tracking to include SageMaker charges

**Files to modify**: `internal/cli/app.go`, `internal/cli/instances.go`

### **Epic 5: Web Service Management Framework**

#### **Task 5.1: Unified Service Interface**
- [ ] Create abstract `Service` interface for all AWS services
- [ ] Implement service factory pattern for different service types
- [ ] Add service-specific configuration validation
- [ ] Create unified cost tracking across EC2 and web services
- [ ] Implement service lifecycle management (start/stop/delete)

**Files to create**: `pkg/services/interface.go`, `pkg/services/factory.go`

#### **Task 5.2: Template Enhancement for Web Services**
- [ ] Extend template schema with `connection_type: web` support
- [ ] Add `service_config` section for service-specific parameters
- [ ] Update template validation for web service requirements
- [ ] Create service-specific parameter inheritance
- [ ] Add web service template examples

**Files to modify**: `pkg/templates/types.go`, `pkg/templates/resolver.go`

#### **Task 5.3: API Enhancement**
- [ ] Extend daemon API with service management endpoints
- [ ] Add web service listing and status endpoints
- [ ] Create service-specific configuration endpoints
- [ ] Update instance handlers to support multiple service types
- [ ] Add service health checking and monitoring

**Files to modify**: `pkg/daemon/service_handlers.go` (new), `pkg/api/client/services.go` (new)

### **Epic 6: Additional AWS Services**

#### **Task 6.1: Modern Development Services**
- [ ] Create AWS CodeCatalyst integration (Cloud9 replacement)
- [ ] Add VS Code Server templates for self-hosted development
- [ ] Implement development service cost tracking
- [ ] Add development-focused template examples
- [ ] Test integration with research user identities

**Files to create**: `pkg/services/codecatalyst.go`, `templates/vscode-server.yml`

#### **Task 6.2: Analytics Services (QuickSight/Athena)**
- [ ] Create QuickSight dashboard provisioning
- [ ] Add Athena query editor integration
- [ ] Implement data source connectivity
- [ ] Add analytics-focused template examples
- [ ] Create cost tracking for analytics workloads

**Files to create**: `pkg/services/analytics.go`, `templates/quicksight-analytics.yml`

#### **Task 6.3: Data Preparation (Glue DataBrew)**
- [ ] Create Glue DataBrew project provisioning
- [ ] Add data preparation templates
- [ ] Implement job monitoring and cost tracking
- [ ] Add S3 integration for data sources
- [ ] Create data science workflow examples

**Files to create**: `pkg/services/glue.go`, `templates/databrew-prep.yml`

### **Phase 5B Success Criteria**
- [ ] SageMaker Studio Lab and Studio integration working end-to-end
- [ ] Unified interface showing both EC2 instances and web services
- [ ] Cost tracking across all service types
- [ ] Web browser launch for direct service access
- [ ] Template system supporting both EC2 and web services
- [ ] Policy framework applying to all service types

---

## Phase 5C: Enterprise Research Ecosystem (v0.6.0)

### **Epic 7: Template Marketplace**

#### **Task 7.1: Community Template System**
- [ ] Create template discovery and sharing infrastructure
- [ ] Add template ratings and reviews system
- [ ] Implement template versioning and changelog tracking
- [ ] Create template submission and approval workflow
- [ ] Add template search and categorization

#### **Task 7.2: Template Governance**
- [ ] Implement digital signature system for templates
- [ ] Add template source verification
- [ ] Create institutional template approval workflows
- [ ] Add compliance metadata to template system
- [ ] Implement template security scanning

### **Epic 8: Advanced Storage Integration**

#### **Task 8.1: OpenZFS/FSx Integration**
- [ ] Add FSx for Lustre integration for HPC workloads
- [ ] Create OpenZFS integration for research data management
- [ ] Implement high-performance storage templates
- [ ] Add storage-specific cost optimization
- [ ] Create storage benchmark and selection tools

### **Epic 9: HPC and Big Data Services**

#### **Task 9.1: AWS ParallelCluster Integration**
- [ ] Create HPC cluster provisioning templates
- [ ] Add job submission and monitoring interface
- [ ] Implement cluster autoscaling integration
- [ ] Add HPC-specific cost tracking and optimization
- [ ] Create research computation workflow examples

#### **Task 9.2: EMR Studio Integration**
- [ ] Add EMR Studio for big data analytics
- [ ] Create Spark/Hadoop workflow templates
- [ ] Implement cluster management interface
- [ ] Add big data cost optimization features
- [ ] Create data processing pipeline examples

### **Phase 5C Success Criteria**
- [ ] Complete template marketplace with community contributions
- [ ] Enterprise policy engine with digital signatures
- [ ] Advanced storage options for specialized workloads
- [ ] HPC and big data processing capabilities
- [ ] Comprehensive research workflow integration

---

## Implementation Dependencies and Timeline

### **Critical Path Analysis**
1. **Phase 5A** → **Phase 5B**: Research user identity required for SageMaker integration
2. **Phase 5B** → **Phase 5C**: Web service framework needed for advanced service integration
3. **Policy Framework** spans all phases and must be maintained consistently

### **Resource Requirements**
- **Backend Development**: Go expertise for AWS service integration
- **Frontend Development**: CLI/TUI/GUI updates across all interfaces  
- **DevOps**: AWS service provisioning and cost management integration
- **Documentation**: User guides and API documentation for new services
- **Testing**: Integration testing across multiple AWS services

### **Risk Mitigation**
- **SageMaker Integration Complexity**: Start with Studio Lab (simpler) before full Studio
- **Cross-Service Cost Tracking**: Implement unified billing integration early
- **Policy Consistency**: Ensure policy framework scales across all service types
- **User Experience**: Maintain CLI/TUI/GUI parity throughout development

---

## Phase 6: Extensibility & Ecosystem (v0.7.0)

### **Epic 10: Auto-AMI Compilation System**

#### **Task 10.1: Auto-AMI Core Engine**
- [ ] Create `AutoAMIManager` with intelligent compilation scheduling
- [ ] Implement `BaseImageMonitor` for OS security update tracking
- [ ] Add `CompilationScheduler` with cost optimization and off-peak building
- [ ] Create `TemplateUsageTracker` for popularity-driven compilation triggers
- [ ] Implement `SecurityUpdateManager` for critical vulnerability response

**Files to create**: `pkg/ami/auto_manager.go`, `pkg/ami/base_monitor.go`, `pkg/ami/scheduler.go`

#### **Task 10.2: Template Compilation Integration**
- [ ] Extend template system with `compile_to_ami` configuration
- [ ] Add AMI metadata embedding with policy information
- [ ] Implement compilation cost estimation and budget controls
- [ ] Create compilation progress tracking and notification system
- [ ] Add template validation for compilable templates

**Files to modify**: `pkg/templates/types.go`, `pkg/templates/compiler.go` (new)

#### **Task 10.3: CLI Integration for Auto-AMI**
- [ ] Add `cws templates auto-ami` command group
- [ ] Implement compilation status monitoring and control commands
- [ ] Create security update notification and management interface
- [ ] Add emergency compilation capabilities for critical updates
- [ ] Implement user preference management for auto-compilation

**Files to modify**: `internal/cli/templates.go`, `internal/cli/auto_ami.go` (new)

### **Epic 11: Unified Plugin Architecture**

#### **Task 11.1: Plugin Framework Foundation**
- [ ] Create `Plugin` interface with unified CLI/daemon capabilities
- [ ] Implement `PluginManager` with lifecycle management
- [ ] Add `PluginSandbox` with security and resource limits
- [ ] Create plugin loading system for Go plugins and executables
- [ ] Implement plugin validation and signature verification

**Files to create**: `pkg/plugin/interface.go`, `pkg/plugin/manager.go`, `pkg/plugin/sandbox.go`

#### **Task 11.2: CLI Extension System**
- [ ] Create `CLIExtensionRegistry` for plugin command registration
- [ ] Implement sandboxed plugin command execution
- [ ] Add plugin flag and argument handling
- [ ] Create plugin command help and documentation integration
- [ ] Implement plugin command error handling and recovery

**Files to create**: `pkg/plugin/cli_extensions.go`, `internal/cli/plugin_loader.go`

#### **Task 11.3: Daemon API Extension System**
- [ ] Create `APIExtensionRegistry` for plugin endpoint registration
- [ ] Implement plugin HTTP handler with authentication and rate limiting
- [ ] Add plugin event system and handler registration
- [ ] Create plugin API documentation generation
- [ ] Implement plugin health monitoring and metrics

**Files to create**: `pkg/plugin/api_extensions.go`, `pkg/daemon/plugin_handlers.go`

#### **Task 11.4: Plugin Distribution System**
- [ ] Create plugin package format (.cwsplugin files)
- [ ] Implement plugin installation and removal system
- [ ] Add plugin repository and discovery mechanism
- [ ] Create plugin security scanning and approval workflow
- [ ] Implement plugin update and version management

**Files to create**: `pkg/plugin/distribution.go`, `internal/cli/plugin_manager.go`

### **Epic 12: GUI Skinning & Theming Architecture**

#### **Task 12.1: Theme System Foundation**
- [ ] Create `ThemeManager` with theme loading and application
- [ ] Implement `Theme` struct with colors, typography, and layout
- [ ] Add institutional branding support with logo and asset management
- [ ] Create theme validation and compatibility checking
- [ ] Implement theme hot-reloading for development

**Files to create**: `pkg/gui/theme/manager.go`, `pkg/gui/theme/types.go`

#### **Task 12.2: Component Theming System**
- [ ] Create themed component factory with style application
- [ ] Implement component override system for custom implementations
- [ ] Add accessibility theme support with high contrast and large fonts
- [ ] Create research workflow-optimized layouts and arrangements
- [ ] Implement theme-aware cost and status displays

**Files to modify**: `cmd/cws-gui/components/`, `cmd/cws-gui/theme/` (new directory)

#### **Task 12.3: Theme Distribution and Management**
- [ ] Create theme package format (.cwstheme files)
- [ ] Implement theme installation and selection system
- [ ] Add automatic institutional theme detection
- [ ] Create theme customization and export capabilities
- [ ] Implement theme sharing and repository system

**Files to create**: `internal/cli/theme_manager.go`, `pkg/gui/theme/distribution.go`

### **Epic 13: Web Services Integration Framework**

#### **Task 13.1: Web Service Template Engine**
- [ ] Extend template system with `connection_type: web` support
- [ ] Create `WebServiceConfig` for service-specific parameters
- [ ] Implement web service health checking and monitoring
- [ ] Add web service URL generation and access management
- [ ] Create web service lifecycle management (start/stop/restart)

**Files to modify**: `pkg/templates/types.go`, `pkg/templates/web_services.go` (new)

#### **Task 13.2: Third-Party Integration Patterns**
- [ ] Create Docker-based service integration templates
- [ ] Implement direct web service integration patterns
- [ ] Add API-driven service integration examples
- [ ] Create OAuth/OIDC authentication integration
- [ ] Implement EFS sharing integration for web services

**Templates to create**: `templates/jupyterhub-custom.yml`, `templates/rstudio-server.yml`, `templates/mlflow-tracking.yml`

#### **Task 13.3: Web Service CLI Integration**
- [ ] Add web service launch and management commands
- [ ] Implement `cws connect` command for browser launching
- [ ] Create web service status and health monitoring
- [ ] Add web service logs and debugging capabilities
- [ ] Implement unified listing for EC2 and web services

**Files to modify**: `internal/cli/instances.go`, `internal/cli/connect.go` (new)

### **Epic 14: Integration Testing & Documentation**

#### **Task 14.1: Extensibility Integration Testing**
- [ ] Create plugin development and testing framework
- [ ] Implement theme compatibility testing across platforms
- [ ] Add auto-AMI compilation integration tests
- [ ] Create web service integration test suite
- [ ] Implement end-to-end extensibility workflow tests

**Files to create**: `test/plugin/`, `test/theme/`, `test/auto_ami/`, `test/web_services/`

#### **Task 14.2: Developer Documentation**
- [ ] Create plugin development guide with examples
- [ ] Write theme development and customization guide
- [ ] Document web service integration patterns and best practices
- [ ] Create institutional deployment guide for extensibility features
- [ ] Write troubleshooting guide for plugin and theme issues

**Documentation to create**: `docs/PLUGIN_DEVELOPMENT_GUIDE.md`, `docs/THEME_DEVELOPMENT_GUIDE.md`

### **Phase 6 Success Criteria**
- [ ] Plugin system supporting CLI commands, API endpoints, and event handlers
- [ ] Auto-AMI system providing 30-second launches with security automation
- [ ] GUI theming system with institutional branding and accessibility support
- [ ] Web service integration framework with third-party tool examples
- [ ] Complete extensibility documentation and developer guides
- [ ] Institutional deployment examples demonstrating customization capabilities

---

## Implementation Dependencies and Timeline

### **Critical Path Analysis**
1. **Phase 5A** → **Phase 5B**: Research user identity required for SageMaker integration
2. **Phase 5B** → **Phase 5C**: Web service framework needed for advanced service integration  
3. **Phase 5C** → **Phase 6**: Enterprise features provide foundation for extensibility
4. **Phase 6 Internal**: Plugin architecture must be completed before theme and auto-AMI systems

### **Phase 6 Dependencies**
- **Auto-AMI System** depends on template compilation framework (Phase 5C)
- **Plugin Architecture** depends on stable daemon API (Phase 5B)
- **GUI Theming** depends on mature GUI system (Phase 5C)
- **Web Services** depends on service management framework (Phase 5B)

### **Resource Requirements**
- **Backend Development**: Go expertise for plugin architecture and auto-AMI systems
- **Frontend Development**: GUI theming and web service integration
- **DevOps**: Auto-AMI compilation and security monitoring systems
- **Documentation**: Comprehensive developer guides and integration examples
- **Testing**: Extensibility framework testing and validation

### **Risk Mitigation**
- **Plugin Security**: Implement comprehensive sandboxing before public plugin support
- **Auto-AMI Costs**: Implement strict budget controls and monitoring from day one
- **Theme Complexity**: Start with simple institutional branding before advanced customization
- **Integration Complexity**: Focus on popular web services (Jupyter, RStudio) first

This development plan transforms CloudWorkstation from an EC2 launcher into the comprehensive "Research Portal for AWS" with full extensibility capabilities while maintaining the simplicity and enterprise governance that makes it valuable for academic institutions.