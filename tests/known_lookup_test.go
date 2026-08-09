package tests

import (
	"testing"

	"github.com/leolaurindo/gixt/internal/known"
)

func TestNameMatchesAliasAndFilename(t *testing.T) {
	st := known.Store{Entries: []known.Entry{
		{ID: "id1", Description: "my script", Filenames: []string{"main.py"}, Owner: "me"},
		{ID: "id2", Description: "other", Filenames: []string{"tool.sh"}, Alias: "tool2", Owner: "you"},
	}}

	if got := known.Name(st, "main"); len(got) != 1 || got[0].ID != "id1" {
		t.Fatalf("expected filename basename match to id1, got %+v", got)
	}
	if got := known.Name(st, "main.py"); len(got) != 1 || got[0].ID != "id1" {
		t.Fatalf("expected full filename match to id1, got %+v", got)
	}
	if got := known.Name(st, "tool2"); len(got) != 1 || got[0].ID != "id2" {
		t.Fatalf("expected alias match to id2, got %+v", got)
	}
	if got := known.Name(st, "missing"); len(got) != 0 {
		t.Fatalf("expected no matches, got %+v", got)
	}
}

func TestUpsertReplacesById(t *testing.T) {
	st := known.Store{Entries: []known.Entry{
		{ID: "id1", Description: "old", Filenames: []string{"main.py"}, Pin: "abc", Owner: "me"},
	}}
	known.Upsert(&st, known.Entry{ID: "id1", Description: "new", Filenames: []string{"main.py", "extra.txt"}, Owner: "me"})
	if len(st.Entries) != 1 {
		t.Fatalf("expected single entry after upsert, got %d", len(st.Entries))
	}
	if st.Entries[0].Description != "new" {
		t.Fatalf("expected entry replaced, got %+v", st.Entries[0])
	}
	if st.Entries[0].Pin != "abc" {
		t.Fatalf("expected pin preserved, got %+v", st.Entries[0])
	}

	known.Upsert(&st, known.Entry{ID: "id2", Filenames: []string{"a.py"}, Owner: "you"})
	if len(st.Entries) != 2 {
		t.Fatalf("expected new entry appended, got %d", len(st.Entries))
	}
}
