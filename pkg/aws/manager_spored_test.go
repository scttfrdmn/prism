package aws

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	ctypes "github.com/scttfrdmn/prism/pkg/types"
)

// TestAddSporedInstallToUserData verifies that the spored installation script
// injected into EC2 UserData contains all required security and configuration
// elements. Changes to the script should be intentional and reflected here.
func TestAddSporedInstallToUserData(t *testing.T) {
	const baseUserData = "#!/bin/bash\nset -e\necho 'base script'\n"

	result := addSporedInstallToUserData(baseUserData)

	t.Run("preserves_base_user_data", func(t *testing.T) {
		assert.True(t, strings.HasPrefix(result, baseUserData),
			"base UserData must be preserved at the start of output")
	})

	t.Run("appends_to_base", func(t *testing.T) {
		assert.Greater(t, len(result), len(baseUserData),
			"spored install script must be appended")
	})

	t.Run("downloads_from_correct_s3_bucket", func(t *testing.T) {
		assert.Contains(t, result, "spawn-binaries-",
			"must reference the spawn-binaries S3 bucket family")
		assert.Contains(t, result, "prism/spored-linux-",
			"must use the prism/ key prefix for the binary")
	})

	t.Run("detects_architecture", func(t *testing.T) {
		assert.Contains(t, result, "uname -m",
			"must detect instance architecture")
		assert.Contains(t, result, "amd64",
			"must handle x86_64 → amd64 mapping")
		assert.Contains(t, result, "arm64",
			"must handle aarch64 → arm64 mapping")
	})

	t.Run("detects_region", func(t *testing.T) {
		assert.Contains(t, result, "169.254.169.254",
			"must query IMDS for instance region")
	})

	t.Run("checksum_verification_present", func(t *testing.T) {
		assert.Contains(t, result, "sha256sum -c",
			"must verify binary checksum (#591)")
		assert.Contains(t, result, ".sha256",
			"must download SHA256 checksum file")
	})

	t.Run("removes_binary_on_checksum_failure", func(t *testing.T) {
		assert.Contains(t, result, "rm -f /usr/local/bin/spored",
			"must remove binary when checksum verification fails (#591)")
	})

	t.Run("systemd_service_unit_present", func(t *testing.T) {
		assert.Contains(t, result, "spored.service",
			"must create systemd service unit")
		assert.Contains(t, result, "systemctl enable spored",
			"must enable service for auto-start")
		assert.Contains(t, result, "systemctl start spored",
			"must start service immediately")
	})

	t.Run("tag_prefix_configured", func(t *testing.T) {
		assert.Contains(t, result, "SPORED_TAG_PREFIX=prism",
			"must configure prism tag namespace so spored reads prism:* EC2 tags")
	})

	t.Run("dns_domain_configured", func(t *testing.T) {
		assert.Contains(t, result, "SPORED_DNS_DOMAIN=prismcloud.host",
			"must configure prismcloud.host as the DNS domain")
	})

	t.Run("privilege_restriction_enforced", func(t *testing.T) {
		assert.Contains(t, result, "NoNewPrivileges=true",
			"systemd unit must prevent privilege escalation")
	})

	t.Run("file_descriptor_limit_set", func(t *testing.T) {
		assert.Contains(t, result, "LimitNOFILE=",
			"systemd unit must set file descriptor limit")
	})

	t.Run("restart_on_failure_configured", func(t *testing.T) {
		assert.Contains(t, result, "Restart=on-failure",
			"systemd unit must restart spored if it crashes")
	})
}

// TestAddSporedInstallToUserDataIdempotency verifies the function works correctly
// with an empty base script and doesn't depend on any particular content.
func TestAddSporedInstallToUserDataIdempotency(t *testing.T) {
	empty := ""
	result := addSporedInstallToUserData(empty)
	assert.NotEmpty(t, result, "should produce output even with empty input")
	assert.Contains(t, result, "spored", "output must contain spored install")

	// Calling twice (different inputs) should produce predictable results
	withBase := addSporedInstallToUserData("#!/bin/bash\n")
	assert.True(t, strings.HasSuffix(withBase, result),
		"script content should be the same regardless of base input")
}

// TestAddParamSweepExporterToUserData verifies the parameter-sweep exporter snippet
// reads the prism-namespaced tags and writes PARAM_*/SWEEP_* to a profile.d file.
func TestAddParamSweepExporterToUserData(t *testing.T) {
	const base = "#!/bin/bash\necho base\n"
	result := addParamSweepExporterToUserData(base)

	assert.True(t, strings.HasPrefix(result, base), "base UserData must be preserved")
	assert.Contains(t, result, "prism:param:", "must read prism:param:* tags (not spawn:param:*)")
	assert.Contains(t, result, "prism:sweep-id", "must read the sweep-id tag")
	assert.Contains(t, result, "/etc/profile.d/prism-params.sh", "must write the profile.d exporter file")
	assert.Contains(t, result, "export PARAM_", "must export PARAM_<key> env vars")
	assert.Contains(t, result, "export SWEEP_INDEX", "must export SWEEP_INDEX")
	assert.NotContains(t, result, "spawn:param:", "must NOT read spawn: namespace (Prism uses prism:)")
}

// TestProcessUserData_ParamExporterOnlyForSweep confirms the PARAM_* exporter is
// appended only when the request is a sweep member, so non-sweep launches are
// byte-identical to before.
func TestProcessUserData_ParamExporterOnlyForSweep(t *testing.T) {
	p := &UserDataProcessor{manager: &Manager{region: "us-west-2"}, region: "us-west-2"}
	tmpl := &ctypes.RuntimeTemplate{UserData: "#!/bin/bash\necho hi\n"}

	// Non-sweep launch: no exporter.
	plain := decodeUserData(t, p.ProcessUserData(tmpl, ctypes.LaunchRequest{Name: "plain", Template: "python-ml"}))
	assert.NotContains(t, plain, "/etc/profile.d/prism-params.sh",
		"non-sweep launch must not carry the param exporter")

	// Sweep member: exporter present.
	sweep := decodeUserData(t, p.ProcessUserData(tmpl, ctypes.LaunchRequest{
		Name: "hp-0", Template: "python-ml", SweepID: "hp-x", SweepIndex: 0, SweepSize: 2,
	}))
	assert.Contains(t, sweep, "/etc/profile.d/prism-params.sh",
		"sweep member must carry the param exporter")
}

// decodeUserData reverses ProcessUserData's gzip+base64 so tests can assert on the
// shell script content.
func decodeUserData(t *testing.T, encoded string) string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		// Not gzipped (fallback path) — return as-is.
		return string(raw)
	}
	defer zr.Close()
	out, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("gzip read: %v", err)
	}
	return string(out)
}
