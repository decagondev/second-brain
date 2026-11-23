package cell

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/audit/syslogtls"
	"example.com/second-brain/internal/adapters/audit/syslogtls/syslogtest"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/auditstream"
)

var seqOf = regexp.MustCompile(` seq="(\d+)" `)

// TestAuditStreamToASIEM (SB-2302): a running cell streams every audit row to a fake SIEM
// in order; the SIEM goes down, writes go on, a Sev-2 alert fires after an hour, and the
// jobs service restarts; when the SIEM is back it gets every row, each once, in order.
func TestAuditStreamToASIEM(t *testing.T) {
	r, svc, c, got := jobsCell(t, false)
	ctx := context.Background()
	c.t = time.Now().UTC() // the audit log writes on the real clock
	siem := syslogtest.New(t)
	r.cell.Config.Audit.Stream = "syslog+tls://" + siem.Addr
	send := func() auditstream.Sender { return &syslogtls.Sender{Addr: siem.Addr, Roots: siem.Roots} }
	write := func(n int) {
		for i := 0; i < n; i++ {
			if _, err := r.cell.Audit.Write(ctx, audit.Event{Actor: "usr_ann", Kind: "config", Subject: fmt.Sprintf("x_%d", i), Reason: "stream test"}); err != nil {
				t.Fatalf("a write failed with the SIEM down: %v", err)
			}
		}
	}
	st, ok := svc.AuditStream(send())
	if !ok {
		t.Fatal("no stream with audit.stream set")
	}
	var last time.Time
	write(3)
	svc.StreamStep(ctx, st, &last)

	siem.Down()
	write(4)
	for i := 0; i < 3; i++ { // a closed peer may take a write or two to notice
		svc.StreamStep(ctx, st, &last)
	}
	c.t = c.t.Add(61 * time.Minute)
	svc.StreamStep(ctx, st, &last)
	behind := 0
	for _, a := range got.got {
		if a.Kind == "audit_stream_behind" && a.Severity == "sev2" {
			behind++
		}
	}
	if behind != 1 {
		t.Fatalf("an hour behind raised %d Sev-2 alerts: %+v", behind, got.got)
	}

	// The jobs service restarts: a new stream from the cursor in the cell's state.
	if err := siem.Up(); err != nil {
		t.Fatal(err)
	}
	st2, _ := svc.AuditStream(send())
	write(2)
	if wait := svc.StreamStep(ctx, st2, &last); wait != 0 && wait > time.Minute {
		t.Fatalf("the stream backs off with the SIEM up: %v", wait)
	}
	rows, err := r.cell.Audit.Store.Range(ctx, 1, 1000)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && len(siem.Got()) < len(rows) {
		time.Sleep(10 * time.Millisecond)
	}
	// Each row once, in order. A row the SIEM's closed connection swallowed before the
	// cell noticed is sent again; the SIEM drops a seq it has (sb-audit(7)).
	var seqs []string
	seen := map[string]bool{}
	for _, m := range siem.Got() {
		s := seqOf.FindStringSubmatch(m)
		if s == nil {
			t.Fatalf("not a row: %s", m)
		}
		if !seen[s[1]] {
			seen[s[1]] = true
			seqs = append(seqs, s[1])
		}
	}
	if len(seqs) != len(rows) {
		t.Fatalf("the SIEM has %d of %d rows: %v", len(seqs), len(rows), seqs)
	}
	for i, s := range seqs {
		if s != fmt.Sprint(rows[i].Seq) {
			t.Fatalf("out of order at %d: %v", i, seqs)
		}
	}
}
