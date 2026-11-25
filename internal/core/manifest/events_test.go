package manifest

import (
	"reflect"
	"testing"
)

// SB-4404, SB-4405: events need an https URL and known kinds, each once, and survive a
// format; a space may pin model: main and nothing else.
func TestEventsAndModelPin(t *testing.T) {
	m, diags := Check([]byte(valid + "events: { url: https://hooks.acme.example/brain, kinds: [decision, action_due] }\n"))
	if len(diags) > 0 || m.Events == nil || m.Events.URL != "https://hooks.acme.example/brain" || !reflect.DeepEqual(m.Events.Kinds, []string{"decision", "action_due"}) {
		t.Fatalf("events: %v %+v", diags, m.Events)
	}
	if again, ds := Check(Format(m)); len(ds) > 0 || !reflect.DeepEqual(again.Events, m.Events) {
		t.Fatalf("format: %v %+v\n%s", ds, again.Events, Format(m))
	}
	for _, bad := range []string{
		"events: { url: http://hooks.acme.example, kinds: [decision] }",
		"events: { url: https://hooks.acme.example, kinds: [] }",
		"events: { url: https://hooks.acme.example, kinds: [decision, decision] }",
		"events: { url: https://hooks.acme.example, kinds: [titles] }",
		"events: { url: https://hooks.acme.example, kinds: [decision], secret: x }",
		"events: { kinds: [decision] }",
	} {
		if _, diags := Check([]byte(valid + bad + "\n")); len(diags) == 0 {
			t.Errorf("accepted: %s", bad)
		}
	}
	sp := "spaces:\n  ops:\n    members: [group:ops]\n    scope: { docs: [\"/Ops/**\"] }\n    model: "
	m, diags = Check([]byte(valid + sp + "main\n"))
	if len(diags) > 0 || m.Spaces["ops"].Model != "main" {
		t.Fatalf("model: main: %v", diags)
	}
	if again, ds := Check(Format(m)); len(ds) > 0 || again.Spaces["ops"].Model != "main" {
		t.Fatalf("format: %v", ds)
	}
	if _, diags := Check([]byte(valid + sp + "small\n")); len(diags) == 0 {
		t.Error("model: small accepted")
	}
}
