package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRunManifestDefaultsDetails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gix.json")
	// details omitted, env nil -> should default details and initialize env
	data := `{"run":"echo hi","version":"1.2.3"}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	m, err := LoadRunManifest(path)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if m.Details != DefaultDetails {
		t.Fatalf("expected default details, got %q", m.Details)
	}
	if m.Version != "1.2.3" {
		t.Fatalf("unexpected version: %q", m.Version)
	}
	if m.Env == nil {
		t.Fatalf("env should be initialized")
	}
}
