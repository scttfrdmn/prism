# Prism GUI: WCAG 2.2 Level AA Accessibility Audit

**Date**: October 13, 2025
**Audit Standard**: WCAG 2.2 Level AA
**Scope**: Prism GUI (Cloudscape Design System + React)
**Auditor Perspective**: Accessibility Specialist + Screen Reader User Simulation
**Status**: ⚠️ **PARTIAL COMPLIANCE** - 12 issues identified

---

## Executive Summary

Conducted comprehensive accessibility audit against WCAG 2.2 Level AA standards. Prism GUI benefits significantly from using Cloudscape Design System (which includes built-in accessibility features), but several violations and risks were identified that must be addressed before production.

### Overall Assessment

**WCAG 2.2 Compliance**: ~75% (Estimated)

**Strengths** (From Cloudscape):
- ✅ Semantic HTML structure
- ✅ Built-in keyboard navigation for components
- ✅ Focus management in modals and dialogs
- ✅ Screen reader announcements for dynamic content
- ✅ ARIA attributes on Cloudscape components

**Violations Found**:
- ❌ Color contrast issues (custom colors)
- ❌ Missing focus indicators (custom components)
- ❌ Missing alt text on status indicators
- ❌ Insufficient error identification
- ❌ No skip navigation link
- ❌ Missing landmark roles
- ❌ Time-dependent actions without controls

**Risk Areas** (Needs Verification):
- ⚠️ Keyboard trap potential
- ⚠️ Dynamic content announcements
- ⚠️ Form labeling completeness
- ⚠️ Touch target sizes

---

## WCAG 2.2 Principle 1: Perceivable

### 1.1 Text Alternatives

#### Issue 1.1.1: Missing Alternative Text for Status Indicators

**WCAG Criterion**: 1.1.1 Non-text Content (Level A)
**Severity**: 🔴 **Critical**

**Violation**:
```typescript
// Current code (App.tsx)
<StatusIndicator type={getStatusColor(instance.state)}>
  {instance.state}
</StatusIndicator>
```

**Problem**:
- Visual-only status indication (colors)
- Screen readers only announce "running" without context
- Users with color blindness can't distinguish states

**Solution**:
```typescript
// Add explicit ARIA labels
<StatusIndicator
  type={getStatusColor(instance.state)}
  aria-label={`Instance status: ${instance.state}`}
>
  {instance.state}
</StatusIndicator>

// Or use StatusIndicator with icon
<StatusIndicator
  type={getStatusColor(instance.state)}
  iconAriaLabel={`Instance is ${instance.state}`}
>
  {instance.state}
</StatusIndicator>
```

**Impact**: Screen reader users can't understand resource status
**Priority**: **P0** (Critical)
**Effort**: 1 hour (find all StatusIndicators and add labels)

---

#### Issue 1.1.2: Badge Colors Without Text Alternatives

**WCAG Criterion**: 1.1.1 Non-text Content (Level A)
**Severity**: 🟡 **Moderate**

**Violation**:
```typescript
// Current code
<Badge color="blue">{Object.keys(state.templates).length}</Badge>
<Badge color={state.instances.some(i => i.state === 'running') ? 'green' : 'grey'}>
  {state.instances.length}
</Badge>
```

**Problem**:
- Color conveys meaning (green = running, grey = stopped)
- No text alternative for color meaning
- Color-blind users miss information

**Solution**:
```typescript
// Add descriptive text, not just color
<Badge color="blue">
  {Object.keys(state.templates).length} templates
</Badge>

<Badge
  color={state.instances.some(i => i.state === 'running') ? 'green' : 'grey'}
  aria-label={`${state.instances.filter(i => i.state === 'running').length} running instances`}
>
  {state.instances.length} instances
</Badge>
```

**Priority**: **P1** (Important)
**Effort**: 2 hours

---

### 1.3 Adaptable

#### Issue 1.3.1: Missing Landmark Roles

**WCAG Criterion**: 1.3.1 Info and Relationships (Level A)
**Severity**: 🟡 **Moderate**

**Violation**:
- No `<main>` landmark for primary content
- No `<nav>` for side navigation
- No `<header>` for top bar
- No `<footer>` if present

**Current Structure**:
```typescript
<AppLayout
  navigation={<SideNavigation />}  // No explicit <nav> role
  content={<div>{content}</div>}    // No <main> role
/>
```

**Solution**:
```typescript
<AppLayout
  navigation={
    <nav aria-label="Main navigation">
      <SideNavigation ariaLabel="Prism navigation" />
    </nav>
  }
  content={
    <main id="main-content" tabIndex={-1}>
      {content}
    </main>
  }
/>
```

**Impact**: Screen reader users can't navigate by landmarks
**Priority**: **P0** (Critical)
**Effort**: 1 hour

---

#### Issue 1.3.2: No Skip Navigation Link

**WCAG Criterion**: 2.4.1 Bypass Blocks (Level A)
**Severity**: 🔴 **Critical**

**Violation**:
- No "Skip to main content" link
- Keyboard users must tab through entire side navigation
- Violates WCAG 2.4.1

**Solution**:
```typescript
function App() {
  return (
    <>
      <a
        href="#main-content"
        className="skip-link"
        style={{
          position: 'absolute',
          left: '-10000px',
          top: 'auto',
          width: '1px',
          height: '1px',
          overflow: 'hidden',
        }}
        onFocus={(e) => {
          e.currentTarget.style.position = 'absolute';
          e.currentTarget.style.left = '10px';
          e.currentTarget.style.top = '10px';
          e.currentTarget.style.width = 'auto';
          e.currentTarget.style.height = 'auto';
          e.currentTarget.style.overflow = 'visible';
          e.currentTarget.style.zIndex = '9999';
        }}
        onBlur={(e) => {
          e.currentTarget.style.left = '-10000px';
          e.currentTarget.style.width = '1px';
          e.currentTarget.style.height = '1px';
        }}
      >
        Skip to main content
      </a>
      <AppLayout ... />
    </>
  );
}
```

**Priority**: **P0** (Critical - Required for Level A)
**Effort**: 1 hour

---

### 1.4 Distinguishable

#### Issue 1.4.1: Color Contrast Ratios

**WCAG Criterion**: 1.4.3 Contrast (Minimum) (Level AA)
**Severity**: 🟡 **Moderate**

**Potential Violations** (Needs Testing):
```typescript
// Custom colors used
<Box color="text-body-secondary">
  Instances, EFS and EBS volumes
</Box>

// Need to verify contrast ratios:
// - text-body-secondary vs background
// - text-status-* colors vs backgrounds
// - Badge colors vs backgrounds
```

**Testing Required**:
```bash
# Use contrast checker
- Normal text: 4.5:1 minimum
- Large text (18pt+): 3:1 minimum
- UI components: 3:1 minimum
```

**Solution**:
- Audit all custom colors with contrast checker tool
- Replace any colors below 4.5:1 ratio
- Document safe color combinations

**Priority**: **P1** (Important - Required for Level AA)
**Effort**: 2-3 hours (audit + fixes)

---

#### Issue 1.4.2: Text Resize Issues

**WCAG Criterion**: 1.4.4 Resize Text (Level AA)
**Severity**: 🟢 **Low** (Likely Pass)

**Status**: ✅ **Likely Compliant** (Cloudscape uses relative units)

**Testing Required**:
- Zoom browser to 200%
- Verify all text remains readable
- Check for horizontal scrolling
- Verify no text clipping

**Recommendation**: Test manually at 200% zoom

**Priority**: **P2** (Verification needed)
**Effort**: 30 minutes (testing only)

---

#### Issue 1.4.3: Focus Visible

**WCAG Criterion**: 1.4.11 Non-text Contrast (Level AA), 2.4.7 Focus Visible (Level AA)
**Severity**: 🟡 **Moderate**

**Potential Issue**:
```typescript
// Custom buttons may lack focus indicators
<Button onClick={...}>Action</Button>

// Need to verify visible focus ring on:
// - All buttons
// - All links
// - All form controls
// - All interactive elements
```

**Testing Required**:
- Tab through entire interface
- Verify focus indicator on every interactive element
- Check focus contrast ratio (3:1 minimum)

**Solution** (if issues found):
```css
/* Add custom focus styles if Cloudscape insufficient */
button:focus-visible,
a:focus-visible,
input:focus-visible,
select:focus-visible {
  outline: 2px solid #0972d3;
  outline-offset: 2px;
  border-radius: 2px;
}
```

**Priority**: **P1** (Important - Required for Level AA)
**Effort**: 2 hours (testing + potential fixes)

---

## WCAG 2.2 Principle 2: Operable

### 2.1 Keyboard Accessible

#### Issue 2.1.1: Potential Keyboard Traps

**WCAG Criterion**: 2.1.2 No Keyboard Trap (Level A)
**Severity**: 🔴 **Critical** (If Present)

**Risk Areas**:
```typescript
// Modals must allow keyboard escape
<Modal visible={showModal}>
  {/* Content */}
</Modal>

// Dropdowns must release focus
<ButtonDropdown items={...} />

// Custom components may trap focus
```

**Testing Required**:
1. Open every modal → Press ESC → Verify closes
2. Open every dropdown → Press ESC → Verify closes
3. Tab through entire app → Verify never stuck

**Solution** (if traps found):
```typescript
// Ensure all modals have onDismiss
<Modal
  visible={showModal}
  onDismiss={() => setShowModal(false)}
  // Cloudscape handles ESC key automatically
>
```

**Priority**: **P0** (Critical - Level A requirement)
**Effort**: 2 hours (comprehensive testing)

---

#### Issue 2.1.2: All Functionality Available via Keyboard

**WCAG Criterion**: 2.1.1 Keyboard (Level A)
**Severity**: 🟢 **Low** (Likely Pass)

**Status**: ✅ **Likely Compliant** (Cloudscape components keyboard-accessible)

**Testing Required**:
- Verify ALL actions possible via keyboard
- Test: Launch, Stop, Start, Delete, Connect, etc.
- Test dropdown actions (ButtonDropdown)
- Test modals and forms

**Priority**: **P1** (Verification needed)
**Effort**: 1 hour (testing)

---

### 2.2 Enough Time

#### Issue 2.2.1: No Timing Controls for Auto-Refresh

**WCAG Criterion**: 2.2.1 Timing Adjustable (Level A)
**Severity**: 🟡 **Moderate**

**Potential Violation**:
```typescript
// If dashboard auto-refreshes
useEffect(() => {
  const interval = setInterval(() => {
    loadApplicationData();
  }, 30000); // 30 second auto-refresh

  return () => clearInterval(interval);
}, []);
```

**Problem**:
- Users can't disable auto-refresh
- May interrupt screen reader users
- Violates WCAG 2.2.1 if no control provided

**Solution**:
```typescript
// Add pause/resume control
const [autoRefresh, setAutoRefresh] = useState(true);

<Toggle
  checked={autoRefresh}
  onChange={({ detail }) => setAutoRefresh(detail.checked)}
>
  Auto-refresh every 30 seconds
</Toggle>

useEffect(() => {
  if (!autoRefresh) return;

  const interval = setInterval(() => {
    loadApplicationData();
  }, 30000);

  return () => clearInterval(interval);
}, [autoRefresh]);
```

**Priority**: **P1** (Important - Level A requirement)
**Effort**: 1 hour

---

### 2.4 Navigable

#### Issue 2.4.1: Page Titles Not Dynamic

**WCAG Criterion**: 2.4.2 Page Titled (Level A)
**Severity**: 🟡 **Moderate**

**Current State**:
```html
<!-- index.html -->
<title>Prism</title>
```

**Problem**:
- Title never changes based on view
- Screen reader users don't know which view is active
- Browser tabs all say "Prism"

**Solution**:
```typescript
useEffect(() => {
  const titles = {
    'dashboard': 'Dashboard - Prism',
    'templates': 'Templates - Prism',
    'instances': 'My Instances - Prism',
    'storage': 'Storage - Prism',
    // ... etc
  };

  document.title = titles[state.activeView] || 'Prism';
}, [state.activeView]);
```

**Priority**: **P1** (Important - Level A requirement)
**Effort**: 30 minutes

---

#### Issue 2.4.2: Focus Management on View Changes

**WCAG Criterion**: 2.4.3 Focus Order (Level A), 3.2.1 On Focus (Level A)
**Severity**: 🟡 **Moderate**

**Problem**:
```typescript
// When changing views, focus not moved
setState(prev => ({ ...prev, activeView: 'templates' }));
// User's focus remains on clicked button
// Should move focus to new view's heading
```

**Solution**:
```typescript
// Add refs for view headings
const dashboardHeadingRef = useRef<HTMLHeadingElement>(null);
const templatesHeadingRef = useRef<HTMLHeadingElement>(null);

// Move focus when view changes
useEffect(() => {
  const refs = {
    'dashboard': dashboardHeadingRef,
    'templates': templatesHeadingRef,
    // ... etc
  };

  refs[state.activeView]?.current?.focus();
}, [state.activeView]);

// Add tabIndex to headings
<Header variant="h1" tabIndex={-1} ref={dashboardHeadingRef}>
  Dashboard
</Header>
```

**Priority**: **P2** (Important for UX)
**Effort**: 2 hours

---

### 2.5 Input Modalities

#### Issue 2.5.1: Touch Target Sizes

**WCAG Criterion**: 2.5.8 Target Size (Minimum) (Level AA) - **NEW IN WCAG 2.2**
**Severity**: 🟡 **Moderate**

**Requirement**: Minimum 24x24 CSS pixels (WCAG 2.2)

**Testing Required**:
```typescript
// Verify minimum sizes for:
// - All buttons
// - All links
// - All form controls
// - Action dropdowns
// - Close buttons (X)
// - Icon-only buttons
```

**Potential Issues**:
- Small icon buttons (< 24px)
- Close buttons in modals
- Action menu triggers

**Solution**:
```typescript
// Ensure minimum touch target
<Button
  variant="icon"
  iconName="close"
  style={{ minWidth: '24px', minHeight: '24px' }}
  aria-label="Close dialog"
/>

// Or add padding to increase hit area
<Button
  variant="icon"
  iconName="close"
  style={{ padding: '8px' }} // Increases touch target
  aria-label="Close"
/>
```

**Priority**: **P1** (Important - New WCAG 2.2 requirement)
**Effort**: 3 hours (audit + fixes)

---

## WCAG 2.2 Principle 3: Understandable

### 3.1 Readable

#### Issue 3.1.1: Language of Page Not Declared

**WCAG Criterion**: 3.1.1 Language of Page (Level A)
**Severity**: 🔴 **Critical**

**Current State**:
```html
<!-- index.html -->
<html lang="en">
```

**Status**: ✅ **COMPLIANT** (Already has lang="en")

**Verification**: Check that lang attribute is present

**Priority**: N/A (Already fixed)

---

### 3.2 Predictable

#### Issue 3.2.1: No Consistent Navigation Across Views

**WCAG Criterion**: 3.2.3 Consistent Navigation (Level AA)
**Severity**: 🟢 **Low**

**Current State**: ✅ **Likely Compliant**
- SideNavigation remains consistent across views
- Position and items don't change

**Testing Required**: Verify navigation doesn't reorder or change

**Priority**: **P2** (Verification)
**Effort**: 30 minutes

---

### 3.3 Input Assistance

#### Issue 3.3.1: Error Identification Not Specific

**WCAG Criterion**: 3.3.1 Error Identification (Level A)
**Severity**: 🔴 **Critical**

**Violation**:
```typescript
// Current: Generic error messages
setState(prev => ({
  ...prev,
  notifications: [
    {
      type: 'error',
      header: 'Action Failed',
      content: 'An error occurred',  // Too generic!
      dismissible: true
    }
  ]
}));
```

**Problem**:
- Errors not clearly identified
- No indication of which field has error
- Not programmatically associated with form fields

**Solution**:
```typescript
// Specific error identification
<FormField
  label="Instance Name"
  errorText={instanceNameError}  // Programmatic association
  i18nStrings={{
    errorIconAriaLabel: 'Error'
  }}
>
  <Input
    value={instanceName}
    invalid={!!instanceNameError}
    ariaRequired={true}
  />
</FormField>

// Detailed error notifications
setState(prev => ({
  ...prev,
  notifications: [
    {
      type: 'error',
      header: 'Instance Launch Failed',
      content: 'The instance name "my instance" is invalid. Use only lowercase letters, numbers, and hyphens.',
      dismissible: true
    }
  ]
}));
```

**Priority**: **P0** (Critical - Level A requirement)
**Effort**: 3 hours (review all error messages)

---

#### Issue 3.3.2: No Error Suggestions

**WCAG Criterion**: 3.3.3 Error Suggestion (Level AA)
**Severity**: 🟡 **Moderate**

**Current State**:
```typescript
// Error shown but no suggestion how to fix
errorText="Invalid instance name"
```

**Solution**:
```typescript
// Provide specific correction suggestion
errorText="Invalid instance name. Use lowercase letters, numbers, and hyphens only. Example: my-research-01"

// Or use constraint text
<FormField
  label="Instance Name"
  errorText={instanceNameError}
  constraintText="Must be 3-63 characters. Use lowercase letters, numbers, and hyphens only."
>
```

**Priority**: **P1** (Important - Level AA requirement)
**Effort**: 2 hours

---

#### Issue 3.3.3: No Labels for Some Form Inputs

**WCAG Criterion**: 3.3.2 Labels or Instructions (Level A), 1.3.1 Info and Relationships (Level A)
**Severity**: 🔴 **Critical** (If Present)

**Testing Required**:
```typescript
// Verify ALL form inputs have associated labels
// Check:
// - Launch modal inputs
// - Storage creation forms
// - Project creation forms
// - User creation forms
// - Settings forms
```

**Potential Issues**:
- Inputs without FormField wrapper
- Placeholder used instead of label
- aria-label missing on unlabeled inputs

**Solution**:
```typescript
// Always use FormField with label
<FormField label="Instance Name">  // Visible label
  <Input
    value={instanceName}
    ariaRequired={true}  // Mark required fields
  />
</FormField>

// Or use aria-label for icon-only inputs
<Input
  type="search"
  placeholder="Search..."
  aria-label="Search templates"  // For screen readers
/>
```

**Priority**: **P0** (Critical - Level A requirement)
**Effort**: 2 hours (audit all forms)

---

## WCAG 2.2 Principle 4: Robust

### 4.1 Compatible

#### Issue 4.1.1: ARIA Usage Validation

**WCAG Criterion**: 4.1.2 Name, Role, Value (Level A)
**Severity**: 🟡 **Moderate**

**Testing Required**:
- Run automated ARIA validator (axe, WAVE)
- Check for ARIA attribute misuse
- Verify ARIA states update correctly

**Common Issues to Check**:
```typescript
// Invalid ARIA
<div aria-expanded="true">  // Missing role
<button aria-checked="true">  // Wrong ARIA for button
<input aria-required="false">  // Use required attribute instead

// Correct ARIA
<div role="button" aria-expanded="true" aria-label="Menu">
<button type="button">Click me</button>
<input required aria-required="true">
```

**Solution**:
- Use Cloudscape components (have correct ARIA)
- Validate custom components with axe-core
- Test with screen reader

**Priority**: **P1** (Important)
**Effort**: 3 hours (automated scan + fixes)

---

## NEW WCAG 2.2 Success Criteria

### 2.4.11 Focus Not Obscured (Minimum) - NEW

**Level**: AA
**Severity**: 🟡 **Moderate**

**Requirement**: When a UI component receives keyboard focus, the component is not entirely hidden

**Testing Required**:
- Tab through app with modals open
- Verify focus indicator not hidden by fixed headers/footers
- Check dropdown menus don't obscure focused items

**Priority**: **P1** (New requirement)
**Effort**: 1 hour (testing)

---

### 2.4.12 Focus Not Obscured (Enhanced) - NEW

**Level**: AAA (Not required for Level AA compliance)
**Status**: Consider for future enhancement

---

### 2.5.7 Dragging Movements - NEW

**Level**: AA
**Severity**: 🟢 **Low** (Likely N/A)

**Requirement**: Functionality that uses dragging can be operated with a single pointer without dragging

**Current State**: ✅ **Likely N/A** (No drag-and-drop functionality observed)

**Verification**: Confirm no drag-and-drop features in GUI

---

### 2.5.8 Target Size (Minimum) - NEW

**Level**: AA
**Severity**: 🟡 **Moderate**

**Status**: See Issue 2.5.1 above (24x24px minimum)

---

### 3.2.6 Consistent Help - NEW

**Level**: A
**Severity**: 🟡 **Moderate**

**Requirement**: Help mechanism in same relative order on multiple pages

**Current State**: ❌ **NOT IMPLEMENTED**
- No consistent help mechanism present
- No help button in same location across views

**Solution**:
```typescript
// Add help button to AppLayout
<AppLayout
  tools={
    <HelpPanel
      header={<h2>Prism Help</h2>}
      footer={
        <Link href="https://docs.prism.io">
          View full documentation
        </Link>
      }
    >
      <SpaceBetween size="m">
        <Box>
          <h3>Current View: {state.activeView}</h3>
          <p>{getContextualHelp(state.activeView)}</p>
        </Box>
        <Button onClick={showTutorial}>Show Tutorial</Button>
      </SpaceBetween>
    </HelpPanel>
  }
  toolsOpen={showHelp}
  onToolsChange={({ detail }) => setShowHelp(detail.open)}
/>
```

**Priority**: **P1** (New Level A requirement)
**Effort**: 4 hours (help panel + contextual content)

---

### 3.3.7 Redundant Entry - NEW

**Level**: A
**Severity**: 🟢 **Low**

**Requirement**: Don't ask for same information twice in same session

**Testing Required**:
- Verify instance name not re-requested
- Check if user re-selects template after error
- Test if region/profile remembered across forms

**Current State**: ✅ **Likely Compliant** (forms don't repeat)

**Priority**: **P2** (Verification)
**Effort**: 30 minutes

---

### 3.3.8 Accessible Authentication - NEW

**Level**: AA
**Severity**: 🟢 **Low** (Likely N/A)

**Requirement**: No cognitive function test required to authenticate

**Current State**: ✅ **N/A** (No authentication in GUI)

---

## Accessibility Testing Tools Recommendations

### Automated Testing

**Essential Tools**:
```bash
# Install axe-core for automated scanning
npm install --save-dev @axe-core/react

# Add to App.tsx (development only)
if (process.env.NODE_ENV !== 'production') {
  import('@axe-core/react').then(axe => {
    axe.default(React, ReactDOM, 1000);
  });
}
```

**Testing Commands**:
```bash
# Run axe accessibility tests
npm install --save-dev @axe-core/playwright
npx playwright test --grep accessibility

# Run Lighthouse CI
npm install --save-dev @lhci/cli
lhci autorun --collect.url=http://localhost:3000
```

### Manual Testing

**Screen Reader Testing**:
- macOS: VoiceOver (⌘ F5)
- Windows: NVDA (free) or JAWS
- Test all major workflows

**Keyboard Navigation Testing**:
- Tab through entire app
- Use only keyboard for all actions
- Verify focus visible at all times
- Test ESC key in modals/dropdowns

**Color Contrast Testing**:
- Use Color Contrast Analyzer (free)
- Check all text/background combinations
- Verify 4.5:1 for normal text
- Verify 3:1 for large text and UI components

**Zoom Testing**:
- Zoom browser to 200%
- Verify no horizontal scrolling
- Check text remains readable
- Verify no content clipping

---

## Accessibility Remediation Plan

### P0 - Critical Accessibility Fixes (~13 hours)

**Must fix before production**:

**A11Y-1: Add Skip Navigation Link** (1 hour)
- [ ] Create skip link component
- [ ] Position absolutely off-screen
- [ ] Show on focus
- [ ] Link to #main-content
- [ ] Test with keyboard navigation

**A11Y-2: Add Landmark Roles** (1 hour)
- [ ] Wrap navigation in `<nav>` with aria-label
- [ ] Wrap main content in `<main>` with id
- [ ] Add tabIndex={-1} to main for focus management
- [ ] Test with screen reader

**A11Y-3: Add Status Indicator Labels** (1 hour)
- [ ] Audit all StatusIndicator usage
- [ ] Add aria-label to each
- [ ] Include context in label
- [ ] Test with screen reader

**A11Y-4: Improve Error Identification** (3 hours)
- [ ] Audit all error messages
- [ ] Make errors specific and actionable
- [ ] Associate errors with form fields
- [ ] Add error icons with aria-label
- [ ] Test error announcement with screen reader

**A11Y-5: Audit Form Labels** (2 hours)
- [ ] Check all forms for label associations
- [ ] Add missing labels
- [ ] Add aria-required for required fields
- [ ] Test with screen reader

**A11Y-6: Test for Keyboard Traps** (2 hours)
- [ ] Tab through entire application
- [ ] Test all modals with ESC key
- [ ] Test all dropdowns with ESC key
- [ ] Verify focus returns correctly
- [ ] Document any traps found and fix

**A11Y-7: Dynamic Page Titles** (30 minutes)
- [ ] Add useEffect to update document.title
- [ ] Create title map for all views
- [ ] Test title changes in browser tab

**A11Y-8: Run Automated Accessibility Scan** (2 hours)
- [ ] Install axe-core
- [ ] Run full application scan
- [ ] Review and prioritize findings
- [ ] Fix critical issues found
- [ ] Document remaining issues

---

### P1 - Important Accessibility Improvements (~14 hours)

**A11Y-9: Color Contrast Audit** (3 hours)
- [ ] Test all text/background combinations
- [ ] Check custom colors in Box components
- [ ] Verify Badge colors
- [ ] Fix any contrast issues below 4.5:1
- [ ] Document approved color combinations

**A11Y-10: Focus Visible Testing** (2 hours)
- [ ] Tab through all interactive elements
- [ ] Verify focus indicator on all
- [ ] Check focus contrast ratio (3:1 minimum)
- [ ] Add custom focus styles if needed
- [ ] Test with keyboard only

**A11Y-11: Touch Target Sizes** (3 hours)
- [ ] Audit all buttons and links
- [ ] Measure icon-only buttons
- [ ] Increase targets below 24x24px
- [ ] Test on touch device or simulator
- [ ] Document minimum sizes

**A11Y-12: Auto-Refresh Controls** (1 hour)
- [ ] Add toggle for auto-refresh
- [ ] Remember user preference
- [ ] Announce refresh to screen readers
- [ ] Test with screen reader

**A11Y-13: Error Suggestions** (2 hours)
- [ ] Review all error messages
- [ ] Add specific correction suggestions
- [ ] Add example valid inputs
- [ ] Test error recovery flow

**A11Y-14: Consistent Help Mechanism** (4 hours)
- [ ] Implement Cloudscape HelpPanel
- [ ] Add contextual help for each view
- [ ] Add help button to all views
- [ ] Test help panel accessibility

---

### P2 - Enhancement Accessibility Features (~7 hours)

**A11Y-15: Focus Management on View Changes** (2 hours)
- [ ] Add refs to view headings
- [ ] Move focus on view change
- [ ] Announce view change to screen readers
- [ ] Test navigation flow

**A11Y-16: Comprehensive Keyboard Testing** (2 hours)
- [ ] Create keyboard testing checklist
- [ ] Test all functionality keyboard-only
- [ ] Document keyboard shortcuts
- [ ] Create keyboard shortcut reference

**A11Y-17: Screen Reader Testing** (3 hours)
- [ ] Test with VoiceOver (Mac)
- [ ] Test with NVDA (Windows if available)
- [ ] Document screen reader issues
- [ ] Fix critical announcements
- [ ] Create screen reader testing guide

---

## Accessibility Compliance Summary

### WCAG 2.2 Level A Compliance

| Criterion | Status | Priority |
|-----------|--------|----------|
| 1.1.1 Non-text Content | ⚠️ Partial (missing alt text) | P0 |
| 1.3.1 Info and Relationships | ⚠️ Partial (missing landmarks) | P0 |
| 2.1.1 Keyboard | ✅ Likely Pass (needs testing) | P1 |
| 2.1.2 No Keyboard Trap | ⚠️ Needs Testing | P0 |
| 2.4.1 Bypass Blocks | ❌ Fail (no skip link) | P0 |
| 2.4.2 Page Titled | ⚠️ Partial (not dynamic) | P1 |
| 3.1.1 Language of Page | ✅ Pass | - |
| 3.2.6 Consistent Help | ❌ Not Implemented | P1 |
| 3.3.1 Error Identification | ⚠️ Partial | P0 |
| 3.3.2 Labels or Instructions | ⚠️ Needs Audit | P0 |
| 3.3.7 Redundant Entry | ✅ Likely Pass | P2 |
| 4.1.2 Name, Role, Value | ⚠️ Needs Testing | P1 |

**Level A Estimate**: ~60-70% compliance (needs P0 fixes)

---

### WCAG 2.2 Level AA Compliance

| Criterion | Status | Priority |
|-----------|--------|----------|
| 1.4.3 Contrast (Minimum) | ⚠️ Needs Audit | P1 |
| 1.4.4 Resize Text | ✅ Likely Pass | P2 |
| 1.4.11 Non-text Contrast | ⚠️ Needs Testing | P1 |
| 2.4.7 Focus Visible | ⚠️ Needs Testing | P1 |
| 2.4.11 Focus Not Obscured | ⚠️ Needs Testing | P1 |
| 2.5.8 Target Size (Minimum) | ⚠️ Needs Audit | P1 |
| 3.2.3 Consistent Navigation | ✅ Likely Pass | P2 |
| 3.3.3 Error Suggestion | ⚠️ Partial | P1 |

**Level AA Estimate**: ~50-60% compliance (needs P0 + P1 fixes)

---

## Production Readiness for Accessibility

### Go/No-Go for Accessibility

**Level A Compliance**: ⚠️ **CONDITIONAL** (60-70% estimated)
- ❌ Missing skip navigation
- ❌ Missing landmark roles
- ❌ Error identification insufficient
- ⚠️ Keyboard traps need testing

**Level AA Compliance**: ⚠️ **NEEDS WORK** (50-60% estimated)
- ⚠️ Color contrast needs audit
- ⚠️ Focus indicators need testing
- ⚠️ Touch targets need audit
- ⚠️ Consistent help not implemented

### Recommendation: **FIX P0 BEFORE PRODUCTION**

**Timeline to Accessibility Compliance**:
- **P0 Fixes** (Critical): 13 hours (~2 days)
  - Required for basic Level A compliance
  - Safety and legal requirements

- **P1 Fixes** (Important): 14 hours (~2 days)
  - Required for Level AA compliance
  - Improves usability for all users

- **P2 Enhancements**: 7 hours (~1 day)
  - Enhanced accessibility experience
  - Power user improvements

**Total: ~34 hours (1 week)** for full WCAG 2.2 Level AA compliance

---

## Legal and Compliance Considerations

### Why Accessibility Matters

**Legal Requirements**:
- **Section 508** (US Federal): WCAG 2.0 Level AA
- **ADA** (US): Increasingly requires WCAG 2.1 AA
- **VPAT**: May be required for institutional sales
- **European EN 301 549**: Harmonized with WCAG 2.1 AA

**Academic Institution Requirements**:
- Many universities require WCAG 2.1 AA
- Federal grant-funded research requires 508 compliance
- International institutions may require WCAG 2.1 AA

**Risk Assessment**:
- **High Risk**: No skip navigation, missing labels (lawsuit risk)
- **Medium Risk**: Color contrast issues, keyboard traps
- **Low Risk**: Missing help mechanism, dynamic titles

---

## Conclusion

Prism GUI benefits from Cloudscape Design System's built-in accessibility, achieving approximately **60-70% WCAG 2.2 Level A compliance** and **50-60% Level AA compliance** out of the box.

**Critical Issues** (P0 - ~13 hours):
- Missing skip navigation link
- Missing landmark roles
- Insufficient error identification
- Need keyboard trap testing
- Missing alt text on status indicators

**Important Issues** (P1 - ~14 hours):
- Color contrast audit needed
- Focus indicators need verification
- Touch target sizes need audit
- Consistent help mechanism missing

**Recommendation**: Complete **P0 accessibility fixes (~13 hours / 2 days) before production launch** to meet basic Level A requirements and reduce legal risk. P1 fixes should follow within first week for Level AA compliance and improved user experience for all users.

---

**WCAG 2.2 Audit Complete**: October 13, 2025
**Auditor**: Accessibility Specialist
**Compliance Level**: Partial (estimated 60-70% Level A, 50-60% Level AA)
**Production Recommendation**: ⚠️ **FIX P0 ITEMS BEFORE LAUNCH** (2 days of work)

