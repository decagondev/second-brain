package openaicompat

import (
	"context"
	"strings"
	"sync"
	"testing"

	"example.com/second-brain/internal/adapters/model/modeltest"
	"example.com/second-brain/internal/core/ports"
)

// vault stands in for vault://tenants/TENANT/model: one key per tenant, changeable at any time.
type vault struct {
	mu   sync.Mutex
	keys map[string]string
}

func (v *vault) key(tenant string) func(context.Context) (string, error) {
	return func(context.Context) (string, error) {
		v.mu.Lock()
		defer v.mu.Unlock()
		return v.keys[tenant], nil
	}
}

func (v *vault) set(tenant, k string) { v.mu.Lock(); v.keys[tenant] = k; v.mu.Unlock() }

// SB-709: the key is read at every call, so a tenant that puts its own provider key in its
// vault is billed from the next call on, with no deploy; and two cells never share a key.
func TestTenantHeldKeyIsUsedFromTheNextCall(t *testing.T) {
	ctx := context.Background()
	f := modeltest.New("sk-factory-billed-000", "ok")
	defer f.Close()
	v := &vault{keys: map[string]string{"gamma-parts": "sk-factory-billed-000", "acme-logistics": "sk-acme-own-111"}}
	gamma := Client{BaseURL: f.URL + "/v1", Key: v.key("gamma-parts"), HTTP: f.Client()}
	req := ports.ModelRequest{System: "s", Messages: []ports.Message{{Role: "user", Content: "q"}}}
	if _, err := gamma.Complete(ctx, req); err != nil {
		t.Fatal(err)
	}
	v.set("gamma-parts", "sk-gamma-own-222") // the company's own provider account
	f.Key = "sk-gamma-own-222"
	if _, err := gamma.Complete(ctx, req); err != nil {
		t.Fatalf("the new key wasn't used: %v", err)
	}
	hs := f.Headers()
	if got := hs[len(hs)-1].Get("Authorization"); got != "Bearer sk-gamma-own-222" {
		t.Fatalf("second call sent %q", got)
	}
	acme := Client{BaseURL: f.URL + "/v1", Key: v.key("acme-logistics"), HTTP: f.Client()}
	if _, err := acme.Complete(ctx, req); err == nil || strings.Contains(f.Headers()[len(f.Headers())-1].Get("Authorization"), "gamma") {
		t.Fatal("another cell's call used gamma's key")
	}
}
