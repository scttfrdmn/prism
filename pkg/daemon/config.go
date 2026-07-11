package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config represents daemon configuration options
type Config struct {
	// Instance retention settings
	InstanceRetentionMinutes int `json:"instance_retention_minutes"` // 0 = indefinite, >0 = minutes to retain terminated instances

	// Server settings
	Port string `json:"port,omitempty"` // Server port (default: 8947)

	// Monitoring settings (future expansion)
	MonitoringIntervalSeconds int `json:"monitoring_interval_seconds,omitempty"` // Future: monitoring frequency

	// Cost Explorer reconciliation (#644). The budget spend ledger always accrues cheap list-price
	// estimates; when enabled, the observer periodically reconciles against authoritative AWS Cost
	// Explorer billed cost. OFF by default because CE calls cost money (~$0.01/call, 1-2 per
	// instance), are rate-limited, and lag ~a day.
	CostReconciliationEnabled         bool `json:"cost_reconciliation_enabled"`          // opt-in; default false
	CostReconciliationIntervalMinutes int  `json:"cost_reconciliation_interval_minutes"` // default 1440 (24h); floored at 60

	// Live budget enforcement (#656). When enabled, the daemon evaluates each project's budget against
	// the spend ledger on a throttled tick and fires its configured auto-actions (hibernate/stop/
	// prevent-launch/notify) and cushion. OFF by default because these actions are destructive
	// (auto-stopping real instances); auto-actions stay advisory until an operator opts in.
	BudgetEnforcementEnabled         bool `json:"budget_enforcement_enabled"`          // opt-in; default false
	BudgetEnforcementIntervalMinutes int  `json:"budget_enforcement_interval_minutes"` // default 10; floored at 1
}

// DefaultConfig returns the default daemon configuration
func DefaultConfig() *Config {
	return &Config{
		InstanceRetentionMinutes:          5,      // Default: 5 minutes retention
		Port:                              "8947", // Default port
		MonitoringIntervalSeconds:         60,     // Future: default monitoring interval
		CostReconciliationEnabled:         false,  // Opt-in: CE reconciliation off by default
		CostReconciliationIntervalMinutes: 1440,   // Default: daily
		BudgetEnforcementEnabled:          false,  // Opt-in: auto-actions off by default
		BudgetEnforcementIntervalMinutes:  10,     // Default: evaluate every 10 minutes
	}
}

// LoadConfig loads daemon configuration from the standard location
func LoadConfig() (*Config, error) {
	configPath := GetConfigPath()

	// If config file doesn't exist, return default config (still honoring env overrides).
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultConfig()
		applyCostReconciliationEnv(config)
		applyBudgetEnforcementEnv(config)
		return config, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read daemon config: %w", err)
	}

	// Parse config
	config := DefaultConfig() // Start with defaults
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse daemon config: %w", err)
	}

	applyCostReconciliationEnv(config)
	applyBudgetEnforcementEnv(config)
	return config, nil
}

// applyCostReconciliationEnv lets env vars override the file/default for the cost-reconciliation
// settings (env wins over file wins over default). PRISM_COST_RECONCILIATION is a bool
// ("true"/"false"); PRISM_COST_RECONCILIATION_INTERVAL is a Go duration string (e.g. "6h", "90m").
func applyCostReconciliationEnv(config *Config) {
	switch os.Getenv("PRISM_COST_RECONCILIATION") {
	case "true", "1":
		config.CostReconciliationEnabled = true
	case "false", "0":
		config.CostReconciliationEnabled = false
	}
	if v := os.Getenv("PRISM_COST_RECONCILIATION_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			config.CostReconciliationIntervalMinutes = int(d.Minutes())
		}
	}
}

// applyBudgetEnforcementEnv lets env vars override the file/default for budget enforcement (env wins
// over file wins over default). PRISM_BUDGET_ENFORCEMENT is a bool ("true"/"false");
// PRISM_BUDGET_ENFORCEMENT_INTERVAL is a Go duration string (e.g. "10m", "1h").
func applyBudgetEnforcementEnv(config *Config) {
	switch os.Getenv("PRISM_BUDGET_ENFORCEMENT") {
	case "true", "1":
		config.BudgetEnforcementEnabled = true
	case "false", "0":
		config.BudgetEnforcementEnabled = false
	}
	if v := os.Getenv("PRISM_BUDGET_ENFORCEMENT_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			config.BudgetEnforcementIntervalMinutes = int(d.Minutes())
		}
	}
}

// SaveConfig saves daemon configuration to the standard location
func SaveConfig(config *Config) error {
	configPath := GetConfigPath()

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal daemon config: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write daemon config: %w", err)
	}

	return nil
}

// GetRetentionDuration returns the retention duration from config
func (c *Config) GetRetentionDuration() time.Duration {
	if c.InstanceRetentionMinutes == 0 {
		// Return a very large duration for "indefinite" retention
		// This means terminated instances stay visible until AWS actually removes them
		return time.Hour * 24 * 365 * 10 // 10 years - effectively indefinite
	}
	return time.Duration(c.InstanceRetentionMinutes) * time.Minute
}

// GetCostReconciliationInterval returns the reconciliation interval, clamped to a 1-hour floor so a
// misconfigured tiny value can't trigger runaway Cost Explorer spend (CE lags ~a day regardless).
func (c *Config) GetCostReconciliationInterval() time.Duration {
	const floor = time.Hour
	d := time.Duration(c.CostReconciliationIntervalMinutes) * time.Minute
	if d < floor {
		return floor
	}
	return d
}

// GetBudgetEnforcementInterval returns the enforcement evaluation interval, clamped to a 1-minute
// floor so a misconfigured value can't spin the per-project pass every tick.
func (c *Config) GetBudgetEnforcementInterval() time.Duration {
	const floor = time.Minute
	d := time.Duration(c.BudgetEnforcementIntervalMinutes) * time.Minute
	if d < floor {
		return floor
	}
	return d
}

// getConfigPath returns the standard daemon configuration file path
// GetConfigPath returns the path to the daemon configuration file
func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "daemon_config.json" // Fallback
	}
	return filepath.Join(homeDir, ".prism", "daemon_config.json")
}
