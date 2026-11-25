package manifest

import (
	"strings"
	"testing"
)

const capsBase = "tenant: acme-logistics\nchannel: stable\nidp: oidc://login.acme.example\nregion: eu\nconnectors: { docs: folder }\n"

// SB-4703: agents as a map carries monthly caps; the list form still works.
func TestAgentCaps(t *testing.T) {
	m, probs := Check([]byte(capsBase + "agents: { ask: {}, case-brief: { monthly_usd: 40 }, incident-review: { monthly_usd: 5.5 } }\n"))
	if len(probs) > 0 {
		t.Fatal(probs)
	}
	if strings.Join(m.Agents, ",") != "ask,case-brief" || m.AgentCaps["case-brief"] != 40 || m.AgentCaps["incident-review"] != 5.5 || len(m.AgentCaps) != 2 {
		t.Fatalf("agents %v, caps %v", m.Agents, m.AgentCaps)
	}
	again, probs := Check(Format(m))
	if len(probs) > 0 || strings.Join(again.Agents, ",") != "ask,case-brief" || again.AgentCaps["incident-review"] != 5.5 {
		t.Fatalf("round trip: %v %v %v\n%s", probs, again.Agents, again.AgentCaps, Format(m))
	}
	for doc, want := range map[string]string{
		"agents: { case-brief: { monthly_usd: 40 } }\n": `"ask" is required`,
		"agents: { ask: {}, nope: {} }\n":               `"nope" is not an agent`,
		"agents: { ask: {}, voc: {} }\n":                "name it here only with monthly_usd",
		"agents: { ask: { monthly_usd: 0 } }\n":         "must be a number from 1 to 100000",
		"agents: { ask: { monthly_usd: \"40\" } }\n":    "must be a number",
		"agents: { ask: { limit: 4 } }\n":               "unknown field; use monthly_usd",
		"agents: { ask: 40 }\n":                         "must be a map",
	} {
		_, probs := Check([]byte(capsBase + doc))
		if !strings.Contains(fmtProblems(probs), want) {
			t.Errorf("%s: problems %v, want %q", doc, probs, want)
		}
	}
	if m, probs := Check([]byte(capsBase + "agents: [ask, case-brief]\n")); len(probs) > 0 || m.AgentCaps != nil || len(m.Agents) != 2 {
		t.Fatalf("list form: %v %v", probs, m)
	}
}

func fmtProblems(ps []Diag) string {
	var b strings.Builder
	for _, p := range ps {
		b.WriteString(p.Msg + "\n")
	}
	return b.String()
}

func TestAPatchKeepsTheCaps(t *testing.T) {
	m, _ := Check([]byte(capsBase + "agents: { ask: {}, case-brief: { monthly_usd: 40 }, voc: { monthly_usd: 9 } }\n"))
	next, ds, err := Patch(m, []Op{{Op: "replace", Path: "/retention_days", Value: []byte("200")}})
	if err != nil || len(ds) > 0 || next.AgentCaps["case-brief"] != 40 || next.AgentCaps["voc"] != 9 {
		t.Fatalf("patched: %v %v %v", err, ds, next.AgentCaps)
	}
	next, ds, err = Patch(m, []Op{{Op: "replace", Path: "/agents", Value: []byte(`["ask"]`)}})
	if err != nil || len(ds) > 0 || next.AgentCaps["case-brief"] != 0 || next.AgentCaps["voc"] != 9 {
		t.Fatalf("case-brief turned off: %v %v %v", err, ds, next.AgentCaps)
	}
}
