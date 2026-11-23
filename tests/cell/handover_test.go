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
	// SB-2106: ann finds herself by name; bob, a colleague, doesn't find her at all.
	if code, body := r.do(t, "GET", "/v1/handover/people?q=ANN", ""); code != 200 || !strings.Contains(body, `"id":"`+ann+`"`) {
		t.Fatalf("ann's own search: %d %s", code, body)
	}
	if code, _ := r.do(t, "GET", "/v1/handover/people?q=a", ""); code != 400 {
		t.Fatalf("a one-letter search: %d", code)
	}
	r.signIn(t, "bob")
	if code, body := r.do(t, "POST", "/v1/handover", `{"person":"`+ann+`"}`); code != http.StatusForbidden || !strings.Contains(body, "their manager") {
		t.Fatalf("a colleague got ann's brief: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/handover/people?q=ann", ""); code != 200 || strings.Contains(body, ann) || !strings.Contains(body, `"items":[]`) {
		t.Fatalf("a colleague found ann: %d %s", code, body)
	}
}
