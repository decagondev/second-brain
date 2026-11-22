package cell

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"example.com/second-brain/internal/cell/ingestsvc"
)

// S16.1 through a running cell (SB-1701 to SB-1703): a search finds the document without
// the model, names its active owner, and shows nothing to someone the source refuses; a
// shared question opens for a colleague, who asks it under their own access.
func TestFindThroughARunningCell(t *testing.T) {
	r, drive := docsCell(t)
	ctx := context.Background()
	r.signIn(t, "ann")
	drive.Put("doc1", "Refunds", "text/plain", "The zebra refund limit is 4200 dollars.", "ann@acme.example")
	drive.Own("doc1", "ann@acme.example")
	svc := &ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}
	if fails := svc.SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	type found struct {
		Trace   string
		Results []struct {
			Cite, Title, Snippet string
			AgeDays              *int `json:"age_days"`
			Owner                *struct{ Email string }
		}
		Refused int
	}
	search := func() found {
		t.Helper()
		code, body := r.do(t, "GET", "/v1/search?q=zebra+refund", "")
		if code != http.StatusOK {
			t.Fatalf("search: %d %s", code, body)
		}
		var f found
		if err := json.Unmarshal([]byte(body), &f); err != nil {
			t.Fatal(err)
		}
		return f
	}
	f := search()
	if len(f.Results) != 1 || f.Results[0].Title != "Refunds" || !strings.Contains(f.Results[0].Snippet, "4200") ||
		f.Results[0].AgeDays == nil || f.Results[0].Owner == nil || f.Results[0].Owner.Email != "ann@acme.example" {
		t.Fatalf("ann's search: %+v", f)
	}
	if spent, _, _ := r.cell.Usage.Tokens(ctx, tenant, r.cell.Usage.Now().AddDate(0, 0, -1), r.cell.Usage.Now().AddDate(0, 0, 1)); spent != 0 {
		t.Fatalf("a search spent %d tokens", spent)
	}

	code, body := r.do(t, "POST", "/v1/shares", `{"question":"What is the zebra refund limit?"}`)
	if code != http.StatusOK {
		t.Fatalf("share: %d %s", code, body)
	}
	var sh struct{ ID, URL string }
	_ = json.Unmarshal([]byte(body), &sh)

	r.signIn(t, "bob")                      // the source doesn't share doc1 with bob
	if f := search(); len(f.Results) != 0 { // the prefilter or the live check refuses it
		t.Fatalf("LAW-01: bob's search showed %+v", f)
	}
	code, body = r.do(t, "GET", "/v1/shares/"+sh.ID, "")
	if code != http.StatusOK || !strings.Contains(body, "zebra refund limit") || strings.Contains(body, "4200") {
		t.Fatalf("bob opening ann's share: %d %s", code, body)
	}
	if r.asks(t, "What is the zebra refund limit?", "4200") {
		t.Fatal("LAW-01: the shared question answered bob from ann's access")
	}
}
