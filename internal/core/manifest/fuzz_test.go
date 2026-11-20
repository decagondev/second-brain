package manifest

import (
	"os"
	"reflect"
	"testing"
)

// FuzzCheck (SB-1402): Check never panics, and a manifest it accepts reads back the same
// after Format: what is checked is what is applied.
func FuzzCheck(f *testing.F) {
	for _, p := range []string{"../../../examples/tenants/pilot.yaml", "../../../examples/tenants/spaces.yaml"} {
		if b, err := os.ReadFile(p); err == nil {
			f.Add(b)
		}
	}
	f.Add([]byte(govBase))
	f.Add([]byte("tenant: a\nspaces:\n  x:\n    label: public #\n"))
	f.Fuzz(func(t *testing.T, src []byte) {
		tn, ds := Check(src)
		if len(ds) > 0 {
			return
		}
		back, ds := Check(Format(tn))
		if len(ds) > 0 {
			t.Fatalf("Format of an accepted manifest doesn't check: %v\n%s", ds, Format(tn))
		}
		if !reflect.DeepEqual(back, tn) {
			t.Fatalf("an accepted manifest doesn't read back the same:\n%s", Format(tn))
		}
	})
}
