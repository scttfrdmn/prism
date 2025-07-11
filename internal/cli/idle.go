package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/idle"
)

// Idle handles idle detection commands.
func (a *App) Idle(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("idle command requires a subcommand")
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "status":
		return a.idleStatus(subargs)
	case "enable":
		return a.idleEnable(subargs)
	case "disable":
		return a.idleDisable(subargs)
	case "config":
		return a.idleConfig(subargs)
	case "profile":
		return a.idleProfile(subargs)
	case "domain":
		return a.idleDomain(subargs)
	case "instance":
		return a.idleInstance(subargs)
	case "history":
		return a.idleHistory(subargs)
	default:
		return fmt.Errorf("unknown idle subcommand: %s", subcommand)
	}
}

// idleStatus shows the idle detection status.
func (a *App) idleStatus(args []string) error {
	// Create idle manager
	idleManager, err := idle.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize idle manager: %w", err)
	}

	// If instance name provided, show status for that instance
	if len(args) > 0 {
		instanceName := args[0]

		// TODO: Get instance ID from instance name
		instanceID := "i-12345678"

		// Get idle state
		state := idleManager.GetIdleState(instanceID)
		if state == nil {
			fmt.Printf("No idle state found for instance %q\n", instanceName)
			return nil
		}

		// Print idle state
		fmt.Printf("Instance: %s (%s)\n", state.InstanceName, state.InstanceID)
		fmt.Printf("Profile: %s\n", state.Profile)
		fmt.Printf("Idle: %t\n", state.IsIdle)
		if state.IsIdle && state.IdleSince != nil {
			fmt.Printf("Idle since: %s\n", state.IdleSince.Format(time.RFC3339))
			fmt.Printf("Idle duration: %s\n", time.Since(*state.IdleSince).Round(time.Second))
		}
		fmt.Printf("Last activity: %s\n", state.LastActivity.Format(time.RFC3339))
		
		if state.NextAction != nil {
			fmt.Printf("Next action: %s at %s\n", state.NextAction.Action, state.NextAction.Time.Format(time.RFC3339))
		}
		
		if state.LastMetrics != nil {
			fmt.Printf("\nLast metrics:\n")
			fmt.Printf("  CPU usage: %.1f%%\n", state.LastMetrics.CPU)
			fmt.Printf("  Memory usage: %.1f%%\n", state.LastMetrics.Memory)
			fmt.Printf("  Network activity: %.1f KBps\n", state.LastMetrics.Network)
			fmt.Printf("  Disk I/O: %.1f KBps\n", state.LastMetrics.Disk)
			if state.LastMetrics.GPU != nil {
				fmt.Printf("  GPU usage: %.1f%%\n", *state.LastMetrics.GPU)
			}
			fmt.Printf("  User activity: %t\n", state.LastMetrics.HasActivity)
		}
		
		return nil
	}

	// Show global idle detection status
	fmt.Printf("Idle detection: %s\n", boolToEnabled(idleManager.IsEnabled()))
	
	// Get default profile
	profile, err := idleManager.GetDefaultProfile()
	if err != nil {
		return fmt.Errorf("failed to get default profile: %w", err)
	}
	
	fmt.Printf("Default profile: %s\n", profile.Name)
	fmt.Printf("\nProfiles:\n")
	
	// List profiles
	profiles := idleManager.GetProfiles()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCPU\tMEM\tNET\tDISK\tGPU\tIDLE\tACTION")
	
	for _, p := range profiles {
		fmt.Fprintf(w, "%s\t%.1f%%\t%.1f%%\t%.1f KBps\t%.1f KBps\t%.1f%%\t%d min\t%s\n",
			p.Name,
			p.CPUThreshold,
			p.MemoryThreshold,
			p.NetworkThreshold,
			p.DiskThreshold,
			p.GPUThreshold,
			p.IdleMinutes,
			p.Action,
		)
	}
	
	w.Flush()
	
	// List domain mappings
	fmt.Printf("\nDomain mappings:\n")
	domainMappings := idleManager.GetDomainMappings()
	w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DOMAIN\tPROFILE")
	
	for domain, profileName := range domainMappings {
		fmt.Fprintf(w, "%s\t%s\n", domain, profileName)
	}
	
	w.Flush()
	
	// List instance overrides
	fmt.Printf("\nInstance overrides:\n")
	instanceOverrides := idleManager.GetInstanceOverrides()
	
	if len(instanceOverrides) == 0 {
		fmt.Println("No instance overrides")
	} else {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "INSTANCE\tPROFILE\tCUSTOM SETTINGS")
		
		for instance, override := range instanceOverrides {
			customSettings := []string{}
			if override.CPUThreshold != nil {
				customSettings = append(customSettings, fmt.Sprintf("CPU:%.1f%%", *override.CPUThreshold))
			}
			if override.MemoryThreshold != nil {
				customSettings = append(customSettings, fmt.Sprintf("MEM:%.1f%%", *override.MemoryThreshold))
			}
			if override.IdleMinutes != nil {
				customSettings = append(customSettings, fmt.Sprintf("IDLE:%dmin", *override.IdleMinutes))
			}
			if override.Action != nil {
				customSettings = append(customSettings, fmt.Sprintf("ACTION:%s", *override.Action))
			}
			
			fmt.Fprintf(w, "%s\t%s\t%s\n", 
				instance, 
				override.Profile,
				strings.Join(customSettings, ", "),
			)
		}
		
		w.Flush()
	}
	
	return nil
}

// idleEnable enables idle detection.
func (a *App) idleEnable(args []string) error {
	// Create idle manager
	idleManager, err := idle.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize idle manager: %w", err)
	}

	// Enable idle detection
	if err := idleManager.Enable(); err != nil {
		return fmt.Errorf("failed to enable idle detection: %w", err)
	}

	fmt.Println("Idle detection enabled")
	return nil
}

// idleDisable disables idle detection.
func (a *App) idleDisable(args []string) error {
	// Create idle manager
	idleManager, err := idle.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize idle manager: %w", err)
	}

	// Disable idle detection
	if err := idleManager.Disable(); err != nil {
		return fmt.Errorf("failed to disable idle detection: %w", err)
	}

	fmt.Println("Idle detection disabled")
	return nil
}

// idleConfig configures idle detection.
func (a *App) idleConfig(args []string) error {
	// Create idle manager
	idleManager, err := idle.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize idle manager: %w", err)
	}

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--default-profile":
			if i+1 >= len(args) {
				return fmt.Errorf("--default-profile requires a profile name")
			}
			profileName := args[i+1]
			i++
			
			if err := idleManager.SetDefaultProfile(profileName); err != nil {
				return fmt.Errorf("failed to set default profile: %w", err)
			}
			
			fmt.Printf("Default profile set to %q\n", profileName)
		}
	}

	return nil
}

// idleProfile manages idle detection profiles.
func (a *App) idleProfile(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("idle profile requires a subcommand")
	}

	subcommand := args[0]
	subargs := args[1:]

	// Create idle manager
	idleManager, err := idle.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize idle manager: %w", err)
	}

	switch subcommand {
	case "list":
		// List profiles
		profiles := idleManager.GetProfiles()
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tCPU\tMEM\tNET\tDISK\tGPU\tIDLE\tACTION")
		
		for _, p := range profiles {
			fmt.Fprintf(w, "%s\t%.1f%%\t%.1f%%\t%.1f KBps\t%.1f KBps\t%.1f%%\t%d min\t%s\n",
				p.Name,
				p.CPUThreshold,
				p.MemoryThreshold,
				p.NetworkThreshold,
				p.DiskThreshold,
				p.GPUThreshold,
				p.IdleMinutes,
				p.Action,
			)
		}
		
		return w.Flush()
		
	case "create", "add":
		if len(subargs) < 1 {
			return fmt.Errorf("profile create requires a name")
		}
		
		name := subargs[0]
		
		// Create default profile
		profile := idle.Profile{
			Name:            name,
			CPUThreshold:    10.0,
			MemoryThreshold: 30.0,
			NetworkThreshold: 50.0,
			DiskThreshold:    100.0,
			GPUThreshold:     5.0,
			IdleMinutes:      30,
			Action:           idle.Stop,
			Notification:     true,
		}
		
		// Parse flags
		for i := 1; i < len(subargs); i++ {
			switch {
			case subargs[i] == "--cpu-threshold" && i+1 < len(subargs):
				value, err := strconv.ParseFloat(subargs[i+1], 64)
				if err != nil {
					return fmt.Errorf("invalid CPU threshold: %s", subargs[i+1])
				}
				profile.CPUThreshold = value
				i++
				
			case subargs[i] == "--memory-threshold" && i+1 < len(subargs):
				value, err := strconv.ParseFloat(subargs[i+1], 64)
				if err != nil {
					return fmt.Errorf("invalid memory threshold: %s", subargs[i+1])
				}
				profile.MemoryThreshold = value
				i++
				
			case subargs[i] == "--network-threshold" && i+1 < len(subargs):
				value, err := strconv.ParseFloat(subargs[i+1], 64)
				if err != nil {
					return fmt.Errorf("invalid network threshold: %s", subargs[i+1])
				}
				profile.NetworkThreshold = value
				i++
				
			case subargs[i] == "--disk-threshold" && i+1 < len(subargs):
				value, err := strconv.ParseFloat(subargs[i+1], 64)
				if err != nil {
					return fmt.Errorf("invalid disk threshold: %s", subargs[i+1])
				}
				profile.DiskThreshold = value
				i++
				
			case subargs[i] == "--gpu-threshold" && i+1 < len(subargs):
				value, err := strconv.ParseFloat(subargs[i+1], 64)
				if err != nil {
					return fmt.Errorf("invalid GPU threshold: %s", subargs[i+1])
				}
				profile.GPUThreshold = value
				i++
				
			case subargs[i] == "--idle-minutes" && i+1 < len(subargs):
				value, err := strconv.Atoi(subargs[i+1])
				if err != nil {
					return fmt.Errorf("invalid idle minutes: %s", subargs[i+1])
				}
				profile.IdleMinutes = value
				i++
				
			case subargs[i] == "--action" && i+1 < len(subargs):
				action := idle.Action(subargs[i+1])
				if action != idle.Stop && action != idle.Hibernate && action != idle.Notify {
					return fmt.Errorf("invalid action: %s", subargs[i+1])
				}
				profile.Action = action
				i++
				
			case subargs[i] == "--notification" && i+1 < len(subargs):
				value, err := strconv.ParseBool(subargs[i+1])
				if err != nil {
					return fmt.Errorf("invalid notification setting: %s", subargs[i+1])
				}
				profile.Notification = value
				i++
			}
		}
		
		// Add profile
		if err := idleManager.AddProfile(profile); err != nil {
			return fmt.Errorf("failed to add profile: %w", err)
		}
		
		fmt.Printf("Created profile %q\n", name)
		return nil
		
	case "remove", "delete":
		if len(subargs) < 1 {
			return fmt.Errorf("profile remove requires a name")
		}
		
		name := subargs[0]
		
		// Remove profile
		if err := idleManager.RemoveProfile(name); err != nil {
			return fmt.Errorf("failed to remove profile: %w", err)
		}
		
		fmt.Printf("Removed profile %q\n", name)
		return nil
		
	default:
		return fmt.Errorf("unknown profile subcommand: %s", subcommand)
	}
}

// idleDomain manages domain-to-profile mappings.
func (a *App) idleDomain(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("idle domain requires a domain name or subcommand")
	}

	// Create idle manager
	idleManager, err := idle.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize idle manager: %w", err)
	}

	// First argument is either a domain name or a subcommand
	first := args[0]

	// Check if it's a subcommand
	switch first {
	case "list":
		// List domain mappings
		domainMappings := idleManager.GetDomainMappings()
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "DOMAIN\tPROFILE")
		
		for domain, profileName := range domainMappings {
			fmt.Fprintf(w, "%s\t%s\n", domain, profileName)
		}
		
		return w.Flush()
		
	case "remove", "delete":
		if len(args) < 2 {
			return fmt.Errorf("domain remove requires a domain name")
		}
		
		domain := args[1]
		
		// Remove domain mapping
		if err := idleManager.RemoveDomainMapping(domain); err != nil {
			return fmt.Errorf("failed to remove domain mapping: %w", err)
		}
		
		fmt.Printf("Removed mapping for domain %q\n", domain)
		return nil
	}

	// If we're here, assume the first argument is a domain name
	domain := first

	// Check for required --profile flag
	profileName := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--profile" && i+1 < len(args) {
			profileName = args[i+1]
			break
		}
	}

	if profileName == "" {
		return fmt.Errorf("domain mapping requires a --profile flag")
	}

	// Set domain mapping
	if err := idleManager.SetDomainMapping(domain, profileName); err != nil {
		return fmt.Errorf("failed to set domain mapping: %w", err)
	}

	fmt.Printf("Mapped domain %q to profile %q\n", domain, profileName)
	return nil
}

// idleInstance manages instance overrides.
func (a *App) idleInstance(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("idle instance requires an instance name or subcommand")
	}

	// Create idle manager
	idleManager, err := idle.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize idle manager: %w", err)
	}

	// First argument is either an instance name or a subcommand
	first := args[0]

	// Check if it's a subcommand
	switch first {
	case "list":
		// List instance overrides
		instanceOverrides := idleManager.GetInstanceOverrides()
		
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "INSTANCE\tPROFILE\tCUSTOM SETTINGS")
		
		for instance, override := range instanceOverrides {
			customSettings := []string{}
			if override.CPUThreshold != nil {
				customSettings = append(customSettings, fmt.Sprintf("CPU:%.1f%%", *override.CPUThreshold))
			}
			if override.MemoryThreshold != nil {
				customSettings = append(customSettings, fmt.Sprintf("MEM:%.1f%%", *override.MemoryThreshold))
			}
			if override.IdleMinutes != nil {
				customSettings = append(customSettings, fmt.Sprintf("IDLE:%dmin", *override.IdleMinutes))
			}
			if override.Action != nil {
				customSettings = append(customSettings, fmt.Sprintf("ACTION:%s", *override.Action))
			}
			
			fmt.Fprintf(w, "%s\t%s\t%s\n", 
				instance, 
				override.Profile,
				strings.Join(customSettings, ", "),
			)
		}
		
		return w.Flush()
		
	case "remove", "delete":
		if len(args) < 2 {
			return fmt.Errorf("instance remove requires an instance name")
		}
		
		instance := args[1]
		
		// Remove instance override
		if err := idleManager.RemoveInstanceOverride(instance); err != nil {
			return fmt.Errorf("failed to remove instance override: %w", err)
		}
		
		fmt.Printf("Removed override for instance %q\n", instance)
		return nil
	}

	// If we're here, assume the first argument is an instance name
	instance := first
	
	// Create override
	override := idle.InstanceOverride{}
	
	// Parse flags
	for i := 1; i < len(args); i++ {
		switch {
		case args[i] == "--profile" && i+1 < len(args):
			override.Profile = args[i+1]
			i++
			
		case args[i] == "--cpu-threshold" && i+1 < len(args):
			value, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				return fmt.Errorf("invalid CPU threshold: %s", args[i+1])
			}
			override.CPUThreshold = &value
			i++
			
		case args[i] == "--memory-threshold" && i+1 < len(args):
			value, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				return fmt.Errorf("invalid memory threshold: %s", args[i+1])
			}
			override.MemoryThreshold = &value
			i++
			
		case args[i] == "--network-threshold" && i+1 < len(args):
			value, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				return fmt.Errorf("invalid network threshold: %s", args[i+1])
			}
			override.NetworkThreshold = &value
			i++
			
		case args[i] == "--disk-threshold" && i+1 < len(args):
			value, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				return fmt.Errorf("invalid disk threshold: %s", args[i+1])
			}
			override.DiskThreshold = &value
			i++
			
		case args[i] == "--gpu-threshold" && i+1 < len(args):
			value, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				return fmt.Errorf("invalid GPU threshold: %s", args[i+1])
			}
			override.GPUThreshold = &value
			i++
			
		case args[i] == "--idle-minutes" && i+1 < len(args):
			value, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("invalid idle minutes: %s", args[i+1])
			}
			override.IdleMinutes = &value
			i++
			
		case args[i] == "--action" && i+1 < len(args):
			action := idle.Action(args[i+1])
			if action != idle.Stop && action != idle.Hibernate && action != idle.Notify {
				return fmt.Errorf("invalid action: %s", args[i+1])
			}
			override.Action = &action
			i++
			
		case args[i] == "--notification" && i+1 < len(args):
			value, err := strconv.ParseBool(args[i+1])
			if err != nil {
				return fmt.Errorf("invalid notification setting: %s", args[i+1])
			}
			override.Notification = &value
			i++
		}
	}
	
	// Set instance override
	if err := idleManager.SetInstanceOverride(instance, override); err != nil {
		return fmt.Errorf("failed to set instance override: %w", err)
	}
	
	fmt.Printf("Set override for instance %q\n", instance)
	return nil
}

// idleHistory shows the idle detection history.
func (a *App) idleHistory(args []string) error {
	// Create idle manager
	idleManager, err := idle.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize idle manager: %w", err)
	}

	// If instance name provided, show history for that instance
	if len(args) > 0 {
		instanceName := args[0]

		// TODO: Get instance ID from instance name
		instanceID := "i-12345678"

		// Get instance history
		history := idleManager.GetInstanceHistory(instanceID)
		if len(history) == 0 {
			fmt.Printf("No idle history for instance %q\n", instanceName)
			return nil
		}

		// Print history
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "Idle history for instance %q:\n\n", instanceName)
		fmt.Fprintln(w, "TIME\tACTION\tDURATION")
		
		for _, entry := range history {
			fmt.Fprintf(w, "%s\t%s\t%s\n", 
				entry.Time.Format(time.RFC3339),
				entry.Action,
				entry.IdleDuration,
			)
		}
		
		return w.Flush()
	}

	// Show global history
	history := idleManager.GetHistory()
	if len(history) == 0 {
		fmt.Println("No idle history")
		return nil
	}

	// Print history
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TIME\tINSTANCE\tACTION\tDURATION")
	
	for _, entry := range history {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", 
			entry.Time.Format(time.RFC3339),
			entry.InstanceName,
			entry.Action,
			entry.IdleDuration,
		)
	}
	
	return w.Flush()
}

// boolToEnabled converts a boolean to "enabled" or "disabled".
func boolToEnabled(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}