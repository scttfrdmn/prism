# Issue #66 Implementation Plan: Storage Terminology Overhaul

**GitHub Issue**: [#66 - Storage Terminology Overhaul: EBS → Local, EFS → Shared, S3 → Cloud](https://github.com/scttfrdmn/prism/issues/66)
**Milestone**: v0.5.7
**Estimated Effort**: 3 weeks
**Status**: Planning

---

## Overview

Transform AWS-centric storage terminology (EBS, EFS, S3) to researcher-friendly terms (Local Storage, Shared Storage, Cloud Storage) across all Prism interfaces while maintaining AWS technical details via `--verbose` flag.

---

## Design Decisions (From Issue Discussion)

### ✅ Approved Decisions

1. **Terminology Mapping**:
   - EBS → "Local Storage"
   - EFS → "Shared Storage"
   - S3 → "Cloud Storage"

2. **No Backward Compatibility**: Pre-release (v0.5.x), so breaking changes are acceptable

3. **Verbose Flag Strategy**: Use `--verbose` to show AWS technical details when needed

4. **Command Structure**:
   ```bash
   # OLD (AWS-centric)
   prism ebs create my-data --size 500GB
   prism efs create shared-data --size 1TB

   # NEW (User-friendly)
   prism storage create-local my-data --size 500GB
   prism storage create-shared shared-data --size 1TB
   ```

---

## Implementation Phases

### Phase 1: Core Type System & API (Week 1)

**Goal**: Update internal types and daemon API to support new terminology

#### Tasks:

1. **Update Type Definitions** (`pkg/types/storage.go`)
   - Add `StorageType` enum: `Local`, `Shared`, `Cloud`
   - Add `AWSService` enum: `EBS`, `EFS`, `S3` (internal use)
   - Update `StorageVolume` struct with `Type` and `AWSService` fields
   - Maintain `aws_service` field in JSON for API compatibility

   ```go
   type StorageType string
   const (
       StorageTypeLocal  StorageType = "local"
       StorageTypeShared StorageType = "shared"
       StorageTypeCloud  StorageType = "cloud"
   )

   type AWSService string
   const (
       AWSServiceEBS AWSService = "ebs"
       AWSServiceEFS AWSService = "efs"
       AWSServiceS3  AWSService = "s3"
   )

   type StorageVolume struct {
       Name       string      `json:"name"`
       Type       StorageType `json:"type"`          // User-facing
       AWSService AWSService  `json:"aws_service"`   // Technical detail
       Size       string      `json:"size"`
       // ... other fields
   }
   ```

2. **Update Daemon API Handlers** (`pkg/daemon/storage_handlers.go`)
   - Modify storage list endpoint to include `type` field
   - Update create/attach/detach endpoints for new terminology
   - Ensure `aws_service` field always included in responses

3. **Update API Client** (`pkg/api/client/storage.go`)
   - Update request/response types to match new API
   - Add helper methods: `IsLocal()`, `IsShared()`, `IsCloud()`

4. **Testing**:
   - Unit tests for type conversions
   - API integration tests
   - Backward compatibility tests (verify old state files work)

**Success Criteria**:
- ✅ All storage API endpoints return `type` and `aws_service` fields
- ✅ Internal code uses new `StorageType` enum
- ✅ Existing state files (with old AWS terminology) still load correctly

---

### Phase 2: CLI Command Restructure (Week 1-2)

**Goal**: Reorganize CLI commands around unified `prism storage` hierarchy

#### Tasks:

1. **Create New Storage Command Structure** (`internal/cli/storage_cobra.go`)

   ```bash
   prism storage                            # Main storage command
   ├── list                               # List all storage
   ├── create-local <name> --size 500GB   # Create local storage (EBS)
   ├── create-shared <name> --size 1TB    # Create shared storage (EFS)
   ├── attach <storage-name> <workspace>  # Attach to workspace
   ├── detach <storage-name> <workspace>  # Detach from workspace
   └── delete <storage-name>              # Delete storage
   ```

2. **Deprecate Old Commands** (`internal/cli/`)
   - Keep `prism ebs` and `prism efs` commands functional but mark as deprecated
   - Add deprecation warnings: "⚠️  This command is deprecated. Use 'cws storage create-local' instead"
   - Update help text to point to new commands

3. **Update Command Help Text**
   - All help text uses "Local Storage", "Shared Storage", "Cloud Storage"
   - Examples show new command syntax
   - `--verbose` flag explanation in help text

4. **Create Migration Guide** (`docs/MIGRATION_v0.5.6_to_v0.5.7.md`)
   - Document command changes
   - Provide migration examples
   - Explain deprecation timeline

5. **Testing**:
   - Test all new storage commands
   - Verify deprecated commands still work
   - Test `--verbose` flag behavior

**Success Criteria**:
- ✅ `prism storage list` shows storage with user-friendly types
- ✅ `prism storage list --verbose` shows AWS service details
- ✅ Old commands work with deprecation warnings
- ✅ Help text uses new terminology consistently

---

### Phase 3: GUI & TUI Updates (Week 2)

**Goal**: Update visual interfaces with new terminology

#### 3.1 GUI Updates (`cmd/cws-gui/frontend/src/`)

1. **Storage List View** (`App.tsx`)
   - Update table headers: "Type" column shows "Local", "Shared", "Cloud"
   - Add tooltip on hover showing AWS service (EBS, EFS, S3)
   - Update badges/icons for storage types
   - Add "Technical Details" toggle in Settings

2. **Storage Creation Dialogs**
   - "Create Local Storage" button (formerly "Create EBS Volume")
   - "Create Shared Storage" button (formerly "Create EFS Filesystem")
   - Form labels use new terminology
   - Help text explains use cases (not AWS services)

3. **Settings Panel**
   - Add toggle: "Show AWS Technical Details"
   - When enabled: Shows AWS service names, resource IDs, technical specs
   - When disabled: Shows only user-friendly terminology

**Success Criteria**:
- ✅ Storage list shows "Local", "Shared", "Cloud" by default
- ✅ AWS details visible only when "Technical Details" enabled
- ✅ All creation workflows use new terminology

#### 3.2 TUI Updates (`internal/tui/models/`)

1. **Storage Page** (`storage.go`)
   - Update table columns: Show "Type" (Local/Shared/Cloud)
   - Add "T" key binding to toggle technical view
   - Default view: User-friendly terminology
   - Technical view: AWS service details

2. **Help Text**
   - Update all storage-related help text
   - Document "T" key for technical details

**Success Criteria**:
- ✅ TUI shows user-friendly storage types by default
- ✅ "T" key toggles AWS technical details
- ✅ Consistent with GUI terminology

---

### Phase 4: Documentation & Examples (Week 2-3)

**Goal**: Update all documentation with new terminology

#### Tasks:

1. **Update User Guides** (`docs/user-guides/`)
   - `STORAGE_GUIDE.md`: Rewrite with new terminology
   - `QUICK_START.md`: Update storage examples
   - All examples use `prism storage` commands

2. **Update Architecture Docs** (`docs/architecture/`)
   - Update diagrams showing storage types
   - Explain mapping to AWS services
   - Document when to use each storage type

3. **Update USER_SCENARIOS** (`docs/USER_SCENARIOS/`)
   - Search/replace AWS storage terms in user scenarios
   - Update command examples
   - Ensure consistent terminology

4. **Update README** (`README.md`)
   - Update feature descriptions
   - Update command examples
   - Update screenshots if needed

5. **Update TERMINOLOGY_GLOSSARY** (`docs/user-guides/TERMINOLOGY_GLOSSARY.md`)
   - Already created in Issue #15
   - Update with final storage terminology

6. **Create Migration Guide** (`docs/MIGRATION_v0.5.6_to_v0.5.7.md`)
   - Document all breaking changes
   - Provide command mapping table
   - Include migration script if needed

**Success Criteria**:
- ✅ All user-facing documentation uses new terminology
- ✅ AWS technical terms explained in glossary
- ✅ Migration guide complete with examples

---

### Phase 5: Testing & Validation (Week 3)

**Goal**: Comprehensive testing across all interfaces

#### Tasks:

1. **Unit Tests**
   - Storage type conversions
   - API request/response handling
   - State file loading (backward compatibility)

2. **Integration Tests**
   - Storage creation workflow (local, shared, cloud)
   - Attach/detach operations
   - List operations with/without --verbose

3. **UI Testing**
   - GUI: All storage workflows with new terminology
   - TUI: All storage operations, technical toggle
   - CLI: All new commands, deprecated command warnings

4. **Documentation Review**
   - Manual review of all updated docs
   - Verify no AWS terms in user-facing content
   - Verify technical details available via --verbose/settings

5. **User Testing** (if available)
   - Test with 2-3 researchers unfamiliar with AWS
   - Verify terminology clarity
   - Collect feedback on usability

**Success Criteria**:
- ✅ All tests passing
- ✅ No AWS terminology in default user experience
- ✅ `--verbose` flag provides AWS details where needed
- ✅ Deprecated commands work with warnings

---

## Command Mapping Reference

### Old Commands → New Commands

| Old Command | New Command | Notes |
|-------------|-------------|-------|
| `prism ebs create NAME --size 500GB` | `prism storage create-local NAME --size 500GB` | EBS → Local Storage |
| `prism efs create NAME --size 1TB` | `prism storage create-shared NAME --size 1TB` | EFS → Shared Storage |
| `prism ebs list` | `prism storage list --type local` | Filter by type |
| `prism efs list` | `prism storage list --type shared` | Filter by type |
| `prism volume attach NAME WORKSPACE` | `prism storage attach NAME WORKSPACE` | Unified command |
| `prism volume detach NAME WORKSPACE` | `prism storage detach NAME WORKSPACE` | Unified command |
| `prism ebs delete NAME` | `prism storage delete NAME` | Auto-detects type |
| `prism efs delete NAME` | `prism storage delete NAME` | Auto-detects type |

### Verbose Flag Examples

```bash
# Default: User-friendly
$ prism storage list
NAME         TYPE     SIZE    ATTACHED TO
my-data      Local    500GB   my-workspace
shared-lab   Shared   1TB     —
datasets     Cloud    —       —

# Verbose: AWS technical details
$ prism storage list --verbose
NAME         TYPE (AWS)        SIZE    AWS RESOURCE         ATTACHED TO
my-data      Local (EBS gp3)   500GB   vol-abc123456789     my-workspace
shared-lab   Shared (EFS)      1TB     fs-def456789012      —
datasets     Cloud (S3)        —       s3://my-bucket       —
```

---

## Code Changes Checklist

### Core Types & API
- [ ] Update `pkg/types/storage.go` with new enums
- [ ] Update `pkg/daemon/storage_handlers.go` API responses
- [ ] Update `pkg/api/client/storage.go` client methods
- [ ] Add unit tests for type system

### CLI Commands
- [ ] Create `internal/cli/storage_cobra.go` with unified commands
- [ ] Add deprecation warnings to `ebs_commands.go` and `efs_commands.go`
- [ ] Update all help text and examples
- [ ] Add `--verbose` flag support to storage commands
- [ ] Create CLI integration tests

### GUI
- [ ] Update `cmd/cws-gui/frontend/src/App.tsx` storage views
- [ ] Update storage creation dialogs
- [ ] Add "Show AWS Technical Details" setting
- [ ] Update tooltips and help text
- [ ] Test all storage workflows

### TUI
- [ ] Update `internal/tui/models/storage.go` display logic
- [ ] Add "T" key binding for technical toggle
- [ ] Update help text
- [ ] Test technical view toggle

### Documentation
- [ ] Update `docs/user-guides/STORAGE_GUIDE.md`
- [ ] Update `docs/user-guides/QUICK_START.md`
- [ ] Update `docs/USER_SCENARIOS/*.md` (7 files)
- [ ] Update `docs/architecture/STORAGE_ARCHITECTURE.md`
- [ ] Update `docs/user-guides/TERMINOLOGY_GLOSSARY.md`
- [ ] Create `docs/MIGRATION_v0.5.6_to_v0.5.7.md`
- [ ] Update `README.md` examples

### Testing
- [ ] Unit tests for storage type system
- [ ] API integration tests
- [ ] CLI command tests (new + deprecated)
- [ ] GUI workflow tests
- [ ] TUI interaction tests
- [ ] Backward compatibility tests (state files)
- [ ] Documentation review

---

## Migration Strategy

### For Users

1. **No Action Required (Automatic)**:
   - Existing storage automatically gets `type` field
   - Old commands continue working (with deprecation warnings)
   - State files automatically migrated on load

2. **Recommended Updates**:
   - Start using `prism storage` commands
   - Update scripts to use new command syntax
   - Review new terminology in documentation

3. **Timeline**:
   - v0.5.7: New commands available, old commands deprecated
   - v0.6.0: Old commands removed (tentative)

### For Developers

1. **Code Updates**:
   - Use `StorageType` enum in new code
   - Add `--verbose` support to custom storage commands
   - Update integration tests

2. **API Changes**:
   - All storage endpoints return `type` and `aws_service` fields
   - `aws_service` field always present for backward compatibility

---

## Success Metrics

### Quantitative

- ✅ Zero AWS service names (EBS/EFS/S3) in default CLI output
- ✅ Zero AWS service names in GUI without "Technical Details" enabled
- ✅ 100% of storage commands have `--verbose` flag
- ✅ All documentation updated (60+ files)
- ✅ All tests passing

### Qualitative

- ✅ New users can create/manage storage without AWS knowledge
- ✅ AWS experts can access technical details when needed
- ✅ Terminology consistent across CLI/TUI/GUI
- ✅ Migration smooth for existing users

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking changes confuse existing users | Medium | Deprecation warnings, migration guide, old commands work |
| AWS experts feel terms are "dumbed down" | Low | `--verbose` flag, technical toggle, glossary |
| Incomplete documentation updates | High | Systematic documentation review, search for AWS terms |
| State file incompatibility | High | Automatic migration on load, extensive testing |
| CLI muscle memory broken | Medium | Keep old commands working with warnings for 1-2 releases |

---

## Dependencies

### Blocks
- Nothing (can start immediately after Issue #15 merge)

### Blocked By
- Issue #15 must be complete first (workspace terminology)

### Related
- Issue #15: Instances → Workspaces (Similar UX improvement)
- Phase 5.0 UX Redesign: Overall terminology simplification initiative

---

## Next Steps

1. **Review this implementation plan** with stakeholders
2. **Create GitHub milestones** for v0.5.7
3. **Break down Phase 1 into GitHub issues** (one per task)
4. **Begin implementation**: Start with Phase 1 (Core Type System & API)
5. **Regular check-ins**: Weekly progress reviews during 3-week implementation

---

**Last Updated**: 2025-10-24
**Status**: Ready for implementation after Issue #15 merge
**Assigned**: TBD
