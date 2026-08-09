package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/gist"
	"github.com/leolaurindo/gixt/internal/known"
)

// resolveTarget maps a user input to a gist ID: direct gist ID/URL, a known
// name/alias, or owner/gist (from the known store, falling back to a live
// lookup).
func resolveTarget(ctx context.Context, input string, paths config.Paths) (string, error) {
	id := gist.ExtractID(input)
	if gist.IsLikelyGistID(id) {
		return id, nil
	}

	st, err := known.Load(paths.KnownFile)
	if err != nil {
		return "", err
	}

	if strings.Contains(input, "/") && !strings.Contains(input, "://") {
		parts := strings.SplitN(input, "/", 2)
		ownerPart := strings.ToLower(parts[0])
		namePart := strings.ToLower(parts[1])

		var matches []known.Entry
		for _, e := range st.Entries {
			if strings.EqualFold(e.Owner, ownerPart) && entryMatchesName(e, namePart) {
				matches = append(matches, e)
			}
		}
		if len(matches) == 1 {
			return matches[0].ID, nil
		}
		if len(matches) > 1 {
			return "", fmt.Errorf("owner/name %s matches multiple known gists", input)
		}

		live, err := findOwnerNameLive(ctx, parts[0], namePart, paths)
		if err != nil {
			return "", err
		}
		if len(live) == 1 {
			return live[0].ID, nil
		}
		if len(live) > 1 {
			return "", fmt.Errorf("owner/name %s matches multiple gists", input)
		}
		return "", fmt.Errorf("could not find %q among %s's gists", parts[1], parts[0])
	}

	matches := known.Name(st, input)
	matches = preferPlatform(matches, strings.ToLower(input))
	if len(matches) == 1 {
		return matches[0].ID, nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("name %q matches multiple known gists (use owner/name or --as)", input)
	}
	return "", fmt.Errorf("could not resolve %q as a gist id, URL, owner/gist, or known name (run `gixt add <target> --as <name>` to remember it)", input)
}

func entryMatchesName(e known.Entry, targetLower string) bool {
	if strings.EqualFold(e.Alias, targetLower) {
		return true
	}
	for _, f := range e.Filenames {
		if filenameMatches(targetLower, f) {
			return true
		}
	}
	return false
}

// findOwnerNameLive resolves owner/name against GitHub live.
func findOwnerNameLive(ctx context.Context, owner, nameLower string, paths config.Paths) ([]known.Entry, error) {
	client := gist.New(loadToken(paths.AuthFile))
	items, err := client.ListForOwner(ctx, owner, 100, 5)
	if err != nil {
		return nil, err
	}
	var matches []known.Entry
	for _, it := range items {
		if entryMatchesName(toKnownEntryFromList(it), nameLower) {
			matches = append(matches, toKnownEntryFromList(it))
		}
	}
	return matches, nil
}

func filenameMatches(targetLower, filename string) bool {
	base := strings.ToLower(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
	full := strings.ToLower(filepath.Base(filename))
	return targetLower == base || targetLower == full
}

func preferPlatform(matches []known.Entry, targetLower string) []known.Entry {
	if len(matches) <= 1 {
		return matches
	}
	allowed := platformAllowedExts()
	preferred := platformPreferredExts()

	type candidate struct {
		entry known.Entry
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

	for _, c := range candidates {
		for _, ext := range c.exts {
			if !allowed[ext] {
				return matches
			}
		}
	}

	seen := map[string]bool{}
	var preferredEntries []known.Entry
	for _, c := range candidates {
		for _, ext := range c.exts {
			if preferred[ext] && !seen[c.entry.ID] {
				preferredEntries = append(preferredEntries, c.entry)
				seen[c.entry.ID] = true
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
