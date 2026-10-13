// Package sshhostkeys verifies the SSH host keys of Prism instances (#739).
//
// Every ssh invocation used to pass StrictHostKeyChecking=no and UserKnownHostsFile=/dev/null, so a
// host key was never checked or remembered: anyone in the network path, or holding a recycled public
// IP, could impersonate an instance and receive Jupyter tokens, template scripts, files and port
// forwards.
//
// Keys are recorded on first use (StrictHostKeyChecking=accept-new) and checked on every later
// connection, in a Prism-owned known_hosts file. They are keyed by instance ID through HostKeyAlias,
// not by IP: EC2 reuses public IPs, so an IP-keyed entry would falsely reject the next instance that
// gets the address, while an ID-keyed entry belongs to exactly one instance for its whole life.
package sshhostkeys

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// KnownHostsPath is Prism's ssh known_hosts file, beside the daemon state
// (PRISM_STATE_DIR, else ~/.prism).
func KnownHostsPath() string {
	dir := os.Getenv("PRISM_STATE_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.TempDir()
		}
		dir = filepath.Join(home, ".prism")
	}
	return filepath.Join(dir, "ssh_known_hosts")
}

// alias is the host name an instance's key is stored under.
func alias(instanceID string) string { return "prism-" + instanceID }

// Options returns the ssh "-o" arguments that verify the host key of the given instance. With an
// empty instanceID the key is keyed by whatever host ssh connects to (still verified, but exposed to
// IP reuse), so callers should pass the ID.
func Options(instanceID string) []string {
	path := KnownHostsPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o700) // ssh creates the file but not its directory; it reports any failure itself
	opts := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile=" + path,
		"-o", "HashKnownHosts=no", // keep aliases readable so Forget can remove them
	}
	if instanceID != "" {
		opts = append(opts, "-o", "HostKeyAlias="+alias(instanceID))
	}
	return opts
}

// Forget removes the recorded host key of an instance, e.g. when it is terminated. A missing file
// or entry is not an error.
func Forget(instanceID string) error {
	if instanceID == "" {
		return nil
	}
	path := KnownHostsPath()
	data, err := os.ReadFile(path) //nolint:gosec // G304: Prism's own known_hosts file under the state dir
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	host := alias(instanceID)
	var kept bytes.Buffer
	removed := false
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) > 0 && hostListContains(fields[0], host) {
			removed = true
			continue
		}
		kept.WriteString(line)
		kept.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if !removed {
		return nil
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, kept.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp) // best effort: the rename error is the one to report
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}

// hostListContains reports whether a known_hosts host field ("h1,h2,[h3]:22") names host.
func hostListContains(field, host string) bool {
	for _, h := range strings.Split(field, ",") {
		if h == host {
			return true
		}
	}
	return false
}
