# Code Quality Best Practices

## Cyclomatic Complexity Management

**Goal**: Keep all functions under complexity 15 (Go Report Card A+ grade requirement)

### Root Causes of High Complexity

1. **Giant Switch Statements** - Nested switches create exponential complexity
2. **Inline Logic** - All logic in one massive function instead of extracted helpers
3. **Complex Conditionals** - Multiple `if/else` chains without extraction

### Solution Pattern: Extract Helper Methods

#### Before (Bad - Complexity 37):
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "a":
            if m.tab == 0 && len(m.items) > 0 {
                // 10 lines of logic
                return m, someCmd
            }
        case "b":
            if m.tab == 1 && m.selected < len(m.items) {
                // 10 lines of logic
                return m, anotherCmd
            }
        // ... 20 more cases
        }
    case OtherMsg:
        // ... more nested logic
    }
    return m, nil
}
```

#### After (Good - Complexity <15):
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        return m.handleWindowSize(msg)
    case tea.KeyMsg:
        return m.handleKeyPress(msg)
    case DataMsg:
        return m.handleData(msg)
    }
    return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "a":
        return m.handleActionA()
    case "b":
        return m.handleActionB()
    // ... each case delegates to a focused function
    }
    return m, nil
}

func (m Model) handleActionA() (tea.Model, tea.Cmd) {
    if m.tab == 0 && len(m.items) > 0 {
        // Focused logic for action A
        return m, someCmd
    }
    return m, nil
}
```

### Benefits of This Pattern

1. **Lower Complexity**: Each function does one thing well
2. **Better Testing**: Can test individual handlers in isolation
3. **Easier Maintenance**: Clear function names document behavior
4. **Better Navigation**: Jump to `handleActionA()` instead of searching a 500-line function

### When to Extract

Extract a helper function when you see:
- **Nested switches** (switch inside switch)
- **Long case blocks** (>5 lines per case)
- **Repeated patterns** (similar logic in multiple cases)
- **Complex conditionals** (multiple `&&` or `||` operators)

### Naming Conventions

- Message handlers: `handleXxx(msg XxxMsg) (tea.Model, tea.Cmd)`
- Key handlers: `handleKeyA()`, `handleEnterKey()`, `handleEscKey()`
- Action handlers: `handleUpdatePolicy()`, `handleEnableFeature()`
- Data handlers: `handleIdleData()`, `handlePolicyAction()`

### TUI-Specific Pattern (BubbleTea)

All TUI `Update()` methods should follow this pattern:
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        return m.handleWindowSize(msg)
    case CustomDataMsg:
        return m.handleCustomData(msg)
    case tea.KeyMsg:
        return m.handleKeyPress(msg)  // Delegates to key-specific handlers
    }
    return m, nil
}
```

### CLI-Specific Pattern

Large CLI command handlers should be split:
```go
// Before: 200-line function with nested flag parsing
func (a *App) AMI(args []string) error {
    // 200 lines of flag parsing and logic
}

// After: Clean dispatcher with focused handlers
func (a *App) AMI(args []string) error {
    if len(args) == 0 {
        return a.showAMIHelp()
    }

    switch args[0] {
    case "list":
        return a.handleAMIList(args[1:])
    case "validate":
        return a.handleAMIValidate(args[1:])
    case "cleanup":
        return a.handleAMICleanup(args[1:])
    default:
        return fmt.Errorf("unknown AMI command: %s", args[0])
    }
}
```

### Real Example from Prism

**File**: `internal/tui/models/idle.go`
**Before**: IdleModel.Update() - Complexity 37
**After**: Extracted 14 handler methods - Complexity <15

See the actual implementation for the pattern in action.

## Additional Best Practices

### 1. Early Returns
Use early returns to reduce nesting:
```go
// Good
func process(item *Item) error {
    if item == nil {
        return errors.New("nil item")
    }
    if !item.IsValid() {
        return errors.New("invalid item")
    }
    // Main logic here
    return nil
}
```

### 2. Guard Clauses
Extract complex conditions into named functions:
```go
// Before
if len(items) > 0 && items[0].Type == "special" && items[0].Status == "active" {
    // do something
}

// After
if hasActiveSpecialItem(items) {
    // do something
}

func hasActiveSpecialItem(items []Item) bool {
    return len(items) > 0 &&
           items[0].Type == "special" &&
           items[0].Status == "active"
}
```

### 3. Table-Driven Logic
Replace long if/else chains with maps:
```go
// Before
func getAction(key string) string {
    if key == "a" { return "add" }
    if key == "d" { return "delete" }
    if key == "u" { return "update" }
    // ... 20 more
}

// After
var keyActions = map[string]string{
    "a": "add",
    "d": "delete",
    "u": "update",
    // ...
}

func getAction(key string) string {
    return keyActions[key]
}
```

## Automated Checking

Run `gocyclo -over 15 .` before committing to catch complexity issues early.

Add to CI/CD pipeline:
```bash
# Fail build if any function exceeds complexity 15
gocyclo -over 15 . | grep -q "." && exit 1
```

## Summary

- **Target**: All functions < 15 cyclomatic complexity
- **Method**: Extract helper functions for nested logic
- **Pattern**: One switch level max, delegate to focused handlers
- **Result**: A+ Go Report Card grade, maintainable code
