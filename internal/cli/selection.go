package cli

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// SelectEntry deterministically selects one file from a gist. An explicit
// filename must match exactly; otherwise run-style fallback rules apply.
func SelectEntry(files []string, requested string) (string, error) {
	sorted := append([]string(nil), files...)
	sort.Strings(sorted)

	if requested != "" {
		for _, filename := range sorted {
			if filename == requested {
				return filename, nil
			}
		}
		return "", fmt.Errorf("entry %q not found; available files: %s", requested, strings.Join(sorted, ", "))
	}
	if len(sorted) == 0 {
		return "", fmt.Errorf("gist has no files")
	}

	main := filterByPrefix(sorted, "main.")
	if chosen := choosePlatformSpecific(main); chosen != "" {
		return chosen, nil
	}
	if len(main) > 0 {
		return main[0], nil
	}

	index := filterByPrefix(sorted, "index.")
	if chosen := choosePlatformSpecific(index); chosen != "" {
		return chosen, nil
	}
	if len(index) > 0 {
		return index[0], nil
	}

	if chosen := choosePlatformSpecific(sorted); chosen != "" {
		return chosen, nil
	}
	return sorted[0], nil
}

func filterByPrefix(files []string, prefix string) []string {
	var out []string
	for _, filename := range files {
		if strings.HasPrefix(strings.ToLower(filepath.Base(filename)), prefix) {
			out = append(out, filename)
		}
	}
	return out
}

func choosePlatformSpecific(files []string) string {
	if len(files) == 0 {
		return ""
	}
	allowed := platformAllowedExts()
	preferred := platformPreferredExts()

	type info struct {
		variants     int
		allAllowed   bool
		preferredHit []string
	}
	byBase := map[string]*info{}
	var order []string
	for _, filename := range files {
		base := strings.ToLower(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
		ext := strings.ToLower(filepath.Ext(filename))
		if _, ok := byBase[base]; !ok {
			byBase[base] = &info{allAllowed: true}
			order = append(order, base)
		}
		entry := byBase[base]
		entry.variants++
		if !allowed[ext] {
			entry.allAllowed = false
		}
		if preferred[ext] {
			entry.preferredHit = append(entry.preferredHit, filename)
		}
	}
	for _, base := range order {
		entry := byBase[base]
		if entry.variants > 1 && entry.allAllowed && len(entry.preferredHit) == 1 {
			return entry.preferredHit[0]
		}
	}
	return ""
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
