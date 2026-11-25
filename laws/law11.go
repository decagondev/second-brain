package laws

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
)

// LAW-11 (SB-401, SB-402): the audit chain is unbroken, and any change to an exported
// chain is detected: an edited, dropped, reordered, duplicated or inserted row, a cut-off
// tail, or a forged anchor.

// AuditChain is one cell's audit log, its export and the offline verifier.
type AuditChain interface {
	Append(kind, subject string) error
	Export() ([]byte, error)                       // NDJSON rows with anchors (sb-audit(5))
	Verify(export []byte) (intact bool, err error) // err is a broken or unreadable file, never nil with intact=false
}

// NewAuditChain builds the system under test.
var NewAuditChain = func() AuditChain { return pendingChain{} }

type pendingChain struct{}

func (pendingChain) Append(string, string) error { return nil }
func (pendingChain) Export() ([]byte, error) {
	return nil, fmt.Errorf("%w: the audit chain arrives with SB-401", ErrPending)
}
func (pendingChain) Verify([]byte) (bool, error) { return false, nil }

var auditKinds = []string{"tool_call", "decision", "approval", "kill", "config", "release", "access", "custody", "retention", "guardrail"}

func law11(r *rand.Rand, c AuditChain) error {
	n := 2 + r.Intn(30)
	for i := 0; i < n; i++ {
		if err := c.Append(auditKinds[r.Intn(len(auditKinds))], fmt.Sprintf("tc_%04d", r.Intn(10000))); err != nil {
			return err
		}
	}
	exp, err := c.Export()
	if err != nil {
		return err
	}
	if ok, err := c.Verify(exp); !ok || err != nil {
		return fmt.Errorf("an untouched export doesn't verify: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(exp), "\n"), "\n")
	var rows []int
	for i, l := range lines {
		if !strings.Contains(l, `"anchor":true`) {
			rows = append(rows, i)
		}
	}
	if len(rows) != n {
		return fmt.Errorf("export has %d rows, want %d", len(rows), n)
	}
	pick := func() int { return rows[r.Intn(len(rows))] }
	tampered := append([]string(nil), lines...)
	var what string
	switch r.Intn(7) {
	case 0:
		i := pick()
		tampered[i] = strings.Replace(tampered[i], `"subject":"tc_`, `"subject":"tc_9`, 1)
		what = "edited subject"
	case 1:
		i := pick()
		tampered = append(tampered[:i:i], tampered[i+1:]...)
		what = "dropped row"
	case 2:
		i, j := rows[0], rows[len(rows)-1]
		tampered[i], tampered[j] = tampered[j], tampered[i]
		what = "swapped rows"
	case 3:
		i := pick()
		tampered = append(tampered[:i+1:i+1], tampered[i:]...)
		what = "duplicated row"
	case 4:
		tampered = tampered[:rows[len(rows)-1]] // the last row and the final anchor are gone
		what = "cut-off tail"
	case 5:
		last := len(tampered) - 1
		tampered[last] = strings.Replace(tampered[last], `"sig":"`, `"sig":"AAAA`, 1)
		what = "forged anchor signature"
	default:
		i := pick()
		tampered[i] = strings.Replace(tampered[i], `"kind":"`, `"kind":"x`, 1)
		what = "edited kind"
	}
	out := []byte(strings.Join(tampered, "\n") + "\n")
	if bytes.Equal(out, exp) {
		return nil // the tamper was a no-op (e.g. swapping a one-row chain)
	}
	if ok, _ := c.Verify(out); ok {
		return fmt.Errorf("a %s was not detected", what)
	}
	return nil
}
