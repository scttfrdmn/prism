package types

import (
	"reflect"
	"testing"
)

func TestSSHConnection_Args(t *testing.T) {
	c := SSHConnection{User: "ubuntu", Host: "203.0.113.7", KeyPath: "/home/u/.ssh/prism key"}
	want := []string{"-i", "/home/u/.ssh/prism key", "--", "ubuntu@203.0.113.7"}
	if got := c.Args(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Args() = %q, want %q", got, want)
	}
	c.Port = 2222
	if got := c.Args(); got[0] != "-p" || got[1] != "2222" {
		t.Fatalf("non-default port not passed: %q", got)
	}
}

// Hostile values stay single, literal arguments: no shell parses them.
func TestSSHConnection_HostileValuesAreLiteralArgs(t *testing.T) {
	c := SSHConnection{User: "-oProxyCommand=touch /tmp/pwned", Host: "h;rm -rf ~", KeyPath: "$(touch /tmp/pwned)"}
	args := c.Args()
	if args[len(args)-2] != "--" {
		t.Fatalf("missing -- before destination: %q", args)
	}
	if args[1] != "$(touch /tmp/pwned)" || args[len(args)-1] != "-oProxyCommand=touch /tmp/pwned@h;rm -rf ~" {
		t.Fatalf("values were altered or split: %q", args)
	}
}

func TestSSHConnection_StringQuotesForDisplay(t *testing.T) {
	c := SSHConnection{User: "ubuntu", Host: "203.0.113.7", KeyPath: "/k/it's here"}
	if got, want := c.String(), `ssh -i '/k/it'\''s here' ubuntu@203.0.113.7`; got != want {
		t.Fatalf("String() = %s, want %s", got, want)
	}
	if got := (SSHConnection{User: "ubuntu", Host: "h"}).String(); got != "ssh ubuntu@h" {
		t.Fatalf("String() = %s", got)
	}
}

func TestSSHConnection_Validate(t *testing.T) {
	good := []SSHConnection{{User: "ubuntu", Host: "203.0.113.7"}, {User: "ec2-user", Host: "ip-10-0-0-1.ec2.internal", Port: 22}, {User: "rocky", Host: "2001:db8::1"}}
	for _, c := range good {
		if err := c.Validate(); err != nil {
			t.Errorf("%+v: %v", c, err)
		}
	}
	bad := []SSHConnection{
		{User: "-oProxyCommand=x", Host: "h"}, {User: "u;id", Host: "h"}, {User: "u", Host: "h & calc"},
		{User: "u", Host: "-oProxyCommand=x"}, {User: "u", Host: "h|x"}, {User: "u", Host: "h", Port: 70000},
		{User: "u", Host: "h", KeyPath: "a\nb"}, {User: "", Host: "h"},
	}
	for _, c := range bad {
		if err := c.Validate(); err == nil {
			t.Errorf("%+v should be rejected", c)
		}
	}
}

func TestSSHConnection_ShellCommandKeepsSeparatorAndQuotes(t *testing.T) {
	c := SSHConnection{User: "ubuntu", Host: "203.0.113.7", Port: 2222, KeyPath: "/k/my key"}
	if got, want := c.ShellCommand(), "ssh -p 2222 -i '/k/my key' -- ubuntu@203.0.113.7"; got != want {
		t.Fatalf("ShellCommand() = %s, want %s", got, want)
	}
}
