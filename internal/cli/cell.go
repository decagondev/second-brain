package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/errs"
)

// cellDo calls a running cell's control-plane endpoint with SB_TOKEN, the control plane's
// identity, and returns the answer or the cell's own error.
func cellDo(method, cellURL, path, contentType string, body []byte, timeout time.Duration) ([]byte, error) {
	if !strings.HasPrefix(cellURL, "https://") && !strings.HasPrefix(cellURL, "http://127.0.0.1") && !strings.HasPrefix(cellURL, "http://localhost") {
		return nil, errs.New(errs.EUSAGE, "--cell must be an https URL")
	}
	token := os.Getenv("SB_TOKEN")
	if token == "" {
		return nil, errs.New(errs.EAUTH, "set SB_TOKEN to the control plane's identity token")
	}
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	hreq, err := http.NewRequest(method, strings.TrimRight(cellURL, "/")+path, rd)
	if err != nil {
		return nil, errs.New(errs.EUSAGE, "bad --cell URL: %v", err)
	}
	hreq.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		hreq.Header.Set("Content-Type", contentType)
	}
	resp, err := (&http.Client{Timeout: timeout}).Do(hreq)
	if err != nil {
		return nil, errs.New(errs.EUNAVAIL, "the cell didn't answer: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<30))
	if resp.StatusCode != http.StatusOK {
		var eb struct {
			Error struct{ Code, Message string }
		}
		_ = json.Unmarshal(raw, &eb)
		c := errs.Code(eb.Error.Code)
		if c == "" {
			c = errs.EUNAVAIL
		}
		return nil, errs.New(c, "the cell refused: %s", eb.Error.Message)
	}
	return raw, nil
}

// cellOpts are the options every command against a running cell takes: --cell URL, or
// --tenant T, whose cell the registry names.
type cellOpts struct {
	cell, tenant string
}

// take consumes --cell and --tenant at args[i], reporting whether it did.
func (o *cellOpts) take(args []string, i *int) (bool, error) {
	switch args[*i] {
	case "--cell", "--tenant":
		if *i+1 >= len(args) {
			return true, errs.New(errs.EUSAGE, "%s needs a value", args[*i])
		}
		if args[*i] == "--cell" {
			o.cell = args[*i+1]
		} else {
			o.tenant = args[*i+1]
		}
		*i++
		return true, nil
	}
	return false, nil
}

func (o cellOpts) set() bool { return o.cell != "" || o.tenant != "" }

// url resolves the cell: --cell as given, or the registry's endpoint for the tenant.
func (e *env) cellURL(o cellOpts) (string, string, error) {
	tenant := o.tenant
	if tenant == "" {
		tenant = e.g.Tenant
	}
	if o.cell != "" {
		return o.cell, tenant, nil
	}
	if tenant == "" {
		return "", "", errs.New(errs.EUSAGE, "name the cell: --cell URL or --tenant T")
	}
	deps, err := e.controlPlane()
	if err != nil {
		return "", "", err
	}
	cells, ok := deps.Cells.(ports.Cells)
	if !ok {
		return "", "", errs.New(errs.EUNAVAIL, "this control plane cannot read cell records")
	}
	c, found, err := cells.CellByTenant(context.Background(), tenant)
	switch {
	case err != nil:
		return "", "", errs.New(errs.EUNAVAIL, "tenant registry unavailable: %v", err)
	case !found:
		return "", "", errs.New(errs.EUSAGE, "no cell for tenant %q", tenant)
	}
	if u, err := url.Parse(c.Endpoint); err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return "", "", errs.New(errs.EUNAVAIL, "the registry's endpoint for %s (%q) isn't a running cell; pass --cell URL", tenant, c.Endpoint)
	}
	return c.Endpoint, tenant, nil
}

// auditVerifyCell verifies a running cell's whole chain from its export (sb-audit(1)).
func (e *env) auditVerifyCell(name string, o cellOpts, since, key string) int {
	cellURL, _, err := e.cellURL(o)
	if err != nil {
		return e.fail(name, err)
	}
	path := "/v1/cell/audit/export"
	if since != "" {
		path += "?since=" + url.QueryEscape(since)
	}
	raw, err := cellDo(http.MethodGet, cellURL, path, "", nil, 10*time.Minute)
	if err != nil {
		return e.fail(name, err)
	}
	return e.verifyExport(name, raw, key)
}

// auditTail prints the cell's rows that match, then follows new ones (sb-audit(1), tail).
func (e *env) auditTail(args []string) int {
	const name = "sb audit tail"
	var o cellOpts
	kinds := map[string]bool{}
	actor, space, follow, from := "", "", true, int64(1)
	args = e.localArgs(args)
	for i := 0; i < len(args); i++ {
		if took, err := o.take(args, &i); took {
			if err != nil {
				return e.fail(name, err)
			}
			continue
		}
		val := func() (string, bool) {
			if i+1 >= len(args) {
				return "", false
			}
			i++
			return args[i], true
		}
		switch a := args[i]; a {
		case "--kind", "--actor", "--space", "--from":
			v, ok := val()
			if !ok {
				return e.fail(name, errs.New(errs.EUSAGE, "%s needs a value", a))
			}
			switch a {
			case "--kind":
				if !audit.Kinds[v] {
					return e.fail(name, errs.New(errs.EUSAGE, "unknown kind %q", v))
				}
				kinds[v] = true
			case "--actor":
				actor = v
			case "--space":
				space = v
			case "--from":
				n, err := strconv.ParseInt(v, 10, 64)
				if err != nil || n < 1 {
					return e.fail(name, errs.New(errs.EUSAGE, "--from needs a row number"))
				}
				from = n
			}
		case "--no-follow":
			follow = false
		default:
			return e.fail(name, errs.New(errs.EUSAGE, "unknown option %q; see sb-audit(1)", a))
		}
	}
	cellURL, _, err := e.cellURL(o)
	if err != nil {
		return e.fail(name, err)
	}
	match := func(r audit.Row) bool {
		if len(kinds) > 0 && !kinds[r.Kind] {
			return false
		}
		if actor != "" && r.Actor != actor {
			return false
		}
		if space != "" && r.Space != "spc_"+space && (r.Kind != "kill" || r.Subject != "space:"+space) {
			return false
		}
		return true
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	defer signal.Stop(stop)
	for {
		raw, err := cellDo(http.MethodGet, cellURL, fmt.Sprintf("/v1/cell/audit?from=%d", from), "", nil, time.Minute)
		if err != nil {
			return e.fail(name, err)
		}
		var page struct {
			Rows []audit.Row `json:"rows"`
			Next int64       `json:"next"`
		}
		if err := json.Unmarshal(raw, &page); err != nil {
			return e.fail(name, errs.New(errs.EUNAVAIL, "the cell's answer is unreadable"))
		}
		for _, r := range page.Rows {
			if match(r) {
				e.printRow(r)
			}
		}
		if page.Next > from {
			from = page.Next
		}
		if len(page.Rows) == 1000 {
			continue
		}
		if !follow {
			return 0
		}
		select {
		case <-stop:
			return 0
		case <-time.After(2 * time.Second):
		}
	}
}

func (e *env) printRow(r audit.Row) {
	if e.g.JSON {
		b, _ := json.Marshal(r)
		fmt.Fprintln(e.out, string(b))
		return
	}
	at := r.At
	if t, err := time.Parse(time.RFC3339Nano, r.At); err == nil {
		at = t.UTC().Format("15:04:05")
	}
	id := r.Approval
	if id == "" {
		id = r.ID
	}
	line := fmt.Sprintf("%s %s %-8s %s by %s  %s", at, id, r.Kind, r.State, r.Actor, r.Subject)
	if r.Space != "" && r.Space != "company" {
		line += "  " + r.Space
	}
	if len(r.PayloadSHA256) >= 4 {
		line += "  hash " + r.PayloadSHA256[:4] + "…"
	}
	fmt.Fprintln(e.out, line)
}

// auditExport writes a running cell's signed export for a window, and checks it before it
// says so (sb-audit(1), export).
func (e *env) auditExport(args []string) int {
	const name = "sb audit export"
	var o cellOpts
	var since, until, out string
	args = e.localArgs(args)
	for i := 0; i < len(args); i++ {
		if took, err := o.take(args, &i); took {
			if err != nil {
				return e.fail(name, err)
			}
			continue
		}
		switch a := args[i]; a {
		case "--since", "--until", "--out":
			if i+1 >= len(args) {
				return e.fail(name, errs.New(errs.EUSAGE, "%s needs a value", a))
			}
			v := args[i+1]
			i++
			switch a {
			case "--since":
				since = v
			case "--until":
				until = v
			default:
				out = v
			}
		default:
			return e.fail(name, errs.New(errs.EUSAGE, "unknown option %q; see sb-audit(1)", a))
		}
	}
	if since == "" || out == "" {
		return e.fail(name, errs.New(errs.EUSAGE, "usage: sb audit export --tenant T --since TIME [--until TIME] --out FILE"))
	}
	for _, v := range []string{since, until} {
		if _, err := time.Parse(time.RFC3339, v); v != "" && err != nil {
			return e.fail(name, errs.New(errs.EUSAGE, "%q isn't an RFC 3339 time", v))
		}
	}
	cellURL, tenant, err := e.cellURL(o)
	if err != nil {
		return e.fail(name, err)
	}
	q := url.Values{"since": {since}}
	if until != "" {
		q.Set("until", until)
	}
	raw, err := cellDo(http.MethodGet, cellURL, "/v1/cell/audit/export?"+q.Encode(), "", nil, 10*time.Minute)
	if err != nil {
		return e.fail(name, err)
	}
	if tenant == "" {
		tenant = firstTenant(raw)
	}
	rep, err := audit.Verify(bytes.NewReader(raw), tenant, "")
	if err != nil {
		return e.fail(name, errs.New(errs.ECHAIN, "the cell's export doesn't verify: %v", err))
	}
	if err := os.WriteFile(out, raw, 0o600); err != nil {
		return e.fail(name, errs.New(errs.EUSAGE, "cannot write %s: %v", out, err))
	}
	if e.g.JSON {
		return e.json(map[string]any{"out": out, "rows": rep.Rows, "first_seq": rep.FirstSeq, "last_seq": rep.LastSeq, "anchors": rep.Anchors, "key": rep.Key})
	}
	fmt.Fprintf(e.out, "export: %s: %d rows (seq %d to %d), %d anchors, verified\n", out, rep.Rows, rep.FirstSeq, rep.LastSeq, rep.Anchors)
	return 0
}
