package syslogtls

import (
	"context"
	"crypto/x509"
	"path/filepath"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/audit/syslogtls/syslogtest"
)

func TestParse(t *testing.T) {
	if a, err := Parse("syslog+tls://siem.acme.example:6514"); err != nil || a != "siem.acme.example:6514" {
		t.Fatalf("%q %v", a, err)
	}
	for _, bad := range []string{"syslog://siem:514", "syslog+tls://siem", "syslog+tls://u:p@siem:6514", "syslog+tls://siem:0", "https://siem:6514", "syslog+tls://siem:6514/x", "syslog+tls://:6514"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("%s accepted", bad)
		}
	}
}

func TestSenderVerifiesAndRedials(t *testing.T) {
	ctx := context.Background()
	rx := syslogtest.New(t)
	s := &Sender{Addr: rx.Addr, Roots: rx.Roots}
	if err := s.Send(ctx, []byte("<110>1 one")); err != nil {
		t.Fatal(err)
	}
	// A SIEM whose certificate the cell doesn't trust is never sent to.
	untrusted := &Sender{Addr: rx.Addr, Roots: x509.NewCertPool()}
	if err := untrusted.Send(ctx, []byte("<110>1 leak")); err == nil {
		t.Fatal("sent to a SIEM with an untrusted certificate")
	}
	waitFor(t, rx, 1)
	rx.Down()
	var failed bool
	time.Sleep(20 * time.Millisecond) // the SIEM's close reaches the cell
	failed = s.Send(ctx, []byte("<110>1 lost")) != nil
	if !failed {
		t.Fatal("a send to a SIEM that hung up didn't fail: the row would be lost")
	}
	if err := rx.Up(); err != nil {
		t.Fatal(err)
	}
	if err := s.Send(ctx, []byte("<110>1 two")); err != nil {
		t.Fatalf("no redial after the SIEM came back: %v", err)
	}
	_ = s.Close()
	waitFor(t, rx, 2)
	got := rx.Got()
	if len(got) < 2 || got[0] != "<110>1 one" || got[len(got)-1] != "<110>1 two" {
		t.Fatalf("got %q", got)
	}
}

func TestFileCursor(t *testing.T) {
	ctx := context.Background()
	c := FileCursor{Path: filepath.Join(t.TempDir(), "audit", "stream.json")}
	if n, err := c.Load(ctx); err != nil || n != 0 {
		t.Fatalf("%d %v", n, err)
	}
	if err := c.Save(ctx, 42); err != nil {
		t.Fatal(err)
	}
	if n, err := (FileCursor{Path: c.Path}).Load(ctx); err != nil || n != 42 {
		t.Fatalf("after a restart: %d %v", n, err)
	}
}

func waitFor(t *testing.T, rx *syslogtest.Receiver, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && len(rx.Got()) < n {
		time.Sleep(10 * time.Millisecond)
	}
}
