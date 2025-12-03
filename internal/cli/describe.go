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

	aliases, _ := alias.Load(paths.AliasFile)
	gistID, owner, _, err := resolveIdentifier(ctx, target, aliases, paths, false, false, normalizeUserPages(0))
	if err != nil {
		return err
	}

	desc := ""

	// Prefer indexed data when available.
	if idx, err := index.Load(paths.IndexFile); err == nil {
		for _, e := range idx.Entries {
			if e.ID == gistID {
				if desc == "" {
					desc = strings.TrimSpace(e.Description)
				}
				if owner == "" {
					owner = e.Owner
				}
				break
			}
		}
	}

	// Fall back to cached manifest.
	if desc == "" || owner == "" {
		if m, ok := latestManifest(paths.CacheDir, gistID); ok {
			if desc == "" {
				desc = strings.TrimSpace(m.Description)
			}
			if owner == "" {
				owner = m.Owner
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

	fmt.Printf("ID: %s\n", gistID)
	if owner != "" {
		fmt.Printf("Owner: %s\n", owner)
	}
	fmt.Printf("Description: %s\n", desc)
	return nil
}

func latestManifest(cacheDir, gistID string) (cache.Manifest, bool) {
	var latest cache.Manifest
	var latestTime time.Time

	root := filepath.Join(cacheDir, gistID)
	entries, err := os.ReadDir(root)
	if err != nil {
		return cache.Manifest{}, false
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
		}
	}
	if latest.GistID == "" {
		return cache.Manifest{}, false
	}
	return latest, true
}
