package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/scttfrdmn/prism/pkg/aws"
	"github.com/scttfrdmn/prism/pkg/course"
	"github.com/scttfrdmn/prism/pkg/profile"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/rbac"
	"github.com/scttfrdmn/prism/pkg/templates"
	"github.com/scttfrdmn/prism/pkg/types"
)

// resolveInstanceIdentifier resolves an instance identifier (name or ID) to the instance name stored in state
// Returns the resolved instance name and true if found, empty string and false if not found
func (s *Server) resolveInstanceIdentifier(identifier string) (string, bool) {
	state, err := s.stateManager.LoadState()
	if err != nil {
		return "", false
	}

	// First try direct name lookup (most common case)
	if _, exists := state.Instances[identifier]; exists {
		return identifier, true
	}

	// If identifier looks like an instance ID (starts with "i-"), search by ID
	if strings.HasPrefix(identifier, "i-") {
		for instanceName, instance := range state.Instances {
			if instance.ID == identifier {
				return instanceName, true
			}
		}
	}

	return "", false
}

// handleInstances handles instance collection operations
func (s *Server) handleInstances(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListInstances(w, r)
	case http.MethodPost:
		s.handleLaunchInstance(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleListInstances lists all instances from local state (fast response)
// Use query parameter ?refresh=true to force refresh from AWS
func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	var instances []types.Instance
	totalCost := 0.0

	// Check if refresh from AWS is explicitly requested
	// In test mode, always serve from local state to avoid real AWS calls
	refreshFromAWS := r.URL.Query().Get("refresh") == "true" && !s.testMode

	if refreshFromAWS {
		// Query AWS for real-time status (slow but accurate)
		var awsErr error
		s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
			var err error
			instances, err = awsManager.ListInstances()
			awsErr = err
			return err
		})

		// If AWS call failed, the withAWSManager already wrote the error response
		if awsErr != nil {
			return
		}

		// Update local state with fresh AWS data.
		// When multiple instances share the same name (e.g. one running, one
		// terminated), prefer the non-terminated instance so that connect and
		// other operations resolve to the live instance.
		bestByName := make(map[string]types.Instance)
		for _, instance := range instances {
			if existing, ok := bestByName[instance.Name]; ok {
				// Keep the non-terminated instance
				if existing.State == "terminated" || existing.State == "terminating" {
					bestByName[instance.Name] = instance
				}
			} else {
				bestByName[instance.Name] = instance
			}
		}
		for _, instance := range bestByName {
			_ = s.stateManager.SaveInstance(instance)
		}
	} else {
		// Serve from local state (fast response)
		state, err := s.stateManager.LoadState()
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to load state")
			return
		}

		// Convert state map to slice
		instances = make([]types.Instance, 0, len(state.Instances))
		for _, instance := range state.Instances {
			instances = append(instances, instance)
		}
	}

	// Filter out terminated instances older than retention period (configurable)
	retentionDuration := s.config.GetRetentionDuration()
	filteredInstances := make([]types.Instance, 0)

	for _, instance := range instances {
		if instance.State != "terminated" {
			filteredInstances = append(filteredInstances, instance)
		} else if s.shouldIncludeTerminated(instance, retentionDuration) {
			filteredInstances = append(filteredInstances, instance)
		}
	}

	// In test mode, add mock instances so lifecycle/connect/mount/attach tests can find them
	if s.testMode {
		filteredInstances = append(filteredInstances, types.Instance{
			ID:           "i-testmockrunning001",
			Name:         "prism-mock-running",
			State:        "running",
			InstanceType: "t3.medium",
			Template:     "python-ml",
			Username:     "ubuntu",
		}, types.Instance{
			ID:           "i-testmockstopped001",
			Name:         "prism-mock-stopped",
			State:        "stopped",
			InstanceType: "t3.medium",
			Template:     "python-ml",
			Username:     "ubuntu",
		}, types.Instance{
			ID:           "i-testmockhibernated01",
			Name:         "prism-mock-hibernated",
			State:        "hibernated",
			InstanceType: "t3.medium",
			Template:     "python-ml",
			Username:     "ubuntu",
		})
	}

	// Calculate total cost for running instances
	for _, instance := range filteredInstances {
		if instance.State == "running" {
			// Use current spend to show actual accumulated cost
			totalCost += instance.CurrentSpend
		}
	}

	response := types.ListResponse{
		Instances: filteredInstances,
		TotalCost: totalCost,
	}

	_ = json.NewEncoder(w).Encode(response)
}

// shouldIncludeTerminated returns true if a terminated instance should still be shown
// based on the configured retention policy.
func (s *Server) shouldIncludeTerminated(instance types.Instance, retentionDuration time.Duration) bool {
	if instance.DeletionTime != nil {
		return time.Since(*instance.DeletionTime) < retentionDuration
	}
	// Legacy instances without deletion time: indefinite retention or launch-time heuristic
	if s.config.InstanceRetentionMinutes == 0 {
		return true
	}
	conservativeRetention := (5 * time.Minute) + retentionDuration
	return time.Since(instance.LaunchTime) < conservativeRetention
}

// formatRetryTime converts a duration in seconds to a human-readable string.
func formatRetryTime(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	}
	retryMin := seconds / 60
	retrySec := seconds % 60
	if retrySec > 0 {
		return fmt.Sprintf("%d minutes %d seconds", retryMin, retrySec)
	}
	return fmt.Sprintf("%d minutes", retryMin)
}

// checkLaunchPolicies enforces RBAC, policy-set constraints, and invitation policy restrictions
// before allowing a launch. Returns false and writes an error response if blocked.
func (s *Server) checkLaunchPolicies(req *types.LaunchRequest, w http.ResponseWriter) bool {
	// Determine the current user identity (profile name).
	userID := "default_user"
	if pm, err := profile.NewManagerEnhanced(); err == nil {
		if cp, err := pm.GetCurrentProfile(); err == nil {
			userID = cp.Name

			// Enforce invitation policy restrictions embedded in the profile.
			if cp.PolicyRestrictions != nil {
				violations := cp.PolicyRestrictions.GetPolicyViolations(
					req.Template, "", req.Region, 0,
				)
				if len(violations) > 0 {
					s.securityManager.LogOperationalEvent("instance.launch.policy_violation",
						req.Name, userID, false, strings.Join(violations, "; "),
						map[string]interface{}{"template": req.Template, "region": req.Region})
					s.writeError(w, http.StatusForbidden,
						fmt.Sprintf("Launch blocked by invitation policy:\n• %s",
							strings.Join(violations, "\n• ")))
					return false
				}
			}
		}
	}

	// RBAC check: does this user have permission to launch instances?
	if ok, reason := s.rbacManager.CanPerformAction(userID, rbac.ActionInstancesLaunch); !ok {
		s.securityManager.LogOperationalEvent("instance.launch.rbac_denied",
			req.Name, userID, false, reason,
			map[string]interface{}{"template": req.Template})
		s.writeError(w, http.StatusForbidden, fmt.Sprintf("Access denied: %s", reason))
		return false
	}

	// Policy-set constraints (template access + resource limits).
	if s.policyService != nil && s.policyService.IsEnabled() {
		// Count running instances for resource limit check.
		instanceCount := 0
		if st, err := s.stateManager.LoadState(); err == nil {
			instanceCount = len(st.Instances)
		}
		resp := s.policyService.CheckLaunchConstraints(req.Template, "", req.Region, 0, instanceCount)
		if !resp.Allowed {
			s.securityManager.LogOperationalEvent("instance.launch.policy_denied",
				req.Name, userID, false, resp.Reason,
				map[string]interface{}{"template": req.Template})
			msg := fmt.Sprintf("Launch blocked by policy: %s", resp.Reason)
			if len(resp.Suggestions) > 0 {
				msg += "\n\nSuggestions:\n• " + strings.Join(resp.Suggestions, "\n• ")
			}
			s.writeError(w, http.StatusForbidden, msg)
			return false
		}
	}

	return true
}

// preLaunchChecks enforces rate limits, throttling, funding, and budget constraints.
// Returns false and writes an error response if the launch should be blocked.
func (s *Server) preLaunchChecks(req *types.LaunchRequest, w http.ResponseWriter, r *http.Request) bool {
	if s.rateLimiter != nil {
		if err := s.rateLimiter.CheckAndRecordLaunch(); err != nil {
			rateLimitErr, ok := err.(*RateLimitError)
			if !ok {
				s.writeError(w, http.StatusTooManyRequests, err.Error())
				return false
			}
			status := s.rateLimiter.GetStatus()
			remaining := status.MaxLaunches - rateLimitErr.Current
			retryTime := formatRetryTime(int(rateLimitErr.RetryAfter.Seconds()))
			s.writeError(w, http.StatusTooManyRequests, fmt.Sprintf(
				"⛔ Launch rate limit exceeded\n\n"+
					"Current Usage: %d/%d launches in last %d minute(s)\n"+
					"Remaining Quota: %d launches available\n"+
					"Next Available: %s\n\n"+
					"💡 Actions:\n"+
					"  • Wait %s and try again\n"+
					"  • Check status: prism admin rate-limit status\n"+
					"  • Adjust limits: prism admin rate-limit configure --max-launches <num>\n\n"+
					"This limit prevents accidental cost overruns and AWS API throttling.",
				rateLimitErr.Current, rateLimitErr.Limit, int(rateLimitErr.Window.Minutes()),
				remaining, retryTime, retryTime))
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
	return !s.isLaunchBlockedByBudget(req, w)
}

// instanceNameCheckPassed returns false if the instance name is already taken.
// In test mode the check is skipped and the method always returns true.
func (s *Server) instanceNameCheckPassed(req *types.LaunchRequest, w http.ResponseWriter, r *http.Request) bool {
	if s.testMode {
		return true
	}
	return !s.checkInstanceNameUniqueness(req, w, r)
}

// setupSSHIfNeeded ensures an SSH key is configured in the launch request.
// Returns false and writes an error if setup fails.
func (s *Server) setupSSHIfNeeded(req *types.LaunchRequest, w http.ResponseWriter) bool {
	if req.SSHKeyName != "" || s.testMode {
		return true
	}
	if err := s.setupSSHKeyForLaunch(req); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("SSH key setup failed: %v", err))
		return false
	}
	return true
}

// maybeStartProgressMonitoring starts SSH-based progress monitoring for running instances.
// No-ops in test mode.
func (s *Server) maybeStartProgressMonitoring(instance *types.Instance) {
	if instance.State != "running" || s.testMode {
		return
	}
	sshKeyPath := os.ExpandEnv("$HOME/.ssh/id_rsa")
	s.progressTracker.StartMonitoring(instance, sshKeyPath)
}

// handleLaunchInstance launches a new instance
func (s *Server) handleLaunchInstance(w http.ResponseWriter, r *http.Request) {
	var req types.LaunchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	// Validate the launch request
	if err := s.validateLaunchRequest(&req, w); err != nil {
		return // Error response already written by validateLaunchRequest
	}

	// Security/policy checks: RBAC, policy-set constraints, and invitation restrictions.
	if !s.checkLaunchPolicies(&req, w) {
		return
	}

	// Resolve ExpiresAt from Hours if not set directly (#146)
	if req.ExpiresAt == nil && req.Hours > 0 {
		t := time.Now().Add(time.Duration(req.Hours) * time.Hour)
		req.ExpiresAt = &t
	}

	// Apply pre-launch checks: rate limits, throttling, funding, and budget constraints
	if !s.preLaunchChecks(&req, w, r) {
		return
	}

	// Project quota check (#151)
	if req.ProjectID != "" && s.projectManager != nil {
		userID := "default_user"
		if pm, err := profile.NewManagerEnhanced(); err == nil {
			if cp, err := pm.GetCurrentProfile(); err == nil {
				userID = cp.Name
			}
		}
		// Count existing instances for this user in the project
		instanceCount := 0
		if st, err := s.stateManager.LoadState(); err == nil {
			for _, inst := range st.Instances {
				if inst.ProjectID == req.ProjectID {
					instanceCount++
				}
			}
		}
		instanceType := req.Size // best effort — actual type determined at launch time
		if err := s.projectManager.CheckQuota(req.ProjectID, userID, instanceType, instanceCount); err != nil {
			s.writeError(w, http.StatusForbidden, fmt.Sprintf("Launch blocked by project quota: %v", err))
			return
		}
	}

	// Course template whitelist enforcement (#46)
	if req.CourseID != "" && s.courseManager != nil {
		if !s.courseManager.IsTemplateApproved(req.CourseID, req.Template) {
			s.writeError(w, http.StatusForbidden, fmt.Sprintf(
				"template %q is not approved for course %s. Contact your instructor to request access.",
				req.Template, req.CourseID))
			return
		}
	}

	// Course per-student budget enforcement (#163)
	if req.CourseID != "" && s.courseManager != nil {
		userID := "default_user"
		if pm, err := profile.NewManagerEnhanced(); err == nil {
			if cp, err := pm.GetCurrentProfile(); err == nil {
				userID = cp.Name
			}
		}
		estimated := estimateHourlyCostFromSize(req.Size)
		if err := s.courseManager.CheckStudentBudget(req.CourseID, userID, estimated); err != nil {
			if budgetErr, ok := err.(*course.BudgetExceededError); ok {
				s.writeError(w, http.StatusForbidden, fmt.Sprintf(
					"Launch blocked: student budget exceeded (spent $%.2f of $%.2f limit). Contact your instructor.",
					budgetErr.Spent, budgetErr.Limit))
			} else {
				s.writeError(w, http.StatusForbidden, err.Error())
			}
			return
		}
	}

	// Approval workflow (#495): explicit --request-approval flag
	if req.RequestApproval {
		if s.approvalManager == nil {
			s.writeError(w, http.StatusServiceUnavailable, "approval manager not initialized")
			return
		}
		if req.ProjectID == "" {
			s.writeError(w, http.StatusBadRequest, "project ID required when requesting approval (use --project <name>)")
			return
		}
		userID := "default_user"
		if pm, err := profile.NewManagerEnhanced(); err == nil {
			if cp, err := pm.GetCurrentProfile(); err == nil {
				userID = cp.Name
			}
		}
		details := map[string]interface{}{
			"template":   req.Template,
			"name":       req.Name,
			"size":       req.Size,
			"spot":       req.Spot,
			"est_hourly": estimateHourlyCostFromSize(req.Size),
		}
		approvalReq, err := s.approvalManager.Submit(req.ProjectID, userID, project.ApprovalTypeExpensiveInstance, details, "")
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create approval request: %v", err))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"approval_request": approvalReq,
			"message":          fmt.Sprintf("Approval request created (ID: %s). Use 'prism approval approve %s' to approve.", approvalReq.ID, approvalReq.ID),
		})
		return
	}

	// Approval workflow (#495): validate pre-approved request
	if req.ApprovalID != "" {
		if s.approvalManager == nil {
			s.writeError(w, http.StatusServiceUnavailable, "approval manager not initialized")
			return
		}
		approvalReq, err := s.approvalManager.Get(req.ApprovalID)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf("approval request not found: %v", err))
			return
		}
		if approvalReq.Status != project.ApprovalStatusApproved {
			s.writeError(w, http.StatusForbidden, fmt.Sprintf("approval request %s is not approved (status: %s)", req.ApprovalID, approvalReq.Status))
			return
		}
	}

	// Approval workflow (#495): budget-threshold gate — auto-require approval when
	// estimated hourly cost exceeds project policy threshold
	if req.ProjectID != "" && s.projectManager != nil && s.approvalManager != nil {
		if proj, err := s.projectManager.GetProject(r.Context(), req.ProjectID); err == nil &&
			proj.ApprovalPolicy != nil && proj.ApprovalPolicy.RequireApprovalAbove > 0 {

			estimatedHourly := estimateHourlyCostFromSize(req.Size)
			if estimatedHourly > proj.ApprovalPolicy.RequireApprovalAbove {
				// Check if caller is admin/owner (they bypass the gate)
				userID := "default_user"
				if pm, err := profile.NewManagerEnhanced(); err == nil {
					if cp, err := pm.GetCurrentProfile(); err == nil {
						userID = cp.Name
					}
				}
				isApprover := false
				for _, m := range proj.Members {
					if m.UserID == userID && (m.Role == "admin" || m.Role == "owner") {
						isApprover = true
						break
					}
				}
				if proj.Owner == userID {
					isApprover = true
				}

				if !isApprover && req.ApprovalID == "" {
					details := map[string]interface{}{
						"template":   req.Template,
						"name":       req.Name,
						"size":       req.Size,
						"est_hourly": estimatedHourly,
						"threshold":  proj.ApprovalPolicy.RequireApprovalAbove,
					}
					approvalReq, err := s.approvalManager.Submit(req.ProjectID, userID, project.ApprovalTypeExpensiveInstance, details, "auto-triggered by cost threshold")
					if err != nil {
						s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create approval request: %v", err))
						return
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusAccepted)
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"approval_request": approvalReq,
						"message": fmt.Sprintf("Launch blocked: estimated cost $%.2f/hr exceeds project threshold $%.2f/hr. "+
							"Approval request created (ID: %s). Use 'prism approval approve %s' to proceed.",
							estimatedHourly, proj.ApprovalPolicy.RequireApprovalAbove, approvalReq.ID, approvalReq.ID),
					})
					return
				}
			}
		}
	}

	// Check instance name uniqueness (skip in test mode)
	if !s.instanceNameCheckPassed(&req, w, r) {
		return
	}

	// Ensure SSH key is configured (skip in test mode)
	if !s.setupSSHIfNeeded(&req, w) {
		return
	}

	// Use AWS manager from request and handle launch
	var instance *types.Instance

	// In test mode, skip AWS entirely and return mock instance
	if s.testMode {
		// Return mock instance for testing — use unique ID per launch to avoid
		// React duplicate-key warnings when multiple instances are created in a test run
		instance = &types.Instance{
			ID:            fmt.Sprintf("i-testlaunch%d", time.Now().UnixNano()%10000000000),
			Name:          req.Name,
			State:         "running",
			PublicIP:      testModePublicIP,
			PrivateIP:     testModePrivateIP,
			InstanceType:  "t3.micro",
			Template:      req.Template,
			Username:      "ubuntu",
			HourlyRate:    0.0104,
			EffectiveRate: 0.0104,
			LaunchTime:    time.Now(),
		}
	} else {
		// Production mode: use AWS manager
		s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
			// Ensure SSH key exists in AWS if specified
			if req.SSHKeyName != "" {
				if err := s.ensureSSHKeyInAWS(awsManager, &req); err != nil {
					return fmt.Errorf("failed to ensure SSH key in AWS: %w", err)
				}
			}

			// Track launch start time
			launchStart := time.Now()

			// Launch instance via AWS (passing HTTP request context for timeout)
			var err error
			instance, err = awsManager.LaunchInstanceWithContext(r.Context(), req)

			// Record usage stats
			launchDuration := int(time.Since(launchStart).Seconds())
			templates.GetUsageStats().RecordLaunch(req.Template, err == nil, launchDuration)

			if err != nil {
				return err
			}

			// Immediately query AWS to get actual current state
			// This keeps our cache fresh and prevents showing stale "pending" state for hours
			refreshedInstance := s.refreshInstanceStateFromAWS(awsManager, instance.Name)
			if refreshedInstance != nil {
				instance = refreshedInstance
			}

			return nil
		})
	}

	// If instance is nil, withAWSManager already wrote an error response
	if instance == nil {
		log.Printf("[DEBUG] handleLaunchInstance: instance is nil, returning")
		return
	}

	// Apply ExpiresAt from request (#146)
	if req.ExpiresAt != nil {
		instance.ExpiresAt = req.ExpiresAt
	}

	log.Printf("[DEBUG] handleLaunchInstance: Instance created: %s (ID: %s, State: %s)", instance.Name, instance.ID, instance.State)

	// Start progress monitoring if instance is running (v0.7.2 - Issue #453)
	s.maybeStartProgressMonitoring(instance)
	log.Printf("[DEBUG] handleLaunchInstance: Progress monitoring setup done for %s", instance.Name)

	// Save state with actual current AWS state
	log.Printf("[DEBUG] handleLaunchInstance: Saving instance state for %s", instance.Name)
	if err := s.stateManager.SaveInstance(*instance); err != nil {
		log.Printf("[ERROR] handleLaunchInstance: Failed to save state: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to save instance state")
		return
	}
	log.Printf("[DEBUG] handleLaunchInstance: Instance state saved for %s", instance.Name)

	// Set ProjectID from CourseID when not already set (#172)
	if req.CourseID != "" && instance.ProjectID == "" {
		instance.ProjectID = req.CourseID
	}

	// Audit the successful launch.
	s.securityManager.LogOperationalEvent("instance.launch", instance.Name, "", true, "",
		map[string]interface{}{
			"template":      instance.Template,
			"instance_type": instance.InstanceType,
			"instance_id":   instance.ID,
			"hourly_rate":   instance.HourlyRate,
		})

	// Course audit log entry (#165)
	if req.CourseID != "" && s.courseManager != nil {
		userID := "default_user"
		if pm, err := profile.NewManagerEnhanced(); err == nil {
			if cp, err := pm.GetCurrentProfile(); err == nil {
				userID = cp.Name
			}
		}
		_ = s.courseManager.AppendCourseAudit(req.CourseID, course.AuditEntry{
			CourseID: req.CourseID,
			Actor:    userID,
			Action:   course.AuditActionInstanceLaunch,
			Target:   instance.Name,
			Detail: map[string]interface{}{
				"instance_id":   instance.ID,
				"template":      instance.Template,
				"instance_type": instance.InstanceType,
			},
		})
	}

	response := types.LaunchResponse{
		Instance:       *instance,
		Message:        fmt.Sprintf("Instance %s launched successfully", instance.Name),
		EstimatedCost:  fmt.Sprintf("$%.3f/hr (effective: $%.3f/hr)", instance.HourlyRate, instance.EffectiveRate),
		ConnectionInfo: fmt.Sprintf("ssh ubuntu@%s", instance.PublicIP),
	}

	log.Printf("[DEBUG] handleLaunchInstance: Encoding response for %s", instance.Name)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[ERROR] Failed to encode launch response: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to encode response: %v", err))
		return
	}
	log.Printf("[DEBUG] handleLaunchInstance: Response sent successfully for %s", instance.Name)
}

// handleInstanceOperations handles operations on specific instances
func (s *Server) handleInstanceOperations(w http.ResponseWriter, r *http.Request) {
	instanceName, pathParts, err := s.parseInstancePath(r.URL.Path)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.routeInstanceOperation(w, r, instanceName, pathParts)
}

func (s *Server) parseInstancePath(urlPath string) (string, []string, error) {
	path := urlPath[len("/api/v1/instances/"):]
	parts := splitPath(path)

	if len(parts) == 0 {
		return "", nil, fmt.Errorf("missing instance name")
	}

	return parts[0], parts, nil
}

func (s *Server) routeInstanceOperation(w http.ResponseWriter, r *http.Request, instanceName string, parts []string) {
	switch len(parts) {
	case 1:
		s.handleDirectInstanceOperation(w, r, instanceName)
	case 2:
		s.handleInstanceSubOperation(w, r, instanceName, parts[1])
	case 3, 4:
		switch parts[1] {
		case "idle":
			if parts[2] == "policies" {
				s.handleIdlePolicyOperation(w, r, instanceName, parts)
			} else {
				s.writeError(w, http.StatusNotFound, "Invalid path")
			}
		case "files":
			// /instances/{name}/files, /files/push, /files/pull
			s.handleInstanceFiles(w, r, instanceName)
		case "s3-mounts":
			// /instances/{name}/s3-mounts, /s3-mounts/{mountPath}
			s.handleInstanceS3Mounts(w, r, instanceName)
		default:
			s.writeError(w, http.StatusNotFound, "Invalid path")
		}
	default:
		s.writeError(w, http.StatusNotFound, "Invalid path")
	}
}

func (s *Server) handleDirectInstanceOperation(w http.ResponseWriter, r *http.Request, instanceName string) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetInstance(w, r, instanceName)
	case http.MethodDelete:
		s.handleDeleteInstance(w, r, instanceName)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleInstanceSubOperation(w http.ResponseWriter, r *http.Request, instanceName, operation string) {
	operationHandlers := map[string]func(http.ResponseWriter, *http.Request, string){
		"start":                 s.handleStartInstance,
		"stop":                  s.handleStopInstance,
		"hibernate":             s.handleHibernateInstance,
		"resume":                s.handleResumeInstance,
		"hibernation-status":    s.handleInstanceHibernationStatus,
		"extend":                s.handleExtendTTL,
		"connect":               s.handleConnectInstance,
		"exec":                  s.handleExecInstance,
		"resize":                s.handleResizeInstance,
		"idle-policies":         s.handleInstanceIdlePolicies,
		"recommend-idle-policy": s.handleInstanceRecommendIdlePolicy,
		"progress":              s.handleGetProgress,      // Launch progress monitoring (v0.7.2 - Issue #453)
		"files":                 s.handleInstanceFiles,    // SSM file ops (#30)
		"s3-mounts":             s.handleInstanceS3Mounts, // S3 mount ops (#22)
		"billed-cost":           s.handleBilledCost,       // AWS-billed cost from Cost Explorer
	}

	if handler, exists := operationHandlers[operation]; exists {
		handler(w, r, instanceName)
		return
	}

	// Special case handlers that don't take instanceName
	switch operation {
	case "layers":
		s.handleInstanceLayers(w, r)
	case "rollback":
		s.handleInstanceRollback(w, r)
	default:
		s.writeError(w, http.StatusNotFound, "Unknown operation")
	}
}

func (s *Server) handleIdlePolicyOperation(w http.ResponseWriter, r *http.Request, instanceName string, parts []string) {
	if len(parts) == 3 {
		// GET /instances/{name}/idle/policies
		s.handleInstanceIdlePolicies(w, r, instanceName)
	} else if len(parts) == 4 {
		// PUT/DELETE /instances/{name}/idle/policies/{policyId}
		policyID := parts[3]
		s.handleInstanceIdlePolicy(w, r, instanceName, policyID)
	} else {
		s.writeError(w, http.StatusNotFound, "Unknown idle operation")
	}
}

// handleGetInstance gets details of a specific instance
func (s *Server) handleGetInstance(w http.ResponseWriter, r *http.Request, identifier string) {
	// Resolve identifier (name or ID) to instance name
	instanceName, found := s.resolveInstanceIdentifier(identifier)
	if !found {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	// Get instance ID from state to query AWS
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	cachedInstance, exists := state.Instances[instanceName]
	if !exists {
		s.writeError(w, http.StatusNotFound, "Instance not found in state")
		return
	}

	// Query AWS for real-time instance data
	var liveInstance *types.Instance
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		var err error
		// If cached ID is empty/corrupt, try to find instance by Name tag
		if cachedInstance.ID == "" {
			// Query all instances and find by name
			instances, listErr := awsManager.ListInstances()
			if listErr != nil {
				return listErr
			}
			for _, inst := range instances {
				if inst.Name == instanceName {
					liveInstance = &inst
					return nil
				}
			}
			return fmt.Errorf("instance not found in AWS")
		}
		liveInstance, err = awsManager.GetInstance(cachedInstance.ID)
		return err
	})

	// If AWS query failed, withAWSManager already wrote error response
	if liveInstance == nil {
		return
	}

	// Merge cached metadata (services, username, etc.) with live AWS data
	// AWS doesn't store our custom metadata, so preserve it from cache
	liveInstance.Services = cachedInstance.Services
	if cachedInstance.Username != "" {
		liveInstance.Username = cachedInstance.Username
	}

	// Update state with latest AWS data
	if err := s.stateManager.SaveInstance(*liveInstance); err != nil {
		// Log error but don't fail - we still have the live data
		log.Printf("[WARNING] handleRefreshInstance: failed to save state for %s: %v", liveInstance.Name, err)
	}

	_ = json.NewEncoder(w).Encode(liveInstance)
}

// handleDeleteInstance deletes a specific instance
func (s *Server) handleDeleteInstance(w http.ResponseWriter, r *http.Request, identifier string) {
	// Resolve identifier (name or ID) to instance name
	instanceName, found := s.resolveInstanceIdentifier(identifier)
	if !found {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	// Mark deletion timestamp before initiating AWS deletion
	now := time.Now()
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	if instance, exists := state.Instances[instanceName]; exists {
		instance.DeletionTime = &now
		if err := s.stateManager.SaveInstance(instance); err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to update instance state")
			return
		}
	}

	// In test mode, skip AWS deletion and remove from local state
	if s.testMode {
		_ = s.stateManager.RemoveInstance(instanceName)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Dry-run instances were never launched on AWS, so just remove from local state
	if instance, exists := state.Instances[instanceName]; exists && instance.State == "dry-run" {
		_ = s.stateManager.RemoveInstance(instanceName)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Initiate AWS deletion and refresh state from AWS
	var deleteErr error
	var updatedInstance *types.Instance
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		// Delete the instance
		deleteErr = awsManager.DeleteInstance(instanceName)
		if deleteErr != nil {
			return deleteErr
		}

		// Get the cached instance to preserve metadata
		state, err := s.stateManager.LoadState()
		if err != nil {
			return err
		}

		cachedInstance, exists := state.Instances[instanceName]
		if !exists {
			return nil // Instance not in cache, nothing to update
		}

		// Query AWS immediately to get actual state (shutting-down or terminated)
		liveInstance, err := awsManager.GetInstance(cachedInstance.ID)
		if err != nil {
			// Instance might already be terminated and not found - that's OK
			// Just mark it as terminated in our state and record transition
			oldState := cachedInstance.State
			cachedInstance.State = "terminated"

			// Record state transition for cost tracking
			if oldState != "terminated" {
				transition := types.StateTransition{
					FromState: oldState,
					ToState:   "terminated",
					Timestamp: time.Now(),
					Reason:    "user_deletion",
					Initiator: "user",
				}
				cachedInstance.StateHistory = append(cachedInstance.StateHistory, transition)
			}

			updatedInstance = &cachedInstance
			return nil
		}

		// Preserve metadata from cache that AWS doesn't store
		liveInstance.Services = cachedInstance.Services
		if cachedInstance.Username != "" {
			liveInstance.Username = cachedInstance.Username
		}

		// Preserve and update state history
		liveInstance.StateHistory = cachedInstance.StateHistory

		// Record state transition if state changed
		if cachedInstance.State != liveInstance.State {
			transition := types.StateTransition{
				FromState: cachedInstance.State,
				ToState:   liveInstance.State,
				Timestamp: time.Now(),
				Reason:    "user_deletion",
				Initiator: "user",
			}
			liveInstance.StateHistory = append(liveInstance.StateHistory, transition)
		}

		updatedInstance = liveInstance
		return nil
	})

	// Only send success response if deletion succeeded
	// (withAWSManager already sent error response if it failed)
	if deleteErr == nil {
		// Update local state with real AWS state (shutting-down or terminated)
		if updatedInstance != nil {
			_ = s.stateManager.SaveInstance(*updatedInstance)
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// applyExpectedTransitionalState sets the expected transitional state for an instance
// after a lifecycle operation, before AWS has had time to propagate the change.
// This ensures the state monitor will pick up the instance for continued polling.
func (s *Server) applyExpectedTransitionalState(instanceName, expectedState string) {
	state, err := s.stateManager.LoadState()
	if err != nil {
		return
	}
	inst, exists := state.Instances[instanceName]
	if !exists {
		return
	}
	if inst.State == expectedState {
		return // Already in expected state
	}
	transition := types.StateTransition{
		FromState: inst.State,
		ToState:   expectedState,
		Timestamp: time.Now(),
		Reason:    "lifecycle_optimistic",
		Initiator: "system",
	}
	inst.State = expectedState
	inst.StateHistory = append(inst.StateHistory, transition)
	_ = s.stateManager.SaveInstance(inst)
}

// refreshInstanceStateFromAWS queries AWS and updates local state with current instance info
// This should be called after state-changing operations to keep cache fresh
// Records state transitions for accurate cost tracking
func (s *Server) refreshInstanceStateFromAWS(awsManager *aws.Manager, instanceName string) *types.Instance {
	state, err := s.stateManager.LoadState()
	if err != nil {
		return nil
	}

	cachedInstance, exists := state.Instances[instanceName]
	if !exists {
		return nil
	}

	// Query AWS for current state
	liveInstance, err := awsManager.GetInstance(cachedInstance.ID)
	if err != nil {
		// Instance might be terminated/not found - return cached version
		return &cachedInstance
	}

	// Preserve metadata that AWS doesn't store
	liveInstance.Services = cachedInstance.Services
	if cachedInstance.Username != "" {
		liveInstance.Username = cachedInstance.Username
	}
	if cachedInstance.DeletionTime != nil {
		liveInstance.DeletionTime = cachedInstance.DeletionTime
	}

	// Preserve existing state history
	liveInstance.StateHistory = cachedInstance.StateHistory

	// Record state transition if state changed, unless AWS hasn't propagated
	// the most recent lifecycle action yet. After a Start/Stop/Hibernate call,
	// AWS may briefly continue to report the prior state for ~1s while it
	// processes the request. Without this guard, the optimistic transitional
	// state written by applyExpectedTransitionalState gets immediately rolled
	// back here (e.g., pending -> stopped, stopping -> running), corrupting
	// state history and breaking cost accounting.
	if cachedInstance.State != liveInstance.State {
		if isUnpropagatedAWSState(cachedInstance.State, liveInstance.State) {
			// Keep the optimistic cached state; the state monitor will
			// record the real transition once AWS actually propagates.
			liveInstance.State = cachedInstance.State
			return liveInstance
		}
		transition := types.StateTransition{
			FromState: cachedInstance.State,
			ToState:   liveInstance.State,
			Timestamp: time.Now(),
			Reason:    "user_action", // State change triggered by user via API
			Initiator: "user",        // User-initiated state change
		}
		liveInstance.StateHistory = append(liveInstance.StateHistory, transition)
	}

	return liveInstance
}

// isUnpropagatedAWSState reports whether an AWS-reported state is the
// pre-lifecycle-action predecessor of the locally cached transitional state.
// When this returns true, the AWS response reflects a request that hasn't
// propagated through EC2's control plane yet; treating it as a state change
// would erase the user's intent and produce a phantom rollback transition.
func isUnpropagatedAWSState(cachedState, awsState string) bool {
	switch cachedState {
	case "pending":
		// Start issued; AWS may still report "stopped" briefly.
		return awsState == "stopped"
	case "stopping":
		// Stop/hibernate issued; AWS may still report "running" briefly.
		return awsState == "running"
	case "shutting-down":
		// Terminate issued; AWS may still report "running" or "stopped".
		return awsState == "running" || awsState == "stopped"
	}
	return false
}

// handleStartInstance starts a stopped instance
func (s *Server) handleStartInstance(w http.ResponseWriter, r *http.Request, identifier string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Resolve identifier (name or ID) to instance name
	instanceName, found := s.resolveInstanceIdentifier(identifier)
	if !found {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	// In test mode, skip all AWS calls for any instance
	if s.testMode {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var operationErr error
	var updatedInstance *types.Instance
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		operationErr = awsManager.StartInstance(instanceName)
		if operationErr != nil {
			return operationErr
		}

		// Set optimistic transitional state before AWS refresh (AWS may lag)
		s.applyExpectedTransitionalState(instanceName, "pending")
		// Refresh state from AWS to get actual current state (pending, running, etc.)
		updatedInstance = s.refreshInstanceStateFromAWS(awsManager, instanceName)
		return nil
	})

	if operationErr == nil {
		// Update local state with real AWS state
		if updatedInstance != nil {
			_ = s.stateManager.SaveInstance(*updatedInstance)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleStopInstance stops a running instance
func (s *Server) handleStopInstance(w http.ResponseWriter, r *http.Request, identifier string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Resolve identifier (name or ID) to instance name
	instanceName, found := s.resolveInstanceIdentifier(identifier)
	if !found {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	// In test mode, skip all AWS calls for any instance
	if s.testMode {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var operationErr error
	var updatedInstance *types.Instance
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		operationErr = awsManager.StopInstance(instanceName)
		if operationErr != nil {
			return operationErr
		}

		// Set optimistic transitional state before AWS refresh (AWS may lag)
		s.applyExpectedTransitionalState(instanceName, "stopping")
		// Refresh state from AWS to get actual current state (stopping, stopped, etc.)
		updatedInstance = s.refreshInstanceStateFromAWS(awsManager, instanceName)
		return nil
	})

	if operationErr == nil {
		// Update local state with real AWS state
		if updatedInstance != nil {
			_ = s.stateManager.SaveInstance(*updatedInstance)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleHibernateInstance hibernates a running instance
func (s *Server) handleHibernateInstance(w http.ResponseWriter, r *http.Request, identifier string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Resolve identifier (name or ID) to instance name
	instanceName, found := s.resolveInstanceIdentifier(identifier)
	if !found {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	// In test mode, skip all AWS calls for any instance
	if s.testMode {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var operationErr error
	var updatedInstance *types.Instance
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		operationErr = awsManager.HibernateInstance(instanceName)
		if operationErr != nil {
			return operationErr
		}

		// Set optimistic transitional state before AWS refresh (AWS may lag)
		s.applyExpectedTransitionalState(instanceName, "stopping")
		// Refresh state from AWS to get actual current state (stopping for hibernation)
		updatedInstance = s.refreshInstanceStateFromAWS(awsManager, instanceName)
		return nil
	})

	if operationErr == nil {
		// Update local state with real AWS state
		if updatedInstance != nil {
			_ = s.stateManager.SaveInstance(*updatedInstance)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleResumeInstance resumes a hibernated instance
func (s *Server) handleResumeInstance(w http.ResponseWriter, r *http.Request, identifier string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Resolve identifier (name or ID) to instance name
	instanceName, found := s.resolveInstanceIdentifier(identifier)
	if !found {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	// In test mode, skip all AWS calls for any instance
	if s.testMode {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var operationErr error
	var updatedInstance *types.Instance
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		operationErr = awsManager.ResumeInstance(instanceName)
		if operationErr != nil {
			return operationErr
		}

		// Set optimistic transitional state before AWS refresh (AWS may lag)
		s.applyExpectedTransitionalState(instanceName, "pending")
		// Refresh state from AWS to get actual current state (pending, running)
		updatedInstance = s.refreshInstanceStateFromAWS(awsManager, instanceName)
		return nil
	})

	if operationErr == nil {
		// Update local state with real AWS state
		if updatedInstance != nil {
			_ = s.stateManager.SaveInstance(*updatedInstance)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleInstanceHibernationStatus gets hibernation status for an instance
func (s *Server) handleInstanceHibernationStatus(w http.ResponseWriter, r *http.Request, identifier string) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Resolve identifier (name or ID) to instance name
	instanceName, found := s.resolveInstanceIdentifier(identifier)
	if !found {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	var hibernationSupported bool
	var instanceState string
	var possiblyHibernated bool

	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		var err error
		hibernationSupported, instanceState, possiblyHibernated, err = awsManager.GetInstanceHibernationStatus(instanceName)
		return err
	})

	response := map[string]interface{}{
		"hibernation_supported": hibernationSupported,
		"instance_state":        instanceState,
		"possibly_hibernated":   possiblyHibernated,
		"instance_name":         instanceName,
		"is_hibernated":         possiblyHibernated, // Deprecated field for backward compatibility
		"note":                  "possibly_hibernated is true when instance is stopped and hibernation is supported",
	}

	_ = json.NewEncoder(w).Encode(response)
}

// handleConnectInstance gets connection information for an instance
func (s *Server) handleConnectInstance(w http.ResponseWriter, r *http.Request, identifier string) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Resolve identifier (name or ID) to instance name
	instanceName, found := s.resolveInstanceIdentifier(identifier)
	if !found {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	var connectionInfo string
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		var err error
		connectionInfo, err = awsManager.GetConnectionInfo(instanceName)
		return err
	})

	if connectionInfo == "" {
		// Error was already handled by withAWSManager
		return
	}

	response := map[string]interface{}{
		"connection_info": connectionInfo,
	}

	// Augment response with DCV-specific fields for desktop instances
	if st, err := s.stateManager.LoadState(); err == nil {
		if inst, ok := st.Instances[instanceName]; ok && inst.ConnectionType == "desktop" {
			dcvPort := inst.WebPort
			if dcvPort == 0 {
				dcvPort = 8443
			}
			response["connection_type"] = "desktop"
			response["dcv_port"] = dcvPort
			response["dcv_username"] = inst.Username
			// Note: dcv_password is intentionally omitted from API — read from local state
		}
	}

	_ = json.NewEncoder(w).Encode(response)
}

// handleExecInstance executes a command on an instance via SSM
func (s *Server) handleExecInstance(w http.ResponseWriter, r *http.Request, identifier string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Resolve identifier (name or ID) to instance name
	instanceName, found := s.resolveInstanceIdentifier(identifier)
	if !found {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	// Parse request body
	var execRequest types.ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&execRequest); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Validate command
	if execRequest.Command == "" {
		s.writeError(w, http.StatusBadRequest, "Command is required")
		return
	}

	// Execute command via AWS manager
	var execResult *types.ExecResult
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		var err error
		execResult, err = awsManager.ExecuteCommand(instanceName, execRequest)
		return err
	})

	if execResult == nil {
		// Error was already handled by withAWSManager
		return
	}

	// Return execution result
	_ = json.NewEncoder(w).Encode(execResult)
}

// handleResizeInstance handles the resize instance operation
func (s *Server) handleResizeInstance(w http.ResponseWriter, r *http.Request, identifier string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	// Resolve identifier (name or ID) to instance name
	instanceName, found := s.resolveInstanceIdentifier(identifier)
	if !found {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	// Parse request body
	var resizeRequest types.ResizeRequest
	if err := json.NewDecoder(r.Body).Decode(&resizeRequest); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Validate resize request
	if resizeRequest.TargetInstanceType == "" {
		s.writeError(w, http.StatusBadRequest, "Target instance type is required")
		return
	}

	// Set instance name from URL (in case it wasn't in the request body)
	resizeRequest.InstanceName = instanceName

	// Execute resize via AWS manager
	var resizeResponse *types.ResizeResponse
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		var err error
		resizeResponse, err = awsManager.ResizeInstance(resizeRequest)
		return err
	})

	if resizeResponse == nil {
		// Error was already handled by withAWSManager
		return
	}

	// Return resize result
	_ = json.NewEncoder(w).Encode(resizeResponse)
}

// setupSSHKeyForLaunch sets up SSH key configuration for a launch request
func (s *Server) setupSSHKeyForLaunch(req *types.LaunchRequest) error {
	// Get current profile (this would be extracted from request context in production)
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		return fmt.Errorf("failed to create profile manager: %w", err)
	}

	currentProfile, err := profileManager.GetCurrentProfile()
	if err != nil {
		return fmt.Errorf("failed to get current profile: %w", err)
	}

	// Create SSH key manager
	sshKeyManager, err := profile.NewSSHKeyManager()
	if err != nil {
		return fmt.Errorf("failed to create SSH key manager: %w", err)
	}

	// Get SSH key configuration for current profile
	_, keyName, err := sshKeyManager.GetSSHKeyForProfile(currentProfile)
	if err != nil {
		return fmt.Errorf("failed to get SSH key for profile: %w", err)
	}

	// Set SSH key in launch request
	req.SSHKeyName = keyName

	return nil
}

// ensureSSHKeyInAWS ensures the SSH key exists in AWS
func (s *Server) ensureSSHKeyInAWS(awsManager *aws.Manager, _ *types.LaunchRequest) error {
	// Get current profile
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		return fmt.Errorf("failed to create profile manager: %w", err)
	}

	currentProfile, err := profileManager.GetCurrentProfile()
	if err != nil {
		return fmt.Errorf("failed to get current profile: %w", err)
	}

	// Create SSH key manager
	sshKeyManager, err := profile.NewSSHKeyManager()
	if err != nil {
		return fmt.Errorf("failed to create SSH key manager: %w", err)
	}

	// Get SSH key configuration
	keyPath, keyName, err := sshKeyManager.GetSSHKeyForProfile(currentProfile)
	if err != nil {
		return fmt.Errorf("failed to get SSH key for profile: %w", err)
	}

	// Get public key content
	publicKeyPath := keyPath + ".pub"
	publicKeyContent, err := sshKeyManager.GetPublicKeyContent(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to get public key content: %w", err)
	}

	// Ensure key exists in AWS
	if err := awsManager.EnsureKeyPairExists(keyName, publicKeyContent); err != nil {
		return fmt.Errorf("failed to ensure key pair exists in AWS: %w", err)
	}

	return nil
}

// validateLaunchRequest validates the launch request and writes error response if needed
// Returns nil if validation passes, error if validation fails (response already written)
func (s *Server) validateLaunchRequest(req *types.LaunchRequest, w http.ResponseWriter) error {
	// Validate required fields
	if req.Template == "" {
		s.writeError(w, http.StatusBadRequest, "Missing required field: template")
		return fmt.Errorf("missing template")
	}

	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "Missing required field: name")
		return fmt.Errorf("missing name")
	}

	// Validate instance size if provided
	if req.Size != "" {
		if err := s.validateInstanceSize(req.Size, w); err != nil {
			return err
		}
	}

	// Validate package manager if provided
	if req.PackageManager != "" {
		if err := s.validatePackageManager(req.PackageManager, w); err != nil {
			return err
		}
	}

	return nil
}

// validateInstanceSize validates the instance size parameter
func (s *Server) validateInstanceSize(size string, w http.ResponseWriter) error {
	validSizes := []string{"XS", "S", "M", "L", "XL", "GPU-S", "GPU-M", "GPU-L", "GPU-XL"}
	for _, validSize := range validSizes {
		if size == validSize {
			return nil
		}
	}

	s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid size '%s'. Valid sizes: %v", size, validSizes))
	return fmt.Errorf("invalid size")
}

// validatePackageManager validates the package manager parameter
func (s *Server) validatePackageManager(packageManager string, w http.ResponseWriter) error {
	validPackageManagers := []string{"apt", "yum", "dnf", "conda", "brew"}
	for _, valid := range validPackageManagers {
		if packageManager == valid {
			return nil
		}
	}

	s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid package manager '%s'. Valid package managers: %v", packageManager, validPackageManagers))
	return fmt.Errorf("invalid package manager")
}

// checkInstanceNameUniqueness checks if the instance name is already taken.
// Returns true if name is in use (response already written), false if available.
// Checks local state first (fast path), then falls back to live AWS (safety net
// for cases where local state was wiped while instances are still running).
func (s *Server) checkInstanceNameUniqueness(req *types.LaunchRequest, w http.ResponseWriter, r *http.Request) bool {
	isActive := func(state string) bool {
		return state != "terminated" && state != "terminating"
	}
	conflictMsg := func(id, state string) string {
		return fmt.Sprintf(
			"Instance named %q already exists (state: %s, id: %s). "+
				"Use a different name, or terminate the existing instance first.",
			req.Name, state, id)
	}

	// Fast path: local state (no AWS call needed)
	if st, err := s.stateManager.LoadState(); err == nil {
		if existing, ok := st.Instances[req.Name]; ok && isActive(existing.State) {
			s.writeError(w, http.StatusConflict, conflictMsg(existing.ID, existing.State))
			return true
		}
	}

	// Slow path: live AWS check — handles stale/missing local state.
	// Skipped in test mode to avoid real AWS calls.
	if !s.testMode {
		var conflictID, conflictState string
		s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
			instances, err := awsManager.ListInstances()
			if err != nil {
				return fmt.Errorf("failed to check existing instances: %w", err)
			}
			for _, inst := range instances {
				if inst.Name == req.Name && isActive(inst.State) {
					conflictID = inst.ID
					conflictState = inst.State
					break
				}
			}
			return nil
		})
		if conflictID != "" {
			s.writeError(w, http.StatusConflict, conflictMsg(conflictID, conflictState))
			return true
		}
	}
	return false
}

// isLaunchBlockedByBudget checks if the launch is blocked by budget hard cap
// Returns true if launch is blocked (error already written), false if allowed
func (s *Server) isLaunchBlockedByBudget(req *types.LaunchRequest, w http.ResponseWriter) bool {
	return s.isLaunchBlockedByBudgetN(req, 1, w)
}

// isLaunchBlockedByBudgetN is the count-aware budget gate: it estimates the cost
// of launching `count` identical instances (a job array) and blocks if the batch
// would breach the project's monthly limit. count=1 is the single-launch path.
// Returns true if launch is blocked (error already written), false if allowed.
func (s *Server) isLaunchBlockedByBudgetN(req *types.LaunchRequest, count int, w http.ResponseWriter) bool {
	if count < 1 {
		count = 1
	}
	// If no project is associated, budget cap doesn't apply
	if req.ProjectID == "" {
		return false
	}

	ctx := context.Background()

	// First check if launches are manually prevented for this project
	launchPrevented, err := s.projectManager.IsLaunchPrevented(ctx, req.ProjectID)
	if err != nil {
		// Log the error but don't block the launch (fail open for safety)
		log.Printf("Warning: Failed to check budget hard cap for project %s: %v", req.ProjectID, err)
		return false
	}

	// If launch is manually prevented, block it
	if launchPrevented {
		budgetStatus, err := s.budgetStatus(ctx, req.ProjectID)
		if err != nil {
			// Fallback error message if we can't get budget details
			s.writeError(w, http.StatusForbidden,
				fmt.Sprintf("Instance launch blocked: Project '%s' has reached its budget hard cap. Contact project owner to increase budget or clear hard cap.", req.ProjectID))
			return true
		}

		// Build detailed error message with budget information
		errorMsg := fmt.Sprintf("Instance launch blocked: Project '%s' budget hard cap reached.\n\n", req.ProjectID)
		errorMsg += "Budget Status:\n"
		errorMsg += fmt.Sprintf("  Total Budget: $%.2f\n", budgetStatus.TotalBudget)
		errorMsg += fmt.Sprintf("  Spent: $%.2f (%.1f%%)\n", budgetStatus.SpentAmount, budgetStatus.SpentPercentage*100)
		errorMsg += fmt.Sprintf("  Remaining: $%.2f\n", budgetStatus.RemainingBudget)

		if len(budgetStatus.TriggeredActions) > 0 {
			errorMsg += "\nTriggered Actions:\n"
			for _, action := range budgetStatus.TriggeredActions {
				errorMsg += fmt.Sprintf("  - %s\n", action)
			}
		}

		errorMsg += "\nTo continue launching instances:\n"
		errorMsg += "  1. Contact project owner to increase the budget\n"
		errorMsg += "  2. Stop or hibernate running instances to reduce costs\n"
		errorMsg += fmt.Sprintf("  3. Clear the hard cap temporarily with: prism project allow-launches %s\n", req.ProjectID)

		s.writeError(w, http.StatusForbidden, errorMsg)
		return true
	}

	// NEW: Proactive budget enforcement - check if launch would exceed monthly budget limit
	proj, err := s.projectManager.GetProject(ctx, req.ProjectID)
	if err != nil {
		log.Printf("Warning: Failed to get project for budget enforcement: %v", err)
		return false // Fail open
	}

	// Only enforce if project has a budget with monthly limit
	if proj.Budget == nil || proj.Budget.MonthlyLimit == nil {
		return false // No monthly limit configured
	}

	// Determine instance type that will be launched
	instanceType := s.getInstanceTypeForLaunch(req)

	// Estimate monthly cost for this launch (using cost calculator from project package).
	// For a job array (count > 1) the estimate is the per-member cost × count, so the
	// gate reflects the whole batch — an array can't slip past the monthly limit.
	var costCalc project.CostCalculator
	estimatedMonthlyCost := costCalc.EstimateMonthlyCost(instanceType, 20) * float64(count) // 20GB default root volume

	// Get current spending
	currentSpend := proj.Budget.SpentAmount
	monthlyLimit := *proj.Budget.MonthlyLimit

	// Budget engine adoption Phase 1 (#643): the projected>limit decision is now made by the
	// standalone budgetengine via engine.CheckLaunch, fed a single-source view of this monthly
	// budget. Block/allow behavior is a faithful parity of the previous flat-limit comparison; the
	// engine gains multi-source/banking semantics in later phases. Fail open on engine error.
	decision, decErr := s.checkMonthlyLimitViaEngine(req.ProjectID, proj.Budget, estimatedMonthlyCost)
	if decErr != nil {
		log.Printf("Warning: budget engine check failed for project %s, allowing launch: %v", req.ProjectID, decErr)
		return false // Fail open
	}
	projectedSpend := decision.Projected
	if !decision.Allowed() {
		// Block the launch - would exceed budget
		errorMsg := fmt.Sprintf("⛔ Instance launch blocked: Would exceed monthly budget limit\n\n")
		errorMsg += "Budget Analysis:\n"
		errorMsg += fmt.Sprintf("  Monthly Limit: $%.2f\n", monthlyLimit)
		errorMsg += fmt.Sprintf("  Current Spend: $%.2f (%.1f%% of limit)\n", currentSpend, (currentSpend/monthlyLimit)*100)
		errorMsg += fmt.Sprintf("  Instance Cost: $%.2f/month (%s)\n", estimatedMonthlyCost, instanceType)
		errorMsg += fmt.Sprintf("  Projected Total: $%.2f (would exceed limit by $%.2f)\n", projectedSpend, projectedSpend-monthlyLimit)
		errorMsg += "\n💡 Options:\n"
		errorMsg += "  • Use a smaller instance size (try --size XS or S)\n"
		errorMsg += fmt.Sprintf("  • Increase monthly budget: prism project update-budget %s --monthly-limit %.2f\n", req.ProjectID, projectedSpend+10)
		errorMsg += "  • Stop or hibernate running instances to free up budget\n"
		errorMsg += "  • Wait for budget period to reset\n"

		s.writeError(w, http.StatusForbidden, errorMsg)
		return true
	}

	return false
}

// getInstanceTypeForLaunch determines the instance type that will be used for a launch request
// This must match the logic in pkg/aws/manager.go:getInstanceTypeForSize to ensure
// budget enforcement estimates match actual instance types that will be launched
func (s *Server) getInstanceTypeForLaunch(req *types.LaunchRequest) string {
	// If size is specified, map it to instance type
	// Use the same mapping as AWS manager (t3 instances, not t4g)
	if req.Size != "" {
		// Map sizes to instance types (must match pkg/aws/manager.go:getInstanceTypeForSize)
		sizeMap := map[string]string{
			"XS": "t3.micro",  // 1 vCPU, 2GB RAM
			"S":  "t3.small",  // 2 vCPU, 4GB RAM
			"M":  "t3.medium", // 2 vCPU, 8GB RAM
			"L":  "t3.large",  // 4 vCPU, 16GB RAM
			"XL": "t3.xlarge", // 8 vCPU, 32GB RAM
		}

		if instanceType, exists := sizeMap[req.Size]; exists {
			return instanceType
		}
	}

	// Try to get from template defaults
	template, err := templates.GetTemplateInfo(req.Template)
	if err == nil && template.InstanceDefaults.Type != "" {
		return template.InstanceDefaults.Type
	}

	// Fallback to default
	return "t3.micro"
}

// resolveFundingAllocation resolves the funding allocation for a launch request (v0.5.10+)
// Priority: 1) Explicit --funding flag, 2) Project's default allocation, 3) Error if neither
func (s *Server) resolveFundingAllocation(req *types.LaunchRequest, w http.ResponseWriter) error {
	ctx := context.Background()

	// If funding allocation is already specified, validate it
	if req.FundingAllocationID != "" {
		// Verify allocation exists and belongs to this project
		allocation, err := s.budgetManager.GetAllocation(ctx, req.FundingAllocationID)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid funding allocation: %v", err))
			return fmt.Errorf("invalid funding allocation")
		}

		// Verify allocation belongs to the project
		if allocation.ProjectID != req.ProjectID {
			s.writeError(w, http.StatusBadRequest,
				fmt.Sprintf("Funding allocation %q does not belong to project %q", req.FundingAllocationID, req.ProjectID))
			return fmt.Errorf("allocation project mismatch")
		}

		// Allocation is valid, continue
		return nil
	}

	// No explicit funding specified - use project's default allocation
	project, err := s.projectManager.GetProject(ctx, req.ProjectID)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to get project: %v", err))
		return fmt.Errorf("project not found")
	}

	// Check if project has a default allocation
	if project.DefaultAllocationID == nil || *project.DefaultAllocationID == "" {
		// No default allocation - check if project has any allocations
		allocations, err := s.budgetManager.GetProjectAllocations(ctx, req.ProjectID)
		if err != nil || len(allocations) == 0 {
			// No allocations found - check if project has a simple budget configured
			// If so, allow launch without funding allocation (simpler budget enforcement)
			if project.Budget != nil {
				log.Printf("Project %s has budget but no funding allocations - allowing launch with simple budget enforcement", req.ProjectID)
				return nil // Allow launch, budget enforcement will check limits
			}

			// No budget and no allocations - require funding allocation setup
			s.writeError(w, http.StatusBadRequest,
				fmt.Sprintf("Project %q has no budget allocations. Please:\n"+
					"  1. Create a budget: prism budget create <name> <amount>\n"+
					"  2. Allocate to project: prism budget allocate <budget-name> --project %s --amount <amount>\n"+
					"  3. Set as default: prism project set-default-funding %s <allocation-id>\n"+
					"Or specify funding explicitly: --funding <allocation-id>",
					req.ProjectID, req.ProjectID, req.ProjectID))
			return fmt.Errorf("no project allocations")
		}

		// Project has allocations but no default - require explicit selection
		allocationNames := []string{}
		for _, alloc := range allocations {
			if budget, err := s.budgetManager.GetBudget(ctx, alloc.BudgetID); err == nil {
				allocationNames = append(allocationNames, fmt.Sprintf("%s (ID: %s, $%.2f allocated)",
					budget.Name, alloc.ID, alloc.AllocatedAmount))
			}
		}

		s.writeError(w, http.StatusBadRequest,
			fmt.Sprintf("Project %q has multiple funding sources but no default set.\n\n"+
				"Available allocations:\n  %s\n\n"+
				"Please either:\n"+
				"  1. Set default: prism project set-default-funding %s <allocation-id>\n"+
				"  2. Specify funding: --funding <allocation-id>",
				req.ProjectID,
				strings.Join(allocationNames, "\n  "),
				req.ProjectID))
		return fmt.Errorf("no default allocation")
	}

	// Use project's default allocation
	defaultAllocationID := *project.DefaultAllocationID

	// Verify default allocation still exists
	allocation, err := s.budgetManager.GetAllocation(ctx, defaultAllocationID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError,
			fmt.Sprintf("Project's default allocation %q not found. "+
				"Please update default allocation or specify --funding explicitly", defaultAllocationID))
		return fmt.Errorf("default allocation not found")
	}

	// Verify allocation belongs to this project (defensive check)
	if allocation.ProjectID != req.ProjectID {
		s.writeError(w, http.StatusInternalServerError,
			fmt.Sprintf("Project's default allocation %q does not belong to this project (data integrity issue)",
				defaultAllocationID))
		return fmt.Errorf("allocation integrity error")
	}

	// Set the resolved allocation ID in the request
	req.FundingAllocationID = defaultAllocationID

	return nil
}

// estimateHourlyCostFromSize returns a conservative hourly cost estimate for a given
// instance size label. Used for per-student budget enforcement at launch time (#163).
func estimateHourlyCostFromSize(size string) float64 {
	switch strings.ToUpper(size) {
	case "XS":
		return 0.02
	case "S":
		return 0.05
	case "M":
		return 0.10
	case "L":
		return 0.20
	case "XL":
		return 0.40
	case "GPU-S", "GPUS":
		return 0.50
	case "GPU-L", "GPUL":
		return 1.20
	default:
		return 0.10
	}
}

// handleExtendTTL extends the time-to-live for an instance (#588).
func (s *Server) handleExtendTTL(w http.ResponseWriter, r *http.Request, identifier string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	instanceName, found := s.resolveInstanceIdentifier(identifier)
	if !found {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	// Parse extension hours from body (default 4)
	var body struct {
		Hours int `json:"hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Hours <= 0 {
		body.Hours = 4
	}

	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	inst, exists := state.Instances[instanceName]
	if !exists {
		s.writeError(w, http.StatusNotFound, "Instance not found in state")
		return
	}

	// Extend ExpiresAt
	extension := time.Duration(body.Hours) * time.Hour
	if inst.ExpiresAt != nil {
		newExpiry := inst.ExpiresAt.Add(extension)
		inst.ExpiresAt = &newExpiry
	} else {
		newExpiry := time.Now().Add(extension)
		inst.ExpiresAt = &newExpiry
	}

	if err := s.stateManager.SaveInstance(inst); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to save state")
		return
	}

	// Update EC2 tag so spored picks up the new TTL on its next config reload
	if !s.testMode {
		s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
			return awsManager.TagInstance(inst.ID, "prism:ttl", inst.ExpiresAt.Sub(inst.LaunchTime).String())
		})
	}

	w.WriteHeader(http.StatusNoContent)
}
