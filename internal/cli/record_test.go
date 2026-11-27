package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/msgraph/graphtest"
	"example.com/second-brain/internal/core/record"
)

// SB-6002: a recording of the Graph fake has every step, keeps the fake's shapes and holds
// none of its values: no name, address, id or text. SB_UPDATE_GRAPH_RECORDING=1 writes it to
// the msgraph package's testdata, which its shape test reads.
func TestRecordGraph(t *testing.T) {
	f := graphtest.Seeded()
	t.Cleanup(f.Server.Close)
	cfg, _, _ := proveFixture(t, f)
	out := filepath.Join(t.TempDir(), "rec.json")
	code, stdout, errOut := run("record", "graph", "--config", cfg, "--out", out, "--person", "sam@acme.example", "--mailbox", "support@acme.example")
	if code != 0 || !strings.Contains(stdout, "6 exchanges") {
		t.Fatalf("record: %d %q %q", code, stdout, errOut)
	}
	b, _ := os.ReadFile(out)
	for _, s := range []string{"sam@", "ann@", "finance@", "acme.example", "close", "Tuesday", "invoice", "please", "g-fin", "lbl-conf", "Finance", "s3cret", "graph-token"} {
		if strings.Contains(string(b), s) {
			t.Fatalf("the recording holds %q", s)
		}
	}
	var rec record.Recording
	if err := json.Unmarshal(b, &rec); err != nil || len(rec.Exchanges) != 6 {
		t.Fatalf("recording: %v %d", err, len(rec.Exchanges))
	}
	if err := record.CheckPeople(b); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("SB_UPDATE_GRAPH_RECORDING") != "" {
		if err := os.WriteFile(filepath.Join("..", "adapters", "msgraph", "testdata", "recording.json"), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if code, _, _ := run("record", "linear", "--config", cfg, "--out", out); code == 0 {
		t.Fatal("an unknown vendor recorded")
	}
}
