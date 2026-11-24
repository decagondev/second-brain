package cli

import (
	"testing"

	"example.com/second-brain/internal/core/audit"
)

// SB-3204: sb audit tail's --actor, --since and --until narrow by person and whole day.
func TestTailFilter(t *testing.T) {
	f := tailFilter{actor: "usr_ann", since: "2027-03-10", until: "2027-03-11"}
	for _, c := range []struct {
		r    audit.Row
		want bool
	}{
		{audit.Row{Actor: "usr_ann", At: "2027-03-10T00:00:00.000Z"}, true},
		{audit.Row{Actor: "usr_ann", At: "2027-03-11T23:59:59.999Z"}, true},
		{audit.Row{Actor: "usr_ann", At: "2027-03-09T23:59:59.999Z"}, false},
		{audit.Row{Actor: "usr_ann", At: "2027-03-12T00:00:00.000Z"}, false},
		{audit.Row{Actor: "usr_bo", At: "2027-03-10T12:00:00.000Z"}, false},
	} {
		if got := f.match(c.r); got != c.want {
			t.Errorf("%+v: %v", c.r, got)
		}
	}
	if code, _, _ := run("audit", "tail", "--since", "yesterday", "--cell", "http://127.0.0.1:1"); code != 2 {
		t.Fatalf("a bad date: exit %d", code)
	}
}
