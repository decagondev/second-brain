package ask_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
)

func TestRecordSignsAndCatchesTampering(t *testing.T) {
	sg, err := audit.NewSigner("acme-logistics", bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	r := ask.AgentRecord{Agent: "case-brief", Tenant: "acme-logistics", Release: "rel_7", Owner: "sam@acme-logistics.example"}
	sr, err := ask.SignRecord(r, sg, time.Date(2027, 1, 4, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := json.Marshal(sr)
	if _, err := ask.VerifyRecord(doc, sg.Key()); err != nil {
		t.Fatalf("a fresh record should verify: %v", err)
	}
	other, _ := audit.NewSigner("acme-logistics", bytes.Repeat([]byte{8}, 32))
	if _, err := ask.VerifyRecord(doc, other.Key()); !errors.Is(err, ask.ErrRecord) {
		t.Fatalf("a record against another pinned key should fail: %v", err)
	}
	bad := bytes.Replace(doc, []byte("rel_7"), []byte("rel_8"), 1)
	if _, err := ask.VerifyRecord(bad, ""); !errors.Is(err, ask.ErrRecord) {
		t.Fatalf("an edited record should fail: %v", err)
	}
	sr.SHA256 = ""
	sr.Record.Release = "rel_8"
	resigned, _ := json.Marshal(sr)
	if _, err := ask.VerifyRecord(resigned, ""); !errors.Is(err, ask.ErrRecord) {
		t.Fatalf("a record with a recomputed body but the old signature should fail: %v", err)
	}
}
