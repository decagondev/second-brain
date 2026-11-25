package ask

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/audit"
)

// SB-5005: a check counts the four weeks either side of a change in its space, says "not
// enough data" under 20 answers a side, and calls a clear move better or worse.
func TestChecksCountBeforeAndAfter(t *testing.T) {
	at := time.Date(2027, 3, 1, 9, 0, 0, 0, time.UTC)
	nothing := NoticeNoSources
	var ts []Trace
	var fb []Feedback
	add := func(n int, from time.Time, space string, gap bool, rating string) {
		for i := 0; i < n; i++ {
			tr := Trace{ID: "t" + from.Format("0102") + space + string(rune('a'+i%26)) + string(rune('a'+i/26)), Space: space, At: from.Add(time.Duration(i) * time.Hour)}
			if gap {
				tr.Response.Notice = &nothing
			}
			ts = append(ts, tr)
			if rating != "" {
				fb = append(fb, Feedback{Trace: tr.ID, Rating: rating})
			}
		}
	}
	before, after := at.AddDate(0, 0, -10), at.AddDate(0, 0, 3)
	add(15, before, "spc_finance", false, "down")
	add(10, before, "spc_finance", true, "")
	add(25, after, "spc_finance", false, "up")
	add(30, after, "spc_ops", true, "") // another space: not counted
	ts = append(ts, Trace{Space: "spc_finance", At: after, Feature: FeatureEval})
	c := Change{Kind: ChangeSkill, Ref: "prop_1", Space: "finance", At: at}
	if CheckDue(c, at.Add(CheckAfter-time.Hour)) || !CheckDue(c, at.Add(CheckAfter)) {
		t.Fatal("due")
	}
	r := RunCheck(c, nil, ts, fb, at.Add(CheckAfter))
	if r.Before.Answers != 25 || r.After.Answers != 25 || r.Before.Gaps != 10 || *r.After.HelpfulRate != 1 || *r.Before.HelpfulRate != 0 {
		t.Fatalf("numbers: %+v %+v", r.Before, r.After)
	}
	if r.Verdict != VerdictBetter || !strings.Contains(r.Text, "better") || !strings.Contains(r.Text, "Nothing was changed or undone") || r.ID != c.Key() {
		t.Fatalf("check: %+v", r)
	}
	worse := RunCheck(Change{Kind: ChangeSpace, Ref: "prop_2", Space: "ops", At: at.AddDate(0, 0, 30)}, nil, ts, fb, at.AddDate(0, 0, 60))
	if worse.Verdict != VerdictTooFew || !strings.Contains(worse.Text, "Fewer than 20") {
		t.Fatalf("too few: %+v", worse)
	}
	if v := verdict(CheckNumbers{Answers: 20, HelpfulRate: f(0.5)}, CheckNumbers{Answers: 20, HelpfulRate: f(0.3)}); v != VerdictWorse {
		t.Fatalf("worse: %s", v)
	}
	if v := verdict(CheckNumbers{Answers: 20, HelpfulRate: f(0.5), GapRate: f(0.1)}, CheckNumbers{Answers: 20, HelpfulRate: f(0.7), GapRate: f(0.3)}); v != VerdictNoClear {
		t.Fatalf("mixed: %s", v)
	}
	if k := ProposalChange([]string{"~ spaces.finance.skills.close"}); k != ChangeSkill {
		t.Fatal(k)
	}
	if k := ProposalChange([]string{"~ spaces.finance.routing", "~ spaces.finance.agents"}); k != ChangeRouting {
		t.Fatal(k)
	}
	rel := []Trace{{At: at.Add(2 * time.Hour), Response: Response{Release: "rel_b", Agent: "case-brief"}}, {At: at, Response: Response{Release: "rel_a", Agent: "case-brief"}}}
	if cs := ReleaseChanges(rel, at, at.AddDate(0, 0, 1)); len(cs) != 1 || cs[0].Agent != "case-brief" || cs[0].Ref != "rel_b" {
		t.Fatalf("releases: %+v", cs)
	}
}

func f(x float64) *float64 { return &x }

// SB-5006: the admin's week groups its lines by section, each with its link.
func TestAdminWeekText(t *testing.T) {
	if got := AdminWeekText("2027-03-01", nil); !strings.Contains(got, "nothing changed") {
		t.Fatal(got)
	}
	got := AdminWeekText("2027-03-01", []AdminLine{{Section: "Changed", Text: "prop_1 skill approved", Link: "/v1/admin/audit?since=4"}, {Section: "Pauses", Text: "agent:x@ops paused", Link: "/v1/admin/audit?since=9"}})
	if !strings.Contains(got, "\nChanged:\n- prop_1 skill approved (/v1/admin/audit?since=4)\nPauses:") {
		t.Fatal(got)
	}
}

// A record with real amounts and a target's rates signs and verifies: amounts and rates hash
// as decimal strings (canonjson.EncodeDocument).
func TestRecordWithAmountsSigns(t *testing.T) {
	sg, err := audit.NewSigner("acme-logistics", bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	least, v := 0.6, 0.25
	r := AgentRecord{Agent: "case-brief", Tenant: "acme-logistics", Value: []RecordQuarter{{Quarter: "2027-Q1", CostUSD: 12.75}},
		Targets: []TargetView{{TargetDef: TargetDef{Metric: MetricPublishedAsIs, AtLeast: &least}, Weeks: []TargetWeek{{Week: "2027-03-01", Value: &v}}}}}
	sr, err := SignRecord(r, sg, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := json.Marshal(sr)
	if _, err := VerifyRecord(doc, sg.Key()); err != nil {
		t.Fatal(err)
	}
}
