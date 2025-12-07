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
	if s.Mode != config.TrustNever {
		t.Fatalf("expected default trust mode %q, got %q", config.TrustNever, s.Mode)
	}
	if s.CacheMode != config.CacheModeDefault {
		t.Fatalf("expected default cache mode %q, got %q", config.CacheModeDefault, s.CacheMode)
	}
	if s.TrustedOwners == nil || s.TrustedGists == nil {
		t.Fatalf("trusted maps should be initialized")
	}
}

func TestSaveAndReloadSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	orig := config.Settings{
		Mode:          config.TrustAll,
		CacheMode:     config.CacheModeCache,
		ExecMode:      "cwd",
		TrustedOwners: map[string]bool{"alice": true},
		TrustedGists:  map[string]bool{"abc123": true},
	}
	if err := config.SaveSettings(path, orig); err != nil {
		t.Fatalf("SaveSettings error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("settings file not written: %v", err)
	}

	loaded, err := config.LoadSettings(path)
	if err != nil {
		t.Fatalf("LoadSettings error: %v", err)
	}
	if loaded.Mode != orig.Mode || loaded.CacheMode != orig.CacheMode || loaded.ExecMode != orig.ExecMode {
		t.Fatalf("loaded settings mismatch: %+v vs %+v", loaded, orig)
	}
	if !loaded.TrustedOwners["alice"] || !loaded.TrustedGists["abc123"] {
		t.Fatalf("trusted entries not round-tripped: %+v", loaded)
	}
}

func TestExecModeInvalidDefaultsToIsolate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	bad := config.Settings{
		Mode:          config.TrustNever,
		CacheMode:     config.CacheModeDefault,
		ExecMode:      "invalid-mode",
		TrustedOwners: map[string]bool{},
		TrustedGists:  map[string]bool{},
	}
	if err := config.SaveSettings(path, bad); err != nil {
		t.Fatalf("SaveSettings error: %v", err)
	}
	loaded, err := config.LoadSettings(path)
	if err != nil {
		t.Fatalf("LoadSettings error: %v", err)
	}
	if loaded.ExecMode != config.ExecModeIsolate {
		t.Fatalf("expected invalid exec mode to normalize to %q, got %q", config.ExecModeIsolate, loaded.ExecMode)
	}
}
