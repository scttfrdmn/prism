package daemon

import (
	"net/http"

	"github.com/scttfrdmn/prism/pkg/aws"
	"github.com/scttfrdmn/prism/pkg/types"
)

// handleBilledCost serves GET /api/v1/instances/{name}/billed-cost.
//
// It returns the AWS-billed ("billed so far") cost for the workspace from AWS
// Cost Explorer -- the authoritative meter -- rather than prism's local
// estimate (Instance.CurrentSpend). The two can diverge; the CLI shows both.
func (s *Server) handleBilledCost(w http.ResponseWriter, r *http.Request, identifier string) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	instanceName, found := s.resolveInstanceIdentifier(identifier)
	if !found {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}
	cached, exists := state.Instances[instanceName]
	if !exists {
		s.writeError(w, http.StatusNotFound, "Instance not found in state")
		return
	}

	// Test mode: return synthetic data and never call AWS, so E2E runs do not
	// block on Cost Explorer (see CLAUDE.md test-mode contract).
	if s.testMode {
		s.writeJSON(w, http.StatusOK, syntheticBilledCost(cached))
		return
	}

	var result *types.BilledCostResult
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		var qerr error
		result, qerr = awsManager.GetBilledCost(r.Context(), instanceName, cached.Region, cached.LaunchTime)
		return qerr
	})
	if result == nil {
		// withAWSManager already wrote an error response.
		return
	}

	s.writeJSON(w, http.StatusOK, result)
}

// syntheticBilledCost returns deterministic billed-cost data for test mode.
// It echoes the instance's locally-estimated spend so reconciliation renders a
// zero delta, which keeps E2E assertions stable.
func syntheticBilledCost(inst types.Instance) types.BilledCostResult {
	return types.BilledCostResult{
		Name:        inst.Name,
		BilledTotal: inst.CurrentSpend,
		Currency:    "USD",
		Region:      inst.Region,
		Source:      "synthetic (PRISM_TEST_MODE)",
		TagActive:   true,
	}
}
