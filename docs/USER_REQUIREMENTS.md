# USER REQUIREMENTS - CRITICAL

**THESE REQUIREMENTS MUST BE FOLLOWED AT ALL TIMES**

## Core Requirements (User has stated these 3+ times)

### 1. NO TIME PRESSURE
- ALL ISSUES AND TODOs ARE TO BE COMPLETED **COMPLETELY AND CORRECTLY**
- **NO SKIPPING OVER ITEMS**
- Quality and completeness over speed

### 2. ZERO PLACEHOLDERS
- ❌ NO "For now, ..." comments
- ❌ NO "In production, this would..." comments
- ❌ NO "would be implemented..." comments
- ❌ NO "TODO: Implement..." markers
- ❌ NO "FIXME" markers
- ❌ NO context.TODO() in production code
- ✅ ONLY real, working implementations

### 3. ALL INTERFACES FUNCTIONAL
- ✅ CLI functionality must exist and be fully functional
- ✅ TUI functionality must exist and be fully functional
- ✅ GUI functionality must exist and be fully functional
- ✅ Complete feature parity across all three interfaces

### 4. COMPREHENSIVE TESTING
- ✅ Tests written for ALL implementations
- ✅ Tests tested against REAL AWS
  - Use AWS_PROFILE=aws
  - Use AWS_REGION=us-west-2
- ✅ TUI must have AWS integration tests
- ✅ GUI must have AWS integration tests
- ✅ All tests must pass against real AWS infrastructure

### 5. PERIODIC COMMITS
- ✅ Commit work frequently
- ✅ Update REMAINING_WORK.md document regularly
- ✅ Clear commit messages documenting progress

### 6. COBRA CLI MIGRATION
- ✅ Complete the CLI migration to Cobra
- ✅ Remove ALL legacy code after migration
- ✅ User wants to know when this request can be removed

### 7. REAL AWS IMPLEMENTATION
The user has emphasized this **multiple times**:
> "I TOLD YOU TO IMPLEMENT AND TEST AGAINST REAL AWS"

**What this means:**
- Every AWS feature must use real AWS SDK calls
- No mock data in production code
- No simulation functions
- No placeholder responses
- CloudWatch metrics must be real
- S3 operations must be real
- EC2 operations must be real
- All AWS services must be real

## Current Status

**Placeholder Count**: 145 across 58 files
**Current Progress**: 30% complete (12/40+ critical items fixed)
**Remaining Work**: Systematic elimination of 145 placeholders

## Work Approach

1. ✅ Create comprehensive audit (DONE)
2. 🔄 Phase 1: Fix critical 15 placeholders
3. 🔄 Phase 2: Fix high priority 38 placeholders
4. 🔄 Phase 3: Fix medium priority 48 placeholders
5. 🔄 Phase 4: Write comprehensive AWS tests
6. 🔄 Phase 5: Complete Cobra migration

## User Frustration Points

The user has expressed frustration when:
- I add NEW placeholders while "fixing" code
- I use comments like "In production, this would..."
- I don't fully implement AWS integrations
- I skip items or take shortcuts

## Success Criteria

✅ Zero placeholder comments
✅ Zero TODOs/FIXMEs in production code
✅ All CLI features functional
✅ All TUI features functional
✅ All GUI features functional
✅ All features tested against real AWS
✅ Cobra migration complete
✅ Legacy code removed
✅ All tests passing

---

**READ THIS FILE AFTER EVERY CONTEXT COMPRESSION**
