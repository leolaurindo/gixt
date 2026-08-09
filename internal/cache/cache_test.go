package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRevisionAndPruneKeepPinned(t *testing.T) {
	root := t.TempDir()
	for i, sha := range []string{"old", "latest", "pinned"} {
		dir := Dir(root, "gist", sha)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := SaveMeta(MetaPath(dir), Meta{GistID: "gist", SHA: sha, CreatedAt: time.Unix(int64(i), 0)}); err != nil {
			t.Fatal(err)
		}
	}

	if dir, m, ok := Revision(root, "gist", "pinned"); !ok || m.SHA != "pinned" || filepath.Base(dir) != "pinned" {
		t.Fatalf("exact revision not found: %q %+v %v", dir, m, ok)
	}
	if err := Prune(root, "gist", "latest", "pinned"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(Dir(root, "gist", "old")); !os.IsNotExist(err) {
		t.Fatalf("old revision was not pruned: %v", err)
	}
	if _, _, ok := Revision(root, "gist", "pinned"); !ok {
		t.Fatal("pinned revision was pruned")
	}
}
