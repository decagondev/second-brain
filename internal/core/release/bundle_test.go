package release

import (
	"strings"
	"testing"
)

func h(s string) string { return SHA256([]byte(s)) }

func base() Bundle {
	return Bundle{
		Code:     map[string]string{"sb": h("sb-binary")},
		Prompts:  map[string]string{"answer": h("answer prompt"), "draft": h("draft prompt")},
		Tools:    []Tool{{Name: "ticket-update.comment", Connector: "tickets", Consequential: true, Workflow: "ticket-update", SchemaSHA256: h("schema")}},
		Policies: map[string]string{"policy/platform/base.cedar": h("base")},
		Model:    Model{Provider: "openaicompat", Model: "local-dev", MaxTokens: 1024},
		Gate:     h("gate"),
		Created:  "2027-03-02",
	}
}

// Any change to any field is a new id (SB-503).
func TestEveryFieldChangesTheID(t *testing.T) {
	id0, err := base().ID()
	if err != nil || !strings.HasPrefix(id0, "rel_2027-03-02_") || len(id0) != len("rel_2027-03-02_")+6 {
		t.Fatalf("%q %v", id0, err)
	}
	changes := map[string]func(*Bundle){
		"code":   func(b *Bundle) { b.Code["sb"] = h("other binary") },
		"prompt": func(b *Bundle) { b.Prompts["answer"] = h("answer prompt.") },
		"tool":   func(b *Bundle) { b.Tools[0].SchemaSHA256 = h("schema2") },
		"policy": func(b *Bundle) { b.Policies["policy/platform/base.cedar"] = h("base2") },
		"model":  func(b *Bundle) { b.Model.Model = "local-dev-2" },
		"tokens": func(b *Bundle) { b.Model.MaxTokens = 2048 },
		"gate":   func(b *Bundle) { b.Gate = h("gate2") },
		"date":   func(b *Bundle) { b.Created = "2027-03-03" },
	}
	seen := map[string]string{id0: "base"}
	for name, f := range changes {
		b := base()
		b.Code, b.Prompts, b.Policies = clone(b.Code), clone(b.Prompts), clone(b.Policies)
		b.Tools = append([]Tool(nil), b.Tools...)
		f(&b)
		id, err := b.ID()
		if err != nil {
			t.Fatal(name, err)
		}
		if prev, dup := seen[id]; dup {
			t.Errorf("%s kept id %s (same as %s)", name, id, prev)
		}
		seen[id] = name
	}
}

func clone(m map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

func TestToolOrderDoesNotMatter(t *testing.T) {
	a, b := base(), base()
	a.Tools = append(a.Tools, Tool{Name: "aaa", Connector: "docs", SchemaSHA256: h("s")})
	b.Tools = append([]Tool{{Name: "aaa", Connector: "docs", SchemaSHA256: h("s")}}, b.Tools...)
	ia, _ := a.ID()
	ib, _ := b.ID()
	if ia != ib {
		t.Fatalf("tool order changed the id: %s %s", ia, ib)
	}
}

func TestSealAndCheck(t *testing.T) {
	m, err := base().Seal()
	if err != nil || m.Check() != nil {
		t.Fatal(err)
	}
	m.Prompts = clone(m.Prompts)
	m.Prompts["answer"] = h("tampered")
	if m.Check() == nil {
		t.Fatal("a tampered bundle kept its id")
	}
	bad := base()
	bad.Gate = "nothex"
	if _, err := bad.ID(); err == nil {
		t.Fatal("a bad gate hash was accepted")
	}
}
