package daemon

import (
	"testing"
	"time"
)

func TestDefaultConfig_CostReconciliationOffByDefault(t *testing.T) {
	c := DefaultConfig()
	if c.CostReconciliationEnabled {
		t.Fatal("cost reconciliation must default to OFF (opt-in)")
	}
	if c.GetCostReconciliationInterval() != 24*time.Hour {
		t.Fatalf("default interval = %v, want 24h", c.GetCostReconciliationInterval())
	}
}

func TestGetCostReconciliationInterval_Floor(t *testing.T) {
	c := &Config{CostReconciliationIntervalMinutes: 5} // below the 1h floor
	if got := c.GetCostReconciliationInterval(); got != time.Hour {
		t.Fatalf("interval = %v, want 1h floor (misconfig guard)", got)
	}
	c2 := &Config{CostReconciliationIntervalMinutes: 360} // 6h, above floor
	if got := c2.GetCostReconciliationInterval(); got != 6*time.Hour {
		t.Fatalf("interval = %v, want 6h", got)
	}
}

func TestEnvOverride_CostReconciliation(t *testing.T) {
	c := DefaultConfig()
	t.Setenv("PRISM_COST_RECONCILIATION", "true")
	t.Setenv("PRISM_COST_RECONCILIATION_INTERVAL", "6h")
	applyCostReconciliationEnv(c)
	if !c.CostReconciliationEnabled {
		t.Fatal("env PRISM_COST_RECONCILIATION=true should enable")
	}
	if c.GetCostReconciliationInterval() != 6*time.Hour {
		t.Fatalf("env interval override = %v, want 6h", c.GetCostReconciliationInterval())
	}
	// env can also force-disable.
	t.Setenv("PRISM_COST_RECONCILIATION", "false")
	applyCostReconciliationEnv(c)
	if c.CostReconciliationEnabled {
		t.Fatal("env PRISM_COST_RECONCILIATION=false should disable")
	}
}
