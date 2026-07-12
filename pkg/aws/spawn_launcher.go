package aws

import (
	"context"

	spawn "github.com/spore-host/spawn/pkg/aws"

	"github.com/scttfrdmn/prism/pkg/launch"
)

// spawnLauncher is Prism's launch/lifecycle engine, backed entirely by
// spore.host/spawn. Launch and lifecycle (stop/start/hibernate/terminate) all
// route through a single *spawn.Client.
//
// This is the completed spawn adoption: launch moved onto spawn once
// spore-host/spawn#284 (v0.72.0) made spawn derive the root block-device name
// from the AMI's RootDeviceName (Prism's Ubuntu/Rocky AMIs use /dev/sda1), which
// had been the one blocker keeping launch on Prism's own RunInstances path.
//
// spawn issues a single AWS call per lifecycle op with no internal retry, so
// each method wraps the spawn call in WithRetry to preserve Prism's
// transient-failure retry behavior. spawn %w-wraps its errors, so callers that
// string-match (e.g. DeleteInstance's InvalidInstanceID.NotFound graceful path)
// keep working.
type spawnLauncher struct {
	client *spawn.Client
}

// newSpawnLauncher builds the launcher from the manager's AWS config.
func newSpawnLauncher(m *Manager) *spawnLauncher {
	return &spawnLauncher{client: spawn.NewClientFromConfig(m.cfg)}
}

// compile-time proof the launcher satisfies the port.
var _ launch.Launcher = (*spawnLauncher)(nil)

// Launch provisions an instance via spawn. spawn builds the RunInstancesInput —
// public-IP NIC, AMI-derived root device (#284), gp3 root volume, spot/EFA/
// placement/hibernation options — and stamps its spawn: baseline tags plus the
// prism: tags Prism passes through LaunchConfig.Tags.
func (l *spawnLauncher) Launch(ctx context.Context, cfg spawn.LaunchConfig) (*spawn.LaunchResult, error) {
	return l.client.Launch(ctx, cfg)
}

// StopInstance stops (or hibernates) an instance via spawn, with Prism's retry.
func (l *spawnLauncher) StopInstance(ctx context.Context, region, instanceID string, hibernate bool) error {
	return WithRetry(ctx, DefaultRetryConfig(), func(ctx context.Context) error {
		return l.client.StopInstance(ctx, region, instanceID, hibernate)
	})
}

// StartInstance starts (or resumes) an instance via spawn, with Prism's retry.
func (l *spawnLauncher) StartInstance(ctx context.Context, region, instanceID string) error {
	return WithRetry(ctx, DefaultRetryConfig(), func(ctx context.Context) error {
		return l.client.StartInstance(ctx, region, instanceID)
	})
}

// Terminate deletes an instance via spawn, with Prism's retry. spawn wraps the
// AWS error with %w so the "already gone" case is still detectable by callers.
func (l *spawnLauncher) Terminate(ctx context.Context, region, instanceID string) error {
	return WithRetry(ctx, DefaultRetryConfig(), func(ctx context.Context) error {
		return l.client.Terminate(ctx, region, instanceID)
	})
}
