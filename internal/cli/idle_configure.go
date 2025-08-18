package cli

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// configureIdleDetection handles runtime idle detection configuration using Strategy Pattern (SOLID: Single Responsibility)
func (a *App) configureIdleDetection(instanceName string, enable, disable bool, idleMinutes, hibernateMinutes, checkInterval int) error {
	// Create and execute idle configuration command
	configCmd := NewIdleConfigurationCommand(a.apiClient)
	return configCmd.Configure(instanceName, enable, disable, idleMinutes, hibernateMinutes, checkInterval)
}

// idleConfigShow displays the current idle configuration for an instance using Strategy Pattern (SOLID: Single Responsibility)
func (a *App) idleConfigShow(instanceName string) error {
	// Create display service and show configuration
	instanceService := NewIdleInstanceService(a.apiClient)
	displayService := NewIdleDisplayService()
	return displayService.ShowConfiguration(instanceName, instanceService)
}

// IdleConfig represents the idle detection configuration
type IdleConfig struct {
	Enabled          bool
	IdleMinutes      int
	HibernateMinutes int
	CheckInterval    int
}

// getSSHKeyPath returns the path to the SSH key to use for connections
func (a *App) getSSHKeyPath() string {
	// Use the standard CloudWorkstation key
	return "~/.ssh/cws-my-account-key"
}

// Idle Configuration Strategy Pattern Implementation (SOLID: Single Responsibility + Open/Closed)

// IdleConfigurationCommand handles idle detection configuration using Strategy Pattern (SOLID: Single Responsibility)
type IdleConfigurationCommand struct {
	apiClient         interface{}
	validationService *IdleValidationService
	instanceService   *IdleInstanceService
	parameterService  *IdleParameterService
	updateService     *IdleUpdateService
	displayService    *IdleDisplayService
}

// NewIdleConfigurationCommand creates a new idle configuration command
func NewIdleConfigurationCommand(apiClient interface{}) *IdleConfigurationCommand {
	return &IdleConfigurationCommand{
		apiClient:         apiClient,
		validationService: NewIdleValidationService(),
		instanceService:   NewIdleInstanceService(apiClient),
		parameterService:  NewIdleParameterService(),
		updateService:     NewIdleUpdateService(),
		displayService:    NewIdleDisplayService(),
	}
}

// Configure configures idle detection using Strategy Pattern
func (c *IdleConfigurationCommand) Configure(instanceName string, enable, disable bool, idleMinutes, hibernateMinutes, checkInterval int) error {
	// Validate input parameters
	if err := c.validationService.ValidateFlags(enable, disable); err != nil {
		return err
	}

	// Check if showing configuration only
	if c.validationService.ShouldShowConfig(enable, disable, idleMinutes, hibernateMinutes, checkInterval) {
		return c.displayService.ShowConfiguration(instanceName, c.instanceService)
	}

	// Find and validate target instance
	targetInstance, err := c.instanceService.FindAndValidateInstance(instanceName)
	if err != nil {
		return err
	}

	// Convert parameters to configuration
	configParams := c.parameterService.ConvertToConfigParams(enable, disable, idleMinutes, hibernateMinutes, checkInterval)

	// Update configuration on the instance
	if err := c.updateService.UpdateConfiguration(targetInstance.PublicIP, configParams); err != nil {
		return fmt.Errorf("failed to update idle configuration: %w", err)
	}

	fmt.Printf("✅ Successfully updated idle detection configuration for %q\n", instanceName)

	// Show updated configuration
	return c.displayService.ShowConfiguration(instanceName, c.instanceService)
}

// IdleValidationService handles parameter validation using Strategy Pattern (SOLID: Single Responsibility)
type IdleValidationService struct{}

func NewIdleValidationService() *IdleValidationService {
	return &IdleValidationService{}
}

func (s *IdleValidationService) ValidateFlags(enable, disable bool) error {
	if enable && disable {
		return fmt.Errorf("cannot specify both --enable and --disable flags")
	}
	return nil
}

func (s *IdleValidationService) ShouldShowConfig(enable, disable bool, idleMinutes, hibernateMinutes, checkInterval int) bool {
	return !enable && !disable && idleMinutes == 0 && hibernateMinutes == 0 && checkInterval == 0
}

// IdleInstanceService handles instance operations using Strategy Pattern (SOLID: Single Responsibility)
type IdleInstanceService struct {
	apiClient interface{}
}

type IdleInstanceInfo struct {
	Name     string
	PublicIP string
	State    string
}

func NewIdleInstanceService(apiClient interface{}) *IdleInstanceService {
	return &IdleInstanceService{apiClient: apiClient}
}

func (s *IdleInstanceService) FindAndValidateInstance(instanceName string) (*IdleInstanceInfo, error) {
	// Check daemon is running
	if pingable, ok := s.apiClient.(interface{ Ping(interface{}) error }); ok {
		if err := pingable.Ping(nil); err != nil {
			return nil, fmt.Errorf("daemon not running. Start with: cws daemon start")
		}
	}

	// Get instance list
	if lister, ok := s.apiClient.(interface {
		ListInstances(interface{}) (interface{}, error)
	}); ok {
		response, err := lister.ListInstances(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list instances: %w", err)
		}

		// Find target instance
		if respData, ok := response.(interface{ Instances() []interface{} }); ok {
			for _, instance := range respData.Instances() {
				if instData, ok := instance.(interface {
					Name() string
					PublicIP() string
					State() string
				}); ok && instData.Name() == instanceName {
					return s.validateInstanceState(&IdleInstanceInfo{
						Name:     instData.Name(),
						PublicIP: instData.PublicIP(),
						State:    instData.State(),
					})
				}
			}
		}
	}

	return nil, fmt.Errorf("instance %q not found", instanceName)
}

func (s *IdleInstanceService) validateInstanceState(instance *IdleInstanceInfo) (*IdleInstanceInfo, error) {
	if instance.State != "running" {
		return nil, fmt.Errorf("instance %q is not running (current state: %s)", instance.Name, instance.State)
	}

	if instance.PublicIP == "" {
		return nil, fmt.Errorf("instance %q has no public IP address", instance.Name)
	}

	return instance, nil
}

// IdleParameterService handles parameter conversion using Strategy Pattern (SOLID: Single Responsibility)
type IdleParameterService struct{}

type IdleConfigParams struct {
	EnablePtr    *bool
	IdlePtr      *int
	HibernatePtr *int
	IntervalPtr  *int
}

func NewIdleParameterService() *IdleParameterService {
	return &IdleParameterService{}
}

func (s *IdleParameterService) ConvertToConfigParams(enable, disable bool, idleMinutes, hibernateMinutes, checkInterval int) *IdleConfigParams {
	params := &IdleConfigParams{}

	// Handle enable/disable flags
	if enable || disable {
		enableVal := enable && !disable
		params.EnablePtr = &enableVal
	}

	// Handle numeric parameters
	if idleMinutes > 0 {
		params.IdlePtr = &idleMinutes
	}
	if hibernateMinutes > 0 {
		params.HibernatePtr = &hibernateMinutes
	}
	if checkInterval > 0 {
		params.IntervalPtr = &checkInterval
	}

	return params
}

// IdleUpdateService handles configuration updates using Strategy Pattern (SOLID: Single Responsibility)
type IdleUpdateService struct{}

func NewIdleUpdateService() *IdleUpdateService {
	return &IdleUpdateService{}
}

func (s *IdleUpdateService) UpdateConfiguration(publicIP string, params *IdleConfigParams) error {
	sshKey := "~/.ssh/cws-my-account-key"

	// Build update script
	scriptBuilder := NewIdleScriptBuilder()
	updateScript := scriptBuilder.BuildUpdateScript(params)

	// Execute via SSH
	return s.executeSSHCommand(publicIP, sshKey, updateScript)
}

func (s *IdleUpdateService) executeSSHCommand(publicIP, sshKey, script string) error {
	cmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "LogLevel=ERROR",
		"-i", sshKey,
		fmt.Sprintf("ubuntu@%s", publicIP),
		script,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("SSH command failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// IdleScriptBuilder builds update scripts using Builder Pattern (SOLID: Single Responsibility)
type IdleScriptBuilder struct{}

func NewIdleScriptBuilder() *IdleScriptBuilder {
	return &IdleScriptBuilder{}
}

func (b *IdleScriptBuilder) BuildUpdateScript(params *IdleConfigParams) string {
	var scriptLines []string

	// Read current config
	scriptLines = append(scriptLines, "# Read current config")
	scriptLines = append(scriptLines, "source /etc/cloudworkstation/idle-config 2>/dev/null || true")

	// Update variables
	scriptLines = append(scriptLines, b.buildVariableUpdates(params)...)

	// Create new config file
	scriptLines = append(scriptLines, b.buildConfigFileCreation()...)

	// Move and set permissions
	scriptLines = append(scriptLines, b.buildFileOperations()...)

	// Update cron if needed
	if params.IntervalPtr != nil {
		scriptLines = append(scriptLines, "sudo /usr/local/bin/cloudworkstation-update-cron.sh")
	}

	return strings.Join(scriptLines, "; ")
}

func (b *IdleScriptBuilder) buildVariableUpdates(params *IdleConfigParams) []string {
	var lines []string

	if params.EnablePtr != nil {
		if *params.EnablePtr {
			lines = append(lines, "ENABLED=true")
			// Set defaults when enabling
			if params.IdlePtr == nil {
				lines = append(lines, "IDLE_THRESHOLD_MINUTES=5")
			}
			if params.HibernatePtr == nil {
				lines = append(lines, "HIBERNATE_THRESHOLD_MINUTES=10")
			}
			if params.IntervalPtr == nil {
				lines = append(lines, "CHECK_INTERVAL_MINUTES=2")
			}
		} else {
			lines = append(lines, "ENABLED=false")
		}
	}

	if params.IdlePtr != nil {
		lines = append(lines, fmt.Sprintf("IDLE_THRESHOLD_MINUTES=%d", *params.IdlePtr))
	}
	if params.HibernatePtr != nil {
		lines = append(lines, fmt.Sprintf("HIBERNATE_THRESHOLD_MINUTES=%d", *params.HibernatePtr))
	}
	if params.IntervalPtr != nil {
		lines = append(lines, fmt.Sprintf("CHECK_INTERVAL_MINUTES=%d", *params.IntervalPtr))
	}

	return lines
}

func (b *IdleScriptBuilder) buildConfigFileCreation() []string {
	return []string{
		"# Write updated config",
		"cat > /tmp/idle-config-new << 'EOF'",
		"# CloudWorkstation Idle Detection Configuration",
		"# This file is automatically updated by runtime configuration",
		"ENABLED=${ENABLED:-false}",
		"IDLE_THRESHOLD_MINUTES=${IDLE_THRESHOLD_MINUTES:-999999}",
		"HIBERNATE_THRESHOLD_MINUTES=${HIBERNATE_THRESHOLD_MINUTES:-999999}",
		"CHECK_INTERVAL_MINUTES=${CHECK_INTERVAL_MINUTES:-60}",
		"EOF",
	}
}

func (b *IdleScriptBuilder) buildFileOperations() []string {
	return []string{
		"sudo mv /tmp/idle-config-new /etc/cloudworkstation/idle-config",
		"sudo chmod 644 /etc/cloudworkstation/idle-config",
	}
}

// IdleDisplayService handles configuration display using Strategy Pattern (SOLID: Single Responsibility)
type IdleDisplayService struct{}

func NewIdleDisplayService() *IdleDisplayService {
	return &IdleDisplayService{}
}

func (s *IdleDisplayService) ShowConfiguration(instanceName string, instanceService *IdleInstanceService) error {
	// Find instance for display
	targetInstance, err := instanceService.FindAndValidateInstance(instanceName)
	if err != nil {
		// Handle non-running instances gracefully
		if strings.Contains(err.Error(), "not running") {
			fmt.Printf("Instance %q is not running. Cannot retrieve idle configuration.\n", instanceName)
			return nil
		}
		return err
	}

	// Get and display current configuration
	configRetriever := NewIdleConfigRetriever()
	config, err := configRetriever.GetConfiguration(targetInstance.PublicIP)
	if err != nil {
		return fmt.Errorf("failed to get idle configuration: %w", err)
	}

	s.displayConfigurationStatus(instanceName, config)
	return nil
}

func (s *IdleDisplayService) displayConfigurationStatus(instanceName string, config *IdleConfig) {
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
}

// IdleConfigRetriever handles configuration retrieval using Strategy Pattern (SOLID: Single Responsibility)
type IdleConfigRetriever struct{}

func NewIdleConfigRetriever() *IdleConfigRetriever {
	return &IdleConfigRetriever{}
}

func (r *IdleConfigRetriever) GetConfiguration(publicIP string) (*IdleConfig, error) {
	sshKey := "~/.ssh/cws-my-account-key"

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

	return r.parseConfigOutput(string(output))
}

func (r *IdleConfigRetriever) parseConfigOutput(output string) (*IdleConfig, error) {
	parts := strings.Fields(strings.TrimSpace(output))
	if len(parts) != 4 {
		return nil, fmt.Errorf("unexpected configuration format: %s", output)
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
