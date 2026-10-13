package types

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

// SSHConnection is the structured SSH endpoint of an instance. Clients run ssh with Args() as an
// argument vector -- no shell ever parses it, so nothing in the instance state (user, host, key
// path) can be interpreted as shell syntax (#740).
type SSHConnection struct {
	User    string `json:"user"`
	Host    string `json:"host"`
	Port    int    `json:"port,omitempty"`     // 0 means the ssh default (22)
	KeyPath string `json:"key_path,omitempty"` // private key; empty uses the ssh agent/defaults
}

// Args returns the ssh arguments for an interactive session: options, then "--" so that a user or
// host beginning with "-" can't be taken as an option, then user@host.
func (c SSHConnection) Args() []string {
	var args []string
	if c.Port != 0 && c.Port != 22 {
		args = append(args, "-p", strconv.Itoa(c.Port))
	}
	if c.KeyPath != "" {
		args = append(args, "-i", c.KeyPath)
	}
	return append(args, "--", c.User+"@"+c.Host)
}

// String renders the equivalent ssh command line for display, quoting any argument a POSIX shell
// would otherwise split or interpret. It is for people to read or copy, never for execution.
func (c SSHConnection) String() string {
	parts := []string{"ssh"}
	for _, a := range c.Args() {
		if a == "--" {
			continue // implicit when shown to a person; Args keeps it for execution
		}
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

// ShellCommand renders the ssh command for the rare places a shell is unavoidable -- opening a
// terminal emulator, which runs its argument through a shell. Every argument is quoted and "--" is
// kept. Call Validate first: quoting covers POSIX shells, Validate also covers cmd.exe.
func (c SSHConnection) ShellCommand() string {
	parts := []string{"ssh"}
	for _, a := range c.Args() {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

var (
	sshUserPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]{0,31}$`)
	sshHostPattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]{0,252}[A-Za-z0-9])?$`)
)

// Validate checks that the connection holds a conventional login name, a plain IP address or DNS
// hostname, and a valid port -- values with no shell (POSIX or cmd.exe) metacharacters.
func (c SSHConnection) Validate() error {
	if !sshUserPattern.MatchString(c.User) {
		return fmt.Errorf("invalid SSH username %q", c.User)
	}
	if net.ParseIP(c.Host) == nil && !sshHostPattern.MatchString(c.Host) {
		return fmt.Errorf("invalid SSH host %q", c.Host)
	}
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("invalid SSH port %d", c.Port)
	}
	if strings.ContainsAny(c.KeyPath, "\x00\n\r") {
		return fmt.Errorf("invalid SSH key path")
	}
	return nil
}

func shellQuote(s string) string {
	if s != "" && strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("@%+=:,./_-~", r))
	}) < 0 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
