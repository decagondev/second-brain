package ask

import (
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
