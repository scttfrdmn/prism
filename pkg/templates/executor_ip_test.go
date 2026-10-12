package templates

import (
	"strings"
	"testing"

	"github.com/scttfrdmn/prism/pkg/types"
)

type fakeStateManager struct{ state *types.State }

func (f fakeStateManager) LoadState() (*types.State, error) { return f.state, nil }

// TestSSHRemoteExecutor_GetInstanceIP: getInstanceIP was an unimplemented stub that always failed, so
// every SSH-based template application (the path for any instance with a public IP) errored out.
func TestSSHRemoteExecutor_GetInstanceIP(t *testing.T) {
	sm := fakeStateManager{&types.State{Instances: map[string]types.Instance{
		"ws":      {Name: "ws", PublicIP: "203.0.113.7"},
		"private": {Name: "private"},
	}}}
	e := NewSSHRemoteExecutor("/tmp/key.pem", "ubuntu", sm)

	ip, err := e.getInstanceIP("ws")
	if err != nil || ip != "203.0.113.7" {
		t.Fatalf("getInstanceIP(ws) = %q, %v; want 203.0.113.7", ip, err)
	}
	if _, err := e.getInstanceIP("private"); err == nil || !strings.Contains(err.Error(), "no public IP") {
		t.Fatalf("instance without a public IP: err = %v", err)
	}
	if _, err := e.getInstanceIP("missing"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unknown instance: err = %v", err)
	}
	if _, err := NewSSHRemoteExecutor("/tmp/key.pem", "ubuntu", nil).getInstanceIP("ws"); err == nil {
		t.Fatal("no state manager: want an error, not a panic or an IP")
	}
}
