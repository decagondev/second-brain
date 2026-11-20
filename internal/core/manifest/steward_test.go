package manifest

import (
	"strings"
	"testing"
)

func TestStewardChange(t *testing.T) {
	cur, ds := Check([]byte(spacesValid))
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	ok := `{"template":"finance","members":["group:finance","group:execs"],"stewards":["group:finance-leads"],"scope":{"docs":["/Finance/**"]}}`
	next, ds, err := StewardChange(cur, "finance", []byte(ok))
	if err != nil || len(ds) > 0 || len(next.Spaces["finance"].Members) != 2 || next.Spaces["finance"].RetentionDays != 730 { // left out: the default, the tenant's
		t.Fatalf("%+v %v %v", next.Spaces["finance"], ds, err)
	}
	for _, tc := range []struct{ name, body, want string }{
		{"template", strings.Replace(ok, `"finance"`, `"custom"`, 1), "can't change the space's template"},
		{"label", strings.Replace(ok, `"template":"finance"`, `"template":"finance","label":"internal"`, 1), "can't lower the space's label"},
		{"no stewards", strings.Replace(ok, `["group:finance-leads"]`, `[]`, 1), "keeps at least one steward"},
		{"unknown field", strings.Replace(ok, `"template"`, `"barriers":[],"template"`, 1), "isn't a space definition"},
	} {
		if _, _, err := StewardChange(cur, "finance", []byte(tc.body)); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
	if _, _, err := StewardChange(cur, "nowhere", []byte(ok)); err == nil {
		t.Fatal("a proposal for a space that doesn't exist")
	}
	bad := `{"template":"finance","members":["group:finance"],"stewards":["group:finance-leads"],"scope":{"docs":["/Finance/**"],"crm":["deals"]}}`
	if _, ds, err := StewardChange(cur, "finance", []byte(bad)); err != nil || len(ds) == 0 {
		t.Fatalf("a scope kind with no connector must fail the check: %v %v", ds, err)
	}
}
