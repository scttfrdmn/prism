//go:build integration && linux
// +build integration,linux

package chaos

import "syscall"

// systemMemory reports total and free system RAM in bytes. syscall.Sysinfo is
// Linux-only; see sysmem_other_test.go for other platforms.
func systemMemory() (total, free uint64, ok bool) {
	var sysinfo syscall.Sysinfo_t
	if err := syscall.Sysinfo(&sysinfo); err != nil {
		return 0, 0, false
	}
	unit := uint64(sysinfo.Unit)
	return uint64(sysinfo.Totalram) * unit, uint64(sysinfo.Freeram) * unit, true
}
