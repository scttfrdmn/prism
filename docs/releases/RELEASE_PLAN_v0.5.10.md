# Prism v0.5.10 Release Plan: Multi-Project Budgets

**Release Date**: Target February 14, 2026
**Focus**: Budget system redesign enabling multi-project allocation

## 🎯 Release Goals

### Primary Objective
Redesign the budget system to allow a single budget to be allocated across multiple projects, enabling realistic grant-funded research workflows.

**Current State**: 1 budget : 1 project (rigid, doesn't match research reality)
**New State**: 1 budget : N projects (flexible, matches grant funding model)

### Success Metrics
- Grant-funded research: Single NSF grant → 3-5 related projects
- Lab budgets: Department budget → 10+ research group projects
- Class budgets: Course budget → 30+ student project groups
- Budget reallocation: Move funds between projects within 60 seconds

---

## 📦 Features & Implementation

### 1. Shared Budget Pools
**Priority**: P0 (Core requirement)
**Effort**: Large (4-5 days)
**Impact**: Critical (Enables multi-project allocation)

**Current Budget Model**:
```go
type Budget struct {
    ID          string
    ProjectID   string  // 1:1 relationship
    Amount      float64
    Period      string
    AlertThreshold float64
}
```

**New Budget Model**:
```go
type Budget struct {
    ID              string
    Name            string        // "NSF Grant #12345", "Department Q1 Budget"
    Description     string
    TotalAmount     float64       // Total budget pool
    Period          string        // "monthly", "quarterly", "grant-period"
    StartDate       time.Time
    EndDate         *time.Time    // Optional for ongoing budgets
    AlertThreshold  float64       // Percentage for global alert
    CreatedBy       string        // User ID
    CreatedAt       time.Time
}

type ProjectBudgetAllocation struct {
    ID              string
    BudgetID        string        // Parent budget pool
    ProjectID       string        // Allocated project
    AllocatedAmount float64       // Amount allocated to this project
    SpentAmount     float64       // Current spending (cached)
    AlertThreshold  *float64      // Optional project-specific threshold
    Notes           string
    AllocatedAt     time.Time
    AllocatedBy     string        // User ID
}
```

**Database Schema**:
```sql
-- budgets table (existing, modified)
ALTER TABLE budgets ADD COLUMN name TEXT NOT NULL;
ALTER TABLE budgets ADD COLUMN description TEXT;
ALTER TABLE budgets DROP COLUMN project_id;  -- Remove 1:1 constraint
ALTER TABLE budgets ADD COLUMN start_date TIMESTAMP NOT NULL DEFAULT NOW();
ALTER TABLE budgets ADD COLUMN end_date TIMESTAMP;
ALTER TABLE budgets ADD COLUMN created_by TEXT;
ALTER TABLE budgets ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT NOW();

-- project_budget_allocations table (new)
CREATE TABLE project_budget_allocations (
    id TEXT PRIMARY KEY,
    budget_id TEXT NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    allocated_amount REAL NOT NULL CHECK (allocated_amount >= 0),
    spent_amount REAL NOT NULL DEFAULT 0,
    alert_threshold REAL,
    notes TEXT,
    allocated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    allocated_by TEXT NOT NULL,
    UNIQUE(budget_id, project_id)  -- One allocation per budget-project pair
);

CREATE INDEX idx_allocations_budget ON project_budget_allocations(budget_id);
CREATE INDEX idx_allocations_project ON project_budget_allocations(project_id);
```

**API Endpoints**:
```
POST   /api/v1/budgets                       # Create shared budget pool
GET    /api/v1/budgets                       # List all budgets
GET    /api/v1/budgets/{id}                  # Get budget with allocations
PUT    /api/v1/budgets/{id}                  # Update budget
DELETE /api/v1/budgets/{id}                  # Delete budget (cascades allocations)

POST   /api/v1/budgets/{id}/allocations      # Allocate budget to project
GET    /api/v1/budgets/{id}/allocations      # List allocations for budget
PUT    /api/v1/allocations/{id}              # Update allocation amount
DELETE /api/v1/allocations/{id}              # Remove project from budget

GET    /api/v1/projects/{id}/budget          # Get project's budget allocation
```

**Implementation Tasks**:
- [ ] Update `pkg/types/project.go` with new budget structures
- [ ] Create database migration script
- [ ] Update `pkg/project/budget_tracker.go` for multi-project tracking
- [ ] Implement allocation validation (total allocations ≤ budget amount)
- [ ] Add allocation API endpoints in `pkg/daemon/project_handlers.go`
- [ ] Update cost calculation to aggregate by allocation
- [ ] Add budget pool summary (allocated, spent, remaining)

---

### 2. Project Budget Allocation Interface
**Priority**: P0 (Core UX)
**Effort**: Medium (3-4 days)
**Impact**: Critical (Primary user workflow)

**GUI Components**:

#### Budget Creation Dialog
```typescript
// cmd/prism-gui/frontend/src/components/BudgetCreateDialog.tsx
interface BudgetCreateForm {
  name: string;              // "NSF Grant CISE-2024-12345"
  description: string;       // Grant details, purpose
  totalAmount: number;       // $50,000
  period: 'monthly' | 'quarterly' | 'annual' | 'grant-period' | 'custom';
  startDate: Date;
  endDate?: Date;            // Optional for ongoing budgets
  alertThreshold: number;    // 80% (global)
}

Features:
- Budget name and description
- Total amount with currency formatting
- Period selection (preset + custom date range)
- Global alert threshold
- Initial project allocations (optional)
```

#### Budget Management Page
```typescript
// cmd/prism-gui/frontend/src/pages/Budgets.tsx
Features:
- Budget pool cards showing:
  - Name and description
  - Total amount vs spent
  - Number of projects allocated
  - Remaining unallocated funds
  - Alert status
  - Period and dates
- Create new budget button
- Filter by period, status, date range
- Search by name/description
```

#### Budget Allocation Dialog
```typescript
// cmd/prism-gui/frontend/src/components/BudgetAllocationDialog.tsx
interface AllocationForm {
  budgetId: string;          // Selected budget pool
  projects: Array<{
    projectId: string;
    name: string;
    allocatedAmount: number;
    currentSpending: number;
    notes: string;
  }>;
  validateTotal: boolean;    // Ensure allocations ≤ budget
}

Features:
- Multi-project allocation table
- Amount input per project
- Real-time validation (total ≤ budget amount)
- Project spending preview
- Bulk allocation (equal split, percentage-based)
- Notes per allocation
```

#### Project Budget Tab
```typescript
// cmd/prism-gui/frontend/src/pages/ProjectDetail.tsx (Budget Tab)
Features:
- Budget source information (parent budget pool)
- Allocated amount
- Current spending (with progress bar)
- Alert threshold (inherited or custom)
- Spending breakdown by service (EC2, EBS, EFS, etc.)
- Spending timeline chart
- Budget reallocation request (if not owner)
```

**Cloudscape Components**:
- `Table` with editable cells for allocation amounts
- `ProgressBar` for budget utilization
- `Alert` for budget threshold warnings
- `Modal` for create/edit dialogs
- `SpaceBetween` for form layout
- `FormField` with validation

**Implementation Tasks**:
- [ ] Create BudgetCreateDialog component
- [ ] Create BudgetManagementPage component
- [ ] Create BudgetAllocationDialog component
- [ ] Update ProjectDetail budget tab
- [ ] Add budget pool summary cards
- [ ] Implement allocation validation UI
- [ ] Add spending visualization charts

---

### 3. Budget Reallocation
**Priority**: P1 (Flexibility)
**Effort**: Small (2 days)
**Impact**: High (Real-world workflow)

**Use Cases**:
1. **Grant Budget Adjustment**: Project A is over budget, move $5k from Project B
2. **Research Pivot**: Project paused, reallocate funds to active projects
3. **End of Period**: Consolidate unused allocations

**Reallocation Workflow**:
```typescript
interface ReallocationRequest {
  sourceAllocationId: string;   // Project losing funds
  targetAllocationId: string;   // Project gaining funds
  amount: number;               // Amount to transfer
  reason: string;               // Audit trail
}

// API:
POST /api/v1/allocations/reallocate
{
  "sourceAllocationId": "alloc-1",
  "targetAllocationId": "alloc-2",
  "amount": 5000.00,
  "reason": "Project A exceeded compute budget for ML training"
}
```

**GUI Features**:
- Budget Reallocation Dialog (drag-and-drop between projects)
- Reallocation history table
- Audit trail with reason and timestamp
- Validation: source allocation has sufficient remaining funds

**Implementation Tasks**:
- [ ] Add reallocation API endpoint
- [ ] Implement atomic reallocation transaction
- [ ] Create ReallocationDialog component
- [ ] Add reallocation history view
- [ ] Add audit trail logging

---

### 4. Multi-Project Cost Rollup & Reporting
**Priority**: P1 (Visibility)
**Effort**: Medium (2-3 days)
**Impact**: High (Decision-making)

**Budget Dashboard Features**:
```typescript
interface BudgetDashboard {
  budget: Budget;
  totalAllocated: number;       // Sum of all allocations
  totalSpent: number;           // Sum of all project spending
  totalRemaining: number;       // Budget - spent
  unallocatedFunds: number;     // Budget - allocated
  allocations: Array<{
    project: Project;
    allocated: number;
    spent: number;
    remaining: number;
    percentUsed: number;
    alertStatus: 'ok' | 'warning' | 'critical';
  }>;
  spendingTimeline: Array<{
    date: Date;
    cumulative: number;
  }>;
  topSpenders: Project[];       // Top 5 projects by spending
}
```

**Visualizations**:
1. **Budget Utilization Chart**: Allocated vs Spent vs Remaining
2. **Project Spending Breakdown**: Pie chart of spending by project
3. **Spending Timeline**: Cumulative spending over time vs budget burn rate
4. **Service Cost Breakdown**: EC2, EBS, EFS costs across all projects
5. **Alert Status**: Projects approaching or exceeding allocations

**Export Features**:
- CSV export for accounting systems
- PDF report for grant reporting
- JSON API for custom integrations

**Implementation Tasks**:
- [ ] Create BudgetDashboard component
- [ ] Implement cost aggregation queries
- [ ] Add spending timeline calculation
- [ ] Create visualization components (charts)
- [ ] Add CSV/PDF export functionality
- [ ] Optimize performance for large project counts

---

### 5. Migration & Compatibility
**Priority**: P0 (Required for release)
**Effort**: Small (1-2 days)
**Impact**: Critical (Existing users)

**Migration Strategy**:
Since user feedback specifies "no need for backwards compatibility", we can simplify:

**Direct Migration**:
```sql
-- Migrate existing budgets
-- Each existing budget becomes a dedicated budget pool with single allocation
BEGIN TRANSACTION;

-- Add new columns to budgets table
ALTER TABLE budgets ADD COLUMN name TEXT;
ALTER TABLE budgets ADD COLUMN description TEXT;
UPDATE budgets SET name = 'Budget for ' || project_id WHERE name IS NULL;

-- Create project_budget_allocations table
CREATE TABLE project_budget_allocations (...);

-- Migrate existing budgets to allocations
INSERT INTO project_budget_allocations (
    id, budget_id, project_id, allocated_amount, allocated_by
)
SELECT
    'alloc-' || budget_id,
    budget_id,
    project_id,
    amount,
    'system-migration'
FROM budgets;

-- Remove project_id from budgets
ALTER TABLE budgets DROP COLUMN project_id;

COMMIT;
```

**API Migration**:
- Old endpoint: `GET /api/v1/projects/{id}/budget` → Returns allocation data
- New endpoints: Budget management APIs
- Remove deprecated endpoints in v0.6.0

**Implementation Tasks**:
- [ ] Create migration script
- [ ] Test migration with sample data
- [ ] Update API handlers for new schema
- [ ] Remove deprecated code paths
- [ ] Document breaking changes in release notes

---

## 📅 Implementation Schedule

### Week 1 (Feb 1-7): Backend & Data Model
**Days 1-2**: Database schema and migration
- Design new budget/allocation schema
- Write migration script
- Test with sample data

**Days 3-5**: API Implementation
- Update types and data structures
- Implement budget pool CRUD endpoints
- Implement allocation endpoints
- Add validation logic
- Write unit tests

### Week 2 (Feb 8-14): Frontend & Integration
**Days 1-2**: Budget Management UI
- BudgetCreateDialog component
- BudgetManagementPage component
- Budget pool summary cards

**Days 3-4**: Allocation UI
- BudgetAllocationDialog component
- Project budget tab updates
- Spending visualizations

**Day 5**: Testing & Polish
- Extended persona walkthroughs
- Performance testing (100+ projects per budget)
- Bug fixes
- Documentation

---

## 🧪 Testing Strategy

### Backend Testing
- [ ] Budget pool CRUD operations
- [ ] Allocation validation (total ≤ budget)
- [ ] Reallocation atomic transactions
- [ ] Cost aggregation accuracy
- [ ] Migration script (existing budgets → allocations)
- [ ] Performance (100+ projects, 10+ budgets)

### Frontend Testing
- [ ] Budget creation workflow
- [ ] Multi-project allocation
- [ ] Budget reallocation
- [ ] Spending visualization
- [ ] Alert threshold triggers
- [ ] CSV/PDF export

### Persona Walkthroughs

#### Grant-Funded Research (Multi-Project)
**Scenario**: PI receives $50k NSF grant for 3 related projects

1. Create budget pool "NSF CISE-2024-12345" ($50,000)
2. Allocate $20k to "Project A: Algorithm Development"
3. Allocate $15k to "Project B: User Study"
4. Allocate $10k to "Project C: Paper Experiments"
5. Launch workspaces under each project
6. Monitor spending across all 3 projects
7. Reallocate $5k from Project C to Project A (algorithm needs more compute)
8. Generate grant report showing spending by project

#### Lab Budget (Department Allocation)
**Scenario**: Lab manager receives $100k department budget for research group

1. Create budget pool "CS Lab Q1 2026" ($100,000)
2. Allocate $40k to "Genomics Project" (Prof. Smith)
3. Allocate $30k to "Climate Modeling" (Prof. Jones)
4. Allocate $20k to "ML Research" (Prof. Lee)
5. Leave $10k unallocated for emergency use
6. Each PI manages their project independently
7. Lab manager monitors total spending
8. Reallocate unused funds at end of quarter

#### Class Budget (Student Projects)
**Scenario**: Professor receives $5k for CS499 class with 25 student projects

1. Create budget pool "CS499 Spring 2026" ($5,000)
2. Allocate $150 per student project (25 × $150 = $3,750)
3. Leave $1,250 for instructor demos and reserves
4. Students launch workspaces under their projects
5. Monitor spending to prevent overruns
6. Hibernate/stop student workspaces approaching limits
7. Generate per-student spending report for grading

---

## 📚 Documentation Updates

### New Documentation
- [ ] Multi-project budget guide
- [ ] Budget allocation best practices
- [ ] Grant reporting workflow
- [ ] Budget reallocation tutorial

### Updated Documentation
- [ ] Budget management section
- [ ] Project budget tab documentation
- [ ] API reference (new endpoints)
- [ ] Migration guide (v0.5.9 → v0.5.10)

### Release Notes
- [ ] Breaking changes (budget schema)
- [ ] New features (multi-project budgets)
- [ ] Migration instructions
- [ ] API changes

---

## 🚀 Release Criteria

### Must Have (Blocking)
- ✅ Shared budget pools implemented
- ✅ Project allocation system working
- ✅ Budget reallocation functional
- ✅ Cost aggregation accurate
- ✅ Migration script tested
- ✅ All persona tests pass
- ✅ Documentation complete

### Nice to Have (Non-Blocking)
- Budget templates (common allocations)
- Budget forecasting (burn rate projection)
- Advanced reporting (custom date ranges)
- Budget approval workflow (for institutions)

---

## 📊 Success Metrics (Post-Release)

Track for 2 weeks after release:

1. **Multi-Project Adoption**
   - Measure: % of budgets with 2+ projects
   - Target: >60% of new budgets

2. **Reallocation Usage**
   - Measure: Number of reallocations per budget
   - Target: Average 2-3 reallocations per grant-period budget

3. **Budget Utilization**
   - Measure: % of allocated funds actually spent
   - Target: >85% utilization (reduced waste)

4. **Time to Allocate**
   - Measure: Time from budget creation to full allocation
   - Target: <5 minutes for 10 projects

5. **Support Tickets**
   - Measure: Budget-related confusion tickets
   - Target: <5% of total support volume

---

## 🔗 Related Documents

- ROADMAP.md - Overall project roadmap
- RELEASE_PLAN_v0.5.9.md - Navigation Restructure (prerequisite)
- RELEASE_PLAN_v0.5.11.md - User Invitation & Roles (follows this)
- User Guide: Budget Management (to be updated)

---

**Last Updated**: October 27, 2025
**Status**: 📋 Planned
**Dependencies**: v0.5.9 (Navigation Restructure)
