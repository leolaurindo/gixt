package index

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Entry struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Filenames   []string  `json:"filenames"`
	UpdatedAt   time.Time `json:"updated_at"`
	Owner       string    `json:"owner"`
}

type Index struct {
	GeneratedAt time.Time `json:"generated_at"`
	Entries     []Entry   `json:"entries"`
}

func Load(path string) (Index, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Index{}, nil
	}
	if err != nil {
		return Index{}, fmt.Errorf("read index: %w", err)
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return Index{}, fmt.Errorf("parse index: %w", err)
	}
	return idx, nil
}

func Save(path string, idx Index) error {
	buf, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("encode index: %w", err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	return nil
}

// Lookup matches by filename base (case-insensitive, sans extension), full filename (case-insensitive, with extension), or exact description (case-insensitive).
func Lookup(idx Index, name string) []Entry {
	matches := LookupName(idx, name)
	matches = append(matches, LookupDescription(idx, name)...)
	return matches
}

// LookupName matches by filename base (case-insensitive, sans extension) or the full filename (case-insensitive, with extension).
func LookupName(idx Index, name string) []Entry {
	target := strings.TrimSpace(strings.ToLower(name))
	if target == "" {
		return nil
	}
	var matches []Entry
	for _, e := range idx.Entries {
		found := false
		for _, f := range e.Filenames {
			base := strings.ToLower(strings.TrimSuffix(filepath.Base(f), filepath.Ext(f)))
			full := strings.ToLower(filepath.Base(f))
			if target == base || target == full {
				found = true
				break
			}
		}
		if found {
			matches = append(matches, e)
		}
	}
	return matches
}

// LookupDescription matches by exact description (case-insensitive).
func LookupDescription(idx Index, desc string) []Entry {
	cleaned := strings.TrimSpace(strings.ToLower(desc))
	if cleaned == "" {
		return nil
	}
	var matches []Entry
	for _, e := range idx.Entries {
		if strings.ToLower(e.Description) == cleaned {
			matches = append(matches, e)
		}
	}
	return matches
}
