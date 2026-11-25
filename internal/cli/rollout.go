package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"example.com/second-brain/internal/core/features"
	"example.com/second-brain/internal/errs"
)

// rollout is sb rollout list|hold|release [FEATURE] [--tenant T | --cell URL] (sb-release(7),
// Feature rollouts; SB-5106): the running cell's features in pilot, held or on, and holding
// or releasing one.
func (e *env) rollout(op string, args []string) int {
	name := "sb rollout " + op
	const usage = "usage: sb rollout list|hold|release [FEATURE] [--tenant T | --cell URL]"
	var o cellOpts
	var feature string
	args = e.localArgs(args)
	for i := 0; i < len(args); i++ {
		if took, err := o.take(args, &i); took {
			if err != nil {
				return e.fail(name, err)
			}
			continue
		}
		if feature != "" || len(args[i]) > 0 && args[i][0] == '-' {
			return e.fail(name, errs.New(errs.EUSAGE, usage))
		}
		feature = args[i]
	}
	if (op == "list") != (feature == "") {
		return e.fail(name, errs.New(errs.EUSAGE, usage))
	}
	cellURL, _, err := e.cellURL(o)
	if err != nil {
		return e.fail(name, err)
	}
	method, path := http.MethodGet, "/v1/cell/rollouts"
	if op != "list" {
		method, path = http.MethodPost, "/v1/cell/rollouts/"+feature+"/"+op
	}
	raw, err := cellDo(method, cellURL, path, "application/json", []byte("{}"), time.Minute)
	if err != nil {
		return e.fail(name, err)
	}
	var out struct{ Features []features.Status }
	if err := json.Unmarshal(raw, &out); err != nil {
		return e.fail(name, errs.New(errs.EUNAVAIL, "the cell's answer isn't a rollout list"))
	}
	if e.g.JSON {
		return e.json(out)
	}
	for _, f := range out.Features {
		line := fmt.Sprintf("%-20s %-8s %-6s %s", f.Name, f.Ticket, f.State, f.Rollout)
		if f.State == features.Pilot {
			line += ", on for everyone " + f.OnAt.Format(time.DateOnly)
		}
		fmt.Fprintln(e.out, line)
	}
	return 0
}
