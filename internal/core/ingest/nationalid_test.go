package ingest

import (
	"strings"
	"testing"
)

func TestNationalIDs(t *testing.T) {
	valid := map[string]string{
		"AB123456C": "nino", "JG 10 37 59 A": "nino", "CE 12 34 56 D": "nino",
		"123-45-6789": "ssn", "078 05 1121": "ssn", "899-99-9999": "ssn",
		"123456782": "bsn", "111222333": "bsn", "1234.56.782": "bsn",
		"86095742719": "steuer_id", "47036892816": "steuer_id", "65929970489": "steuer_id", "86 095 742 719": "steuer_id",
	}
	for n, kind := range valid {
		for _, text := range []string{n, "Ref: " + n + ".", "(" + n + ")", "Nr.\t" + n + "\n"} {
			sp := PIISpans(text)
			if len(sp) != 1 || sp[0].Kind != kind || string([]rune(text)[sp[0].Start:sp[0].End]) != n {
				t.Errorf("%q: %v, want one %s", text, sp, kind)
			}
		}
		m := MaskPII("id "+n+" ok", false)
		if !strings.HasSuffix(compactID(m), compactID(n)[len(compactID(n))-4:]+"ok") || strings.Count(m, string(Mask)) != len(compactID(n))-4 {
			t.Errorf("%s masked as %q, want all but the last four", n, m)
		}
	}
	nearMiss := []string{
		// NINO: a forbidden prefix, a forbidden letter, a bad suffix, lower case, mixed spacing, run on.
		"GB123456A", "ZZ123456A", "DA123456A", "AO123456A", "AB123456E", "ab123456c", "AB 123456 C", "XAB123456C", "AB123456CD",
		// SSN: area 000, 666 or 9xx, group 00, serial 0000, no separators, mixed separators, run on.
		"000-12-3456", "666-12-3456", "900-12-3456", "123-00-4567", "123-45-0000", "123 45-6789", "1123-45-6789", "123-45-67890", "123-45-6789-1",
		// BSN: a failed 11-check, all zeros, eight or ten digits, run on.
		"123456789", "000000000", "12345678", "1234567820", "A123456782", "1234.56.782.1",
		// Steuer-IdNr: a wrong check digit, a leading zero, no digit repeated, two repeated, three in a row.
		"86095742718", "06095742719", "12345678903", "11223456789", "11123456789", "86095742719.2",
	}
	for _, n := range nearMiss {
		if sp := PIISpans("x " + n + " y"); len(sp) != 0 {
			t.Errorf("%q isn't a number but was found: %v", n, sp)
		}
	}
}

func compactID(s string) string {
	return strings.NewReplacer(" ", "", "-", "", ".", "", string(Mask), "").Replace(s)
}
