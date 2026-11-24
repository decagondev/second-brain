package laws

import (
	"strings"
	"testing"
)

// SB-3107 (LAW-16): a suggested glossary term is a steward proposal like any other, listed
// for the operators; nothing changes until it is approved and applied, and the manifest an
// approval returns has the term and its spellings.
func TestSuggestedTermIsAProposal(t *testing.T) {
	ev, err := LoadPolicies()
	if err != nil {
		t.Fatal(err)
	}
	c, err := newStewardCell(ev, nil)
	if err != nil {
		t.Fatal(err)
	}
	term := map[string]any{"term": "COD", "meaning": "cash on delivery", "spellings": []string{"C.O.D."}}
	if code, _ := c.call("usr_eve", []string{"finance"}, "POST", "/v1/spaces/finance/proposals/term", term); code != 403 {
		t.Fatalf("a member who isn't a steward suggested a term: %d", code)
	}
	code, out := c.call("usr_lead", []string{"finance-leads"}, "POST", "/v1/spaces/finance/proposals/term", term)
	if code != 200 {
		t.Fatalf("suggest: %d %v", code, out)
	}
	if len(c.applied.t.Spaces["finance"].Glossary) != 0 {
		t.Fatal("the glossary changed before approval")
	}
	code, list := c.call("usr_op1", []string{"sb-operators"}, "GET", "/v1/admin/proposals", nil)
	items, _ := list["items"].([]any)
	if code != 200 || len(items) != 1 || items[0].(map[string]any)["id"] != out["id"] {
		t.Fatalf("not in the operators' proposals: %d %v", code, list)
	}
	code, ok := c.call("usr_op1", []string{"sb-operators"}, "POST", "/v1/admin/proposals/"+out["id"].(string)+"/approve", map[string]any{"hash": out["hash"]})
	m, _ := ok["manifest"].(string)
	if code != 200 || !strings.Contains(m, `"COD": "cash on delivery"`) || !strings.Contains(m, `"C.O.D.": "cash on delivery"`) {
		t.Fatalf("approve: %d %s", code, m)
	}
	many := map[string]any{"term": "PO", "meaning": "purchase order", "spellings": strings.Split("a,b,c,d,e,f,g,h,i,j,k", ",")}
	if code, _ := c.call("usr_lead", []string{"finance-leads"}, "POST", "/v1/spaces/finance/proposals/term", many); code != 400 {
		t.Fatalf("eleven spellings: %d", code)
	}
	twice := map[string]any{"term": "PO", "meaning": "purchase order", "spellings": []string{"po"}}
	if code, _ := c.call("usr_lead", []string{"finance-leads"}, "POST", "/v1/spaces/finance/proposals/term", twice); code != 400 {
		t.Fatalf("a spelling the glossary has: %d", code)
	}
}
