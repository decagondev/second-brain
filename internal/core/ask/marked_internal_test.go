package ask

import "testing"

func TestMarkFrontAndRewrote(t *testing.T) {
	th := &SavedThread{Agent: "case-brief", Release: "rel_7", Text: "Line one\nLine two"}
	q := StewardQuestion{Thread: th, Reply: &StewardReply{Text: "Line one\nLine two"}}
	got := markFront(q, "Sam@acme-logistics.example")
	if len(got) != 2 || got[0] != "generated_by: case-brief@rel_7" || got[1] != "approved_by: sam@acme-logistics.example" {
		t.Fatalf("unchanged draft: %v", got)
	}
	q.Reply.Text = "Line one\nA line the steward added"
	if got := markFront(q, "sam@acme-logistics.example"); len(got) != 3 || got[2] != "edited: true" {
		t.Fatalf("edited draft: %v", got)
	}
	q.Reply.Text = "Entirely the steward's own words"
	if got := markFront(q, "sam@acme-logistics.example"); got != nil {
		t.Fatalf("a rewrite carries no mark: %v", got)
	}
	th.Versions = []string{"An earlier line"}
	if Rewrote(th, "An earlier line") {
		t.Fatal("a line from an earlier version is still the agent's")
	}
	if !Rewrote(nil, "anything") {
		t.Fatal("a thread a person saved is theirs")
	}
}

func TestCiteMark(t *testing.T) {
	for _, c := range []struct {
		gb, ab string
		ed     bool
		want   string
	}{
		{"", "", false, ""},
		{"case-brief", "", false, "Written by case-brief"},
		{"case-brief@rel_7", "sam@x", false, "Written by case-brief, version rel_7, approved by sam@x"},
		{"case-brief@rel_7", "sam@x", true, "Written by case-brief, version rel_7, edited and approved by sam@x"},
	} {
		if got := CiteMark(c.gb, c.ab, c.ed); got != c.want {
			t.Errorf("CiteMark(%q,%q,%v) = %q, want %q", c.gb, c.ab, c.ed, got, c.want)
		}
	}
}
