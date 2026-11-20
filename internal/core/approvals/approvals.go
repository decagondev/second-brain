// Package approvals is the approval state machine for consequential actions
// (sb-approvals(7); SB-406). State names are identical to proofs/tla/Approvals.tla. The
// payload hash binds an approval to one payload, target and workflow (LAW-15).
package approvals

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"example.com/second-brain/internal/core/canonjson"
	"fmt"
	"time"
)

// State is one of the eight states in sb-approvals(7).
type State string

// The states, as in Approvals.tla.
const (
	Drafted         State = "Drafted"
	PendingApproval State = "PendingApproval"
	Approved        State = "Approved"
	Rejected        State = "Rejected"
	Expired         State = "Expired"
	Executing       State = "Executing"
	Ran             State = "Ran"
	DidNotRun       State = "DidNotRun"
)

// Next is every allowed transition (sb-approvals(7), States).
var Next = map[State][]State{
	Drafted:         {PendingApproval},
	PendingApproval: {Approved, Rejected, Expired, Drafted},
	Approved:        {Executing, Expired, Drafted},
	Executing:       {Ran, DidNotRun},
}

// CanMove reports whether from -> to is allowed.
func CanMove(from, to State) bool {
	for _, s := range Next[from] {
		if s == to {
			return true
		}
	}
	return false
}

// TTL is how long an action may wait for approval.
const TTL = 24 * time.Hour

// Action is one consequential action.
type Action struct {
	ID            string          `json:"id"`
	Tenant        string          `json:"tenant"`
	Workflow      string          `json:"workflow"`
	Tool          string          `json:"tool"`
	Target        string          `json:"target"` // e.g. tickets:OPS-1182
	Payload       json.RawMessage `json:"payload"`
	PayloadSHA256 string          `json:"payload_sha256"`
	State         State           `json:"state"`
	Requester     string          `json:"requester"`
	Approvers     string          `json:"approvers"` // e.g. group:grp_it_leads
	ExpiresAt     time.Time       `json:"expires_at"`
	Approval      string          `json:"approval,omitempty"` // apr_…
	ApprovedBy    string          `json:"approved_by,omitempty"`
	ApprovedHash  string          `json:"approved_hash,omitempty"`
	Note          string          `json:"note,omitempty"`
	Receipt       string          `json:"receipt,omitempty"`
	Space         string          `json:"space,omitempty"` // spc_NAME it was proposed in (SB-1008)
	Reason        string          `json:"reason,omitempty"`
	PolicyReason  string          `json:"policy_reason,omitempty"`
	Unconfirmed   bool            `json:"unconfirmed,omitempty"` // Act failed or timed out: the target may have it
	FinishedAt    time.Time       `json:"finished_at,omitempty"`
	// Segregation of duties (SB-1204): how many approvals it needs (0 and 1 mean one), and
	// the first approval of an action that needs two.
	Needs           int    `json:"needs,omitempty"`
	SecondApprovers string `json:"second_approvers,omitempty"` // the duty's group, for the second approval
	FirstApproval   string `json:"first_approval,omitempty"`
	FirstApprovedBy string `json:"first_approved_by,omitempty"`
	FirstHash       string `json:"first_hash,omitempty"`
}

// MaxReportingLine is the longest reporting line the cell follows; longer refuses.
const MaxReportingLine = 20

// ErrReportingLine is a reporting line the directory can't give now.
var ErrReportingLine = errors.New("approvals: the reporting line can't be read")

// ReportingLine is everyone above user: their manager, their manager's manager, and so on,
// from manager (the directory). A loop ends the line; an error or a line longer than
// MaxReportingLine is an error, so the approval is refused (sb-approvals(7), SB-1204).
func ReportingLine(ctx context.Context, manager func(ctx context.Context, user string) (string, error), user string) ([]string, error) {
	var line []string
	seen := map[string]bool{user: true}
	for cur := user; ; {
		m, err := manager(ctx, cur)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrReportingLine, err)
		}
		if m == "" || seen[m] {
			return line, nil
		}
		if len(line) == MaxReportingLine {
			return nil, fmt.Errorf("%w: longer than %d", ErrReportingLine, MaxReportingLine)
		}
		line, seen[m], cur = append(line, m), true, m
	}
}

// Amount is the payload's top-level "amount", a whole number; ok is false when it has none
// or it isn't one, and then a duty asks for both approvals.
func Amount(payload json.RawMessage) (amount int64, ok bool) {
	var p map[string]json.RawMessage
	if json.Unmarshal(payload, &p) != nil {
		return 0, false
	}
	raw := bytes.TrimSpace(p["amount"])
	if len(raw) == 0 || (raw[0] != '-' && (raw[0] < '0' || raw[0] > '9')) {
		return 0, false // none, a string, or anything else that isn't a number
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var n json.Number
	if d.Decode(&n) != nil {
		return 0, false
	}
	v, err := n.Int64()
	return v, err == nil
}

// Store keeps actions. Move changes an action's state only if it is still in from (a
// compare-and-set), applying change to the stored copy.
type Store interface {
	Put(ctx context.Context, a Action) error
	Get(ctx context.Context, tenant, id string) (Action, bool, error)
	Move(ctx context.Context, tenant, id string, from State, change func(*Action) error) (Action, error)
	// Unconfirmed lists the tenant's DidNotRun actions still marked unconfirmed.
	Unconfirmed(ctx context.Context, tenant string) ([]Action, error)
}

// Errors (sb-actions(3)).
var (
	ErrNoApproval = errors.New("approvals: ENOAPPROVAL")
	ErrPayload    = errors.New("approvals: payload not canonicalisable")
	ErrMoved      = errors.New("approvals: the action changed state; reload it")
)

// Canonical is the RFC 8785 JSON of a payload (sb-approvals(7), The payload hash): see
// canonjson. Errors wrap ErrPayload.
func Canonical(raw []byte) ([]byte, error) {
	c, err := canonjson.Encode(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPayload, err)
	}
	return c, nil
}

// Hash is the payload hash: SHA-256 of the canonical {"payload","target","workflow"}.
func Hash(workflow, target string, payload []byte) (string, error) {
	c, err := Canonical(payload)
	if err != nil {
		return "", err
	}
	wrapped, err := json.Marshal(map[string]any{"payload": json.RawMessage(c), "target": target, "workflow": workflow})
	if err != nil {
		return "", err
	}
	c2, err := Canonical(wrapped)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(c2)
	return hex.EncodeToString(sum[:]), nil
}
