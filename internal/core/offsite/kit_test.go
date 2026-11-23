package offsite

import (
	"bytes"
	"crypto/rand"
	"errors"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/shamir"
)

func TestSharePages(t *testing.T) {
	key, _ := NewKey()
	shares, err := shamir.Split(key, KitShares, KitNeed, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var back []shamir.Share
	for _, sh := range shares[1:4] {
		page := SharePage("kit_0a1b2c3d", "acme-logistics", "2026-11-23", sh)
		kit, tenant, got, err := ParseSharePage(strings.ToLower(page)) // typed back in lower case
		if err != nil || kit != "kit_0a1b2c3d" || tenant != "acme-logistics" || got.X != sh.X || !bytes.Equal(got.Y, sh.Y) {
			t.Fatalf("share %d: %v", sh.X, err)
		}
		back = append(back, got)
	}
	k, err := shamir.Combine(back)
	if err != nil || KeyCheck(k) != KeyCheck(key) {
		t.Fatalf("three pages don't rebuild the key: %v", err)
	}
	// One mistyped character fails the checksum.
	page := SharePage("kit_0a1b2c3d", "acme-logistics", "2026-11-23", shares[0])
	lines := strings.Split(page, "\n")
	for i, l := range lines {
		if isB32Line(l) && strings.TrimSpace(l) != "" {
			c := []byte(l)
			if c[0] == 'A' {
				c[0] = 'B'
			} else {
				c[0] = 'A'
			}
			lines[i] = string(c)
			break
		}
	}
	if _, _, _, err := ParseSharePage(strings.Join(lines, "\n")); !errors.Is(err, ErrShare) || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("mistyped: %v", err)
	}
	if _, _, _, err := ParseSharePage("share: 2\n"); !errors.Is(err, ErrShare) {
		t.Fatalf("incomplete: %v", err)
	}
}
