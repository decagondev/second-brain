package approvals

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestReportingLine(t *testing.T) {
	m := map[string]string{"a": "b", "b": "c", "x": "y", "y": "x"}
	get := func(_ context.Context, u string) (string, error) { return m[u], nil }
	if l, err := ReportingLine(context.Background(), get, "a"); err != nil || fmt.Sprint(l) != "[b c]" {
		t.Fatalf("line %v %v", l, err)
	}
	if l, err := ReportingLine(context.Background(), get, "x"); err != nil || fmt.Sprint(l) != "[y]" {
		t.Fatalf("a loop must end the line: %v %v", l, err)
	}
	long := func(_ context.Context, u string) (string, error) { return u + "+", nil }
	if _, err := ReportingLine(context.Background(), long, "z"); !errors.Is(err, ErrReportingLine) {
		t.Fatalf("a line longer than %d: %v", MaxReportingLine, err)
	}
	bad := func(context.Context, string) (string, error) { return "", errors.New("down") }
	if _, err := ReportingLine(context.Background(), bad, "a"); !errors.Is(err, ErrReportingLine) {
		t.Fatalf("an unreadable line must refuse: %v", err)
	}
}

func TestAmount(t *testing.T) {
	for in, want := range map[string]int64{`{"amount": 5}`: 5, `{"amount": -3, "x": 1}`: -3} {
		if v, ok := Amount([]byte(in)); !ok || v != want {
			t.Errorf("%s: %d %v", in, v, ok)
		}
	}
	for _, in := range []string{`{}`, `{"amount": "5"}`, `{"amount": 1.5}`, `not json`, `{"amount": 1e30}`} {
		if _, ok := Amount([]byte(in)); ok {
			t.Errorf("%s has no whole amount", in)
		}
	}
}
