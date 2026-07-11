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

// TestSpawnLifecycle_FullCycle validates the Phase-2 spawn adoption end to end
// against real AWS: instance lifecycle (stop/start/hibernate/resume/terminate)
// now routes through spore.host/spawn's client via the daemon's hybridLauncher.
//
// Launch still uses Prism's own ec2Launcher (spawn's launch path is deferred),
// so this test also confirms the hybrid split works: launch on ec2Launcher,
// lifecycle on spawn, with no observable behavior change.
//
// Requires a running daemon (real AWS, NOT PRISM_TEST_MODE) and its API key in
// PRISM_TEST_API_KEY. INCURS AWS CHARGES (~$0.05–0.20).
func TestSpawnLifecycle_FullCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping spawn lifecycle integration test in short mode")
	}

	apiKey := os.Getenv("PRISM_TEST_API_KEY")

	apiClient := client.NewClientWithOptions("http://localhost:8947", client.Options{
		AWSProfile: "aws",
		AWSRegion:  "us-west-2",
		APIKey:     apiKey,
	})

	ctx := context.Background()
	instanceName := fmt.Sprintf("spawn-lifecycle-%d", time.Now().Unix())

	// Launch (ec2Launcher path). Register cleanup up front so a mid-test failure
	// still terminates the instance (terminate itself is the spawn path).
	t.Logf("Launching instance %s (Ubuntu Basic, S)...", instanceName)
	_, err := apiClient.LaunchInstance(ctx, types.LaunchRequest{
		Template: "Ubuntu 24.04 LTS (x86_64)",
		Name:     instanceName,
		Size:     "S",
	})
	require.NoError(t, err, "launch should succeed")

	t.Cleanup(func() {
		t.Logf("Cleanup: terminating %s (spawn Terminate)...", instanceName)
		if derr := apiClient.DeleteInstance(context.Background(), instanceName); derr != nil {
			t.Logf("cleanup delete error (may already be gone): %v", derr)
		}
	})

	require.NoError(t, fixtures.WaitForInstanceState(t, apiClient, instanceName, "running", 5*time.Minute),
		"instance should reach running")

	// Stop → spawn.StopInstance(hibernate=false)
	t.Run("Stop", func(t *testing.T) {
		require.NoError(t, apiClient.StopInstance(ctx, instanceName), "spawn stop should succeed")
		require.NoError(t, fixtures.WaitForInstanceState(t, apiClient, instanceName, "stopped", 4*time.Minute),
			"instance should reach stopped")
		inst, err := apiClient.GetInstance(ctx, instanceName)
		require.NoError(t, err)
		assert.Equal(t, "stopped", inst.State)
	})

	// Start → spawn.StartInstance
	t.Run("Start", func(t *testing.T) {
		require.NoError(t, apiClient.StartInstance(ctx, instanceName), "spawn start should succeed")
		require.NoError(t, fixtures.WaitForInstanceState(t, apiClient, instanceName, "running", 5*time.Minute),
			"instance should reach running")
	})

	// Hibernate → spawn.StopInstance(hibernate=true) (validation + fallback in Manager).
	// Ubuntu Basic/S may not support hibernation; Prism falls back to a plain stop,
	// so either way the instance must reach stopped without error.
	t.Run("Hibernate", func(t *testing.T) {
		require.NoError(t, apiClient.HibernateInstance(ctx, instanceName),
			"spawn hibernate (or stop fallback) should succeed")
		require.NoError(t, fixtures.WaitForInstanceState(t, apiClient, instanceName, "stopped", 4*time.Minute),
			"instance should reach stopped after hibernate")
	})

	// Resume → spawn.StartInstance
	t.Run("Resume", func(t *testing.T) {
		require.NoError(t, apiClient.ResumeInstance(ctx, instanceName), "spawn resume should succeed")
		require.NoError(t, fixtures.WaitForInstanceState(t, apiClient, instanceName, "running", 5*time.Minute),
			"instance should reach running after resume")
	})

	// Terminate → spawn.Terminate
	t.Run("Terminate", func(t *testing.T) {
		require.NoError(t, apiClient.DeleteInstance(ctx, instanceName), "spawn terminate should succeed")
		require.NoError(t, fixtures.WaitForInstanceState(t, apiClient, instanceName, "terminated", 4*time.Minute),
			"instance should reach terminated")
	})

	t.Log("✅ Spawn lifecycle full cycle succeeded (stop/start/hibernate/resume/terminate via spawn)")
}
