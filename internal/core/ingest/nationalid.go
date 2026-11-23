package ingest

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// National identity numbers (sb-ingest(7), Personal data; SB-2207): a UK National
// Insurance number, a US Social Security number, a Dutch citizen service number (BSN) and
// a German tax id (Steuer-IdNr), each in the forms it is written in and only when it keeps
// its format's rules and check digits. A match that runs on into a letter or digit, or
// out of one, isn't one.

var (
	// NINO: two prefix letters, six digits, a suffix A to D; spaced in pairs or not.
	ninoRe = regexp.MustCompile(`[A-Z]{2}(?:[0-9]{6}|(?: [0-9]{2}){3} )[A-D]`)
	// SSN: three, two and four digits, the same separator twice.
	ssnRe = regexp.MustCompile(`[0-9]{3}-[0-9]{2}-[0-9]{4}|[0-9]{3} [0-9]{2} [0-9]{4}`)
	// BSN: nine digits, or 4.2.3 with dots.
	bsnRe = regexp.MustCompile(`[0-9]{4}\.[0-9]{2}\.[0-9]{3}|[0-9]{9}`)
	// Steuer-IdNr: eleven digits, or 2 3 3 3 with spaces.
	steuerRe = regexp.MustCompile(`[0-9]{2} [0-9]{3} [0-9]{3} [0-9]{3}|[0-9]{11}`)
)

func nationalIDs(text string) []PIISpan {
	var out []PIISpan
	for _, d := range []struct {
		kind string
		re   *regexp.Regexp
		ok   func(string) bool
	}{{"nino", ninoRe, validNINO}, {"ssn", ssnRe, validSSN}, {"bsn", bsnRe, validBSN}, {"steuer_id", steuerRe, validSteuerID}} {
		for _, m := range d.re.FindAllStringIndex(text, -1) {
			if !standsAlone(text, m[0], m[1]) || !d.ok(text[m[0]:m[1]]) {
				continue
			}
			out = append(out, PIISpan{utf8.RuneCountInString(text[:m[0]]), utf8.RuneCountInString(text[:m[1]]), d.kind})
		}
	}
	return out
}

// standsAlone: the match isn't part of a longer word or number, nor of a longer run of
// groups ("1234.56.782.1", "123-45-6789-0").
func standsAlone(text string, from, to int) bool {
	if from > 0 && isAlnum(text[from-1]) || to < len(text) && isAlnum(text[to]) {
		return false
	}
	if from > 1 && strings.ContainsRune(" .-", rune(text[from-1])) && text[from-2] >= '0' && text[from-2] <= '9' && text[from-1] != ' ' {
		return false
	}
	if to+1 < len(text) && strings.ContainsRune(".-", rune(text[to])) && text[to+1] >= '0' && text[to+1] <= '9' {
		return false
	}
	return true
}

func digitsOf(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// validNINO keeps the prefix rules: the first letter not D, F, I, Q, U or V, the second
// not D, F, I, O, Q, U or V, and never BG, GB, KN, NK, NT, TN or ZZ.
func validNINO(s string) bool {
	a, b := s[0], s[1]
	if strings.IndexByte("DFIQUV", a) >= 0 || strings.IndexByte("DFIOQUV", b) >= 0 {
		return false
	}
	switch s[:2] {
	case "BG", "GB", "KN", "NK", "NT", "TN", "ZZ":
		return false
	}
	return true
}

// validSSN: the area isn't 000, 666 or 900 to 999, the group isn't 00, the serial isn't 0000.
func validSSN(s string) bool {
	d := digitsOf(s)
	area, group, serial := atoi(d[:3]), atoi(d[3:5]), atoi(d[5:])
	return area != 0 && area != 666 && area < 900 && group != 0 && serial != 0
}

// validBSN is the 11-check: 9·d1 + 8·d2 + … + 2·d8 − d9 is a multiple of 11, not all zero.
func validBSN(s string) bool {
	d := digitsOf(s)
	sum := 0
	for i := 0; i < 8; i++ {
		sum += (9 - i) * int(d[i]-'0')
	}
	sum -= int(d[8] - '0')
	return sum%11 == 0 && strings.Trim(d, "0") != ""
}

// validSteuerID: the first digit isn't 0; in the first ten, one digit appears two or
// three times (three never in a row) and every other at most once; the last is the ISO
// 7064 MOD 11,10 check digit.
func validSteuerID(s string) bool {
	d := digitsOf(s)
	if d[0] == '0' {
		return false
	}
	var count [10]int
	for i := 0; i < 10; i++ {
		count[d[i]-'0']++
	}
	repeated := 0
	for _, c := range count {
		switch {
		case c == 2 || c == 3:
			repeated++
		case c > 3:
			return false
		}
	}
	if repeated != 1 {
		return false
	}
	for i := 0; i+2 < 10; i++ {
		if d[i] == d[i+1] && d[i+1] == d[i+2] {
			return false
		}
	}
	product := 10
	for i := 0; i < 10; i++ {
		sum := (int(d[i]-'0') + product) % 10
		if sum == 0 {
			sum = 10
		}
		product = (sum * 2) % 11
	}
	check := 11 - product
	if check == 10 {
		check = 0
	}
	return check == int(d[10]-'0')
}
