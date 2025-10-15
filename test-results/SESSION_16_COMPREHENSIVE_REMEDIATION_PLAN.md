# CloudWorkstation GUI: Comprehensive Remediation Plan
## UX Improvements + WCAG 2.2 Accessibility Compliance

**Date**: October 13, 2025
**Scope**: Integrated remediation of UX issues and accessibility violations
**Timeline**: 3-week sprint plan (85-95 hours total)
**Status**: 📋 **READY FOR IMPLEMENTATION**

---

## Executive Summary

This comprehensive plan integrates **18 UX improvements** and **17 accessibility fixes** into a prioritized, sprint-based implementation schedule. The plan ensures CloudWorkstation GUI meets both user experience best practices and WCAG 2.2 Level AA compliance standards before production launch.

### Overall Statistics

**Total Issues**: 35 items (18 UX + 17 Accessibility)
**Total Effort**: 85-95 hours (~12-13 days)
**Critical Path**: 21 hours (P0 items - required before launch)
**Launch Blockers**: 8 items (must complete before production)

### Compliance Goals

**Current State**:
- UX Quality: ⭐⭐⭐⭐☆ (4/5)
- WCAG 2.2 Level A: ~60-70% compliant
- WCAG 2.2 Level AA: ~50-60% compliant

**Target State** (After remediation):
- UX Quality: ⭐⭐⭐⭐⭐ (5/5)
- WCAG 2.2 Level A: 100% compliant
- WCAG 2.2 Level AA: 95%+ compliant

---

## Priority Framework

### Priority Definitions

**P0 (Critical - Launch Blockers)**:
- Blocks production deployment
- Legal/compliance risk
- Data loss risk
- Critical accessibility violations
- Timeline: Must complete before launch (~21 hours)

**P1 (High - Week 1)**:
- Significantly impacts user experience
- Required for WCAG Level AA
- High user impact
- Timeline: Complete in first week post-launch (~34 hours)

**P2 (Important - Weeks 2-3)**:
- Important usability improvements
- Enhanced accessibility features
- Power user features
- Timeline: Complete within 2-3 weeks (~30 hours)

**P3 (Nice to Have - Future)**:
- Polish and refinement
- Advanced features
- Optional enhancements
- Timeline: Post-launch iterations

---

## Sprint Plan Overview

```
┌─────────────────────────────────────────────────────────────┐
│ SPRINT 0: Pre-Launch Critical Fixes (P0)                   │
│ Duration: 3 days | Effort: 21 hours | Items: 8             │
│ Goal: Production-ready safety & accessibility              │
└─────────────────────────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────────────────────────┐
│ SPRINT 1: High Priority UX + Accessibility (P1)            │
│ Duration: 5 days | Effort: 34 hours | Items: 11            │
│ Goal: WCAG AA compliance + major UX improvements           │
└─────────────────────────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────────────────────────┐
│ SPRINT 2: Important Enhancements (P2)                      │
│ Duration: 7 days | Effort: 30 hours | Items: 11            │
│ Goal: Power user features + enhanced accessibility         │
└─────────────────────────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────────────────────────┐
│ BACKLOG: Future Enhancements (P3)                          │
│ Duration: TBD | Effort: ~25 hours | Items: 5               │
│ Goal: Polish and advanced features                         │
└─────────────────────────────────────────────────────────────┘
```

---

## SPRINT 0: Pre-Launch Critical Fixes (P0)

**Duration**: 3 days (21 hours)
**Goal**: Make GUI safe and accessible for production launch
**Launch Blocker**: ❌ Cannot deploy without completing this sprint

### Sprint 0 Items

#### UX-P0-1: Delete Confirmations (SAFETY CRITICAL)

**Issue**: No confirmation for destructive actions
**Impact**: Users can accidentally delete instances/volumes/projects
**Risk**: Data loss, support escalations, user frustration

**Implementation**:
```typescript
// File: src/components/ConfirmDeleteModal.tsx
interface ConfirmDeleteModalProps {
  visible: boolean;
  resourceType: 'instance' | 'volume' | 'project' | 'user';
  resourceName: string;
  requireNameConfirmation?: boolean;
  onConfirm: () => void;
  onDismiss: () => void;
  additionalWarning?: string;
}

export const ConfirmDeleteModal: React.FC<ConfirmDeleteModalProps> = ({
  visible,
  resourceType,
  resourceName,
  requireNameConfirmation = false,
  onConfirm,
  onDismiss,
  additionalWarning
}) => {
  const [confirmText, setConfirmText] = useState('');
  const canConfirm = !requireNameConfirmation || confirmText === resourceName;

  return (
    <Modal
      visible={visible}
      onDismiss={onDismiss}
      header={`Delete ${resourceType}?`}
      footer={
        <Box float="right">
          <SpaceBetween direction="horizontal" size="xs">
            <Button variant="link" onClick={onDismiss}>
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={onConfirm}
              disabled={!canConfirm}
            >
              Delete {resourceType}
            </Button>
          </SpaceBetween>
        </Box>
      }
    >
      <SpaceBetween size="m">
        <Alert type="warning" header="This action cannot be undone">
          You are about to permanently delete <strong>{resourceName}</strong>.
          {additionalWarning && <Box padding={{ top: 's' }}>{additionalWarning}</Box>}
        </Alert>

        {requireNameConfirmation && (
          <FormField
            label={`Type "${resourceName}" to confirm deletion`}
            description="This is a critical resource. Confirm by typing its exact name."
          >
            <Input
              value={confirmText}
              onChange={({ detail }) => setConfirmText(detail.value)}
              placeholder={resourceName}
              autoFocus
            />
          </FormField>
        )}
      </SpaceBetween>
    </Modal>
  );
};
```

**Integration Points**:
- Instance delete actions
- EFS volume delete
- EBS volume delete
- Project delete
- Research user delete

**Testing**:
- [ ] Test all delete flows
- [ ] Verify modal keyboard navigation (ESC to cancel)
- [ ] Test name confirmation for critical resources
- [ ] Verify focus returns after dismiss

**Effort**: 2-3 hours
**Priority**: P0 (CRITICAL)

---

#### UX-P0-2: Fix API Error Logging

**Issue**: Console cluttered with `/api/v1/rightsizing/stats` 400 errors
**Impact**: Hides real errors, unprofessional, confusing to developers

**Implementation**:
```typescript
// File: src/App.tsx
async getRightsizingStats(): Promise<RightsizingStats | null> {
  try {
    const data = await this.safeRequest('/api/v1/rightsizing/stats');
    return data;
  } catch (error: any) {
    // Silently handle 400/404 - endpoint may not be implemented yet
    if (error.status === 400 || error.status === 404) {
      return null; // Don't log, just return null
    }
    // Only log unexpected errors (500, network failures, etc.)
    console.error('Unexpected error fetching rightsizing stats:', error);
    return null;
  }
}

// Apply same pattern to all optional API endpoints
async getAMIBuilds(): Promise<AMIBuild[]> {
  try {
    const data = await this.safeRequest('/api/v1/ami/builds');
    return data || [];
  } catch (error: any) {
    if (error.status === 400 || error.status === 404) {
      return []; // Endpoint not implemented, silent fail
    }
    console.error('Unexpected error fetching AMI builds:', error);
    return [];
  }
}
```

**Affected Endpoints**:
- `/api/v1/rightsizing/stats` (currently failing)
- `/api/v1/ami/builds` (may not exist)
- Any other optional/future endpoints

**Testing**:
- [ ] Launch GUI and check console
- [ ] Verify no 400/404 errors logged
- [ ] Verify 500 errors still logged
- [ ] Test with daemon missing optional endpoints

**Effort**: 1 hour
**Priority**: P0 (CRITICAL for production)

---

#### A11Y-P0-1: Add Skip Navigation Link

**WCAG**: 2.4.1 Bypass Blocks (Level A)
**Issue**: Keyboard users must tab through entire side nav to reach content
**Impact**: Violates WCAG Level A, poor keyboard UX

**Implementation**:
```typescript
// File: src/components/SkipLink.tsx
export const SkipLink: React.FC = () => {
  return (
    <a
      href="#main-content"
      className="skip-link"
      style={{
        position: 'absolute',
        left: '-10000px',
        top: '10px',
        zIndex: 9999,
        padding: '8px 16px',
        backgroundColor: '#0972d3',
        color: 'white',
        textDecoration: 'none',
        borderRadius: '4px',
        fontSize: '14px',
        fontWeight: 'bold',
      }}
      onFocus={(e) => {
        // Show on focus
        e.currentTarget.style.left = '10px';
      }}
      onBlur={(e) => {
        // Hide on blur
        e.currentTarget.style.left = '-10000px';
      }}
    >
      Skip to main content
    </a>
  );
};

// File: src/App.tsx
function App() {
  return (
    <>
      <SkipLink />
      <AppLayout
        navigation={...}
        content={
          <main id="main-content" tabIndex={-1}>
            {renderContent()}
          </main>
        }
      />
    </>
  );
}
```

**Testing**:
- [ ] Press Tab from page load → see skip link
- [ ] Press Enter → focus moves to main content
- [ ] Verify skip link hidden when not focused
- [ ] Test with screen reader (VoiceOver/NVDA)

**Effort**: 1 hour
**Priority**: A11Y-P0 (WCAG Level A violation)

---

#### A11Y-P0-2: Add Landmark Roles

**WCAG**: 1.3.1 Info and Relationships (Level A)
**Issue**: No semantic landmarks for screen reader navigation
**Impact**: Screen reader users can't navigate by landmarks

**Implementation**:
```typescript
// File: src/App.tsx
<AppLayout
  navigation={
    <nav aria-label="Main navigation" role="navigation">
      <SideNavigation
        header={{ text: 'CloudWorkstation', href: '#/' }}
        items={navItems}
        activeHref={`#${state.activeView}`}
      />
    </nav>
  }
  content={
    <main id="main-content" tabIndex={-1} role="main">
      {state.activeView === 'dashboard' && <DashboardView />}
      {state.activeView === 'templates' && <TemplateSelectionView />}
      {/* ... other views */}
    </main>
  }
  notifications={
    <Flashbar items={state.notifications} />
  }
/>
```

**Testing**:
- [ ] Use screen reader landmark navigation
- [ ] Verify "Main" landmark exists
- [ ] Verify "Navigation" landmark exists
- [ ] Test with VoiceOver rotor (⌘ U)

**Effort**: 1 hour
**Priority**: A11Y-P0 (WCAG Level A violation)

---

#### A11Y-P0-3: Status Indicator Labels

**WCAG**: 1.1.1 Non-text Content (Level A)
**Issue**: Visual-only status (colors) without text alternatives
**Impact**: Screen readers don't convey status, color-blind users miss info

**Implementation**:
```typescript
// File: src/utils/statusUtils.ts
export const getStatusLabel = (state: string, resourceType: string = 'instance'): string => {
  const labels = {
    'running': `${resourceType} is running`,
    'stopped': `${resourceType} is stopped`,
    'stopping': `${resourceType} is stopping`,
    'starting': `${resourceType} is starting`,
    'terminated': `${resourceType} is terminated`,
    'pending': `${resourceType} is pending`,
    'available': `${resourceType} is available`,
    'in-use': `${resourceType} is in use`,
    'deleting': `${resourceType} is being deleted`,
  };
  return labels[state] || `${resourceType} status: ${state}`;
};

// File: src/App.tsx - Update all StatusIndicator usage
<StatusIndicator
  type={getStatusColor(instance.state)}
  aria-label={getStatusLabel(instance.state, 'instance')}
>
  {instance.state}
</StatusIndicator>

// For volumes
<StatusIndicator
  type={getStatusColor(volume.state)}
  aria-label={getStatusLabel(volume.state, 'volume')}
>
  {volume.state}
</StatusIndicator>
```

**Locations to Update** (grep for StatusIndicator):
- Instance list/table
- Volume lists (EFS + EBS)
- Project status
- User status
- AMI status
- Build status

**Testing**:
- [ ] Use screen reader on all status indicators
- [ ] Verify meaningful announcement (not just color)
- [ ] Test with VoiceOver/NVDA

**Effort**: 1-2 hours
**Priority**: A11Y-P0 (WCAG Level A violation)

---

#### A11Y-P0-4: Improve Error Identification

**WCAG**: 3.3.1 Error Identification (Level A)
**Issue**: Generic error messages, not associated with fields
**Impact**: Users don't know what's wrong or how to fix

**Implementation**:
```typescript
// File: src/components/LaunchInstanceModal.tsx
const [errors, setErrors] = useState<{
  instanceName?: string;
  template?: string;
  region?: string;
}>({});

const validateForm = () => {
  const newErrors: typeof errors = {};

  // Specific validation with helpful messages
  if (!instanceName) {
    newErrors.instanceName = 'Instance name is required';
  } else if (instanceName.length < 3) {
    newErrors.instanceName = 'Instance name must be at least 3 characters';
  } else if (!/^[a-z0-9-]+$/.test(instanceName)) {
    newErrors.instanceName = 'Use only lowercase letters, numbers, and hyphens. Example: my-research-01';
  } else if (instanceName.startsWith('-') || instanceName.endsWith('-')) {
    newErrors.instanceName = 'Instance name cannot start or end with a hyphen';
  }

  if (!selectedTemplate) {
    newErrors.template = 'Please select a template to continue';
  }

  setErrors(newErrors);
  return Object.keys(newErrors).length === 0;
};

return (
  <Modal header="Launch Instance" visible={visible} onDismiss={onDismiss}>
    <Form
      actions={
        <SpaceBetween direction="horizontal" size="xs">
          <Button variant="link" onClick={onDismiss}>Cancel</Button>
          <Button
            variant="primary"
            onClick={handleLaunch}
            disabled={!canLaunch}
          >
            Launch
          </Button>
        </SpaceBetween>
      }
      errorText={Object.keys(errors).length > 0 ? 'Please fix the errors below' : undefined}
    >
      <SpaceBetween size="l">
        <FormField
          label="Instance Name"
          description="Choose a unique name for your instance"
          constraintText="3-63 characters. Lowercase letters, numbers, and hyphens only."
          errorText={errors.instanceName}
          i18nStrings={{ errorIconAriaLabel: 'Error' }}
        >
          <Input
            value={instanceName}
            onChange={({ detail }) => {
              setInstanceName(detail.value);
              // Clear error when user starts typing
              if (errors.instanceName) {
                setErrors(prev => ({ ...prev, instanceName: undefined }));
              }
            }}
            placeholder="my-research-instance"
            invalid={!!errors.instanceName}
            ariaRequired={true}
          />
        </FormField>

        {/* More fields with similar error handling */}
      </SpaceBetween>
    </Form>
  </Modal>
);
```

**Apply Pattern To**:
- Launch instance modal
- Create volume forms
- Create project form
- Create user form
- Settings forms
- All other forms

**Testing**:
- [ ] Trigger each validation error
- [ ] Verify error associated with field
- [ ] Test with screen reader
- [ ] Verify error clears on fix

**Effort**: 3 hours
**Priority**: A11Y-P0 (WCAG Level A violation)

---

#### A11Y-P0-5: Audit Form Labels

**WCAG**: 3.3.2 Labels or Instructions (Level A), 1.3.1 Info and Relationships (Level A)
**Issue**: Some inputs may lack proper labels
**Impact**: Screen readers can't identify form fields

**Audit Checklist**:
```typescript
// Search for all Input, Select, Textarea, etc. and verify:

// ✅ CORRECT - Has FormField wrapper with label
<FormField label="Instance Name">
  <Input value={name} onChange={...} />
</FormField>

// ✅ CORRECT - Has aria-label for icon-only inputs
<Input
  type="search"
  placeholder="Search..."
  aria-label="Search templates"
/>

// ❌ INCORRECT - No label association
<Input value={name} onChange={...} />

// ❌ INCORRECT - Placeholder used as label
<Input placeholder="Instance Name" />
```

**Audit Process**:
1. Search codebase for `<Input`, `<Select>`, `<Textarea>`
2. Verify each has either:
   - FormField wrapper with label prop
   - aria-label attribute
   - aria-labelledby attribute
3. Document any missing labels
4. Add labels to all unlabeled inputs

**Testing**:
- [ ] Use screen reader on all forms
- [ ] Verify every field announced correctly
- [ ] Test with VoiceOver forms list (⌘ U)

**Effort**: 2 hours
**Priority**: A11Y-P0 (WCAG Level A violation)

---

#### A11Y-P0-6: Keyboard Trap Testing

**WCAG**: 2.1.2 No Keyboard Trap (Level A)
**Issue**: Must verify no keyboard traps in modals/dropdowns
**Impact**: Critical accessibility violation if traps exist

**Testing Protocol**:
```markdown
## Modal Testing
- [ ] Open launch instance modal → Tab through → Press ESC → Verify closes
- [ ] Open delete confirmation → Tab through → Press ESC → Verify closes
- [ ] Open settings modal → Tab through → Press ESC → Verify closes
- [ ] Verify focus returns to trigger button after close

## Dropdown Testing
- [ ] Open instance actions dropdown → Tab/Arrow keys → Press ESC → Verify closes
- [ ] Open volume actions dropdown → Tab/Arrow keys → Press ESC → Verify closes
- [ ] Verify focus returns to dropdown button

## Form Testing
- [ ] Tab through entire launch form → Verify can reach all fields
- [ ] Tab from last field → Verify reaches buttons
- [ ] Shift+Tab from first field → Verify goes to modal close button

## Complete Application Testing
- [ ] Tab through entire app from start to finish
- [ ] Verify never stuck/trapped
- [ ] Document any traps found
```

**If Traps Found**:
```typescript
// Ensure all modals have onDismiss
<Modal
  visible={visible}
  onDismiss={onDismiss}  // Required!
  // Cloudscape handles ESC automatically
>

// Ensure focus management correct
useEffect(() => {
  if (visible) {
    // Store focus to return to
    previousFocus.current = document.activeElement;
  } else {
    // Return focus when closed
    previousFocus.current?.focus();
  }
}, [visible]);
```

**Effort**: 2 hours
**Priority**: A11Y-P0 (WCAG Level A requirement)

---

#### UX-P0-3: Minimum Viable Onboarding

**Issue**: No first-run experience for new users
**Impact**: Graduate students confused, high abandonment risk

**Implementation** (Simplified version for P0):
```typescript
// File: src/components/OnboardingModal.tsx
export const OnboardingModal: React.FC = () => {
  const [visible, setVisible] = useState(() => {
    return !localStorage.getItem('cws_onboarding_completed');
  });
  const [step, setStep] = useState(1);

  const handleComplete = () => {
    localStorage.setItem('cws_onboarding_completed', 'true');
    setVisible(false);
  };

  return (
    <Modal
      visible={visible}
      onDismiss={handleComplete}
      header="Welcome to CloudWorkstation!"
      size="large"
      footer={
        <Box float="right">
          <SpaceBetween direction="horizontal" size="xs">
            <Button variant="link" onClick={handleComplete}>
              Skip Tutorial
            </Button>
            {step < 3 && (
              <Button onClick={() => setStep(step + 1)}>
                Next
              </Button>
            )}
            {step === 3 && (
              <Button variant="primary" onClick={handleComplete}>
                Get Started
              </Button>
            )}
          </SpaceBetween>
        </Box>
      }
    >
      {step === 1 && (
        <SpaceBetween size="l">
          <Box variant="h2">Launch Research Environments in Seconds</Box>
          <Box>
            CloudWorkstation provides pre-configured cloud computing environments
            for data analysis, machine learning, and research computing.
          </Box>
          <ColumnLayout columns={3}>
            <Box>
              <Icon name="bolt" size="large" />
              <Box variant="strong">Fast Setup</Box>
              <Box variant="small">Launch in 30 seconds</Box>
            </Box>
            <Box>
              <Icon name="settings" size="large" />
              <Box variant="strong">Pre-Configured</Box>
              <Box variant="small">27 research templates</Box>
            </Box>
            <Box>
              <Icon name="money" size="large" />
              <Box variant="strong">Cost Optimized</Box>
              <Box variant="small">Pay only when running</Box>
            </Box>
          </ColumnLayout>
        </SpaceBetween>
      )}

      {step === 2 && (
        <SpaceBetween size="l">
          <Box variant="h2">Choose a Template</Box>
          <Box>
            Templates are pre-configured environments with software, packages,
            and tools installed for specific research tasks.
          </Box>
          <Container>
            <SpaceBetween size="s">
              <Box variant="strong">Popular Templates:</Box>
              <Box>• Python ML: Jupyter, TensorFlow, PyTorch</Box>
              <Box>• R Research: RStudio, tidyverse, ggplot2</Box>
              <Box>• Data Science: Python + R + Julia</Box>
            </SpaceBetween>
          </Container>
        </SpaceBetween>
      )}

      {step === 3 && (
        <SpaceBetween size="l">
          <Box variant="h2">Launch Your First Instance</Box>
          <Box>
            It's easy:
          </Box>
          <ColumnLayout columns={1} variant="text-grid">
            <Box>
              <Badge color="blue">1</Badge> Browse Templates → Choose one
            </Box>
            <Box>
              <Badge color="blue">2</Badge> Enter instance name → Click Launch
            </Box>
            <Box>
              <Badge color="blue">3</Badge> Wait 30-60 seconds → Connect via SSH
            </Box>
          </ColumnLayout>
          <Alert type="info">
            💡 Tip: Start with "test-ssh" template for a simple Ubuntu environment
          </Alert>
        </SpaceBetween>
      )}
    </Modal>
  );
};

// Add to App.tsx
function App() {
  return (
    <>
      <OnboardingModal />
      {/* Rest of app */}
    </>
  );
}
```

**Testing**:
- [ ] Clear localStorage → Reload → See wizard
- [ ] Complete wizard → Verify localStorage set
- [ ] Reload → Verify wizard doesn't show
- [ ] Test skip button

**Effort**: 4-5 hours
**Priority**: UX-P0 (Critical for new user experience)

---

### Sprint 0 Summary

**Total Items**: 8 (3 UX + 5 Accessibility)
**Total Effort**: 21 hours (~3 days)
**Deliverables**:
- ✅ Safe delete confirmations
- ✅ Clean console (no error spam)
- ✅ WCAG Level A skip navigation
- ✅ WCAG Level A landmarks
- ✅ WCAG Level A status labels
- ✅ WCAG Level A error identification
- ✅ WCAG Level A form labels verified
- ✅ No keyboard traps confirmed
- ✅ Basic onboarding for new users

**Production Readiness After Sprint 0**: ✅ **APPROVED**
- Safety critical issues resolved
- WCAG Level A compliance achieved (~90%)
- Minimum viable onboarding in place

---

## SPRINT 1: High Priority UX + Accessibility (P1)

**Duration**: 5 days (34 hours)
**Goal**: WCAG Level AA compliance + major UX improvements
**Success Criteria**: 95%+ WCAG AA, major usability enhancements

### Sprint 1 Items

#### UX-P1-1: Cost Visibility Dashboard (3 hours)

**Implementation**:
```typescript
// Add to dashboard
<Container header={<Header variant="h2">Cost Overview</Header>}>
  <ColumnLayout columns={3} variant="text-grid">
    <Box>
      <Box variant="awsui-key-label">Today's Cost</Box>
      <Box fontSize="display-l" fontWeight="bold">
        ${calculateDailyCost().toFixed(2)}
      </Box>
    </Box>
    <Box>
      <Box variant="awsui-key-label">Monthly Estimate</Box>
      <Box fontSize="display-l" fontWeight="bold">
        ${(calculateDailyCost() * 30).toFixed(2)}
      </Box>
    </Box>
    <Box>
      <Box variant="awsui-key-label">Running Instances</Box>
      <Box fontSize="display-l" fontWeight="bold">
        {state.instances.filter(i => i.state === 'running').length}
      </Box>
    </Box>
  </ColumnLayout>
</Container>

// Add cost column to instance table
{
  id: 'cost',
  header: 'Est. Daily Cost',
  cell: (item: Instance) => `$${calculateInstanceCost(item).toFixed(2)}`,
  sortingField: 'cost'
}
```

**Testing**: Verify cost displays correctly with running instances

---

#### UX-P1-2: Template Search & Filters (4 hours)

**Implementation**:
```typescript
const [searchQuery, setSearchQuery] = useState('');
const [selectedCategory, setSelectedCategory] = useState<string>('');

const filteredTemplates = useMemo(() => {
  let filtered = templateList;

  if (searchQuery) {
    const query = searchQuery.toLowerCase();
    filtered = filtered.filter(t =>
      t.Name?.toLowerCase().includes(query) ||
      t.Description?.toLowerCase().includes(query)
    );
  }

  if (selectedCategory) {
    filtered = filtered.filter(t => t.category === selectedCategory);
  }

  return filtered;
}, [templateList, searchQuery, selectedCategory]);

return (
  <>
    <Header
      variant="h1"
      counter={`(${filteredTemplates.length} templates)`}
      actions={
        <SpaceBetween direction="horizontal" size="s">
          <Input
            type="search"
            placeholder="Search templates..."
            value={searchQuery}
            onChange={({ detail }) => setSearchQuery(detail.value)}
            aria-label="Search templates"
          />
          <Select
            selectedOption={{ label: selectedCategory || 'All Categories', value: selectedCategory }}
            onChange={({ detail }) => setSelectedCategory(detail.selectedOption.value || '')}
            options={[
              { label: 'All Categories', value: '' },
              { label: 'Machine Learning', value: 'ml' },
              { label: 'Data Science', value: 'data-science' },
              { label: 'Bioinformatics', value: 'bio' },
              { label: 'General Purpose', value: 'general' }
            ]}
          />
        </SpaceBetween>
      }
    >
      Research Templates
    </Header>

    <Cards
      items={filteredTemplates}
      cardDefinition={...}
      loading={state.loading}
      empty={
        <Box textAlign="center" padding="xxl">
          <Box variant="strong">No templates found</Box>
          <Box variant="p" color="text-body-secondary">
            Try a different search term or category
          </Box>
        </Box>
      }
    />
  </>
);
```

**Testing**: Search for templates, filter by category, verify results

---

#### UX-P1-3: Keyboard Shortcuts Framework (4 hours)

See previous documentation for full implementation of keyboard shortcuts system.

---

#### UX-P1-4: Enhanced Loading States (3 hours)

See previous documentation for loading state improvements with progress bars.

---

#### UX-P1-5: Contextual Help System (6 hours)

See previous documentation for help panel implementation.

---

#### A11Y-P1-1: Color Contrast Audit (3 hours)

**Audit Process**:
1. Install color contrast checker
2. Test all text/background combinations
3. Document violations
4. Replace colors below 4.5:1 ratio
5. Create approved color palette

**Testing**: Verify all text meets 4.5:1 ratio (or 3:1 for large text)

---

#### A11Y-P1-2: Focus Indicators (2 hours)

Test focus visibility throughout app and add custom styles if needed.

---

#### A11Y-P1-3: Touch Target Audit (3 hours)

Verify all interactive elements meet 24x24px minimum per WCAG 2.2.

---

#### A11Y-P1-4: Auto-Refresh Controls (1 hour)

Add toggle to disable auto-refresh for screen reader users.

---

#### A11Y-P1-5: Error Suggestions (2 hours)

Enhance all error messages with specific correction suggestions.

---

#### A11Y-P1-6: Consistent Help (4 hours)

Implement consistent help mechanism across all views per WCAG 3.2.6.

---

### Sprint 1 Summary

**Total Items**: 11 (5 UX + 6 Accessibility)
**Total Effort**: 34 hours (~5 days)
**Deliverables**:
- Cost visibility throughout app
- Template search and filters
- Keyboard shortcuts (basic)
- Enhanced loading states
- Contextual help system
- WCAG AA color contrast
- WCAG AA focus indicators
- WCAG 2.2 touch targets
- Auto-refresh controls
- Error suggestion improvements
- Consistent help mechanism

**Compliance After Sprint 1**:
- WCAG Level AA: 95%+ compliant
- UX Quality: Significant improvement in efficiency

---

## SPRINT 2: Important Enhancements (P2)

**Duration**: 7 days (30 hours)
**Goal**: Power user features + enhanced accessibility

### Sprint 2 Items (Summary)

**UX Items** (19 hours):
- Navigation grouping (2 hours)
- Project setup wizard (8 hours)
- Storage wizard (6 hours)
- Error recovery/undo (4 hours)
- Hibernation UX (4 hours)
- Troubleshooting tools (6 hours) [Moved from 19 to account for total]

**Accessibility Items** (11 hours):
- Focus management on view changes (2 hours)
- Keyboard testing comprehensive (2 hours)
- Screen reader testing (3 hours)
- Page title dynamics (0.5 hours)
- ARIA validation (3 hours)
- Text resize testing (0.5 hours)

Total: ~30 hours over 7 days

---

## Implementation Tracking

### Todo Checklist Format

```markdown
## Sprint 0: Pre-Launch Critical (P0) - 21 hours

### Week 1, Day 1-3

**UX-P0-1: Delete Confirmations** [2-3 hours]
- [ ] Create ConfirmDeleteModal component
- [ ] Add to instance delete (with name confirmation)
- [ ] Add to EFS volume delete
- [ ] Add to EBS volume delete
- [ ] Add to project delete
- [ ] Add to user delete
- [ ] Test all delete flows
- [ ] Test keyboard navigation (ESC to cancel)
- [ ] Verify focus returns after dismiss

**UX-P0-2: Fix API Error Logging** [1 hour]
- [ ] Update getRightsizingStats() error handling
- [ ] Update getAMIBuilds() error handling
- [ ] Audit all API methods for 400/404 handling
- [ ] Test with daemon missing endpoints
- [ ] Verify console is clean

**A11Y-P0-1: Skip Navigation** [1 hour]
- [ ] Create SkipLink component
- [ ] Add to App.tsx before AppLayout
- [ ] Style with sr-only pattern
- [ ] Show on focus
- [ ] Link to #main-content
- [ ] Add main tag with id
- [ ] Test keyboard navigation
- [ ] Test with screen reader

**A11Y-P0-2: Landmark Roles** [1 hour]
- [ ] Wrap SideNavigation in <nav> with aria-label
- [ ] Wrap content in <main> with id and role
- [ ] Add tabIndex={-1} to main
- [ ] Test with screen reader landmarks
- [ ] Verify rotor navigation (VoiceOver)

**A11Y-P0-3: Status Labels** [1-2 hours]
- [ ] Create getStatusLabel() utility
- [ ] Find all StatusIndicator components (grep)
- [ ] Add aria-label to instance statuses
- [ ] Add aria-label to volume statuses
- [ ] Add aria-label to project statuses
- [ ] Add aria-label to user statuses
- [ ] Add aria-label to AMI/build statuses
- [ ] Test with screen reader

**A11Y-P0-4: Error Identification** [3 hours]
- [ ] Audit launch instance modal errors
- [ ] Add specific validation messages
- [ ] Associate errors with FormField errorText
- [ ] Add real-time validation
- [ ] Apply pattern to volume creation forms
- [ ] Apply pattern to project creation
- [ ] Apply pattern to user creation
- [ ] Apply pattern to settings forms
- [ ] Test with screen reader

**A11Y-P0-5: Form Labels Audit** [2 hours]
- [ ] Search codebase for <Input> components
- [ ] Verify each has FormField or aria-label
- [ ] Document unlabeled inputs
- [ ] Add labels to all unlabeled inputs
- [ ] Search for <Select> components
- [ ] Search for <Textarea> components
- [ ] Test all forms with screen reader
- [ ] Verify VoiceOver forms list complete

**A11Y-P0-6: Keyboard Trap Testing** [2 hours]
- [ ] Test launch modal (Tab + ESC)
- [ ] Test delete confirmation (Tab + ESC)
- [ ] Test settings modal (Tab + ESC)
- [ ] Test all ButtonDropdowns (Tab/Arrow + ESC)
- [ ] Test complete app tab-through
- [ ] Document any traps found
- [ ] Fix any keyboard traps
- [ ] Verify focus returns correctly

**UX-P0-3: Minimum Onboarding** [4-5 hours]
- [ ] Create OnboardingModal component
- [ ] Add step 1: Welcome + overview
- [ ] Add step 2: Templates explanation
- [ ] Add step 3: Launch instructions
- [ ] Add Skip button
- [ ] Add localStorage completion tracking
- [ ] Integrate into App.tsx
- [ ] Test first-run experience
- [ ] Test localStorage persistence
- [ ] Add "Show Tutorial Again" to settings

---

## Sprint 1: High Priority (P1) - 34 hours

### Week 2, Days 1-5

**UX-P1-1: Cost Visibility** [3 hours]
- [ ] Add cost calculation utilities
- [ ] Add Cost Overview to dashboard
- [ ] Add daily cost counter
- [ ] Add monthly estimate
- [ ] Add cost column to instance table
- [ ] Test cost calculations
- [ ] Verify updates when instances change

**UX-P1-2: Template Search** [4 hours]
- [ ] Add search state management
- [ ] Add search Input to header
- [ ] Implement filter logic
- [ ] Add category dropdown
- [ ] Add sort options (cost, name)
- [ ] Persist search/filter preferences
- [ ] Add empty state for no results
- [ ] Test search functionality

**UX-P1-3: Keyboard Shortcuts** [4 hours]
- [ ] Add keyboard event handler
- [ ] Implement ⌘K for search
- [ ] Implement ⌘L for launch
- [ ] Implement ⌘1-9 for views
- [ ] Create KeyboardShortcutsModal
- [ ] Add ? to show shortcuts
- [ ] Document all shortcuts
- [ ] Test on Mac and Windows

**UX-P1-4: Loading States** [3 hours]
- [ ] Add loadingMessage to state
- [ ] Update all loading calls with messages
- [ ] Create ProgressBar component
- [ ] Add progress for instance launch
- [ ] Add step indicators
- [ ] Test loading UX

**UX-P1-5: Help System** [6 hours]
- [ ] Implement Cloudscape HelpPanel
- [ ] Create contextual help content
- [ ] Add help for each view
- [ ] Add glossary of terms
- [ ] Add tutorial links
- [ ] Make help available everywhere
- [ ] Test help panel accessibility

**A11Y-P1-1: Color Contrast** [3 hours]
- [ ] Install contrast checker tool
- [ ] Audit all text colors
- [ ] Audit all Badge colors
- [ ] Audit custom Box colors
- [ ] Document violations
- [ ] Replace colors below 4.5:1
- [ ] Create approved palette
- [ ] Re-test all colors

**A11Y-P1-2: Focus Indicators** [2 hours]
- [ ] Tab through entire app
- [ ] Check focus on all buttons
- [ ] Check focus on all links
- [ ] Check focus on all inputs
- [ ] Check focus contrast (3:1)
- [ ] Add custom focus styles if needed
- [ ] Test keyboard-only navigation

**A11Y-P1-3: Touch Targets** [3 hours]
- [ ] Audit all buttons
- [ ] Audit all links
- [ ] Measure icon buttons
- [ ] Measure close buttons
- [ ] Fix targets below 24x24px
- [ ] Add padding to small targets
- [ ] Test on touch device
- [ ] Document minimum sizes

**A11Y-P1-4: Auto-Refresh** [1 hour]
- [ ] Add autoRefresh state
- [ ] Add Toggle control
- [ ] Persist preference
- [ ] Update useEffect
- [ ] Test disable/enable

**A11Y-P1-5: Error Suggestions** [2 hours]
- [ ] Review all error messages
- [ ] Add correction suggestions
- [ ] Add example valid inputs
- [ ] Add constraintText to fields
- [ ] Test error recovery flows

**A11Y-P1-6: Consistent Help** [4 hours]
- [ ] Already covered in UX-P1-5
- [ ] Verify help in same location
- [ ] Test help accessibility
- [ ] Verify keyboard access

---

## Sprint 2: Important (P2) - 30 hours

### Weeks 3-4

[Detailed todo items for each P2 task - similar format]

---

## Backlog: Future (P3) - ~25 hours

[P3 items for future sprints]
```

---

## Testing Strategy

### After Each Sprint

**Sprint 0 Testing** (Critical):
- [ ] Manual keyboard navigation (entire app)
- [ ] Screen reader testing (VoiceOver)
- [ ] Delete confirmation flows
- [ ] Console error check
- [ ] First-run onboarding

**Sprint 1 Testing**:
- [ ] Color contrast verification
- [ ] Touch target measurements
- [ ] Keyboard shortcuts
- [ ] Search functionality
- [ ] Cost calculations

**Sprint 2 Testing**:
- [ ] Wizard flows
- [ ] Bulk actions
- [ ] Error recovery
- [ ] Screen reader comprehensive

---

## Success Metrics

### Compliance Metrics

| Metric | Pre-Sprint | After Sprint 0 | After Sprint 1 | After Sprint 2 | Target |
|--------|-----------|----------------|----------------|----------------|---------|
| WCAG 2.2 Level A | 60-70% | 90%+ | 95%+ | 98%+ | 100% |
| WCAG 2.2 Level AA | 50-60% | 70%+ | 95%+ | 98%+ | 95%+ |
| UX Quality Score | 4/5 | 4.2/5 | 4.7/5 | 5/5 | 5/5 |

### User Impact Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Time to first instance launch | <2 min | User testing |
| Task completion rate | >90% | User testing |
| Keyboard navigation efficiency | <30 sec to any view | Manual testing |
| Screen reader satisfaction | >4/5 | User testing |
| Error recovery success | >95% | User testing |

---

## Risk Assessment

### High Risk Items

1. **Keyboard Trap Testing** (A11Y-P0-6)
   - Risk: May find critical traps
   - Mitigation: Test early, fix immediately

2. **Form Labels Audit** (A11Y-P0-5)
   - Risk: Many unlabeled inputs
   - Mitigation: Systematic audit with grep

3. **Color Contrast** (A11Y-P1-1)
   - Risk: Many color changes needed
   - Mitigation: Create palette first

### Medium Risk Items

1. **Touch Target Audit** (A11Y-P1-3)
   - Risk: Many small buttons
   - Mitigation: Padding solution

2. **Project Wizard** (UX-P2-1)
   - Risk: Complex workflow
   - Mitigation: Start with simple version

---

## Resource Requirements

### Development Tools Needed

**Accessibility Testing**:
- Color Contrast Analyzer (free)
- axe DevTools (free browser extension)
- WAVE browser extension (free)
- Screen reader (VoiceOver on Mac, NVDA on Windows)

**Development**:
- React DevTools
- TypeScript
- Cloudscape Design System docs

**Testing**:
- Manual testing checklist
- User testing participants (3-5 researchers)

---

## Timeline Summary

```
Week 1 (Sprint 0): 3 days
├─ Day 1: Delete confirmations, API fixes, skip link
├─ Day 2: Landmarks, status labels, error ID
└─ Day 3: Form audit, keyboard traps, onboarding

Week 2 (Sprint 1): 5 days
├─ Day 1: Cost visibility, template search
├─ Day 2: Keyboard shortcuts, loading states
├─ Day 3: Help system
├─ Day 4: Color contrast, focus indicators
└─ Day 5: Touch targets, auto-refresh, errors

Weeks 3-4 (Sprint 2): 7 days
├─ Days 1-2: Navigation, project wizard
├─ Days 3-4: Storage wizard, bulk actions
├─ Days 5-6: Error recovery, hibernation, troubleshooting
└─ Day 7: Focus management, SR testing
```

**Total Duration**: 3 weeks (15 working days)
**Total Effort**: 85-95 hours

---

## Conclusion

This comprehensive remediation plan addresses all identified UX and accessibility issues in a structured, prioritized manner. Completing Sprint 0 (21 hours, 3 days) makes the GUI production-ready with critical safety features and WCAG Level A compliance. Sprints 1 and 2 add major usability improvements and achieve WCAG Level AA compliance.

**Recommendation**: **Begin Sprint 0 immediately**. Complete all P0 items before production launch. Schedule Sprints 1 and 2 for first two weeks post-launch.

---

**Comprehensive Remediation Plan Complete**: October 13, 2025
**Total Items**: 35 (18 UX + 17 A11Y)
**Total Effort**: 85-95 hours
**Production Ready After**: Sprint 0 (21 hours / 3 days)
**Full Compliance**: Sprint 2 completion (3 weeks total)

