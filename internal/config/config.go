package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Paths struct {
	ConfigDir string
	CacheDir  string
	KnownFile string
	AuthFile  string
	TrustFile string
	Settings  string
}

func Discover(cacheOverride string) (Paths, error) {
	cfgRoot, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("detect config dir: %w", err)
	}
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return Paths{}, fmt.Errorf("detect cache dir: %w", err)
	}

	cfgDir := filepath.Join(cfgRoot, "gixt")
	cacheDir := filepath.Join(cacheRoot, "gixt")
	if cacheOverride != "" {
		if filepath.IsAbs(cacheOverride) {
			cacheDir = cacheOverride
		} else {
			cacheDir = filepath.Join(cacheRoot, cacheOverride)
		}
	}

	return Paths{
		ConfigDir: cfgDir,
		CacheDir:  cacheDir,
		KnownFile: filepath.Join(cfgDir, "known.json"),
		AuthFile:  filepath.Join(cfgDir, "auth.json"),
		TrustFile: filepath.Join(cfgDir, "trust.json"),
		Settings:  filepath.Join(cfgDir, "settings.json"),
	}, nil
}

func EnsureDirs(p Paths) error {
	if err := os.MkdirAll(p.ConfigDir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.MkdirAll(p.CacheDir, 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	return nil
}

type Settings struct {
	Mine bool `json:"mine"`
}

func LoadSettings(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Settings{Mine: true}, nil
		}
		return Settings{}, fmt.Errorf("read settings: %w", err)
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, fmt.Errorf("parse settings: %w", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Settings{}, fmt.Errorf("parse settings: %w", err)
	}
	if _, ok := raw["mine"]; !ok {
		s.Mine = true
	}
	return s, nil
}

func SaveSettings(path string, s Settings) error {
	buf, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}
