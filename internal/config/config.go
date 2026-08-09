package config

import (
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
