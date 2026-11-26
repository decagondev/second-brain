package ingest

import (
	"reflect"
	"testing"
)

// SB-5403: a group, a domain or anyone added widens; a person added, or a grant removed,
// doesn't.
func TestSharingChangeWidens(t *testing.T) {
	for _, c := range []struct {
		before, after []string
		widened       bool
		added         []string
	}{
		{[]string{"ann@acme.example"}, []string{"ann@acme.example", "domain:acme.example"}, true, []string{"domain:acme.example"}},
		{[]string{"ann@acme.example"}, []string{"anyone", "ann@acme.example"}, true, []string{"anyone"}},
		{nil, []string{"group:finance@acme.example"}, true, []string{"group:finance@acme.example"}},
		{[]string{"ann@acme.example"}, []string{"ann@acme.example", "kim@acme.example"}, false, []string{"kim@acme.example"}},
		{[]string{"domain:acme.example", "ann@acme.example"}, []string{"ann@acme.example"}, false, nil},
	} {
		ch := SharingChangeOf(c.before, c.after)
		if ch.Widened() != c.widened || !reflect.DeepEqual(ch.Added, c.added) || !ch.Changed() {
			t.Fatalf("%v -> %v: %+v", c.before, c.after, ch)
		}
	}
	if SharingChangeOf([]string{"a"}, []string{"a"}).Changed() {
		t.Fatal("no change changed")
	}
}
