package drift

import (
	"testing"

	"example.com/second-brain/internal/core/ports"
)

func TestMessages(t *testing.T) {
	applied := []ports.Resource{{Kind: "connector", Name: "tickets", Attrs: map[string]string{"vendor": "jira"}}, {Kind: "agent", Name: "ask"}}
	live := []ports.Resource{{Kind: "connector", Name: "tickets", Attrs: map[string]string{"vendor": "linear"}}, {Kind: "agent", Name: "draft-reply"}}
	got := Detect(applied, live)
	want := []string{
		"agent ask: applied, not present live",
		"agent draft-reply: present live, not applied",
		`connector tickets: attribute vendor: applied "jira", live "linear"`,
	}
	if len(got) != len(want) {
		t.Fatalf("%+v", got)
	}
	for i := range want {
		if got[i].Message() != want[i] {
			t.Errorf("%q, want %q", got[i].Message(), want[i])
		}
	}
	if len(Detect(applied, applied)) != 0 {
		t.Fatal("no drift should report nothing")
	}
}
