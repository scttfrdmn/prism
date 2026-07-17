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

// TestJobArray_LaunchTwo validates the job-array fan-out against real AWS: a
// count=2 array launches two independent members named <base>-0 and <base>-1 that
// share a job-array id and carry distinct indices, both reaching running. Both are
// terminated at the end.
//
// Requires a running daemon (real AWS, NOT PRISM_TEST_MODE) and its API key in
// PRISM_TEST_API_KEY. INCURS AWS CHARGES (~2× a single small instance, ~$0.10–0.40).
func TestJobArray_LaunchTwo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping job-array integration test in short mode")
	}

	apiKey := os.Getenv("PRISM_TEST_API_KEY")

	apiClient := client.NewClientWithOptions("http://localhost:8947", client.Options{
		AWSProfile: "aws",
		AWSRegion:  "us-west-2",
		APIKey:     apiKey,
	})

	ctx := context.Background()
	base := fmt.Sprintf("jobarray-%d", time.Now().Unix())
	members := []string{base + "-0", base + "-1"}

	// Register cleanup up front so a mid-test failure still terminates both members.
	t.Cleanup(func() {
		for _, name := range members {
			t.Logf("Cleanup: terminating %s...", name)
			if derr := apiClient.DeleteInstance(context.Background(), name); derr != nil {
				t.Logf("cleanup delete error (may already be gone): %v", derr)
			}
		}
	})

	t.Logf("Launching job array %s (count=2, Ubuntu 24.04, S)...", base)
	resp, err := apiClient.LaunchArray(ctx, types.LaunchArrayRequest{
		LaunchRequest: types.LaunchRequest{
			Template: "Ubuntu 24.04 LTS (x86_64)",
			Name:     base,
			Size:     "S",
		},
		Count: 2,
	})
	require.NoError(t, err, "job-array launch should succeed")
	require.Equal(t, 2, resp.Requested)
	require.Equal(t, 2, resp.Launched, "both members should launch; errors: %v", resp.Errors)
	require.NotEmpty(t, resp.JobArrayID, "response should carry the shared array id")
	require.Len(t, resp.Instances, 2)

	// Members are distinct instances sharing the array id with distinct indices.
	seenIdx := map[int]bool{}
	for _, inst := range resp.Instances {
		assert.Equal(t, resp.JobArrayID, inst.JobArrayID, "member %s should share the array id", inst.Name)
		seenIdx[inst.JobArrayIndex] = true
	}
	assert.True(t, seenIdx[0] && seenIdx[1], "members should have indices 0 and 1, got %v", seenIdx)

	// Both members must reach running.
	for _, name := range members {
		require.NoError(t, fixtures.WaitForInstanceState(t, apiClient, name, "running", 5*time.Minute),
			"member %s should reach running", name)
	}

	// Terminate both.
	for _, name := range members {
		require.NoError(t, apiClient.DeleteInstance(ctx, name), "terminate %s should succeed", name)
		require.NoError(t, fixtures.WaitForInstanceState(t, apiClient, name, "terminated", 4*time.Minute),
			"member %s should reach terminated", name)
	}

	t.Log("✅ Job-array launch of 2 members succeeded (shared array id, distinct indices, both running then terminated)")
}
