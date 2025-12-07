package tests

import (
	"testing"
	"time"

	"github.com/leolaurindo/gixt/internal/index"
)

func TestLookupMatchesDescriptionAndFilename(t *testing.T) {
	idx := index.Index{
		GeneratedAt: time.Now(),
		Entries: []index.Entry{
			{ID: "id1", Description: "my script", Filenames: []string{"main.py"}, Owner: "me"},
			{ID: "id2", Description: "other", Filenames: []string{"tool.sh"}, Owner: "you"},
		},
	}

	if got := index.Lookup(idx, "my script"); len(got) != 1 || got[0].ID != "id1" {
		t.Fatalf("expected description match to id1, got %+v", got)
	}
	if got := index.Lookup(idx, "tool"); len(got) != 1 || got[0].ID != "id2" {
		t.Fatalf("expected filename basename match to id2, got %+v", got)
	}
	if got := index.Lookup(idx, "missing"); len(got) != 0 {
		t.Fatalf("expected no matches, got %+v", got)
	}
}
