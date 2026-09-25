package cli

import (
	"testing"

	"github.com/leolaurindo/gixt/internal/known"
)

func TestEnsureAliasAvailable(t *testing.T) {
	paths := writeKnown(t, known.Store{Entries: []known.Entry{
		{ID: "existing", Alias: "review"},
	}})
	if err := ensureAliasAvailable(paths, "other", "review"); err == nil {
		t.Fatal("expected duplicate alias error")
	}
	if err := ensureAliasAvailable(paths, "existing", "review"); err != nil {
		t.Fatalf("expected same gist alias update to be allowed: %v", err)
	}
}

func TestReplaceOwnerPreservesPinsAndAliases(t *testing.T) {
	entries := []known.Entry{
		{ID: "kept", Owner: "me", Pin: "aaa", Alias: "review"},
		{ID: "missing", Owner: "me", Pin: "bbb"},
		{ID: "stale", Owner: "me"},
		{ID: "other", Owner: "you"},
	}
	fresh := []known.Entry{{ID: "kept", Owner: "me"}, {ID: "new", Owner: "me"}}

	got := replaceOwner(entries, "me", fresh)
	if len(got) != 4 || got[0].ID != "missing" || got[1].ID != "other" || got[2].Pin != "aaa" || got[2].Alias != "review" || got[3].ID != "new" {
		t.Fatalf("unexpected entries: %+v", got)
	}
}
