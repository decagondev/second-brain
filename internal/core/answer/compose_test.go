package answer

import (
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

func ctx() Context {
	return NewContext([]ports.Chunk{
		{ID: "a", Cite: "docs:refunds@3", Text: "Refunds up to 500 need no approval.", Start: 0, End: 36},
		{ID: "b", Cite: "tickets:LIN-7@2", Text: "Raised to 750 in Q3.", Flags: []string{"injection_suspect"}},
	})
}

func TestComposeMapsLabels(t *testing.T) {
	raw := `{"claims":[{"text":"Refunds up to 500.","cites":["C1","C1"]},{"text":"Now 750.","cites":["C2","C9","docs:secret@1"]},{"text":"Ask finance.","cites":[]}]}`
	res := Compose(raw, ctx(), map[string]string{"docs:refunds@3": "Refund policy"})
	if len(res.Claims) != 3 {
		t.Fatalf("%+v", res)
	}
	c := res.Claims[0]
	if c.Uncited || len(c.Cites) != 1 || c.Cites[0].Cite != "docs:refunds@3" || c.Cites[0].Title != "Refund policy" || c.Cites[0].Span != [2]int{0, 36} {
		t.Errorf("claim 1: %+v", c)
	}
	c = res.Claims[1]
	if c.Uncited || len(c.Cites) != 1 || c.Cites[0].Flags[0] != "injection_suspect" {
		t.Errorf("claim 2: %+v", c)
	}
	if !res.Claims[2].Uncited || len(res.Claims[2].Cites) != 0 {
		t.Errorf("claim 3: %+v", res.Claims[2])
	}
	if strings.Join(res.Spoofs, ",") != "C9,docs:secret@1" {
		t.Errorf("spoofs %v", res.Spoofs)
	}
}

func TestComposeNotJSON(t *testing.T) {
	for _, raw := range []string{"Refunds are 500.", `{"claims":[]}`, `[1]`} {
		res := Compose(raw, ctx(), nil)
		if len(res.Claims) != 1 || !res.Claims[0].Uncited || len(res.Claims[0].Cites) != 0 {
			t.Errorf("%q: %+v", raw, res)
		}
	}
}

func TestDataBlocks(t *testing.T) {
	d := ctx().Data()
	for _, want := range []string{"C1 docs:refunds@3", "C2 tickets:LIN-7@2", "Refunds up to 500"} {
		if !strings.Contains(d, want) {
			t.Errorf("data lacks %q:\n%s", want, d)
		}
	}
}
