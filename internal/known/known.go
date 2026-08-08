package known

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entry is one remembered gist. Name lookup matches the alias or the gist's
// filenames (basename or full name).
type Entry struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Filenames   []string  `json:"filenames"`
	Alias       string    `json:"alias,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
	Owner       string    `json:"owner"`
}

type Store struct {
	GeneratedAt time.Time `json:"generated_at"`
	Entries     []Entry   `json:"entries"`
}

func Load(path string) (Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Store{}, nil
		}
		return Store{}, fmt.Errorf("read known gists: %w", err)
	}
	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return Store{}, fmt.Errorf("parse known gists: %w", err)
	}
	return s, nil
}

func Save(path string, s Store) error {
	buf, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode known gists: %w", err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return fmt.Errorf("write known gists: %w", err)
	}
	return nil
}

// Name returns entries matching an alias or filename (basename or full name).
func Name(s Store, name string) []Entry {
	target := strings.ToLower(strings.TrimSpace(name))
	if target == "" {
		return nil
	}
	var matches []Entry
	for _, e := range s.Entries {
		if strings.EqualFold(e.Alias, name) {
			matches = append(matches, e)
			continue
		}
		for _, f := range e.Filenames {
			if filenameMatches(target, f) {
				matches = append(matches, e)
				break
			}
		}
	}
	return matches
}

func filenameMatches(targetLower, filename string) bool {
	base := strings.ToLower(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
	full := strings.ToLower(filepath.Base(filename))
	return targetLower == base || targetLower == full
}

// Upsert adds or replaces the entry for a gist ID.
func Upsert(s *Store, e Entry) {
	for i := range s.Entries {
		if s.Entries[i].ID == e.ID {
			s.Entries[i] = e
			return
		}
	}
	s.Entries = append(s.Entries, e)
}

// Sorted returns a copy of the entries ordered by owner, then ID.
func (s Store) Sorted() []Entry {
	out := append([]Entry(nil), s.Entries...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Owner == out[j].Owner {
			return out[i].ID < out[j].ID
		}
		return out[i].Owner < out[j].Owner
	})
	return out
}
