package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leolaurindo/gix/internal/cache"
	"github.com/leolaurindo/gix/internal/config"
	"github.com/leolaurindo/gix/internal/index"
)

func TestGatherListRowsPrefersDescriptionOverrides(t *testing.T) {
	tmp := t.TempDir()
	idxPath := filepath.Join(tmp, "index.json")
	overridePath := filepath.Join(tmp, "index_descriptions.json")
	cacheDir := filepath.Join(tmp, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}

	idx := index.Index{
		GeneratedAt: time.Now(),
		Entries: []index.Entry{
			{ID: "id1", Owner: "alice", Filenames: []string{"main.py"}, Description: "old"},
		},
	}
	if err := index.Save(idxPath, idx); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(overridePath, []byte(`{"id1":"new desc"}`), 0o644); err != nil {
		t.Fatalf("write overrides: %v", err)
	}

	paths := config.Paths{
		IndexFile:     idxPath,
		IndexDescFile: overridePath,
		CacheDir:      cacheDir,
	}
	rows, err := gatherListRows(paths, map[string][]string{})
	if err != nil {
		t.Fatalf("gather rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Description != "new desc" {
		t.Fatalf("expected override description, got %q", rows[0].Description)
	}

	// also ensure cached manifests pick override
	workDir := filepath.Join(cacheDir, "id1", "sha1")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("mkdir cached workdir: %v", err)
	}
	manifest := cache.Manifest{
		GistID:      "id1",
		SHA:         "sha1",
		Description: "cached desc",
		Owner:       "alice",
		Files:       []string{"main.py"},
		CreatedAt:   time.Now(),
	}
	if err := cache.SaveManifest(cache.ManifestPath(workDir), manifest); err != nil {
		t.Fatalf("save manifest: %v", err)
	}
	rows, err = gatherListRows(paths, map[string][]string{})
	if err != nil {
		t.Fatalf("gather rows with cache: %v", err)
	}
	if rows[0].Description != "new desc" {
		t.Fatalf("expected override description with cache, got %q", rows[0].Description)
	}
}
