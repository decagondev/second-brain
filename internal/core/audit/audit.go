// Package audit is the append-only, hash-chained audit log (sb-audit(5), sb-audit(7);
// SB-401, SB-402): rows, their canonical form and hash, signed anchors, export and the
// offline verifier. Stores keep rows; this package decides what a row and a chain are.
package audit

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
	"strings"
	"sync"
	"time"
	"unicode"

	"example.com/second-brain/internal/core/policy"
)

// Zero is row 1's prev.
var Zero = strings.Repeat("0", 64)

// Kinds are the valid row kinds.
var Kinds = map[string]bool{"tool_call": true, "decision": true, "approval": true, "kill": true, "config": true,
	"release": true, "access": true, "custody": true}

// MaxField is the longest string field, in bytes.
const MaxField = 512

// Row is one audit row (sb-audit(5), Fields).
type Row struct {
	Seq           int64  `json:"seq"`
	ID            string `json:"id"`
	At            string `json:"at"`
	Tenant        string `json:"tenant"`
	Actor         string `json:"actor"`
	Kind          string `json:"kind"`
	Subject       string `json:"subject"`
	State         string `json:"state,omitempty"`
	Approval      string `json:"approval,omitempty"`
	PayloadSHA256 string `json:"payload_sha256,omitempty"`
	Receipt       string `json:"receipt,omitempty"`
	Release       string `json:"release"`
	Reason        string `json:"reason"`
	Prev          string `json:"prev"`
	Hash          string `json:"hash,omitempty"`
}

// Event is what a writer supplies; the log fills in seq, id, at, tenant, release, prev and hash.
type Event struct {
	Actor, Kind, Subject, State, Approval, PayloadSHA256, Receipt, Reason string
}

// ErrInvalid is a row that can't be written (bad kind, missing field, unsafe string).
var ErrInvalid = errors.New("audit: invalid row")

// Canonical is the RFC 8785 form of r without hash (sb-audit(5), Canonical form).
func Canonical(r Row) ([]byte, error) {
	fields := map[string]any{"seq": r.Seq}
	for k, v := range map[string]string{"id": r.ID, "at": r.At, "tenant": r.Tenant, "actor": r.Actor, "kind": r.Kind,
		"subject": r.Subject, "state": r.State, "approval": r.Approval, "payload_sha256": r.PayloadSHA256,
		"receipt": r.Receipt, "release": r.Release, "reason": r.Reason, "prev": r.Prev} {
		if err := safe(k, v); err != nil {
			return nil, err
		}
		if v != "" {
			fields[k] = v
		}
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Quote(k))
		b.WriteByte(':')
		switch v := fields[k].(type) {
		case int64:
			b.WriteString(strconv.FormatInt(v, 10))
		case string:
			enc := json.NewEncoder(&b)
			enc.SetEscapeHTML(false)
			if err := enc.Encode(v); err != nil {
				return nil, err
			}
			b.Truncate(b.Len() - 1) // Encode's newline
		}
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

func safe(field, v string) error {
	if len(v) > MaxField {
		return fmt.Errorf("%w: %s longer than %d bytes", ErrInvalid, field, MaxField)
	}
	for _, c := range v {
		if unicode.IsControl(c) || c == ' ' || c == ' ' {
			return fmt.Errorf("%w: %s has a control or separator character", ErrInvalid, field)
		}
	}
	return nil
}

// HashOf is hex(SHA-256(prev ‖ canonical(r))).
func HashOf(r Row) (string, error) {
	c, err := Canonical(r)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write([]byte(r.Prev))
	h.Write(c)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Store keeps one cell's rows. Append holds the cell's write lock, calls build with the
// last row's seq and hash (0 and Zero when empty), and inserts what build returns.
type Store interface {
	Append(ctx context.Context, build func(lastSeq int64, lastHash string) (Row, error)) (Row, error)
	Range(ctx context.Context, fromSeq int64, limit int) ([]Row, error) // seq >= fromSeq, ascending
}

// Log writes rows for one cell.
type Log struct {
	Tenant  string
	Release func() string
	Store   Store
	Now     func() time.Time
	mu      sync.Mutex
}

// Write appends one event and returns the committed row. It returns only after the store
// has committed, so a caller that waits for it audits before it acts (LAW-04).
func (l *Log) Write(ctx context.Context, e Event) (Row, error) {
	if !Kinds[e.Kind] {
		return Row{}, fmt.Errorf("%w: unknown kind %q", ErrInvalid, e.Kind)
	}
	if e.Actor == "" || e.Subject == "" || e.Reason == "" {
		return Row{}, fmt.Errorf("%w: actor, subject and reason are required", ErrInvalid)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.Store.Append(ctx, func(lastSeq int64, lastHash string) (Row, error) {
		r := Row{Seq: lastSeq + 1, At: l.Now().UTC().Format("2006-01-02T15:04:05.000Z"), Tenant: l.Tenant,
			Actor: e.Actor, Kind: e.Kind, Subject: e.Subject, State: e.State, Approval: e.Approval,
			PayloadSHA256: e.PayloadSHA256, Receipt: e.Receipt, Release: l.Release(), Reason: e.Reason, Prev: lastHash}
		r.ID = fmt.Sprintf("aud_%012d", r.Seq)
		h, err := HashOf(r)
		if err != nil {
			return Row{}, err
		}
		r.Hash = h
		return r, nil
	})
}

// RecordDecision makes the log a policy.Recorder: a decision row, written before the
// kernel returns the decision (sb-audit(7), Policy decisions).
func (l *Log) RecordDecision(ctx context.Context, d policy.Record) error {
	actor := d.Actor
	if actor == "" {
		actor = "sys_kernel"
	}
	_, err := l.Write(ctx, Event{Actor: actor, Kind: "decision", Subject: d.Subject, State: d.Decision.String(),
		Reason: d.Reason}) // the kernel's reason already starts with the action
	return err
}

var _ policy.Recorder = (*Log)(nil)
