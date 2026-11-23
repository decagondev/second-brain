package laws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"time"

	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/auditstream"
)

// The audit stream to a SIEM (SB-2302) under the laws.
//
// LAW-11: the SIEM gets every row of the chain, in order, each one it can check (its hash
// and its link to the one before), across a SIEM that goes down and a stream that
// restarts; the stream never blocks a write. LAW-07: each message is the row exactly as
// the export writes it, and nothing else. The real stream over the memory audit log and a
// receiver that is sometimes down.

// StreamSystem turns the receiver's sender into the stream's.
type StreamSystem interface {
	Sender(real auditstream.Sender) auditstream.Sender
}

// RealStream is the cell's: the sender as it is, so a row the SIEM didn't take is sent again.
type RealStream struct{}

// Sender implements StreamSystem.
func (RealStream) Sender(real auditstream.Sender) auditstream.Sender { return real }

// siem receives messages, and is down when down is set.
type siem struct {
	down bool
	got  [][]byte
}

var errDown = errors.New("siem: down")

func (s *siem) Send(_ context.Context, msg []byte) error {
	if s.down {
		return errDown
	}
	s.got = append(s.got, append([]byte(nil), msg...))
	return nil
}

type memCursor struct{ seq int64 }

func (c *memCursor) Load(context.Context) (int64, error)     { return c.seq, nil }
func (c *memCursor) Save(_ context.Context, seq int64) error { c.seq = seq; return nil }

var streamMsg = regexp.MustCompile(`^<110>1 (\S+) cell\.acme\.example second-brain - (\S+) \[sb@32473 tenant="acme-logistics" seq="(\d+)" hash="([0-9a-f]{64})"\] (.*)$`)

func law11Stream(r *rand.Rand, sys StreamSystem) error {
	ctx := context.Background()
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	st := auditmem.New()
	alog := &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_law" }, Store: st, Now: clock}
	rx := &siem{}
	cur := &memCursor{} // the cursor file outlives a restart
	stream := func() auditstream.Stream {
		return auditstream.Stream{Tenant: "acme-logistics", Host: "cell.acme.example", Store: st, Cursor: cur, Send: sys.Sender(rx), Now: clock}
	}
	s := stream()
	kinds := []string{"tool_call", "decision", "approval", "config", "access"}
	written := 0
	for step := 0; step < 10+r.Intn(30); step++ {
		switch r.Intn(5) {
		case 0:
			rx.down = !rx.down
		case 1:
			s = stream() // a restart
		default:
			for i := 0; i < 1+r.Intn(4); i++ {
				// A write never waits for the SIEM.
				if _, err := alog.Write(ctx, audit.Event{Actor: "usr_ann", Kind: kinds[r.Intn(len(kinds))], Subject: fmt.Sprintf("x_%d", r.Intn(99)), Reason: "law", State: "ok"}); err != nil {
					return fmt.Errorf("a write failed while the SIEM was down %v: %w", rx.down, err)
				}
				written++
			}
		}
		now = now.Add(time.Duration(r.Intn(600)) * time.Second)
		if _, err := s.Step(ctx); err != nil && !errors.Is(err, errDown) {
			return err
		}
	}
	rx.down = false
	if _, err := stream().Step(ctx); err != nil {
		return err
	}
	rows, err := st.Range(ctx, 1, written+1)
	if err != nil {
		return err
	}
	if len(rx.got) != len(rows) {
		return fmt.Errorf("LAW-11: the SIEM got %d messages for %d rows", len(rx.got), len(rows))
	}
	prev := audit.Zero
	for i, msg := range rx.got {
		m := streamMsg.FindSubmatch(msg)
		if m == nil {
			return fmt.Errorf("message %d isn't the stream's RFC 5424 form: %q", i, msg)
		}
		var row audit.Row
		if err := json.Unmarshal(m[5], &row); err != nil {
			return fmt.Errorf("message %d's text isn't a row: %v", i, err)
		}
		seq, _ := strconv.ParseInt(string(m[3]), 10, 64)
		if row.Seq != int64(i+1) || seq != row.Seq {
			return fmt.Errorf("LAW-11: message %d carries row %d (header %d): out of order or missing", i, row.Seq, seq)
		}
		if h, err := audit.HashOf(row); err != nil || h != row.Hash || row.Prev != prev || string(m[4]) != row.Hash {
			return fmt.Errorf("LAW-11: row %d as streamed doesn't check: its hash or its link to the row before", row.Seq)
		}
		prev = row.Hash
		var want bytes.Buffer
		enc := json.NewEncoder(&want)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(rows[i])
		if !bytes.Equal(bytes.TrimRight(want.Bytes(), "\n"), m[5]) {
			return fmt.Errorf("LAW-07: message %d isn't the row as the export has it", i)
		}
	}
	return nil
}
