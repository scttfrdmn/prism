package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/scttfrdmn/prism/pkg/aws"
	"github.com/scttfrdmn/prism/pkg/templates"
	"github.com/scttfrdmn/prism/pkg/types"
)

// handleLaunchSweep launches a parameter sweep: one instance per parameter set,
// named <base>-0..<base>-(N-1), all sharing a generated sweep id. Each member
// carries its own parameter set, stamped by Manager.BuildTags as prism:param:<k>
// tags; an on-instance shell (installed with spored) exports them as PARAM_<k>
// env vars. The daemon does NOT parse parameter files — the client resolves the
// param sets and sends them in the request.
//
// Members are independent workspaces: partial success is normal (some launch,
// some fail); successes are not torn down on a partial failure.
func (s *Server) handleLaunchSweep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var swReq types.LaunchSweepRequest
	if err := json.NewDecoder(r.Body).Decode(&swReq); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	count := len(swReq.ParamSets)
	if !s.validateSweepSize(count, w) {
		return
	}

	req := swReq.LaunchRequest

	// Validate the embedded launch request (template, name, etc.)
	if err := s.validateLaunchRequest(&req, w); err != nil {
		return // Error response already written
	}

	// Security/policy checks: RBAC, policy-set constraints, invitation restrictions.
	if !s.checkLaunchPolicies(&req, w) {
		return
	}

	// Resolve ExpiresAt from Hours if not set directly (#146)
	if req.ExpiresAt == nil && req.Hours > 0 {
		t := time.Now().Add(time.Duration(req.Hours) * time.Hour)
		req.ExpiresAt = &t
	}

	// Generate the shared sweep id up front so it can be reported in the response.
	sweepID := aws.GenerateSweepID(req.Name)
	sweepName := swReq.SweepName
	if sweepName == "" {
		sweepName = req.Name
	}

	// Build every member name and uniqueness-check them ALL up front.
	memberNames := make([]string, count)
	for i := 0; i < count; i++ {
		memberNames[i] = fmt.Sprintf("%s-%d", req.Name, i)
	}
	if !s.checkArrayNamesUnique(memberNames, w, r) {
		return
	}

	// Count-aware budget gate + batch-aware rate limiter (shared with job arrays).
	if !s.preLaunchChecksArray(&req, count, w) {
		return
	}

	// Populate sweep fields on the base request; Manager.LaunchSweep clones per member.
	req.SweepID = sweepID
	req.SweepName = sweepName

	resp := types.LaunchSweepResponse{
		SweepID:   sweepID,
		Requested: count,
	}

	// Test mode: synthesize members without touching AWS (E2E / unit).
	if s.testMode {
		s.fanOutSweepTestMode(&req, sweepID, memberNames, swReq.ParamSets, &resp)
		s.writeJSON(w, http.StatusOK, resp)
		return
	}

	// Production mode: fan out through the AWS manager.
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		return s.fanOutSweep(r.Context(), awsManager, &req, swReq.ParamSets, sweepID, &resp)
	})

	if resp.Launched == 0 && len(resp.Errors) > 0 {
		s.writeJSON(w, http.StatusInternalServerError, resp)
		return
	}
	log.Printf("[DEBUG] handleLaunchSweep: sweep %s launched %d/%d members", sweepID, resp.Launched, count)
	s.writeJSON(w, http.StatusOK, resp)
}

// validateSweepSize bounds the number of parameter sets. Returns false (error
// already written) if out of range.
func (s *Server) validateSweepSize(count int, w http.ResponseWriter) bool {
	if count < 1 {
		s.writeError(w, http.StatusBadRequest, "param_sets must contain at least one parameter set")
		return false
	}
	if count > maxJobArraySize {
		s.writeError(w, http.StatusBadRequest,
			fmt.Sprintf("sweep size %d exceeds the maximum of %d", count, maxJobArraySize))
		return false
	}
	return true
}

// fanOutSweepTestMode synthesizes sweep members without touching AWS.
func (s *Server) fanOutSweepTestMode(req *types.LaunchRequest, sweepID string, memberNames []string, paramSets []map[string]string, resp *types.LaunchSweepResponse) {
	for i, name := range memberNames {
		inst := types.Instance{
			ID:            fmt.Sprintf("i-testsweep%d-%d", time.Now().UnixNano()%10000000000, i),
			Name:          name,
			State:         "running",
			PublicIP:      testModePublicIP,
			PrivateIP:     testModePrivateIP,
			InstanceType:  "t3.micro",
			Template:      req.Template,
			Username:      "ubuntu",
			HourlyRate:    0.0104,
			EffectiveRate: 0.0104,
			LaunchTime:    time.Now(),
			SweepID:       sweepID,
			SweepIndex:    i,
		}
		if req.ExpiresAt != nil {
			inst.ExpiresAt = req.ExpiresAt
		}
		if err := s.stateManager.SaveInstance(inst); err != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("%s: failed to save state: %v", name, err))
			continue
		}
		resp.Instances = append(resp.Instances, inst)
		resp.Launched++
	}
	_ = paramSets // params are stamped as tags on the real path; unused in test mode
}

// fanOutSweep launches the sweep through the AWS manager and persists each
// successful member. Per-member errors are aggregated into resp (partial success).
func (s *Server) fanOutSweep(ctx context.Context, awsManager *aws.Manager, req *types.LaunchRequest, paramSets []map[string]string, sweepID string, resp *types.LaunchSweepResponse) error {
	if req.SSHKeyName != "" {
		if err := s.ensureSSHKeyInAWS(awsManager, req); err != nil {
			return fmt.Errorf("failed to ensure SSH key in AWS: %w", err)
		}
	}

	launchStart := time.Now()
	instances, errs := awsManager.LaunchSweep(ctx, *req, paramSets)
	templates.GetUsageStats().RecordLaunch(req.Template, len(errs) == 0, int(time.Since(launchStart).Seconds()))

	for _, err := range errs {
		resp.Errors = append(resp.Errors, err.Error())
	}

	for _, inst := range instances {
		if inst == nil {
			continue
		}
		if refreshed := s.refreshInstanceStateFromAWS(awsManager, inst.Name); refreshed != nil {
			inst = refreshed
		}
		if req.ExpiresAt != nil {
			inst.ExpiresAt = req.ExpiresAt
		}
		s.maybeStartProgressMonitoring(inst)
		if err := s.stateManager.SaveInstance(*inst); err != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("%s: failed to save state: %v", inst.Name, err))
			continue
		}
		resp.Instances = append(resp.Instances, *inst)
		resp.Launched++

		s.securityManager.LogOperationalEvent("instance.launch", inst.Name, "", true, "",
			map[string]interface{}{
				"template":      inst.Template,
				"instance_type": inst.InstanceType,
				"instance_id":   inst.ID,
				"sweep_id":      sweepID,
			})
	}
	return nil
}
