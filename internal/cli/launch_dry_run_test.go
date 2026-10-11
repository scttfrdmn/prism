package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scttfrdmn/prism/pkg/types"
)

// Dry-run regression tests for the client side (#703).
//
// These guard against a hang introduced by the daemon-side fix. The CLI's
// dry-run success message used to come from LaunchProgressMonitor: it polled
// GetInstance, found the phantom record the daemon had saved, read state
// "dry-run", and printed success. Once the daemon stops writing that record,
// GetInstance fails on every poll -- and Monitor swallows errors with no early
// exit, so the loop runs its full 240 iterations at 5 seconds each. A dry run
// would sit silent for 20 minutes before timing out.
//
// So the CLI must decide not to monitor a dry run on its own, without consulting
// the daemon. shouldMonitorLaunch is where that decision is made.

// noAMITemplate registers a package-based template on the mock: launching it
// really does need progress monitoring, because packages install after boot.
// Every template the mock ships with carries an AMI specifically to suppress
// monitoring, so a template without one has to be added here to exercise the
// monitored path at all.
func noAMITemplate(mock *MockAPIClient) {
	mock.Templates["packages-only"] = types.Template{
		Name:        "packages-only",
		Description: "Package-based template with no prebuilt AMI",
	}
}

// TestShouldMonitorLaunch_PackageTemplateIsMonitored is the control: without
// --dry-run, a package-based template must still be monitored. It fixes the
// meaning of the two tests below, which would otherwise pass for a CLI that had
// simply stopped monitoring anything.
func TestShouldMonitorLaunch_PackageTemplateIsMonitored(t *testing.T) {
	mock := NewMockAPIClient()
	noAMITemplate(mock)
	app := NewAppWithClient("1.0.0", mock)

	req := types.LaunchRequest{Template: "packages-only", Name: "real", Quiet: true}

	assert.True(t, app.shouldMonitorLaunch(&req),
		"a real launch of a package-based template installs packages after boot, so it is monitored")
}

// TestShouldMonitorLaunch_DryRunIsNotMonitored covers the template shape that
// would otherwise hang: package-based, so the CLI's usual answer is "monitor".
func TestShouldMonitorLaunch_DryRunIsNotMonitored(t *testing.T) {
	mock := NewMockAPIClient()
	noAMITemplate(mock)
	app := NewAppWithClient("1.0.0", mock)

	req := types.LaunchRequest{Template: "packages-only", Name: "dry", DryRun: true, Quiet: true}

	assert.False(t, app.shouldMonitorLaunch(&req),
		"a dry run created no workspace, so there is nothing to watch and nothing to wait for")
}

// TestShouldMonitorLaunch_DryRunWithWaitIsNotMonitored covers the other route
// into monitoring. --wait short-circuits to true before the template is even
// consulted, so it needs its own test: "wait for it to be ready" has no meaning
// when nothing was created.
func TestShouldMonitorLaunch_DryRunWithWaitIsNotMonitored(t *testing.T) {
	mock := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mock)

	req := types.LaunchRequest{Template: "python-ml", Name: "dry", DryRun: true, Wait: true, Quiet: true}

	assert.False(t, app.shouldMonitorLaunch(&req),
		"--wait cannot wait for a workspace a dry run declined to create")

	// Control: the same request without --dry-run is still monitored.
	realReq := req
	realReq.DryRun = false
	require.True(t, app.shouldMonitorLaunch(&realReq),
		"--wait on a real launch must still monitor")
}
