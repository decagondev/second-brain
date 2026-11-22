package cell

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/jobsvc"
)

// operator provisions olga in sb-operators and signs her in, replacing the session.
func (r running) operator(t *testing.T) {
	t.Helper()
	code, body := r.do(t, "POST", "/scim/v2/Users", `{"userName":"olga","emails":[{"value":"olga@acme.example"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json")
	if code != http.StatusCreated {
		t.Fatalf("SCIM create: %d %s", code, body)
	}
	var u struct{ ID string }
	_ = json.Unmarshal([]byte(body), &u)
	if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"sb-operators","members":[{"value":"`+u.ID+`"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
		t.Fatalf("SCIM group: %d %s", code, body)
	}
	r.idp.User = u.ID
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatalf("operator sign-in: %d", code)
	}
}

// SB-1607: the console's health, from the cell's own records. A connector that synced is
// fresh; once its source fails it counts the failures and, past 15 minutes, is stale.
// Jobs that never ran are overdue and come first. Only operators read it.
func TestHealthFromTheCellsRecords(t *testing.T) {
	r, drive := docsCell(t)
	ctx := context.Background()
	r.signIn(t, "ann")
	if code, _ := r.do(t, "GET", "/v1/admin/health", ""); code != http.StatusForbidden {
		t.Fatalf("a non-operator read health: %d", code)
	}
	drive.Put("doc1", "Refunds", "text/plain", "The refund limit is 4200 dollars.", "ann@acme.example")
	now := time.Now()
	svc := &ingestsvc.Service{Cell: r.cell, Alert: &alerts{}, Now: func() time.Time { return now }}
	if fails := svc.SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	_ = r.cell.Runs.Finished(ctx, "sync", now)
	r.operator(t)
	type rep struct {
		Jobs []struct {
			Name    string
			Overdue bool
		}
		Connectors []struct {
			Kind     string
			Failures int
			Stale    bool
		}
	}
	read := func() rep {
		t.Helper()
		code, body := r.do(t, "GET", "/v1/admin/health", "")
		if code != 200 {
			t.Fatalf("health: %d %s", code, body)
		}
		var x rep
		if err := json.Unmarshal([]byte(body), &x); err != nil {
			t.Fatal(err)
		}
		return x
	}
	h := read()
	if len(h.Connectors) != 1 || h.Connectors[0].Kind != "docs" || h.Connectors[0].Stale || h.Connectors[0].Failures != 0 {
		t.Fatalf("a connector that just synced: %+v", h.Connectors)
	}
	if !h.Jobs[0].Overdue || h.Jobs[len(h.Jobs)-1].Name != "sync" || h.Jobs[len(h.Jobs)-1].Overdue {
		t.Fatalf("jobs, overdue first, sync fresh: %+v", h.Jobs)
	}
	drive.Server.Close() // the source goes away
	for i := 0; i < 3; i++ {
		_ = svc.SyncAll(ctx)
	}
	if h := read(); h.Connectors[0].Failures != 3 || h.Connectors[0].Stale {
		t.Fatalf("three failures, still within 15 minutes: %+v", h.Connectors)
	}
	// A last success older than the 15-minute freshness target is stale (the API's clock is
	// the real one, so the record is put back in time).
	old, _ := json.Marshal(map[string]any{"last_success": time.Now().Add(-20 * time.Minute), "failures": 3})
	if err := os.WriteFile(filepath.Join(r.cell.Config.State, "runs", "sync-docs.json"), old, 0o600); err != nil {
		t.Fatal(err)
	}
	if h := read(); !h.Connectors[0].Stale {
		t.Fatalf("a connector 20 minutes since its last success isn't stale: %+v", h.Connectors)
	}
	// An unreadable record is EUNAVAIL, never a healthy report.
	if err := os.WriteFile(filepath.Join(r.cell.Config.State, "runs", "sync-docs.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _ := r.do(t, "GET", "/v1/admin/health", ""); code != http.StatusServiceUnavailable {
		t.Fatalf("an unreadable sync record: %d, want EUNAVAIL", code)
	}
}

// SB-1609: the queue of answers marked unhelpful: counts, decisions by verdict and reason,
// and the asker's comment; never the question, the answer or a cite. Reviewing writes an
// audit row first, and a later feedback puts the item back.
func TestUnhelpfulAnswerQueue(t *testing.T) {
	r, drive := docsCell(t)
	ctx := context.Background()
	r.signIn(t, "ann")
	drive.Put("doc1", "Refunds", "text/plain", "The zebra refund limit is 4200 dollars.", "ann@acme.example")
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	code, body := r.do(t, "POST", "/v1/ask", `{"question":"What is the zebra refund limit?"}`)
	if code != 200 {
		t.Fatalf("ask: %d %s", code, body)
	}
	var a struct{ Trace string }
	_ = json.Unmarshal([]byte(body), &a)
	if code, body := r.do(t, "POST", "/v1/feedback", `{"trace":"`+a.Trace+`","rating":"down","comment":"it missed the EU limit"}`); code != 200 {
		t.Fatalf("feedback: %d %s", code, body)
	}
	if code, _ := r.do(t, "GET", "/v1/admin/feedback?rating=down", ""); code != http.StatusForbidden {
		t.Fatalf("a non-operator read the queue: %d", code)
	}
	r.operator(t)
	code, body = r.do(t, "GET", "/v1/admin/feedback?rating=down", "")
	if code != 200 {
		t.Fatalf("queue: %d %s", code, body)
	}
	for _, leak := range []string{"zebra", "4200", "docs:doc1"} {
		if strings.Contains(body, leak) {
			t.Fatalf("the queue shows %q: %s", leak, body)
		}
	}
	var q struct {
		Items []struct {
			Trace, Comment string
			Claims         int
			Decisions      []struct {
				Verdict, Reason string
				Count           int
			}
			Reviewed *struct{ By string }
		}
	}
	_ = json.Unmarshal([]byte(body), &q)
	if len(q.Items) != 1 || q.Items[0].Trace != a.Trace || q.Items[0].Comment != "it missed the EU limit" || len(q.Items[0].Decisions) == 0 {
		t.Fatalf("queue %s", body)
	}
	if code, _ := r.do(t, "GET", "/v1/admin/feedback?rating=sideways", ""); code != http.StatusBadRequest {
		t.Fatalf("a bad rating: %d", code)
	}
	if code, _ := r.do(t, "POST", "/v1/admin/feedback/tr_nope/reviewed", ""); code != http.StatusBadRequest {
		t.Fatalf("no feedback on that trace: %d", code)
	}
	// Review rows (the admin guard's own policy decisions are rows too).
	reviews := func() int {
		rs, _ := r.cell.Audit.Store.Range(ctx, 1, 1<<20)
		n := 0
		for _, x := range rs {
			if x.Kind == "config" && x.Subject == "feedback:"+a.Trace && x.State == "Reviewed" {
				n++
			}
		}
		return n
	}
	if code, body := r.do(t, "POST", "/v1/admin/feedback/"+a.Trace+"/reviewed", ""); code != 200 {
		t.Fatalf("review: %d %s", code, body)
	}
	if reviews() != 1 {
		t.Fatal("no audit row for the review")
	}
	if code, _ := r.do(t, "POST", "/v1/admin/feedback/"+a.Trace+"/reviewed", ""); code != 200 || reviews() != 1 {
		t.Fatal("a second review wrote another row or failed")
	}
	if _, body := r.do(t, "GET", "/v1/admin/feedback?rating=down", ""); strings.Contains(body, a.Trace) {
		t.Fatalf("a reviewed item is still in the queue: %s", body)
	}
	if _, body := r.do(t, "GET", "/v1/admin/feedback?rating=down&reviewed=true", ""); !strings.Contains(body, `"by":"usr_`) {
		t.Fatalf("reviewed=true leaves out the reviewed item or its reviewer: %s", body)
	}
}

// SB-1608: the weekly report goes to the report webhook as well as the file, and a
// delivery that fails fails the run.
func TestWeeklyReportIsDelivered(t *testing.T) {
	r, svc, c, _ := jobsCell(t, true)
	ctx := context.Background()
	var got []map[string]any
	fail := false
	hook := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if fail {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		var m map[string]any
		_ = json.NewDecoder(req.Body).Decode(&m)
		got = append(got, m)
	}))
	defer hook.Close()
	r.cell.Config.Alerts.ReportWebhook = hook.URL
	svc.HTTP = hook.Client()
	c.t = time.Date(2027, 3, 15, 7, 0, 0, 0, time.UTC) // a Monday
	if err := svc.Run(ctx, "report"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0]["kind"] != "weekly_report" || got[0]["tenant"] != tenant || got[0]["week"] != "2027-W10" ||
		!strings.Contains(got[0]["markdown"].(string), "Asks") {
		t.Fatalf("delivered %+v", got)
	}
	if _, err := os.Stat(filepath.Join(r.cell.Config.State, "reports", "2027-W10.md")); err != nil {
		t.Fatalf("the file is still written: %v", err)
	}
	fail = true
	if err := svc.Run(ctx, "report"); err == nil {
		t.Fatal("a failed delivery passed the run")
	}
	_ = jobsvc.ErrAccess
}
