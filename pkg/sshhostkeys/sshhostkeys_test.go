package sshhostkeys

import (
	"os"
	"strings"
	"testing"
)

func TestOptionsVerifyAndKeyByInstanceID(t *testing.T) {
	t.Setenv("PRISM_STATE_DIR", t.TempDir())
	opts := strings.Join(Options("i-0abc"), " ")
	for _, want := range []string{"StrictHostKeyChecking=accept-new", "UserKnownHostsFile=" + KnownHostsPath(), "HostKeyAlias=prism-i-0abc", "HashKnownHosts=no"} {
		if !strings.Contains(opts, want) {
			t.Errorf("options %q missing %q", opts, want)
		}
	}
	for _, bad := range []string{"StrictHostKeyChecking=no", "/dev/null"} {
		if strings.Contains(opts, bad) {
			t.Errorf("options %q must not contain %q", opts, bad)
		}
	}
	if strings.Contains(strings.Join(Options(""), " "), "HostKeyAlias") {
		t.Error("no alias without an instance ID")
	}
}

func TestForgetRemovesOnlyThatInstance(t *testing.T) {
	t.Setenv("PRISM_STATE_DIR", t.TempDir())
	content := "prism-i-1 ssh-ed25519 AAAA1\nprism-i-2 ssh-ed25519 AAAA2\nprism-i-10 ssh-ed25519 AAAA10\nother,prism-i-1 ssh-rsa BBBB\n"
	if err := os.WriteFile(KnownHostsPath(), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Forget("i-1"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(KnownHostsPath())
	if want := "prism-i-2 ssh-ed25519 AAAA2\nprism-i-10 ssh-ed25519 AAAA10\n"; string(got) != want {
		t.Fatalf("after Forget(i-1):\n%s\nwant:\n%s", got, want)
	}
	if err := Forget("i-missing"); err != nil {
		t.Fatalf("forgetting an unknown instance: %v", err)
	}
}

func TestForgetWithoutFile(t *testing.T) {
	t.Setenv("PRISM_STATE_DIR", t.TempDir())
	if err := Forget("i-1"); err != nil {
		t.Fatalf("no known_hosts file yet: %v", err)
	}
}
