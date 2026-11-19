package laws

import (
	"fmt"
	"math/rand"
)

// LAW-15 (SB-406): an approval authorises exactly one payload, identified by its hash, by
// someone other than the requester, and any edit voids it. Whatever mix of proposals,
// edits and approval attempts (stale hashes, made-up hashes, self-approval), every
// approval that succeeds matches the payload at that moment, and what executes is that
// payload.

// ApprovalWorld drives the real approval flow for one tenant.
type ApprovalWorld interface {
	Propose(requester, payload string) (id, hash string, err error)
	Edit(id, editor, payload string) (hash string, err error)           // back to Drafted, then re-proposed
	Approve(id, approver, echoedHash string) (approved bool, err error) // a refusal is approved=false, err=nil
	Executed(id string) (payloads []string)                             // every payload executed for id
}

// NewApprovalWorld builds the system under test.
var NewApprovalWorld = func() ApprovalWorld { return pendingApprovals{} }

type pendingApprovals struct{}

func (pendingApprovals) Propose(string, string) (string, string, error) {
	return "", "", fmt.Errorf("%w: approvals arrive with SB-406", ErrPending)
}
func (pendingApprovals) Edit(string, string, string) (string, error)  { return "", nil }
func (pendingApprovals) Approve(string, string, string) (bool, error) { return false, nil }
func (pendingApprovals) Executed(string) []string                     { return nil }

func law15(r *rand.Rand, w ApprovalWorld) error {
	users := []string{"usr_dana", "usr_erin", "usr_finn"}
	type act struct {
		requester, payload, hash string
		hashes                   []string // every hash it has had
		approvedPayload          string
		approved                 bool
	}
	acts := map[string]*act{}
	var ids []string
	for step := 0; step < 25; step++ {
		switch {
		case len(ids) == 0 || r.Intn(4) == 0:
			req := users[r.Intn(len(users))]
			p := fmt.Sprintf("comment-%d", r.Intn(1000))
			id, h, err := w.Propose(req, p)
			if err != nil {
				return err
			}
			acts[id] = &act{requester: req, payload: p, hash: h, hashes: []string{h}}
			ids = append(ids, id)
		case r.Intn(4) == 0:
			id := ids[r.Intn(len(ids))]
			a := acts[id]
			if a.approved {
				continue // an executed action can't be edited
			}
			p := fmt.Sprintf("edited-%d", r.Intn(1000))
			h, err := w.Edit(id, a.requester, p)
			if err != nil {
				return err
			}
			a.payload, a.hash = p, h
			a.hashes = append(a.hashes, h)
		default:
			id := ids[r.Intn(len(ids))]
			a := acts[id]
			approver := users[r.Intn(len(users))]
			echo := a.hash
			switch r.Intn(4) {
			case 0:
				echo = a.hashes[r.Intn(len(a.hashes))] // possibly stale
			case 1:
				echo = fmt.Sprintf("%064x", r.Int63()) // made up
			}
			ok, err := w.Approve(id, approver, echo)
			if err != nil {
				return err
			}
			if ok {
				if a.approved {
					return fmt.Errorf("%s was approved twice", id)
				}
				if echo != a.hash {
					return fmt.Errorf("%s was approved with hash %.8s… but the payload's hash is %.8s…", id, echo, a.hash)
				}
				if approver == a.requester {
					return fmt.Errorf("%s was approved by its own requester %s", id, approver)
				}
				a.approved, a.approvedPayload = true, a.payload
			}
		}
	}
	for _, id := range ids {
		a := acts[id]
		got := w.Executed(id)
		switch {
		case !a.approved && len(got) > 0:
			return fmt.Errorf("%s executed %v without an approval", id, got)
		case len(got) > 1:
			return fmt.Errorf("%s executed %d times", id, len(got))
		case len(got) == 1 && got[0] != a.approvedPayload:
			return fmt.Errorf("%s executed %q, but %q was approved", id, got[0], a.approvedPayload)
		}
	}
	return nil
}
