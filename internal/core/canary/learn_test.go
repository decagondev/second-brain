package canary

import (
	"strings"
	"testing"
)

func TestLearnAndScanAll(t *testing.T) {
	var s Scanner
	s.Learn(`{"access_key":"sb-worm","secret_key":"Zq81kT0pWmV3nRx7Lc2"}`)
	s.Learn("-----BEGIN PRIVATE KEY-----\nMC4CAQAwBQYDK2VwBCIEIHk3t9XUy3hbQd0\n-----END PRIVATE KEY-----\n")
	s.Learn("postgres://sb:9fK2mPq7xL@postgres:5432/sb?sslmode=disable")
	s.Learn("abc") // too short to learn
	text := strings.Join([]string{
		"level=info msg=started bucket=sb-worm",                // the access key isn't a secret
		"level=error msg=\"s3 said Zq81kT0pWmV3nRx7Lc2\"",      // the worm secret alone
		"seed MC4CAQAwBQYDK2VwBCIEIHk3t9XUy3hbQd0",             // one line of a key
		"dial postgres://sb:9fK2mPq7xL@postgres:5432/sb?x",     // known and shaped
		"proxy http://olga:unknownpw@proxy:3128",               // shaped only
		"Authorization: Bearer eyJhbGciOiJFZERTQSJ9.e30.sig00", // shaped only
		"-----BEGIN OPENSSH PRIVATE KEY-----",                  // shaped only
		"abc def",
	}, "\n")
	lines := map[int][]string{}
	for _, f := range s.ScanAll("svc.log", text) {
		lines[f.Line] = append(lines[f.Line], f.ID)
	}
	for _, want := range []int{2, 3, 4, 5, 6, 7} {
		if len(lines[want]) == 0 {
			t.Errorf("line %d: not found", want)
		}
	}
	for _, clean := range []int{1, 8} {
		if len(lines[clean]) != 0 {
			t.Errorf("line %d: false finding %v", clean, lines[clean])
		}
	}
	for _, ids := range lines {
		for _, id := range ids {
			if strings.Contains(id, "Zq81") || strings.Contains(id, "9fK2") {
				t.Fatalf("a finding carries the value: %s", id)
			}
		}
	}
}
