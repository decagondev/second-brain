package laws

import (
	"fmt"
	"math/rand"

	"example.com/second-brain/internal/core/ports"
)

// Fault is injected into every live ACL call of a world (LAW-03).
type Fault int

const (
	FaultNone    Fault = iota
	FaultTimeout       // every ACL call times out: every verdict is Unknown
)

type User struct {
	ID, Tenant string
	Groups     []string
}

// Doc is a source document. Prefilter is the ACL copy taken at ingest and may be
// stale; Live is the source's answer now. A user missing from Live means the
// source can't say (Unknown). Missing means the document was deleted at the source.
// Public means "anyone with the link": the source says Allow to users of *any*
// tenant, so tenant isolation (LAW-02) must never rely on the live check.
type Doc struct {
	ID, Tenant string
	Version    int
	Prefilter  []string
	Live       map[string]ports.Verdict
	Missing    bool
	Public     bool
}

// Event is one step a system takes on a tool call, in order.
type Event struct {
	Kind     string // "audit" or "exec"
	Call     string
	Approval string
	Receipt  string
}

// World is one generated situation: tenants, users, documents, approvals, a target
// system and the record of everything the system under test did.
type World struct {
	Users     []User
	Docs      []Doc
	Fault     Fault
	Approvals map[string]bool // valid approval ids
	NoReceipt bool            // the target system returns no receipt
	Log       []Event
	asked     map[string]bool
	conseq    map[string]bool // which generated calls are consequential
}

// Gen builds a random world with 2–3 tenants. It deliberately includes stale
// prefilters (prefilter says yes, source says no), unknown verdicts and missing docs.
func Gen(r *rand.Rand) *World {
	w := &World{Approvals: map[string]bool{"apr_ok": true}, asked: map[string]bool{}, conseq: map[string]bool{}}
	tenants := []string{"acme-logistics", "beta-freight", "gamma-parts"}[:2+r.Intn(2)]
	for _, t := range tenants {
		groups := []string{t + "/g1", t + "/g2", t + "/g3"}
		var users []User
		for i := 0; i < 2+r.Intn(4); i++ {
			u := User{ID: fmt.Sprintf("%s/u%d", t, i), Tenant: t}
			for _, g := range groups {
				if r.Intn(2) == 0 {
					u.Groups = append(u.Groups, g)
				}
			}
			users = append(users, u)
		}
		w.Users = append(w.Users, users...)
		for i := 0; i < 3+r.Intn(6); i++ {
			d := Doc{ID: fmt.Sprintf("docs:%s-%d", t, i), Tenant: t, Version: 1 + r.Intn(20), Live: map[string]ports.Verdict{}, Missing: r.Intn(10) == 0, Public: r.Intn(7) == 0}
			for _, g := range groups {
				if r.Intn(2) == 0 {
					d.Prefilter = append(d.Prefilter, g)
				}
			}
			for _, u := range users {
				if r.Intn(4) == 0 {
					d.Prefilter = append(d.Prefilter, u.ID)
				}
				switch n := r.Intn(10); {
				case n < 6:
					d.Live[u.ID] = ports.VerdictAllow
				case n < 9:
					d.Live[u.ID] = ports.VerdictDeny
				} // else: absent, so Unknown
			}
			w.Docs = append(w.Docs, d)
		}
	}
	if r.Intn(4) == 0 {
		w.Fault = FaultTimeout
	}
	w.NoReceipt = r.Intn(3) == 0
	return w
}

func (w *World) doc(id string) (Doc, bool) {
	for _, d := range w.Docs {
		if d.ID == id {
			return d, true
		}
	}
	return Doc{}, false
}

// user looks up a generated user; an unknown id is a harness bug, so it panics.
func (w *World) user(id string) User {
	for _, u := range w.Users {
		if u.ID == id {
			return u
		}
	}
	panic("laws: unknown user " + id)
}

// ACL is the live source check a system under test must call. It records the call.
func (w *World) ACL(userID, docID string) ports.Verdict {
	w.asked[userID+"|"+docID] = true
	d, ok := w.doc(docID)
	if !ok || d.Missing || w.Fault == FaultTimeout {
		return ports.VerdictUnknown
	}
	if d.Public {
		return ports.VerdictAllow
	}
	return d.Live[userID] // zero value is VerdictUnknown
}

// PrefilterMatch is what an index prefilter would say. It may be stale.
func (w *World) PrefilterMatch(userID, docID string) bool {
	d, ok := w.doc(docID)
	if !ok {
		return false
	}
	u := w.user(userID)
	for _, p := range d.Prefilter {
		if p == u.ID {
			return true
		}
		for _, g := range u.Groups {
			if p == g {
				return true
			}
		}
	}
	return false
}

// Audit records a write-ahead audit row for a tool call.
func (w *World) Audit(call, approval string) {
	w.Log = append(w.Log, Event{Kind: "audit", Call: call, Approval: approval})
}

// Act performs a tool call against the target and returns its receipt ("" if none).
func (w *World) Act(call string) string {
	rc := "rcpt_" + call
	if w.NoReceipt {
		rc = ""
	}
	w.Log = append(w.Log, Event{Kind: "exec", Call: call, Receipt: rc})
	return rc
}

// truthAllow is ground truth for checks: does the source allow this right now?
// It doesn't record a call.
func (w *World) truthAllow(userID, docID string) bool {
	d, ok := w.doc(docID)
	return ok && !d.Missing && w.Fault != FaultTimeout && (d.Public || d.Live[userID] == ports.VerdictAllow)
}
