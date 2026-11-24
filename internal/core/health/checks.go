package health

import (
	"example.com/second-brain/internal/core/box"
)

// Check is one line of health as /metrics shows it (SB-3203): Warning when it warns,
// Unknown when the cell can't read it. An unreadable line is never fine on both.
type Check struct {
	Name    string
	Warning bool
	Unknown bool
}

// Checks are the report's lines, each once, in the report's order.
func (r Report) Checks() []Check {
	var out []Check
	for _, j := range r.Jobs {
		out = append(out, Check{Name: "job:" + j.Name, Warning: j.Overdue, Unknown: j.Unreadable})
	}
	for _, c := range r.Connectors {
		out = append(out, Check{Name: "connector:" + c.Kind, Warning: c.Stale || c.Failure != ""})
	}
	if r.Box != nil {
		for _, l := range r.Box.Checks {
			out = append(out, Check{Name: "box:" + l.Check, Warning: l.Verdict == box.Warn || l.Verdict == box.Fail, Unknown: l.Verdict == box.Unknown})
		}
	}
	if r.Notify != nil {
		out = append(out, Check{Name: "notify", Warning: r.Notify.Warning})
	}
	if r.Index != nil {
		out = append(out, Check{Name: "index", Warning: r.Index.Warning})
	}
	if r.Prompt != nil {
		out = append(out, Check{Name: "prompt", Warning: r.Prompt.Warning})
	}
	if r.AnswerTime != nil {
		out = append(out, Check{Name: "answer_time", Warning: r.AnswerTime.Warning})
	}
	if r.Unstewarded != nil {
		out = append(out, Check{Name: "stewards", Warning: len(r.Unstewarded.Spaces) > 0, Unknown: len(r.Unstewarded.Unknown) > 0})
	}
	return out
}
