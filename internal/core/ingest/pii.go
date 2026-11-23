package ingest

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// FlagPersonal is a chunk that holds personal data: an IBAN whose check digits are right,
// or a payment card number that passes the Luhn check (sb-ingest(7), Personal data;
// SB-1904). The chunk is indexed and can be cited; its snippets and the text the model
// reads are masked (Mask), so an answer quotes only the last four characters.
const FlagPersonal = "personal_data"

// Mask is the character that replaces a masked one: one rune, so spans keep their places.
const Mask = '•'

var (
	ibanLike = regexp.MustCompile(`\b[A-Z]{2}[0-9]{2}(?: ?[A-Z0-9]){11,30}`)
	digitRun = regexp.MustCompile(`[0-9]+(?:[ -][0-9]+)*`)
)

// PIISpan is one number found, in runes.
type PIISpan struct {
	Start, End int
	Kind       string // iban or card
}

// PIISpans finds the IBANs and card numbers in text.
func PIISpans(text string) []PIISpan {
	var out []PIISpan
	add := func(kind string, m []int, ok func(string) (int, bool)) {
		if n, good := ok(text[m[0]:m[1]]); good {
			out = append(out, PIISpan{utf8.RuneCountInString(text[:m[0]]), utf8.RuneCountInString(text[:m[0]+n]), kind})
		}
	}
	for _, m := range ibanLike.FindAllStringIndex(text, -1) {
		add("iban", m, validIBANPrefix)
	}
	for _, m := range digitRun.FindAllStringIndex(text, -1) {
		if (m[0] > 0 && isAlnum(text[m[0]-1])) || (m[1] < len(text) && isAlnum(text[m[1]])) {
			continue // part of a word
		}
		out = append(out, cardsIn(text, m[0], m[1])...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}

// HasPII reports whether text holds an IBAN or a card number.
func HasPII(text string) bool { return len(PIISpans(text)) > 0 }

func isAlnum(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z'
}

// validIBANPrefix returns the longest prefix of s, ending at a character, that is an IBAN
// (15 to 34 characters without spaces, mod-97 = 1) and isn't followed by more of a word.
func validIBANPrefix(s string) (int, bool) {
	for n := len(s); n > 0; n-- {
		if s[n-1] == ' ' {
			continue
		}
		if n < len(s) && s[n] != ' ' {
			continue // would cut a group in two
		}
		compact := strings.ReplaceAll(s[:n], " ", "")
		if len(compact) < 15 || len(compact) > 34 {
			continue
		}
		if mod97(compact[4:]+compact[:4]) == 1 {
			return n, true
		}
	}
	return 0, false
}

func mod97(s string) int {
	r := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			r = (r*10 + int(c-'0')) % 97
		case c >= 'A' && c <= 'Z':
			v := int(c-'A') + 10
			r = (r*100 + v) % 97
		default:
			return -1
		}
	}
	return r
}

// cardsIn finds card numbers in the run of digit groups text[from:to]: any run of whole
// groups that is a card, longest first, so "Ref 12 4111 1111 1111 1111" finds the card.
func cardsIn(text string, from, to int) []PIISpan {
	type group struct{ start, end int }
	var gs []group
	for i := from; i < to; {
		j := i
		for j < to && text[j] >= '0' && text[j] <= '9' {
			j++
		}
		gs = append(gs, group{i, j})
		i = j + 1 // skip the separator
	}
	var out []PIISpan
	buf := make([]byte, 0, 19)
	for a := 0; a < len(gs); {
		// Grow the digits group by group, at most 19, and keep the longest card.
		buf, best := buf[:0], -1
		for b := a; b < len(gs) && len(buf)+(gs[b].end-gs[b].start) <= 19; b++ {
			buf = append(buf, text[gs[b].start:gs[b].end]...)
			if len(buf) >= 13 && issuer(string(buf)) && luhn(string(buf)) {
				best = b
			}
		}
		if best < 0 {
			a++
			continue
		}
		out = append(out, PIISpan{utf8.RuneCountInString(text[:gs[a].start]), utf8.RuneCountInString(text[:gs[best].end]), "card"})
		a = best + 1
	}
	return out
}

// issuer: a prefix a card network issues, with that network's length, so a phone or
// order number that happens to pass Luhn is mostly left alone.
func issuer(d string) bool {
	p2 := atoi(d[:2])
	p4 := atoi(d[:4])
	switch {
	case d[0] == '4': // Visa
		return true
	case p2 >= 51 && p2 <= 55, p4 >= 2221 && p4 <= 2720: // Mastercard
		return len(d) == 16
	case p2 == 34 || p2 == 37: // American Express
		return len(d) == 15
	case p4 == 6011 || p2 == 65 || (atoi(d[:3]) >= 644 && atoi(d[:3]) <= 649): // Discover
		return true
	case p2 == 35: // JCB
		return true
	case p2 == 36 || p2 == 38 || p2 == 30: // Diners
		return len(d) >= 14
	}
	return false
}

func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		n = n*10 + int(s[i]-'0')
	}
	return n
}

func luhn(d string) bool {
	sum := 0
	for i := 0; i < len(d); i++ {
		v := int(d[len(d)-1-i] - '0')
		if i%2 == 1 {
			v *= 2
			if v > 9 {
				v -= 9
			}
		}
		sum += v
	}
	return sum%10 == 0
}

// edgeRun is a run of an IBAN's or a card's characters at the start or end of a chunk: the
// rest of a number a chunk boundary cut (Mask masks it in a flagged chunk).
var (
	leadRun  = regexp.MustCompile(`^[A-Z0-9](?:[ -]?[A-Z0-9])*`)
	trailRun = regexp.MustCompile(`[A-Z0-9](?:[ -]?[A-Z0-9])*$`)
)

// MaskPII masks every IBAN and card number in text but its last four characters,
// keeping separators and the text's length in runes. With edges, a run of number
// characters at either end holding four or more digits is masked the same way: in a
// chunk flagged personal_data, that is a number the chunk boundary cut.
func MaskPII(text string, edges bool) string {
	rs := []rune(text)
	for _, s := range PIISpans(text) {
		maskRunes(rs, s.Start, s.End)
	}
	if edges {
		for _, re := range []*regexp.Regexp{leadRun, trailRun} {
			if m := re.FindStringIndex(string(rs)); m != nil && digitsIn(string(rs)[m[0]:m[1]]) >= 4 {
				s := string(rs)
				maskRunes(rs, utf8.RuneCountInString(s[:m[0]]), utf8.RuneCountInString(s[:m[1]]))
			}
		}
	}
	return string(rs)
}

func digitsIn(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			n++
		}
	}
	return n
}

// maskRunes masks the letters and digits of rs[start:end] but the last four.
func maskRunes(rs []rune, start, end int) {
	keep := 4
	for i := end - 1; i >= start; i-- {
		r := rs[i]
		if (r < '0' || r > '9') && (r < 'A' || r > 'Z') {
			continue
		}
		if keep > 0 {
			keep--
			continue
		}
		rs[i] = Mask
	}
}

// Shown is a chunk's text as anyone sees it, a snippet or the model's context: masked
// when the chunk carries FlagPersonal (SB-1904).
func Shown(text string, flags []string) string {
	for _, f := range flags {
		if f == FlagPersonal {
			return MaskPII(text, true)
		}
	}
	return text
}

// personal is [FlagPersonal] for a chunk span that overlaps a number found in the text.
func personal(spans []PIISpan, sp Span) []string {
	for _, s := range spans {
		if s.Start < sp.End && sp.Start < s.End {
			return []string{FlagPersonal}
		}
	}
	return nil
}
