package cli

import (
	"testing"

	"github.com/leolaurindo/gixt/internal/known"
)

func TestReplaceOwnerPreservesPins(t *testing.T) {
	entries := []known.Entry{
		{ID: "kept", Owner: "me", Pin: "aaa"},
		{ID: "missing", Owner: "me", Pin: "bbb"},
		{ID: "stale", Owner: "me"},
		{ID: "other", Owner: "you"},
	}
	fresh := []known.Entry{{ID: "kept", Owner: "me"}, {ID: "new", Owner: "me"}}

	got := replaceOwner(entries, "me", fresh)
	if len(got) != 4 || got[0].ID != "missing" || got[1].ID != "other" || got[2].Pin != "aaa" || got[3].ID != "new" {
		t.Fatalf("unexpected entries: %+v", got)
	}
}
