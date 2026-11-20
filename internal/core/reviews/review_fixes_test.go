package reviews

import (
	"testing"

	"example.com/second-brain/internal/core/manifest"
)

// SB-1401: a removal from a manifest group whose list doesn't hold the person's email is
// an IdP request, never dropped.
func TestRemovalNotInTheListIsAnIdPRequest(t *testing.T) {
	tn := manifest.Tenant{Groups: map[string][]string{"group:finance": {"ana@acme.example"}}}
	r := Review{Items: []Item{{User: "usr_somebody_else", Via: []string{"group:finance"}, Decision: Remove}}}
	_, changed, reqs := Requests(tn, r)
	if changed || len(reqs) != 1 || reqs[0].Kind != "idp" {
		t.Fatalf("changed %v, requests %+v", changed, reqs)
	}
}
