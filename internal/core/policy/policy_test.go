package policy

import (
	"context"
	"testing"

	"example.com/second-brain/internal/core/tenancy"
)

type allow struct{}

func (allow) Evaluate(context.Context, Request) (Result, error) { return Result{Allow, "test"}, nil }

type rows struct{ n int }

func (r *rows) RecordDecision(context.Context, Record) error { r.n++; return nil }

func TestReadSampling(t *testing.T) {
	s, _ := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", "usr_1", nil)
	res := Entity{Type: "Chunk", ID: "c1"}
	for _, c := range []struct {
		sample, roll float64
		want         int
	}{{0, 0, 0}, {1, 0.99, 1}, {0.01, 0.005, 1}, {0.01, 0.5, 0}} {
		r := &rows{}
		k := Kernel{Engine: allow{}, Recorder: r, ReadSample: c.sample, Rand: func() float64 { return c.roll }}
		if got := k.Authorise(context.Background(), s, "read", res, nil); got.Decision != Allow || r.n != c.want {
			t.Errorf("sample %v roll %v: %v, %d rows; want %d", c.sample, c.roll, got.Decision, r.n, c.want)
		}
	}
	// No recorder: anything that must be recorded is Unknown.
	if got := (Kernel{Engine: allow{}}).Authorise(context.Background(), s, "call_tool", Entity{Type: "Tool", ID: "t"}, nil); got.Decision != Unknown {
		t.Fatalf("no recorder: %v", got.Decision)
	}
}

func TestReadSampleFrom(t *testing.T) {
	env := func(v string) func(string) string { return func(string) string { return v } }
	if f, err := ReadSampleFrom(env("")); err != nil || f != DefaultReadSample {
		t.Fatalf("%v %v", f, err)
	}
	if f, err := ReadSampleFrom(env("0.25")); err != nil || f != 0.25 {
		t.Fatalf("%v %v", f, err)
	}
	for _, bad := range []string{"x", "-0.1", "1.5"} {
		if _, err := ReadSampleFrom(env(bad)); err == nil {
			t.Errorf("%q: want an error", bad)
		}
	}
}
