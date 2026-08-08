package trust

import (
	"encoding/json"
	"fmt"
	"os"
)

// Entry records the single commit a gist was last approved at. Trust is
// per (gist, commit): re-running the same commit never re-prompts; a new
// commit prompts again.
type Entry struct {
	SHA   string `json:"sha"`
	Owner string `json:"owner,omitempty"`
}

type Store struct {
	Entries map[string]Entry `json:"entries"`
}

func Load(path string) (Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Store{Entries: map[string]Entry{}}, nil
		}
		return Store{}, fmt.Errorf("read trust store: %w", err)
	}
	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return Store{}, fmt.Errorf("parse trust store: %w", err)
	}
	if s.Entries == nil {
		s.Entries = map[string]Entry{}
	}
	return s, nil
}

func Save(path string, s Store) error {
	if s.Entries == nil {
		s.Entries = map[string]Entry{}
	}
	buf, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode trust store: %w", err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return fmt.Errorf("write trust store: %w", err)
	}
	return nil
}

func (s Store) Trusted(id, sha string) bool {
	e, ok := s.Entries[id]
	return ok && e.SHA != "" && e.SHA == sha
}

func (s *Store) Trust(id, sha, owner string) {
	if s.Entries == nil {
		s.Entries = map[string]Entry{}
	}
	s.Entries[id] = Entry{SHA: sha, Owner: owner}
}
