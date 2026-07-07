package daemon

import (
	"testing"

	"github.com/scttfrdmn/prism/pkg/seam"
)

func TestResolveSeamBackend(t *testing.T) {
	cases := []struct {
		env  string
		want seamBackendKind
	}{
		{"", seamBackendFile},          // default: standalone desktop
		{"file", seamBackendFile},      // anything unrecognized stays file-backed
		{"filestore", seamBackendFile}, //
		{"dynamodb", seamBackendDynamo},
		{"dynamo", seamBackendDynamo},
	}
	for _, c := range cases {
		t.Run("env="+c.env, func(t *testing.T) {
			t.Setenv("PRISM_SEAM_BACKEND", c.env)
			if got := resolveSeamBackend(); got != c.want {
				t.Fatalf("resolveSeamBackend()=%v, want %v", got, c.want)
			}
		})
	}
}

func TestSeamTablePrefix(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("PRISM_SEAM_TABLE_PREFIX", "")
		if got := seamTablePrefix(); got != "prism" {
			t.Fatalf("seamTablePrefix()=%q, want %q", got, "prism")
		}
	})
	t.Run("override", func(t *testing.T) {
		t.Setenv("PRISM_SEAM_TABLE_PREFIX", "acme")
		if got := seamTablePrefix(); got != "acme" {
			t.Fatalf("seamTablePrefix()=%q, want %q", got, "acme")
		}
	})
}

// initSeamManagers with the dynamo backend selected but no AWS config must fall back to the
// file-backed managers rather than returning an error — a daemon with degraded (local) persistence
// beats a daemon that won't start.
func TestInitSeamManagers_DynamoWithoutConfigFallsBackToFile(t *testing.T) {
	t.Setenv("PRISM_SEAM_BACKEND", "dynamodb")
	t.Setenv("PRISM_STATE_DIR", t.TempDir()) // isolate budget manager's ~/.prism writes
	t.Setenv("HOME", t.TempDir())            // isolate project/rbac/approval ~/.prism writes

	mgrs, err := initSeamManagers(nil)
	if err != nil {
		t.Fatalf("initSeamManagers(nil) with dynamo selected: %v", err)
	}
	if mgrs.project == nil || mgrs.budget == nil || mgrs.rbac == nil || mgrs.approval == nil {
		t.Fatalf("fallback produced incomplete managers: %+v", mgrs)
	}
}

// The file backend (the default) must produce a full, usable set of managers.
func TestInitSeamManagers_FileBackendDefault(t *testing.T) {
	t.Setenv("PRISM_SEAM_BACKEND", "")
	t.Setenv("PRISM_STATE_DIR", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	mgrs, err := initSeamManagers(nil)
	if err != nil {
		t.Fatalf("initSeamManagers(nil): %v", err)
	}
	if mgrs.project == nil || mgrs.budget == nil || mgrs.rbac == nil || mgrs.approval == nil {
		t.Fatalf("incomplete managers: %+v", mgrs)
	}
}

// seamScope is the zero Scope today; this pins that contract so the follow-up that makes it
// configurable is a deliberate, reviewed change rather than a silent one.
func TestSeamScopeIsZeroForNow(t *testing.T) {
	if got := seamScope(); got != (seam.Scope{}) {
		t.Fatalf("seamScope()=%+v, want zero Scope", got)
	}
}
