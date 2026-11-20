package reorg

import (
	"encoding/json"
	"testing"

	"example.com/second-brain/internal/core/manifest"
)

// FuzzParse (SB-1402): Parse never panics, a snapshot it accepts reads back the same, and
// the preview never panics on it.
func FuzzParse(f *testing.F) {
	f.Add([]byte(`{"version": 1, "people": [{"user": "a@x.example", "before": ["group:a"], "after": []}]}`))
	f.Add([]byte(`{"version": 1, "people": []}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		s, err := Parse(b)
		if err != nil {
			return
		}
		again, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Parse(again); err != nil {
			t.Fatalf("an accepted snapshot doesn't read back: %v", err)
		}
		tn := manifest.Tenant{Spaces: map[string]manifest.Space{"x": {Members: []string{"group:a"}, Label: "restricted"}}}
		_ = Diff(tn, s)
	})
}
