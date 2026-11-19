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
	"fmt"
	"sort"
	"strconv"
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
	Reason        string          `json:"reason,omitempty"`
}

// Store keeps actions. Move changes an action's state only if it is still in from (a
// compare-and-set), applying change to the stored copy.
type Store interface {
	Put(ctx context.Context, a Action) error
	Get(ctx context.Context, tenant, id string) (Action, bool, error)
	Move(ctx context.Context, tenant, id string, from State, change func(*Action) error) (Action, error)
}

// Errors (sb-actions(3)).
var (
	ErrNoApproval = errors.New("approvals: ENOAPPROVAL")
	ErrPayload    = errors.New("approvals: payload not canonicalisable")
	ErrMoved      = errors.New("approvals: the action changed state; reload it")
)

// Canonical is the RFC 8785 JSON of v for the payload subset: objects, arrays, strings,
// booleans, null, and integers within ±2^53 (sb-approvals(7), The payload hash).
func Canonical(raw []byte) ([]byte, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPayload, err)
	}
	if d.More() {
		return nil, fmt.Errorf("%w: trailing data", ErrPayload)
	}
	var b bytes.Buffer
	if err := canon(&b, v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func canon(b *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		b.WriteString(strconv.FormatBool(x))
	case json.Number:
		i, err := strconv.ParseInt(string(x), 10, 64)
		if err != nil || i > 1<<53 || i < -(1<<53) {
			return fmt.Errorf("%w: number %s is not an integer within ±2^53", ErrPayload, x)
		}
		b.WriteString(strconv.FormatInt(i, 10))
	case string:
		enc := json.NewEncoder(b)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(x); err != nil {
			return err
		}
		b.Truncate(b.Len() - 1)
	case []any:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := canon(b, e); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return utf16Less(keys[i], keys[j]) })
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := canon(b, k); err != nil {
				return err
			}
			b.WriteByte(':')
			if err := canon(b, x[k]); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	default:
		return fmt.Errorf("%w: unsupported value %T", ErrPayload, v)
	}
	return nil
}

// utf16Less orders keys by UTF-16 code units, as RFC 8785 requires.
func utf16Less(a, b string) bool {
	ua, ub := utf16(a), utf16(b)
	for i := 0; i < len(ua) && i < len(ub); i++ {
		if ua[i] != ub[i] {
			return ua[i] < ub[i]
		}
	}
	return len(ua) < len(ub)
}

func utf16(s string) []uint16 {
	var out []uint16
	for _, r := range s {
		if r >= 0x10000 {
			r -= 0x10000
			out = append(out, uint16(0xD800+(r>>10)), uint16(0xDC00+(r&0x3FF)))
		} else {
			out = append(out, uint16(r))
		}
	}
	return out
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
