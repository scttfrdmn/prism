# CloudWorkstation Development TODOs

## High Priority - Critical Issues

### 🚨 Template System Validation
- [ ] **CRITICAL: Test sampling of normal templates for software install issues**
  - The idle detection template deployment issues suggest broader template UserData problems
  - Test key templates: R Research Environment, Python ML, Web Development
  - Validate that packages are installing correctly
  - Check UserData vs generated script conflicts
  - Ensure template inheritance system works with software installation

### 🔄 Idle Detection System Improvements
- [ ] **Test idle tag reset behavior on instance restart**
  - Verify that stopped/hibernated instances reset `IdleStatus` to `active` on restart
  - Test hibernated instance resume cycle
  - Validate cron job execution after system restart
  - Ensure agent survives reboot and continues monitoring

- [ ] **Agent Update and Versioning System**
  - Implement versioned agent deployment with semantic versioning
  - Create agent update mechanism for existing instances
  - Add version comparison and automatic update capabilities
  - Implement agent self-update via CloudWorkstation daemon push

- [ ] **AWS CLI Maintenance Automation**
  - Add periodic AWS CLI version checking (weekly cron job)
  - Implement automatic AWS CLI v2 updates on instances
  - Push updates to running instances via daemon
  - Track CLI versions across fleet for security and compatibility

### 🎨 User Experience Improvements  
- [ ] **Template Slug Names for CLI**
  - Add slug versions of template names with dashes (e.g., `r-research-env`) as **alternatives**
  - Support both display names and slugs for launch commands (user choice)
  - Update `cws list` to show both display name and slug for reference
  - Examples (both work):
    - `cws launch "R Research Environment" my-project` (existing)
    - `cws launch r-research-env my-project` (new slug alternative)
    - `cws launch "Idle Detection Test" my-test` (existing)  
    - `cws launch idle-detection-test my-test` (new slug alternative)

- [ ] **Enhanced Template Information**
  - Implement `cws templates info <name>` command for detailed template information
  - Show packages included, services configured, ports exposed
  - Display template inheritance chain and dependencies  
  - Include cost estimates and instance type recommendations
  - Show UserData preview and configuration options
  - Example output:
    ```
    Template: R Research Environment (r-research-env)
    Version: 1.2.0
    Description: R + RStudio Server + tidyverse packages
    
    Inherits: ubuntu-22.04-base
    Package Manager: apt + conda
    
    Packages:
      System: r-base, r-base-dev, curl, wget, git
      R: tidyverse, ggplot2, dplyr, shiny
      
    Services:
      - RStudio Server (port 8787)
      - SSH (port 22)
      
    Instance Types: t3.medium (x86_64), t4g.medium (arm64)  
    Cost: $0.0464/hour (x86_64), $0.0368/hour (arm64)
    
    Idle Detection: 5min idle → 10min hibernate
    ```

## Medium Priority - Feature Enhancements

### 🧠 Intelligent Idle Detection
- [ ] **Enhanced System Metrics**
  - GPU utilization monitoring (NVIDIA-SMI integration)
  - Network I/O activity thresholds
  - Disk I/O activity detection  
  - Memory usage pattern analysis
  - Process-specific monitoring (Jupyter, RStudio, Docker containers)

- [ ] **Research Domain Intelligence**
  - ML training job detection (long-running GPU processes)
  - Data analysis pattern recognition
  - Scheduled job awareness (cron, batch systems)
  - Domain-specific idle thresholds (ML vs data analysis vs web dev)

- [ ] **Hibernation Policy Optimization**
  - Machine learning-based hibernation vs stop decisions
  - Instance type-specific policies (GPU instances hibernate faster)
  - Cost-based decision making (hibernation vs stop cost analysis)
  - User preference learning and adaptation

### 👥 Multi-User and Collaboration
- [ ] **Per-User Idle Detection**
  - Individual user activity tracking
  - Multi-user instance support
  - User-specific notification before actions
  - Collaborative instance management

- [ ] **Notification System**
  - Slack/Discord integration for idle warnings
  - Email notifications before hibernation/stop
  - Team notifications for shared instances
  - Cost savings reports and summaries

### 📊 Analytics and Monitoring
- [ ] **Cost Analytics Integration**
  - Real-time savings calculation and reporting
  - Historical cost analysis (with vs without idle detection)
  - Per-project cost optimization metrics
  - Research group cost dashboards

- [ ] **Advanced Monitoring**
  - Web UI dashboard for idle detection status
  - Fleet-wide idle detection overview
  - Alert system for agent failures
  - Performance metrics and optimization suggestions

## Low Priority - Quality of Life

### 🔧 System Administration
- [ ] **Enhanced IAM Role Management**
  - Automatic IAM role updates and versioning
  - Cross-region IAM role synchronization
  - Role permission validation and health checks
  - Least-privilege principle enforcement

- [ ] **Template System Improvements** 
  - Template validation pipeline
  - Automated template testing
  - Template performance benchmarking
  - Community template marketplace integration

### 🧪 Testing and Validation
- [ ] **Automated Testing Suite**
  - End-to-end idle detection integration tests
  - Template deployment validation tests
  - Cross-platform compatibility testing (x86_64 vs ARM64)
  - Performance regression testing

- [ ] **Documentation Improvements**
  - Interactive troubleshooting guides
  - Video tutorials for idle detection system
  - Research domain-specific guides
  - Cost optimization best practices

## Technical Debt

### 🏗️ Architecture Improvements
- [ ] **Profile System Integration**
  - Better error handling for profile mismatches
  - Automatic profile detection and configuration
  - Profile validation during daemon startup
  - Profile-specific idle detection settings

- [ ] **Error Handling and Resilience**
  - Retry logic for AWS API calls
  - Graceful degradation when AWS services unavailable
  - Agent health checking and self-recovery
  - Network connectivity validation

### 🧹 Code Quality
- [ ] **Logging Standardization**
  - Structured logging across all components
  - Log rotation and retention policies
  - Centralized log aggregation
  - Debug mode and verbose logging controls

- [ ] **Configuration Management**
  - Centralized configuration system
  - Environment-specific configurations
  - Configuration validation and schema
  - Hot configuration reloading

## Research and Development

### 🔬 Advanced Features
- [ ] **Machine Learning Integration**
  - Usage pattern prediction
  - Anomaly detection for research workloads
  - Predictive scaling and hibernation
  - Research productivity optimization

- [ ] **Integration Ecosystem**
  - Jupyter Hub integration
  - Kubernetes cluster support
  - CI/CD pipeline integration
  - External monitoring system connectors

### 🌐 Cloud Platform Expansion
- [ ] **Multi-Cloud Support Planning**
  - Azure VM idle detection
  - Google Compute Engine integration
  - Cross-cloud cost comparison
  - Unified multi-cloud management

---

## Priority Matrix

| Task | Impact | Effort | Priority |
|------|--------|--------|----------|
| Template system validation | High | Medium | **P0** |
| Idle tag reset testing | High | Low | **P0** |
| Agent versioning system | High | Medium | **P1** |
| AWS CLI maintenance | Medium | Medium | **P1** |
| Enhanced metrics | Medium | High | **P2** |
| Web UI dashboard | Low | High | **P3** |

---

*Last Updated: 2025-08-08*
*Next Review: Weekly*