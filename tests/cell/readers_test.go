package cell

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

// readDigits reads the vision gate's picture the way a model would have to: by its pixels.
// Each digit is a 5x7 grid of 12-pixel cells, two cells apart.
func readDigits(b []byte) string {
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return ""
	}
	n := (img.Bounds().Dx()/12 - 2) / 7
	var out []byte
	for i := 0; i < n; i++ {
		var grid strings.Builder
		for y := 0; y < 7; y++ {
			for x := 0; x < 5; x++ {
				r, _, _, _ := img.At((2+i*7+x)*12+6, (2+y)*12+6).RGBA()
				if r < 0x8000 {
					grid.WriteByte('#')
				} else {
					grid.WriteByte('.')
				}
			}
		}
		for d := 0; d < 10; d++ {
			one, _ := png.Decode(bytes.NewReader(wire.DigitsPNG(string(rune('0' + d)))))
			var g strings.Builder
			for y := 0; y < 7; y++ {
				for x := 0; x < 5; x++ {
					r, _, _, _ := one.At((2+x)*12+6, (2+y)*12+6).RGBA()
					if r < 0x8000 {
						g.WriteByte('#')
					} else {
						g.WriteByte('.')
					}
				}
			}
			if g.String() == grid.String() {
				out = append(out, byte('0'+d))
			}
		}
	}
	return string(out)
}

// SB-6101 through a running cell: a scan isn't read until the evals job gates the vision
// tier, which must read back random digits from a picture; until then it is counted unread
// on health. Once gated, the scan is cited with the mark of what a model read, and its
// page. A vision model that can't read the digits never gets the gate.
func TestScansThroughARunningCell(t *testing.T) {
	canRead := true
	var pictures int
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(b, &req)
		reply := `{"claims":[{"text":"nothing","cites":[]}]}`
		for _, m := range req.Messages {
			var parts []struct {
				Type     string            `json:"type"`
				ImageURL map[string]string `json:"image_url"`
			}
			if json.Unmarshal(m.Content, &parts) != nil {
				continue
			}
			for _, p := range parts {
				if p.Type != "image_url" || req.Model != "vision-1" {
					continue
				}
				pictures++
				raw, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(p.ImageURL["url"], "data:image/png;base64,"))
				if d := readDigits(raw); d != "" && canRead {
					reply = d
				} else {
					reply = "Invoice 4471 from beta-freight. Total due 1,200 by 31 March."
				}
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": reply}}},
			"usage": map[string]int{"prompt_tokens": 10, "completion_tokens": 5}})
	}))
	t.Cleanup(model.Close)
	share := t.TempDir()
	_ = os.MkdirAll(filepath.Join(share, "Ops"), 0o755)
	for p, data := range map[string][]byte{"Ops/invoice.png": []byte("\x89PNG not really"), "Ops/.acl": []byte("read domain:acme.example\n")} {
		if err := os.WriteFile(filepath.Join(share, p), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
`
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, m, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		cfg.Model.Provider, cfg.Model.Base, cfg.Model.ID, cfg.Model.Local = "openaicompat", model.URL, "main-model", true
		cfg.Model.Vision.ID = "vision-1"
		return cfg
	})
	ctx := context.Background()
	r.signIn(t, "sam")
	svc := &ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}
	if fails := svc.SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	if pictures != 0 {
		t.Fatal("an ungated vision tier was sent a picture")
	}
	st, _, _ := r.cell.Syncs.Sync(ctx, "docs")
	if st.Unread["image"] != 1 {
		t.Fatalf("unread: %+v", st.Unread)
	}
	// A vision model that can't read the digits isn't gated.
	canRead = false
	if _, err := r.cell.RunEvals(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wire.VisionGatePath(r.cell.Config.State)); err == nil {
		t.Fatal("a vision model that can't read was gated")
	}
	canRead = true
	if _, err := r.cell.RunEvals(ctx); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(wire.VisionGatePath(r.cell.Config.State)); err != nil || strings.TrimSpace(string(b)) != r.cell.Release {
		t.Fatalf("gate: %q %v", b, err)
	}
	if err := os.WriteFile(filepath.Join(share, "Ops", "invoice.png"), []byte("\x89PNG a new version"), 0o644); err != nil {
		t.Fatal(err)
	}
	if fails := svc.SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	code, body := r.do(t, "GET", "/v1/search?q=invoice+4471+total+due", "")
	if code != 200 || !strings.Contains(body, `"read_by":"vision"`) || !strings.Contains(body, "invoice.png") {
		t.Fatalf("search: %d %s", code, body)
	}
}

// SB-6102 through a running cell: recordings aren't transcribed until an operator's sample
// is heard right by the evals job's gate; then a recording is cited as a transcript a model
// read, with its time; one over model.transcribe.max_minutes is skipped and counted.
func TestRecordingsThroughARunningCell(t *testing.T) {
	calls := 0
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		f, h, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = f.Close()
		text := "The close moves to Tuesday."
		if strings.HasPrefix(h.Filename, "sample") {
			text = "Hello from the sample."
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"duration": 70.0, "segments": []map[string]any{{"start": 65.0, "end": 70.0, "text": text}}})
	}))
	t.Cleanup(whisper.Close)
	share := t.TempDir()
	_ = os.MkdirAll(filepath.Join(share, "Ops"), 0o755)
	for p, data := range map[string][]byte{"Ops/standup.m4a": make([]byte, 4000), "Ops/allhands.mp3": make([]byte, 16000*60*3), "Ops/.acl": []byte("read domain:acme.example\n")} {
		if err := os.WriteFile(filepath.Join(share, p), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
`
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, m, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		cfg.Model.Transcribe.Provider, cfg.Model.Transcribe.Base, cfg.Model.Transcribe.Local, cfg.Model.Transcribe.MaxMinutes = "whispercpp", whisper.URL, true, 2
		return cfg
	})
	ctx := context.Background()
	r.signIn(t, "sam")
	if _, err := r.cell.RunEvals(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wire.TranscribeGatePath(r.cell.Config.State)); err == nil {
		t.Fatal("gated with no sample")
	}
	samples := wire.TranscribeSamples(r.cell.Config.State)
	_ = os.MkdirAll(samples, 0o700)
	_ = os.WriteFile(filepath.Join(samples, "sample.wav"), []byte("RIFF"), 0o600)
	_ = os.WriteFile(filepath.Join(samples, "sample.txt"), []byte("hello from the sample"), 0o600)
	if _, err := r.cell.RunEvals(ctx); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(wire.TranscribeGatePath(r.cell.Config.State)); strings.TrimSpace(string(b)) != r.cell.Release {
		t.Fatalf("not gated: %q", b)
	}
	before := calls
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	if calls != before+1 {
		t.Fatalf("transcribed %d recordings, want 1 (the 3-minute one is over the 2-minute cap)", calls-before)
	}
	st, _, _ := r.cell.Syncs.Sync(ctx, "docs")
	if st.Unread["recording"] != 1 {
		t.Fatalf("unread: %+v", st.Unread)
	}
	code, body := r.do(t, "GET", "/v1/search?q=close+moves+tuesday", "")
	if code != 200 || !strings.Contains(body, `"read_by":"transcribe"`) || !strings.Contains(body, "standup.m4a") {
		t.Fatalf("search: %d %s", code, body)
	}
}
