# Phase 5.1 Universal AMI System Implementation Roadmap

**Phase**: 5.1 - Universal AMI System
**Version Target**: v0.5.2
**Timeline**: March 2026
**Status**: 🚧 **Ready to Begin Implementation**

## 🎯 Phase Objectives

Phase 5.1 transforms CloudWorkstation from script-only provisioning to intelligent hybrid deployment, delivering:

- **30-second launches** for optimized environments vs 5-8 minute script provisioning
- **Universal AMI support** for any template (not just commercial software)
- **Intelligent fallback strategies** with transparent user communication
- **Cross-region AMI intelligence** with automated discovery and copying
- **AMI creation and sharing** for community-driven optimization

## 📋 Implementation Plan

### **Week 1-2: Core Infrastructure (March 3-16, 2026)**

#### **Week 1: Foundation Architecture**

**Day 1-2: Template Schema Extensions**
- [ ] Extend `pkg/templates/types.go` with `AMIConfig` struct
- [ ] Add AMI strategy enums (`ami_preferred`, `ami_required`, `ami_fallback`)
- [ ] Implement AMI configuration validation in `pkg/templates/validator.go`
- [ ] Update template YAML parser to handle AMI configuration

**Files to Create/Modify**:
```go
pkg/templates/types.go          // Add AMIConfig struct
pkg/templates/ami_validator.go  // NEW: AMI config validation
pkg/templates/loader.go         // Update YAML parsing
templates/examples/ami-*.yml    // NEW: Example AMI templates
```

**Day 3-4: AMI Resolver Foundation**
- [ ] Create `pkg/aws/ami_resolver.go` with multi-tier resolution
- [ ] Implement `UniversalAMIResolver` struct and core methods
- [ ] Add regional fallback mapping in `pkg/aws/region_mapping.go`
- [ ] Create AMI resolution result types and error handling

**Files to Create**:
```go
pkg/aws/ami_resolver.go      // NEW: Core AMI resolution engine
pkg/aws/region_mapping.go    // NEW: Regional fallback logic
pkg/aws/ami_cache.go         // NEW: AMI metadata caching
pkg/types/ami.go             // NEW: AMI-related types
```

**Day 5: Integration with Instance Manager**
- [ ] Update `pkg/aws/manager.go` to integrate AMI resolution
- [ ] Modify instance launch logic to handle AMI vs script deployment
- [ ] Add AMI resolution result tracking in instance state
- [ ] Update cost calculation to include AMI deployment benefits

#### **Week 2: API and CLI Integration**

**Day 1-2: REST API Endpoints**
- [ ] Create `pkg/daemon/ami_handlers.go` with new endpoints
- [ ] Implement AMI resolution API (`/api/v1/ami/resolve/{template}`)
- [ ] Add AMI testing API (`/api/v1/ami/test`)
- [ ] Create cost comparison API (`/api/v1/ami/costs/{template}`)

**Files to Create**:
```go
pkg/daemon/ami_handlers.go   // NEW: AMI API endpoints
pkg/api/client/ami.go        // NEW: AMI client methods
pkg/types/ami_requests.go    // NEW: API request/response types
```

**Day 3-4: CLI Command Extensions**
- [ ] Update `internal/cli/launch.go` with AMI resolution options
- [ ] Add `--show-ami-resolution` flag for deployment preview
- [ ] Implement `--ami-strategy` override flag
- [ ] Create `cws ami test` command for template AMI testing

**Day 5: Testing and Validation**
- [ ] Create comprehensive unit tests for AMI resolver
- [ ] Add integration tests for AMI resolution workflow
- [ ] Test cross-region fallback logic
- [ ] Validate backwards compatibility with existing templates

### **Week 3-4: AMI Management System (March 17-30, 2026)**

#### **Week 3: AMI Creation and Management**

**Day 1-2: AMI Creation Engine**
- [ ] Create `pkg/ami/manager.go` with AMI lifecycle management
- [ ] Implement `pkg/ami/creator.go` for instance-to-AMI conversion
- [ ] Add multi-region AMI deployment in `pkg/ami/multi_region.go`
- [ ] Create AMI validation and testing in `pkg/ami/validator.go`

**Files to Create**:
```go
pkg/ami/manager.go           // NEW: AMI lifecycle management
pkg/ami/creator.go           // NEW: AMI creation from instances
pkg/ami/multi_region.go      // NEW: Multi-region deployment
pkg/ami/validator.go         // NEW: AMI validation and testing
pkg/ami/cost_calculator.go   // NEW: AMI cost analysis
```

**Day 3-4: CLI AMI Commands**
- [ ] Create `internal/cli/ami_commands.go` with full AMI command suite
- [ ] Implement `cws ami create` command
- [ ] Add `cws ami list` and `cws ami info` commands
- [ ] Create `cws ami create-multi` for multi-region deployment

**CLI Commands to Implement**:
```bash
cws ami create <template> <instance> [options]
cws ami list [--template name] [--region region]
cws ami info <ami-id>
cws ami test <template> [--all-regions]
cws ami create-multi <template> <instance> --regions us-east-1,us-west-2
```

**Day 5: AMI API Integration**
- [ ] Create AMI creation API endpoints in `pkg/daemon/ami_handlers.go`
- [ ] Add AMI listing and information APIs
- [ ] Implement AMI testing across regions API
- [ ] Create AMI cost analysis endpoints

#### **Week 4: Cross-Region Intelligence**

**Day 1-2: Cross-Region AMI Discovery**
- [ ] Implement cross-region AMI search in `pkg/aws/ami_resolver.go`
- [ ] Add AMI copying logic with cost calculation
- [ ] Create regional optimization algorithms
- [ ] Add user consent flow for cross-region operations

**Day 3-4: Cost Optimization Engine**
- [ ] Create `pkg/ami/optimizer.go` for deployment optimization
- [ ] Implement ARM64 vs x86_64 architecture preference
- [ ] Add instance family optimization based on AMI characteristics
- [ ] Create cost comparison algorithms (AMI vs script deployment)

**Day 5: Performance and Monitoring**
- [ ] Add AMI resolution performance metrics in `pkg/metrics/ami_metrics.go`
- [ ] Create AMI deployment success rate tracking
- [ ] Implement cost savings calculation and reporting
- [ ] Add AMI system health monitoring

### **Week 5-6: Community AMI System (April 1-14, 2026)**

#### **Week 5: Community Registry Foundation**

**Day 1-2: Community Registry Architecture**
- [ ] Create `pkg/ami/community.go` for community AMI registry
- [ ] Design community AMI metadata schema
- [ ] Implement AMI discovery and search functionality
- [ ] Create AMI rating and review system

**Files to Create**:
```go
pkg/ami/community.go         // NEW: Community registry client
pkg/ami/rating_system.go     // NEW: Rating and review system
pkg/ami/sharing.go           // NEW: AMI sharing permissions
pkg/api/community_client.go  // NEW: HTTP client for registry
```

**Day 3-4: Database Schema and Storage**
- [ ] Design AMI registry database schema
- [ ] Create migration scripts for AMI metadata tables
- [ ] Implement AMI metadata storage and retrieval
- [ ] Add AMI rating and review storage

**Database Tables**:
```sql
ami_registry          # Core AMI metadata
ami_regions          # Regional availability
ami_ratings          # Community ratings and reviews
ami_downloads        # Download tracking
ami_sharing          # Sharing permissions
```

**Day 5: Community Registry Server**
- [ ] Create `cmd/cws-registry/main.go` for registry server
- [ ] Implement community registry REST API
- [ ] Add AMI submission and approval workflow
- [ ] Create registry server Docker deployment

#### **Week 6: Community Integration**

**Day 1-2: CLI Community Commands**
- [ ] Extend `internal/cli/ami_commands.go` with community features
- [ ] Implement `cws ami browse` for community AMI discovery
- [ ] Add `cws ami rate` and `cws ami share` commands
- [ ] Create `cws ami submit` for community contributions

**Community CLI Commands**:
```bash
cws ami browse [--category ml] [--creator user]
cws ami rate <ami-id> <1-5> [--review "text"]
cws ami share <ami-id> --community cloudworkstation
cws ami submit <ami-id> --description "..." --public
```

**Day 3-4: GUI Integration**
- [ ] Update `cmd/cws-gui/` with AMI management interface
- [ ] Create AMI selection and preview components
- [ ] Add community AMI browser interface
- [ ] Implement AMI creation wizard in GUI

**Day 5: Integration Testing**
- [ ] Create end-to-end AMI workflow tests
- [ ] Test community registry integration
- [ ] Validate AMI creation and sharing workflow
- [ ] Performance test AMI resolution across regions

### **Week 7-8: Advanced Features and Polish (April 15-28, 2026)**

#### **Week 7: Security and Validation**

**Day 1-2: AMI Security Framework**
- [ ] Create `pkg/ami/security.go` for AMI security validation
- [ ] Implement AMI signature verification
- [ ] Add security scanning integration
- [ ] Create security policy enforcement

**Day 3-4: Access Control Integration**
- [ ] Implement AMI access control in `pkg/ami/access_control.go`
- [ ] Add institutional policy enforcement for AMI usage
- [ ] Create AMI sharing permission system
- [ ] Integrate with existing user management system

**Day 5: Audit and Compliance**
- [ ] Add AMI usage audit logging
- [ ] Create compliance reporting for AMI system
- [ ] Implement data retention policies for AMI metadata
- [ ] Add GDPR compliance features

#### **Week 8: Performance Optimization and Documentation**

**Day 1-2: Performance Optimization**
- [ ] Optimize AMI resolution performance with caching
- [ ] Implement concurrent AMI availability checking
- [ ] Add AMI metadata preloading for popular templates
- [ ] Optimize cross-region AMI discovery algorithms

**Day 3-4: Comprehensive Testing**
- [ ] Complete integration test suite for AMI system
- [ ] Add performance benchmarks for AMI resolution
- [ ] Test AMI system under load with multiple regions
- [ ] Validate backwards compatibility with all existing templates

**Day 5: Documentation Completion**
- [ ] Complete user documentation for AMI system
- [ ] Finish developer documentation and API references
- [ ] Create AMI system troubleshooting guide
- [ ] Update installation and deployment documentation

## 🧪 Testing Strategy

### Unit Testing (Throughout Implementation)

**Core Components Testing**:
```go
// Test Coverage Requirements (>90%)
pkg/aws/ami_resolver_test.go     // AMI resolution logic
pkg/ami/manager_test.go          // AMI lifecycle management
pkg/ami/creator_test.go          // AMI creation from instances
pkg/ami/community_test.go        // Community registry integration
pkg/templates/ami_validator_test.go  // Template AMI validation
```

**Test Scenarios**:
- AMI resolution with direct mapping
- Cross-region fallback logic
- AMI creation and multi-region deployment
- Community AMI discovery and rating
- Error handling and graceful degradation

### Integration Testing

**End-to-End Workflows**:
1. **Template to AMI Workflow**:
   - Launch instance from script template
   - Create AMI from optimized instance
   - Launch new instance using AMI
   - Verify functionality matches original

2. **Cross-Region AMI Access**:
   - Deploy AMI in us-east-1
   - Launch instance in eu-west-1
   - Verify automatic cross-region copy
   - Validate cost tracking

3. **Community AMI Workflow**:
   - Create and publish community AMI
   - Discover AMI through registry
   - Rate and review AMI
   - Verify community features

### Performance Testing

**Performance Targets**:
- AMI resolution: < 5 seconds
- Direct mapping: < 1 second
- Cross-region discovery: < 15 seconds
- AMI creation: < 10 minutes
- Community registry lookup: < 3 seconds

### Load Testing

**Concurrent Usage Testing**:
- 100 concurrent AMI resolutions
- 50 concurrent AMI creations
- Community registry under load
- Cross-region AMI copying stress test

## 📊 Success Metrics

### Performance Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| **AMI Launch Time** | < 30 seconds | Time from launch to ready |
| **Resolution Time** | < 5 seconds | AMI discovery and validation |
| **Success Rate** | > 95% | Successful AMI launches |
| **Cost Reduction** | > 4x | Setup cost savings vs scripts |

### Adoption Metrics

| Metric | Target | Timeline |
|--------|--------|----------|
| **AMI-Enabled Templates** | 80% | 6 months post-launch |
| **Community AMIs** | 100+ | 3 months post-launch |
| **User Adoption** | 70% | 6 months post-launch |
| **Launch Speed Improvement** | 10x average | Immediate |

### Quality Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Test Coverage** | > 90% | Automated testing |
| **Bug Reports** | < 5/month | Post-launch tracking |
| **User Satisfaction** | > 4.5/5 | User feedback surveys |
| **Documentation Completeness** | 100% | All features documented |

## 🚀 Risk Mitigation

### Technical Risks

**AMI Availability Risk**:
- **Risk**: AMIs unavailable in some regions
- **Mitigation**: Robust cross-region fallback with automatic copying
- **Monitoring**: Regional availability dashboard

**Performance Degradation Risk**:
- **Risk**: AMI resolution slower than script provisioning
- **Mitigation**: Aggressive caching and concurrent resolution
- **Monitoring**: Performance metrics and alerting

**Community AMI Quality Risk**:
- **Risk**: Low-quality or malicious community AMIs
- **Mitigation**: Rating system, security scanning, and verification
- **Monitoring**: Community AMI quality metrics

### Operational Risks

**Cost Overrun Risk**:
- **Risk**: Increased AWS costs from AMI storage and transfer
- **Mitigation**: Cost monitoring, automated cleanup, and user alerts
- **Monitoring**: Cost tracking dashboard

**Complexity Risk**:
- **Risk**: System too complex for users to understand
- **Mitigation**: Excellent documentation, intuitive defaults, and clear error messages
- **Monitoring**: User feedback and support ticket analysis

## 📈 Post-Launch Plan

### Phase 5.1.1: Immediate Post-Launch (May 2026)
- Monitor system performance and user adoption
- Address bugs and performance issues
- Gather user feedback and iterate
- Expand AMI coverage to popular templates

### Phase 5.1.2: Enhancement Phase (June 2026)
- Implement user feedback improvements
- Add advanced AMI optimization features
- Expand community registry capabilities
- Integrate with template marketplace (Phase 5.2)

### Phase 5.1.3: Optimization Phase (July 2026)
- Performance optimization based on usage patterns
- Advanced cost optimization algorithms
- Machine learning-driven AMI recommendations
- Preparation for Phase 5.2 features

## 🔗 Dependencies and Prerequisites

### Technical Prerequisites
- [ ] AWS SDK integration for Marketplace and advanced EC2 features
- [ ] Database system for community registry (PostgreSQL recommended)
- [ ] Caching system for AMI metadata (Redis recommended)
- [ ] Container orchestration for community registry deployment

### Team Prerequisites
- [ ] AWS expertise in EC2, AMI management, and Marketplace integration
- [ ] Database design and management skills
- [ ] Community platform development experience
- [ ] Security and access control implementation expertise

### Infrastructure Prerequisites
- [ ] Multi-region AWS deployment capability
- [ ] Database hosting and management
- [ ] CDN setup for AMI metadata distribution
- [ ] Monitoring and alerting infrastructure

## 📚 Documentation Deliverables

### User Documentation
- [ ] **AMI System User Guide**: Complete guide for researchers
- [ ] **AMI Creation Tutorial**: Step-by-step AMI creation process
- [ ] **Community AMI Guide**: Using and contributing community AMIs
- [ ] **Troubleshooting Guide**: Common issues and solutions

### Developer Documentation
- [ ] **AMI System Architecture**: Technical architecture overview
- [ ] **API Reference**: Complete API documentation
- [ ] **Development Guide**: Contributing to AMI system
- [ ] **Deployment Guide**: Production deployment instructions

### Administrative Documentation
- [ ] **AMI Registry Administration**: Managing community registry
- [ ] **Security Guide**: AMI security best practices
- [ ] **Monitoring Guide**: System monitoring and alerting
- [ ] **Cost Management Guide**: AMI cost optimization

---

**Phase 5.1 Implementation Roadmap** provides a comprehensive plan for delivering the **Universal AMI System** that will revolutionize CloudWorkstation's deployment speed and reliability. The 8-week timeline balances ambitious performance goals with thorough testing and documentation to ensure a production-ready system that maintains CloudWorkstation's core principles of simplicity and reliability.