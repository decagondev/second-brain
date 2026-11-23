package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"example.com/second-brain/internal/core/route"
	"example.com/second-brain/internal/errs"
)

// spaceScopeReport prints what each space's scope patterns match in the cell's index now
// (sb-space(1), scope-report; SB-2903): GET /v1/admin/index/scopes, which changes nothing.
func (e *env) spaceScopeReport(args []string) int {
	const name = "sb space scope-report"
	var o cellOpts
	args = e.localArgs(args)
	for i := 0; i < len(args); i++ {
		took, err := o.take(args, &i)
		if err != nil {
			return e.fail(name, err)
		}
		if !took {
			return e.fail(name, errs.New(errs.EUSAGE, "unexpected %s; usage: sb space scope-report (--cell URL | --tenant T) [--json]", args[i]))
		}
	}
	cellURL, _, err := e.cellURL(o)
	if err != nil {
		return e.fail(name, err)
	}
	raw, err := cellDo(http.MethodGet, cellURL, "/v1/admin/index/scopes", "", nil, time.Minute)
	if err != nil {
		return e.fail(name, err)
	}
	var r struct {
		Patterns []route.PatternCount `json:"patterns"`
	}
	if json.Unmarshal(raw, &r) != nil || r.Patterns == nil {
		return e.fail(name, errs.New(errs.EUNAVAIL, "the cell's answer isn't a scope report"))
	}
	if e.g.JSON {
		return e.json(r)
	}
	if len(r.Patterns) == 0 {
		fmt.Fprintln(e.out, "No space has a scope pattern.")
		return 0
	}
	fmt.Fprintf(e.out, "%-16s %-10s %-28s %s\n", "SPACE", "KIND", "PATTERN", "MATCHES")
	for _, p := range r.Patterns {
		note := ""
		switch p.Problem {
		case route.ProblemNone:
			note = "  matches nothing: a typo, or a folder that moved?"
		case route.ProblemTooWide:
			note = "  matches more than half: too wide?"
		}
		fmt.Fprintf(e.out, "%-16s %-10s %-28s %d of %d%s\n", p.Space, p.Kind, p.Pattern, p.Matches, p.Of, note)
	}
	return 0
}

// indexPrune lists the index's leftovers, or removes the documents no source lists (sb(1);
// SB-2905): GET /v1/admin/index/leftovers, and POST /v1/admin/index/prune with a reason.
func (e *env) indexPrune(args []string) int {
	const name = "sb index prune"
	var o cellOpts
	dry, reason := false, ""
	args = e.localArgs(args)
	for i := 0; i < len(args); i++ {
		took, err := o.take(args, &i)
		if err != nil {
			return e.fail(name, err)
		}
		if took {
			continue
		}
		switch {
		case args[i] == "--dry-run":
			dry = true
		case args[i] == "--reason" && i+1 < len(args):
			reason = args[i+1]
			i++
		default:
			return e.fail(name, errs.New(errs.EUSAGE, "unexpected %s; usage: sb index prune [--dry-run] [--reason TEXT] (--cell URL | --tenant T)", args[i]))
		}
	}
	if !dry && strings.TrimSpace(reason) == "" {
		return e.fail(name, errs.New(errs.EUSAGE, "say why with --reason, or look first with --dry-run"))
	}
	cellURL, _, err := e.cellURL(o)
	if err != nil {
		return e.fail(name, err)
	}
	if !dry {
		body, _ := json.Marshal(map[string]string{"reason": reason})
		raw, err := cellDo(http.MethodPost, cellURL, "/v1/admin/index/prune", "application/json", body, 10*time.Minute)
		if err != nil {
			return e.fail(name, err)
		}
		var r struct {
			Pruned *int `json:"pruned"`
		}
		if json.Unmarshal(raw, &r) != nil || r.Pruned == nil {
			return e.fail(name, errs.New(errs.EUNAVAIL, "the cell's answer isn't a prune"))
		}
		if e.g.JSON {
			return e.json(r)
		}
		fmt.Fprintf(e.out, "Removed %d documents their source no longer lists.\n", *r.Pruned)
		return 0
	}
	raw, err := cellDo(http.MethodGet, cellURL, "/v1/admin/index/leftovers", "", nil, 10*time.Minute)
	if err != nil {
		return e.fail(name, err)
	}
	var l struct {
		Gone   []string `json:"gone"`
		Unheld []string `json:"unheld"`
	}
	if json.Unmarshal(raw, &l) != nil || l.Gone == nil || l.Unheld == nil {
		return e.fail(name, errs.New(errs.EUNAVAIL, "the cell's answer isn't a list of leftovers"))
	}
	if e.g.JSON {
		return e.json(l)
	}
	fmt.Fprintf(e.out, "Gone from their source (prune removes these): %d\n", len(l.Gone))
	for _, c := range l.Gone {
		fmt.Fprintf(e.out, "  %s\n", c)
	}
	fmt.Fprintf(e.out, "Held by no space (kept; fix the scopes): %d\n", len(l.Unheld))
	for _, c := range l.Unheld {
		fmt.Fprintf(e.out, "  %s\n", c)
	}
	return 0
}
