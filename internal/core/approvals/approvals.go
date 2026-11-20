// Package approvals is the approval state machine for consequential actions
// (sb-approvals(7); SB-406). State names are identical to proofs/tla/Approvals.tla. The
// payload hash binds an approval to one payload, target and workflow (LAW-15).
package approvals

import (
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
