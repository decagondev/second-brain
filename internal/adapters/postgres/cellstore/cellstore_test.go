package cellstore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/holds"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/proposals"
	"example.com/second-brain/internal/core/retention"
	"example.com/second-brain/internal/core/reviews"
)

const tenant = "acme-logistics"

// open connects to SB_TEST_POSTGRES_DSN in a fresh schema, or skips (make integration).
func open(t *testing.T) DB {
	t.Helper()
	dsn := os.Getenv("SB_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("SB_TEST_POSTGRES_DSN not set")
	}
	ctx := context.Background()
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	name := "t_" + hex.EncodeToString(b)
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+name+" CASCADE")
		admin.Close()
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = name
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	d := DB{Pool: pool, Tenant: tenant}
	for i := 0; i < 2; i++ { // idempotent
		if err := d.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	return d
}

func TestDirectoryAndSessions(t *testing.T) {
	d := open(t)
	ctx := context.Background()
	dir := Directory{d}
	ann := ports.User{ID: "usr_ann", Tenant: tenant, UserName: "ann", Emails: []string{"Ann@acme.example"}, Active: true, Manager: "usr_bo"}
	if err := dir.PutUser(ctx, ann); err != nil {
		t.Fatal(err)
	}
	_ = dir.PutUser(ctx, ports.User{ID: "usr_bo", Tenant: tenant, UserName: "bo"})
	if active, known, err := dir.Active(ctx, tenant, "usr_ann"); !active || !known || err != nil {
		t.Fatalf("active %v %v %v", active, known, err)
	}
	if active, known, _ := dir.Active(ctx, tenant, "usr_bo"); active || !known {
		t.Fatal("an inactive user is active")
	}
	if _, known, _ := dir.Active(ctx, "beta-freight", "usr_ann"); known {
		t.Fatal("LAW-02: a user known in another tenant")
	}
	if u, found, _ := dir.UserByName(ctx, tenant, "ann"); !found || u.Manager != "usr_bo" {
		t.Fatalf("by name %+v", u)
	}
	if us, _ := dir.UsersByEmail(ctx, tenant, "ann@ACME.example"); len(us) != 1 {
		t.Fatalf("by email %v", us)
	}
	_ = dir.PutGroup(ctx, ports.Group{ID: "grp_1", Tenant: tenant, Name: "it-leads", Members: []string{"usr_bo", "usr_ann"}})
	if gs, _ := dir.GroupsOf(ctx, tenant, "usr_ann"); len(gs) != 1 || gs[0] != "it-leads" {
		t.Fatalf("groups %v", gs)
	}
	if ms, err := dir.Members(ctx, tenant, "it-leads"); err != nil || ms[0] != "usr_ann" {
		t.Fatalf("members %v %v", ms, err)
	}
	if _, err := dir.Members(ctx, tenant, "nobody"); err == nil {
		t.Fatal("an unknown group has members")
	}
	if n, _ := dir.ActiveUsers(ctx, tenant); n != 1 {
		t.Fatalf("seats %d", n)
	}
	_ = dir.DeleteGroup(ctx, tenant, "grp_1")
	if gs, _ := dir.GroupsOf(ctx, tenant, "usr_ann"); len(gs) != 0 {
		t.Fatal("a deleted group still counts")
	}

	ss := Sessions{d}
	now := time.Now().UTC().Truncate(time.Microsecond)
	_ = ss.Put(ctx, ports.Session{ID: "cookie-1", Tenant: tenant, User: "usr_ann", SignedIn: now, Validated: now})
	_ = ss.Put(ctx, ports.Session{ID: "cookie-2", Tenant: tenant, User: "usr_ann", SignedIn: now, Validated: now})
	if s, found, _ := ss.Get(ctx, "cookie-1"); !found || s.User != "usr_ann" || !s.SignedIn.Equal(now) {
		t.Fatalf("session %+v", s)
	}
	var raw int
	_ = d.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM cell_docs WHERE id = 'cookie-1'`).Scan(&raw)
	if raw != 0 {
		t.Fatal("a session id is stored as is")
	}
	if n, _ := ss.RevokeUser(ctx, tenant, "usr_ann"); n != 2 {
		t.Fatalf("revoked %d", n)
	}
	if _, found, _ := ss.Get(ctx, "cookie-2"); found {
		t.Fatal("a revoked session is found")
	}
}

func TestTracesAndRetention(t *testing.T) {
	d := open(t)
	ctx := context.Background()
	tr := Traces{d}
	old := time.Now().UTC().AddDate(-2, 0, 0)
	for _, x := range []ask.Trace{{ID: "tr_1", Tenant: tenant, User: "usr_ann", Space: "spc_finance", At: old},
		{ID: "tr_2", Tenant: tenant, User: "usr_bo", Space: "company", At: old}, {ID: "tr_3", Tenant: tenant, User: "usr_ann", At: time.Now().UTC()}} {
		if err := tr.PutTrace(ctx, x); err != nil {
			t.Fatal(err)
		}
	}
	if err := tr.PutTrace(ctx, ask.Trace{ID: "tr_1", Tenant: tenant}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("a trace id reused: %v", err)
	}
	if _, found, _ := tr.GetTrace(ctx, "beta-freight", "tr_1"); found {
		t.Fatal("LAW-02: another tenant's trace")
	}
	_ = tr.AppendQuestion(ctx, tenant, "usr_ann", "cv_1", "q1")
	_ = tr.AppendQuestion(ctx, tenant, "usr_ann", "cv_1", "q2")
	if err := tr.AppendQuestion(ctx, tenant, "usr_bo", "cv_1", "q3"); err == nil {
		t.Fatal("someone else appended to a conversation")
	}
	if qs, found, _ := tr.Questions(ctx, tenant, "usr_ann", "cv_1"); !found || len(qs) != 2 {
		t.Fatalf("questions %v", qs)
	}
	if _, found, _ := tr.Questions(ctx, tenant, "usr_bo", "cv_1"); found {
		t.Fatal("someone else read a conversation")
	}
	_ = tr.TagSpace(ctx, tenant, "cv_1", "spc_finance")
	_ = tr.TagSpace(ctx, tenant, "cv_1", "spc_people")
	if err := tr.TagSpace(ctx, tenant, "cv_x", "spc_finance"); err == nil {
		t.Fatal("tagged a conversation that doesn't exist")
	}
	_ = tr.PutFeedback(ctx, ask.Feedback{Tenant: tenant, User: "usr_ann", Trace: "tr_1", Rating: "up", At: old})
	if fs, _ := tr.FeedbackBetween(ctx, tenant, old.Add(-time.Hour), time.Now()); len(fs) != 1 {
		t.Fatalf("feedback %v", fs)
	}
	if ts, _ := tr.Between(ctx, tenant, old.Add(-time.Hour), old.Add(time.Hour)); len(ts) != 2 {
		t.Fatalf("between %v", ts)
	}
	// A hold on finance keeps tr_1 and its feedback; tr_2 goes; tr_3 is too young.
	held := func(space, user string) bool { return space == "" || space == "spc_finance" }
	held2 := func(space, user string) bool { return space == "spc_finance" }
	cut := time.Now().UTC().AddDate(-1, 0, 0)
	planned := map[string][2]int{}
	for _, v := range tr.Retention() {
		h := held2
		if v.Name() == "conversations" {
			h = held
		}
		n, k, err := v.Plan(ctx, tenant, cut, h)
		if err != nil {
			t.Fatal(err)
		}
		planned[v.Name()] = [2]int{n, k}
	}
	for _, v := range tr.Retention() {
		h := held2
		if v.Name() == "conversations" {
			h = held
		}
		n, k, err := v.Expire(ctx, tenant, cut, h)
		if err != nil {
			t.Fatal(err)
		}
		if planned[v.Name()] != [2]int{n, k} { // SB-1905: the plan is the sweep
			t.Fatalf("%s: planned %v, expired %d kept %d", v.Name(), planned[v.Name()], n, k)
		}
	}
	if _, found, _ := tr.GetTrace(ctx, tenant, "tr_1"); !found {
		t.Fatal("LAW-13: a held trace was deleted")
	}
	if _, found, _ := tr.GetTrace(ctx, tenant, "tr_2"); found {
		t.Fatal("an old trace with no hold was kept")
	}
	if fs, _ := tr.FeedbackBetween(ctx, tenant, old.Add(-time.Hour), time.Now()); len(fs) != 1 {
		t.Fatal("LAW-13: feedback on a held trace was deleted")
	}
	exported := map[string]int{}
	for _, v := range tr.Retention() {
		rs, err := v.Export(ctx, tenant, "usr_ann")
		if err != nil {
			t.Fatal(err)
		}
		exported[v.Name()] = len(rs)
	}
	n := 0
	for _, v := range tr.Retention() {
		k, _ := v.Erase(ctx, tenant, "usr_ann")
		if k != exported[v.Name()] { // SB-1903: the export is what erase removes
			t.Fatalf("%s: exported %d, erased %d", v.Name(), exported[v.Name()], k)
		}
		if rs, _ := v.Export(ctx, tenant, "usr_ann"); len(rs) != 0 {
			t.Fatalf("%s: export after erase: %d", v.Name(), len(rs))
		}
		n += k
	}
	if n != 4 { // feedback, conversation, tr_1, tr_3
		t.Fatalf("erased %d", n)
	}
}

// SB-2101: settings are kept past retention, erased with the person, and per tenant.
func TestNotifyPrefs(t *testing.T) {
	d := open(t)
	ctx := context.Background()
	tr := Traces{d}
	if p, err := tr.NotifyPrefs(ctx, tenant, "usr_ann"); err != nil || len(p.Off) != 0 {
		t.Fatalf("none: %+v %v", p, err)
	}
	for _, off := range [][]string{{"digest"}, {"replies", "watch"}} {
		if err := tr.PutNotifyPrefs(ctx, tenant, "usr_ann", notify.Prefs{Off: off}); err != nil {
			t.Fatal(err)
		}
	}
	if p, _ := tr.NotifyPrefs(ctx, tenant, "usr_ann"); len(p.Off) != 2 || p.Wants("replies") {
		t.Fatalf("replaced: %+v", p)
	}
	if p, _ := tr.NotifyPrefs(ctx, "beta-freight", "usr_ann"); len(p.Off) != 0 {
		t.Fatal("LAW-02: another tenant's settings")
	}
	var pref retention.Store
	for _, v := range tr.Retention() {
		if v.Name() == "notification settings" {
			pref = v
		}
	}
	if n, _, err := pref.Expire(ctx, tenant, time.Now().Add(time.Hour), nil); n != 0 || err != nil {
		t.Fatalf("settings expired: %d %v", n, err)
	}
	if rows, _ := pref.Export(ctx, tenant, "usr_ann"); len(rows) != 1 {
		t.Fatalf("export: %d", len(rows))
	}
	if n, err := pref.Erase(ctx, tenant, "usr_ann"); n != 1 || err != nil {
		t.Fatalf("erase: %d %v", n, err)
	}
}

func TestActionsAndDecisions(t *testing.T) {
	d := open(t)
	ctx := context.Background()
	as := Actions{d}
	a := approvals.Action{ID: "act_1", Tenant: tenant, State: approvals.PendingApproval, Requester: "usr_ann", ExpiresAt: time.Now().Add(time.Hour)}
	if err := as.Put(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := as.Put(ctx, a); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := as.Move(ctx, tenant, "act_1", approvals.Approved, func(*approvals.Action) error { return nil }); !errors.Is(err, approvals.ErrMoved) {
		t.Fatalf("a move from the wrong state: %v", err)
	}
	// Two decisions at once: exactly one wins.
	var wg sync.WaitGroup
	wins := make(chan bool, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := as.Move(ctx, tenant, "act_1", approvals.PendingApproval, func(x *approvals.Action) error { x.State = approvals.Approved; return nil })
			wins <- err == nil
		}()
	}
	wg.Wait()
	close(wins)
	n := 0
	for w := range wins {
		if w {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("%d decisions won, want 1", n)
	}
	if _, err := as.Move(ctx, tenant, "act_1", approvals.Approved, func(x *approvals.Action) error { x.State = approvals.PendingApproval; return nil }); err == nil {
		t.Fatal("a move that isn't a transition")
	}
	rc := Receipts{d}
	_ = rc.Put(ctx, tenant, "k1", "rcpt_1")
	_ = rc.Put(ctx, tenant, "k1", "rcpt_2")
	if r, found, _ := rc.Get(ctx, tenant, "k1"); !found || r != "rcpt_1" {
		t.Fatalf("a key's receipt changed: %q", r)
	}

	ps := Proposals{d}
	_ = ps.Put(ctx, proposals.Proposal{ID: "prp_1"})
	if _, err := ps.Update(ctx, "prp_x", func(*proposals.Proposal) error { return nil }); !errors.Is(err, proposals.ErrNotFound) {
		t.Fatalf("update of nothing: %v", err)
	}
	stop := errors.New("no")
	if _, err := ps.Update(ctx, "prp_1", func(p *proposals.Proposal) error { p.ID = "changed"; return stop }); !errors.Is(err, stop) {
		t.Fatal(err)
	}
	if p, _, _ := ps.Get(ctx, "prp_1"); p.ID != "prp_1" {
		t.Fatal("a refused update was stored")
	}
	rs := Reviews{d}
	_ = rs.Put(ctx, reviews.Review{ID: "rev_2"})
	_ = rs.Put(ctx, reviews.Review{ID: "rev_1"})
	if l, _ := rs.List(ctx); len(l) != 2 || l[0].ID != "rev_1" {
		t.Fatalf("reviews %v", l)
	}
	hs := Holds{d}
	_ = hs.Put(ctx, holds.Hold{ID: "hld_1", Space: "finance", Users: []string{"usr_ann"}})
	if set, err := holds.Active(ctx, hs); err != nil || !set.HeldSpace("spc_finance") {
		t.Fatalf("holds %v %v", set, err)
	}
}

func TestQueue(t *testing.T) {
	d := open(t)
	ctx := context.Background()
	q := Queue{DB: d, Kind: "slack"}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); _ = q.Append(ctx, map[string]int{"n": i}) }(i)
	}
	wg.Wait()
	vs, next, err := q.Read(ctx, 4)
	if err != nil || next != 10 || len(vs) != 6 {
		t.Fatalf("%d values, next %d, %v", len(vs), next, err)
	}
	if _, _, err := q.Read(ctx, 11); !errors.Is(err, ErrCursor) {
		t.Fatalf("a cursor past the end: %v", err)
	}
}
