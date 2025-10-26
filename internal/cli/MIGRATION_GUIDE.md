# CLI Cobra Migration Guide

## Overview
This guide explains how to migrate Prism CLI commands from internal routing to proper Cobra subcommands for consistent flag handling and better user experience.

## Migration Status

### ✅ Completed
- **templates** - Full Cobra implementation with all subcommands and flags

### 🚧 In Progress
- **daemon** - Example implementation created, needs integration
- **idle** - Structure defined in cobra_migration.go
- **project** - Structure defined in cobra_migration.go
- **storage/volume** - Structure defined in cobra_migration.go
- **repo** - Structure defined in cobra_migration.go

### 📋 To Do
- **pricing** - Cost analysis commands
- **scaling** - Instance resizing
- **rightsizing** - Optimization recommendations
- **ami** - AMI discovery

## Step-by-Step Migration Process

### 1. Create Cobra Command Structure

Create a new file `<command>_cobra.go`:

```go
package cli

import "github.com/spf13/cobra"

type <Command>CobraCommands struct {
    app *App
    originalCommands *<Command>Commands  // Reference to existing implementation
}

func New<Command>CobraCommands(app *App) *<Command>CobraCommands {
    return &<Command>CobraCommands{
        app: app,
        originalCommands: New<Command>Commands(app),
    }
}

func (c *<Command>CobraCommands) Create<Command>Command() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "command",
        Short: "Short description",
        Long:  "Long description",
    }
    
    // Add subcommands
    cmd.AddCommand(
        c.createSubcommand1(),
        c.createSubcommand2(),
    )
    
    return cmd
}
```

### 2. Define Subcommands with Flags

```go
func (c *<Command>CobraCommands) createSubcommand1() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "subcommand [args]",
        Short: "Description",
        Args:  cobra.ExactArgs(1),  // or MinimumNArgs, etc.
        RunE: func(cmd *cobra.Command, args []string) error {
            // Get flag values
            flag1, _ := cmd.Flags().GetString("flag1")
            flag2, _ := cmd.Flags().GetBool("flag2")
            
            // Call existing implementation
            return c.originalCommands.subcommandImpl(args, flag1, flag2)
        },
    }
    
    // Define flags for this subcommand
    cmd.Flags().String("flag1", "default", "Description")
    cmd.Flags().BoolP("flag2", "f", false, "Description")
    cmd.MarkFlagRequired("flag1")  // If needed
    
    return cmd
}
```

### 3. Update Root Command Registration

In `root_command.go`, update the command factory:

```go
func (f *CommandFactory) createCommand() *cobra.Command {
    // Old way (remove this):
    // return &cobra.Command{
    //     Use: "command",
    //     RunE: func(cmd *cobra.Command, args []string) error {
    //         return f.app.Command(args)
    //     },
    // }
    
    // New way:
    cmdCobra := New<Command>CobraCommands(f.app)
    return cmdCobra.Create<Command>Command()
}
```

### 4. Maintain Backward Compatibility

During migration, keep the original implementation working:

```go
func (c *<Command>CobraCommands) createSubcommand() *cobra.Command {
    return &cobra.Command{
        Use: "subcommand",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Build args array for original implementation
            var origArgs []string
            
            if flag1, _ := cmd.Flags().GetString("flag1"); flag1 != "" {
                origArgs = append(origArgs, "--flag1", flag1)
            }
            
            // Call original implementation
            return c.originalCommands.Subcommand(origArgs)
        },
    }
}
```

### 5. Test the Migration

Test all command variations:

```bash
# Test help generation
prism command --help
prism command subcommand --help

# Test flag parsing
prism command subcommand --flag1 value --flag2
prism command subcommand -f  # Short flags

# Test argument validation
prism command subcommand arg1 arg2  # Should fail if expecting 1 arg

# Test required flags
prism command subcommand  # Should fail if flag is required
```

### 6. Update Documentation

Update command documentation to reflect new structure:

```markdown
## Command Usage

### command subcommand
Description of what this does.

**Usage:**
```
prism command subcommand [flags]
```

**Flags:**
- `--flag1 string`: Description (default: "value")
- `-f, --flag2`: Description

**Examples:**
```bash
prism command subcommand --flag1 value
prism command subcommand -f
```
```

### 7. Remove Old Implementation

Once fully tested, remove the old internal routing:

1. Remove switch statements from original command handlers
2. Remove manual flag parsing code
3. Update tests to use new structure
4. Remove `*Commands` types if no longer needed

## Benefits After Migration

### Before (Internal Routing)
```go
func (c *Commands) Command(args []string) error {
    if len(args) > 0 {
        switch args[0] {
        case "subcommand":
            // Manual flag parsing
            var flag1 string
            for i := 1; i < len(args); i++ {
                if args[i] == "--flag1" && i+1 < len(args) {
                    flag1 = args[i+1]
                    i++
                }
            }
            return c.subcommand(flag1)
        }
    }
}
```

### After (Cobra Subcommands)
```go
func (c *CobraCommands) createSubcommand() *cobra.Command {
    cmd := &cobra.Command{
        Use: "subcommand",
        RunE: func(cmd *cobra.Command, args []string) error {
            flag1, _ := cmd.Flags().GetString("flag1")
            return c.subcommand(flag1)
        },
    }
    cmd.Flags().String("flag1", "", "Description")
    return cmd
}
```

## Common Patterns

### Pattern 1: Subcommand Groups
```go
// For commands like: prism project members add
func createMembersCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "members",
        Short: "Manage members",
    }
    
    cmd.AddCommand(
        &cobra.Command{Use: "add"},
        &cobra.Command{Use: "remove"},
        &cobra.Command{Use: "list"},
    )
    
    return cmd
}
```

### Pattern 2: Optional Arguments
```go
cmd := &cobra.Command{
    Use:   "list [filter]",
    Short: "List items",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        filter := ""
        if len(args) > 0 {
            filter = args[0]
        }
        return list(filter)
    },
}
```

### Pattern 3: Multiple Value Flags
```go
cmd.Flags().StringSlice("tags", []string{}, "Tags to apply")
// Usage: prism command --tags tag1 --tags tag2
// Or: prism command --tags tag1,tag2
```

### Pattern 4: Mutually Exclusive Flags
```go
cmd.MarkFlagsMutuallyExclusive("json", "yaml", "table")
```

## Testing Checklist

- [ ] Help text generates correctly at all levels
- [ ] All flags are parsed correctly
- [ ] Short flags work (`-v` for `--verbose`)
- [ ] Required flags are enforced
- [ ] Argument counts are validated
- [ ] Flag defaults work
- [ ] Subcommand routing works
- [ ] Error messages are clear
- [ ] Tab completion works (if configured)

## Rollback Plan

If issues arise during migration:

1. Keep original implementation files
2. Add feature flag to switch between implementations
3. Test thoroughly in development before production
4. Have ability to quickly revert command registration

## Next Steps

1. Complete migration of high-priority commands (daemon, idle, project)
2. Test each migrated command thoroughly
3. Update CLI documentation
4. Remove old routing code once all commands migrated
5. Add shell completion support