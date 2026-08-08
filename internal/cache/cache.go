package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

type Meta struct {
	GistID      string    `json:"gist_id"`
	SHA         string    `json:"sha"`
	Description string    `json:"description"`
	Owner       string    `json:"owner"`
	Files       []string  `json:"files"`
	Source      string    `json:"source_url,omitempty"`
	Etag        string    `json:"etag,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

var cleaner = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func Dir(cacheRoot, gistID, sha string) string {
	cleanID := cleaner.ReplaceAllString(gistID, "-")
	cleanSHA := cleaner.ReplaceAllString(sha, "-")
	return filepath.Join(cacheRoot, cleanID, cleanSHA)
}

func MetaPath(cacheDir string) string {
	return filepath.Join(cacheDir, "meta.json")
}

func LoadMeta(path string) (Meta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Meta{}, err
	}
	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		return Meta{}, fmt.Errorf("parse meta: %w", err)
	}
	return m, nil
}

func SaveMeta(path string, m Meta) error {
	buf, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encode meta: %w", err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return fmt.Errorf("write meta: %w", err)
	}
	return nil
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func PresentFiles(dir string, files []string) bool {
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			return false
		}
	}
	return true
}

func Shorten(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func JoinPath(base string, elems ...string) string {
	return filepath.Join(append([]string{base}, elems...)...)
}

// Latest returns the most recently cached (dir, meta) for a gist.
func Latest(cacheRoot, gistID string) (string, Meta, bool) {
	root := filepath.Join(cacheRoot, cleaner.ReplaceAllString(gistID, "-"))
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", Meta{}, false
	}
	var best string
	var bestMan Meta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m, err := LoadMeta(MetaPath(filepath.Join(root, e.Name())))
		if err != nil {
			continue
		}
		if best == "" || m.CreatedAt.After(bestMan.CreatedAt) {
			best = filepath.Join(root, e.Name())
			bestMan = m
		}
	}
	if best == "" {
		return "", Meta{}, false
	}
	return best, bestMan, true
}

// Prune removes every cached revision of a gist except keepSHA.
func Prune(cacheRoot, gistID, keepSHA string) error {
	root := filepath.Join(cacheRoot, cleaner.ReplaceAllString(gistID, "-"))
	keep := cleaner.ReplaceAllString(keepSHA, "-")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() && e.Name() != keep {
			if err := os.RemoveAll(filepath.Join(root, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
