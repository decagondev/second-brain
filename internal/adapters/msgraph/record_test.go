package msgraph

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"example.com/second-brain/internal/adapters/msgraph/graphtest"
	"example.com/second-brain/internal/core/record"
)

// SB-6002: the fake answers every recorded step with at least the recording's fields, of
// the same types; a fake missing a field the recording has fails.
func TestFakeMatchesTheRecording(t *testing.T) {
	b, err := os.ReadFile("testdata/recording.json")
	if err != nil {
		t.Fatal(err)
	}
	var rec record.Recording
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatal(err)
	}
	f := graphtest.Seeded()
	t.Cleanup(f.Server.Close)
	g := &Client{Base: f.Base(), Login: f.Login(), HTTP: f.Server.Client(), Creds: func(context.Context) (string, error) { return graphtest.Creds, nil }}
	ex, err := g.Script(context.Background(), []string{"d1"}, "sam@acme.example", "support@acme.example")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]map[string]string{}
	for _, x := range ex {
		got[x.Step] = record.Shape(x.Body)
	}
	for _, x := range rec.Exchanges {
		g, ok := got[x.Step]
		if !ok {
			t.Errorf("the fake has no step %s", x.Step)
			continue
		}
		if m := record.Missing(record.Shape(x.Body), g); len(m) > 0 {
			t.Errorf("%s: the fake lacks %v", x.Step, m)
		}
	}
	// A fake that drops a field is caught.
	want := map[string]string{"value": "array", "value[].id": "string", "value[].webUrl": "string"}
	if m := record.Missing(want, map[string]string{"value": "array", "value[].id": "string"}); len(m) != 1 || m[0] != "value[].webUrl" {
		t.Fatalf("missing: %v", m)
	}
}
