package auditstream

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/audit"
)

type rx struct {
	down bool
	got  []string
}

func (r *rx) Send(_ context.Context, m []byte) error {
	if r.down {
		return errors.New("down")
	}
	r.got = append(r.got, string(m))
	return nil
}

// rows is an audit store in memory.
type rows struct{ rs []audit.Row }

func (s *rows) Append(_ context.Context, build func(int64, string) (audit.Row, error)) (audit.Row, error) {
	seq, prev := int64(0), audit.Zero
	if n := len(s.rs); n > 0 {
		seq, prev = s.rs[n-1].Seq, s.rs[n-1].Hash
	}
	r, err := build(seq, prev)
	if err == nil {
		s.rs = append(s.rs, r)
	}
	return r, err
}

func (s *rows) Range(_ context.Context, from int64, limit int) ([]audit.Row, error) {
	var out []audit.Row
	for _, r := range s.rs {
		if r.Seq >= from && len(out) < limit {
			out = append(out, r)
		}
	}
	return out, nil
}

type cur struct{ n int64 }

func (c *cur) Load(context.Context) (int64, error)   { return c.n, nil }
func (c *cur) Save(_ context.Context, n int64) error { c.n = n; return nil }

func TestStep(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	now := at
	st := &rows{}
	log := &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_x" }, Store: st, Now: func() time.Time { return at }}
	write := func(n int) {
		for i := 0; i < n; i++ {
			if _, err := log.Write(ctx, audit.Event{Actor: "usr_ann", Kind: "config", Subject: fmt.Sprintf("s%d", i), Reason: `a "quoted" ] reason`}); err != nil {
				t.Fatal(err)
			}
		}
	}
	r := &rx{}
	c := &cur{}
	s := Stream{Tenant: "acme-logistics", Host: "cell.acme.example", Store: st, Cursor: c, Send: r, Now: func() time.Time { return now }}
	write(3)
	if res, err := s.Step(ctx); err != nil || res.Sent != 3 || c.n != 3 {
		t.Fatalf("%+v %v cursor %d", res, err, c.n)
	}
	if !strings.HasPrefix(r.got[0], `<110>1 2026-11-23T09:00:00.000Z cell.acme.example second-brain - config [sb@32473 tenant="acme-logistics" seq="1" hash="`) ||
		!strings.Contains(r.got[0], `"reason":"a \"quoted\" ] reason"`) {
		t.Fatalf("message: %s", r.got[0])
	}
	// Down: nothing is lost, the cursor waits, and Behind says how long.
	r.down = true
	write(2)
	now = at.Add(90 * time.Minute)
	res, err := s.Step(ctx)
	if err == nil || res.Sent != 0 || c.n != 3 || res.Behind != 90*time.Minute {
		t.Fatalf("down: %+v %v cursor %d", res, err, c.n)
	}
	r.down = false
	if res, err := s.Step(ctx); err != nil || res.Sent != 2 || len(r.got) != 5 || res.Behind != 0 {
		t.Fatalf("back: %+v %v", res, err)
	}
	// Rows older than a day when the SIEM is back are named in a gap, not sent.
	r.down = true
	write(4)
	at = now.Add(30 * time.Hour)
	write(1)
	now = at
	_, _ = s.Step(ctx)
	r.down = false
	res, err = s.Step(ctx)
	if err != nil || res.Skipped != 4 || res.Sent != 1 {
		t.Fatalf("gap: %+v %v", res, err)
	}
	if g := r.got[5]; !strings.Contains(g, ` gap [sb@32473 tenant="acme-logistics" from="6" to="9"]`) || !strings.HasPrefix(g, "<108>1 ") {
		t.Fatalf("gap message: %s", g)
	}
	if !strings.Contains(r.got[6], `seq="10"`) || c.n != 10 {
		t.Fatalf("after the gap: %s cursor %d", r.got[6], c.n)
	}
	if string(Frame([]byte("abc"))) != "3 abc" {
		t.Fatal("frame")
	}
}
