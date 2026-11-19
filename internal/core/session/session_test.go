package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ports"
)

type dir struct {
	active, known bool
	err           error
	calls         int
}

func (d *dir) Active(context.Context, string, string) (bool, bool, error) {
	d.calls++
	return d.active, d.known, d.err
}

var t0 = time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC)

func TestRules(t *testing.T) {
	s := ports.Session{ID: "s", Tenant: "acme-logistics", User: "usr_1", SignedIn: t0, Validated: t0}
	ctx := context.Background()
	ok := &dir{active: true, known: true}
	if _, changed, err := Check(ctx, t0.Add(59*time.Second), s, ok); err != nil || changed || ok.calls != 0 {
		t.Fatalf("within 60s: changed=%v err=%v calls=%d", changed, err, ok.calls)
	}
	got, changed, err := Check(ctx, t0.Add(60*time.Second), s, ok)
	if err != nil || !changed || !got.Validated.Equal(t0.Add(60*time.Second)) || ok.calls != 1 {
		t.Fatalf("at 60s: %+v %v %v", got, changed, err)
	}
	if _, _, err := Check(ctx, t0.Add(MaxAge), got, ok); !errors.Is(err, ErrExpired) {
		t.Fatalf("at 12h: %v", err)
	}
	for name, d := range map[string]*dir{
		"inactive": {active: false, known: true}, "unknown user": {active: true, known: false}, "no answer": {err: errors.New("timeout")},
	} {
		if _, _, err := Check(ctx, t0.Add(2*time.Minute), s, d); err == nil {
			t.Errorf("%s: want the session to end", name)
		}
	}
}
