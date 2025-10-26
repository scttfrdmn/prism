# Cobra Command Refactoring Plan

## Problem
The current Prism CLI has a fundamental architectural issue where commands handle their own subcommands internally using `switch` statements, bypassing Cobra's built-in subcommand and flag handling. This causes:

1. **Flags don't work**: `prism templates validate --verbose` fails because Cobra can't parse flags for internally-handled subcommands
2. **No help generation**: Subcommands don't appear in `prism templates --help`
3. **Inconsistent behavior**: Some commands use Cobra properly, others don't
4. **Poor maintainability**: Each command reimplements routing logic

## Current Anti-Pattern Examples

### Templates Command
```go
// Current: internal routing
func (tc *TemplateCommands) Templates(args []string) error {
    if len(args) > 0 {
        switch args[0] {
        case "validate":
            return tc.validateTemplates(args[1:])
        case "search":
            return tc.templatesSearch(args[1:])
        // ... more cases
        }
    }
    return tc.templatesList(args)
}
```

### Daemon Command  
```go
// Current: internal routing
func (s *SystemCommands) Daemon(args []string) error {
    switch args[0] {
    case "start":
        return s.daemonStart()
    case "stop":
        return s.daemonStop()
    // ... more cases
    }
}
```

## Proposed Solution

Convert all internally-routed commands to use proper Cobra subcommands. This allows:
- Proper flag parsing at each level
- Automatic help generation
- Consistent CLI behavior
- Better testability

### Example Refactoring (Templates)

```go
// New: Proper Cobra subcommands
func CreateTemplatesCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "templates",
        Short: "Manage Prism templates",
    }
    
    // Add subcommands
    cmd.AddCommand(
        createValidateCommand(),
        createSearchCommand(),
        createTestCommand(),
        // ... more subcommands
    )
    
    return cmd
}

func createValidateCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "validate [template]",
        Short: "Validate templates",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Now flags work!
            verbose, _ := cmd.Flags().GetBool("verbose")
            strict, _ := cmd.Flags().GetBool("strict")
            return validateTemplates(args, verbose, strict)
        },
    }
    
    // Flags properly attached to subcommand
    cmd.Flags().BoolP("verbose", "v", false, "Verbose output")
    cmd.Flags().Bool("strict", false, "Treat warnings as errors")
    
    return cmd
}
```

## Commands Needing Refactoring

### High Priority (Complex subcommands with flags)
1. **templates** - Has 10+ subcommands with various flags
   - validate (--verbose, --strict)
   - search (--category, --domain, --popular, etc.)
   - test (--suite, --verbose)
   - install (--force, --version)
   - snapshot (--name, --description, --save)

2. **daemon** - Critical system commands
   - start/stop/status
   - config subcommands
   - logs (--follow, --tail)

3. **idle** - Complex hibernation policy commands
   - profile (list, create, update, delete)
   - instance (set policies)
   - history

4. **project** - Enterprise project management
   - create/list/update/delete
   - members (add, remove, list)
   - budget subcommands

### Medium Priority (Some subcommands)
5. **volume/storage** - EFS/EBS management
   - create (--size, --type)
   - attach/detach
   - list

6. **repo** - Repository management
   - add/remove/list
   - sync

7. **pricing** - Cost analysis
   - show/estimate
   - compare

### Low Priority (Simple or less used)
8. **scaling** - Instance resizing
9. **rightsizing** - Optimization recommendations
10. **ami** - AMI discovery

## Implementation Strategy

### Phase 1: Create Cobra Command Builders
1. Create `*_cobra.go` files for each command group
2. Implement proper Cobra command structures
3. Maintain backward compatibility during transition

### Phase 2: Update Root Command Registration
1. Replace simple command registration with Cobra command builders
2. Update `root_command.go` to use new command structures

### Phase 3: Testing & Migration
1. Test all command combinations with flags
2. Update documentation
3. Remove old routing code

## Benefits

1. **Proper Flag Support**: All flags work as expected
2. **Auto-generated Help**: `prism templates validate --help` works
3. **Consistency**: All commands behave the same way
4. **Maintainability**: Less custom routing code
5. **Shell Completion**: Better support for bash/zsh completion
6. **Testing**: Easier to test individual subcommands

## Example Commands After Refactoring

```bash
# All of these will work properly:
prism templates validate --verbose --strict
prism templates search python --category "Machine Learning" --popular
prism templates test --suite performance --verbose
prism daemon logs --follow --tail 100
prism idle profile create aggressive --idle-minutes 5 --action hibernate
prism project members add my-project user@example.com --role admin
```

## Backwards Compatibility

During transition, we can maintain both patterns:
1. Keep existing internal routing as fallback
2. Gradually migrate commands to Cobra structure
3. Once all migrated, remove old routing code

## Next Steps

1. Start with `templates` command as proof of concept (already created `templates_cobra.go`)
2. If successful, apply pattern to `daemon` command
3. Continue with other commands in priority order
4. Update tests and documentation
5. Remove legacy routing code