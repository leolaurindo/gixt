package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leolaurindo/gixt/internal/config"
)

func TestLoadSettingsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	s, err := config.LoadSettings(path)
	if err != nil {
		t.Fatalf("LoadSettings unexpected error: %v", err)
	}
	if !s.Mine {
		t.Fatalf("expected mine auto-trust to default true")
	}
}

func TestSaveAndReloadSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	if err := config.SaveSettings(path, config.Settings{Mine: false}); err != nil {
		t.Fatalf("SaveSettings error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("settings file not written: %v", err)
	}

	loaded, err := config.LoadSettings(path)
	if err != nil {
		t.Fatalf("LoadSettings error: %v", err)
	}
	if loaded.Mine {
		t.Fatalf("expected Mine=false after round trip")
	}
}

func TestLegacySettingsDefaultsMineToTrue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"mode":"mine","cache_mode":"never"}`), 0o644); err != nil {
		t.Fatalf("write legacy settings: %v", err)
	}
	s, err := config.LoadSettings(path)
	if err != nil {
		t.Fatalf("LoadSettings error: %v", err)
	}
	if !s.Mine {
		t.Fatalf("expected Mine to default true when the key is absent")
	}
}
