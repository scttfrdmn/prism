# CloudWorkstation CLI Architecture

## Overview

CloudWorkstation CLI uses a **two-layer architecture** that separates user interface concerns from business logic. This document explains the architecture to prevent confusion about "duplicate" command files.

## Architecture Pattern: Facade/Adapter

### Layer 1: Cobra Command Layer (`*_cobra.go` files)
**Purpose:** User-facing CLI interface
**Responsibilities:**
- Define command structure and subcommands
- Parse and validate command-line flags
- Provide help text, examples, and usage information
- Delegate to implementation layer for execution

**Example:** `storage_cobra.go`, `templates_cobra.go`, `idle_cobra.go`

### Layer 2: Implementation Layer (`*_impl.go` files)
**Purpose:** Business logic and API integration
**Responsibilities:**
- Execute actual operations (API calls, data processing)
- Format output for display
- Handle errors and edge cases
- Provide reusable logic for multiple interfaces (CLI, TUI, tests)

**Example:** `storage_impl.go`, `template_impl.go`, `instance_impl.go`

## Why This Architecture?

### 1. Separation of Concerns
- **CLI concerns** (flags, help text) stay in Cobra layer
- **Business logic** (API calls, formatting) stays in implementation layer
- Changes to CLI interface don't require touching business logic

### 2. Reusability
Implementation layer methods can be called from:
- Cobra commands (CLI interface)
- TUI interface (terminal UI)
- Test code (unit and integration tests)
- Internal app methods

### 3. Consistency
All commands follow the same pattern, making the codebase easier to understand and maintain.

### 4. Testability
- Cobra layer can be tested for CLI behavior
- Implementation layer can be tested for business logic
- Each layer has focused, specific tests

## File Naming Convention

| Layer | Pattern | Example | Purpose |
|-------|---------|---------|---------|
| Cobra | `*_cobra.go` | `storage_cobra.go` | User-facing CLI interface |
| Implementation | `*_impl.go` | `storage_impl.go` | Business logic implementation |

## Code Example

### Cobra Layer (storage_cobra.go)
```go
func (sc *StorageCobraCommands) createCreateCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "create <name>",
        Short: "Create a new EBS volume",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            // Parse flags
            size, _ := cmd.Flags().GetString("size")
            volumeType, _ := cmd.Flags().GetString("type")

            // Delegate to implementation layer
            return sc.app.storageCommands.CreateStorage(args[0], size, volumeType)
        },
    }

    // Define flags
    cmd.Flags().String("size", "100", "Volume size in GB")
    cmd.Flags().String("type", "gp3", "Volume type")

    return cmd
}
```

### Implementation Layer (storage_impl.go)
```go
func (sc *StorageCommands) CreateStorage(name, size, volumeType string) error {
    // Business logic
    req := types.StorageCreateRequest{
        Name:       name,
        Size:       size,
        VolumeType: volumeType,
    }

    // API call
    volume, err := sc.app.apiClient.CreateStorage(sc.app.ctx, req)
    if err != nil {
        return fmt.Errorf("failed to create storage: %w", err)
    }

    // Format output
    fmt.Printf("✅ Created EBS volume: %s (%s, %s)\n", volume.VolumeID, size, volumeType)
    return nil
}
```

## Architecture Documentation Status

### ✅ Fully Documented Two-Layer Commands

These commands have both layers with comprehensive architecture documentation:

- **templates** - `templates_cobra.go` (CLI) + `template_impl.go` (business logic)
- **storage** - `storage_cobra.go` (CLI) + `storage_impl.go` (business logic)
- **idle** - `idle_cobra.go` + idle policy management
- **daemon** - `daemon_cobra.go` + `system_impl.go`
- **project** - `project_cobra.go` + project management
- **repo** - `repo_cobra.go` + repository management
- **ami** - `ami_cobra.go` + AMI operations
- **marketplace** - `marketplace_cobra.go` + marketplace features
- **policy** - `policy_cobra.go` + policy enforcement
- **rightsizing** - `rightsizing_cobra.go` + `scaling_impl.go`
- **research-user** - `research_user_cobra.go` + multi-user management

### ✅ Documented Single-Layer Commands

These commands have architecture documentation explaining their single-layer pattern:

- **instance** - `instance_impl.go` (business logic, no dedicated Cobra file - integrated in root)
- **backup** - `backup_impl.go` (straightforward operations)
- **snapshot** - `snapshot_impl.go` (instance snapshot management)
- **system** - `system_impl.go` (daemon lifecycle and configuration)
- **scaling** - `scaling_impl.go` (scaling and rightsizing operations)

### Single-Layer Commands (No Separate Documentation Needed)

These are simple enough not to need architecture documentation:

- **admin** - `admin_commands.go` (simple passthrough)
- **logs** - `logs_commands.go` (straightforward log display)
- **budget** - `budget_commands.go` (comprehensive but direct)
- **user** - `user_commands.go` (multi-user management)

## Common Misconceptions

### ❌ "The `*_impl.go` files are old/deprecated code"
**Reality:** These files contain the current, active business logic. They are not old code - they are the implementation layer.

### ❌ "We should delete the `*_impl.go` files after Cobra migration"
**Reality:** Cobra commands DEPEND on these files. Deleting them would break the CLI.

### ❌ "There's duplicate code between Cobra and implementation files"
**Reality:** There's no duplication - they serve different purposes. Cobra handles CLI interface, implementation handles business logic.

## Future Development

When adding new commands:

1. **Simple commands** - Can be implemented directly in Cobra (single file)
2. **Complex commands** - Use two-layer architecture:
   - Create `command_cobra.go` for CLI interface
   - Create `command_impl.go` for business logic
   - Follow existing patterns for consistency

## Questions?

If you're unsure whether a file is needed, check:
1. Does a `*_cobra.go` file reference it? → Keep it
2. Does `app.go` initialize it? → Keep it
3. Do tests use it? → Keep it
4. Is it listed in this document? → Keep it

**When in doubt, keep the file.** The architecture is working and production-ready.
