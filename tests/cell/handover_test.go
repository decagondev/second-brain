package cell

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"example.com/second-brain/internal/cell/ingestsvc"
)

// SB-1707 through a running cell: a person's own handover brief cites their documents;
// a colleague with no relation to them is refused.
func TestHandoverThroughARunningCell(t *testing.T) {
	r, drive := docsCell(t)
	ctx := context.Background()
	r.signIn(t, "ann")
	ann := r.idp.User
	drive.Put("doc1", "Carrier contracts", "text/plain", "The Nordic carrier contract renews in March.", "ann@acme.example", "bob@acme.example")
	drive.Own("doc1", "ann@acme.example")
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	code, body := r.do(t, "POST", "/v1/handover", `{"person":"`+ann+`"}`)
	if code != 200 || !strings.Contains(body, "docs:doc1@") || !strings.Contains(body, `"hidden":0`) ||
		!strings.Contains(body, `"locations":{"docs:doc1@`) {
		t.Fatalf("ann's own brief: %d %s", code, body)
	}
	r.signIn(t, "bob")
	if code, body := r.do(t, "POST", "/v1/handover", `{"person":"`+ann+`"}`); code != http.StatusForbidden || !strings.Contains(body, "their manager") {
		t.Fatalf("a colleague got ann's brief: %d %s", code, body)
	}
}
