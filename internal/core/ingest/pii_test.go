package ingest

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPIISpans(t *testing.T) {
	for _, ok := range []string{
		"Pay to DE89 3704 0044 0532 0130 00 by Friday.",
		"IBAN GB82WEST12345698765432.",
		"card 4111 1111 1111 1111 exp 12/29",
		"amex 3782-822463-10005",
		"5555555555554444",
		"Ref 12 4111 1111 1111 1111",
	} {
		if !HasPII(ok) {
			t.Errorf("missed: %q", ok)
		}
	}
	for _, no := range []string{
		"DE89 3704 0044 0532 0130 01", // check digits wrong
		"card 4111 1111 1111 1112",    // Luhn fails
		"order 1234567890123",         // no issuer prefix
		"call +44 20 7946 0958",
		"ISO 27001 and SOC 2",
		"build 41111111111111111111111", // too long a number
	} {
		if HasPII(no) {
			t.Errorf("flagged: %q %v", no, PIISpans(no))
		}
	}
}

func TestMaskKeepsTheLastFour(t *testing.T) {
	in := "Pay DE89 3704 0044 0532 0130 00 or card 4111-1111-1111-1111 today."
	got := MaskPII(in, false)
	if utf8.RuneCountInString(got) != utf8.RuneCountInString(in) {
		t.Fatalf("length changed: %q", got)
	}
	if !strings.Contains(got, "•••• •••• •••• •••• ••30 00") || !strings.Contains(got, "••••-••••-••••-1111") || HasPII(got) {
		t.Fatalf("masked: %q", got)
	}
	if !strings.HasPrefix(got, "Pay ") || !strings.HasSuffix(got, " today.") {
		t.Fatalf("masked too much: %q", got)
	}
	// A chunk boundary cut a number: the flagged chunk's edges are masked too.
	if got := Shown("3704 0044 0532 0130 00 was the account", []string{FlagPersonal}); !strings.HasPrefix(got, "•••• •••• •••• ••30 00 was") {
		t.Fatalf("edge: %q", got)
	}
	if got := Shown("4111 1111 1111 1111", nil); got != "4111 1111 1111 1111" {
		t.Fatalf("an unflagged chunk was masked: %q", got)
	}
}

func FuzzMaskPII(f *testing.F) {
	for _, s := range []string{"DE89 3704 0044 0532 0130 00", "4111111111111111", "GB82WEST12345698765432 x", "", "ü 4111 1111 1111 1111", "AB 12 34 56 C", "123-45-6789", "1234.56.782", "86 095 742 719"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if !utf8.ValidString(s) {
			return
		}
		m := MaskPII(s, true)
		if utf8.RuneCountInString(m) != utf8.RuneCountInString(s) {
			t.Fatalf("length: %q -> %q", s, m)
		}
		if HasPII(m) {
			t.Fatalf("still holds personal data: %q -> %q %v", s, m, PIISpans(m))
		}
	})
}
