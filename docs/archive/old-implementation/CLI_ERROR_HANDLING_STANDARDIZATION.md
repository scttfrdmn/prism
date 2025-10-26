# CLI Error Handling Standardization - Phase 4 POLISH

## Overview

This document summarizes the comprehensive standardization of error handling patterns across Prism CLI modules, completed as part of Phase 4 POLISH to improve code consistency and user experience.

## Goals Achieved

### 1. Consistent Error Message Formats ✅

**Before:**
- Mixed error formats: `"failed to X: %w"`, `"error doing X"`, `"cannot X"`
- Inconsistent daemon connection messages
- Technical jargon in user-facing errors

**After:**
- Standardized API errors: `WrapAPIError("action", err)`
- Consistent daemon errors: `WrapDaemonError(err)`
- User-friendly validation errors: `NewValidationError(field, value, expected)`

### 2. Error Context Enhancement ✅

**Before:**
```go
return fmt.Errorf("failed to get instance status: %w", err)
```

**After:**
```go
return WrapAPIError("get instance status", err)
```

### 3. Daemon Connection Errors ✅

**Before:** Mixed messages and inconsistent daemon checks

**After:** 
- Standardized `WrapDaemonError(err)` with consistent recovery message
- All daemon checks use same pattern: "daemon not running. Start with: prism daemon start"

### 4. Validation Errors ✅

**Before:**
```go
return fmt.Errorf("unknown flag: %s", flag)
```

**After:**
```go
return NewValidationError("flag", flag, "--verbose or -v")
```

### 5. User Experience Improvements ✅

**Before:** Inconsistent emoji and message formats

**After:**
- `FormatSuccessMessage("action", "resource", "details")`
- `FormatProgressMessage("action", "status")`  
- `FormatWarningMessage("context", "message")`
- `FormatErrorMessage("context", "message")`
- `FormatInfoMessage("tip")`

## Implementation Details

### Error Helper Functions Created

Added to `/Users/scttfrdmn/src/prism/internal/cli/constants.go`:

```go
// WrapAPIError wraps API errors with consistent context and formatting
func WrapAPIError(action string, err error) error {
	return fmt.Errorf("failed to %s: %w", action, err)
}

// WrapDaemonError wraps daemon connection errors with standard recovery message
func WrapDaemonError(err error) error {
	return fmt.Errorf("%s", DaemonNotRunningMessage)
}

// NewUsageError creates a consistent usage error with command and example
func NewUsageError(command, example string) error {
	if example != "" {
		return fmt.Errorf("usage: %s\n\nExample: %s", command, example)
	}
	return fmt.Errorf("usage: %s", command)
}

// NewValidationError creates a validation error with field context
func NewValidationError(field, value, expected string) error {
	if expected != "" {
		return fmt.Errorf("invalid %s '%s': expected %s", field, value, expected)
	}
	return fmt.Errorf("invalid %s '%s'", field, value)
}

// NewNotFoundError creates a resource not found error with suggestions
func NewNotFoundError(resourceType, name, suggestion string) error {
	if suggestion != "" {
		return fmt.Errorf("%s '%s' not found. %s", resourceType, name, suggestion)
	}
	return fmt.Errorf("%s '%s' not found", resourceType, name)
}

// NewStateError creates an error for invalid resource states
func NewStateError(resourceType, name, currentState, expectedState string) error {
	if expectedState != "" {
		return fmt.Errorf("%s '%s' is in state '%s', expected '%s'", resourceType, name, currentState, expectedState)
	}
	return fmt.Errorf("%s '%s' is in invalid state '%s'", resourceType, name, currentState)
}

// FormatSuccessMessage, FormatProgressMessage, FormatWarningMessage, 
// FormatErrorMessage, FormatInfoMessage - consistent user message formatting
```

### Files Updated

1. **constants.go** - Added error helper functions and import fmt
2. **instance_commands.go** - Standardized all error handling patterns
3. **storage_commands.go** - Standardized volume and storage command errors  
4. **template_commands.go** - Standardized template operation errors
5. **system_commands.go** - Standardized daemon management errors
6. **scaling_commands.go** - Standardized rightsizing and scaling errors
7. **app.go** - Standardized core application errors

### Pattern Examples

#### Usage Errors
**Before:**
```go
return fmt.Errorf("usage: prism connect <instance-name> [--verbose]")
```

**After:**
```go
return NewUsageError("cws connect <instance-name> [--verbose]", "cws connect my-workstation")
```

#### API Errors
**Before:**
```go
return fmt.Errorf("failed to stop instance: %w", err)
```

**After:**
```go
return WrapAPIError("stop instance "+name, err)
```

#### Validation Errors
**Before:**
```go
return fmt.Errorf("invalid size '%s'. Valid sizes: XS, S, M, L, XL", size)
```

**After:**
```go
return NewValidationError("size", size, "XS, S, M, L, XL")
```

#### User Messages
**Before:**
```go
fmt.Printf("⏹️ Stopping instance %s...\n", name)
```

**After:**
```go
fmt.Printf("%s\n", FormatProgressMessage("Stopping instance", name))
```

## Benefits Achieved

### 1. Consistency
- All error messages follow the same patterns
- Consistent emoji usage and formatting
- Standardized recovery suggestions

### 2. User Experience
- Clear, actionable error messages
- Examples provided for usage errors
- Helpful suggestions for problem resolution

### 3. Maintainability
- Centralized error message logic
- Easy to update error formats globally
- Reduced code duplication

### 4. Developer Experience
- Clear patterns to follow for new code
- Type-safe error construction
- Consistent error context information

## Testing

- All CLI modules compile successfully
- Error helper functions properly integrated
- Consistent behavior across all commands
- Maintains backward compatibility

## Future Improvements

1. **Internationalization Support** - Error helper functions can be extended to support multiple languages
2. **Error Codes** - Could add error codes for programmatic handling
3. **Structured Logging** - Error context could be enhanced for better logging
4. **Error Recovery** - More intelligent error recovery suggestions based on context

## Conclusion

The CLI error handling standardization successfully achieved:
- ✅ Consistent error message formats
- ✅ Enhanced error context 
- ✅ Standardized daemon connection errors
- ✅ Improved validation error messages
- ✅ Better user experience with helpful messaging

This improvement enhances Prism's professional quality and user experience while maintaining the existing functionality and improving code maintainability.