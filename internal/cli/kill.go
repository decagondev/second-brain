package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	auditfs "example.com/second-brain/internal/adapters/audit/localfs"
	killfs "example.com/second-brain/internal/adapters/kill/localfs"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/errs"
	"example.com/second-brain/internal/version"
)

// killRoot is the local control plane's directory (file:///DIR); a remote control plane
// needs its client (SB-105).
func (e *env) killRoot() (string, error) {
	ep := e.g.Endpoint
	if ep == "" {
		ep = os.Getenv("SB_ENDPOINT")
	}
	switch {
	case ep == "":
		return "", errs.New(errs.EUNAVAIL, "no control-plane endpoint; pass --endpoint or set SB_ENDPOINT (file:///DIR for a local control plane)")
	case strings.HasPrefix(ep, "file://"):
		return strings.TrimPrefix(ep, "file://"), nil
	}
	return "", errs.New(errs.ENOTIMPL, "control-plane client not built yet (SB-105); use --endpoint file:///DIR")
}

// actor is who runs sb: SB_ACTOR (a usr_ id) when set, else the local operator.
func actor() string {
	if a := os.Getenv("SB_ACTOR"); a != "" {
		return a
	}
	return "sys_cli_local"
}

// kill handles sb kill|unkill agent|connector|tenant (sb-kill(1)).
func (e *env) kill(verb string, args []string) int {
	name := "sb " + verb
	args = e.localArgs(args)
	if len(args) == 0 {
		return e.fail(name, errs.New(errs.EUSAGE, "usage: sb %s agent NAME|connector KIND|space NAME|tenant --tenant T --reason TEXT", verb))
	}
	if verb == "kill" && args[0] == "list" {
		return e.killList(args[1:])
	}
	what, rest := args[0], args[1:]
	var scope string
	switch what {
	case "agent", "connector", "space":
		if len(rest) == 0 || strings.HasPrefix(rest[0], "--") {
			return e.fail(name, errs.New(errs.EUSAGE, "sb %s %s needs a name", verb, what))
		}
		scope, rest = what+":"+rest[0], rest[1:]
	case "tenant":
		scope = "tenant"
		if len(rest) > 0 && !strings.HasPrefix(rest[0], "--") {
			rest = rest[1:] // sb unkill tenant [NAME]
		}
	default:
		return e.fail(name, errs.New(errs.EUSAGE, "unknown scope %q: agent, connector, space or tenant", what))
	}
	var reason string
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--reason":
			if i+1 >= len(rest) {
				return e.fail(name, errs.New(errs.EUSAGE, "--reason needs a value"))
			}
			reason = rest[i+1]
			i++
		case "--all-tenants":
			return e.fail(name, errs.New(errs.ENOTIMPL, "--all-tenants needs the control-plane client (SB-105)"))
		default:
			return e.fail(name, errs.New(errs.EUSAGE, "unknown option %q; see sb-kill(1)", rest[i]))
		}
	}
	if strings.TrimSpace(reason) == "" {
		return e.fail(name, errs.New(errs.EUSAGE, "--reason is required; it goes into the audit log"))
	}
	if e.g.Tenant == "" {
		return e.fail(name, errs.New(errs.EUSAGE, "--tenant is required"))
	}
	if !kill.ValidScope(scope) {
		return e.fail(name, errs.New(errs.EUSAGE, "invalid scope %q", scope))
	}
	root, err := e.killRoot()
	if err != nil {
		return e.fail(name, err)
	}
	log := &audit.Log{Tenant: e.g.Tenant, Release: func() string { return "cli-" + version.Version }, Store: auditfs.Dir{Root: root, Tenant: e.g.Tenant}, Now: time.Now}
	ctl := kill.Control{Store: killfs.Dir{Root: root}, Audit: log, Now: time.Now}
	ctx := context.Background()
	if verb == "kill" {
		err = ctl.Kill(ctx, e.g.Tenant, scope, reason, actor())
	} else {
		err = ctl.Unkill(ctx, e.g.Tenant, scope, reason, actor())
	}
	if err != nil {
		return e.fail(name, errs.New(errs.EUNAVAIL, "%v", err))
	}
	state := "killed"
	if verb == "unkill" {
		state = "cleared"
	}
	if e.g.JSON {
		e.json(map[string]any{"tenant": e.g.Tenant, "scope": scope, "state": state})
		return 0
	}
	fmt.Fprintf(e.out, "%s: %s %s for %s; cells see it within 10 seconds\n", verb, scope, state, e.g.Tenant)
	return 0
}

func (e *env) killList(args []string) int {
	const name = "sb kill list"
	if len(args) > 0 {
		return e.fail(name, errs.New(errs.EUSAGE, "usage: sb kill list --tenant T"))
	}
	if e.g.Tenant == "" {
		return e.fail(name, errs.New(errs.EUSAGE, "--tenant is required"))
	}
	root, err := e.killRoot()
	if err != nil {
		return e.fail(name, err)
	}
	list, err := killfs.Dir{Root: root}.List(context.Background(), e.g.Tenant)
	if err != nil {
		return e.fail(name, errs.New(errs.EUNAVAIL, "%v", err))
	}
	if e.g.JSON {
		e.json(map[string]any{"tenant": e.g.Tenant, "switches": list})
		return 0
	}
	if len(list) == 0 {
		fmt.Fprintf(e.out, "no kill switches set for %s\n", e.g.Tenant)
	}
	for _, s := range list {
		fmt.Fprintf(e.out, "%-22s %s  by %s: %s\n", s.Scope, s.At.Format(time.RFC3339), s.Actor, s.Reason)
	}
	return 0
}
