package api

import (
	"testing"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// SB-3803: live tokens of the tenant only, oldest unused first, never the hash.
func TestAdminTokensOrder(t *testing.T) {
	now := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	day := func(d int) time.Time { return now.AddDate(0, 0, d) }
	all := []ports.Session{
		{ID: "tokh_1", Tenant: "t", User: "usr_a", Token: "tok_used_late", SignedIn: day(-30), Validated: day(-1), Expires: day(10)},
		{ID: "tokh_2", Tenant: "t", User: "usr_b", Token: "tok_unused_new", SignedIn: day(-2), Validated: day(-2), Expires: day(10)},
		{ID: "tokh_3", Tenant: "t", User: "usr_c", Token: "tok_used_early", SignedIn: day(-30), Validated: day(-20), Expires: day(10)},
		{ID: "tokh_4", Tenant: "t", User: "usr_d", Token: "tok_unused_old", SignedIn: day(-9), Expires: day(10)},
		{ID: "tokh_5", Tenant: "t", User: "usr_e", Token: "tok_expired", SignedIn: day(-90), Expires: day(-1)},
		{ID: "tokh_6", Tenant: "other", User: "usr_f", Token: "tok_other", SignedIn: day(-9), Expires: day(10)},
		{ID: "ses_7", Tenant: "t", User: "usr_g", SignedIn: day(-1)},
	}
	got := AdminTokens(all, "t", now)
	want := []string{"tok_unused_old", "tok_unused_new", "tok_used_early", "tok_used_late"}
	if len(got) != len(want) {
		t.Fatalf("%+v", got)
	}
	for i, w := range want {
		if got[i].ID != w {
			t.Fatalf("order: %+v", got)
		}
	}
	if got[0].LastUsed != "" || got[3].LastUsed != "2027-02-28" {
		t.Fatalf("last used: %+v", got)
	}
}
