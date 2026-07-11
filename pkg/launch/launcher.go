// Package launch is Prism's seam over the instance launch/lifecycle engine.
//
// It defines a narrow, host-owned Launcher port expressed entirely in terms of
// spore.host/spawn's types, plus a pure mapping from Prism's already-resolved
// launch inputs into a spawn.LaunchConfig. The real *spawn.Client satisfies
// Launcher directly (see the compile-time assertion below), and so does the
// EC2-backed default implementation that lives in pkg/aws. This mirrors the
// pattern proven in the sibling project prp (pkg/engine).
//
// pkg/launch is deliberately dependency-light: it imports spawn's types and
// Prism's shared types only, never pkg/aws. That keeps the mapping unit-testable
// without an AWS client and lets pkg/aws depend on pkg/launch rather than the
// other way around.
package launch

import (
	"context"

	spawn "github.com/spore-host/spawn/pkg/aws"
)

// Launcher is the narrow port Prism drives for instance launch and lifecycle.
// It is expressed in spawn's own types so that swapping the default EC2-backed
// implementation for *spawn.Client is a one-line change with no signature churn.
type Launcher interface {
	// Launch provisions a single instance from the given config and returns the
	// initial result (instance id, IPs, AZ, state). It must not be called for
	// dry-run requests — the caller short-circuits those before reaching here.
	Launch(ctx context.Context, cfg spawn.LaunchConfig) (*spawn.LaunchResult, error)

	// StopInstance stops a running instance. When hibernate is true the instance
	// is hibernated (RAM state preserved) instead of a plain stop.
	StopInstance(ctx context.Context, region, instanceID string, hibernate bool) error

	// StartInstance starts a stopped instance. It also resumes a hibernated one.
	StartInstance(ctx context.Context, region, instanceID string) error

	// Terminate permanently deletes an instance.
	Terminate(ctx context.Context, region, instanceID string) error
}

// Compile-time proof that spawn's real client satisfies the narrowed port, so
// Phase 2's swap (default impl -> *spawn.Client) needs no interface changes.
var _ Launcher = (*spawn.Client)(nil)
