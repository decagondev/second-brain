package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The bundler pins every real input: both prompts, the workflow tool, the platform
// policies and schema, the model and the gate, and the result passes its own check.
func TestBuildFromTheRepository(t *testing.T) {
	art := filepath.Join(t.TempDir(), "sb")
	if err := os.WriteFile(art, []byte("binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := buildWith("../../..", []string{"sb=" + art}, "2027-03-02", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Check(); err != nil {
		t.Fatal(err)
	}
	if len(m.Prompts) < 2 || len(m.Tools) != 2 || m.Tools[0].Name != "note-publish" || m.Policies["policy/schema.cedarschema"] == "" || m.Policies["policy/platform/base.cedar"] == "" || m.Model.Model == "" {
		t.Fatalf("%+v", m)
	}
	if _, err := buildWith("../../..", []string{"no-equals"}, "2027-03-02", ""); err == nil {
		t.Error("a bad --artefact was accepted")
	}
}

// SB-1601: --image records a service image by digest as image/NAME; anything else is refused.
func TestImages(t *testing.T) {
	d := "sha256:" + strings.Repeat("ab", 32)
	m, err := buildWith("../../..", nil, "2027-03-02", "", "api="+d)
	if err != nil || m.Code["image/api"] != strings.Repeat("ab", 32) {
		t.Fatalf("%v %v", m.Code, err)
	}
	for _, bad := range []string{"api", "api=latest", "=" + d, "api=sha256:abc"} {
		if _, err := buildWith("../../..", nil, "2027-03-02", "", bad); err == nil {
			t.Errorf("--image %q accepted", bad)
		}
	}
}

// SB-1710: a model on the box pins its weights and embedder; a malformed pin is refused.
func TestModelWeights(t *testing.T) {
	dir := t.TempDir()
	w := dir + "/model.json"
	h := strings.Repeat("a", 64)
	if err := os.WriteFile(w, []byte(`{"model":"qwen2.5-7b-instruct-q4","weights_sha256":"`+h+`","embedder":"nomic-embed@sha256:`+h+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := buildWith("../../..", []string{"sb=" + w}, "2027-03-02", w)
	if err != nil || m.Model.WeightsSHA256 != h || m.Model.Provider != "openaicompat" {
		t.Fatalf("%+v %v", m.Model, err)
	}
	plain, _ := buildWith("../../..", []string{"sb=" + w}, "2027-03-02", "")
	if plain.ID == m.ID {
		t.Fatal("pinning weights didn't change the release id")
	}
	_ = os.WriteFile(w, []byte(`{"model":"x","weights_sha256":"nothex"}`), 0o600)
	if _, err := buildWith("../../..", []string{"sb=" + w}, "2027-03-02", w); err == nil {
		t.Fatal("a malformed weights pin was accepted")
	}
}
