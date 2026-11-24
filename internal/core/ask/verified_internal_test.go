package ask

import (
	"context"
	"strconv"
	"testing"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ports"
)

// SB-3603: a mark's own interval sets when it lapses, on the list and for Coming due.
func TestMarkInterval(t *testing.T) {
	now := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	c := ports.Chunk{Kind: "docs", SourceID: "a", Version: 3, Seen: now.AddDate(0, 0, -400)}
	m := ReviewMark{Doc: "docs:a", Version: 3, At: now.AddDate(0, 0, -31), Days: 30}
	if covers(m, true, c, time.Time{}, 365, now) {
		t.Fatal("a 30-day mark 31 days old still covers")
	}
	m.Days = 90
	if !covers(m, true, c, time.Time{}, 365, now) {
		t.Fatal("a 90-day mark 31 days old lapsed")
	}
	at, ok := staleAt(c, ReviewMark{Version: 3, At: now, Days: 30}, true, 365)
	if !ok || !at.Equal(now.AddDate(0, 0, 30)) {
		t.Fatalf("stale at %v", at)
	}
	if markAfter(ReviewMark{}, 180) != 180 {
		t.Fatal("no interval isn't the default")
	}
}

// SB-3604: Not helpful ratings after the mark, of answers citing the marked version, list
// it again at 3; earlier ones and other versions don't count.
func TestUnhelpfulSinceTheMark(t *testing.T) {
	now := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	marks := map[string]ReviewMark{"docs:a": {Doc: "docs:a", Version: 3, At: now.Add(-48 * time.Hour)}}
	cite := func(c string) Response {
		return Response{Answer: Answer{Claims: []answer.Claim{{Text: "x", Cites: []answer.Cite{{Cite: c}}}}}}
	}
	traces := []Trace{{ID: "t1", Response: cite("docs:a@3")}, {ID: "t2", Response: cite("docs:a@3")}, {ID: "t3", Response: cite("docs:a@3")},
		{ID: "t4", Response: cite("docs:a@2")}, {ID: "t5", Response: cite("docs:a@3")}}
	fbs := []Feedback{{Trace: "t1", Rating: "down", At: now}, {Trace: "t2", Rating: "down", At: now}, {Trace: "t3", Rating: "up", At: now},
		{Trace: "t4", Rating: "down", At: now}, {Trace: "t5", Rating: "down", At: now.Add(-72 * time.Hour)}}
	if n := unhelpfulSince(traces, fbs, marks)["docs:a"]; n != 2 {
		t.Fatalf("counted %d", n)
	}
	fbs[2].Rating = "down"
	got := unhelpfulSince(traces, fbs, marks)
	c := ports.Chunk{Kind: "docs", SourceID: "a", Version: 3, Spaces: []string{"spc_ops"}, Seen: now}
	cand, _ := dueFor("spc_ops", []ports.Chunk{c}, reviewFacts{marks: marks, unhelpful: got}, 365, now)
	if got["docs:a"] != 3 || len(cand) != 1 {
		t.Fatalf("3 unhelpful since the mark: %v %d", got, len(cand))
	}
	cand, _ = dueFor("spc_ops", []ports.Chunk{c}, reviewFacts{marks: marks, unhelpful: map[string]int{"docs:a": 2}}, 365, now)
	if len(cand) != 0 {
		t.Fatal("2 unhelpful listed it")
	}
}

// SB-3602: notes are verified; other documents only with a live mark.
func TestIsNote(t *testing.T) {
	if !isNote(ports.Chunk{Kind: "docs", SourceID: "notes/ops/qs_1.md"}) || isNote(ports.Chunk{Kind: "docs", SourceID: "Ops/notes/qs_1.md"}) || isNote(ports.Chunk{Kind: "chat", SourceID: "notes/ops/qs_1.md"}) {
		t.Fatal("isNote")
	}
}

type marksOf []ReviewMark

func (m marksOf) PutReviewMark(context.Context, ReviewMark) error { return nil }
func (m marksOf) ReviewMarks(context.Context, string, string) ([]ReviewMark, error) {
	return m, nil
}

// SB-3607: each cited document counts once; a live mark at the cited version or a note is
// verified, a mark at another version or past its interval isn't, and no cites is no count.
func TestVerifiedCount(t *testing.T) {
	now := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	a := Agent{Tenant: "acme-logistics", Now: func() time.Time { return now }, Marks: marksOf{
		{Doc: "docs:a", Version: 3, At: now.AddDate(0, 0, -10), Days: 30},
		{Doc: "docs:b", Version: 1, At: now.AddDate(0, 0, -10), Days: 30},
		{Doc: "docs:c", Version: 2, At: now.AddDate(0, 0, -40), Days: 30},
	}}
	ch := func(id string, v int) ports.Chunk {
		return ports.Chunk{Kind: "docs", SourceID: id, Version: v, Cite: "docs:" + id + "@" + strconv.Itoa(v), Spaces: []string{"spc_ops"}, Seen: now}
	}
	chunks := []ports.Chunk{ch("a", 3), ch("b", 2), ch("c", 2), ch("notes/ops/qs_1.md", 1)}
	cl := func(cs ...string) answer.Claim {
		var out []answer.Cite
		for _, c := range cs {
			out = append(out, answer.Cite{Cite: c})
		}
		return answer.Claim{Text: "x", Cites: out}
	}
	v := a.verifiedCount(context.Background(), []answer.Claim{cl("docs:a@3", "docs:b@2"), cl("docs:a@3", "docs:c@2"), cl("docs:notes/ops/qs_1.md@1")}, chunks)
	if v == nil || v.Of != 4 || v.Marked != 2 {
		t.Fatalf("count %+v", v)
	}
	if VerifiedLine(*v) != "2 of 4 sources verified." {
		t.Fatal(VerifiedLine(*v))
	}
	if a.verifiedCount(context.Background(), []answer.Claim{{Text: "x", Uncited: true}}, chunks) != nil {
		t.Fatal("a count without cites")
	}
}
