package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"example.com/second-brain/internal/errs"
)

// tenantRetention prints what the cell's retention job would delete (sb-tenant(1),
// retention; SB-1905): GET /v1/admin/retention/plan, which deletes nothing.
func (e *env) tenantRetention(args []string) int {
	const name = "sb tenant retention"
	var o cellOpts
	plan, days := false, 0
	args = e.localArgs(args)
	for i := 0; i < len(args); i++ {
		if took, err := o.take(args, &i); took {
			if err != nil {
				return e.fail(name, err)
			}
			continue
		}
		switch args[i] {
		case "--plan":
			plan = true
		case "--days":
			if i+1 >= len(args) {
				return e.fail(name, errs.New(errs.EUSAGE, "--days needs a value"))
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n < 0 || n > 3650 {
				return e.fail(name, errs.New(errs.EUSAGE, "--days must be 0 to 3650"))
			}
			days = n
			i++
		default:
			return e.fail(name, errs.New(errs.EUSAGE, "unexpected %s; usage: sb tenant retention --plan [--days N] (--cell URL | --tenant T)", args[i]))
		}
	}
	if !plan {
		return e.fail(name, errs.New(errs.EUSAGE, "say --plan: the job itself runs nightly in the cell"))
	}
	cellURL, _, err := e.cellURL(o)
	if err != nil {
		return e.fail(name, err)
	}
	raw, err := cellDo(http.MethodGet, cellURL, fmt.Sprintf("/v1/admin/retention/plan?days=%d", days), "", nil, time.Minute)
	if err != nil {
		return e.fail(name, err)
	}
	var p struct {
		Days   int            `json:"days"`
		Cutoff string         `json:"cutoff"`
		Would  map[string]int `json:"would_delete"`
		Held   int            `json:"held"`
	}
	if json.Unmarshal(raw, &p) != nil {
		return e.fail(name, errs.New(errs.EUNAVAIL, "the cell's answer isn't a retention plan"))
	}
	if e.g.JSON {
		return e.json(p)
	}
	when := "tonight"
	if p.Days > 0 {
		when = fmt.Sprintf("in %d days", p.Days)
	}
	fmt.Fprintf(e.out, "The retention job %s deletes records from before %s:\n", when, p.Cutoff)
	kinds := make([]string, 0, len(p.Would))
	for k := range p.Would {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	total := 0
	for _, k := range kinds {
		fmt.Fprintf(e.out, "  %-18s %d\n", k, p.Would[k])
		total += p.Would[k]
	}
	fmt.Fprintf(e.out, "  %-18s %d\n", "total", total)
	fmt.Fprintf(e.out, "Kept by a legal hold: %d\n", p.Held)
	return 0
}
