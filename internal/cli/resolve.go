package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leolaurindo/gix/internal/config"
	"github.com/leolaurindo/gix/internal/gist"
	"github.com/leolaurindo/gix/internal/index"
	"github.com/leolaurindo/gix/internal/indexdesc"
)

func resolveIdentifier(ctx context.Context, input string, aliases map[string]string, paths config.Paths, userLookup bool, descLookup bool, userPages int) (string, string, bool, error) {
	if val, ok := aliases[input]; ok {
		return val, "", false, nil
	}

	id := gist.ExtractID(input)
	if gist.IsLikelyGistID(id) {
		return id, "", false, nil
	}

	idx, err := index.Load(paths.IndexFile)
	overrides, _ := indexdesc.Load(paths.IndexDescFile)
	if err == nil && len(idx.Entries) > 0 {
		if strings.Contains(input, "/") && !strings.Contains(input, "://") {
			parts := strings.SplitN(input, "/", 2)
			ownerPart := strings.ToLower(parts[0])
			namePart := strings.ToLower(parts[1])
			var matches []index.Entry
			for _, e := range idx.Entries {
				if v, ok := overrides[e.ID]; ok {
					e.Description = indexdesc.Normalize(v)
				}
				if !strings.EqualFold(e.Owner, ownerPart) {
					continue
				}
				if descLookup && strings.ToLower(strings.TrimSpace(e.Description)) == namePart {
					matches = append(matches, e)
					continue
				}
				for _, f := range e.Filenames {
					base := strings.ToLower(strings.TrimSuffix(filepath.Base(f), filepath.Ext(f)))
					if base == namePart {
						matches = append(matches, e)
						break
					}
				}
			}
			if len(matches) == 1 {
				return matches[0].ID, matches[0].Owner, true, nil
			}
			if len(matches) > 1 {
				return "", "", false, fmt.Errorf("owner/name matches multiple gists for %s: %d candidates", ownerPart, len(matches))
			}
		}

		idx = applyDescriptionOverrides(idx, overrides)
		matches := index.LookupName(idx, input)
		if descLookup {
			matches = append(matches, index.LookupDescription(idx, input)...)
		}
		if len(matches) == 1 {
			return matches[0].ID, matches[0].Owner, true, nil
		}
		if len(matches) > 1 {
			var opts []string
			for _, m := range matches {
				opts = append(opts, fmt.Sprintf("%s (%s)", m.ID, m.Description))
			}
			return "", "", false, fmt.Errorf("friendly name matches multiple gists: %s", strings.Join(opts, "; "))
		}
	}

	if userLookup && strings.Contains(input, "/") && !strings.Contains(input, "://") {
		parts := strings.SplitN(input, "/", 2)
		ownerPart := parts[0]
		namePart := strings.ToLower(parts[1])
		matches, err := findOwnerNameLive(ctx, ownerPart, namePart, userPages, descLookup)
		if err != nil {
			return "", "", false, err
		}
		if len(matches) == 1 {
			return matches[0].ID, matches[0].Owner, false, nil
		}
		if len(matches) > 1 {
			return "", "", false, fmt.Errorf("owner/name matches multiple gists for %s: %d candidates", ownerPart, len(matches))
		}
	}

	if id == "" || !gist.IsLikelyGistID(id) {
		return "", "", false, fmt.Errorf(
			"could not resolve %q as alias, gist id, URL, indexed name, or owner/name (try `gix index-mine`, `gix index-owner`, or `owner/name` with -u)",
			input,
		)
	}

	return id, "", false, nil
}

func findOwnerNameLive(ctx context.Context, owner string, nameLower string, pages int, descLookup bool) ([]index.Entry, error) {
	items, err := gist.ListForOwner(ctx, owner, 100, pages)
	if err != nil {
		return nil, err
	}
	var matches []index.Entry
	for _, it := range items {
		desc := strings.ToLower(strings.TrimSpace(it.Description))
		if descLookup && desc == nameLower {
			matches = append(matches, index.Entry{
				ID:          it.ID,
				Description: it.Description,
				Filenames:   mapFileNames(it.Files),
				UpdatedAt:   it.UpdatedAt,
				Owner:       it.Owner.Login,
			})
			continue
		}
		for fname := range it.Files {
			base := strings.ToLower(strings.TrimSuffix(filepath.Base(fname), filepath.Ext(fname)))
			if base == nameLower {
				matches = append(matches, index.Entry{
					ID:          it.ID,
					Description: it.Description,
					Filenames:   mapFileNames(it.Files),
					UpdatedAt:   it.UpdatedAt,
					Owner:       it.Owner.Login,
				})
				break
			}
		}
	}
	return matches, nil
}

func mapFileNames(m map[string]gist.File) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func normalizeUserPages(val int) int {
	if val <= 0 {
		return 2
	}
	return val
}

func applyDescriptionOverrides(idx index.Index, overrides map[string]string) index.Index {
	if len(overrides) == 0 {
		return idx
	}
	out := index.Index{
		GeneratedAt: idx.GeneratedAt,
		Entries:     make([]index.Entry, 0, len(idx.Entries)),
	}
	for _, e := range idx.Entries {
		if v, ok := overrides[e.ID]; ok {
			e.Description = indexdesc.Normalize(v)
		}
		out.Entries = append(out.Entries, e)
	}
	return out
}
