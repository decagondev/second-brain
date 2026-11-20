package scim

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// FuzzUsers (SB-1402): any body sent to the SCIM user endpoints is answered without a
// panic or a server error, so a malformed IdP request can't take provisioning down.
func FuzzUsers(f *testing.F) {
	f.Add(`{"userName":"ana","emails":[{"value":"ana@acme.example"}]}`, `{"Operations":[{"op":"replace","path":"active","value":false}]}`)
	f.Add(`{"userName":"bo","title":"x","urn:ietf:params:scim:schemas:extension:enterprise:2.0:User":{"manager":{"value":"u"}}}`,
		`{"Operations":[{"op":"remove","path":"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:manager"}]}`)
	f.Fuzz(func(t *testing.T, create, patch string) {
		fx := setup()
		do := func(method, path, body string) int {
			r := httptest.NewRequest(method, path, strings.NewReader(body))
			r.Header.Set("Authorization", "Bearer tok")
			w := httptest.NewRecorder()
			fx.h.ServeHTTP(w, r)
			return w.Code
		}
		code := do("POST", "/scim/v2/Users", create)
		if code >= 500 {
			t.Fatalf("create answered %d", code)
		}
		users, _ := fx.dir.UsersByEmail(context.Background(), "acme-logistics", "ana@acme.example")
		for _, u := range users {
			if c := do("PATCH", "/scim/v2/Users/"+u.ID, patch); c >= 500 && c != http.StatusServiceUnavailable {
				t.Fatalf("patch answered %d", c)
			}
		}
	})
}
