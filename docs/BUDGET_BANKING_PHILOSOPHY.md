# Budget Banking Philosophy

## The Core Value Proposition

**Prism's budget banking system is the killer feature that makes cloud computing viable for academic research.**

Unlike traditional owned hardware (fixed amortization costs), cloud computing allows researchers to "bank" unused budget for future bursting. This document explains how Prism leverages this fundamental cloud advantage to eliminate the primary hesitation academics have about AWS.

---

## The Problem Prism Solves

Academic researchers receive grant funding with fixed budgets over specific time periods (quarterly, annually, or grant-specific periods). They face several challenges:

1. **Budget Shortfall Fear**: "What if I run out of budget before the period ends?"
2. **Committed Costs Anxiety**: "Storage costs continue even when instances are stopped"
3. **Unpredictable Workloads**: Research requires bursting (intensive computation) followed by quiet periods (data analysis)
4. **Grant Reporting Requirements**: Need to demonstrate responsible budget management

Traditional cloud billing makes these problems worse because researchers lack visibility into their spending trajectory and can't easily predict whether they'll have budget through the end of their grant period.

---

## The Budget Banking Solution

### Core Concept: Surplus Accumulation

When you stop an instance, you don't just reduce your burn rate - you **bank the difference** between actual spend and target spend. This creates a **surplus** that can be used for bursting above the target rate when needed.

### Mathematical Foundation

```
Budget Period: $10,000 over 90 days (quarterly grant)
Target Burn Rate: $111.11/day

Week 1-2 (14 days): Light usage
- Running: 1× t3.medium ($0.04/hr) + 100GB EBS ($0.10/day) = $14.54/day
- Target: $111.11/day × 14 days = $1,555.54
- Actual: $14.54/day × 14 days = $203.56
- SURPLUS BANKED: $1,351.98

Week 3-4 (14 days): Data processing
- Running: 3× c5.4xlarge ($0.68/hr) + 500GB EBS ($0.50/day) = $49.34/day
- Target: $111.11/day × 14 days = $1,555.54
- Actual: $49.34/day × 14 days = $690.76
- ADDITIONAL SURPLUS: $864.78
- TOTAL SURPLUS: $2,216.76

Week 5 (7 days): BURST - GPU Training
- Running: 2× p3.2xlarge ($3.06/hr) + 1TB EBS ($1.00/day) = $147.88/day
- Target: $111.11/day × 7 days = $777.77
- Actual: $147.88/day × 7 days = $1,035.16
- Surplus used: $257.39
- REMAINING SURPLUS: $1,959.37

Week 6-13 (56 days): Analysis & writing
- Stopped: 0× instances + 1TB EBS ($1.00/day) = $1.00/day
- Target: $111.11/day × 56 days = $6,222.16
- Actual: $1.00/day × 56 days = $56.00
- MASSIVE SURPLUS: $6,166.16
- TOTAL SURPLUS: $8,125.53

Quarter End Budget Status:
- Total Spent: $1,985.48
- Total Budget: $10,000
- Remaining: $8,014.52
- Researcher stayed WELL under budget
- Storage costs fully covered
- GPU burst easily accommodated
```

### Key Advantages

**1. Burst Capability**
- Bank budget during quiet periods (writing, data analysis)
- Burst during compute-intensive periods (model training, simulations)
- No anxiety about temporary overspending

**2. Storage Cost Protection**
- EBS, EFS, S3 costs continue regardless of instance state
- Surplus ensures storage always covered
- Optional "cushion" provides safety buffer beyond period end

**3. Shortfall Prevention**
- Real-time visibility: "Are you on track?"
- Predictive analytics: "Will you hit zero before period end?"
- Early warning: "Adjust usage now to avoid shortfall"

**4. Cloud Advantage Realization**
- Unlike owned hardware (use or lose), cloud lets you bank savings
- Stopped instance = banked budget for future use
- Aligns perfectly with research workflow patterns

---

## The Safety Net: Budget Cushion

Researchers can optionally configure a "cushion" - reserved budget beyond the period end date to ensure storage continuity even if compute budget exhausted.

```
Budget: $10,000 for 90 days
Cushion: 14 days (2 weeks)
Storage Costs: $5/day (EBS + EFS + S3)

Calculation:
- Reserve for cushion: $5/day × 14 days = $70
- Available for period: $10,000 - $70 = $9,930
- Target burn rate: $9,930 / 90 days = $110.33/day

If budget exhausted on day 90:
- Cushion covers storage for days 91-104
- Researcher has 2 weeks to secure additional funding
- No data loss, no emergency scrambling
```

### Cushion Use Cases

1. **Grant Transitions**: Bridge funding between grant periods
2. **Emergency Reserve**: Unexpected expenses in final weeks
3. **Storage Continuity**: Ensure data accessible during extension requests
4. **Peace of Mind**: "Safety blanket" for budget anxiety

---

## Academic-Specific Features

### GDEW (Global Data Egress Waiver) Integration

AWS provides academic researchers with automatic credits for up to 15% of monthly spend on egress charges. For 99% of research workloads, this effectively eliminates egress costs.

```
Monthly Spend: $3,333 (one month of quarterly budget)
Egress Charges: $250 (data downloads, inter-region transfers)
GDEW Credit: min($250, $3,333 × 0.15) = min($250, $500) = $250
Net Egress Cost: $0

Effective Impact:
- Egress fully covered by GDEW
- No impact on research budget
- Researcher can freely download results
```

**Prism GDEW Tracking**:
- Optional checkbox: "I have AWS GDEW"
- Automatic monthly tracking window
- Real-time estimation of available credits
- Alerts when approaching 15% threshold
- Proactive guidance: "Egress on track to be covered"

### Discount & Credit Support

**Enterprise Discount Programs (EDP)**:
- Across-the-board percentage discount
- Applied to all services
- Discoverable via AWS Cost Explorer API

**Private Pricing Agreements (PPA)**:
- Service-specific discounts (often S3, EC2 Reserved Instances)
- Custom pricing negotiated with AWS
- Applied automatically in billing

**Time-Limited Credits**:
- Research grants, promotional credits, support credits
- Have start date, expiration date, remaining balance
- Discoverable via AWS Billing API
- Track utilization: "You have $500 in credits expiring next month"

---

## Implementation Priorities

### Phase 1: Foundation (v0.5.9)
1. Budget period tracking (start date, end date, total budget)
2. Target burn rate calculation
3. Surplus/deficit tracking
4. Basic "on track" indicator

### Phase 2: Storage Costs (v0.5.10)
1. EBS volume cost tracking (independent of instances)
2. EFS filesystem usage monitoring
3. S3 bucket storage tracking
4. Committed costs calculation

### Phase 3: Analytics & Prediction (v0.5.11)
1. Burn rate trend analysis
2. Shortfall prediction engine
3. Surplus projection
4. Budget cushion configuration

### Phase 4: Academic Features (v0.5.12)
1. GDEW tracking and estimation
2. Discount/credit discovery and application
3. Calendar month alignment
4. Grant reporting exports

### Phase 5: Alerting & Integration (v0.5.13)
1. Unified alerting system (apprise-go integration)
2. Alert templates (burn rate, surplus, shortfall, GDEW)
3. Multi-channel delivery (email, Slack, MS Teams, etc.)
4. Customizable thresholds

---

## Why This Matters

**Quote from the creator:**

> "This is SO CORE to this whole project it is hard to convey to you. This alone breaks the hesitation people have with the cloud (AWS). Plus, if you add in an optional cushion... folks have a safety blanket."

Budget banking transforms cloud computing from a source of anxiety ("Am I spending too much?") to a source of empowerment ("I have surplus to burst when I need it"). It's the difference between:

- **Traditional approach**: "I'm afraid to launch a GPU instance because I might run out of budget"
- **Prism approach**: "I have $2,000 in surplus, so I can safely burst for 3 days of GPU training"

This psychological shift - from fear to confidence - is what makes Prism viable for academic research.

---

## Technical Architecture

See related issues for implementation details:
- #TBD: Budget Period & Burn Rate Foundation
- #TBD: Surplus/Banking Calculation Engine
- #TBD: EBS Cost Tracking
- #TBD: EFS Cost Tracking
- #TBD: S3 Cost Tracking & Egress
- #TBD: GDEW Integration
- #TBD: Discount & Credit Discovery
- #TBD: Predictive Analytics & Shortfall Detection
- #TBD: Unified Alerting System (apprise-go)

---

## Appendix: Real-World Example

**Dr. Sarah Chen - Computational Biology Research**

Grant: NIH R01 ($15,000 AWS budget for 12 months)

**Month 1-3: Dataset preparation**
- 1× t3.large running 24/7 ($60/month)
- 2TB EBS storage ($200/month)
- Target: $1,250/month
- Actual: $260/month
- **Surplus banked: $2,970**

**Month 4: Model training burst**
- 4× p3.8xlarge running 10 days ($5,000)
- 5TB EBS temporary storage ($500)
- Target: $1,250
- Actual: $5,500
- **Surplus used: $4,250**
- **Remaining surplus: +$720** (still in surplus!)

**Month 5-12: Analysis & writing**
- All instances stopped
- 2TB EBS storage only ($200/month × 8 = $1,600)
- Target: $1,250/month × 8 = $10,000
- Actual: $1,600
- **Additional surplus: $8,400**

**End of grant:**
- Total spent: $7,360
- Total budget: $15,000
- **Under budget by $7,640 (51%)**
- Researcher demonstrated responsible budget management
- All data preserved on EBS throughout entire period
- GPU burst completed without anxiety
- Grant renewal shows efficient resource utilization

**Without budget banking visibility:**
- Dr. Chen would have hesitated to launch GPUs
- Training might have been delayed or compromised
- Anxiety about storage costs throughout quiet periods
- Uncertainty about budget status until monthly bill

**With Prism budget banking:**
- Confidence to burst when needed
- Clear visibility into surplus
- No anxiety during quiet periods
- Real-time "on track" status
- Professional grant reporting

This is the transformation Prism enables.
