# CloudWorkstation System Implementation - v0.5.x Series

**Version**: 0.5.x Development Series
**Last Updated**: December 2025
**Target Audience**: Developers, DevOps, System Administrators

## Architecture Overview

The v0.5.x series introduces the **Universal AMI System** as a foundational component that transforms CloudWorkstation from script-only provisioning to intelligent hybrid deployment. This document outlines the complete technical implementation.

## 🏗️ Core Architecture Changes

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                   CloudWorkstation v0.5.x                  │
├─────────────────────────────────────────────────────────────┤
│  CLI Client  │  TUI Client  │  GUI Client  │  REST API     │
│  (cmd/cws)   │  (cws tui)   │  (cws-gui)   │  (external)   │
└──────┬───────┴──────┬───────┴──────┬───────┴───────┬────────┘
       │              │              │               │
       └──────────────┼──────────────┼───────────────┘
                      │              │
              ┌───────────────────────▼─────────────────────────┐
              │            Daemon Core (cwsd:8947)             │
              ├─────────────────────────────────────────────────┤
              │  Template    │  AMI         │  Instance        │
              │  Manager     │  Resolver    │  Manager         │
              └──────┬───────┴──────┬───────┴──────┬───────────┘
                     │              │              │
              ┌──────▼──────┐┌──────▼──────┐┌──────▼──────┐
              │   AWS SDK   ││   AWS SDK   ││   AWS SDK   │
              │   EC2       ││   EC2       ││   SSM       │
              │             ││   Marketplace│             │
              └─────────────┘└─────────────┘└─────────────┘
```

### New Core Components

#### 1. Universal AMI Resolver (`pkg/aws/ami_resolver.go`)

```go
type UniversalAMIResolver struct {
    ec2Client         EC2ClientInterface
    marketplaceClient MarketplaceClientInterface
    stsClient         STSClientInterface
    regionMapping     map[string][]string
    cache            *AMICache
}

type AMIResolutionResult struct {
    AMI              *AMIInfo
    ResolutionMethod AMIResolutionMethod
    FallbackChain    []string
    Warning          string
    EstimatedCost    float64
    LaunchTime       time.Duration
}

func (r *UniversalAMIResolver) ResolveAMI(template *Template, region string) (*AMIResolutionResult, error) {
    // Multi-tier resolution implementation
    // 1. Direct mapping check
    // 2. Dynamic search
    // 3. Marketplace lookup
    // 4. Cross-region search
    // 5. Fallback decision
}
```

#### 2. Enhanced Template System (`pkg/templates/types.go`)

```go
type Template struct {
    Name        string     `yaml:"name" json:"name"`
    Category    string     `yaml:"category" json:"category"`
    AMIConfig   *AMIConfig `yaml:"ami_config,omitempty" json:"ami_config,omitempty"`
    UserData    string     `yaml:"user_data" json:"user_data"`
    // Existing fields...
}

type AMIConfig struct {
    Strategy            AMIStrategy            `yaml:"strategy" json:"strategy"`
    AMIMappings         map[string]string      `yaml:"ami_mappings,omitempty" json:"ami_mappings,omitempty"`
    AMISearch           *AMISearchConfig       `yaml:"ami_search,omitempty" json:"ami_search,omitempty"`
    MarketplaceSearch   *MarketplaceConfig     `yaml:"marketplace_search,omitempty" json:"marketplace_search,omitempty"`
    FallbackStrategy    string                 `yaml:"fallback_strategy" json:"fallback_strategy"`
    FallbackTimeout     string                 `yaml:"fallback_timeout" json:"fallback_timeout"`
    PreferredArch       string                 `yaml:"preferred_architecture" json:"preferred_architecture"`
    InstanceFamilyPref  []string              `yaml:"instance_family_preference" json:"instance_family_preference"`
}
```

#### 3. AMI Management System (`pkg/ami/`)

```go
// pkg/ami/manager.go
type AMIManager struct {
    ec2Client      EC2ClientInterface
    resolver       *UniversalAMIResolver
    registry       *CommunityAMIRegistry
    costCalculator *AMICostCalculator
}

type AMICreationRequest struct {
    InstanceID    string            `json:"instance_id"`
    Name          string            `json:"name"`
    Description   string            `json:"description"`
    Public        bool              `json:"public"`
    Tags          map[string]string `json:"tags"`
    MultiRegion   []string          `json:"multi_region,omitempty"`
}

func (m *AMIManager) CreateAMI(req *AMICreationRequest) (*AMICreationResult, error)
func (m *AMIManager) ShareAMI(amiID string, targets []string) error
func (m *AMIManager) CopyAMIToRegions(amiID string, regions []string) (*MultiRegionResult, error)
```

#### 4. Community AMI Registry (`pkg/ami/community.go`)

```go
type CommunityAMIRegistry struct {
    registry    map[string]map[string]*CommunityAMI
    httpClient  HTTPClient
    cacheTTL    time.Duration
    localCache  sync.Map
}

type CommunityAMI struct {
    Version       string            `yaml:"version" json:"version"`
    Creator       string            `yaml:"creator" json:"creator"`
    Description   string            `yaml:"description" json:"description"`
    Regions       map[string]string `yaml:"regions" json:"regions"`
    Verification  *AMIVerification  `yaml:"verification" json:"verification"`
    Ratings       *AMIRatings       `yaml:"ratings" json:"ratings"`
    DownloadCount int              `yaml:"download_count" json:"download_count"`
    LastUpdated   time.Time        `yaml:"last_updated" json:"last_updated"`
}

func (r *CommunityAMIRegistry) FindBestAMI(templateName, region string) (*CommunityAMI, error)
func (r *CommunityAMIRegistry) SubmitAMI(ami *CommunityAMI) error
func (r *CommunityAMIRegistry) RateAMI(amiID string, rating int, review string) error
```

## 🚀 Implementation Phases

### Phase 5.1.1: Core AMI Resolution (March 2026)

**New Files**:
```
pkg/aws/
├── ami_resolver.go          # Multi-tier AMI resolution engine
├── ami_cache.go             # AMI metadata caching system
├── region_mapping.go        # Cross-region fallback logic
└── marketplace_client.go    # AWS Marketplace integration

pkg/ami/
├── manager.go              # AMI lifecycle management
├── creator.go              # AMI creation from instances
├── validator.go            # AMI validation and testing
└── cost_calculator.go      # AMI cost analysis

pkg/templates/
├── ami_validator.go        # Template AMI config validation
└── ami_merger.go          # AMI + script hybrid handling
```

**Modified Files**:
```
pkg/templates/types.go      # Enhanced with AMIConfig
pkg/aws/manager.go          # Integrated AMI resolution
pkg/daemon/instance_handlers.go  # AMI resolution API endpoints
internal/cli/launch.go      # AMI-aware launch commands
```

**API Endpoints Added**:
```
GET  /api/v1/ami/resolve/{template}     # Resolve AMI for template
POST /api/v1/ami/test                   # Test AMI availability
GET  /api/v1/ami/costs/{template}       # Get cost comparison
POST /api/v1/ami/create                 # Create AMI from instance
GET  /api/v1/regions/fallbacks          # Regional fallback mapping
```

### Phase 5.1.2: Community AMI System (April 2026)

**New Files**:
```
pkg/ami/
├── community.go            # Community registry client
├── registry_server.go      # Community registry server
├── rating_system.go        # AMI rating and review system
└── sharing.go             # AMI sharing and permissions

pkg/api/
└── community_client.go     # HTTP client for registry API

cmd/
└── cws-registry/          # Community registry server binary
    └── main.go
```

**CLI Commands Added**:
```
cws ami create <template> <instance>     # Create AMI from instance
cws ami list [--template name]           # List available AMIs
cws ami share <ami-id> <targets>         # Share AMI
cws ami rate <ami-id> <rating>           # Rate community AMI
cws ami test <template> [--all-regions]  # Test AMI availability
cws ami browse [--category cat]          # Browse community AMIs
cws ami info <ami-id>                    # Detailed AMI information
```

### Phase 5.1.3: Advanced Intelligence (May 2026)

**New Files**:
```
pkg/ami/
├── optimizer.go            # Cost and performance optimization
├── updater.go             # Automated AMI updates
├── security_scanner.go     # AMI security validation
└── analytics.go           # Usage analytics and recommendations

pkg/intelligence/
├── cost_analyzer.go        # Advanced cost analysis
├── performance_tracker.go  # Performance benchmarking
└── recommendation_engine.go # AMI recommendations
```

**Enhanced Features**:
- Automated AMI creation for popular templates
- Security scanning and vulnerability updates
- Performance benchmarking and optimization
- Advanced cost analysis with usage patterns
- Machine learning-driven AMI recommendations

## 🔧 Database Schema Changes

### AMI Metadata Storage

```sql
-- New tables for AMI management
CREATE TABLE ami_registry (
    ami_id VARCHAR(21) PRIMARY KEY,
    template_name VARCHAR(100) NOT NULL,
    creator VARCHAR(100) NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    version VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    public BOOLEAN DEFAULT FALSE,
    verified BOOLEAN DEFAULT FALSE,
    download_count INTEGER DEFAULT 0,
    INDEX idx_template (template_name),
    INDEX idx_creator (creator),
    INDEX idx_public (public)
);

CREATE TABLE ami_regions (
    ami_id VARCHAR(21),
    region VARCHAR(20),
    ami_image_id VARCHAR(21) NOT NULL,
    available BOOLEAN DEFAULT TRUE,
    last_tested TIMESTAMP,
    PRIMARY KEY (ami_id, region),
    FOREIGN KEY (ami_id) REFERENCES ami_registry(ami_id)
);

CREATE TABLE ami_ratings (
    ami_id VARCHAR(21),
    user_id VARCHAR(100),
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (ami_id, user_id),
    FOREIGN KEY (ami_id) REFERENCES ami_registry(ami_id)
);
```

### State Management Updates

```json
{
  "instances": {
    "my-instance": {
      "id": "i-1234567890abcdef0",
      "name": "my-instance",
      "template": "python-ml",
      "ami_used": "ami-0123456789abcdef0",
      "resolution_method": "direct_mapping",
      "launch_time_seconds": 30,
      "cost_savings": 0.045,
      // existing fields...
    }
  },
  "ami_cache": {
    "python-ml": {
      "us-east-1": {
        "ami_id": "ami-0123456789abcdef0",
        "last_verified": "2026-03-15T10:30:00Z",
        "performance_score": 4.8,
        "cached_at": "2026-03-15T10:35:00Z"
      }
    }
  }
}
```

## 🌐 REST API Specification

### AMI Resolution Endpoints

```yaml
# AMI Resolution API
/api/v1/ami/resolve/{template}:
  get:
    parameters:
      - name: template
        in: path
        required: true
        schema:
          type: string
      - name: region
        in: query
        schema:
          type: string
      - name: strategy
        in: query
        schema:
          type: string
          enum: [ami_preferred, ami_required, ami_fallback]
    responses:
      200:
        content:
          application/json:
            schema:
              type: object
              properties:
                ami_id:
                  type: string
                resolution_method:
                  type: string
                estimated_launch_time:
                  type: integer
                cost_comparison:
                  type: object
                warning:
                  type: string

/api/v1/ami/create:
  post:
    requestBody:
      content:
        application/json:
          schema:
            type: object
            properties:
              instance_id:
                type: string
              name:
                type: string
              description:
                type: string
              public:
                type: boolean
              multi_region:
                type: array
                items:
                  type: string
    responses:
      202:
        content:
          application/json:
            schema:
              type: object
              properties:
                ami_id:
                  type: string
                creation_status:
                  type: string
                estimated_completion:
                  type: string
```

### Community Registry API

```yaml
/api/v1/community/amis:
  get:
    parameters:
      - name: template
        in: query
        schema:
          type: string
      - name: category
        in: query
        schema:
          type: string
      - name: region
        in: query
        schema:
          type: string
    responses:
      200:
        content:
          application/json:
            schema:
              type: array
              items:
                type: object
                properties:
                  ami_id:
                    type: string
                  name:
                    type: string
                  creator:
                    type: string
                  rating:
                    type: number
                  download_count:
                    type: integer
                  regions:
                    type: array
                    items:
                      type: string

/api/v1/community/amis/{ami_id}/rate:
  post:
    parameters:
      - name: ami_id
        in: path
        required: true
        schema:
          type: string
    requestBody:
      content:
        application/json:
          schema:
            type: object
            properties:
              rating:
                type: integer
                minimum: 1
                maximum: 5
              review:
                type: string
```

## 🔒 Security Implementation

### AMI Security Validation

```go
// pkg/ami/security.go
type SecurityValidator struct {
    scannerClient SecurityScannerInterface
    signatureValidator SignatureValidator
}

func (v *SecurityValidator) ValidateAMI(amiID string) (*SecurityReport, error) {
    // 1. Verify AMI signature and ownership
    // 2. Scan for known vulnerabilities
    // 3. Check for suspicious modifications
    // 4. Validate against security policies
    // 5. Generate security report
}

type SecurityReport struct {
    AMI_ID           string                 `json:"ami_id"`
    OverallScore     float64               `json:"overall_score"`
    Vulnerabilities  []VulnerabilityReport `json:"vulnerabilities"`
    SignatureValid   bool                  `json:"signature_valid"`
    TrustedSource    bool                  `json:"trusted_source"`
    Recommendations  []string              `json:"recommendations"`
}
```

### Access Control Integration

```go
// pkg/ami/access_control.go
type AMIAccessController struct {
    iamClient     IAMClientInterface
    policyEngine  PolicyEngineInterface
}

func (ac *AMIAccessController) CheckAMIAccess(userID, amiID string) (*AccessResult, error) {
    // 1. Verify user permissions
    // 2. Check AMI sharing permissions
    // 3. Validate institutional policies
    // 4. Return access decision
}
```

## 📊 Monitoring and Analytics

### Performance Metrics

```go
// pkg/metrics/ami_metrics.go
type AMIMetrics struct {
    ResolutionTimes    map[string]time.Duration  // Resolution method -> avg time
    LaunchSuccessRate  map[string]float64        // Template -> success rate
    CostSavings        map[string]float64        // Template -> avg cost savings
    RegionalAvailability map[string]map[string]bool  // Region -> Template -> available
}

func (m *AMIMetrics) RecordResolution(template, method string, duration time.Duration)
func (m *AMIMetrics) RecordLaunch(template, result string, cost float64)
func (m *AMIMetrics) GenerateReport() (*PerformanceReport, error)
```

### Cost Analytics

```go
// pkg/ami/cost_analytics.go
type CostAnalyzer struct {
    pricingClient AWSPricingInterface
    usageTracker  UsageTracker
}

func (ca *CostAnalyzer) AnalyzeCosts(template string, region string) (*CostAnalysis, error) {
    // 1. Calculate AMI storage costs
    // 2. Compare launch time cost savings
    // 3. Factor in cross-region transfer costs
    // 4. Provide cost optimization recommendations
}
```

## 🧪 Testing Strategy

### Unit Test Coverage

```go
// pkg/aws/ami_resolver_test.go
func TestAMIResolver_DirectMapping(t *testing.T)
func TestAMIResolver_DynamicSearch(t *testing.T)
func TestAMIResolver_CrossRegionFallback(t *testing.T)
func TestAMIResolver_GracefulFallback(t *testing.T)

// pkg/ami/manager_test.go
func TestAMIManager_CreateAMI(t *testing.T)
func TestAMIManager_ShareAMI(t *testing.T)
func TestAMIManager_MultiRegionDeployment(t *testing.T)

// pkg/ami/community_test.go
func TestCommunityRegistry_FindBestAMI(t *testing.T)
func TestCommunityRegistry_RatingSystem(t *testing.T)
```

### Integration Test Suite

```bash
# tests/integration/ami_system_test.go
TestAMISystemEndToEnd()
- Create instance from script template
- Create AMI from instance
- Launch new instance using AMI
- Verify functionality matches original
- Clean up resources

TestCrossRegionAMIAccess()
- Deploy AMI in us-east-1
- Launch instance in eu-west-1 (triggers cross-region copy)
- Verify launch success and cost tracking
- Clean up AMIs in both regions

TestCommunityAMIWorkflow()
- Create and publish community AMI
- Discover AMI through registry
- Launch instance using community AMI
- Rate and review AMI
- Verify rating system
```

### Performance Benchmarking

```go
// tests/performance/ami_benchmark_test.go
func BenchmarkAMIResolution(b *testing.B)
func BenchmarkDirectMapping(b *testing.B)
func BenchmarkCrossRegionSearch(b *testing.B)
func BenchmarkCommunityRegistryLookup(b *testing.B)

// Performance targets
// - AMI resolution < 5 seconds
// - Direct mapping < 1 second
// - Community registry lookup < 3 seconds
// - Cross-region search < 15 seconds
```

## 🚧 Deployment Architecture

### Infrastructure Requirements

```yaml
# docker-compose.yml for development
services:
  cloudworkstation:
    build: .
    ports:
      - "8947:8947"
    environment:
      - AWS_REGION=${AWS_REGION}
      - AMI_CACHE_TTL=3600
    volumes:
      - ~/.aws:/root/.aws:ro

  ami-registry:
    build: ./cmd/cws-registry
    ports:
      - "8948:8948"
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_URL=${REDIS_URL}
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: ami_registry
      POSTGRES_USER: cwsuser
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7
    volumes:
      - redis_data:/data
```

### Production Deployment

```bash
# Production deployment checklist
1. Database Migration
   - Run AMI registry schema migration
   - Set up read replicas for performance
   - Configure backup and recovery

2. AMI Registry Deployment
   - Deploy community registry service
   - Configure CDN for AMI metadata
   - Set up monitoring and alerting

3. Cache Configuration
   - Redis cluster for AMI metadata caching
   - CloudFront for static AMI information
   - Regional cache invalidation strategy

4. Monitoring Setup
   - AMI resolution performance metrics
   - Cost tracking and reporting
   - Security scanning automation
   - Community registry health monitoring
```

## 📈 Performance Optimization

### Caching Strategy

```go
// pkg/ami/cache.go
type AMICache struct {
    localCache   sync.Map              // In-memory cache
    redisClient  RedisClientInterface  // Distributed cache
    cacheConfig  CacheConfiguration
}

type CacheConfiguration struct {
    LocalTTL     time.Duration  // Local cache TTL
    RedisTTL     time.Duration  // Redis cache TTL
    MaxLocalSize int           // Max local cache entries
}

func (c *AMICache) GetAMI(key string) (*AMIInfo, bool) {
    // 1. Check local cache first (fastest)
    // 2. Check Redis cache (fast)
    // 3. Query AWS API (slowest)
    // 4. Cache results at all levels
}
```

### Regional Optimization

```go
// pkg/aws/region_optimizer.go
type RegionOptimizer struct {
    costCalculator CostCalculatorInterface
    latencyTracker LatencyTracker
}

func (ro *RegionOptimizer) OptimalRegionForLaunch(userRegion string, amiAvailability map[string]bool) (string, error) {
    // 1. Prefer user's region (lowest latency, no transfer costs)
    // 2. Consider neighboring regions (acceptable latency)
    // 3. Factor in cross-region transfer costs
    // 4. Account for AMI availability
    // 5. Return optimal region with cost estimate
}
```

## 🔄 Migration and Compatibility

### Backwards Compatibility

```go
// pkg/templates/compatibility.go
type CompatibilityManager struct {
    templateValidator TemplateValidator
    amiResolver      AMIResolver
}

func (cm *CompatibilityManager) ProcessTemplate(template *Template) (*ProcessedTemplate, error) {
    // 1. Check if template has AMI config
    // 2. If not, use script-based provisioning (existing behavior)
    // 3. If yes, attempt AMI resolution with script fallback
    // 4. Return processed template with deployment strategy
}
```

### Migration Tools

```bash
# CLI tools for migrating existing deployments
cws migrate analyze                    # Analyze current templates for AMI opportunities
cws migrate template python-research  # Convert template to use AMI optimization
cws migrate instance my-instance      # Create AMI from existing instance
cws migrate test-all                  # Test all templates with AMI resolution
```

## 📚 Documentation Integration

### Developer Documentation

```markdown
# New documentation files required:

docs/development/
├── AMI_SYSTEM_ARCHITECTURE.md    # Technical architecture overview
├── AMI_RESOLVER_DEVELOPMENT.md   # AMI resolution engine development
├── COMMUNITY_REGISTRY_API.md     # Community registry API specification
└── AMI_TESTING_GUIDE.md          # Testing AMI system components

docs/deployment/
├── AMI_REGISTRY_DEPLOYMENT.md    # Community registry deployment
├── PRODUCTION_AMI_SETUP.md       # Production AMI system setup
└── MONITORING_AMI_SYSTEM.md      # AMI system monitoring and alerting
```

### API Documentation Updates

All existing API documentation requires updates to include:
- AMI resolution endpoints
- Community registry integration
- Cost analysis API changes
- Template schema updates with AMI configuration

---

**CloudWorkstation v0.5.x System Implementation** provides the technical foundation for **revolutionary improvements in research environment deployment speed and reliability** while maintaining the platform's core principles of simplicity, reliability, and cost-effectiveness. The Universal AMI System represents a **major architectural advancement** that positions CloudWorkstation as the leading platform for research cloud computing.