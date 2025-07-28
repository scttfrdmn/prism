# AWS Pricing Discounts Configuration

## Problem Statement

AWS costs are complex and customer-specific. While CloudWorkstation can (and should) look up current pricing using the AWS Pricing API, many customers receive significant discounts that make the public pricing inaccurate for cost estimation and budgeting:

- **Enterprise Discount Program (EDP)**: Large organizations often receive 10-30% discounts
- **Reserved Instance Discounts**: Up to 75% savings for committed usage
- **Spot Instance Pricing**: 70-90% discounts for interruptible workloads  
- **Savings Plans**: 72% savings for committed compute usage
- **Volume Discounts**: Graduated discounts based on monthly spend
- **Academic/Non-Profit Pricing**: Special institutional rates
- **AWS Credits**: Promotional credits that effectively reduce costs to zero

## Current State Analysis

### **Existing Pricing Integration**
CloudWorkstation currently handles pricing through:
- **Template-level estimates**: Hard-coded cost estimates in template definitions
- **API pricing lookups**: Real-time AWS Pricing API integration for current rates
- **Cost tracking**: Project-level budget monitoring and spend tracking
- **Regional pricing**: Different costs across AWS regions

### **Limitations of Current Approach**
- **Inaccurate estimates**: Public pricing doesn't reflect customer discounts
- **Budget confusion**: Real costs differ significantly from displayed estimates
- **Poor decision making**: Researchers make choices based on incorrect pricing
- **Enterprise friction**: Institutions can't rely on cost projections for budgeting

## Proposed Solution Architecture

### **Configuration-Based Discount System**

#### **1. Global Discount Configuration**
Allow customers to specify enterprise-wide discounts in their configuration:

```json
{
  "aws": {
    "region": "us-west-2",
    "profile": "research-account"
  },
  "pricing": {
    "discount_model": "enterprise",
    "global_discounts": {
      "ec2_compute": 0.25,     // 25% discount on EC2 compute
      "ebs_storage": 0.15,     // 15% discount on EBS storage  
      "efs_storage": 0.10,     // 10% discount on EFS storage
      "data_transfer": 0.20    // 20% discount on data transfer
    },
    "reserved_instance_coverage": 0.80,  // 80% of usage covered by RIs
    "savings_plan_coverage": 0.60       // 60% additional coverage by Savings Plans
  }
}
```

#### **2. Instance Family Specific Discounts**
Different discounts for different instance types based on customer agreements:

```json
{
  "pricing": {
    "instance_family_discounts": {
      "compute_optimized": {
        "c5": 0.30,      // 30% discount on c5 instances
        "c6i": 0.25,     // 25% discount on c6i instances
        "m5": 0.20       // 20% discount on m5 instances
      },
      "gpu_instances": {
        "p3": 0.40,      // 40% discount on p3 instances
        "p4": 0.35,      // 35% discount on p4 instances
        "g4dn": 0.45     // 45% discount on g4dn instances
      }
    }
  }
}
```

#### **3. Project-Level Budget Adjustments**
Allow projects to specify their own discount models:

```bash
# Create project with custom pricing model
cws project create ml-research \
  --budget 10000 \
  --pricing-model academic \
  --discount-rate 0.30

# Override global discounts for specific project
cws project set-pricing ml-research \
  --ec2-discount 0.40 \
  --gpu-discount 0.50
```

#### **4. Template-Level Discount Integration**
Templates can specify recommended discount assumptions:

```yaml
# templates/gpu-ml.yml
name: "gpu-ml"
description: "GPU machine learning environment"
pricing:
  recommended_discounts:
    ec2_compute: 0.35    # Assumes 35% GPU discount
    spot_usage: 0.70     # Assumes 70% spot instance usage
  cost_model: "enterprise"
estimated_cost_per_hour:
  x86_64: 0.50          # After discount pricing
  x86_64_list: 0.80     # List price for comparison
```

### **Implementation Strategy**

#### **Phase 1: Configuration Framework**
- Extend existing config system to support pricing discounts
- Add validation for discount ranges (0.0-1.0)
- Implement discount application to all cost calculations
- CLI commands for discount management

#### **Phase 2: Advanced Discount Models**
- Reserved Instance and Savings Plan modeling
- Spot instance pricing integration
- Academic/non-profit discount presets
- Multi-tier volume discount calculations

#### **Phase 3: Dynamic Pricing Integration**
- Real-time AWS Pricing API with discount overlays
- Automatic cost optimization recommendations
- Integration with AWS Cost Explorer for actual vs. estimated tracking
- Machine learning for discount optimization

### **CLI Interface Design**

#### **Discount Management Commands**
```bash
# Set global enterprise discount
cws config set-discount ec2-compute 25%
cws config set-discount ebs-storage 15%

# View current discount configuration
cws config show-discounts

# Set project-specific discounts
cws project set-discount ml-research --ec2 35% --gpu 50%

# Launch with discount-aware pricing
cws launch gpu-ml expensive-training --show-pricing
# → List Price: $2.40/hour
# → Your Price: $1.20/hour (50% enterprise discount)
# → Estimated Daily: $28.80 (after discounts)
```

#### **Cost Estimation Integration**
```bash
# All cost displays show discounted pricing
cws list
# NAME         TEMPLATE    STATE    YOUR_COST/DAY    LIST_COST/DAY
# ml-training  gpu-ml      running  $28.80          $57.60

# Budget tracking uses discounted costs
cws project budget ml-research
# → Budget: $10,000
# → Spent: $2,847 (at your discounted rates)  
# → Remaining: $7,153
# → List Price Equivalent: $5,694 spent
```

### **Enterprise Integration Features**

#### **1. Institutional Pricing Models**
Pre-configured discount models for common institutional agreements:

```json
{
  "pricing_models": {
    "aws_enterprise": {
      "ec2_compute": 0.25,
      "storage": 0.15,
      "data_transfer": 0.20
    },
    "academic_research": {
      "ec2_compute": 0.30,
      "gpu_instances": 0.40,
      "storage": 0.20
    },
    "startup_credits": {
      "all_services": 1.00,      // 100% discount (credits)  
      "credit_expiry": "2024-12-31"
    }
  }
}
```

#### **2. Budget Reconciliation**
Help institutions reconcile CloudWorkstation estimates with actual AWS bills:

```bash
# Compare CloudWorkstation estimates with actual AWS costs
cws billing reconcile --month 2024-01
# → CloudWorkstation Estimate: $12,450 (with your discounts)
# → Actual AWS Bill: $11,890
# → Variance: -4.5% (better than expected)
# → Suggested Discount Adjustment: Increase EC2 discount to 27%
```

#### **3. Cost Optimization Recommendations**
```bash
# Get personalized cost optimization based on actual discounts
cws optimize costs --project ml-research
# → Current Spend: $2,847/month (discounted)
# → Optimization Opportunities:
#   • Switch 60% of instances to Spot: Save $1,139/month
#   • Purchase Reserved Instances for ml5.large: Save $456/month  
#   • Use EFS Infrequent Access: Save $89/month
# → Total Potential Savings: $1,684/month (59% reduction)
```

### **Configuration Examples**

#### **Research University Configuration**
```json
{
  "pricing": {
    "model": "academic_research",
    "global_discounts": {
      "ec2_compute": 0.30,
      "gpu_instances": 0.40, 
      "storage": 0.20,
      "data_transfer": 0.25
    },
    "spot_preference": 0.70,        // Prefer spot for 70% of workloads
    "reserved_instance_target": 0.50, // Target 50% RI coverage
    "academic_credits": {
      "remaining": 50000,
      "expiry": "2024-12-31"
    }
  }
}
```

#### **Enterprise Customer Configuration**
```json
{
  "pricing": {
    "model": "enterprise_edp",
    "edp_discount": 0.28,           // 28% Enterprise Discount Program
    "committed_spend": 500000,      // $500K annual commitment
    "instance_family_discounts": {
      "compute": 0.25,
      "memory": 0.22, 
      "gpu": 0.35,
      "storage": 0.18
    },
    "cost_allocation_tags": {
      "Department": "required",
      "Project": "required",
      "CostCenter": "required"
    }
  }
}
```

### **Benefits for Different User Types**

#### **For Individual Researchers**
- **Accurate budgeting**: Real costs instead of inflated public pricing
- **Better decisions**: Choose instance types based on actual costs
- **Transparent spending**: Understand true research computing costs

#### **For Research Teams**
- **Project budgeting**: Accurate multi-month project cost planning
- **Resource optimization**: Make informed decisions about spot vs. on-demand
- **Cost allocation**: Fair sharing of costs among team members

#### **For Institutions**
- **Financial planning**: Align CloudWorkstation estimates with actual AWS bills  
- **Budget approval**: Present realistic costs to administrators
- **Compliance**: Meet institutional financial reporting requirements
- **Optimization**: Identify opportunities for additional cost savings

### **Implementation Priority**

This feature should be **high priority** because:

1. **Immediate value**: Fixes a major pain point for enterprise customers
2. **Adoption barrier**: Inaccurate pricing estimates prevent institutional adoption
3. **Competitive advantage**: Most cloud management tools don't handle custom pricing well
4. **Foundation feature**: Required for accurate project budgeting and cost optimization

### **Integration Points**

#### **Existing Systems**
- **Configuration management**: Extend `~/.cloudworkstation/config.json`
- **Project budgets**: Apply discounts to all project cost calculations  
- **Template pricing**: Override template cost estimates with discounted pricing
- **Cost tracking**: Use discounted rates for all spend monitoring

#### **AWS Services**
- **AWS Pricing API**: Overlay discounts on top of public pricing
- **AWS Cost Explorer**: Compare estimates with actual usage
- **AWS Budgets**: Align CloudWorkstation budgets with AWS budget alerts
- **AWS Organizations**: Inherit pricing from master account settings

---

## Roadmap Position

**Priority**: High (after project-based instance organization)
**Complexity**: Medium (configuration + pricing calculation updates)
**Impact**: High (essential for enterprise adoption)
**Dependencies**: None (builds on existing config system)

This feature transforms CloudWorkstation from "good estimates" to "accurate financial planning tool" - essential for enterprise customers who need to align cloud spending with institutional budgets and make data-driven decisions about research computing resources.