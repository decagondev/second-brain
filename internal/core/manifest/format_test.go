package manifest

import (
	"os"
	"reflect"
	"testing"
)

// Check(Format(t)) == t for every valid example (SB-602).
func TestFormatRoundTrip(t *testing.T) {
	for _, f := range []string{"../../../examples/tenants/pilot.yaml", "../../../examples/tenants/full.yaml"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		m, ds := Check(src)
		if len(ds) > 0 {
			t.Fatal(f, ds)
		}
		out := Format(m)
		back, ds := Check(out)
		if len(ds) > 0 {
			t.Fatalf("%s: formatted manifest doesn't check: %v\n%s", f, ds, out)
		}
		if !reflect.DeepEqual(m, back) {
			t.Fatalf("%s: round trip changed the manifest\n%+v\n%+v\n%s", f, m, back, out)
		}
	}
}
