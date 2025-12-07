package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leolaurindo/gix/internal/alias"
	"github.com/leolaurindo/gix/internal/cache"
	"github.com/leolaurindo/gix/internal/gist"
	"github.com/leolaurindo/gix/internal/index"
	"github.com/leolaurindo/gix/internal/indexdesc"
	"github.com/leolaurindo/gix/internal/runner"
)

func handleDescribe(ctx context.Context, input string) error {
	target := strings.TrimSpace(input)
	if target == "" {
		return errors.New("usage: gix describe <gist-id|url|alias|name|owner/name>")
	}

	paths, err := ensurePaths("")
	if err != nil {
		return err
	}

	overrides, _ := indexdesc.Load(paths.IndexDescFile)
	aliases, _ := alias.Load(paths.AliasFile)
	gistID, owner, _, err := resolveIdentifier(ctx, target, aliases, paths, false, false, normalizeUserPages(0))
	if err != nil {
		return err
	}

	desc := ""
	manifestDetails := ""
	manifestVersion := ""

	// Prefer indexed data when available.
	if idx, err := index.Load(paths.IndexFile); err == nil {
		for _, e := range idx.Entries {
			if e.ID == gistID {
				desc = strings.TrimSpace(e.Description)
				if owner == "" {
					owner = e.Owner
				}
				if v, ok := overrides[e.ID]; ok {
					desc = indexdesc.Normalize(v)
				}
				break
			}
		}
	}
	if desc == "" {
		if v, ok := overrides[gistID]; ok {
			desc = indexdesc.Normalize(v)
		}
	}

	// Fall back to cached manifest.
	if desc == "" || owner == "" || manifestDetails == "" || manifestVersion == "" {
		if m, dir, ok := latestManifest(paths.CacheDir, gistID); ok {
			if desc == "" {
				desc = strings.TrimSpace(m.Description)
			}
			if owner == "" {
				owner = m.Owner
			}
			if manifestDetails == "" || manifestVersion == "" {
				if manifestPath := findManifestFile(dir, m.Files); manifestPath != "" {
					if rm, err := runner.LoadRunManifest(manifestPath); err == nil {
						manifestDetails = rm.Details
						manifestVersion = strings.TrimSpace(rm.Version)
						if desc == "" {
							desc = strings.TrimSpace(rm.Details)
						}
					}
				}
			}
		}
	}

	// Final fallback: live fetch.
	if desc == "" || owner == "" {
		if g, err := gist.Fetch(ctx, gistID, ""); err == nil {
			if desc == "" {
				desc = strings.TrimSpace(g.Description)
			}
			if owner == "" {
				owner = strings.TrimSpace(gist.GuessOwner(g))
			}
		} else {
			return err
		}
	}

	if desc == "" {
		desc = "(no description)"
	}
	if manifestDetails == "" {
		manifestDetails = runner.DefaultDetails
	}

	fmt.Printf("ID: %s\n", gistID)
	if owner != "" {
		fmt.Printf("Owner: %s\n", owner)
	}
	if manifestVersion != "" {
		fmt.Printf("Manifest version: %s\n", manifestVersion)
	}
	if manifestDetails != "" {
		fmt.Printf("Manifest details: %s\n", manifestDetails)
	}
	fmt.Printf("Description: %s\n", desc)
	return nil
}

func latestManifest(cacheDir, gistID string) (cache.Manifest, string, bool) {
	var latest cache.Manifest
	var latestTime time.Time
	var manifestDir string

	root := filepath.Join(cacheDir, gistID)
	entries, err := os.ReadDir(root)
	if err != nil {
		return cache.Manifest{}, "", false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		mpath := cache.ManifestPath(filepath.Join(root, e.Name()))
		m, err := cache.LoadManifest(mpath)
		if err != nil {
			continue
		}
		info, err := os.Stat(mpath)
		if err != nil {
			continue
		}
		if info.ModTime().After(latestTime) {
			latest = m
			latestTime = info.ModTime()
			manifestDir = filepath.Dir(mpath)
		}
	}
	if latest.GistID == "" {
		return cache.Manifest{}, "", false
	}
	return latest, manifestDir, true
}

func findManifestFile(dir string, files []string) string {
	if dir == "" {
		return ""
	}
	candidates := []string{"gix.json", "manifest.json"}
	for _, cand := range candidates {
		path := filepath.Join(dir, cand)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	for _, f := range files {
		if strings.EqualFold(filepath.Base(f), "gix.json") {
			path := filepath.Join(dir, f)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	return ""
}
