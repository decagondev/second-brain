package cell

import (
	"reflect"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/manifest"
)

func TestTemplate(t *testing.T) {
	got := Template(manifest.Tenant{Tenant: "acme-logistics", Region: "eu"})
	var lines []string
	for _, r := range got {
		lines = append(lines, r.Kind+" "+r.Name+" "+r.Tenant+" "+r.Region)
	}
	want := []string{
		"database sb_acme_logistics acme-logistics eu",
		"index-schema idx acme-logistics eu",
		"vault-namespace tenants/acme-logistics acme-logistics eu",
		"root-key cell-acme-logistics-root acme-logistics eu",
		"queue acme-logistics-jobs acme-logistics eu",
	}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("template:\n%s", strings.Join(lines, "\n"))
	}
}

// Parameterised only by manifest tenant and region: every other field leaves the cell unchanged.
func TestOnlyTenantAndRegionMatter(t *testing.T) {
	base := manifest.Tenant{Tenant: "acme-logistics", Region: "eu"}
	full := base
	full.Channel, full.IdP, full.Agents = "testing", "oidc://x.example", []string{"ask"}
	full.Connectors = map[string]string{"docs": "sharepoint"}
	full.RetentionDays, full.BudgetUSDMonth, full.PilotGroup = 30, 5, "group:a"
	if !reflect.DeepEqual(Template(base), Template(full)) {
		t.Fatal("a manifest field other than tenant or region changed the cell template")
	}
}
