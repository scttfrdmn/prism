# Prism Security Hardening Guide

## Overview

This guide provides comprehensive security hardening recommendations for Prism production deployments. Following these guidelines will ensure enterprise-grade security for your research computing infrastructure.

## 🎯 Security Objectives

- **Confidentiality**: Protect research data and access credentials
- **Integrity**: Ensure system and data integrity through tamper detection
- **Availability**: Maintain service availability while enforcing security controls
- **Auditability**: Complete audit trail for compliance and incident response
- **Defense in Depth**: Multiple security layers for comprehensive protection

## 📊 Security Maturity Levels

### Basic Security (Score: 0-49)
- Minimal security features enabled
- Suitable for development/testing only
- **Not recommended for production**

### Standard Security (Score: 50-74)
- Core security features enabled
- Basic audit logging and monitoring
- Suitable for internal/pilot deployments

### Hardened Security (Score: 75-89)
- Comprehensive security features
- Advanced monitoring and correlation
- Recommended for production deployments

### Enterprise Security (Score: 90-100)
- Full security feature set enabled
- Native keychain integration
- Recommended for enterprise/compliance environments

## 🔧 Core Security Components

### 1. Audit Logging

**Configuration:**
```yaml
audit_log_enabled: true
log_retention_days: 90  # Minimum 30 days, recommend 90+ for compliance
```

**Hardening Steps:**
- Enable comprehensive audit logging for all security events
- Set appropriate log retention period (90+ days for compliance)
- Secure audit log directory with 0700 permissions
- Implement log rotation and archival processes
- Monitor audit log integrity with checksums

**Production Requirements:**
```bash
# Verify audit directory permissions
chmod 700 ~/.prism/security/audit
chown $(whoami):$(whoami) ~/.prism/security/audit

# Enable audit log monitoring
prism security config  # Verify audit_log_enabled: true
```

### 2. Security Monitoring

**Configuration:**
```yaml
monitoring_enabled: true
monitor_interval: 30s
alert_threshold: MEDIUM
```

**Hardening Steps:**
- Enable real-time security monitoring
- Configure appropriate alert thresholds (MEDIUM for production)
- Set monitoring interval to 30 seconds or less
- Implement automated alerting to security team
- Regular review of security dashboard and metrics

**Production Commands:**
```bash
# Check security monitoring status
prism security status
prism security dashboard

# Monitor security events
prism security correlations
```

### 3. Event Correlation Analysis

**Configuration:**
```yaml
correlation_enabled: true
analysis_interval: 5m
```

**Hardening Steps:**
- Enable advanced security event correlation
- Configure analysis interval (5 minutes recommended)
- Review correlation rules and attack patterns
- Implement custom correlation rules for environment
- Regular analysis of security correlations

### 4. Registry Security

**Configuration:**
```yaml
registry_security_enabled: true
registry_url: "https://registry.prism.io"
```

**Hardening Steps:**
- Enable HMAC-SHA256 request signing
- Configure certificate pinning for registry communication
- Use secure registry endpoints (HTTPS only)
- Regularly update pinned certificate fingerprints
- Monitor registry communication for anomalies

### 5. Keychain Security

**Hardening Steps:**
- Use native keychain providers when available:
  - **macOS**: Security framework with Keychain Services
  - **Windows**: Credential Manager with DPAPI
  - **Linux**: Secret Service with desktop keyring
- Verify keychain provider security level
- Regular keychain validation and diagnostics
- Secure fallback configuration for unsupported platforms

**Validation Commands:**
```bash
# Check keychain status
prism security keychain

# Validate keychain provider
prism security health
```

## 🛡️ System Hardening

### File System Security

```bash
# Secure Prism directories
chmod 700 ~/.prism
chmod 700 ~/.prism/security
chmod 600 ~/.prism/security/*.json

# Verify file permissions
find ~/.prism -type f -exec chmod 600 {} \;
find ~/.prism -type d -exec chmod 700 {} \;
```

### Network Security

```bash
# Restrict daemon access (example with iptables)
iptables -A INPUT -p tcp --dport 8947 -s 127.0.0.1 -j ACCEPT
iptables -A INPUT -p tcp --dport 8947 -j DROP

# Use TLS for all communications
export PRISM_TLS_ENABLED=true
export PRISM_TLS_CERT_PATH=/path/to/cert.pem
export PRISM_TLS_KEY_PATH=/path/to/key.pem
```

### Process Security

```bash
# Run daemon with limited privileges
systemctl edit prism --force
```

Create systemd service configuration:
```ini
[Unit]
Description=Prism Daemon
After=network.target

[Service]
Type=simple
User=prism
Group=prism
ExecStart=/usr/local/bin/cwsd -port 8947
Restart=always
RestartSec=5
PrivateTmp=true
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/prism

[Install]
WantedBy=multi-user.target
```

## 🔐 Production Deployment Checklist

### Pre-Deployment Security Validation

```bash
# 1. Validate security configuration
prism security config
prism security health

# 2. Run comprehensive health check
prism security health

# 3. Verify all security components
prism security status

# 4. Check keychain provider
prism security keychain

# 5. Test security monitoring
prism security dashboard
```

### Required Security Features for Production

- [ ] **Audit Logging**: Enabled with 90+ day retention
- [ ] **Security Monitoring**: Enabled with MEDIUM+ alert threshold
- [ ] **Health Checks**: Enabled with 15-minute intervals
- [ ] **Registry Security**: Enabled with certificate pinning
- [ ] **Native Keychain**: Configured and validated
- [ ] **File Permissions**: Properly secured (700/600)
- [ ] **Network Security**: Restricted access configured
- [ ] **Process Security**: Non-root execution with limited privileges

### Security Score Requirements

- **Development**: Score ≥ 50 (Standard Security)
- **Staging**: Score ≥ 75 (Hardened Security)
- **Production**: Score ≥ 90 (Enterprise Security)

## 📋 Compliance Considerations

### NIST 800-171 (Protecting Controlled Unclassified Information)
**Critical for research institutions handling federal contracts or CUI data**

#### Access Control (AC)
- ✅ **AC.1.001**: Limit system access to authorized users
  - Native keychain integration with device binding
  - Multi-factor authentication support via keychain providers
  - Session-based access controls with tamper detection

- ✅ **AC.1.002**: Limit system access to authorized transactions  
  - RESTful API with role-based access controls
  - Comprehensive audit logging of all system transactions
  - Request signing and authentication for all operations

- ✅ **AC.2.005**: Provide privacy and security notices
  - Security event notifications and alerting
  - Transparent security status reporting
  - Clear documentation of data handling practices

#### Audit and Accountability (AU)  
- ✅ **AU.2.041**: Create audit records with required content
  - Comprehensive security audit logging system
  - Structured audit records with timestamps, user identification, event types
  - Configurable retention periods (90+ days for compliance)

- ✅ **AU.2.042**: Provide audit record generation capability
  - Real-time audit record generation for all security events
  - Automated correlation and analysis of audit events
  - Integration with external SIEM systems

#### Configuration Management (CM)
- ✅ **CM.2.061**: Establish configuration baselines
  - Template-based system configuration with inheritance
  - Version-controlled security configurations
  - Automated validation of security configuration compliance

- ✅ **CM.2.062**: Employ configuration change control
  - Template application with rollback capabilities
  - Change tracking and approval workflows
  - Security impact assessment for configuration changes

#### Identification and Authentication (IA)
- ✅ **IA.2.076**: Identify users uniquely
  - Device fingerprinting and binding for unique identification  
  - Integration with institutional identity providers
  - Session management with user tracking

- ✅ **IA.2.078**: Use multifactor authentication
  - Native keychain integration supports MFA flows
  - Hardware-backed authentication on supported platforms
  - Fallback authentication methods with appropriate controls

#### System and Communications Protection (SC)
- ✅ **SC.2.179**: Use encryption to protect CUI
  - AES-256-GCM encryption for all sensitive data at rest
  - TLS 1.3 for all network communications
  - End-to-end encryption for invitation system

- ✅ **SC.2.181**: Use session authentication  
  - Cryptographic session tokens with expiration
  - Session binding to device fingerprints
  - Automatic session invalidation on security events

#### System and Information Integrity (SI)
- ✅ **SI.2.214**: Monitor security events
  - Real-time security event monitoring and correlation
  - Behavioral analysis and anomaly detection
  - Automated threat response and alerting

- ✅ **SI.2.216**: Monitor communications for attacks
  - Network communication monitoring and analysis
  - Certificate pinning for secure communications
  - Intrusion detection and prevention capabilities

**NIST 800-171 Configuration Example:**
```yaml
# NIST 800-171 Compliant Configuration
audit_log_enabled: true
log_retention_days: 2555  # 7 years for federal compliance
monitoring_enabled: true
correlation_enabled: true
registry_security_enabled: true
health_check_enabled: true
monitor_interval: 30s
analysis_interval: 5m
alert_threshold: MEDIUM
```

**NIST 800-171 Production Checklist:**
- [ ] **System Security Plan (SSP)**: Document Prism security architecture
- [ ] **Plan of Action & Milestones (POA&M)**: Address any identified gaps
- [ ] **Security Assessment**: Independent validation of security controls
- [ ] **Continuous Monitoring**: Ongoing security posture assessment
- [ ] **Incident Response Plan**: Documented procedures for security events
- [ ] **Supply Chain Risk Management**: Vendor security assessment

### SOC 2 Type II
- Enable comprehensive audit logging (90+ days retention)
- Implement continuous security monitoring
- Regular security health checks and validation
- Documented incident response procedures

### HIPAA
- Enable all security features with maximum settings
- Implement additional access controls and encryption
- Extended audit log retention (7+ years)
- Regular security assessments and penetration testing

### GDPR
- Implement data protection by design and default
- Enable audit logging for data access and processing
- Implement privacy-preserving security monitoring
- Data breach detection and notification procedures

## 🚨 Incident Response

### Security Event Response

1. **Detection**: Automated alerting through security monitoring
2. **Assessment**: Review security dashboard and correlations
3. **Containment**: Isolate affected systems and instances
4. **Investigation**: Analyze audit logs and security events
5. **Recovery**: Restore systems and implement improvements
6. **Lessons Learned**: Update security configuration and procedures

### Common Security Events

| Event Type | Severity | Response |
|------------|----------|----------|
| Tamper Detection | CRITICAL | Immediate isolation and investigation |
| Multiple Failed Logins | HIGH | Review access logs, potential lockout |
| Unusual Activity Hours | MEDIUM | Verify legitimacy, investigate if needed |
| Keychain Failures | HIGH | Check system integrity, restore if needed |
| Registry Communication Errors | MEDIUM | Verify network and certificate status |

## 🔍 Monitoring and Alerting

### Key Security Metrics

- **Security Score**: Target ≥ 90 for production
- **Failed Authentication Rate**: Target < 5%
- **Tamper Detection Events**: Target = 0
- **System Health Status**: Target = 100% OK
- **Audit Log Availability**: Target = 100%

### Alerting Integration

```bash
# Example: Integrate with external monitoring
export PRISM_WEBHOOK_URL="https://monitoring.example.com/webhook"
export PRISM_SLACK_WEBHOOK="https://hooks.slack.com/services/..."
export PRISM_EMAIL_ALERTS="security@example.com"
```

## 📚 Additional Resources

- [Prism Security Architecture](./SECURITY_ARCHITECTURE.md)
- [API Security Reference](./API_SECURITY.md)
- [Compliance Checklist](./COMPLIANCE_CHECKLIST.md)
- [Incident Response Playbook](./INCIDENT_RESPONSE.md)

## 🆘 Support and Troubleshooting

### Security Support Channels

- **Security Issues**: security@prism.io
- **Documentation**: https://docs.prism.io/security
- **Community**: https://github.com/prism/community

### Common Troubleshooting

```bash
# Debug security configuration
prism security status --verbose
prism security health --debug

# Check audit logs
tail -f ~/.prism/security/audit/*.log

# Validate keychain
prism security keychain --validate

# Test security monitoring
prism security dashboard --refresh
```

---

## 📝 Security Hardening Summary

Following this guide ensures:
- ✅ **Enterprise-grade security** with comprehensive protection
- ✅ **Compliance readiness** for SOC 2, HIPAA, and GDPR requirements  
- ✅ **Operational security** with monitoring, alerting, and incident response
- ✅ **Defense in depth** with multiple security layers and controls
- ✅ **Audit compliance** with complete security event logging and retention

**Security Contact**: For security questions or to report vulnerabilities, contact security@prism.io

**Last Updated**: 2025-08-05
**Version**: 1.0 (Phase 4: Final Integration & Deployment)