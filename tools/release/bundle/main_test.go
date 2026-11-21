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
	m, err := build("../../..", []string{"sb=" + art}, "2027-03-02")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Check(); err != nil {
		t.Fatal(err)
	}
	if len(m.Prompts) < 2 || len(m.Tools) != 1 || m.Policies["policy/schema.cedarschema"] == "" || m.Policies["policy/platform/base.cedar"] == "" || m.Model.Model == "" {
		t.Fatalf("%+v", m)
	}
	if _, err := build("../../..", []string{"no-equals"}, "2027-03-02"); err == nil {
		t.Error("a bad --artefact was accepted")
	}
}

// SB-1601: --image records a service image by digest as image/NAME; anything else is refused.
func TestImages(t *testing.T) {
	d := "sha256:" + strings.Repeat("ab", 32)
	m, err := build("../../..", nil, "2027-03-02", "api="+d)
	if err != nil || m.Code["image/api"] != strings.Repeat("ab", 32) {
		t.Fatalf("%v %v", m.Code, err)
	}
	for _, bad := range []string{"api", "api=latest", "=" + d, "api=sha256:abc"} {
		if _, err := build("../../..", nil, "2027-03-02", bad); err == nil {
			t.Errorf("--image %q accepted", bad)
		}
	}
}
