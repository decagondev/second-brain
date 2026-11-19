// Package canary is the CI secret scan (sb-observability(7), Canaries and the secret
// scanner; SB-411): canaries are planted in the vault and a source, the real paths run,
// and every log line, model request and audit export is scanned. A deliberately leaky
// path proves the scanner finds what it should.
package canary

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/alert/webhook"
	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	"example.com/second-brain/internal/adapters/model/modeltest"
	"example.com/second-brain/internal/adapters/model/openaicompat"
	"example.com/second-brain/internal/adapters/secrets/localvault"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/canary"
	"example.com/second-brain/internal/core/gateway"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/secrets"
	"example.com/second-brain/internal/obs"
)

const tenant = "acme-logistics"

type fanout []secrets.Registrar

func (f fanout) Register(v string) {
	for _, r := range f {
		r.Register(v)
	}
}

func TestNoCanaryLeaks(t *testing.T) {
	ctx := context.Background()
	scan := &canary.Scanner{}
	alerts := &webhook.Memory{}
	watch := &canary.Watch{Tenant: tenant, Scanner: scan, Alerts: alerts}
	red := obs.NewRedactor()

	// Plant: one canary in the vault, one in a source document.
	vc, err := canary.New(tenant, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sc, _ := canary.New(tenant, rand.Reader)
	scan.Add(vc)
	scan.Add(sc)
	red.Register(vc.Value)
	red.Register(sc.Value)
	vault := localvault.Dir{Root: t.TempDir()}
	if err := vault.Put("vault://tenants/"+tenant+"/canary", vc.Value); err != nil {
		t.Fatal(err)
	}
	if err := vault.Put("vault://tenants/"+tenant+"/model", "sk-model-key-0123456789"); err != nil {
		t.Fatal(err)
	}
	res := secrets.Resolver{Tenant: tenant, Vault: vault, Redact: fanout{red, scan}}

	// Logs: every line goes through the scanner as it's written, and through the redactor.
	var logs bytes.Buffer
	logger := obs.NewLogger(canary.Writer{W: &logs, Watch: watch, Where: "log"}, red)
	got, err := res.Get(ctx, "vault://tenants/"+tenant+"/canary")
	if err != nil || got != vc.Value {
		t.Fatalf("resolve: %v", err)
	}
	logger.Info("resolved", "ref", "vault://tenants/"+tenant+"/canary", "value", got, "err", fmt.Errorf("401 with token %s", got))

	// Prompts: the source canary arrives inside a document; the gateway refuses to send it.
	fake := modeltest.New("sk-model-key-0123456789", `{"claims":[]}`)
	defer fake.Close()
	key := func(ctx context.Context) (string, error) { return res.Get(ctx, "vault://tenants/"+tenant+"/model") }
	leaked := 0
	gw := gateway.Gateway{Tenant: tenant, Provider: "openaicompat", ModelID: "m", Model: openaicompat.Client{BaseURL: fake.URL, Key: key},
		Secrets: red, OnSecret: func(context.Context) { leaked++ }}
	doc := "<<data cite=\"C1 docs:runbook@3\">>\nRotate the key " + sc.Value + " monthly.\n<</data>>"
	if _, err := gw.Complete(ctx, ports.ModelRequest{System: "answer", Messages: []ports.Message{{Role: "user", Content: doc}}}); err == nil {
		t.Fatal("a prompt holding a canary was sent")
	}
	if _, err := gw.Complete(ctx, ports.ModelRequest{System: "answer", Messages: []ports.Message{{Role: "user", Content: "What is the refund limit?"}}}); err != nil {
		t.Fatal(err)
	}
	if leaked != 1 {
		t.Errorf("OnSecret fired %d times, want 1", leaked)
	}

	// Audit: an error text quoting the canary is redacted before it's hashed.
	log := &audit.Log{Tenant: tenant, Release: func() string { return "rel_ci" }, Store: auditmem.New(), Now: time.Now, Redact: red}
	if _, err := log.Write(ctx, audit.Event{Actor: "sys_executor", Kind: "tool_call", Subject: "tc_1", State: "DidNotRun", Reason: "act failed: 401 token " + vc.Value}); err != nil {
		t.Fatal(err)
	}
	s, _ := audit.NewSigner(tenant, bytes.Repeat([]byte{5}, 32))
	var exp bytes.Buffer
	if _, err := audit.Export(ctx, log.Store, s, 1, time.Now(), &exp); err != nil {
		t.Fatal(err)
	}

	var findings []canary.Finding
	findings = append(findings, scan.Scan("log", logs.String())...)
	// Prompts are the request bodies. The API key legitimately travels in the
	// Authorization header to its own provider, so headers aren't a prompt.
	findings = append(findings, scan.Scan("model-request", strings.Join(fake.Bodies(), "\n"))...)
	findings = append(findings, scan.Scan("audit-export", exp.String())...)
	if len(findings) != 0 || len(alerts.Got) != 0 {
		t.Fatalf("leaks: %+v, alerts %+v", findings, alerts.Got)
	}
	if !strings.Contains(exp.String(), obs.Redacted) || !strings.Contains(logs.String(), obs.Redacted) {
		t.Fatal("the scan proved nothing: no redacted value reached the outputs")
	}
}

// The control: a path that skips the redactor is found, named by id only, and alerted.
func TestScannerFindsALeak(t *testing.T) {
	scan := &canary.Scanner{}
	c, _ := canary.New(tenant, rand.Reader)
	scan.Add(c)
	var hits []canary.Alert
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b bytes.Buffer
		_, _ = b.ReadFrom(r.Body)
		if strings.Contains(b.String(), c.Value) {
			t.Error("the alert carried the canary's value")
		}
		hits = append(hits, canary.Alert{Ref: "seen"})
		w.WriteHeader(204)
	}))
	defer srv.Close()
	watch := &canary.Watch{Tenant: tenant, Scanner: scan, Alerts: webhook.Sender{URL: srv.URL, HTTP: srv.Client()}}
	var out bytes.Buffer
	w := canary.Writer{W: &out, Watch: watch, Where: "log"}
	fmt.Fprintf(w, "line one\nleaky debug: %s\n", c.Value)
	fs := scan.Scan("log", out.String())
	if len(fs) != 1 || fs[0].ID != c.ID || fs[0].Line != 2 || len(hits) != 1 {
		t.Fatalf("findings %+v, alerts %d", fs, len(hits))
	}
	if err := (webhook.Sender{URL: "http://insecure.example/hook"}).Alert(context.Background(), canary.Alert{}); err == nil {
		t.Error("a plain-http alert URL was accepted")
	}
}
