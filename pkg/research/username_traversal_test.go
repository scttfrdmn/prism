package research

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The research username arrives from the HTTP path and becomes a file name
// (<config>/users/<profile>/<username>.json). Delete used to remove whatever .json file a
// traversal reached; Get/Delete now validate the username first.
func TestResearchUserManager_RejectsTraversalUsernames(t *testing.T) {
	configDir := t.TempDir()
	victim := filepath.Join(configDir, "victim.json")
	if err := os.WriteFile(victim, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	rum := NewResearchUserManager(nil, configDir)

	for _, name := range []string{"../../victim", "../victim", "a/b", `..\..\victim`, "."} {
		if err := rum.DeleteResearchUser("p1", name); err == nil || !strings.Contains(err.Error(), "invalid research username") {
			t.Errorf("Delete(%q): err = %v, want invalid research username", name, err)
		}
		if _, err := rum.GetResearchUser("p1", name); err == nil || !strings.Contains(err.Error(), "invalid research username") {
			t.Errorf("Get(%q): err = %v, want invalid research username", name, err)
		}
	}
	if _, err := os.Stat(victim); err != nil {
		t.Fatalf("victim file outside the users dir was removed: %v", err)
	}
}
