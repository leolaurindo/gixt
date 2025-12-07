package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leolaurindo/gixt/internal/alias"
	"github.com/leolaurindo/gixt/internal/cache"
	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/gist"
	"github.com/leolaurindo/gixt/internal/index"
)

func handleRemove(ctx context.Context, cacheList, indexList, bothList, owners []string, cacheOverride string) error {
	if len(cacheList)+len(indexList)+len(bothList)+len(owners) == 0 {
		return errors.New("usage: gixt remove [--cache id/name ...] [--index id/name ...] [--cache-index id/name ...] [--owner owner ...]")
	}

	paths, err := discoverPaths(cacheOverride)
	if err != nil {
		return err
	}
	aliases, _ := alias.Load(paths.AliasFile)
	idx, _ := index.Load(paths.IndexFile)

	cacheIDs, err := resolveTargets(ctx, cacheList, aliases, paths, idx)
	if err != nil {
		return err
	}
	indexIDs, err := resolveTargets(ctx, indexList, aliases, paths, idx)
	if err != nil {
		return err
	}
	bothIDs, err := resolveTargets(ctx, bothList, aliases, paths, idx)
	if err != nil {
		return err
	}

	ownerKeys := normalizeOwners(owners)

	if len(indexIDs) > 0 || len(bothIDs) > 0 || len(ownerKeys) > 0 {
		if err := removeFromIndex(paths, idx, append(indexIDs, bothIDs...), ownerKeys); err != nil {
			return err
		}
	}
	if len(cacheIDs) > 0 || len(bothIDs) > 0 || len(ownerKeys) > 0 {
		if err := removeFromCache(paths, append(cacheIDs, bothIDs...), ownerKeys); err != nil {
			return err
		}
	}

	return nil
}

func resolveTargets(ctx context.Context, items []string, aliases map[string]string, paths config.Paths, idx index.Index) ([]string, error) {
	var out []string
	for _, it := range items {
		id := resolveAliasTarget(it, idx)
		if id == "" {
			resolved, _, _, err := resolveIdentifier(ctx, it, aliases, paths, false, true, normalizeUserPages(0))
			if err != nil {
				return nil, err
			}
			id = resolved
		}
		out = append(out, gist.ExtractID(id))
	}
	return out, nil
}

func removeFromIndex(paths config.Paths, idx index.Index, ids []string, owners []string) error {
	idSet := makeIDSet(ids)
	orig := len(idx.Entries)
	filtered := make([]index.Entry, 0, len(idx.Entries))
	for _, e := range idx.Entries {
		if idSet[strings.ToLower(strings.TrimSpace(e.ID))] {
			continue
		}
		if ownerMatch(owners, e.Owner) {
			continue
		}
		filtered = append(filtered, e)
	}
	if len(filtered) == orig && len(ids)+len(owners) > 0 {
		fmt.Println("no matching entries removed from index")
	} else if len(filtered) != orig {
		idx.Entries = filtered
		if err := index.Save(paths.IndexFile, idx); err != nil {
			return fmt.Errorf("save index: %w", err)
		}
		fmt.Printf("removed %d entries from index\n", orig-len(filtered))
	}

	return nil
}

func removeFromCache(paths config.Paths, ids []string, owners []string) error {
	idSet := makeIDSet(ids)
	rootEntries, _ := os.ReadDir(paths.CacheDir)
	if len(rootEntries) == 0 && len(ids) == 0 && len(owners) == 0 {
		return nil
	}

	removed := 0
	for _, entry := range rootEntries {
		if !entry.IsDir() {
			continue
		}
		gistID := entry.Name()
		if idSet[strings.ToLower(strings.TrimSpace(gistID))] {
			if err := os.RemoveAll(filepath.Join(paths.CacheDir, gistID)); err != nil {
				return fmt.Errorf("remove cache for %s: %w", gistID, err)
			}
			removed++
			continue
		}
		if len(owners) == 0 {
			continue
		}
		owner := ownerFromCache(paths.CacheDir, gistID)
		if ownerMatch(owners, owner) {
			if err := os.RemoveAll(filepath.Join(paths.CacheDir, gistID)); err != nil {
				return fmt.Errorf("remove cache for %s: %w", gistID, err)
			}
			removed++
		}
	}
	if removed == 0 && (len(ids)+len(owners) > 0) {
		fmt.Println("no matching cache entries removed")
	} else if removed > 0 {
		fmt.Printf("removed cache for %d gist(s)\n", removed)
	}
	return nil
}

func ownerFromCache(cacheRoot, gistID string) string {
	shaDirs, err := os.ReadDir(filepath.Join(cacheRoot, gistID))
	if err != nil {
		return ""
	}
	var latest cache.Manifest
	var latestTime time.Time
	for _, shaDir := range shaDirs {
		if !shaDir.IsDir() {
			continue
		}
		mpath := cache.ManifestPath(filepath.Join(cacheRoot, gistID, shaDir.Name()))
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
	return latest.Owner
}

func ownerMatch(owners []string, owner string) bool {
	for _, o := range owners {
		if strings.EqualFold(strings.TrimSpace(o), strings.TrimSpace(owner)) {
			return true
		}
	}
	return false
}

func normalizeOwners(values []string) []string {
	var out []string
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			continue
		}
		out = append(out, strings.TrimSpace(v))
	}
	return out
}

func makeIDSet(ids []string) map[string]bool {
	set := map[string]bool{}
	for _, v := range ids {
		key := strings.ToLower(strings.TrimSpace(v))
		if key != "" {
			set[key] = true
		}
	}
	return set
}
