package cli

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// configureIdleDetection handles runtime idle detection configuration with proper types
func (a *App) configureIdleDetection(instanceName string, enable, disable bool, idleMinutes, hibernateMinutes, checkInterval int) error {
	// Validate enable/disable flags
	if enable && disable {
		return fmt.Errorf("cannot specify both --enable and --disable flags")
	}

	// If no parameters provided, just show current config
	if !enable && !disable && idleMinutes == 0 && hibernateMinutes == 0 && checkInterval == 0 {
		return a.idleConfigShow(instanceName)
	}

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	var targetInstance *struct {
		Name     string `json:"name"`
		PublicIP string `json:"public_ip"`
		State    string `json:"state"`
	}

	for _, instance := range response.Instances {
		if instance.Name == instanceName {
			targetInstance = &struct {
				Name     string `json:"name"`
				PublicIP string `json:"public_ip"`
				State    string `json:"state"`
			}{
				Name:     instance.Name,
				PublicIP: instance.PublicIP,
				State:    instance.State,
			}
			break
		}
	}

	if targetInstance == nil {
		return fmt.Errorf("instance %q not found", instanceName)
	}

	if targetInstance.State != "running" {
		return fmt.Errorf("instance %q is not running (current state: %s)", instanceName, targetInstance.State)
	}

	if targetInstance.PublicIP == "" {
		return fmt.Errorf("instance %q has no public IP address", instanceName)
	}

	// Convert to pointers for updateIdleConfig
	var enablePtr *bool
	var idlePtr, hibernatePtr, intervalPtr *int

	if enable || disable {
		enablePtr = &enable // enable=true means enable, enable=false with disable=true means disable
		if disable {
			enableVal := false
			enablePtr = &enableVal
		}
	}
	if idleMinutes > 0 {
		idlePtr = &idleMinutes
	}
	if hibernateMinutes > 0 {
		hibernatePtr = &hibernateMinutes
	}
	if checkInterval > 0 {
		intervalPtr = &checkInterval
	}

	// Update configuration on the instance
	if err := a.updateIdleConfig(targetInstance.PublicIP, enablePtr, idlePtr, hibernatePtr, intervalPtr); err != nil {
		return fmt.Errorf("failed to update idle configuration: %w", err)
	}

	fmt.Printf("✅ Successfully updated idle detection configuration for %q\n", instanceName)

	// Show updated configuration
	return a.idleConfigShow(instanceName)
}

// idleConfigShow displays the current idle configuration for an instance
func (a *App) idleConfigShow(instanceName string) error {
	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	var publicIP string
	var state string
	found := false

	for _, instance := range response.Instances {
		if instance.Name == instanceName {
			publicIP = instance.PublicIP
			state = instance.State
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("instance %q not found", instanceName)
	}

	if state != "running" {
		fmt.Printf("Instance %q is not running (state: %s). Cannot retrieve idle configuration.\n", instanceName, state)
		return nil
	}

	if publicIP == "" {
		return fmt.Errorf("instance %q has no public IP address", instanceName)
	}

	// Get current configuration
	config, err := a.getIdleConfig(publicIP)
	if err != nil {
		return fmt.Errorf("failed to get idle configuration: %w", err)
	}

	fmt.Printf("🔧 Idle Detection Configuration for %q:\n", instanceName)

	if config.Enabled {
		fmt.Printf("   Status:              ✅ ENABLED\n")
		fmt.Printf("   Idle threshold:      %d minutes\n", config.IdleMinutes)
		fmt.Printf("   Hibernate threshold: %d minutes\n", config.HibernateMinutes)
		fmt.Printf("   Check interval:      %d minutes\n", config.CheckInterval)
	} else {
		fmt.Printf("   Status:              ❌ DISABLED\n")
		fmt.Printf("   Idle threshold:      %d minutes (ignored while disabled)\n", config.IdleMinutes)
		fmt.Printf("   Hibernate threshold: %d minutes (ignored while disabled)\n", config.HibernateMinutes)
		fmt.Printf("   Check interval:      %d minutes (minimal overhead)\n", config.CheckInterval)
		fmt.Printf("   💡 Enable with: cws idle configure %s --enable\n", instanceName)
	}

	return nil
}

// IdleConfig represents the idle detection configuration
type IdleConfig struct {
	Enabled          bool
	IdleMinutes      int
	HibernateMinutes int
	CheckInterval    int
}

// updateIdleConfig updates the idle configuration on a running instance
func (a *App) updateIdleConfig(publicIP string, enabled *bool, idleMinutes, hibernateMinutes, checkInterval *int) error {
	sshKey := a.getSSHKeyPath()

	// Create a comprehensive update script that handles permissions properly
	var scriptLines []string

	// First, get current config and create updated version
	scriptLines = append(scriptLines, "# Read current config")
	scriptLines = append(scriptLines, "source /etc/cloudworkstation/idle-config 2>/dev/null || true")

	// Update variables with new values if provided
	if enabled != nil {
		if *enabled {
			scriptLines = append(scriptLines, "ENABLED=true")
			// When enabling, set reasonable defaults if no specific values provided
			if idleMinutes == nil {
				scriptLines = append(scriptLines, "IDLE_THRESHOLD_MINUTES=5")
			}
			if hibernateMinutes == nil {
				scriptLines = append(scriptLines, "HIBERNATE_THRESHOLD_MINUTES=10")
			}
			if checkInterval == nil {
				scriptLines = append(scriptLines, "CHECK_INTERVAL_MINUTES=2")
			}
		} else {
			scriptLines = append(scriptLines, "ENABLED=false")
		}
	}
	if idleMinutes != nil {
		scriptLines = append(scriptLines, fmt.Sprintf("IDLE_THRESHOLD_MINUTES=%d", *idleMinutes))
	}
	if hibernateMinutes != nil {
		scriptLines = append(scriptLines, fmt.Sprintf("HIBERNATE_THRESHOLD_MINUTES=%d", *hibernateMinutes))
	}
	if checkInterval != nil {
		scriptLines = append(scriptLines, fmt.Sprintf("CHECK_INTERVAL_MINUTES=%d", *checkInterval))
	}

	// Create new config file content
	scriptLines = append(scriptLines, "# Write updated config")
	scriptLines = append(scriptLines, "cat > /tmp/idle-config-new << 'EOF'")
	scriptLines = append(scriptLines, "# CloudWorkstation Idle Detection Configuration")
	scriptLines = append(scriptLines, "# This file is automatically updated by runtime configuration")
	scriptLines = append(scriptLines, "ENABLED=${ENABLED:-false}")
	scriptLines = append(scriptLines, "IDLE_THRESHOLD_MINUTES=${IDLE_THRESHOLD_MINUTES:-999999}")
	scriptLines = append(scriptLines, "HIBERNATE_THRESHOLD_MINUTES=${HIBERNATE_THRESHOLD_MINUTES:-999999}")
	scriptLines = append(scriptLines, "CHECK_INTERVAL_MINUTES=${CHECK_INTERVAL_MINUTES:-60}")
	scriptLines = append(scriptLines, "EOF")

	// Move the new config to the proper location with sudo
	scriptLines = append(scriptLines, "sudo mv /tmp/idle-config-new /etc/cloudworkstation/idle-config")
	scriptLines = append(scriptLines, "sudo chmod 644 /etc/cloudworkstation/idle-config")

	// Update cron job if check interval changed
	if checkInterval != nil {
		scriptLines = append(scriptLines, "sudo /usr/local/bin/cloudworkstation-update-cron.sh")
	}

	updateScript := strings.Join(scriptLines, "; ")

	// Execute the update via SSH
	cmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "LogLevel=ERROR",
		"-i", sshKey,
		fmt.Sprintf("ubuntu@%s", publicIP),
		updateScript,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("SSH command failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// getIdleConfig retrieves the current idle configuration from a running instance
func (a *App) getIdleConfig(publicIP string) (*IdleConfig, error) {
	sshKey := a.getSSHKeyPath()

	// Get configuration via SSH
	cmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "LogLevel=ERROR",
		"-i", sshKey,
		fmt.Sprintf("ubuntu@%s", publicIP),
		"sudo bash -c 'source /etc/cloudworkstation/idle-config && echo $ENABLED $IDLE_THRESHOLD_MINUTES $HIBERNATE_THRESHOLD_MINUTES $CHECK_INTERVAL_MINUTES'",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("SSH command failed: %w\nOutput: %s", err, string(output))
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) != 4 {
		return nil, fmt.Errorf("unexpected configuration format: %s", string(output))
	}

	enabled := parts[0] == "true"

	idleMinutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid idle minutes: %s", parts[1])
	}

	hibernateMinutes, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid hibernate minutes: %s", parts[2])
	}

	checkInterval, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, fmt.Errorf("invalid check interval: %s", parts[3])
	}

	return &IdleConfig{
		Enabled:          enabled,
		IdleMinutes:      idleMinutes,
		HibernateMinutes: hibernateMinutes,
		CheckInterval:    checkInterval,
	}, nil
}

// getSSHKeyPath returns the path to the SSH key to use for connections
func (a *App) getSSHKeyPath() string {
	// Use the standard CloudWorkstation key
	return "~/.ssh/cws-my-account-key"
}
