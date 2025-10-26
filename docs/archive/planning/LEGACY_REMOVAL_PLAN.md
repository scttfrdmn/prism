# Legacy Code Removal Plan

## Overview
Prism is early in development and should not maintain backwards compatibility. All legacy code should be removed to simplify the codebase.

## Major Legacy Components to Remove

### 1. Template Compatibility System ❌
**Files to modify/remove:**
- `pkg/templates/compatibility.go` - ENTIRE FILE (maintains backward compatibility with hardcoded templates)
- `pkg/templates/templates.go` - Remove GetTemplatesForRegion, GetTemplate legacy methods
- `pkg/templates/types.go` - Remove compatibility comment (line 135)

**Impact:** Templates are now YAML-based only, no legacy format support needed.

### 2. Legacy Idle System ❌
**Files to modify/remove:**
- `pkg/types/idle_legacy.go` - ENTIRE FILE
- `pkg/types/runtime.go` - Remove legacy idle comments (lines 113-115)
- `pkg/daemon/server.go` - Remove all "Legacy idle" comments

**Impact:** Using new hibernation/idle policy system instead.

### 3. API Backward Compatibility ❌
**Files to modify/remove:**
- `pkg/api/api.go` - ENTIRE FILE (just backward compatibility aliases)
- `pkg/types/types.go` - Remove backward compatibility aliases (lines 14-22)
- `pkg/api/mock/mock_client.go` - Remove deprecated ConnectInstance (line 431)

### 4. Profile Legacy Code ❌
**Files to modify/remove:**
- `pkg/profile/security/binding.go` - Remove legacy fields (lines 21-22, 49-50, 154, 266)
- `pkg/profile/security/keychain.go` - Remove deprecated functions (lines 121-141)
- `pkg/profile/manager_test.go` - Remove or update legacy test

### 5. State Manager Legacy ❌
**Files to modify/remove:**
- `pkg/state/unified.go` - Remove legacy compatibility functions (lines 120-121)
- `pkg/state/profile_integration.go` - Remove legacy comment (line 51)
- `pkg/state/manager.go` - Remove backward compatibility comment (line 71)
- `pkg/state/user_storage.go` - Remove backward compatibility comment (line 62)

### 6. Daemon Legacy Handling ❌
**Files to modify/remove:**
- `pkg/daemon/instance_handlers.go` - Remove legacy instance handling (lines 65-70)
- `pkg/daemon/middleware.go` - Remove backward compatibility comment (line 74)
- `pkg/daemon/api_versioning.go` - Simplify, remove deprecation handling

## TODO 3: HTTP Router Standardization

### Current Situation:
- **Daemon**: Uses standard `http.ServeMux` (Go standard library)
- **Some tests**: Reference gorilla/mux

### Analysis:

**http.ServeMux (Current):**
✅ Standard library, no dependencies
✅ Simple and lightweight
✅ Perfect for our REST API needs
❌ Less routing features (but we don't need them)

**gorilla/mux:**
✅ More routing features (regex, subrouters)
❌ External dependency
❌ Project is now archived/unmaintained
❌ Overkill for our simple REST API

### Recommendation: 
**KEEP http.ServeMux** - It's the Go standard, has no dependencies, and handles our routing needs perfectly. Remove any gorilla/mux references.

## Renaming: Hibernation → Idle Policies

Since these policies handle multiple actions (hibernate, stop, terminate, alert), they should be renamed:

### Files to Rename:
1. `pkg/hibernation/` → `pkg/idle/`
   - `policies.go` → `policies.go`
   - `scheduler.go` → `scheduler.go`
   - `savings.go` → `savings.go`

2. `internal/cli/hibernation_cobra.go` → `internal/cli/idle_cobra.go`

3. `pkg/daemon/hibernation_handlers.go` → `pkg/daemon/idle_policy_handlers.go`

4. `pkg/api/client/hibernation_policies.go` → `pkg/api/client/idle_policies.go`

### CLI Command Changes:
- `prism hibernation` → `prism idle`
- `prism hibernation policy list` → `prism idle policy list`
- `prism hibernation policy apply` → `prism idle policy apply`
- `prism hibernation savings` → `prism idle savings`

### Type/Struct Renames:
- `HibernationPolicy` → `IdlePolicy`
- `HibernationScheduler` → `IdleScheduler`
- `hibernationScheduler` → `idleScheduler`
- `policyManager` → `idlePolicyManager`

## Implementation Order

1. **Phase 1: Remove Pure Legacy Code**
   - Remove compatibility.go
   - Remove idle_legacy.go
   - Remove api.go (backward compat)
   - Remove deprecated functions

2. **Phase 2: Clean Comments & Dead Code**
   - Remove all "legacy" comments
   - Remove "backward compatibility" comments
   - Remove TODO comments about removal

3. **Phase 3: Rename Hibernation → Idle**
   - Rename package directories
   - Update all imports
   - Update CLI commands
   - Update API endpoints

4. **Phase 4: Remove gorilla/mux**
   - Remove any remaining gorilla/mux imports
   - Ensure all routing uses http.ServeMux

## Benefits
- **Simpler codebase**: ~1000+ lines of legacy code removed
- **Clearer architecture**: No confusion about which system to use
- **Better naming**: "Idle policies" more accurately describes the feature
- **No external router dependency**: Using Go standard library

## Breaking Changes
✅ **Acceptable** - Project is early stage, breaking changes are expected
- Templates must be YAML (no hardcoded support)
- Old idle detection API removed (replaced with idle policies)
- CLI command renamed from `hibernation` to `idle`