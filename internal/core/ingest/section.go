package ingest

import (
	"sort"
	"strings"
	"unicode/utf8"
)

// MaxSection is the longest section a chunk records, in characters (SB-3002).
const MaxSection = 120

// heading is one heading line: where it starts, in runes, and its text.
type heading struct {
	at   int
	text string
}

// headings are the Markdown heading lines of levels 1 to 3 in normalised text: "#", "##"
// or "###", a space, then the heading. Connectors write other formats' headings this way.
func headings(text string) []heading {
	var out []heading
	at := 0
	for _, line := range strings.Split(text, "\n") {
		if h, ok := headingText(line); ok {
			out = append(out, heading{at: at, text: h})
		}
		at += utf8.RuneCountInString(line) + 1
	}
	return out
}

func headingText(line string) (string, bool) {
	n := 0
	for n < len(line) && n < 4 && line[n] == '#' {
		n++
	}
	if n == 0 || n > 3 || n >= len(line) || line[n] != ' ' {
		return "", false
	}
	h := strings.Join(strings.Fields(strings.Trim(line[n:], " #")), " ")
	if h == "" {
		return "", false
	}
	if rs := []rune(h); len(rs) > MaxSection {
		h = strings.TrimSpace(string(rs[:MaxSection-1])) + "…"
	}
	return h, true
}

// sectionAt is the nearest heading at or above start, or "".
func sectionAt(hs []heading, start int) string {
	s := ""
	for _, h := range hs {
		if h.at > start {
			break
		}
		s = h.text
	}
	return s
}

// lines maps a rune offset in the normalised text (the title, a blank line, then the text)
// to its line in the document's own text, from 1 (SB-3006). A line in the title is line 1.
func lines(rs []rune, title, body string) func(int) int {
	var breaks []int // offsets of each newline
	for i, r := range rs {
		if r == '\n' {
			breaks = append(breaks, i)
		}
	}
	first := 2 // the body starts after the title's line and a blank one
	if strings.TrimSpace(title) == "" {
		lead := body[:len(body)-len(strings.TrimLeft(body, " \t\r\n"))]
		first = -strings.Count(strings.ReplaceAll(strings.ReplaceAll(lead, "\r\n", "\n"), "\r", "\n"), "\n")
	}
	return func(at int) int {
		n := sort.SearchInts(breaks, at) // newlines before at
		if l := n - first + 1; l > 1 {
			return l
		}
		return 1
	}
}

// Section is the section of a chunk starting at start in normalised text (SB-3002).
func Section(text string, start int) string { return sectionAt(headings(text), start) }
