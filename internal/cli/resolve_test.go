package cli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/index"
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
	paths := config.Paths{IndexFile: idxPath}

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

func TestResolveIdentifierMatchesFullFilename(t *testing.T) {
	tmp := t.TempDir()
	idxPath := filepath.Join(tmp, "index.json")
	paths := config.Paths{IndexFile: idxPath}

	idx := index.Index{
		Entries: []index.Entry{
			{ID: "id-bat", Owner: "dana", Filenames: []string{"hello-world.bat"}, Description: "bat tool"},
			{ID: "id-js", Owner: "erin", Filenames: []string{"util.js"}, Description: "js util"},
		},
	}
	if err := index.Save(idxPath, idx); err != nil {
		t.Fatalf("write index: %v", err)
	}

	id, owner, fromIndex, err := resolveIdentifier(context.Background(), "hello-world.bat", nil, paths, false, false, 1)
	if err != nil {
		t.Fatalf("full filename resolution error: %v", err)
	}
	if !fromIndex || id != "id-bat" || owner != "dana" {
		t.Fatalf("unexpected resolution for full filename: id=%s owner=%s fromIndex=%v", id, owner, fromIndex)
	}

	id, owner, fromIndex, err = resolveIdentifier(context.Background(), "erin/util.js", nil, paths, false, false, 1)
	if err != nil {
		t.Fatalf("owner/full filename resolution error: %v", err)
	}
	if !fromIndex || id != "id-js" || owner != "erin" {
		t.Fatalf("unexpected resolution for owner/full filename: id=%s owner=%s fromIndex=%v", id, owner, fromIndex)
	}

	// Still resolves without the extension.
	id, owner, fromIndex, err = resolveIdentifier(context.Background(), "util", nil, paths, false, false, 1)
	if err != nil {
		t.Fatalf("basename resolution error: %v", err)
	}
	if !fromIndex || id != "id-js" || owner != "erin" {
		t.Fatalf("unexpected resolution for basename: id=%s owner=%s fromIndex=%v", id, owner, fromIndex)
	}
}
