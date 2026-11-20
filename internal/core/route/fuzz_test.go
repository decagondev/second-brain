package route

import "testing"

// FuzzMatch (SB-1402): Match never panics and stays fast on hostile patterns; a pattern
// matches itself when it has no wildcards.
func FuzzMatch(f *testing.F) {
	f.Add("/People/**", "/People/Cases/x")
	f.Add("#hr-*", "#hr-cases")
	f.Add("**/**/**/**/**/**/**/**/**/**/**/**/a", "/b/b/b/b/b/b/b/b/b/b/b/b/b/b/b/b/b/b/b")
	f.Fuzz(func(t *testing.T, pattern, location string) {
		if len(pattern) > 256 || len(location) > 512 {
			return
		}
		_ = Match(pattern, location)
		plain := true
		for _, r := range pattern {
			plain = plain && r != '*' && r != '?' && r != '[' && r != 0x5c
		}
		if plain && !Match(pattern, pattern) {
			t.Fatalf("a plain pattern %q doesn't match itself", pattern)
		}
	})
}
