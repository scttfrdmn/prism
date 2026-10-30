//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// detachProcess starts cmd in its own process group, so the daemon isn't killed when the GUI exits.
func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
