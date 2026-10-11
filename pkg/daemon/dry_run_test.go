package daemon

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scttfrdmn/prism/pkg/types"
)

// Dry-run regression tests (#703).
//
// A dry run promises three things, and every test in this file pins one of them:
//
//  1. No AWS call. Already covered by TestLaunchInstance_DryRunNeverCallsLauncher
//     in pkg/aws, so it is not repeated here.
//  2. No state written. The daemon used to persist the synthetic dry-run instance,
//     which left a phantom record in ~/.prism/state.json for a workspace that was
//     never created.
//  3. No success claimed. The daemon used to answer "Instance X launched
//     successfully" for a launch it had deliberately skipped.
//
// The phantom from (2) then collided with the name-uniqueness check, so the real
// launch of that name came back 409. That check has its own table test in
// handlers_test.go; the "dry-run record allows name reuse" case there covers
// phantoms already sitting in state.json from an earlier version, which fixing
// the save path does not clean up.

// TestLaunchDryRun_WritesNoInstanceState is the core of the phantom-record bug:
// after a dry run, ~/.prism/state.json must hold no record for that name, so the
// real launch of the same name is free to proceed.
func TestLaunchDryRun_WritesNoInstanceState(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	w := postJSON(t, handler, "/api/v1/instances", types.LaunchRequest{
		Template: "test-template",
		Name:     "phantom-check",
		DryRun:   true,
	})
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	state, err := server.stateManager.LoadState()
	require.NoError(t, err)
	_, exists := state.Instances["phantom-check"]
	assert.False(t, exists,
		"a dry run created nothing on AWS, so it must leave no instance record behind")
}

// TestLaunchDryRun_DoesNotClaimLaunch pins the message the user actually sees.
// The CLI prints the daemon's Message verbatim behind a check mark, so a Message
// claiming success is a claim the user has no reason to doubt.
func TestLaunchDryRun_DoesNotClaimLaunch(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	w := postJSON(t, handler, "/api/v1/instances", types.LaunchRequest{
		Template: "test-template",
		Name:     "message-check",
		DryRun:   true,
	})
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var resp types.LaunchResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.NotContains(t, strings.ToLower(resp.Message), "launched successfully",
		"nothing was launched, so the message must not say it was")
	assert.Regexp(t, "(?i)dry[- ]run", resp.Message,
		"the message should tell the user this was a dry run")

	// The dry-run instance has no IP, so a connection string built from it reads
	// "ssh ubuntu@" with nothing after the @.
	assert.Empty(t, resp.ConnectionInfo,
		"there is no host to connect to after a dry run")

	// Test mode must report the same state production does, or it hides the bug.
	assert.Equal(t, "dry-run", resp.Instance.State)
}

// TestLaunchDryRun_RealLaunchOfSameNameSucceeds is the user-visible symptom:
// dry-run a name, then launch it for real. The second call used to fail with 409.
//
// Test mode skips the uniqueness check entirely (instanceNameCheckPassed returns
// early when s.testMode), so this test asserts the state precondition the check
// reads rather than the check itself, and it passes both before and after the
// fix. The check is exercised directly by the "dry-run record allows name reuse"
// case in TestCheckInstanceNameUniqueness (handlers_test.go).
func TestLaunchDryRun_RealLaunchOfSameNameSucceeds(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	dry := postJSON(t, handler, "/api/v1/instances", types.LaunchRequest{
		Template: "test-template",
		Name:     "reuse-me",
		DryRun:   true,
	})
	require.Equal(t, http.StatusOK, dry.Code, "body: %s", dry.Body.String())

	real := postJSON(t, handler, "/api/v1/instances", types.LaunchRequest{
		Template: "test-template",
		Name:     "reuse-me",
	})
	require.Equal(t, http.StatusOK, real.Code,
		"the real launch must not be blocked by the preceding dry run; body: %s", real.Body.String())

	state, err := server.stateManager.LoadState()
	require.NoError(t, err)
	inst, exists := state.Instances["reuse-me"]
	require.True(t, exists, "the real launch must be recorded")
	assert.NotEqual(t, "dry-run", inst.State)
}

// TestLaunchDryRun_RecordsNoLaunchAuditEvent covers the third thing a dry run
// must not do: tell the audit log a workspace was launched. The daemon appends
// an "instance.launch" event on every successful launch, and the dry-run path
// used to reach that call like any other.
func TestLaunchDryRun_RecordsNoLaunchAuditEvent(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	w := postJSON(t, handler, "/api/v1/instances", types.LaunchRequest{
		Template: "test-template",
		Name:     "audit-check",
		DryRun:   true,
	})
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	// createTestServer points HOME at a fresh temp dir, so this log holds only
	// what this test produced. No file at all is the cleanest possible pass.
	logPath := filepath.Join(os.Getenv("HOME"), ".prism", "audit", "operations.jsonl")
	raw, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return
	}
	require.NoError(t, err)

	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line == "" {
			continue
		}
		var event struct {
			Action   string `json:"action"`
			Resource string `json:"resource"`
		}
		require.NoError(t, json.Unmarshal([]byte(line), &event))
		if event.Action == "instance.launch" && event.Resource == "audit-check" {
			t.Errorf("a dry run launched nothing, so it must not be audited as a launch: %s", line)
		}
	}
}

// TestLaunchArrayDryRun_WritesNoInstanceState covers the same phantom bug on the
// job-array path: DryRun rides in the embedded LaunchRequest, so --count 3
// --dry-run used to write three phantoms.
func TestLaunchArrayDryRun_WritesNoInstanceState(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	w := postJSON(t, handler, "/api/v1/instances/array", types.LaunchArrayRequest{
		LaunchRequest: types.LaunchRequest{Template: "test-template", Name: "arr", DryRun: true},
		Count:         3,
	})
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var resp types.LaunchArrayResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Launched, "a dry run launches nothing, whatever it validates")

	state, err := server.stateManager.LoadState()
	require.NoError(t, err)
	for _, name := range []string{"arr-0", "arr-1", "arr-2"} {
		_, exists := state.Instances[name]
		assert.False(t, exists, "dry-run array member %s must leave no record", name)
	}
}

// TestLaunchSweepDryRun_WritesNoInstanceState covers the parameter-sweep path,
// which fans out the same way as a job array and had the same unguarded save.
func TestLaunchSweepDryRun_WritesNoInstanceState(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	w := postJSON(t, handler, "/api/v1/instances/sweep", types.LaunchSweepRequest{
		LaunchRequest: types.LaunchRequest{Template: "test-template", Name: "swp", DryRun: true},
		ParamSets: []map[string]string{
			{"lr": "0.01"},
			{"lr": "0.1"},
		},
	})
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var resp types.LaunchSweepResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Launched, "a dry run launches nothing, whatever it validates")

	state, err := server.stateManager.LoadState()
	require.NoError(t, err)
	for _, name := range []string{"swp-0", "swp-1"} {
		_, exists := state.Instances[name]
		assert.False(t, exists, "dry-run sweep member %s must leave no record", name)
	}
}
