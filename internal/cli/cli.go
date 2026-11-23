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
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	"example.com/second-brain/internal/core/verify"
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
var planned = map[string]string{}

const usage = `usage: sb [--json] [--tenant NAME] [--endpoint URL] [-q] COMMAND [ARGS]

commands:
  tenant  check|apply|verify FILE      validate, apply and verify a tenant manifest
  space   init|templates ...           start a department space from a template
  kill    agent|connector|space|tenant ...   set a kill switch
  unkill  agent|connector|space|tenant ...   clear a kill switch
  audit   verify|tail|export           audit log tools
  release status|promote|rollback      release controller (maintainers)
  eval    run|diff SUITE               evaluation suites
  factory status|handoff|detach        custody and hand-off (factory operators)
  selfhost doctor|verify-bundle|check-proxy|check-dsn   a self-hosted box's checks, bundles and settings (sb-selfhost(7))
  secrets scan [--known PATH]... PATH...   find stored secrets and secret shapes in files
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
		if len(rest) > 1 && (rest[1] == "run" || rest[1] == "gate" || rest[1] == "diff") {
			return e.evalPython(rest[1], rest[2:])
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
	case "tenant verify":
		return e.tenantVerify(rest[2:])
	case "factory status":
		return e.factoryStatus(rest[2:])
	case "factory access":
		return e.factoryAccess(rest[2:])
	case "factory verify-export":
		return e.factoryVerifyExport(rest[2:])
	case "audit verify":
		return e.auditVerify(rest[2:])
	case "audit tail":
		return e.auditTail(rest[2:])
	case "audit export":
		return e.auditExport(rest[2:])
	case "factory handoff":
		return e.factoryHandoff(rest[2:])
	case "factory detach":
		return e.factoryDetach(rest[2:])
	case "space init":
		return e.spaceInit(rest[2:])
	case "space templates":
		return e.spaceTemplates(rest[2:])
	case "space diff":
		return e.spaceDiff(rest[2:])
	}
	if code, ok := e.boxCommand(name, rest[2:]); ok {
		return code
	}
	if t, ok := planned[name]; ok {
		return e.fail("sb "+name, errs.New(errs.ENOTIMPL, "not built yet (%s)", t))
	}
	if hasSub(cmd) {
		return e.fail("sb "+cmd, errs.New(errs.EUSAGE, "unknown subcommand %q; see sb %s --help", rest[1], cmd))
	}
	return e.fail("sb", errs.New(errs.EUSAGE, "unknown command %q; see sb --help", cmd))
}

// boxCommand runs the commands for a self-hosted box (sb-selfhost(7)) and the secret
// scanner; ok is false for any other name.
func (e *env) boxCommand(name string, args []string) (code int, ok bool) {
	switch name {
	case "selfhost doctor":
		return e.selfhostDoctor(args), true
	case "selfhost verify-bundle":
		return e.selfhostVerifyBundle(args), true
	case "selfhost check-proxy":
		return e.selfhostCheckProxy(args), true
	case "selfhost check-dsn":
		return e.selfhostCheckDSN(args), true
	case "secrets scan":
		return e.secretsScan(args), true
	}
	return 0, false
}

func hasSub(cmd string) bool {
	if cmd == "tenant" || cmd == "space" || cmd == "selfhost" || cmd == "secrets" {
		return true
	}
	for k := range planned {
		if strings.HasPrefix(k, cmd+" ") {
			return true
		}
	}
	return cmd == "eval" || cmd == "kill" || cmd == "unkill" || cmd == "release" || cmd == "audit" || cmd == "factory"
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

// tenantVerify implements `sb tenant verify FILE --cell URL [--law LAW-nn]... [--sample N]`
// against a deployed cell's verifier endpoint (sb-tenant(1); SB-905).
func (e *env) tenantVerify(args []string) int {
	const name = "sb tenant verify"
	var file, cellURL string
	req := map[string]any{"laws": []string{}, "sample": 50, "seed": 1}
	laws := []string{}
	args = e.localArgs(args)
	for i := 0; i < len(args); i++ {
		a := args[i]
		next := func() (string, bool) {
			if i+1 >= len(args) {
				return "", false
			}
			i++
			return args[i], true
		}
		switch {
		case a == "--cell":
			v, ok := next()
			if !ok {
				return e.fail(name, errs.New(errs.EUSAGE, "--cell needs a URL"))
			}
			cellURL = v
		case a == "--law":
			v, ok := next()
			if !ok {
				return e.fail(name, errs.New(errs.EUSAGE, "--law needs a law, e.g. LAW-01"))
			}
			laws = append(laws, v)
		case a == "--foreign":
			v, ok := next()
			if !ok {
				return e.fail(name, errs.New(errs.EUSAGE, "--foreign needs a file"))
			}
			b, err := os.ReadFile(v)
			var ids []string
			if err != nil || json.Unmarshal(b, &ids) != nil {
				return e.fail(name, errs.New(errs.EUSAGE, "--foreign FILE must be a JSON list of document ids"))
			}
			req["foreign_ids"] = ids
		case a == "--sample":
			v, ok := next()
			n, err := strconv.Atoi(v)
			if !ok || err != nil || n < 1 || n > 1000 {
				return e.fail(name, errs.New(errs.EUSAGE, "--sample needs a number from 1 to 1000"))
			}
			req["sample"] = n
		case file == "" && !strings.HasPrefix(a, "-"):
			file = a
		default:
			return e.fail(name, errs.New(errs.EUSAGE, "usage: sb tenant verify FILE --cell URL [--law LAW-nn]... [--sample N] [--foreign FILE]"))
		}
	}
	if file == "" {
		return e.fail(name, errs.New(errs.EUSAGE, "usage: sb tenant verify FILE --cell URL [--law LAW-nn]... [--sample N] [--foreign FILE]"))
	}
	m, code, ok := e.readManifest(name, file)
	if !ok {
		return code
	}
	if cellURL == "" {
		return e.fail(name, errs.New(errs.EUSAGE, "--cell URL is required (the deployed cell); the local cell runs it with tools/evalcell --verify"))
	}
	req["laws"] = laws
	body, _ := json.Marshal(req)
	raw, err := cellCall(cellURL, "/v1/verifier/verify", "application/json", body, 10*time.Minute)
	if err != nil {
		return e.fail(name, err)
	}
	var rep verify.Report
	if err := json.Unmarshal(raw, &rep); err != nil {
		return e.fail(name, errs.New(errs.EUNAVAIL, "the verifier's report is unreadable"))
	}
	_ = m
	if e.g.JSON {
		if c := e.json(rep); c != 0 || rep.Passed() {
			return c
		}
		return 1
	}
	fmt.Fprint(e.out, rep.String())
	if !rep.Passed() {
		return 1
	}
	return 0
}

// cellCall posts body to a running cell's control-plane endpoint (cellDo).
func cellCall(cellURL, path, contentType string, body []byte, timeout time.Duration) ([]byte, error) {
	return cellDo(http.MethodPost, cellURL, path, contentType, body, timeout)
}

// tenantApply implements `sb tenant apply FILE [--plan] [--cell URL]` (SB-102, SB-104,
// SB-1506): after the control plane's apply, --cell sends the manifest to the running cell.
func (e *env) tenantApply(args []string) int {
	const name = "sb tenant apply"
	const usage = "usage: sb tenant apply FILE [--plan] [--cell URL]"
	var file, cellURL string
	planOnly := false
	args = e.localArgs(args)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--plan":
			planOnly = true
		case a == "--cell" && i+1 < len(args):
			i++
			cellURL = args[i]
		case file == "" && !strings.HasPrefix(a, "-"):
			file = a
		default:
			return e.fail(name, errs.New(errs.EUSAGE, usage))
		}
	}
	if file == "" || (planOnly && cellURL != "") {
		return e.fail(name, errs.New(errs.EUSAGE, usage))
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
	case cellURL != "":
		return e.applyToCell(name, file, cellURL)
	case e.g.JSON:
		return e.json(p)
	case len(p.Changes) == 0:
		fmt.Fprintln(e.out, "apply: no changes")
	default:
		fmt.Fprintf(e.out, "apply: %d added, %d changed, %d removed\n", p.Add, p.Change, p.Remove)
	}
	return 0
}

// applyToCell sends the checked manifest to the running cell (POST /v1/cell/manifest).
func (e *env) applyToCell(name, file, cellURL string) int {
	src, err := os.ReadFile(file)
	if err != nil {
		return e.fail(name, errs.New(errs.EUSAGE, "%s: %v", file, err))
	}
	raw, err := cellCall(cellURL, "/v1/cell/manifest", "application/yaml", src, time.Minute)
	if err != nil {
		return e.fail(name, err)
	}
	var res struct {
		SHA256  string   `json:"sha256"`
		Changed bool     `json:"changed"`
		Changes []string `json:"changes"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return e.fail(name, errs.New(errs.EUNAVAIL, "the cell's answer is unreadable"))
	}
	if e.g.JSON {
		return e.json(res)
	}
	for _, c := range res.Changes {
		fmt.Fprintln(e.out, c)
	}
	if !res.Changed {
		fmt.Fprintln(e.out, "apply: the cell already runs this manifest")
		return 0
	}
	fmt.Fprintf(e.out, "apply: the cell runs sha256:%s\n", res.SHA256)
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
	keys := map[string]ports.Keys{cell.KeyCustodianFactory: deps.Keys}
	phase := "none"
	if ctl, _, err := e.controller(c.Tenant); err == nil {
		if st, err := ctl.State(ctx, c.Tenant); err == nil && st.Plan != nil {
			phase = string(st.Phase)
			if provider, id, ok := cell.ParseTenantAccount(st.Plan.Target); ok && provider == "local" {
				root := strings.TrimPrefix(e.endpoint(), "file://")
				keys[cell.KeyCustodianTenant] = local.Account{Root: filepath.Join(root, "accounts"), Name: id}
			}
		}
	}
	wraps, orphaned := "unknown", false
	if deps.Wraps != nil {
		inv, err := keywrap.Inventory(ctx, c.Tenant, deps.Wraps, keys)
		if err != nil {
			return e.fail(name, errs.New(errs.EUNAVAIL, "key-wrap inventory unavailable: %v", err))
		}
		wraps, orphaned = wrapLine(inv), keywrap.Orphaned(inv)
	}
	if e.g.JSON {
		code := e.json(map[string]string{"tenant": c.Tenant, "custody": c.Custody, "placement": c.Placement,
			"key_custodian": c.KeyCustodian, "key_wraps": wraps, "channel": channel, "handoff": phase})
		if code == 0 && orphaned {
			return 1
		}
		return code
	}
	for _, kv := range [][2]string{{"custody", c.Custody}, {"placement", c.Placement}, {"key_custodian", c.KeyCustodian},
		{"key_wraps", wraps}, {"channel", channel}, {"handoff", phase}} {
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
	}
	var retain []string
	if c.Custody != cell.CustodyFactory {
		// After a hand-off the record is the plan both sides approved (sb-factory(1), access).
		ctl, _, err := e.controller(c.Tenant)
		if err != nil {
			return e.fail(name, err)
		}
		st, err := ctl.State(ctx, c.Tenant)
		if err != nil || st.Plan == nil {
			return e.fail(name, errs.New(errs.EUNAVAIL, "the hand-off record for %s can't be read", c.Tenant))
		}
		retain = st.Plan.Retain
	}
	res, err := access.Check(ctx, c.Tenant, []access.Source{access.Wraps{Store: deps.Wraps}, deps.AccessFile}, retain)
	if err != nil {
		return e.fail(name, errs.New(errs.EUNAVAIL, "%v", err))
	}
	if c.Custody != cell.CustodyFactory {
		if e.g.JSON {
			e.json(map[string]any{"tenant": c.Tenant, "custody": c.Custody, "paths": res.Paths, "unlisted": res.Unlisted})
		} else {
			for _, p := range res.Paths {
				fmt.Fprintf(e.out, "  %-12s %-40s %s\n", p.Kind, p.ID, p.Grant)
			}
		}
		if len(res.Unlisted) > 0 {
			ids := make([]string, len(res.Unlisted))
			for i, p := range res.Unlisted {
				ids[i] = p.ID
			}
			return e.fail(name, errs.New(errs.ELAW, "factory access not in the hand-off record (LAW-17): %s", strings.Join(ids, ", ")))
		}
		if !e.g.JSON {
			fmt.Fprintf(e.out, "access: %s: every factory path serves a retained grant (LAW-17)\n", c.Tenant)
		}
		return 0
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

// evalPython delegates run, gate and diff to the Python eval harness (SB-015, SB-502,
// SB-1507).
func (e *env) evalPython(sub string, args []string) int {
	root, ok := findRepo()
	if !ok {
		return e.fail("sb eval "+sub, errs.New(errs.EUNAVAIL, "eval runner not found; run from inside the repository"))
	}
	c := exec.Command("python3", append([]string{"-m", "evals", sub}, args...)...)
	c.Dir, c.Stdout, c.Stderr = root, e.out, e.err
	c.Env = append(os.Environ(), "PYTHONPATH="+root)
	if err := c.Run(); err != nil {
		var x *exec.ExitError
		if errors.As(err, &x) {
			return x.ExitCode()
		}
		return e.fail("sb eval "+sub, errs.New(errs.EUNAVAIL, "python3 not available: %v", err))
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
	var x *errs.Error
	if errors.As(err, &x) {
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
	var file, key, since string
	var o cellOpts
	for i := 0; i < len(args); i++ {
		if took, err := o.take(args, &i); took {
			if err != nil {
				return e.fail(name, err)
			}
			continue
		}
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
			case "--since":
				since = args[i+1]
			}
			i++
		default:
			return e.fail(name, errs.New(errs.EUSAGE, "unknown option %q; see sb-audit(1)", args[i]))
		}
	}
	if file == "" {
		if !o.set() && e.g.Tenant == "" {
			return e.fail(name, errs.New(errs.EUSAGE, "name a cell (--tenant T or --cell URL) or a file (--export FILE)"))
		}
		return e.auditVerifyCell(name, o, since, key)
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return e.fail(name, errs.New(errs.EUSAGE, "cannot read %s: %v", file, err))
	}
	return e.verifyExport(name, b, key)
}

// verifyExport checks an export's chain and anchors (sb-audit(1), verify).
func (e *env) verifyExport(name string, b []byte, key string) int {
	tenant := e.g.Tenant
	if tenant == "" {
		tenant = firstTenant(b)
	}
	if tenant == "" { // an anchor alone doesn't say whose it is (BUG-0027)
		return e.fail(name, errs.New(errs.EUSAGE, "the log has no rows yet, so its tenant can't be read from it; pass --tenant"))
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
