# Prism Design Principles

> These principles ensure Prism remains simple, intuitive, and researcher-focused.

## 🎯 **Default to Success**
Every template must work out of the box in every supported region.
```bash
prism launch python-ml my-project  # This should always work
```

## ⚡ **Optimize by Default**
Templates automatically choose the best configuration for their workload.
- ML templates → GPU instances
- R templates → Memory-optimized
- Cost-performance optimized for academics

## 🔍 **Transparent Fallbacks**
When ideal config isn't available, users know what changed and why.
```
🏗️ Architecture fallback: arm64 → x86_64 (regional availability)
💡 ARM GPU not available in us-west-1, using x86 GPU instead
```

## 💡 **Helpful Warnings**
Gentle guidance for suboptimal choices with clear alternatives.
```
⚠️ Size S has no GPU - consider GPU-S for ML workloads
Continue with S? [y/N] or use GPU-S? [G]: 
```

## 🚫 **Zero Surprises**
Clear communication about what's happening.
- Configuration preview before launch
- Real-time progress reporting
- Accurate cost estimates
- Dry-run validation

## 📈 **Progressive Disclosure**
Simple by default, detailed when needed.
```bash
# Simple (90% of users)
prism launch template-name project-name

# Intermediate (power users)
prism launch template-name project-name --size L

# Advanced (infrastructure experts)  
prism launch template-name project-name --instance-type c5.2xlarge --spot
```

---

## Development Guidelines

### ✅ Do
- Make the common case trivial
- Provide actionable error messages
- Test the happy path first
- Default to the most cost-effective option
- Explain what's happening during long operations

### ❌ Avoid
- Requiring configuration for basic usage
- Silent fallbacks or failures
- Technical jargon in user-facing messages
- Surprising users with costs or instance types
- Adding complexity to simple workflows

---

*These principles guide every feature decision and code change in Prism.*