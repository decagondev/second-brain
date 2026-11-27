package ingest

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/ports"
)

// Scans, images and recordings (sb-ingest(7), What a model read; SB-6101, SB-6102): the
// parts of a document a connector couldn't read as text are read by the gated vision or
// transcribe tier, and their text joins the document after a marker naming the reader, the
// part and its page. Every chunk that holds any of it carries ReadBy, the AI mark of text a
// model read (LAW-28), and is screened as any text is (LAW-09). With no reader, or a reader
// that fails, the part is skipped and counted.

// Readers of a document's parts.
const (
	ReadByVision     = "vision"
	ReadByTranscribe = "transcribe"
)

// MaxParts is how many images and recordings of one document are read.
const MaxParts = 20

// markerRe finds a part's marker in the normalised text: its reader and page.
var markerRe = regexp.MustCompile(`\[Read by the (vision|transcribe) model from [^\]\n]{0,120}?(?:, page ([0-9]+))?\]`)

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9 ._-]`)

func marker(reader string, p ports.Part) string {
	name := unsafeName.ReplaceAllString(p.Name, "")
	if len(name) > 80 {
		name = name[:80]
	}
	m := "[Read by the " + reader + " model from " + name
	if p.Page > 0 {
		m += fmt.Sprintf(", page %d", p.Page)
	}
	return m + "]"
}

// readParts is the text the readers make of doc's images and recordings, each after its
// marker; "" when there is none.
func (p Pipeline) readParts(ctx context.Context, kind string, doc ports.Document) string {
	var b strings.Builder
	n := 0
	read := func(reader string, f func(context.Context, ports.Part) (string, error), what string, parts []ports.Part) {
		for _, part := range parts {
			if n >= MaxParts {
				p.unread(kind, what)
				continue
			}
			n++
			if f == nil {
				p.unread(kind, what)
				continue
			}
			t, err := f(ctx, part)
			if t = strings.TrimSpace(t); err != nil || t == "" {
				p.unread(kind, what)
				continue
			}
			b.WriteString("\n\n" + marker(reader, part) + "\n" + t)
		}
	}
	read(ReadByVision, p.Vision, "image", doc.Images)
	read(ReadByTranscribe, p.Transcribe, "recording", doc.Media)
	return b.String()
}

func (p Pipeline) unread(kind, what string) {
	if p.Unread != nil {
		p.Unread(kind, what)
	}
}

// readMarks are the markers in the normalised text: where each starts, in runes, its reader
// and its page.
type readMark struct {
	at     int
	reader string
	page   int
}

func readMarks(text string) []readMark {
	var out []readMark
	for _, m := range markerRe.FindAllStringSubmatchIndex(text, -1) {
		rm := readMark{at: utf8.RuneCountInString(text[:m[0]]), reader: text[m[2]:m[3]]}
		if m[4] >= 0 {
			_, _ = fmt.Sscanf(text[m[4]:m[5]], "%d", &rm.page)
		}
		out = append(out, rm)
	}
	return out
}

// readBy is the reader and page of a chunk from start to end: the marker it starts in or
// after, or the first one it reaches; "" for a chunk of the source's own text only.
func readBy(marks []readMark, start, end int) (string, int) {
	if len(marks) == 0 || end <= marks[0].at {
		return "", 0
	}
	cur := marks[0]
	for _, m := range marks {
		if m.at <= start {
			cur = m
		}
	}
	return cur.reader, cur.page
}
