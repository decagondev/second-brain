package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
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
