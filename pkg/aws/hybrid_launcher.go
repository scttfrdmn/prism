package aws

import (
	"context"

	spawn "github.com/spore-host/spawn/pkg/aws"

	"github.com/scttfrdmn/prism/pkg/launch"
)

// hybridLauncher is the Phase-2 launch/lifecycle engine: it routes instance
// LIFECYCLE (stop/start/hibernate/terminate) through spore.host/spawn's client,
// while keeping LAUNCH on Prism's own ec2Launcher.
//
// Launch stays on ec2Launcher deliberately: spawn v0.65.0 hardcodes the root
// block-device name to /dev/xvda and never derives it from the AMI, but Prism's
// default templates are Ubuntu/Rocky whose AMIs register root as /dev/sda1.
// Routing launch through spawn as-is would land the root-volume-size override on
// a non-root device. Launch swaps to spawn once that gap is closed upstream.
//
// Lifecycle is a safe swap: spawn's Stop/Start/Terminate touch no block-device
// or tag logic, wrap AWS errors with %w (so Prism's InvalidInstanceID.NotFound
// graceful-delete match still works), and StopInstance maps hibernate directly.
type hybridLauncher struct {
	launch    *ec2Launcher  // launch path (Prism's own RunInstances build)
	lifecycle *spawn.Client // stop/start/terminate via spawn
}

// newHybridLauncher builds the default launcher: Prism's ec2Launcher for launch,
// a spawn client (from the manager's AWS config) for lifecycle.
func newHybridLauncher(m *Manager) *hybridLauncher {
	return &hybridLauncher{
		launch:    newEC2Launcher(m),
		lifecycle: spawn.NewClientFromConfig(m.cfg),
	}
}

// compile-time proof the hybrid satisfies the port.
var _ launch.Launcher = (*hybridLauncher)(nil)

// Launch delegates to Prism's EC2-backed launcher (see type doc for why).
func (h *hybridLauncher) Launch(ctx context.Context, cfg spawn.LaunchConfig) (*spawn.LaunchResult, error) {
	return h.launch.Launch(ctx, cfg)
}

// StopInstance stops (or hibernates) via spawn, preserving Prism's retry policy
// for transient failures (spawn issues a single call with no internal retry).
func (h *hybridLauncher) StopInstance(ctx context.Context, region, instanceID string, hibernate bool) error {
	return WithRetry(ctx, DefaultRetryConfig(), func(ctx context.Context) error {
		return h.lifecycle.StopInstance(ctx, region, instanceID, hibernate)
	})
}

// StartInstance starts (or resumes) via spawn, with Prism's retry policy.
func (h *hybridLauncher) StartInstance(ctx context.Context, region, instanceID string) error {
	return WithRetry(ctx, DefaultRetryConfig(), func(ctx context.Context) error {
		return h.lifecycle.StartInstance(ctx, region, instanceID)
	})
}

// Terminate deletes via spawn, with Prism's retry policy. spawn wraps the AWS
// error with %w, so callers can still detect the already-gone case.
func (h *hybridLauncher) Terminate(ctx context.Context, region, instanceID string) error {
	return WithRetry(ctx, DefaultRetryConfig(), func(ctx context.Context) error {
		return h.lifecycle.Terminate(ctx, region, instanceID)
	})
}
