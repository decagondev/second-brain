package approvals

import (
	"errors"
	"testing"
)

func TestCanonical(t *testing.T) {
	cases := map[string]string{
		`{"b":1, "a":[true,null,"x<y"]}`: `{"a":[true,null,"x<y"],"b":1}`,
		`{"é":1,"z":2,"A":3}`:            `{"A":3,"z":2,"é":1}`,
		`"plain"`:                        `"plain"`,
	}
	for in, want := range cases {
		got, err := Canonical([]byte(in))
		if err != nil || string(got) != want {
			t.Errorf("%s: got %s %v, want %s", in, got, err, want)
		}
	}
	for _, bad := range []string{`{"a":1.5}`, `{"a":9007199254740993}`, `{"a":1} {}`, `nope`} {
		if _, err := Canonical([]byte(bad)); !errors.Is(err, ErrPayload) {
			t.Errorf("%s accepted", bad)
		}
	}
}

func TestHashBindsPayloadTargetAndWorkflow(t *testing.T) {
	h1, _ := Hash("ticket-update", "tickets:OPS-1", []byte(`{"comment":"x","n":1}`))
	h2, _ := Hash("ticket-update", "tickets:OPS-1", []byte(`{ "n":1 , "comment":"x"}`))
	h3, _ := Hash("ticket-update", "tickets:OPS-2", []byte(`{"comment":"x","n":1}`))
	h4, _ := Hash("other", "tickets:OPS-1", []byte(`{"comment":"x","n":1}`))
	if h1 != h2 || h1 == h3 || h1 == h4 || len(h1) != 64 {
		t.Fatalf("%s %s %s %s", h1, h2, h3, h4)
	}
}

func TestTransitionsMatchTheModel(t *testing.T) {
	for _, ok := range [][2]State{{Drafted, PendingApproval}, {PendingApproval, Approved}, {Approved, Executing}, {Executing, Ran}, {Executing, DidNotRun}, {Approved, Drafted}} {
		if !CanMove(ok[0], ok[1]) {
			t.Errorf("%s -> %s refused", ok[0], ok[1])
		}
	}
	for _, bad := range [][2]State{{Drafted, Executing}, {PendingApproval, Executing}, {Ran, Executing}, {Rejected, Approved}, {DidNotRun, Ran}} {
		if CanMove(bad[0], bad[1]) {
			t.Errorf("%s -> %s allowed", bad[0], bad[1])
		}
	}
}
