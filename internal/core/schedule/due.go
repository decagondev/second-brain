package schedule

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// WatchdogEvery is how often the watchdog runs (sb-runtime(7), Schedules).
const WatchdogEvery = 5 * time.Minute

// Due reports whether a five-field cron expression (minute hour day-of-month month
// day-of-week, UTC) fires in the minute that holds t. Each field is *, */N, or a list of
// numbers. When both day fields are restricted, either matching is enough, as in cron.
// A malformed expression is an error, never "due" or "not due".
func Due(cron string, t time.Time) (bool, error) {
	f := strings.Fields(cron)
	if len(f) != 5 {
		return false, fmt.Errorf("schedule: %q is not five cron fields", cron)
	}
	t = t.UTC()
	limits := [5][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 6}}
	vals := [5]int{t.Minute(), t.Hour(), t.Day(), int(t.Month()), int(t.Weekday())}
	var ok [5]bool
	for i := range f {
		m, err := match(f[i], vals[i], limits[i][0], limits[i][1])
		if err != nil {
			return false, fmt.Errorf("schedule: %q field %d: %w", cron, i+1, err)
		}
		ok[i] = m
	}
	day := ok[2] && ok[4]
	if f[2] != "*" && f[4] != "*" {
		day = ok[2] || ok[4]
	}
	return ok[0] && ok[1] && ok[3] && day, nil
}

func match(field string, v, lo, hi int) (bool, error) {
	if field == "*" {
		return true, nil
	}
	if step, found := strings.CutPrefix(field, "*/"); found {
		n, err := strconv.Atoi(step)
		if err != nil || n < 1 || n > hi {
			return false, fmt.Errorf("bad step %q", field)
		}
		return (v-lo)%n == 0, nil
	}
	hit := false
	for _, p := range strings.Split(field, ",") {
		n, err := strconv.Atoi(p)
		if err != nil || n < lo || n > hi {
			return false, fmt.Errorf("bad value %q", p)
		}
		hit = hit || n == v
	}
	return hit, nil
}
