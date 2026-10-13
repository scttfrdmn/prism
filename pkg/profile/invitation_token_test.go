package profile

import "testing"

// An invitation token comes from an externally supplied invitation and becomes a file name under
// the invitations/received directory, so it must be one plain path component.
func TestIsSafeFileComponent(t *testing.T) {
	for _, ok := range []string{"abc123", "dGVzdC10b2tlbg", "tok_en-1.v2"} {
		if !isSafeFileComponent(ok) {
			t.Errorf("%q should be accepted", ok)
		}
	}
	for _, bad := range []string{"", ".", "..", "../x", "a/b", `a\b`, "../../.ssh/authorized_keys", "x\x00y"} {
		if isSafeFileComponent(bad) {
			t.Errorf("%q should be rejected", bad)
		}
	}
}
