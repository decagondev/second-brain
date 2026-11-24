package folder

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Meeting transcripts (sb-connector(3), folder; SB-3701): WebVTT and SubRip, cue by cue,
// each a heading with its start time and a line with its speaker and text.

var (
	timingRe = regexp.MustCompile(`^\s*((?:\d+:)?\d{1,2}:\d{2}[.,]\d{1,3})\s+-->\s+((?:\d+:)?\d{1,2}:\d{2}[.,]\d{1,3})`)
	voiceRe  = regexp.MustCompile(`^<v(?:\.[^ >]*)?\s+([^>]+)>`)
	tagRe    = regexp.MustCompile(`<[^>]*>`)
	speakRe  = regexp.MustCompile(`^([\p{L}][\p{L}\p{M} .'-]{0,40}):\s+(.+)$`)
)

// TranscriptText is a .vtt or .srt file's cues as text, and how many cues were skipped.
func TranscriptText(b []byte) (string, int) {
	if !utf8.Valid(b) {
		return "", 0
	}
	s := strings.TrimPrefix(string(b), "\ufeff")
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	var out strings.Builder
	skipped := 0
	for _, block := range strings.Split(s, "\n\n") {
		lines := strings.Split(strings.Trim(block, "\n"), "\n")
		if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
			continue
		}
		head := strings.TrimSpace(lines[0])
		if strings.HasPrefix(head, "WEBVTT") || strings.HasPrefix(head, "NOTE") || head == "STYLE" || head == "REGION" {
			continue
		}
		at, text, ok := cue(lines)
		if !ok {
			skipped++
			continue
		}
		fmt.Fprintf(&out, "### %s\n%s\n\n", at, text)
	}
	switch {
	case skipped == 1:
		out.WriteString("1 cue couldn't be read.\n")
	case skipped > 1:
		fmt.Fprintf(&out, "%d cues couldn't be read.\n", skipped)
	}
	return strings.TrimRight(out.String(), "\n"), skipped
}

// cue is one block's start time as hh:mm:ss and its speaker and text on one line.
func cue(lines []string) (string, string, bool) {
	i := 0
	if !timingRe.MatchString(lines[0]) && len(lines) > 1 {
		i = 1 // an identifier or an SRT index
	}
	m := timingRe.FindStringSubmatch(lines[i])
	if m == nil {
		return "", "", false
	}
	at, ok := clock(m[1])
	if !ok {
		return "", "", false
	}
	speaker := ""
	var parts []string
	for _, l := range lines[i+1:] {
		l = strings.TrimSpace(l)
		if v := voiceRe.FindStringSubmatch(l); v != nil && speaker == "" {
			speaker = strings.TrimSpace(v[1])
		}
		if l = strings.TrimSpace(tagRe.ReplaceAllString(l, "")); l != "" {
			parts = append(parts, l)
		}
	}
	text := strings.Join(strings.Fields(strings.Join(parts, " ")), " ")
	if text == "" {
		return "", "", false
	}
	if speaker == "" {
		if s := speakRe.FindStringSubmatch(text); s != nil {
			speaker, text = strings.TrimSpace(s[1]), s[2]
		}
	}
	if speaker != "" {
		text = speaker + ": " + text
	}
	return at, text, true
}

// clock is a cue time ([h:]mm:ss.mmm or ,mmm) as hh:mm:ss.
func clock(t string) (string, bool) {
	t = strings.NewReplacer(",", ".").Replace(t)
	t, _, _ = strings.Cut(t, ".")
	ps := strings.Split(t, ":")
	if len(ps) == 2 {
		ps = append([]string{"0"}, ps...)
	}
	var n [3]int
	for i, p := range ps {
		v, err := strconv.Atoi(p)
		if err != nil || v < 0 || (i > 0 && v > 59) {
			return "", false
		}
		n[i] = v
	}
	return fmt.Sprintf("%02d:%02d:%02d", n[0], n[1], n[2]), true
}
