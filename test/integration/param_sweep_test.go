//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/api/client"
	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/scttfrdmn/prism/test/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParamSweep_LaunchTwo validates the parameter-sweep fan-out against real AWS:
// a 2-set sweep launches two independent members named <base>-0 and <base>-1 that
// share a sweep id and carry distinct indices, both reaching running. Each member
// gets its own parameter set (stamped as prism:param:* tags; the on-instance
// exporter — unit-tested separately — turns them into PARAM_* env vars). Both are
// terminated at the end.
//
// Requires a running daemon (real AWS, NOT PRISM_TEST_MODE) and its API key in
// PRISM_TEST_API_KEY. INCURS AWS CHARGES (~2× a single small instance).
func TestParamSweep_LaunchTwo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping parameter-sweep integration test in short mode")
	}

	apiKey := os.Getenv("PRISM_TEST_API_KEY")

	apiClient := client.NewClientWithOptions("http://localhost:8947", client.Options{
		AWSProfile: "aws",
		AWSRegion:  "us-west-2",
		APIKey:     apiKey,
	})

	ctx := context.Background()
	base := fmt.Sprintf("sweep-%d", time.Now().Unix())
	members := []string{base + "-0", base + "-1"}

	t.Cleanup(func() {
		for _, name := range members {
			t.Logf("Cleanup: terminating %s...", name)
			if derr := apiClient.DeleteInstance(context.Background(), name); derr != nil {
				t.Logf("cleanup delete error (may already be gone): %v", derr)
			}
		}
	})

	t.Logf("Launching parameter sweep %s (2 sets, Ubuntu 24.04, S)...", base)
	resp, err := apiClient.LaunchSweep(ctx, types.LaunchSweepRequest{
		LaunchRequest: types.LaunchRequest{
			Template: "Ubuntu 24.04 LTS (x86_64)",
			Name:     base,
			Size:     "S",
		},
		ParamSets: []map[string]string{
			{"lr": "0.01", "epochs": "10"},
			{"lr": "0.10", "epochs": "20"},
		},
	})
	require.NoError(t, err, "parameter-sweep launch should succeed")
	require.Equal(t, 2, resp.Requested)
	require.Equal(t, 2, resp.Launched, "both members should launch; errors: %v", resp.Errors)
	require.NotEmpty(t, resp.SweepID, "response should carry the shared sweep id")
	require.Len(t, resp.Instances, 2)

	seenIdx := map[int]bool{}
	for _, inst := range resp.Instances {
		assert.Equal(t, resp.SweepID, inst.SweepID, "member %s should share the sweep id", inst.Name)
		seenIdx[inst.SweepIndex] = true
	}
	assert.True(t, seenIdx[0] && seenIdx[1], "members should have indices 0 and 1, got %v", seenIdx)

	for _, name := range members {
		require.NoError(t, fixtures.WaitForInstanceState(t, apiClient, name, "running", 5*time.Minute),
			"member %s should reach running", name)
	}

	for _, name := range members {
		require.NoError(t, apiClient.DeleteInstance(ctx, name), "terminate %s should succeed", name)
		require.NoError(t, fixtures.WaitForInstanceState(t, apiClient, name, "terminated", 4*time.Minute),
			"member %s should reach terminated", name)
	}

	t.Log("✅ Parameter-sweep launch of 2 members succeeded (shared sweep id, distinct indices + params, both running then terminated)")
}
