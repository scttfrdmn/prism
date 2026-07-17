// Package client provides the Prism API client interface and implementation.
package client

import (
	"context"
	"time"

	"github.com/scttfrdmn/prism/pkg/idle"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/storage"
	"github.com/scttfrdmn/prism/pkg/templates"
	"github.com/scttfrdmn/prism/pkg/throttle"
	"github.com/scttfrdmn/prism/pkg/types"
)

// Options represents configuration options for the API client
type Options struct {
	AWSProfile      string
	AWSRegion       string
	InvitationToken string
	OwnerAccount    string
	S3ConfigPath    string
	APIKey          string // API key for daemon authentication
}

// PrismAPI defines the interface for interacting with the Prism API
type PrismAPI interface {
	// Configuration
	SetOptions(Options)

	// Instance operations
	LaunchInstance(context.Context, types.LaunchRequest) (*types.LaunchResponse, error)
	LaunchArray(context.Context, types.LaunchArrayRequest) (*types.LaunchArrayResponse, error)
	LaunchSweep(context.Context, types.LaunchSweepRequest) (*types.LaunchSweepResponse, error)
	ListInstances(context.Context) (*types.ListResponse, error)
	ListInstancesWithRefresh(context.Context, bool) (*types.ListResponse, error)
	GetInstance(context.Context, string) (*types.Instance, error)
	GetProgress(context.Context, string) (*types.ProgressResponse, error)   // Launch progress (v0.7.2)
	GetBilledCost(context.Context, string) (*types.BilledCostResult, error) // AWS-billed cost from Cost Explorer
	StartInstance(context.Context, string) error
	StopInstance(context.Context, string) error
	HibernateInstance(context.Context, string) error
	ResumeInstance(context.Context, string) error
	GetInstanceHibernationStatus(context.Context, string) (*types.HibernationStatus, error)
	DeleteInstance(context.Context, string) error
	ConnectInstance(context.Context, string) (string, error)
	ExecInstance(context.Context, string, types.ExecRequest) (*types.ExecResult, error)
	ResizeInstance(context.Context, types.ResizeRequest) (*types.ResizeResponse, error)

	// Tunnel operations
	CreateTunnels(context.Context, string, []string) (*CreateTunnelsResponse, error)
	ListTunnels(context.Context, string) (*ListTunnelsResponse, error)
	CloseTunnel(context.Context, string, string) error
	CloseInstanceTunnels(context.Context, string) error

	// Log operations
	GetInstanceLogs(context.Context, string, types.LogRequest) (*types.LogResponse, error)
	GetInstanceLogTypes(context.Context, string) (*types.LogTypesResponse, error)
	GetLogsSummary(context.Context) (*types.LogSummaryResponse, error)

	// Template operations
	ListTemplates(context.Context) (map[string]types.Template, error)
	GetTemplate(context.Context, string) (*types.Template, error)

	// Template application operations
	ApplyTemplate(context.Context, templates.ApplyRequest) (*templates.ApplyResponse, error)
	DiffTemplate(context.Context, templates.DiffRequest) (*templates.TemplateDiff, error)
	GetInstanceLayers(context.Context, string) ([]templates.AppliedTemplate, error)
	RollbackInstance(context.Context, types.RollbackRequest) error

	// Idle detection operations (new system)
	GetIdlePendingActions(context.Context) ([]types.IdleState, error)
	ExecuteIdleActions(context.Context) (*types.IdleExecutionResponse, error)
	GetIdleHistory(context.Context) ([]types.IdleHistoryEntry, error)

	// Volume operations (EFS - shared storage)
	CreateVolume(context.Context, types.VolumeCreateRequest) (*types.StorageVolume, error)
	ListVolumes(context.Context) ([]*types.StorageVolume, error)
	GetVolume(context.Context, string) (*types.StorageVolume, error)
	DeleteVolume(context.Context, string) error
	AttachVolume(context.Context, string, string) error
	DetachVolume(context.Context, string) error
	MountVolume(context.Context, string, string, string) error
	UnmountVolume(context.Context, string, string) error

	// Storage operations (EBS - local storage)
	CreateStorage(context.Context, types.StorageCreateRequest) (*types.StorageVolume, error)
	ListStorage(context.Context) ([]*types.StorageVolume, error)
	GetStorage(context.Context, string) (*types.StorageVolume, error)
	DeleteStorage(context.Context, string) error
	AttachStorage(context.Context, string, string) error
	DetachStorage(context.Context, string) error

	// Project management operations
	CreateProject(context.Context, project.CreateProjectRequest) (*types.Project, error)
	ListProjects(context.Context, *project.ProjectFilter) (*project.ProjectListResponse, error)
	GetProject(context.Context, string) (*types.Project, error)
	UpdateProject(context.Context, string, project.UpdateProjectRequest) (*types.Project, error)
	DeleteProject(context.Context, string) error
	AddProjectMember(context.Context, string, project.AddMemberRequest) error
	UpdateProjectMember(context.Context, string, string, project.UpdateMemberRequest) error
	RemoveProjectMember(context.Context, string, string) error
	GetProjectMembers(context.Context, string) ([]types.ProjectMember, error)
	GetProjectBudgetStatus(context.Context, string) (*project.BudgetStatus, error)
	SetProjectBudget(context.Context, string, SetProjectBudgetRequest) (map[string]interface{}, error)
	UpdateProjectBudget(context.Context, string, UpdateProjectBudgetRequest) (map[string]interface{}, error)
	DisableProjectBudget(context.Context, string) (map[string]interface{}, error)
	GetProjectCostBreakdown(context.Context, string, time.Time, time.Time) (*types.ProjectCostBreakdown, error)
	GetProjectResourceUsage(context.Context, string, time.Duration) (*types.ProjectResourceUsage, error)
	GetProjectBudgetHistory(context.Context, string, int) ([]float64, error) // Issue #482
	GetCostTrends(context.Context, string, string) (map[string]interface{}, error)
	PreventProjectLaunches(context.Context, string) (map[string]interface{}, error)
	AllowProjectLaunches(context.Context, string) (map[string]interface{}, error)
	GetProjectCushion(context.Context, string) (map[string]interface{}, error)
	SetProjectCushion(context.Context, string, map[string]interface{}) (map[string]interface{}, error)
	DeleteProjectCushion(context.Context, string) error
	GetProjectGDEW(context.Context, string) (map[string]interface{}, error)
	UpdateProjectGDEW(context.Context, string, float64, float64) (map[string]interface{}, error)
	GetProjectDiscounts(context.Context, string) (map[string]interface{}, error)
	GetProjectCredits(context.Context, string) (map[string]interface{}, error)

	// Budget management operations (v0.6.2)
	CreateBudget(context.Context, project.CreateBudgetRequest) (*types.Budget, error)
	GetBudget(context.Context, string) (*types.Budget, error)
	ListBudgets(context.Context) ([]*types.Budget, error)
	UpdateBudget(context.Context, string, project.UpdateBudgetRequest) (*types.Budget, error)
	DeleteBudget(context.Context, string) error
	GetBudgetSummary(context.Context, string) (*types.BudgetSummary, error)
	GetBudgetAllocations(context.Context, string) ([]*types.ProjectBudgetAllocation, error)

	// Allocation management operations (v0.6.2)
	CreateAllocation(context.Context, project.CreateAllocationRequest) (*types.ProjectBudgetAllocation, error)
	GetAllocation(context.Context, string) (*types.ProjectBudgetAllocation, error)
	GetProjectAllocations(context.Context, string) ([]*types.ProjectBudgetAllocation, error)
	UpdateAllocation(context.Context, string, project.UpdateAllocationRequest) (*types.ProjectBudgetAllocation, error)
	DeleteAllocation(context.Context, string) error
	RecordSpending(context.Context, string, float64) (*project.SpendingResult, error)
	CheckAllocationStatus(context.Context, string) (*AllocationStatus, error)
	GetProjectFundingSummary(context.Context, string) (*types.ProjectFundingSummary, error)

	// Invitation operations (v0.5.11)
	GetInvitationByToken(context.Context, string) (*GetInvitationResponse, error)
	GetInvitationByID(context.Context, string) (*GetInvitationResponse, error) // v0.6.2 (Issue #383)
	AcceptInvitation(context.Context, string) (*InvitationActionResponse, error)
	DeclineInvitation(context.Context, string, string) (*InvitationActionResponse, error)
	SendInvitation(context.Context, string, SendInvitationRequest) (*SendInvitationResponse, error)
	SendBulkInvitation(context.Context, string, *types.BulkInvitationRequest) (*types.BulkInvitationResponse, error)
	RevokeInvitation(context.Context, string) error                                                  // v0.6.2 (Issue #383)
	ResendInvitation(context.Context, string) error                                                  // v0.6.2 (Issue #383)
	ListProjectInvitations(context.Context, string, string) (*ListProjectInvitationsResponse, error) // projectID, status (Issue #50)

	// Shared token operations (v0.5.13)
	CreateSharedToken(context.Context, string, *types.CreateSharedTokenRequest) (*types.SharedInvitationToken, error)
	RedeemSharedToken(context.Context, *types.RedeemSharedTokenRequest) (*types.RedeemSharedTokenResponse, error)
	GetSharedToken(context.Context, string) (*types.SharedInvitationToken, error)
	ListSharedTokens(context.Context, string) ([]*types.SharedInvitationToken, error)
	ExtendSharedToken(context.Context, string, int) error
	RevokeSharedToken(context.Context, string) error
	GetSharedTokenQR(context.Context, string) ([]byte, error)

	// Policy management operations (Phase 5A.5)
	GetPolicyStatus(context.Context) (*PolicyStatusResponse, error)
	ListPolicySets(context.Context) (*PolicySetsResponse, error)
	AssignPolicySet(context.Context, string) (*PolicyAssignResponse, error)
	SetPolicyEnforcement(context.Context, bool) (*PolicyEnforcementResponse, error)
	CheckTemplateAccess(context.Context, string) (*PolicyCheckResponse, error)

	// Idle policy operations
	ListIdlePolicies(context.Context) ([]*idle.PolicyTemplate, error)
	GetIdlePolicy(context.Context, string) (*idle.PolicyTemplate, error)
	ApplyIdlePolicy(context.Context, string, string) error
	RemoveIdlePolicy(context.Context, string, string) error
	GetInstanceIdlePolicies(context.Context, string) ([]*idle.PolicyTemplate, error)
	RecommendIdlePolicy(context.Context, string) (*idle.PolicyTemplate, error)
	GetIdleSavingsReport(context.Context, string) (map[string]interface{}, error)

	// Rightsizing analysis operations
	AnalyzeRightsizing(context.Context, types.RightsizingAnalysisRequest) (*types.RightsizingAnalysisResponse, error)
	GetRightsizingRecommendations(context.Context) (*types.RightsizingRecommendationsResponse, error)
	GetRightsizingStats(context.Context, string) (*types.RightsizingStatsResponse, error)
	ExportRightsizingData(context.Context, string) ([]types.InstanceMetrics, error)
	GetRightsizingSummary(context.Context) (*types.RightsizingSummaryResponse, error)
	GetInstanceMetrics(context.Context, string, int) ([]types.InstanceMetrics, error)

	// Transfer operations (S3-backed file transfer)
	StartTransfer(context.Context, TransferRequest) (*TransferResponse, error)
	GetTransferStatus(context.Context, string) (*storage.TransferProgress, error)
	ListTransfers(context.Context) ([]*storage.TransferProgress, error)
	CancelTransfer(context.Context, string) error

	// SSM file operations (#30a)
	ListInstanceFiles(context.Context, string, string) ([]interface{}, error)
	PushFileToInstance(context.Context, string, string, string) (map[string]interface{}, error)
	PullFileFromInstance(context.Context, string, string, string) (map[string]interface{}, error)

	// Throttling operations (v0.6.0 - Issue #90)
	GetThrottlingStatus(context.Context, string) (*throttle.Status, error)
	ConfigureThrottling(context.Context, throttle.Config) error
	GetThrottlingRemaining(context.Context, string) (map[string]interface{}, error)
	SetProjectThrottleOverride(context.Context, string, throttle.Override) error
	RemoveProjectThrottleOverride(context.Context, string) error
	ListProjectThrottleOverrides(context.Context) ([]throttle.Override, error)

	// Status operations
	GetStatus(context.Context) (*types.DaemonStatus, error)
	Ping(context.Context) error
	Shutdown(context.Context) error

	// Raw API request method for generic endpoint access
	MakeRequest(method, path string, body interface{}) ([]byte, error)

	// Registry operations
	GetRegistryStatus(context.Context) (*RegistryStatusResponse, error)
	SetRegistryStatus(context.Context, bool) error
	LookupAMI(context.Context, string, string, string) (*AMIReferenceResponse, error)
	ListTemplateAMIs(context.Context, string) ([]AMIReferenceResponse, error)

	// Universal AMI System operations (Phase 5.1 Week 2)
	ResolveAMI(context.Context, string, map[string]interface{}) (map[string]interface{}, error)
	TestAMIAvailability(context.Context, map[string]interface{}) (map[string]interface{}, error)
	GetAMICosts(context.Context, string) (map[string]interface{}, error)
	PreviewAMIResolution(context.Context, string) (map[string]interface{}, error)

	// AMI Creation operations (Phase 5.1 AMI Creation)
	CreateAMI(context.Context, types.AMICreationRequest) (map[string]interface{}, error)
	GetAMIStatus(context.Context, string) (map[string]interface{}, error)
	ListUserAMIs(context.Context) (map[string]interface{}, error)

	// AMI Lifecycle Management operations
	CleanupAMIs(context.Context, map[string]interface{}) (map[string]interface{}, error)
	DeleteAMI(context.Context, map[string]interface{}) (map[string]interface{}, error)

	// AMI Snapshot operations
	ListAMISnapshots(context.Context, map[string]interface{}) (map[string]interface{}, error)
	CreateAMISnapshot(context.Context, map[string]interface{}) (map[string]interface{}, error)
	RestoreAMIFromSnapshot(context.Context, map[string]interface{}) (map[string]interface{}, error)
	DeleteAMISnapshot(context.Context, map[string]interface{}) (map[string]interface{}, error)

	// AMI Freshness Checking (v0.5.4 - Universal Version System)
	CheckAMIFreshness(context.Context) (map[string]interface{}, error)

	// Instance Snapshot operations
	CreateInstanceSnapshot(context.Context, types.InstanceSnapshotRequest) (*types.InstanceSnapshotResult, error)
	ListInstanceSnapshots(context.Context) (*types.InstanceSnapshotListResponse, error)
	GetInstanceSnapshot(context.Context, string) (*types.InstanceSnapshotInfo, error)
	DeleteInstanceSnapshot(context.Context, string) (*types.InstanceSnapshotDeleteResult, error)
	RestoreInstanceFromSnapshot(context.Context, string, types.InstanceRestoreRequest) (*types.InstanceRestoreResult, error)

	// Template Marketplace operations (Phase 5.2)
	SearchMarketplace(context.Context, map[string]interface{}) (map[string]interface{}, error)
	GetMarketplaceTemplate(context.Context, string) (map[string]interface{}, error)
	PublishMarketplaceTemplate(context.Context, map[string]interface{}) (map[string]interface{}, error)
	AddMarketplaceReview(context.Context, string, map[string]interface{}) (map[string]interface{}, error)
	ForkMarketplaceTemplate(context.Context, string, map[string]interface{}) (map[string]interface{}, error)
	GetMarketplaceFeatured(context.Context) (map[string]interface{}, error)
	GetMarketplaceTrending(context.Context) (map[string]interface{}, error)

	// Data Backup operations
	CreateBackup(context.Context, types.BackupCreateRequest) (*types.BackupCreateResult, error)
	ListBackups(context.Context) (*types.BackupListResponse, error)
	GetBackup(context.Context, string) (*types.BackupInfo, error)
	DeleteBackup(context.Context, string) (*types.BackupDeleteResult, error)
	GetBackupContents(context.Context, types.BackupContentsRequest) (*types.BackupContentsResponse, error)
	VerifyBackup(context.Context, types.BackupVerifyRequest) (*types.BackupVerifyResult, error)

	// Data Restore operations
	RestoreBackup(context.Context, types.RestoreRequest) (*types.RestoreResult, error)
	GetRestoreStatus(context.Context, string) (*types.RestoreResult, error)
	ListRestoreOperations(context.Context) ([]types.RestoreResult, error)

	// Version compatibility checking
	CheckVersionCompatibility(context.Context, string) error

	// v0.12.0 — Budget Enterprise & Governance

	// Resource quota management (#151)
	GetProjectQuotas(ctx context.Context, projectID string) ([]types.RoleQuota, error)
	SetProjectQuota(ctx context.Context, projectID string, quota types.RoleQuota) error

	// Grant period management (#152, #144)
	GetGrantPeriod(ctx context.Context, projectID string) (*types.GrantPeriod, error)
	SetGrantPeriod(ctx context.Context, projectID string, gp types.GrantPeriod) error
	DeleteGrantPeriod(ctx context.Context, projectID string) error

	// Budget sharing / reallocation (#143, #145, #155, #156)
	ShareProjectBudget(ctx context.Context, req types.BudgetShareRequest) (*types.BudgetShareRecord, error)
	ListProjectBudgetShares(ctx context.Context, projectID string) ([]*types.BudgetShareRecord, error)
	DeleteProjectBudgetShare(ctx context.Context, projectID, shareID string) error

	// Approval workflow (#148, #149, #153, #157)
	SubmitApproval(ctx context.Context, projectID string, approvalType project.ApprovalType, details map[string]interface{}, reason string) (*project.ApprovalRequest, error)
	ListApprovals(ctx context.Context, projectID string, status project.ApprovalStatus) ([]*project.ApprovalRequest, error)
	ListAllApprovals(ctx context.Context, status project.ApprovalStatus) ([]*project.ApprovalRequest, error)
	ApproveRequest(ctx context.Context, projectID, requestID, note string) (*project.ApprovalRequest, error)
	DenyRequest(ctx context.Context, projectID, requestID, note string) (*project.ApprovalRequest, error)
	GetApproval(ctx context.Context, projectID, approvalID string) (*project.ApprovalRequest, error)

	// Monthly budget reports (#141)
	GetMonthlyReport(ctx context.Context, projectID, month, format string) (string, error)

	// Onboarding templates (#154)
	ListOnboardingTemplates(ctx context.Context, projectID string) ([]types.OnboardingTemplate, error)
	AddOnboardingTemplate(ctx context.Context, projectID string, tmpl types.OnboardingTemplate) error
	DeleteOnboardingTemplate(ctx context.Context, projectID, nameOrID string) error

	// v0.14.0 — University Education System (core CRUD, in interface for fixture cleanup)
	DeleteCourse(ctx context.Context, courseID string) error

	// v0.18.0 — Workshop & Event Management (#135, #176-#184)
	ListWorkshops(ctx context.Context, params string) (map[string]interface{}, error)
	CreateWorkshop(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error)
	GetWorkshop(ctx context.Context, workshopID string) (map[string]interface{}, error)
	UpdateWorkshop(ctx context.Context, workshopID string, req map[string]interface{}) (map[string]interface{}, error)
	DeleteWorkshop(ctx context.Context, workshopID string) error
	ProvisionWorkshop(ctx context.Context, workshopID string) (map[string]interface{}, error)
	GetWorkshopDashboard(ctx context.Context, workshopID string) (map[string]interface{}, error)
	EndWorkshop(ctx context.Context, workshopID string) (map[string]interface{}, error)
	GetWorkshopDownload(ctx context.Context, workshopID string) (map[string]interface{}, error)
	ListWorkshopConfigs(ctx context.Context) (map[string]interface{}, error)
	SaveWorkshopConfig(ctx context.Context, workshopID, name string) (map[string]interface{}, error)
	CreateWorkshopFromConfig(ctx context.Context, configName string, req map[string]interface{}) (map[string]interface{}, error)
	AddWorkshopParticipant(ctx context.Context, workshopID string, req map[string]interface{}) (map[string]interface{}, error)
	RemoveWorkshopParticipant(ctx context.Context, workshopID, userID string) error
}

// Registry-specific response types for API operations

// RegistryStatusResponse represents the response from GetRegistryStatus
type RegistryStatusResponse struct {
	// Active indicates if the registry is active
	Active bool `json:"active"`

	// LastSync is when the registry was last synchronized
	LastSync *time.Time `json:"last_sync,omitempty"`

	// TemplateCount is the number of templates in the registry
	TemplateCount int `json:"template_count"`

	// AMICount is the total number of AMIs in the registry
	AMICount int `json:"ami_count"`

	// Status provides additional status information
	Status string `json:"status"`
}

// AMIReferenceResponse represents an AMI reference response
type AMIReferenceResponse struct {
	// AMIID is the AMI identifier
	AMIID string `json:"ami_id"`

	// Region is the AWS region where the AMI is located
	Region string `json:"region"`

	// Architecture is the AMI architecture (x86_64 or arm64)
	Architecture string `json:"architecture"`

	// TemplateName is the name of the template this AMI was built from
	TemplateName string `json:"template_name"`

	// Version is the semantic version of the template
	Version string `json:"version"`

	// BuildDate is when the AMI was built
	BuildDate time.Time `json:"build_date"`

	// Status is the current status of the AMI
	Status string `json:"status"`

	// Tags contains metadata tags for the AMI
	Tags map[string]string `json:"tags,omitempty"`
}

// Policy management response types (Phase 5A.5)

// PolicyStatusResponse represents the policy enforcement status
type PolicyStatusResponse struct {
	Enabled          bool     `json:"enabled"`
	Status           string   `json:"status"`
	StatusIcon       string   `json:"status_icon"`
	AssignedPolicies []string `json:"assigned_policies"`
	Message          string   `json:"message,omitempty"`
}

// PolicySetsResponse represents available policy sets
type PolicySetsResponse struct {
	PolicySets map[string]PolicySetInfo `json:"policy_sets"`
}

// PolicySetInfo provides information about a policy set
type PolicySetInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Policies    int               `json:"policies"`
	Status      string            `json:"status"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// PolicyAssignResponse represents the response to a policy assignment
type PolicyAssignResponse struct {
	Success           bool   `json:"success"`
	Message           string `json:"message"`
	AssignedPolicySet string `json:"assigned_policy_set"`
	EnforcementStatus string `json:"enforcement_status"`
}

// PolicyEnforcementResponse represents the response to enforcement changes
type PolicyEnforcementResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

// PolicyCheckResponse represents the result of a policy check
type PolicyCheckResponse struct {
	Allowed         bool     `json:"allowed"`
	TemplateName    string   `json:"template_name"`
	Reason          string   `json:"reason"`
	MatchedPolicies []string `json:"matched_policies,omitempty"`
	Suggestions     []string `json:"suggestions,omitempty"`
}

// Budget and allocation operation types (v0.6.2)

// AllocationStatus represents the status of a budget allocation
type AllocationStatus struct {
	Exhausted bool    `json:"exhausted"`
	Remaining float64 `json:"remaining"`
}

// Invitation operation types (v0.5.11)

// GetInvitationResponse represents the response from GetInvitationByToken
type GetInvitationResponse struct {
	Invitation *types.Invitation `json:"invitation"`
	Project    *types.Project    `json:"project"`
}

// InvitationActionResponse represents the response from accept/decline operations
type InvitationActionResponse struct {
	Invitation *types.Invitation `json:"invitation"`
	Project    *types.Project    `json:"project,omitempty"`
	Message    string            `json:"message"`
}

// SendInvitationRequest represents a request to send an invitation
type SendInvitationRequest struct {
	Email     string            `json:"email"`
	Role      types.ProjectRole `json:"role"`
	Message   string            `json:"message,omitempty"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
}

// SendInvitationResponse represents the response from sending an invitation
type SendInvitationResponse struct {
	Invitation *types.Invitation `json:"invitation"`
	Project    *types.Project    `json:"project"`
	Message    string            `json:"message"`
}

// ListProjectInvitationsResponse represents the response from listing project invitations
type ListProjectInvitationsResponse struct {
	Invitations []*types.Invitation `json:"invitations"`
	Total       int                 `json:"total"`
}
