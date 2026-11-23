package health

import (
	"testing"
	"time"
)

// SB-2906: a doubled week warns; a flat one doesn't; no calls is zero, not a warning.
func TestPromptGrowth(t *testing.T) {
	week := func(n, size int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = size + i%7
		}
		return out
	}
	if p := PromptFrom(week(100, 2000), week(100, 1000)); !p.Warning || p.P95 < 2000 || p.P95Before < 1000 {
		t.Fatalf("doubled: %+v", p)
	}
	if p := PromptFrom(week(100, 1000), week(100, 1000)); p.Warning {
		t.Fatalf("flat: %+v", p)
	}
	if p := PromptFrom(nil, nil); p.Warning || p.P95 != 0 {
		t.Fatalf("none: %+v", p)
	}
	if P95([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 100}) != 19 {
		t.Fatal("nearest rank")
	}
}

// SB-2905: leftovers above zero, or a count missing or too old, warn.
func TestIndexWarning(t *testing.T) {
	now := time.Now()
	if i := IndexFrom(now, 0, 0, true, now); i.Warning {
		t.Fatalf("clean: %+v", i)
	}
	if i := IndexFrom(now, 1, 0, true, now); !i.Warning {
		t.Fatal("a leftover didn't warn")
	}
	if i := IndexFrom(now.Add(-3*24*time.Hour), 0, 0, true, now); !i.Warning {
		t.Fatal("an old count didn't warn")
	}
	if i := IndexFrom(time.Time{}, 0, 0, false, now); !i.Warning || i.At != nil {
		t.Fatal("no count didn't warn")
	}
}
