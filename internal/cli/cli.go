// Package cli implements the sb command (sb(1)). It is a thin dispatcher: every
// command in the manual's SYNOPSIS exists, and a command whose ticket hasn't
// landed exits with ENOTIMPL naming that ticket. Nothing pretends to succeed.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"example.com/second-brain/internal/adapters/placement/local"
	"example.com/second-brain/internal/adapters/registry/localfs"
	"example.com/second-brain/internal/core/apply"
	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/plan"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/errs"
	"example.com/second-brain/internal/version"
)

// Globals are the options accepted before the command (sb(1) OPTIONS).
type Globals struct {
	JSON, Quiet      bool
	Tenant, Endpoint string
}

type env struct {
	g        Globals
	out, err io.Writer
}

// planned maps "command sub" to the ticket that builds it.
var planned = map[string]string{
	"tenant verify": "SB-601",
	"kill agent":    "SB-403", "kill connector": "SB-403", "kill tenant": "SB-403", "kill list": "SB-403",
	"unkill agent": "SB-403", "unkill connector": "SB-403", "unkill tenant": "SB-403",
	"audit verify": "SB-402", "audit tail": "SB-402", "audit export": "SB-402",
	"release status": "SB-507", "release promote": "SB-507", "release rollback": "SB-508",
	"eval diff":       "SB-501",
	"factory handoff": "SB-705", "factory detach": "SB-708",
}

const usage = `usage: sb [--json] [--tenant NAME] [--endpoint URL] [-q] COMMAND [ARGS]

commands:
  tenant  check|apply|verify FILE      validate, apply and verify a tenant manifest
  kill    agent|connector|tenant ...   set a kill switch
  unkill  agent|connector|tenant ...   clear a kill switch
  audit   verify|tail|export           audit log tools
  release status|promote|rollback      release controller (maintainers)
  eval    run|diff SUITE               evaluation suites
  factory status|handoff|detach        custody and hand-off (factory operators)
  version                              print the version

See sb(1). Exit codes are listed in sb-errors(7).
`

// Run executes sb with args (without the program name) and returns the exit status.
func Run(args []string, stdout, stderr io.Writer) int {
	e := &env{out: stdout, err: stderr}
	rest, err := parseGlobals(args, &e.g)
	if err != nil {
		return e.fail("sb", err)
	}
	if len(rest) == 0 || rest[0] == "-h" || rest[0] == "--help" || rest[0] == "help" {
		fmt.Fprint(stdout, usage)
		if len(rest) == 0 {
			return 2
		}
		return 0
	}
	cmd := rest[0]
	switch cmd {
	case "version":
		return e.version()
	case "eval":
		if len(rest) > 1 && rest[1] == "run" {
			return e.evalRun(rest[2:])
		}
	}
	if len(rest) < 2 {
		if hasSub(cmd) {
			return e.fail("sb "+cmd, errs.New(errs.EUSAGE, "missing subcommand; see sb %s --help", cmd))
		}
		return e.fail("sb", errs.New(errs.EUSAGE, "unknown command %q; see sb --help", cmd))
	}
	name := cmd + " " + rest[1]
	switch name {
	case "tenant check":
		return e.tenantCheck(rest[2:])
	case "tenant apply":
		return e.tenantApply(rest[2:])
	case "factory status":
		return e.factoryStatus(rest[2:])
	}
	if t, ok := planned[name]; ok {
		return e.fail("sb "+name, errs.New(errs.ENOTIMPL, "not built yet (%s)", t))
	}
	if hasSub(cmd) {
		return e.fail("sb "+cmd, errs.New(errs.EUSAGE, "unknown subcommand %q; see sb %s --help", rest[1], cmd))
	}
	return e.fail("sb", errs.New(errs.EUSAGE, "unknown command %q; see sb --help", cmd))
}

func hasSub(cmd string) bool {
	if cmd == "tenant" {
		return true
	}
	for k := range planned {
		if strings.HasPrefix(k, cmd+" ") {
			return true
		}
	}
	return cmd == "eval"
}

func parseGlobals(args []string, g *Globals) ([]string, error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--json":
			g.JSON = true
		case a == "-q" || a == "--quiet":
			g.Quiet = true
		case a == "--tenant" || a == "--endpoint":
			if i+1 >= len(args) {
				return nil, errs.New(errs.EUSAGE, "%s needs a value", a)
			}
			if a == "--tenant" {
				g.Tenant = args[i+1]
			} else {
				g.Endpoint = args[i+1]
			}
			i++
		case strings.HasPrefix(a, "-") && a != "-h" && a != "--help":
			return nil, errs.New(errs.EUSAGE, "unknown option %s", a)
		default:
			return args[i:], nil
		}
	}
	return nil, nil
}

// localArgs takes --json after the command too (sb-tenant(1) OPTIONS), and returns the rest.
func (e *env) localArgs(args []string) []string {
	var rest []string
	for _, a := range args {
		if a == "--json" {
			e.g.JSON = true
			continue
		}
		rest = append(rest, a)
	}
	return rest
}

// readManifest reads and checks FILE. If it is invalid, it prints the errors as `check`
// does and returns ok=false with the exit status.
func (e *env) readManifest(name, file string) (t manifest.Tenant, code int, ok bool) {
	src, err := os.ReadFile(file)
	if err != nil {
		return t, e.fail(name, errs.New(errs.EUSAGE, "cannot read %s", file)), false
	}
	t, ds := manifest.Check(src)
	if len(ds) == 0 {
		return t, 0, true
	}
	if e.g.JSON {
		e.json(map[string]any{"file": file, "errors": ds})
		return t, errs.Exit(errs.New(errs.EMANIFEST, "%s", manifest.Count(len(ds)))), false
	}
	for _, d := range ds {
		fmt.Fprintln(e.out, d.Format(file))
	}
	fmt.Fprintln(e.out, manifest.Count(len(ds)))
	return t, e.fail(name, errs.New(errs.EMANIFEST, "%s is invalid", file)), false
}

// tenantCheck validates a manifest file (sb-tenant(1) check). It needs no network.
func (e *env) tenantCheck(args []string) int {
	const name = "sb tenant check"
	args = e.localArgs(args)
	if len(args) != 1 {
		return e.fail(name, errs.New(errs.EUSAGE, "usage: sb tenant check FILE"))
	}
	if _, code, ok := e.readManifest(name, args[0]); !ok {
		return code
	}
	if e.g.JSON {
		return e.json(map[string]any{"file": args[0], "errors": []manifest.Diag{}})
	}
	fmt.Fprintf(e.out, "%s: ok\n", args[0])
	return 0
}

// tenantApply implements `sb tenant apply FILE [--plan]` (SB-102, SB-104).
func (e *env) tenantApply(args []string) int {
	const name = "sb tenant apply"
	var file string
	planOnly := false
	for _, a := range e.localArgs(args) {
		switch {
		case a == "--plan":
			planOnly = true
		case file == "" && !strings.HasPrefix(a, "-"):
			file = a
		default:
			return e.fail(name, errs.New(errs.EUSAGE, "usage: sb tenant apply FILE [--plan]"))
		}
	}
	if file == "" {
		return e.fail(name, errs.New(errs.EUSAGE, "usage: sb tenant apply FILE [--plan]"))
	}
	t, code, ok := e.readManifest(name, file)
	if !ok {
		return code
	}
	deps, err := e.controlPlane()
	if err != nil {
		return e.fail(name, err)
	}
	ctx := context.Background()
	if planOnly {
		applied, found, err := deps.Registry.Applied(ctx, t.Tenant)
		if err != nil {
			// Never fall back to "no cell": that would plan a second cell (sb-tenant(1)).
			return e.fail(name, errs.New(errs.EUNAVAIL, "tenant registry unavailable: %v", err))
		}
		p, err := plan.Make(t, applied, found)
		if err != nil {
			return e.fail(name, errs.New(errs.EMANIFEST, "%s: %v", file, err))
		}
		if e.g.JSON {
			return e.json(p)
		}
		fmt.Fprint(e.out, p.Text())
		return 0
	}
	p, err := apply.Run(ctx, deps, t, func(c plan.Change) {
		if !e.g.JSON {
			fmt.Fprintln(e.out, c.Line())
		}
	})
	var ie *plan.ImmutableError
	switch {
	case errors.As(err, &ie):
		return e.fail(name, errs.New(errs.EMANIFEST, "%s: %v", file, err))
	case err != nil:
		return e.fail(name, errs.New(errs.EUNAVAIL, "%v; run apply again to resume", err))
	case e.g.JSON:
		return e.json(p)
	case len(p.Changes) == 0:
		fmt.Fprintln(e.out, "apply: no changes")
	default:
		fmt.Fprintf(e.out, "apply: %d added, %d changed, %d removed\n", p.Add, p.Change, p.Remove)
	}
	return 0
}

// factoryStatus prints a cell's custody record (sb-factory(7)); it changes nothing.
func (e *env) factoryStatus(args []string) int {
	const name = "sb factory status"
	args = e.localArgs(args)
	if len(args) != 1 {
		return e.fail(name, errs.New(errs.EUSAGE, "usage: sb factory status TENANT"))
	}
	deps, err := e.controlPlane()
	if err != nil {
		return e.fail(name, err)
	}
	cells, ok := deps.Cells.(ports.Cells)
	if !ok {
		return e.fail(name, errs.New(errs.EUNAVAIL, "this control plane cannot read cell records"))
	}
	ctx := context.Background()
	c, found, err := cells.CellByTenant(ctx, args[0])
	switch {
	case err != nil:
		return e.fail(name, errs.New(errs.EUNAVAIL, "tenant registry unavailable: %v", err))
	case !found:
		return e.fail(name, errs.New(errs.EUSAGE, "no cell for tenant %q", args[0]))
	}
	channel := "unknown"
	if rs, found, err := deps.Registry.Applied(ctx, c.Tenant); err == nil && found {
		for _, r := range rs {
			if r.Kind == plan.KindSetting && r.Name == "channel" {
				channel = r.Attrs["value"]
			}
		}
	}
	if e.g.JSON {
		return e.json(map[string]string{"tenant": c.Tenant, "custody": c.Custody, "placement": c.Placement,
			"key_custodian": c.KeyCustodian, "channel": channel, "handoff": "none"})
	}
	for _, kv := range [][2]string{{"custody", c.Custody}, {"placement", c.Placement}, {"key_custodian", c.KeyCustodian},
		{"channel", channel}, {"handoff", "none"}} {
		fmt.Fprintf(e.out, "%-14s %s\n", kv[0], kv[1])
	}
	return 0
}

// controlPlane opens the control plane named by --endpoint or SB_ENDPOINT (sb(1)).
// file:///DIR is the local one: DIR/tenants, DIR/accounts, DIR/journal (ADR-0018).
func (e *env) controlPlane() (apply.Deps, error) {
	ep := e.g.Endpoint
	if ep == "" {
		ep = os.Getenv("SB_ENDPOINT")
	}
	switch {
	case ep == "":
		return apply.Deps{}, errs.New(errs.EUNAVAIL, "no control-plane endpoint; pass --endpoint or set SB_ENDPOINT (file:///DIR for a local control plane)")
	case strings.HasPrefix(ep, "file://"):
		root := strings.TrimPrefix(ep, "file://")
		reg := localfs.Dir{Root: root}
		// v1 custody is always placement factory-account (sb-factory(7)); no account id here.
		acct := local.Account{Root: filepath.Join(root, "accounts"), Name: cell.PlacementFactoryAccount}
		return apply.Deps{Registry: reg, Writer: reg, Journal: reg, Cells: reg, BaseDomain: "sb.localhost", Placement: acct, Keys: acct}, nil
	}
	return apply.Deps{}, errs.New(errs.ENOTIMPL, "control-plane client not built yet (SB-105); use --endpoint file:///DIR")
}

func (e *env) version() int {
	if e.g.JSON {
		return e.json(map[string]string{"version": version.Version, "commit": version.Commit})
	}
	fmt.Fprintf(e.out, "sb %s (commit %s)\n", version.Version, version.Commit)
	return 0
}

// evalRun delegates to the Python eval runner (SB-015). In W0 only --model mock exists.
func (e *env) evalRun(args []string) int {
	root, ok := findRepo()
	if !ok {
		return e.fail("sb eval run", errs.New(errs.EUNAVAIL, "eval runner not found; run from inside the repository"))
	}
	c := exec.Command("python3", append([]string{"-m", "evals", "run"}, args...)...)
	c.Dir, c.Stdout, c.Stderr = root, e.out, e.err
	c.Env = append(os.Environ(), "PYTHONPATH="+root)
	if err := c.Run(); err != nil {
		if x, ok := err.(*exec.ExitError); ok {
			return x.ExitCode()
		}
		return e.fail("sb eval run", errs.New(errs.EUNAVAIL, "python3 not available: %v", err))
	}
	return 0
}

func findRepo() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "evals", "__main__.py")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func (e *env) json(v any) int {
	b, _ := json.Marshal(v)
	fmt.Fprintln(e.out, string(b))
	return 0
}

// fail prints err in human or JSON form and returns its exit status.
func (e *env) fail(prefix string, err error) int {
	c := errs.CodeOf(err)
	msg := err.Error()
	if x, ok := err.(*errs.Error); ok {
		msg = x.Msg
	}
	if e.g.JSON {
		b, _ := json.Marshal(map[string]any{"error": map[string]string{"code": string(c), "message": msg, "remedy": errs.Remedy(c)}})
		fmt.Fprintln(e.out, string(b))
	} else {
		fmt.Fprintf(e.err, "%s: %s [%s]\n", prefix, msg, c)
	}
	return errs.Exit(err)
}
