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

// maxJobArraySize caps a single job-array request. This is an accidental-runaway
// guard, not a capacity limit — the count-aware budget gate is the real cost
// control. Kept generous enough for realistic MPI cohorts / parameter sweeps.
const maxJobArraySize = 64

// Synthetic addresses used for test-mode instances (RFC 5737 TEST-NET-3 public IP
// + RFC 1918 private IP), so tests never surface a plausibly-real address.
const (
	testModePublicIP  = "203.0.113.1"
	testModePrivateIP = "10.0.1.100"
)

// handleLaunchArray launches a job array: Count homogeneous instances sharing a
// generated job-array id, named <base>-0..<base>-(Count-1). spored discovers the
// members via the prism:job-array-* tags Manager.BuildTags stamps and writes
// /etc/spawn/job-array-peers.json for MPI / tightly-coupled workloads.
//
// Members are independent workspaces: some may launch while others fail (partial
// success). Successes are NOT torn down on a partial failure — unlike spawn's MPI
// all-or-nothing cohort semantics, which are out of scope for this increment.
func (s *Server) handleLaunchArray(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var arrReq types.LaunchArrayRequest
	if err := json.NewDecoder(r.Body).Decode(&arrReq); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	count := arrReq.Count
	if !s.validateArrayCount(count, w) {
		return
	}

	req := arrReq.LaunchRequest

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

	// Generate the shared array id up front so it can be reported in the response
	// and stamped identically on every member.
	arrayID := aws.GenerateJobArrayID(req.Name)
	jobArrayName := arrReq.JobArrayName
	if jobArrayName == "" {
		jobArrayName = req.Name
	}

	// Build every member name and uniqueness-check them ALL up front, so we don't
	// half-launch an array and then 409 mid-way.
	memberNames := make([]string, count)
	for i := 0; i < count; i++ {
		memberNames[i] = fmt.Sprintf("%s-%d", req.Name, i)
	}
	if !s.checkArrayNamesUnique(memberNames, w, r) {
		return
	}

	// Pre-launch checks made count-aware: batch rate limiter + count-aware budget
	// gate + throttling + funding. An array must not bypass per-instance cost controls.
	if !s.preLaunchChecksArray(&req, count, w) {
		return
	}

	// Populate array fields on the base request; Manager.LaunchArray clones per member.
	req.JobArrayID = arrayID
	req.JobArrayName = jobArrayName

	resp := types.LaunchArrayResponse{
		JobArrayID: arrayID,
		Requested:  count,
	}

	// Test mode: synthesize members without touching AWS (E2E / unit).
	if s.testMode {
		s.fanOutArrayTestMode(&req, arrayID, memberNames, &resp)
		s.writeJSON(w, http.StatusOK, resp)
		return
	}

	// Production mode: fan out through the AWS manager.
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		return s.fanOutArray(r.Context(), awsManager, &req, count, arrayID, &resp)
	})

	// If nothing launched and every member errored, surface as an error status;
	// otherwise report partial/full success with per-member errors in the body.
	if resp.Launched == 0 && len(resp.Errors) > 0 {
		s.writeJSON(w, http.StatusInternalServerError, resp)
		return
	}
	log.Printf("[DEBUG] handleLaunchArray: array %s launched %d/%d members", arrayID, resp.Launched, count)
	s.writeJSON(w, http.StatusOK, resp)
}

// validateArrayCount bounds the requested member count. Returns false (error
// already written) if out of range.
func (s *Server) validateArrayCount(count int, w http.ResponseWriter) bool {
	if count < 1 {
		s.writeError(w, http.StatusBadRequest, "count must be >= 1")
		return false
	}
	if count > maxJobArraySize {
		s.writeError(w, http.StatusBadRequest,
			fmt.Sprintf("count %d exceeds the maximum job-array size of %d", count, maxJobArraySize))
		return false
	}
	return true
}

// fanOutArrayTestMode synthesizes array members without touching AWS.
func (s *Server) fanOutArrayTestMode(req *types.LaunchRequest, arrayID string, memberNames []string, resp *types.LaunchArrayResponse) {
	for i, name := range memberNames {
		inst := types.Instance{
			ID:            fmt.Sprintf("i-testarray%d-%d", time.Now().UnixNano()%10000000000, i),
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
			JobArrayID:    arrayID,
			JobArrayIndex: i,
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
}

// fanOutArray launches the array through the AWS manager and persists each
// successful member. Per-member errors are aggregated into resp (partial success).
func (s *Server) fanOutArray(ctx context.Context, awsManager *aws.Manager, req *types.LaunchRequest, count int, arrayID string, resp *types.LaunchArrayResponse) error {
	if req.SSHKeyName != "" {
		if err := s.ensureSSHKeyInAWS(awsManager, req); err != nil {
			return fmt.Errorf("failed to ensure SSH key in AWS: %w", err)
		}
	}

	launchStart := time.Now()
	instances, errs := awsManager.LaunchArray(ctx, *req, count)
	templates.GetUsageStats().RecordLaunch(req.Template, len(errs) == 0, int(time.Since(launchStart).Seconds()))

	for _, err := range errs {
		resp.Errors = append(resp.Errors, err.Error())
	}

	for _, inst := range instances {
		if inst == nil {
			continue
		}
		// Refresh from AWS so we persist real state rather than "pending".
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
				"job_array_id":  arrayID,
			})
	}
	return nil
}

// checkArrayNamesUnique verifies that none of the member names is already taken.
// It loads local state once and (outside test mode) lists AWS once, checking all
// names against both — far cheaper than N calls to checkInstanceNameUniqueness.
// Returns true if all names are free; false if any collides (error already written).
func (s *Server) checkArrayNamesUnique(names []string, w http.ResponseWriter, r *http.Request) bool {
	taken := s.activeInstanceNames(w, r)
	for _, name := range names {
		if detail, ok := taken[name]; ok {
			s.writeError(w, http.StatusConflict, fmt.Sprintf(
				"Instance named %q already exists (%s). "+
					"Use a different array base name, or terminate the existing instance first.",
				name, detail))
			return false
		}
	}
	return true
}

// activeInstanceNames returns a map of active (non-terminated) instance name ->
// "state, id: <id>", from local state and (outside test mode) a single live AWS
// list. Used to uniqueness-check all array member names in one pass.
func (s *Server) activeInstanceNames(w http.ResponseWriter, r *http.Request) map[string]string {
	isActive := func(state string) bool {
		return state != "terminated" && state != "terminating"
	}
	record := func(taken map[string]string, name, state, id string) {
		if isActive(state) {
			taken[name] = fmt.Sprintf("%s, id: %s", state, id)
		}
	}

	taken := make(map[string]string)
	if st, err := s.stateManager.LoadState(); err == nil {
		for _, inst := range st.Instances {
			record(taken, inst.Name, inst.State, inst.ID)
		}
	}

	if !s.testMode {
		s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
			instances, err := awsManager.ListInstances()
			if err != nil {
				return fmt.Errorf("failed to check existing instances: %w", err)
			}
			for _, inst := range instances {
				record(taken, inst.Name, inst.State, inst.ID)
			}
			return nil
		})
	}
	return taken
}

// preLaunchChecksArray is the count-aware analogue of preLaunchChecks: it admits
// the batch through the rate limiter as a unit, then runs throttling, funding, and
// the count-aware budget gate. Returns true if the array may proceed.
func (s *Server) preLaunchChecksArray(req *types.LaunchRequest, count int, w http.ResponseWriter) bool {
	if s.rateLimiter != nil {
		if err := s.rateLimiter.CheckAndRecordLaunches(count); err != nil {
			s.writeError(w, http.StatusTooManyRequests, err.Error())
			return false
		}
	}
	if !s.checkLaunchThrottling(req, w) {
		return false
	}
	if req.ProjectID != "" {
		if err := s.resolveFundingAllocation(req, w); err != nil {
			return false
		}
	}
	return !s.isLaunchBlockedByBudgetN(req, count, w)
}
