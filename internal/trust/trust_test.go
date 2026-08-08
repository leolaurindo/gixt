package trust

import (
	"path/filepath"
	"testing"
)

func TestTrustedPerCommit(t *testing.T) {
	s := Store{Entries: map[string]Entry{}}
	if s.Trusted("gist1", "aaa") {
		t.Fatalf("expected untrusted before first trust")
	}

	s.Trust("gist1", "aaa", "me")
	if !s.Trusted("gist1", "aaa") {
		t.Fatalf("expected the approved commit to be trusted")
	}
	if s.Trusted("gist1", "bbb") {
		t.Fatalf("expected a new commit to require a new prompt")
	}
	if s.Trusted("gist2", "aaa") {
		t.Fatalf("expected other gists to stay untrusted")
	}
}

func TestPersistRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trust.json")

	s := Store{Entries: map[string]Entry{}}
	s.Trust("g1", "abc", "alice")
	if err := Save(path, s); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if !loaded.Trusted("g1", "abc") {
		t.Fatalf("expected trust to survive round trip")
	}
}
