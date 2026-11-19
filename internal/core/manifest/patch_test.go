package manifest

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func pilot(t *testing.T) Tenant {
	src, _ := os.ReadFile("../../../examples/tenants/pilot.yaml")
	m, ds := Check(src)
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	return m
}

func ops(s string) []Op {
	var o []Op
	_ = json.Unmarshal([]byte(s), &o)
	return o
}

func TestPatch(t *testing.T) {
	m := pilot(t)
	next, ds, err := Patch(m, ops(`[{"op":"add","path":"/agents/-","value":"draft-reply"}]`))
	if err != nil || len(ds) > 0 || len(next.Agents) != 2 {
		t.Fatalf("%+v %v %v", next.Agents, ds, err)
	}
	d := strings.Join(Diff(m, next), "\n")
	if d != "- agents: [ask]\n+ agents: [ask, draft-reply]" {
		t.Fatalf("diff:\n%s", d)
	}
	// make permanent: remove an agent
	back, _, err := Patch(next, ops(`[{"op":"remove","path":"/agents/draft-reply"}]`))
	if err != nil || len(back.Agents) != 1 {
		t.Fatalf("%v %v", back.Agents, err)
	}
	// invalid results come back as the checker's diagnostics
	if _, ds, err := Patch(m, ops(`[{"op":"add","path":"/agents/-","value":"autopilot"}]`)); err != nil || len(ds) == 0 {
		t.Fatalf("unknown agent: %v %v", ds, err)
	}
	if _, ds, _ := Patch(m, ops(`[{"op":"replace","path":"/budget_usd_month","value":-5}]`)); len(ds) == 0 {
		t.Fatal("a negative budget was accepted")
	}
	for _, bad := range []string{`[{"op":"replace","path":"/tenant","value":"beta-freight"}]`, `[{"op":"replace","path":"/region","value":"us"}]`, `[{"op":"add","path":"/custody","value":"tenant"}]`} {
		if _, _, err := Patch(m, ops(bad)); err == nil {
			t.Errorf("%s accepted", bad)
		}
	}
	if _, _, err := Patch(m, ops(`[{"op":"remove","path":"/agents/nope"}]`)); err == nil {
		t.Error("removing a missing agent was accepted")
	}
}
