package manifest

import (
	"strings"
	"testing"
)

// SB-4905: a space opts in to source-changed redrafts with on; nothing else is a trigger,
// and Format reads it back.
func TestSpaceOnSourceChanged(t *testing.T) {
	with := func(line string) string {
		return strings.Replace(treeHead, "    label: confidential\n", "    label: confidential\n"+line, 1)
	}
	for _, bad := range []string{"    on: [file-changed]\n", "    on: [source-changed, source-changed]\n", "    on: source-changed\n"} {
		if _, ds := Check([]byte(with(bad))); len(ds) == 0 {
			t.Errorf("%q was accepted", bad)
		}
	}
	got, ds := Check([]byte(with("    on: [source-changed]\n")))
	if len(ds) > 0 || !got.Spaces["finance"].Watches("source-changed") {
		t.Fatalf("on: [source-changed]: %v %+v", ds, got.Spaces["finance"].On)
	}
	again, ds := Check(Format(got))
	if len(ds) > 0 || !again.Spaces["finance"].Watches("source-changed") {
		t.Fatalf("Format dropped on: %v\n%s", ds, Format(got))
	}
	if plain, _ := Check([]byte(treeHead)); plain.Spaces["finance"].Watches("source-changed") {
		t.Fatal("a space watches without on")
	}
}
