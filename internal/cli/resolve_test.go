package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/leolaurindo/gix/internal/config"
	"github.com/leolaurindo/gix/internal/index"
)

func TestResolveIdentifierPrefersAliasThenID(t *testing.T) {
	paths := config.Paths{IndexFile: filepath.Join(t.TempDir(), "index.json")}

	id, owner, fromIndex, err := resolveIdentifier(context.Background(), "cool", map[string]string{"cool": "alias-id"}, paths, false, false, 1)
	if err != nil {
		t.Fatalf("alias resolution error: %v", err)
	}
	if id != "alias-id" || owner != "" || fromIndex {
		t.Fatalf("unexpected alias resolution: id=%s owner=%s fromIndex=%v", id, owner, fromIndex)
	}

	rawID := "deadbeefcafebabe"
	id, owner, fromIndex, err = resolveIdentifier(context.Background(), rawID, nil, paths, false, false, 1)
	if err != nil {
		t.Fatalf("id resolution error: %v", err)
	}
	if id != rawID || owner != "" || fromIndex {
		t.Fatalf("unexpected id resolution: id=%s owner=%s fromIndex=%v", id, owner, fromIndex)
	}
}

func TestResolveIdentifierUsesIndexAndHandlesAmbiguity(t *testing.T) {
	tmp := t.TempDir()
	idxPath := filepath.Join(tmp, "index.json")
	paths := config.Paths{IndexFile: idxPath, IndexDescFile: filepath.Join(tmp, "index_descriptions.json")}

	idx := index.Index{
		Entries: []index.Entry{
			{ID: "id1", Owner: "alice", Filenames: []string{"main.py"}, Description: "tool one"},
			{ID: "id2", Owner: "bob", Filenames: []string{"tool.sh"}, Description: "tool two"},
			{ID: "id3", Owner: "bob", Filenames: []string{"other.go"}, Description: "other"},
			{ID: "id4", Owner: "charlie", Filenames: []string{"same.sh"}, Description: "dup"},
			{ID: "id5", Owner: "charlie", Filenames: []string{"same.py"}, Description: "dup2"},
		},
	}
	if err := index.Save(idxPath, idx); err != nil {
		t.Fatalf("write index: %v", err)
	}

	// Bare name lookup should hit the index.
	id, owner, fromIndex, err := resolveIdentifier(context.Background(), "tool", nil, paths, false, false, 1)
	if err != nil {
		t.Fatalf("index resolution error: %v", err)
	}
	if !fromIndex || id != "id2" || owner != "bob" {
		t.Fatalf("unexpected index resolution: id=%s owner=%s fromIndex=%v", id, owner, fromIndex)
	}

	// Ambiguous owner/name should error.
	if _, _, _, err := resolveIdentifier(context.Background(), "charlie/same", nil, paths, false, false, 1); err == nil {
		t.Fatalf("expected ambiguity error for charlie/same")
	}

	// Unknown input should error.
	if _, _, _, err := resolveIdentifier(context.Background(), "missing", nil, paths, false, false, 1); err == nil {
		t.Fatalf("expected error for missing identifier")
	}
}

func TestResolveIdentifierUsesDescriptionOverride(t *testing.T) {
	tmp := t.TempDir()
	idxPath := filepath.Join(tmp, "index.json")
	overridePath := filepath.Join(tmp, "index_descriptions.json")
	paths := config.Paths{IndexFile: idxPath, IndexDescFile: overridePath}

	idx := index.Index{
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

	id, owner, fromIndex, err := resolveIdentifier(context.Background(), "new desc", nil, paths, false, true, 1)
	if err != nil {
		t.Fatalf("resolution error: %v", err)
	}
	if !fromIndex || id != "id1" || owner != "alice" {
		t.Fatalf("unexpected resolution: id=%s owner=%s fromIndex=%v", id, owner, fromIndex)
	}
}
