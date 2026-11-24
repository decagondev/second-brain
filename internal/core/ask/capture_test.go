package ask_test

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/policy"
)

// words is a small bag-of-words embedder for the repeat tests.
type words struct{}

func (words) Model() string { return "test-words" }
func (words) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		v := make([]float32, 256)
		for _, w := range strings.Fields(strings.ToLower(t)) {
			h := fnv.New32a()
			_, _ = h.Write([]byte(w))
			v[h.Sum32()%256]++
		}
		out[i] = v
	}
	return out, nil
}

func captureWorld(t *testing.T) (ask.Stewards, *fakeBus, *stq) {
	t.Helper()
	a, _ := setup(t, &model{reply: `{"claims":[]}`})
	a.Emails = func(_ context.Context, u string) ([]string, error) { return []string{u + "@acme.example"}, nil }
	a.Embedder = words{}
	bus := &fakeBus{}
	store := &stq{m: map[string]ask.StewardQuestion{}}
	n := 0
	notice := ask.NoticeNoAccess
	for _, u := range []string{"alice", "bob", "cat"} {
		_ = a.Store.PutTrace(context.Background(), ask.Trace{ID: "tr_" + u, Tenant: tenant, User: u, Space: "company", Response: ask.Response{Notice: &notice}, At: time.Now()})
	}
	x := ask.Stewards{Agent: a, Store: store, Audit: &rows{}, Bus: bus,
		Policy: policy.Kernel{Engine: allowAll{}, Recorder: rec{}},
		Facts:  func(string) ([]string, string, bool) { return []string{"group:answerers"}, "", true },
		Readers: func(context.Context, string) ([]string, error) {
			return []string{"alice@acme.example", "sam@acme.example"}, nil
		},
		NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }, Now: time.Now}
	return x, bus, store
}

// SB-3401, SB-3403: a note is the reply, the question only when the steward keeps it,
// never the asker, with review_by 180 days ahead; a republish keeps the choice.
func TestNoteKeepsQuestionAndComesDue(t *testing.T) {
	ctx := context.Background()
	x, bus, _ := captureWorld(t)
	sam := scope(t, "sam")
	zero := 0
	for _, keep := range []bool{true, false} {
		sq, err := x.Send(ctx, scope(t, "alice"), "where is the parking garage?", "tr_alice")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := x.Reply(ctx, sam, "company", sq.ID, "Behind building B.", nil); err != nil {
			t.Fatal(err)
		}
		if _, err := x.Note(ctx, sam, "company", sq.ID, ask.NoteRequest{Title: "Parking", KeepQuestion: keep, Confirm: &zero}); err != nil {
			t.Fatal(err)
		}
		var p ask.NotePayload
		_ = json.Unmarshal(bus.calls[len(bus.calls)-1].Payload, &p)
		if strings.Contains(p.Markdown, "alice") {
			t.Fatal("the note names its asker")
		}
		if got := strings.Contains(p.Markdown, "where is the parking garage?"); got != keep {
			t.Fatalf("keep %v, the note %q", keep, p.Markdown)
		}
		due, ok := ingest.ParseReviewBy(ingest.FrontMatterDate(p.Markdown))
		want := time.Now().UTC().Add(ask.NoteReviewIn)
		if !ok || due.Sub(want) > 24*time.Hour || want.Sub(due) > 24*time.Hour {
			t.Fatalf("review_by %v, want about %v", due, want)
		}
	}
}

// SB-3402: a near-identical question is linked for the stewards both ways, Answer both
// replies to each asker, and neither asker sees the other's question.
func TestRepeatsAndAnswerBoth(t *testing.T) {
	ctx := context.Background()
	x, _, store := captureWorld(t)
	a, err := x.Send(ctx, scope(t, "alice"), "how many vacation days do I get each year", "tr_alice")
	if err != nil || a.Like != "" {
		t.Fatalf("%+v %v", a, err)
	}
	b, err := x.Send(ctx, scope(t, "bob"), "how many vacation days do we get each year", "tr_bob")
	if err != nil || b.Like != a.ID {
		t.Fatalf("bob's question doesn't look like alice's: %+v %v", b, err)
	}
	c, err := x.Send(ctx, scope(t, "cat"), "where is the printer on floor two", "tr_cat")
	if err != nil || c.Like != "" {
		t.Fatalf("an unrelated question was linked: %+v %v", c, err)
	}
	sam := scope(t, "sam")
	items, err := x.Queue(ctx, sam, "company")
	if err != nil {
		t.Fatal(err)
	}
	like := map[string]string{}
	for _, it := range items {
		like[it.ID] = it.Like
	}
	if like[a.ID] != b.ID || like[b.ID] != a.ID || like[c.ID] != "" {
		t.Fatalf("likes %v", like)
	}
	if _, err := x.ReplyBoth(ctx, sam, "company", a.ID, "25 days.", nil, true); err != nil {
		t.Fatal(err)
	}
	if store.m[a.ID].Reply == nil || store.m[b.ID].Reply == nil || store.m[c.ID].Reply != nil {
		t.Fatal("Answer both didn't answer exactly the two")
	}
	for _, u := range []string{"alice", "bob"} {
		mine, _ := x.Mine(ctx, scope(t, u))
		js, _ := json.Marshal(mine)
		other := map[string]string{"alice": b.ID, "bob": a.ID}[u]
		if len(mine) != 1 || mine[0].Reply == nil || strings.Contains(string(js), other) || strings.Contains(string(js), "like") {
			t.Fatalf("%s sees %s", u, js)
		}
	}
	// Without an embedder nothing is linked, and the question is still sent.
	x.Agent.Embedder = nil
	d, err := x.Send(ctx, scope(t, "cat"), "how many vacation days do I get each year", "tr_cat")
	if err != nil || d.Like != "" {
		t.Fatalf("%+v %v", d, err)
	}
}

// SB-3404: an open question 5 working days old is overdue on the queue; the company-wide
// queue shows one that waited, which is published as a note only from its own space.
func TestOverdueAndWaitedOnTheQueue(t *testing.T) {
	ctx := context.Background()
	x, _, store := captureWorld(t)
	now := time.Now().UTC()
	x.Zone = time.UTC
	store.m["qs_old"] = ask.StewardQuestion{ID: "qs_old", Tenant: tenant, Asker: "alice", Question: "old one", Space: "company", At: now.AddDate(0, 0, -8)}
	store.m["qs_new"] = ask.StewardQuestion{ID: "qs_new", Tenant: tenant, Asker: "alice", Question: "new one", Space: "company", At: now}
	store.m["qs_fin"] = ask.StewardQuestion{ID: "qs_fin", Tenant: tenant, Asker: "alice", Question: "finance one", Space: "spc_finance", At: now.AddDate(0, 0, -15),
		Also: "company", Waited: 10, Reply: &ask.StewardReply{By: "sam", Text: "Yes."}}
	store.m["qs_fin2"] = ask.StewardQuestion{ID: "qs_fin2", Tenant: tenant, Asker: "alice", Question: "finance two", Space: "spc_finance", At: now}
	x.Store = between{store}
	items, err := x.Queue(ctx, scope(t, "sam"), "company")
	if err != nil {
		t.Fatal(err)
	}
	over, seen := map[string]bool{}, map[string]ask.StewardItem{}
	for _, it := range items {
		over[it.ID], seen[it.ID] = it.Overdue, it
	}
	if !over["qs_old"] || over["qs_new"] {
		t.Fatalf("overdue %v", over)
	}
	if f, ok := seen["qs_fin"]; !ok || f.Waited != 10 || f.FromSpace != "finance" {
		t.Fatalf("the question that waited: %+v", f)
	}
	if _, ok := seen["qs_fin2"]; ok {
		t.Fatal("a finance question that didn't wait is on the company-wide queue")
	}
	zero := 0
	if _, err := x.Note(ctx, scope(t, "sam"), "company", "qs_fin", ask.NoteRequest{Title: "Fin", Confirm: &zero}); err == nil || !strings.Contains(err.Error(), "own space") {
		t.Fatalf("a note of a waited question from company: %v", err)
	}
}

// between is stq with every question in StewardQuestionsBetween.
type between struct{ *stq }

func (b between) StewardQuestionsBetween(_ context.Context, _ string, from, to time.Time) ([]ask.StewardQuestion, error) {
	var out []ask.StewardQuestion
	for _, q := range b.m {
		if !q.At.Before(from) && q.At.Before(to) {
			out = append(out, q)
		}
	}
	return out, nil
}
