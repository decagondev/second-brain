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
	"time"

	accessfs "example.com/second-brain/internal/adapters/access/localfs"
	wrapfs "example.com/second-brain/internal/adapters/keywrap/localfs"
	"example.com/second-brain/internal/adapters/placement/local"
	"example.com/second-brain/internal/adapters/registry/localfs"
	"example.com/second-brain/internal/core/access"
	"example.com/second-brain/internal/core/apply"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/export"
	"example.com/second-brain/internal/core/keywrap"
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
	"tenant verify": "SB-601, which needs a deployed cell's verifier endpoint (W7); the local cell runs it: tools/evalcell --verify",
	"audit tail":    "SB-208", "audit export": "SB-208", // need a running cell; verify --export works offline
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
	case "kill", "unkill":
		return e.kill(cmd, rest[1:])
	case "release":
		if len(rest) > 1 && (rest[1] == "status" || rest[1] == "promote" || rest[1] == "rollback") {
			return e.release(rest[1], rest[2:])
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
	case "factory access":
		return e.factoryAccess(rest[2:])
	case "factory verify-export":
		return e.factoryVerifyExport(rest[2:])
	case "audit verify":
		return e.auditVerify(rest[2:])
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
	return cmd == "eval" || cmd == "kill" || cmd == "unkill" || cmd == "release"
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
	var ee *apply.EvalError
	switch {
	case errors.As(err, &ie):
		return e.fail(name, errs.New(errs.EMANIFEST, "%s: %v", file, err))
	case errors.As(err, &ee):
		return e.fail(name, errs.New(errs.EEVAL, "%v", err))
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
	wraps, orphaned := "unknown", false
	if deps.Wraps != nil {
		inv, err := keywrap.Inventory(ctx, c.Tenant, deps.Wraps, map[string]ports.Keys{cell.KeyCustodianFactory: deps.Keys})
		if err != nil {
			return e.fail(name, errs.New(errs.EUNAVAIL, "key-wrap inventory unavailable: %v", err))
		}
		wraps, orphaned = wrapLine(inv), keywrap.Orphaned(inv)
	}
	if e.g.JSON {
		code := e.json(map[string]string{"tenant": c.Tenant, "custody": c.Custody, "placement": c.Placement,
			"key_custodian": c.KeyCustodian, "key_wraps": wraps, "channel": channel, "handoff": "none"})
		if code == 0 && orphaned {
			return 1
		}
		return code
	}
	for _, kv := range [][2]string{{"custody", c.Custody}, {"placement", c.Placement}, {"key_custodian", c.KeyCustodian},
		{"key_wraps", wraps}, {"channel", channel}, {"handoff", "none"}} {
		fmt.Fprintf(e.out, "%-14s %s\n", kv[0], kv[1])
	}
	if orphaned {
		return e.fail(name, errs.New(errs.ELAW, "a data key has no working wrap (LAW-18)"))
	}
	return 0
}

// factoryAccess runs the access-path check (sb-factory(1), access; SB-706). On the local
// control plane the sources are the wrap inventory and DIR/access/TENANT.json, the local
// accounts' stand-in for IAM.
func (e *env) factoryAccess(args []string) int {
	const name = "sb factory access"
	args = e.localArgs(args)
	if len(args) != 1 {
		return e.fail(name, errs.New(errs.EUSAGE, "usage: sb factory access TENANT"))
	}
	deps, err := e.controlPlane()
	if err != nil {
		return e.fail(name, err)
	}
	cells, ok := deps.Cells.(ports.Cells)
	if !ok || deps.Wraps == nil || deps.AccessFile == nil {
		return e.fail(name, errs.New(errs.EUNAVAIL, "this control plane cannot enumerate access paths"))
	}
	ctx := context.Background()
	c, found, err := cells.CellByTenant(ctx, args[0])
	switch {
	case err != nil:
		return e.fail(name, errs.New(errs.EUNAVAIL, "tenant registry unavailable: %v", err))
	case !found:
		return e.fail(name, errs.New(errs.EUSAGE, "no cell for tenant %q", args[0]))
	case c.Custody != cell.CustodyFactory:
		return e.fail(name, errs.New(errs.EUNAVAIL, "the hand-off record store for %s custody arrives with sb factory handoff (SB-705)", c.Custody))
	}
	res, err := access.Check(ctx, c.Tenant, []access.Source{access.Wraps{Store: deps.Wraps}, deps.AccessFile}, nil)
	if err != nil {
		return e.fail(name, errs.New(errs.EUNAVAIL, "%v", err))
	}
	if e.g.JSON {
		return e.json(map[string]any{"tenant": c.Tenant, "custody": c.Custody, "paths": res.Paths, "law": "LAW-17 applies after hand-off"})
	}
	fmt.Fprintf(e.out, "custody %s: LAW-17 applies after hand-off; the factory's paths now:\n", c.Custody)
	for _, p := range res.Paths {
		fmt.Fprintf(e.out, "  %-12s %s\n", p.Kind, p.ID)
	}
	return 0
}

// factoryVerifyExport checks a detach's export package offline (sb-factory(1); SB-708).
func (e *env) factoryVerifyExport(args []string) int {
	const name = "sb factory verify-export"
	if len(args) != 1 {
		return e.fail(name, errs.New(errs.EUSAGE, "usage: sb factory verify-export DIR"))
	}
	rep, err := export.Verify(context.Background(), args[0])
	switch {
	case errors.Is(err, export.ErrChain):
		return e.fail(name, errs.New(errs.ECHAIN, "%v", err))
	case errors.Is(err, export.ErrBundle):
		return e.fail(name, errs.New(errs.EUNSIGNED, "%v", err))
	case err != nil:
		return e.fail(name, errs.New(errs.ELAW, "%v", err))
	}
	if e.g.JSON {
		return e.json(rep)
	}
	fmt.Fprintf(e.out, "files     %d listed, all present and unaltered, nothing extra\n", rep.Files)
	fmt.Fprintf(e.out, "audit     %d rows to seq %d, verified with the pinned anchor key\n", rep.Rows, rep.LastSeq)
	fmt.Fprintf(e.out, "release   %s, signature and gate verified\n", rep.Release)
	fmt.Fprintf(e.out, "verify-export: %s passed\n", rep.Tenant)
	return 0
}

// wrapLine prints the inventory, e.g. "audit-signing: factory, tenant (broken)".
func wrapLine(inv []keywrap.Entry) string {
	if len(inv) == 0 {
		return "none recorded"
	}
	var parts []string
	for _, en := range inv {
		cs := append([]string(nil), en.Working...)
		for _, b := range en.Broken {
			cs = append(cs, b+" (broken)")
		}
		parts = append(parts, en.Purpose+": "+strings.Join(cs, ", "))
	}
	return strings.Join(parts, "; ")
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
		return apply.Deps{Registry: reg, Writer: reg, Journal: reg, Cells: reg, BaseDomain: "sb.localhost", Placement: acct, Keys: acct,
			Evals: reg, Now: time.Now, Wraps: wrapfs.Dir{Root: filepath.Join(root, "keywraps")},
			AccessFile: accessfs.Dir{Root: filepath.Join(root, "access")}}, nil
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

// auditVerify is sb audit verify (sb-audit(1)). Offline with --export FILE; against a
// running cell it needs the cell runtime.
func (e *env) auditVerify(args []string) int {
	const name = "sb audit verify"
	var file, key string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--export", "--key", "--since":
			if i+1 >= len(args) {
				return e.fail(name, errs.New(errs.EUSAGE, "%s needs a value", args[i]))
			}
			switch args[i] {
			case "--export":
				file = args[i+1]
			case "--key":
				key = args[i+1]
			}
			i++
		default:
			return e.fail(name, errs.New(errs.EUSAGE, "unknown option %q; see sb-audit(1)", args[i]))
		}
	}
	if file == "" {
		return e.fail(name, errs.New(errs.ENOTIMPL, "verifying a running cell needs the cell runtime (SB-208); use --export FILE"))
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return e.fail(name, errs.New(errs.EUSAGE, "cannot read %s: %v", file, err))
	}
	tenant := e.g.Tenant
	if tenant == "" {
		tenant = firstTenant(b)
	}
	rep, err := audit.Verify(strings.NewReader(string(b)), tenant, key)
	if err != nil {
		return e.fail(name, errs.New(errs.ECHAIN, "%s", strings.TrimPrefix(err.Error(), "audit: ECHAIN: ")))
	}
	if e.g.JSON {
		out, _ := json.Marshal(map[string]any{"ok": true, "tenant": tenant, "rows": rep.Rows, "first_seq": rep.FirstSeq,
			"last_seq": rep.LastSeq, "anchors": rep.Anchors, "key": rep.Key, "key_pinned": rep.KeyPinned})
		fmt.Fprintln(e.out, string(out))
		return 0
	}
	fmt.Fprintf(e.out, "verify: %s: %d rows (seq %d to %d), %d anchors, chain intact\n", tenant, rep.Rows, rep.FirstSeq, rep.LastSeq, rep.Anchors)
	if !rep.KeyPinned {
		fmt.Fprintln(e.err, "verify: key not pinned: pass --key with the key from the cell's custody record")
	}
	return 0
}

// firstTenant is the tenant of the first row in an export, or "".
func firstTenant(b []byte) string {
	for _, l := range strings.Split(string(b), "\n") {
		var r struct {
			Tenant string `json:"tenant"`
			Anchor bool   `json:"anchor"`
		}
		if json.Unmarshal([]byte(l), &r) == nil && !r.Anchor && r.Tenant != "" {
			return r.Tenant
		}
	}
	return ""
}
