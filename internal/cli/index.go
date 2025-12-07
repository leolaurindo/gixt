package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/leolaurindo/gix/internal/alias"
	"github.com/leolaurindo/gix/internal/cache"
	"github.com/leolaurindo/gix/internal/config"
	"github.com/leolaurindo/gix/internal/gist"
	"github.com/leolaurindo/gix/internal/index"
	"github.com/leolaurindo/gix/internal/indexdesc"
)

type listRow struct {
	ID          string
	Owner       string
	Description string
	Files       []string
	Cached      bool
	Indexed     bool
	Source      string
}

func handleUpdateIndex(ctx context.Context) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}

	current, err := index.Load(paths.IndexFile)
	if err != nil {
		return err
	}
	entries, removed, err := refreshIndexedGists(ctx, current.Entries)
	if err != nil {
		return err
	}

	idx := index.Index{GeneratedAt: time.Now(), Entries: entries}
	if err := index.Save(paths.IndexFile, idx); err != nil {
		return err
	}
	if removed > 0 {
		fmt.Printf("%sremoved %d missing gists from index%s\n", clrWarn, removed, clrReset)
	}
	fmt.Printf("%sstored %d gists in index %s%s\n", clrInfo, len(idx.Entries), paths.IndexFile, clrReset)
	return nil
}

func handleIndexMine(ctx context.Context) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}

	fmt.Println("fetching your gists via gh...")
	mine, err := gist.List(ctx, 100, 5)
	if err != nil {
		return err
	}
	freshEntries := entriesFromList(mine)
	ownerSet := map[string]bool{}
	for _, e := range freshEntries {
		ownerKey := strings.ToLower(strings.TrimSpace(e.Owner))
		if ownerKey != "" {
			ownerSet[ownerKey] = true
		}
	}

	idx, err := index.Load(paths.IndexFile)
	if err != nil {
		return err
	}
	merged := map[string]index.Entry{}
	for _, e := range idx.Entries {
		if ownerSet[strings.ToLower(strings.TrimSpace(e.Owner))] {
			continue
		}
		merged[e.ID] = e
	}
	for _, e := range freshEntries {
		merged[e.ID] = e
	}

	entries := make([]index.Entry, 0, len(merged))
	for _, e := range merged {
		entries = append(entries, e)
	}
	sortIndexEntries(entries)

	out := index.Index{GeneratedAt: time.Now(), Entries: entries}
	if err := index.Save(paths.IndexFile, out); err != nil {
		return err
	}
	fmt.Printf("%sstored %d gists in index %s%s\n", clrInfo, len(out.Entries), paths.IndexFile, clrReset)
	return nil
}

func refreshIndexedGists(ctx context.Context, entries []index.Entry) ([]index.Entry, int, error) {
	if len(entries) == 0 {
		fmt.Println("index is empty; nothing to refresh (add entries via index-mine, index-owner, or register).")
		return nil, 0, nil
	}

	fmt.Println("refreshing indexed gists individually via gh...")
	var refreshed []index.Entry
	missing := 0
	for _, ent := range entries {
		fmt.Printf("  %s\n", ent.ID)
		g, err := gist.Fetch(ctx, ent.ID, "")
		if err != nil {
			if gist.IsNotFound(err) {
				fmt.Printf("%sskip missing gist %s (removed from index)%s\n", clrWarn, ent.ID, clrReset)
				missing++
				continue
			}
			return nil, missing, err
		}
		refreshed = append(refreshed, toIndexEntryFromGist(g))
	}
	// dedupe in case of duplicates
	uniq := map[string]index.Entry{}
	for _, e := range refreshed {
		uniq[e.ID] = e
	}
	deduped := make([]index.Entry, 0, len(uniq))
	for _, e := range uniq {
		deduped = append(deduped, e)
	}
	sortIndexEntries(deduped)
	return deduped, missing, nil
}

func handleCleanCache(cacheDir string) error {
	paths, err := discoverPaths(cacheDir)
	if err != nil {
		return err
	}
	fmt.Printf("removing cache at %s...\n", paths.CacheDir)
	return os.RemoveAll(paths.CacheDir)
}

func handleClearIndex(cacheDir string) error {
	paths, err := discoverPaths(cacheDir)
	if err != nil {
		return err
	}
	if err := os.Remove(paths.IndexFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	fmt.Printf("removed index file %s (cache untouched)\n", paths.IndexFile)
	return nil
}

func handleList(ctx context.Context, cacheOnly bool, mine bool) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	aliasesMap, _ := alias.Load(paths.AliasFile)
	aliasByID := map[string][]string{}
	for name, id := range aliasesMap {
		aliasByID[id] = append(aliasByID[id], name)
	}

	currentUser := ""
	if mine {
		if login, err := gist.CurrentUser(ctx); err == nil {
			currentUser = login
		} else {
			return fmt.Errorf("detect current user for --mine: %w", err)
		}
	}

	rows, err := gatherListRows(paths, aliasByID)
	if err != nil {
		return err
	}

	var filtered []listRow
	for _, r := range rows {
		if cacheOnly && !r.Cached {
			continue
		}
		if currentUser != "" && !strings.EqualFold(r.Owner, currentUser) {
			continue
		}
		filtered = append(filtered, r)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Owner == filtered[j].Owner {
			return filtered[i].Description < filtered[j].Description
		}
		return filtered[i].Owner < filtered[j].Owner
	})

	printListTable(filtered, aliasByID)
	return nil
}

func handleIndexOwner(ctx context.Context, owner string) error {
	if owner == "" {
		return errors.New("usage: gix index-owner --owner <login>")
	}

	paths, err := ensurePaths("")
	if err != nil {
		return err
	}

	fmt.Printf("fetching gists for owner %s via gh...\n", owner)
	items, err := gist.ListForOwner(ctx, owner, 100, 5)
	if err != nil {
		return err
	}

	idx, err := index.Load(paths.IndexFile)
	if err != nil {
		return err
	}
	existing := map[string]bool{}
	for _, e := range idx.Entries {
		existing[e.ID] = true
	}
	for _, it := range items {
		if existing[it.ID] {
			continue
		}
		idx.Entries = append(idx.Entries, toIndexEntry(it))
	}
	idx.GeneratedAt = time.Now()
	if err := index.Save(paths.IndexFile, idx); err != nil {
		return err
	}
	fmt.Printf("indexed %d gists for owner %s (total %d entries)\n", len(items), owner, len(idx.Entries))
	return nil
}

func gatherListRows(paths config.Paths, aliasByID map[string][]string) ([]listRow, error) {
	rows := map[string]listRow{}
	overrides, _ := indexdesc.Load(paths.IndexDescFile)

	idx, _ := index.Load(paths.IndexFile)
	for _, e := range idx.Entries {
		if v, ok := overrides[e.ID]; ok {
			e.Description = indexdesc.Normalize(v)
		}
		rows[e.ID] = listRow{
			ID:          e.ID,
			Owner:       e.Owner,
			Description: strings.TrimSpace(e.Description),
			Files:       append([]string{}, e.Filenames...),
			Cached:      false,
			Indexed:     true,
			Source:      "index",
		}
	}

	rootEntries, _ := os.ReadDir(paths.CacheDir)
	for _, e := range rootEntries {
		if !e.IsDir() {
			continue
		}
		gistID := e.Name()
		shaEntries, err := os.ReadDir(filepath.Join(paths.CacheDir, gistID))
		if err != nil {
			continue
		}
		var latest cache.Manifest
		var latestTime time.Time
		for _, shaDir := range shaEntries {
			if !shaDir.IsDir() {
				continue
			}
			mpath := cache.ManifestPath(filepath.Join(paths.CacheDir, gistID, shaDir.Name()))
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
			continue
		}
		existing, ok := rows[gistID]
		if !ok || !existing.Cached {
			if v, ok := overrides[gistID]; ok {
				latest.Description = indexdesc.Normalize(v)
			}
			rows[gistID] = listRow{
				ID:          latest.GistID,
				Owner:       latest.Owner,
				Description: strings.TrimSpace(latest.Description),
				Files:       append([]string{}, latest.Files...),
				Cached:      true,
				Indexed:     existing.Indexed,
				Source:      sourceLabel(true, existing.Indexed),
			}
		} else {
			existing.Cached = true
			existing.Source = sourceLabel(true, existing.Indexed)
			if len(latest.Files) > 0 {
				existing.Files = append([]string{}, latest.Files...)
			}
			if latest.Description != "" {
				existing.Description = strings.TrimSpace(latest.Description)
			}
			if v, ok := overrides[gistID]; ok {
				existing.Description = indexdesc.Normalize(v)
			}
			if latest.Owner != "" {
				existing.Owner = latest.Owner
			}
			rows[gistID] = existing
		}
	}

	out := make([]listRow, 0, len(rows))
	for _, r := range rows {
		if r.Source == "" {
			r.Source = sourceLabel(r.Cached, r.Indexed)
		}
		out = append(out, r)
	}
	return out, nil
}

func sourceLabel(cached, indexed bool) string {
	switch {
	case cached && indexed:
		return "cache+index"
	case cached:
		return "cache"
	case indexed:
		return "index"
	default:
		return ""
	}
}

func printListTable(rows []listRow, aliasByID map[string][]string) {
	const (
		idMax     = 12
		sourceMax = 12
		ownerMax  = 18
		filesMax  = 24
		aliasMax  = 18
		descMax   = 36
	)

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSource\tOwner\tFiles\tAliases\tDescription")
	fmt.Fprintln(tw, "--\t------\t-----\t-----\t-------\t-----------")
	for _, r := range rows {
		files := strings.Join(r.Files, ",")
		aliases := strings.Join(aliasByID[r.ID], ",")
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			trimCell(cache.Shorten(r.ID), idMax),
			trimCell(r.Source, sourceMax),
			trimCell(r.Owner, ownerMax),
			trimCell(files, filesMax),
			trimCell(aliases, aliasMax),
			trimCell(r.Description, descMax),
		)
	}
	_ = tw.Flush()
}
func trimCell(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	if len(s) <= max-1 {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func entriesFromList(items []gist.ListItem) []index.Entry {
	entries := make([]index.Entry, 0, len(items))
	for _, it := range items {
		entries = append(entries, toIndexEntry(it))
	}
	return entries
}

func toIndexEntry(it gist.ListItem) index.Entry {
	var names []string
	for name := range it.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	return index.Entry{
		ID:          it.ID,
		Description: strings.TrimSpace(it.Description),
		Filenames:   names,
		UpdatedAt:   it.UpdatedAt,
		Owner:       it.Owner.Login,
	}
}

func toIndexEntryFromGist(g gist.Gist) index.Entry {
	var names []string
	for name := range g.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	return index.Entry{
		ID:          g.ID,
		Description: strings.TrimSpace(g.Description),
		Filenames:   names,
		UpdatedAt:   g.UpdatedAt,
		Owner:       strings.TrimSpace(gist.GuessOwner(g)),
	}
}

func sortIndexEntries(entries []index.Entry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Owner == entries[j].Owner {
			return entries[i].Description < entries[j].Description
		}
		return entries[i].Owner < entries[j].Owner
	})
}
