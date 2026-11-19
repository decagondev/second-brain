package laws

import (
	"fmt"
	"math/rand"
)

// Each check returns nil, ErrPending (wrapped), or a description of the violation.

// LAW-01: no answer cites or uses a document the asker can't open now, and the
// source must actually have been asked (a prefilter match is not enough).
func law01(w *World, sys System) error {
	for _, u := range w.Users {
		a, err := sys.Ask(w, u.ID, "anything")
		if err != nil {
			return err
		}
		for _, d := range used(a) {
			if !w.truthAllow(u.ID, d) {
				return fmt.Errorf("%s was given or cited %s, which the source does not allow now", u.ID, d)
			}
			if !w.asked[u.ID+"|"+d] {
				return fmt.Errorf("%s was given %s without asking the source", u.ID, d)
			}
		}
	}
	return nil
}

// LAW-02: nothing from another tenant, ever.
func law02(w *World, sys System) error {
	for _, u := range w.Users {
		a, err := sys.Ask(w, u.ID, "anything")
		if err != nil {
			return err
		}
		for _, id := range used(a) {
			if d, ok := w.doc(id); !ok || d.Tenant != u.Tenant {
				return fmt.Errorf("%s (tenant %s) was given %s from another tenant", u.ID, u.Tenant, id)
			}
		}
	}
	return nil
}

// LAW-03: unknown (including timeouts and missing documents) is refused.
func law03(w *World, sys System) error {
	for _, u := range w.Users {
		a, err := sys.Ask(w, u.ID, "anything")
		if err != nil {
			return err
		}
		for _, id := range used(a) {
			d, ok := w.doc(id)
			if !ok || d.Missing || w.Fault == FaultTimeout {
				return fmt.Errorf("%s was given %s although the source could not answer", u.ID, id)
			}
			if _, known := d.Live[u.ID]; !known && !d.Public {
				return fmt.Errorf("%s was given %s on an unknown verdict", u.ID, id)
			}
		}
	}
	return nil
}

// calls makes a random mix of plain and consequential calls, with valid, invalid
// and missing approvals, and records which are consequential.
func calls(w *World, r *rand.Rand) []ToolCall {
	var out []ToolCall
	for i := 0; i < 4+r.Intn(5); i++ {
		tc := ToolCall{ID: fmt.Sprintf("tc_%d", i), Consequential: r.Intn(2) == 0}
		switch r.Intn(3) {
		case 0:
			tc.Approval = "apr_ok"
		case 1:
			tc.Approval = "apr_forged"
		}
		w.conseq[tc.ID] = tc.Consequential
		out = append(out, tc)
	}
	return out
}

// LAW-04: every execution has an earlier audit row for that call; consequential
// calls also need a valid approval id in that row.
func law04(w *World, sys System, r *rand.Rand) error {
	for _, tc := range calls(w, r) {
		if _, err := sys.Call(w, w.Users[0].ID, tc); err != nil {
			return err
		}
	}
	audited := map[string]string{}
	for _, e := range w.Log {
		switch e.Kind {
		case "audit":
			audited[e.Call] = e.Approval
		case "exec":
			apr, ok := audited[e.Call]
			if !ok {
				return fmt.Errorf("%s executed without an earlier audit row", e.Call)
			}
			if w.conseq[e.Call] && !w.Approvals[apr] {
				return fmt.Errorf("consequential %s executed with approval %q", e.Call, apr)
			}
		}
	}
	return nil
}

// LAW-05: Ran only if the target returned a receipt.
func law05(w *World, sys System, r *rand.Rand) error {
	for _, tc := range calls(w, r) {
		o, err := sys.Call(w, w.Users[0].ID, tc)
		if err != nil {
			return err
		}
		if o.State == "Ran" && !receiptIssued(w, tc.ID, o.Receipt) {
			return fmt.Errorf("%s reported Ran with receipt %q the target never issued", tc.ID, o.Receipt)
		}
	}
	return nil
}

func used(a Answer) []string {
	out := append([]string{}, a.Context...)
	for _, c := range a.Citations {
		out = append(out, c.Doc)
	}
	return out
}

func receiptIssued(w *World, call, rc string) bool {
	if rc == "" {
		return false
	}
	for _, e := range w.Log {
		if e.Kind == "exec" && e.Call == call && e.Receipt == rc {
			return true
		}
	}
	return false
}
