// Tests for cost-strategy lifetime computation in commands.go.
package cli

import (
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestBasicCostStrategy_LifetimeIgnoresStaleDeletionTime(t *testing.T) {
	launch := time.Date(2026, 3, 18, 14, 21, 34, 0, time.UTC)
	// Stale DeletionTime that precedes LaunchTime by ~19 minutes — what we
	// saw on a real resurrected instance whose LaunchTime was later
	// corrected. Must not produce a negative duration.
	staleDeletion := launch.Add(-19 * time.Minute)

	inst := types.Instance{
		LaunchTime:   launch,
		DeletionTime: &staleDeletion,
		State:        "stopped",
	}

	analysis := (&BasicCostStrategy{}).CalculateInstanceCost(inst, nil)

	// Age must be non-negative. Cheap check: it should not start with "0:-"
	// or contain a negative segment. Better: re-parse via formatAge logic
	// by checking the duration is positive.
	assert.NotContains(t, analysis.Age, "-",
		"age should never contain a negative segment, got %q", analysis.Age)
}

func TestBasicCostStrategy_LifetimeUsesValidDeletionTime(t *testing.T) {
	launch := time.Date(2026, 3, 18, 14, 0, 0, 0, time.UTC)
	deletion := launch.Add(2 * time.Hour)

	inst := types.Instance{
		LaunchTime:   launch,
		DeletionTime: &deletion,
		State:        "terminated",
	}

	analysis := (&BasicCostStrategy{}).CalculateInstanceCost(inst, nil)

	// 2 hours exactly: "2:00:00".
	assert.Equal(t, "2:00:00", analysis.Age)
}
