package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/gist"
	"github.com/leolaurindo/gixt/internal/index"
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
	if err == nil && len(idx.Entries) > 0 {
		if strings.Contains(input, "/") && !strings.Contains(input, "://") {
			parts := strings.SplitN(input, "/", 2)
			ownerPart := strings.ToLower(parts[0])
			namePart := strings.ToLower(parts[1])
			var matches []index.Entry
			for _, e := range idx.Entries {
				if !strings.EqualFold(e.Owner, ownerPart) {
					continue
				}
				if descLookup && strings.ToLower(strings.TrimSpace(e.Description)) == namePart {
					matches = append(matches, e)
					continue
				}
				for _, f := range e.Filenames {
					if filenameMatches(namePart, f) {
						matches = append(matches, e)
						break
					}
				}
			}
			matches = preferPlatform(matches, namePart)
			if len(matches) == 1 {
				return matches[0].ID, matches[0].Owner, true, nil
			}
			if len(matches) > 1 {
				return "", "", false, fmt.Errorf("owner/name matches multiple gists for %s: %d candidates (try owner/fullname.ext or add an alias)", ownerPart, len(matches))
			}
		}

		matches := index.LookupName(idx, input)
		if descLookup {
			matches = append(matches, index.LookupDescription(idx, input)...)
		}
		matches = preferPlatform(matches, strings.ToLower(strings.TrimSpace(input)))
		if len(matches) == 1 {
			return matches[0].ID, matches[0].Owner, true, nil
		}
		if len(matches) > 1 {
			var opts []string
			for _, m := range matches {
				opts = append(opts, fmt.Sprintf("%s (%s)", m.ID, m.Description))
			}
			return "", "", false, fmt.Errorf("friendly name matches multiple gists: %s (disambiguate with owner/name, full filename like name.ext, or an alias)", strings.Join(opts, "; "))
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
		matches = preferPlatform(matches, namePart)
		if len(matches) == 1 {
			return matches[0].ID, matches[0].Owner, false, nil
		}
		if len(matches) > 1 {
			return "", "", false, fmt.Errorf("owner/name matches multiple gists for %s: %d candidates (try owner/fullname.ext or add an alias)", ownerPart, len(matches))
		}
	}

	if id == "" || !gist.IsLikelyGistID(id) {
		return "", "", false, fmt.Errorf(
			"could not resolve %q as alias, gist id, URL, indexed name, or owner/name (try `gixt index-mine`, `gixt index-owner`, or `owner/name` with -u)",
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
			if filenameMatches(nameLower, fname) {
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

func filenameMatches(targetLower string, filename string) bool {
	base := strings.ToLower(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
	full := strings.ToLower(filepath.Base(filename))
	return targetLower == base || targetLower == full
}

func preferPlatform(matches []index.Entry, targetLower string) []index.Entry {
	if len(matches) <= 1 {
		return matches
	}
	allowed := platformAllowedExts()
	preferred := platformPreferredExts()

	type candidate struct {
		entry index.Entry
		exts  []string
	}
	var candidates []candidate
	for _, e := range matches {
		var matchedExts []string
		for _, f := range e.Filenames {
			if filenameMatches(targetLower, f) {
				matchedExts = append(matchedExts, strings.ToLower(filepath.Ext(f)))
			}
		}
		if len(matchedExts) == 0 {
			continue
		}
		candidates = append(candidates, candidate{entry: e, exts: matchedExts})
	}

	// If any candidate matches a non-platform extension, skip preference and keep ambiguity.
	for _, c := range candidates {
		for _, ext := range c.exts {
			if !allowed[ext] {
				return matches
			}
		}
	}

	var preferredEntries []index.Entry
	seen := map[string]bool{}
	for _, c := range candidates {
		for _, ext := range c.exts {
			if preferred[ext] {
				if !seen[c.entry.ID] {
					preferredEntries = append(preferredEntries, c.entry)
					seen[c.entry.ID] = true
				}
				break
			}
		}
	}
	if len(preferredEntries) == 1 {
		return preferredEntries
	}
	return matches
}

func platformAllowedExts() map[string]bool {
	return map[string]bool{
		".bat":  true,
		".cmd":  true,
		".ps1":  true,
		".sh":   true,
		".bash": true,
		".zsh":  true,
	}
}

func platformPreferredExts() map[string]bool {
	if runtime.GOOS == "windows" {
		return map[string]bool{
			".bat": true,
			".cmd": true,
			".ps1": true,
		}
	}
	return map[string]bool{
		".sh":   true,
		".bash": true,
		".zsh":  true,
	}
}
