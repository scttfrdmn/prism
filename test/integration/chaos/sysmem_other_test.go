//go:build integration && !linux
// +build integration,!linux

package chaos

// systemMemory is unavailable off Linux (syscall.Sysinfo is Linux-only).
func systemMemory() (total, free uint64, ok bool) {
	return 0, 0, false
}
