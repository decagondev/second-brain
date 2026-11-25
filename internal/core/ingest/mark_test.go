package ingest

import "testing"

func TestMarkOf(t *testing.T) {
	doc := "---\ngenerated_by: case-brief@rel_7\napproved_by: Sam@acme-logistics.example\nedited: true\n---\nbody"
	m := MarkOf(doc, "sam@acme-logistics.example")
	if m.GeneratedBy != "case-brief@rel_7" || m.ApprovedBy != "sam@acme-logistics.example" || !m.Edited {
		t.Fatalf("mark: %+v", m)
	}
	if m := MarkOf(doc, "lee@acme-logistics.example"); m.ApprovedBy != "" || m.GeneratedBy == "" {
		t.Fatalf("an approver who isn't the owner should be dropped, the mark kept: %+v", m)
	}
	if m := MarkOf("---\ngenerated_by: <script>\n---\n", ""); m != (Mark{}) {
		t.Fatalf("a malformed generated_by should give no mark: %+v", m)
	}
	if m := MarkOf("no front matter", ""); m != (Mark{}) {
		t.Fatalf("a person's document has no mark: %+v", m)
	}
}

// SB-5204: asked_by and edited_by are read as addresses; anything else is dropped.
func TestMarkOfWhoMadeIt(t *testing.T) {
	m := MarkOf("---\ngenerated_by: case-brief@rel_7\nasked_by: Kim@Acme.example\nedited_by: kim@acme.example, not an address, lee@acme.example\n---\n# x\n", "")
	if m.AskedBy != "kim@acme.example" || m.EditedBy != "kim@acme.example, lee@acme.example" {
		t.Fatalf("%+v", m)
	}
}
