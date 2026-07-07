package daemon

import (
	"testing"

	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyExpectedTransitionalState(t *testing.T) {
	tests := []struct {
		name          string
		initialState  string
		expectedState string
		wantState     string
		wantHistory   bool
	}{
		{
			name:          "start: stopped → pending",
			initialState:  "stopped",
			expectedState: "pending",
			wantState:     "pending",
			wantHistory:   true,
		},
		{
			name:          "stop: running → stopping",
			initialState:  "running",
			expectedState: "stopping",
			wantState:     "stopping",
			wantHistory:   true,
		},
		{
			name:          "hibernate: running → stopping",
			initialState:  "running",
			expectedState: "stopping",
			wantState:     "stopping",
			wantHistory:   true,
		},
		{
			name:          "resume: hibernated → pending",
			initialState:  "hibernated",
			expectedState: "pending",
			wantState:     "pending",
			wantHistory:   true,
		},
		{
			name:          "no-op: already in expected state",
			initialState:  "stopping",
			expectedState: "stopping",
			wantState:     "stopping",
			wantHistory:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createTestServer(t)

			// Seed instance in state
			instance := types.Instance{
				Name:  "test-instance",
				ID:    "i-test123",
				State: tt.initialState,
			}
			err := server.stateManager.SaveInstance(instance)
			require.NoError(t, err)

			// Apply optimistic state
			server.applyExpectedTransitionalState("test-instance", tt.expectedState)

			// Verify
			state, err := server.stateManager.LoadState()
			require.NoError(t, err)

			inst, exists := state.Instances["test-instance"]
			require.True(t, exists)
			assert.Equal(t, tt.wantState, inst.State)

			if tt.wantHistory {
				require.NotEmpty(t, inst.StateHistory)
				last := inst.StateHistory[len(inst.StateHistory)-1]
				assert.Equal(t, tt.initialState, last.FromState)
				assert.Equal(t, tt.expectedState, last.ToState)
				assert.Equal(t, "lifecycle_optimistic", last.Reason)
			} else {
				assert.Empty(t, inst.StateHistory, "no-op should not add history")
			}
		})
	}

	t.Run("nonexistent instance is no-op", func(t *testing.T) {
		server := createTestServer(t)
		// Should not panic
		server.applyExpectedTransitionalState("does-not-exist", "pending")
	})
}

func TestIsUnpropagatedAWSState(t *testing.T) {
	tests := []struct {
		name   string
		cached string
		aws    string
		want   bool
	}{
		// Propagation lag cases (must skip rollback)
		{"start lag: pending vs stopped", "pending", "stopped", true},
		{"stop lag: stopping vs running", "stopping", "running", true},
		{"terminate lag: shutting-down vs running", "shutting-down", "running", true},
		{"terminate lag: shutting-down vs stopped", "shutting-down", "stopped", true},

		// Legitimate forward progressions (must NOT skip)
		{"pending -> running", "pending", "running", false},
		{"stopping -> stopped", "stopping", "stopped", false},
		{"shutting-down -> terminated", "shutting-down", "terminated", false},

		// Non-transitional cached states (must NOT skip)
		{"running observed as stopped", "running", "stopped", false},
		{"stopped observed as running", "stopped", "running", false},
		{"running observed as stopping", "running", "stopping", false},
		{"stopped observed as pending", "stopped", "pending", false},

		// Same-state queries (caller guards on != already, but verify)
		{"pending vs pending", "pending", "pending", false},
		{"stopping vs stopping", "stopping", "stopping", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUnpropagatedAWSState(tt.cached, tt.aws)
			assert.Equal(t, tt.want, got)
		})
	}
}
