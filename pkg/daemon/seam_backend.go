package daemon

import (
	"fmt"
	"os"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/scttfrdmn/prism/pkg/daemon/logger"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/rbac"
	"github.com/scttfrdmn/prism/pkg/seam"
	"github.com/scttfrdmn/prism/pkg/seam/dynamostore"
	"github.com/scttfrdmn/prism/pkg/types"
)

// seamManagers bundles the four governance managers that persist through the seam. They are built
// together because they share one backend decision (filestore vs. dynamostore) and, in the cloud
// case, one Scope and one DynamoDB client.
type seamManagers struct {
	project  *project.Manager
	budget   *project.BudgetManager
	rbac     *rbac.Manager
	approval *project.ApprovalManager
}

// seamBackendKind is the persistence backend the converged managers run on.
type seamBackendKind int

const (
	seamBackendFile seamBackendKind = iota // ~/.prism file tree (desktop-standalone, the default)
	seamBackendDynamo
)

// resolveSeamBackend reads PRISM_SEAM_BACKEND. Anything other than an explicit "dynamodb"/"dynamo"
// keeps the file-backed desktop path — the default-to-success posture: a plain desktop install is
// standalone unless the operator opts into the shared cloud table.
func resolveSeamBackend() seamBackendKind {
	switch os.Getenv("PRISM_SEAM_BACKEND") {
	case "dynamodb", "dynamo":
		return seamBackendDynamo
	default:
		return seamBackendFile
	}
}

// seamTablePrefix is the per-type table-name prefix for the DynamoDB backend. Each record type gets
// its own table (prefix-<type>): the dynamostore's only type namespace is the table name, since
// List(scope) returns every item under a partition regardless of Go type. This mirrors the
// file-backed layout, which separates the same types by directory.
func seamTablePrefix() string {
	if p := os.Getenv("PRISM_SEAM_TABLE_PREFIX"); p != "" {
		return p
	}
	return "prism"
}

// seamScope is the tenancy key the daemon reads/writes under. It is the zero Scope today (single
// partition) — the cloud path is wired but not yet tenant-aware. Reading a per-Principal scope from
// configuration (so a configured desktop sees the portal's tenant-scoped records) is the next step;
// see PR discussion. Kept as one function so that change lands in exactly one place.
func seamScope() seam.Scope {
	return seam.Scope{}
}

// initSeamManagers builds the four converged managers on the selected backend. The file path is the
// existing behavior-preserving constructors (legacy JSON migration, ~/.prism); the dynamo path
// injects per-type dynamostore.Store[T] values under seamScope().
//
// On any dynamo setup failure it logs and falls back to the file backend rather than failing daemon
// startup — persistence degrading to local is strictly better than no daemon.
func initSeamManagers(awsCfg *awssdk.Config) (seamManagers, error) {
	if resolveSeamBackend() == seamBackendDynamo {
		if awsCfg == nil {
			logger.Warn("PRISM_SEAM_BACKEND=dynamodb but AWS config unavailable; falling back to file-backed seam")
		} else if mgrs, err := initSeamManagersDynamo(*awsCfg); err != nil {
			logger.Warn("Failed to initialize DynamoDB-backed seam; falling back to file-backed seam", "error", err)
		} else {
			logger.Info("Seam backend: DynamoDB", "table_prefix", seamTablePrefix())
			return mgrs, nil
		}
	}
	return initSeamManagersFile()
}

// initSeamManagersFile builds the managers over the file-backed seam via the existing constructors,
// preserving legacy-JSON migration and the ~/.prism layout.
func initSeamManagersFile() (seamManagers, error) {
	projectManager, err := project.NewManager()
	if err != nil {
		return seamManagers{}, fmt.Errorf("failed to initialize project manager: %w", err)
	}
	budgetManager, err := project.NewBudgetManager()
	if err != nil {
		return seamManagers{}, fmt.Errorf("failed to initialize budget manager: %w", err)
	}
	rbacManager, err := rbac.NewManager()
	if err != nil {
		return seamManagers{}, fmt.Errorf("failed to initialize RBAC manager: %w", err)
	}
	approvalManager, err := project.NewApprovalManager()
	if err != nil {
		return seamManagers{}, fmt.Errorf("failed to initialize approval manager: %w", err)
	}
	return seamManagers{
		project:  projectManager,
		budget:   budgetManager,
		rbac:     rbacManager,
		approval: approvalManager,
	}, nil
}

// initSeamManagersDynamo builds the managers over the DynamoDB-backed seam. Each record type gets a
// prefix-<type> table; all share one Scope and one dynamodb.Client. No legacy migration runs here —
// the cloud table is authoritative shared state, not a local file to import. The project manager
// gets its own budget tracker, mirroring how the file-backed project.NewManager builds one.
func initSeamManagersDynamo(cfg awssdk.Config) (seamManagers, error) {
	budgetTracker, err := project.NewBudgetTracker()
	if err != nil {
		return seamManagers{}, fmt.Errorf("budget tracker (dynamo): %w", err)
	}
	ddb := dynamodb.NewFromConfig(cfg)
	prefix := seamTablePrefix()
	scope := seamScope()
	table := func(kind string) string { return prefix + "-" + kind }

	projectManager, err := project.NewManagerForScope(
		dynamostore.New[types.Project](ddb, table("projects")), scope, budgetTracker)
	if err != nil {
		return seamManagers{}, fmt.Errorf("project manager (dynamo): %w", err)
	}
	budgetManager, err := project.NewBudgetManagerForScope(
		dynamostore.New[types.Budget](ddb, table("budgets")),
		dynamostore.New[types.ProjectBudgetAllocation](ddb, table("budget-allocations")),
		dynamostore.New[project.ReallocationRecord](ddb, table("budget-reallocations")),
		scope)
	if err != nil {
		return seamManagers{}, fmt.Errorf("budget manager (dynamo): %w", err)
	}
	rbacManager := rbac.NewManagerForScope(
		dynamostore.New[rbac.State](ddb, table("rbac")), scope)
	approvalManager := project.NewApprovalManagerForScope(
		dynamostore.New[project.ApprovalRequest](ddb, table("approvals")), scope)

	return seamManagers{
		project:  projectManager,
		budget:   budgetManager,
		rbac:     rbacManager,
		approval: approvalManager,
	}, nil
}
