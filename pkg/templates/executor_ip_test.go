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
		"ws":      {Name: "ws", ID: "i-0ws", PublicIP: "203.0.113.7"},
		"private": {Name: "private"},
	}}}
	e := NewSSHRemoteExecutor("/tmp/key.pem", "ubuntu", sm)

	ip, id, err := e.getInstanceIP("ws")
	if err != nil || ip != "203.0.113.7" || id != "i-0ws" {
		t.Fatalf("getInstanceIP(ws) = %q, %q, %v; want 203.0.113.7, i-0ws", ip, id, err)
	}
	if _, _, err := e.getInstanceIP("private"); err == nil || !strings.Contains(err.Error(), "no public IP") {
		t.Fatalf("instance without a public IP: err = %v", err)
	}
	if _, _, err := e.getInstanceIP("missing"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unknown instance: err = %v", err)
	}
	if _, _, err := NewSSHRemoteExecutor("/tmp/key.pem", "ubuntu", nil).getInstanceIP("ws"); err == nil {
		t.Fatal("no state manager: want an error, not a panic or an IP")
	}
}

// TestScriptCommand_DelimiterNotInScript: a script containing the heredoc delimiter on a line of its
// own used to end the heredoc early (fixed 'EOF'); the delimiter must never occur in the script.
func TestScriptCommand_DelimiterNotInScript(t *testing.T) {
	for _, script := range []string{
		"echo hi",
		"cat <<EOF\nnested\nEOF\necho after",
		"echo PRISM_SCRIPT_EOF\necho still inside",
	} {
		cmd := scriptCommand(script)
		first := strings.SplitN(cmd, "\n", 2)[0] // cat > /tmp/prism-script.sh << '<delim>'
		delim := strings.TrimSuffix(strings.SplitN(first, "<< '", 2)[1], "'")
		if strings.Contains(script, delim) {
			t.Fatalf("delimiter %q occurs in script %q", delim, script)
		}
		if !strings.Contains(cmd, "\n"+script+"\n"+delim+"\n") {
			t.Fatalf("script not wrapped intact:\n%s", cmd)
		}
	}
}
