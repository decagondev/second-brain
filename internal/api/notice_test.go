package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	tracemem "example.com/second-brain/internal/adapters/traces/memory"
)

func TestNoticeToEveryone(t *testing.T) {
	h, rows := adminHandler(t)
	now := time.Date(2027, 3, 10, 9, 0, 0, 0, time.UTC)
	h.Now = func() time.Time { return now }
	h.Notices = tracemem.New()
	mux := http.NewServeMux()
	h.Register(mux)
	op, staff := []string{"sb-operators"}, []string{"grp_support"}
	until := now.Add(48 * time.Hour).Format(time.RFC3339)
	body := fmt.Sprintf(`{"text":"Finance is read-only & frozen until Friday.","until":%q}`, until)

	if w := adminCall(t, mux, staff, "PUT", "/v1/admin/notice", body); w.Code != 403 {
		t.Fatalf("only admins set it: %d", w.Code)
	}
	if w := adminCall(t, mux, staff, "GET", "/v1/notice", ""); w.Code != 200 || strings.TrimSpace(w.Body.String()) != "{}" {
		t.Fatalf("none yet: %d %s", w.Code, w.Body)
	}
	for _, bad := range []string{
		fmt.Sprintf(`{"text":"see https://x.example","until":%q}`, until),
		fmt.Sprintf(`{"text":"<script>","until":%q}`, until),
		fmt.Sprintf(`{"text":"too long","until":%q}`, now.Add(8*24*time.Hour).Format(time.RFC3339)),
		`{"text":"no end"}`,
	} {
		if w := adminCall(t, mux, op, "PUT", "/v1/admin/notice", bad); w.Code != 400 {
			t.Fatalf("%s: %d", bad, w.Code)
		}
	}
	if w := adminCall(t, mux, op, "PUT", "/v1/admin/notice", body); w.Code != 200 {
		t.Fatalf("set: %d %s", w.Code, w.Body)
	}
	w := adminCall(t, mux, staff, "GET", "/v1/notice", "")
	var got struct {
		Text  string    `json:"text"`
		Until time.Time `json:"until"`
	}
	if json.Unmarshal(w.Body.Bytes(), &got) != nil || got.Text != "Finance is read-only & frozen until Friday." || got.Until.Format(time.RFC3339) != until {
		t.Fatalf("anyone signed in reads it: %s", w.Body)
	}
	now = now.Add(49 * time.Hour)
	if w := adminCall(t, mux, staff, "GET", "/v1/notice", ""); strings.TrimSpace(w.Body.String()) != "{}" {
		t.Fatalf("it ends by itself: %s", w.Body)
	}
	if w := adminCall(t, mux, op, "DELETE", "/v1/admin/notice", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"cleared":true`) {
		t.Fatalf("clear: %d %s", w.Code, w.Body)
	}
	all, _ := rows.Range(context.Background(), 1, 100)
	states := []string{}
	for _, r := range all {
		if r.Kind == "config" && r.Subject == "notice" {
			if r.PayloadSHA256 == "" || strings.Contains(r.Reason, "Finance") {
				t.Fatalf("the row has the text's digest, never the text: %+v", r)
			}
			states = append(states, r.State)
		}
	}
	if strings.Join(states, ",") != "set,cleared" {
		t.Fatalf("audit %v", states)
	}
}
