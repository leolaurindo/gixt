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
	AliasFile string
	IndexFile string
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
		AliasFile: filepath.Join(cfgDir, "aliases.json"),
		IndexFile: filepath.Join(cfgDir, "index.json"),
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

type TrustMode string

const (
	TrustNever TrustMode = "never"
	TrustMine  TrustMode = "mine" // trust gists owned by the authenticated user
	TrustAll   TrustMode = "all"
)

type CacheMode string

const (
	CacheModeDefault CacheMode = "never" // default to temp/uvx-like
	CacheModeCache   CacheMode = "cache"
)

type ExecMode string

const (
	ExecModeIsolate ExecMode = "isolate"
	ExecModeCWD     ExecMode = "cwd"
)

type Settings struct {
	Mode          TrustMode       `json:"mode,omitempty"`
	TrustedOwners map[string]bool `json:"trusted_owners,omitempty"`
	TrustedGists  map[string]bool `json:"trusted_gists,omitempty"`
	CacheMode     CacheMode       `json:"cache_mode,omitempty"`
	ExecMode      ExecMode        `json:"exec_mode,omitempty"`
}

func LoadSettings(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Settings{
				Mode:          TrustNever,
				TrustedOwners: map[string]bool{},
				TrustedGists:  map[string]bool{},
				CacheMode:     CacheModeDefault,
			}, nil
		}
		return Settings{}, fmt.Errorf("read settings: %w", err)
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, fmt.Errorf("parse settings: %w", err)
	}
	if s.TrustedOwners == nil {
		s.TrustedOwners = map[string]bool{}
	}
	if s.TrustedGists == nil {
		s.TrustedGists = map[string]bool{}
	}
	if s.Mode == "" {
		s.Mode = TrustNever
	}
	if s.CacheMode == "" {
		s.CacheMode = CacheModeDefault
	}
	if s.ExecMode != "" && s.ExecMode != ExecModeIsolate && s.ExecMode != ExecModeCWD {
		s.ExecMode = ExecModeIsolate
	}
	return s, nil
}

func SaveSettings(path string, s Settings) error {
	if s.TrustedOwners == nil {
		s.TrustedOwners = map[string]bool{}
	}
	if s.TrustedGists == nil {
		s.TrustedGists = map[string]bool{}
	}
	if s.Mode == "" {
		s.Mode = TrustNever
	}
	if s.CacheMode == "" {
		s.CacheMode = CacheModeDefault
	}
	if s.ExecMode != "" && s.ExecMode != ExecModeIsolate && s.ExecMode != ExecModeCWD {
		s.ExecMode = ExecModeIsolate
	}
	buf, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}
