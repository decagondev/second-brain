package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func run(args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := Run(args, &o, &e)
	return code, o.String(), e.String()
}

func TestVersion(t *testing.T) {
	if c, out, _ := run("version"); c != 0 || !strings.HasPrefix(out, "sb 0.0.0-dev") {
		t.Fatalf("version: %d %q", c, out)
	}
	c, out, _ := run("--json", "version")
	var v map[string]string
	if c != 0 || json.Unmarshal([]byte(out), &v) != nil || v["version"] == "" {
		t.Fatalf("--json version: %d %q", c, out)
	}
}

// Every command in the sb(1) SYNOPSIS exists and exits ENOTIMPL (2) naming its ticket.
func TestPlannedCommandsAreHonest(t *testing.T) {
	for name, ticket := range planned {
		args := append(strings.Fields(name), "x")
		c, _, errOut := run(args...)
		if c != 2 || !strings.Contains(errOut, ticket) || !strings.Contains(errOut, "[ENOTIMPL]") {
			t.Errorf("%s: exit %d, stderr %q; want 2 and %s ENOTIMPL", name, c, errOut, ticket)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{{"nosuch"}, {"tenant"}, {"tenant", "nosuch"}, {"--bogus", "version"}, {"--tenant"}} {
		if c, _, errOut := run(args...); c != 2 || !strings.Contains(errOut, "[EUSAGE]") {
			t.Errorf("%v: exit %d %q; want 2 EUSAGE", args, c, errOut)
		}
	}
	if c, _, _ := run(); c != 2 {
		t.Error("no command must exit 2")
	}
	if c, out, _ := run("--help"); c != 0 || !strings.Contains(out, "usage: sb") {
		t.Error("--help must exit 0 with usage")
	}
}

func TestJSONErrors(t *testing.T) {
	c, out, _ := run("--json", "tenant", "verify", "a.yaml")
	var v struct {
		Error struct{ Code, Message, Remedy string }
	}
	if c != 2 || json.Unmarshal([]byte(out), &v) != nil || v.Error.Code != "ENOTIMPL" || v.Error.Remedy == "" {
		t.Fatalf("json error: %d %q", c, out)
	}
}

// The manual's SYNOPSIS and the dispatcher must agree (P2: the manual is the spec).
func TestSynopsisMatchesManual(t *testing.T) {
	for _, cmd := range []string{"tenant", "kill", "unkill", "audit", "release", "eval", "factory"} {
		if !hasSub(cmd) {
			t.Errorf("sb(1) lists %q but the CLI has no subcommands for it", cmd)
		}
	}
}
