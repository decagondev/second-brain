package cell

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/inventory"
	"example.com/second-brain/internal/core/notify"
)

// SB-5005 to SB-5007 through a running cell. A steward proposal approved four weeks ago is
// checked once, drafted for the space's stewards and kept, and says "not enough data" with
// few answers. The operators get one admin's week, each line with its source record, except
// the operator who turned it off. The inventory lists the registry's agents and the space's
// starter questions, counts a personal routine without naming it, and its signature
// verifies.
func TestLearnWhatWorksThroughARunningCell(t *testing.T) {
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, opsManifest,
			map[string]string{"audit/anchor-seed": base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))})
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + t.TempDir()}}
		return cfg
	})
	ctx := context.Background()
	told := &agentsTold{}
	r.cell.Send = told
	r.signIn(t, "lee")
	lee := r.idp.User
	r.signIn(t, "sam")
	sam := r.idp.User
	for name, members := range map[string][]string{"ops": {sam, lee}, "sb-operators": {sam, lee}} {
		var ms []string
		for _, x := range members {
			ms = append(ms, `{"value":"`+x+`"}`)
		}
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group: %d %s", code, body)
		}
	}
	as := func(id string) {
		t.Helper()
		r.idp.User = id
		if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
			t.Fatalf("sign-in: %d", code)
		}
	}
	// lee turns the admin's week off for lee alone.
	as(lee)
	if code, out := r.do(t, "PUT", "/v1/me/notifications", `{"off":["adminweek"]}`); code != 200 {
		t.Fatalf("notification settings: %d %s", code, out)
	}
	as(sam)
	// A personal routine, never named in the inventory.
	if code, out := r.do(t, "PUT", "/v1/spaces/ops/suggested", `{"questions":["When is the zebra cutoff?"]}`); code != 200 {
		t.Fatalf("starters: %d %s", code, out)
	}
	code, out := r.do(t, "POST", "/v1/saved?space=ops", `{"question":"my private zebra routine"}`)
	var sq struct{ ID string }
	if err := json.Unmarshal([]byte(out), &sq); err != nil || code != 200 || sq.ID == "" {
		t.Fatalf("save: %d %s", code, out)
	}
	if code, out := r.do(t, "PUT", "/v1/saved/"+sq.ID+"/routine", `{"every":"daily"}`); code != 200 {
		t.Fatalf("routine: %d %s", code, out)
	}

	// Rows written as if a proposal was approved 29 days ago and an agent paused last week.
	now := time.Now()
	week := notify.Monday(now, time.UTC)
	write := func(at time.Time, e audit.Event) {
		t.Helper()
		r.cell.Audit.Now = func() time.Time { return at }
		defer func() { r.cell.Audit.Now = time.Now }()
		if _, err := r.cell.Audit.Write(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	write(now.Add(-29*24*time.Hour), audit.Event{Actor: sam, Kind: "config", Subject: "prop_old", State: "approved", Reason: "approve steward proposal prop_old", Space: "spc_ops"})
	write(week.Add(-3*24*time.Hour), audit.Event{Actor: sam, Kind: "kill", Subject: "agent:ask@ops", State: "killed", Reason: "too many wrong answers"})
	if _, err := r.cell.NotifyJob(told).Run(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := r.cell.NotifyJob(told).Run(ctx); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(told.got, "\n")
	for _, who := range []string{sam, lee} {
		if n := strings.Count(got, "check "+who+": Did it help? space prop_old in ops"); n != 1 {
			t.Fatalf("%s was told of the check %d times: %v", who, n, told.got)
		}
	}
	if !strings.Contains(got, "not enough data") {
		t.Fatalf("the check: %v", told.got)
	}
	if n := strings.Count(got, "adminweek "+sam+": Your admin's week"); n != 1 {
		t.Fatalf("sam got %d admin's weeks: %v", n, told.got)
	}
	if strings.Contains(got, "adminweek "+lee) {
		t.Fatalf("lee turned the admin's week off: %v", told.got)
	}
	if !strings.Contains(got, "agent:ask@ops paused: too many wrong answers (/v1/admin/audit?since=") {
		t.Fatalf("the admin's week has no pause line with its record: %v", told.got)
	}

	// The checks, kept, for the operators and the space's stewards.
	var cs struct{ Checks []ask.Check }
	code, out = r.do(t, "GET", "/v1/admin/checks", "")
	if err := json.Unmarshal([]byte(out), &cs); err != nil || code != 200 || len(cs.Checks) != 1 || cs.Checks[0].Verdict != ask.VerdictTooFew {
		t.Fatalf("checks: %d %s", code, out)
	}
	if code, one := r.do(t, "GET", "/v1/admin/checks/"+cs.Checks[0].ID, ""); code != 200 || !strings.Contains(one, "prop_old") {
		t.Fatalf("one check: %d %s", code, one)
	}
	if code, out := r.do(t, "GET", "/v1/spaces/ops/checks", ""); code != 200 || !strings.Contains(out, cs.Checks[0].ID) {
		t.Fatalf("the space's checks: %d %s", code, out)
	}

	// SB-5007: the inventory, signed, and its CSV.
	code, out = r.do(t, "GET", "/v1/admin/inventory", "")
	if code != 200 {
		t.Fatalf("inventory: %d %s", code, out)
	}
	inv, err := inventory.Verify([]byte(out), "")
	if err != nil {
		t.Fatalf("the inventory doesn't verify: %v\n%s", err, out)
	}
	kinds := map[string]int{}
	for _, it := range inv.Inventory.Items {
		kinds[it.Kind]++
	}
	if kinds[inventory.ItemAgent] == 0 || kinds[inventory.ItemStarter] != 1 || len(inv.Inventory.PersonalRoutines) != 1 || inv.Inventory.PersonalRoutines[0].Count != 1 {
		t.Fatalf("inventory: %s", out)
	}
	code, csv := r.do(t, "GET", "/v1/admin/inventory?format=csv", "")
	if code != 200 || !strings.HasPrefix(csv, "kind,name,owner,spaces") || !strings.Contains(csv, "personal-routines,,,ops,") {
		t.Fatalf("CSV: %d %s", code, csv)
	}
	for _, doc := range []string{out, csv} {
		if strings.Contains(doc, "my private zebra routine") || strings.Contains(doc, sq.ID) {
			t.Fatalf("the inventory names a personal routine: %s", doc)
		}
	}
	if _, err := inventory.Verify([]byte(strings.Replace(out, `"count":1`, `"count":0`, 1)), ""); err == nil {
		t.Fatal("a changed inventory verified")
	}
}

// SB-5007: sb inventory export reaches a running cell, checks the export against a pinned
// key, and writes the JSON and the CSV.
func TestInventoryExportAgainstARunningCell(t *testing.T) {
	l := startLive(t)
	out := t.TempDir()
	sg, err := l.cell.AnchorSigner(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if code, o := sb(t, "inventory", "export", "--cell", l.url, "--out", out, "--key", sg.Key()); code != 0 || !strings.Contains(o, "verified") {
		t.Fatalf("sb inventory export: %d %s", code, o)
	}
	raw, err := os.ReadFile(filepath.Join(out, "ai-inventory.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inventory.Verify(raw, sg.Key()); err != nil {
		t.Fatal(err)
	}
	if csv, err := os.ReadFile(filepath.Join(out, "ai-inventory.csv")); err != nil || !strings.HasPrefix(string(csv), "kind,name,") {
		t.Fatalf("the CSV: %v %s", err, csv)
	}
	if code, o := sb(t, "inventory", "export", "--cell", l.url, "--out", out, "--key", "ed25519:AAAA"); code == 0 {
		t.Fatalf("an export passed against another key: %s", o)
	}
}
