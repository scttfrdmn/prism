# Commercial Software Templates Planning

## Executive Summary

This document outlines the technical approach for implementing commercial software templates in CloudWorkstation, enabling researchers to launch instances with pre-licensed commercial software like MATLAB, ArcGIS, Mathematica, and others.

## Problem Statement

Researchers frequently need access to commercial software that requires:
- Complex licensing configurations
- Specific installation procedures
- Pre-configured environments
- License compliance tracking
- Regional AMI availability

Currently, CloudWorkstation only supports open-source software templates, limiting its utility for research that depends on commercial tools.

## Technical Architecture

### 1. AMI-Based Template System

**Direct AMI Reference**:
```yaml
# templates/commercial/matlab-r2024a.yml
name: "MATLAB R2024a"
category: commercial
license_type: byol
ami_strategy: "direct_reference"

ami_mappings:
  us-east-1: "ami-0123456789abcdef0"
  us-west-2: "ami-0fedcba9876543210"
  eu-west-1: "ami-0abcdef123456789a"

# Fallback to search if direct AMI not available
ami_search:
  owner: "679593333241"  # MathWorks official account
  name_pattern: "MATLAB-R2024a-*"
  architecture: "x86_64"
  state: "available"
```

**Automatic AMI Discovery**:
```yaml
# Alternative approach for dynamic discovery
ami_strategy: "marketplace_search"
marketplace_search:
  product_code: "aw0evgkw8e5c1q413zgy5pjce"  # MATLAB Marketplace product
  version_pattern: "R2024a"
  instance_types: ["m5.xlarge", "m5.2xlarge", "c5.2xlarge"]
```

### 2. License Management Integration

**BYOL (Bring Your Own License) Workflow**:
```yaml
license_config:
  type: "network_license"
  license_server: "{{ user_input:license_server_url }}"
  license_validation: true
  usage_tracking: true

startup_script: |
  #!/bin/bash
  # Configure MATLAB license
  echo "SERVER {{license_server}} 27000" > /usr/local/MATLAB/R2024a/licenses/network.lic
  export MLM_LICENSE_FILE=/usr/local/MATLAB/R2024a/licenses/network.lic
```

**Cost Integration**:
```yaml
cost_calculation:
  base_instance_cost: true
  software_license_cost:
    type: "user_provided"
    description: "Enter hourly license cost (optional for budgeting)"
    default: 0.0
  marketplace_cost: true  # Automatically calculated for Marketplace AMIs
```

### 3. Template Schema Extensions

**Enhanced Template Structure**:
```go
// pkg/templates/types.go additions
type CommercialConfig struct {
    LicenseType     string            `yaml:"license_type" json:"license_type"`
    AMIStrategy     string            `yaml:"ami_strategy" json:"ami_strategy"`
    AMIMappings     map[string]string `yaml:"ami_mappings,omitempty" json:"ami_mappings,omitempty"`
    AMISearch       *AMISearchConfig  `yaml:"ami_search,omitempty" json:"ami_search,omitempty"`
    LicenseConfig   *LicenseConfig    `yaml:"license_config,omitempty" json:"license_config,omitempty"`
    CostCalculation *CommercialCost   `yaml:"cost_calculation,omitempty" json:"cost_calculation,omitempty"`
}

type AMISearchConfig struct {
    Owner         string   `yaml:"owner" json:"owner"`
    NamePattern   string   `yaml:"name_pattern" json:"name_pattern"`
    ProductCode   string   `yaml:"product_code,omitempty" json:"product_code,omitempty"`
    Architecture  string   `yaml:"architecture" json:"architecture"`
    InstanceTypes []string `yaml:"instance_types,omitempty" json:"instance_types,omitempty"`
}
```

### 4. AMI Resolution Strategy

**Multi-Tier Resolution**:
1. **Direct Mapping**: Check ami_mappings for current region
2. **Dynamic Search**: Use EC2 DescribeImages with search criteria
3. **Marketplace Lookup**: Query AWS Marketplace for product codes
4. **Cross-Region Search**: Search neighboring regions if not found locally
5. **Graceful Fallback**: Provide clear error with manual AMI instructions

**Implementation**:
```go
// pkg/aws/commercial_ami.go
type CommercialAMIResolver struct {
    ec2Client     EC2ClientInterface
    marketplaceClient MarketplaceClientInterface
}

func (r *CommercialAMIResolver) ResolveAMI(template *Template, region string) (*AMIInfo, error) {
    // 1. Try direct mapping first
    if directAMI := template.Commercial.AMIMappings[region]; directAMI != "" {
        return r.validateAMI(directAMI, region)
    }

    // 2. Try dynamic search
    if template.Commercial.AMISearch != nil {
        return r.searchAMIByPattern(template.Commercial.AMISearch, region)
    }

    // 3. Try marketplace lookup
    if productCode := template.Commercial.AMISearch.ProductCode; productCode != "" {
        return r.lookupMarketplaceAMI(productCode, region)
    }

    return nil, fmt.Errorf("no AMI resolution strategy available for commercial template")
}
```

### 5. User Experience Flow

**Launch Command Enhancement**:
```bash
# Standard launch with license prompts
cws launch matlab-r2024a my-research

# Launch with pre-configured license
cws launch matlab-r2024a my-research --license-server "license.university.edu:27000"

# Launch with license file
cws launch matlab-r2024a my-research --license-file ./matlab.lic

# Show license requirements before launch
cws templates info matlab-r2024a --license-info
```

**Interactive License Setup**:
```bash
🔐 Commercial Software License Configuration
Template: MATLAB R2024a
License Type: Network License

Please provide your license server information:
License Server: license.university.edu:27000
Port (default 27000): [Enter]
License File (optional):

💰 Estimated Costs:
Instance Cost: $0.45/hour (m5.xlarge)
Software License: $[user configured]/hour
Total Estimated: $0.45+/hour

Continue? [y/N]: y
```

### 6. Security Considerations

**License Protection**:
- Store license server URLs in user profiles, not templates
- Encrypt sensitive license information
- Support temporary license tokens where possible
- Audit license usage for compliance

**AMI Validation**:
- Verify AMI signatures where available
- Check AMI ownership and authenticity
- Validate AMI permissions before launch
- Log AMI resolution attempts for debugging

## Implementation Phases

### Phase 1: Core AMI Resolution (v0.5.2)
- Basic AMI mapping system
- Direct AMI reference support
- Simple BYOL license configuration
- Enhanced template schema

### Phase 2: Advanced Discovery (v0.5.3)
- Dynamic AMI search capabilities
- AWS Marketplace integration
- Cross-region AMI discovery
- License validation system

### Phase 3: Enterprise Features (v0.5.4)
- License usage tracking
- Cost integration for licensed software
- Institutional license server support
- Compliance reporting

## Initial Commercial Templates

### Priority Templates:
1. **MATLAB** (R2024a, R2023b)
   - Network license support
   - Parallel Computing Toolbox
   - Common research toolboxes

2. **ArcGIS Desktop/Pro**
   - Named user licensing
   - ArcGIS Online integration
   - Spatial analysis extensions

3. **Mathematica**
   - Site license support
   - Wolfram Cloud integration
   - Parallel processing setup

4. **Stata**
   - Network license support
   - Statistical packages
   - Data visualization tools

### Template Repository Structure:
```
templates/commercial/
├── matlab/
│   ├── matlab-r2024a.yml
│   ├── matlab-r2023b.yml
│   └── matlab-parallel-server.yml
├── esri/
│   ├── arcgis-desktop-10.9.yml
│   ├── arcgis-pro-3.2.yml
│   └── arcgis-server.yml
├── wolfram/
│   ├── mathematica-14.yml
│   └── wolfram-engine.yml
└── stata/
    ├── stata-18-mp.yml
    └── stata-18-se.yml
```

## Cost Estimation Strategy

**Multi-Component Pricing**:
- Base AWS infrastructure costs (computed instances, storage, networking)
- AWS Marketplace software costs (where applicable)
- User-provided license costs (for BYOL scenarios)
- Data transfer and storage costs

**Budget Integration**:
- Include software licensing in project budget calculations
- Separate line items for infrastructure vs. software costs
- License cost alerts and controls
- Usage-based cost optimization recommendations

## Success Metrics

**Technical Success**:
- AMI resolution success rate >95% across all supported regions
- License configuration success rate >90%
- Template launch time <2 minutes for commercial software
- Zero license compliance violations in testing

**User Adoption**:
- 50+ commercial software launches per month after 6 months
- Positive user feedback on license configuration UX
- Successful deployment at 3+ research institutions
- Cost savings documentation vs. manual commercial software deployment

## Risk Mitigation

**License Compliance Risks**:
- Clear documentation of user responsibilities
- Audit trails for license usage
- Integration with institutional license management
- Regular compliance reviews

**Technical Risks**:
- AMI availability variations across regions
- Licensing server connectivity requirements
- Software version compatibility issues
- Marketplace pricing changes

**Mitigation Strategies**:
- Comprehensive testing across regions
- Fallback procedures for AMI unavailability
- Clear error messages and troubleshooting guides
- Regular template maintenance and updates

This implementation provides a robust foundation for commercial software support while maintaining CloudWorkstation's ease-of-use principles and cost transparency.