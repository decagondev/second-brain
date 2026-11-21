package schedule

import (
	"testing"
	"time"
)

func TestDue(t *testing.T) {
	at := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	for _, c := range []struct {
		cron, at string
		want     bool
	}{
		{"* * * * *", "2027-03-10T12:34:56Z", true},
		{"5 * * * *", "2027-03-10T12:05:00Z", true},
		{"5 * * * *", "2027-03-10T12:06:00Z", false},
		{"0 3 * * *", "2027-03-10T03:00:59Z", true},
		{"0 3 * * *", "2027-03-10T04:00:00Z", false},
		{"0 7 * * 1", "2027-03-08T07:00:00Z", true}, // a Monday
		{"0 7 * * 1", "2027-03-09T07:00:00Z", false},
		{"0 4 1 * *", "2027-04-01T04:00:00Z", true},
		{"0 5 1 1,4,7,10 *", "2027-07-01T05:00:00Z", true},
		{"0 5 1 1,4,7,10 *", "2027-08-01T05:00:00Z", false},
		{"*/5 * * * *", "2027-03-10T12:35:00Z", true},
		{"*/5 * * * *", "2027-03-10T12:36:00Z", false},
		{"0 0 1 * 1", "2027-03-08T00:00:00Z", true}, // either day field, as in cron
		{"0 3 * * *", "2027-03-10T05:00:00+02:00", true},
	} {
		got, err := Due(c.cron, at(c.at))
		if err != nil || got != c.want {
			t.Errorf("Due(%q, %s) = %v, %v; want %v", c.cron, c.at, got, err, c.want)
		}
	}
	for _, bad := range []string{"", "* * * *", "60 * * * *", "*/0 * * * *", "a * * * *", "* * 0 * *"} {
		if _, err := Due(bad, time.Now()); err == nil {
			t.Errorf("Due(%q): no error", bad)
		}
	}
	for _, j := range Jobs {
		if _, err := Due(j.Cron, time.Now()); err != nil {
			t.Errorf("job %s: %v", j.Name, err)
		}
	}
}
