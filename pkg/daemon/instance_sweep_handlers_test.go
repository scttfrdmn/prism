package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleLaunchSweep_FansOutDistinctNames verifies a 3-set sweep launches 3
// members with distinct <base>-i names, a shared sweep id, and indices 0..N-1.
func TestHandleLaunchSweep_FansOutDistinctNames(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	swReq := types.LaunchSweepRequest{
		LaunchRequest: types.LaunchRequest{Template: "test-template", Name: "hp"},
		ParamSets: []map[string]string{
			{"lr": "0.01"},
			{"lr": "0.1"},
			{"lr": "1.0"},
		},
	}
	w := postJSON(t, handler, "/api/v1/instances/sweep", swReq)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var resp types.LaunchSweepResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, 3, resp.Requested)
	assert.Equal(t, 3, resp.Launched)
	assert.NotEmpty(t, resp.SweepID)
	require.Len(t, resp.Instances, 3)

	seenNames := map[string]bool{}
	seenIdx := map[int]bool{}
	for _, inst := range resp.Instances {
		seenNames[inst.Name] = true
		seenIdx[inst.SweepIndex] = true
		assert.Equal(t, resp.SweepID, inst.SweepID, "every member shares the sweep id")
	}
	for i := 0; i < 3; i++ {
		assert.True(t, seenNames[fmt.Sprintf("hp-%d", i)], "expected member hp-%d", i)
		assert.True(t, seenIdx[i], "expected index %d", i)
	}
}

// TestHandleLaunchSweep_RejectsEmptyAndOversized covers the size bounds.
func TestHandleLaunchSweep_RejectsEmptyAndOversized(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	// Empty param sets.
	w := postJSON(t, handler, "/api/v1/instances/sweep", types.LaunchSweepRequest{
		LaunchRequest: types.LaunchRequest{Template: "test-template", Name: "empty"},
		ParamSets:     nil,
	})
	assert.Equal(t, http.StatusBadRequest, w.Code, "empty param sets should be rejected")

	// Oversized.
	big := make([]map[string]string, maxJobArraySize+1)
	for i := range big {
		big[i] = map[string]string{"i": fmt.Sprintf("%d", i)}
	}
	w = postJSON(t, handler, "/api/v1/instances/sweep", types.LaunchSweepRequest{
		LaunchRequest: types.LaunchRequest{Template: "test-template", Name: "big"},
		ParamSets:     big,
	})
	assert.Equal(t, http.StatusBadRequest, w.Code, "oversized sweep should be rejected")
}

// TestHandleLaunchSweep_RejectsDuplicateName confirms up-front collision checking.
func TestHandleLaunchSweep_RejectsDuplicateName(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	require.NoError(t, server.stateManager.SaveInstance(types.Instance{
		Name: "dup-1", ID: "i-existing", State: "running",
	}))

	w := postJSON(t, handler, "/api/v1/instances/sweep", types.LaunchSweepRequest{
		LaunchRequest: types.LaunchRequest{Template: "test-template", Name: "dup"},
		ParamSets:     []map[string]string{{"a": "1"}, {"a": "2"}, {"a": "3"}},
	})
	assert.Equal(t, http.StatusConflict, w.Code, "body: %s", w.Body.String())
}

// TestHandleLaunchSweep_MissingTemplate confirms embedded-request validation runs.
func TestHandleLaunchSweep_MissingTemplate(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	w := postJSON(t, handler, "/api/v1/instances/sweep", map[string]interface{}{
		"name":       "notemplate",
		"param_sets": []map[string]string{{"a": "1"}},
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleLaunchSweep_RouteNotSwallowedByCatchAll guards the route ordering.
func TestHandleLaunchSweep_RouteNotSwallowedByCatchAll(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	w := postJSON(t, handler, "/api/v1/instances/sweep", types.LaunchSweepRequest{
		LaunchRequest: types.LaunchRequest{Template: "test-template", Name: "routed"},
		ParamSets:     []map[string]string{{"a": "1"}},
	})
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	var resp types.LaunchSweepResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Launched)
}
