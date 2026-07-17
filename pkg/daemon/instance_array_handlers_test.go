package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// postJSON marshals req (or sends it verbatim if it's a string) and POSTs it to
// path, returning the recorder. Shared by the array and sweep handler tests.
func postJSON(t *testing.T, handler http.Handler, path string, req interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var body []byte
	if str, ok := req.(string); ok {
		body = []byte(str)
	} else {
		var err error
		body, err = json.Marshal(req)
		require.NoError(t, err)
	}
	httpReq := httptest.NewRequest("POST", path, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httpReq)
	return w
}

// postArray POSTs req to the array endpoint, returning the recorder.
func postArray(t *testing.T, handler http.Handler, req interface{}) *httptest.ResponseRecorder {
	t.Helper()
	return postJSON(t, handler, "/api/v1/instances/array", req)
}

// TestHandleLaunchArray_FansOutDistinctNames verifies a count=3 array launches 3
// members with distinct <base>-i names, a shared job-array id, and indices 0..N-1.
func TestHandleLaunchArray_FansOutDistinctNames(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	arrReq := types.LaunchArrayRequest{
		LaunchRequest: types.LaunchRequest{Template: "test-template", Name: "sweep"},
		Count:         3,
	}
	w := postArray(t, handler, arrReq)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var resp types.LaunchArrayResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, 3, resp.Requested)
	assert.Equal(t, 3, resp.Launched)
	assert.NotEmpty(t, resp.JobArrayID)
	require.Len(t, resp.Instances, 3)

	seenNames := map[string]bool{}
	seenIdx := map[int]bool{}
	for _, inst := range resp.Instances {
		seenNames[inst.Name] = true
		seenIdx[inst.JobArrayIndex] = true
		assert.Equal(t, resp.JobArrayID, inst.JobArrayID, "every member shares the array id")
	}
	for i := 0; i < 3; i++ {
		assert.True(t, seenNames[fmt.Sprintf("sweep-%d", i)], "expected member sweep-%d", i)
		assert.True(t, seenIdx[i], "expected index %d", i)
	}
}

// TestHandleLaunchArray_RejectsBadCount covers count < 1 and count over the cap.
func TestHandleLaunchArray_RejectsBadCount(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	for _, count := range []int{0, maxJobArraySize + 1} {
		arrReq := types.LaunchArrayRequest{
			LaunchRequest: types.LaunchRequest{Template: "test-template", Name: "bad"},
			Count:         count,
		}
		w := postArray(t, handler, arrReq)
		assert.Equal(t, http.StatusBadRequest, w.Code, "count=%d should be rejected", count)
	}
}

// TestHandleLaunchArray_RejectsDuplicateName confirms an up-front collision check:
// if a member name already exists in state, the whole array is rejected before any
// launch (no half-launched array).
func TestHandleLaunchArray_RejectsDuplicateName(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	// Pre-create an instance named "dup-1" so member index 1 collides.
	require.NoError(t, server.stateManager.SaveInstance(types.Instance{
		Name:  "dup-1",
		ID:    "i-existing",
		State: "running",
	}))

	arrReq := types.LaunchArrayRequest{
		LaunchRequest: types.LaunchRequest{Template: "test-template", Name: "dup"},
		Count:         3,
	}
	w := postArray(t, handler, arrReq)
	assert.Equal(t, http.StatusConflict, w.Code, "body: %s", w.Body.String())
}

// TestHandleLaunchArray_MissingTemplate confirms embedded-request validation runs.
func TestHandleLaunchArray_MissingTemplate(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	w := postArray(t, handler, map[string]interface{}{
		"name":  "notemplate",
		"count": 2,
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleLaunchArray_RouteNotSwallowedByCatchAll guards the route ordering: the
// array endpoint must be registered before /api/v1/instances/ so it isn't routed
// to handleInstanceOperations (which would 400/404 on "array" as an instance name).
func TestHandleLaunchArray_RouteNotSwallowedByCatchAll(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	arrReq := types.LaunchArrayRequest{
		LaunchRequest: types.LaunchRequest{Template: "test-template", Name: "routed"},
		Count:         1,
	}
	w := postArray(t, handler, arrReq)
	// A count=1 array is valid and should launch one member — proving the route
	// reached handleLaunchArray, not the instance-operations catch-all.
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	var resp types.LaunchArrayResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Launched)
}
