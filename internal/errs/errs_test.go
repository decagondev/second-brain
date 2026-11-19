package errs

import (
	"errors"
	"fmt"
	"testing"
)

func TestEveryCodeHasExitAndRemedy(t *testing.T) {
	for _, c := range Codes() {
		r := registry[c]
		if r.exit < 1 || r.exit > 9 {
			t.Errorf("%s: exit %d outside 1..9", c, r.exit)
		}
		if r.remedy == "" || r.meaning == "" {
			t.Errorf("%s: missing meaning or remedy", c)
		}
	}
}

func TestExit(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, 0},
		{New(EPERM, "refused"), 3},
		{New(ENOTIMPL, "x"), 2},
		{New(EKILLED, "x"), 4},
		{fmt.Errorf("wrapped: %w", New(EBUDGET, "x")), 8},
		{errors.New("uncoded"), 1}, // an uncoded failure is never success
	}
	for _, c := range cases {
		if got := Exit(c.err); got != c.want {
			t.Errorf("Exit(%v) = %d, want %d", c.err, got, c.want)
		}
	}
}

func TestUnknownCodePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("New with an unregistered code must panic")
		}
	}()
	_ = New(Code("ENOPE"), "x")
}

func TestHTTP(t *testing.T) {
	if HTTP(EPERM) != 403 || HTTP(ECHAIN) != 0 || HTTP(ENOTIMPL) != 501 {
		t.Fatal("HTTP statuses do not match sb-errors(7)")
	}
}
