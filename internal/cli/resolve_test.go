package cli

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/known"
)

func TestResolveTargetOfflineSkipsLiveLookup(t *testing.T) {
	paths := config.Paths{KnownFile: filepath.Join(t.TempDir(), "missing.json")}
	_, err := resolveTarget(context.Background(), "owner/tool", paths, false)
	if err == nil || !strings.Contains(err.Error(), "offline") {
		t.Fatalf("expected offline resolution error, got %v", err)
	}
	var notFound *TargetNotFoundError
	if !errors.As(err, &notFound) || notFound.Suggest {
		t.Fatalf("expected non-suggestible typed not-found error, got %T: %v", err, err)
	}
}

func TestResolveTargetAcceptsIDAndURL(t *testing.T) {
	paths := config.Paths{KnownFile: filepath.Join(t.TempDir(), "missing.json")}
	for _, input := range []string{"aaa111aaa111", "https://gist.github.com/me/aaa111aaa111"} {
		got, err := ResolveTarget(context.Background(), input, paths, false)
		if err != nil {
			t.Fatalf("resolve %q: %v", input, err)
		}
		if got.GistID != "aaa111aaa111" || got.RequestedFile != "" {
			t.Fatalf("unexpected result for %q: %+v", input, got)
		}
	}
}

func TestResolveTargetPreservesExactFilename(t *testing.T) {
	paths := writeKnown(t, known.Store{Entries: []known.Entry{
		{ID: "aaa111aaa111", Owner: "me", Filenames: []string{"scripts/review.zsh", "scripts/review.py"}},
	}})

	got, err := ResolveTarget(context.Background(), "scripts/review.zsh", paths, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.GistID != "aaa111aaa111" || got.RequestedFile != "scripts/review.zsh" {
		t.Fatalf("unexpected resolved target: %+v", got)
	}

	got, err = ResolveTarget(context.Background(), "review", paths, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.RequestedFile != "" {
		t.Fatalf("stem should not select an entry: %+v", got)
	}
}

func TestResolveTargetAliasTakesPrecedence(t *testing.T) {
	paths := writeKnown(t, known.Store{Entries: []known.Entry{
		{ID: "aaa111aaa111", Alias: "review", Owner: "me", Filenames: []string{"other.txt"}},
		{ID: "bbb222bbb222", Owner: "you", Filenames: []string{"review"}},
	}})

	got, err := ResolveTarget(context.Background(), "review", paths, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.GistID != "aaa111aaa111" || got.RequestedFile != "" {
		t.Fatalf("alias did not take precedence: %+v", got)
	}
}

func TestResolveTargetReportsFilenameAmbiguity(t *testing.T) {
	paths := writeKnown(t, known.Store{Entries: []known.Entry{
		{ID: "aaa111aaa111", Owner: "same", Filenames: []string{"tool.sh"}},
		{ID: "bbb222bbb222", Owner: "same", Filenames: []string{"tool.sh"}},
	}})

	_, err := ResolveTarget(context.Background(), "tool.sh", paths, false)
	var ambiguous *AmbiguousTargetError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("expected ambiguity error, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "aaa111aaa111") || !strings.Contains(err.Error(), "bbb222bbb222") {
		t.Fatalf("expected disambiguating IDs, got %v", err)
	}
}

func TestResolveTargetOwnerQualification(t *testing.T) {
	paths := writeKnown(t, known.Store{Entries: []known.Entry{
		{ID: "aaa111aaa111", Owner: "me", Filenames: []string{"tool.sh"}},
		{ID: "bbb222bbb222", Owner: "you", Filenames: []string{"tool.sh"}},
	}})

	got, err := ResolveTarget(context.Background(), "me/tool.sh", paths, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.GistID != "aaa111aaa111" || got.RequestedFile != "tool.sh" {
		t.Fatalf("unexpected owner-qualified result: %+v", got)
	}
}

func writeKnown(t *testing.T, store known.Store) config.Paths {
	t.Helper()
	paths := config.Paths{KnownFile: filepath.Join(t.TempDir(), "known.json")}
	if err := known.Save(paths.KnownFile, store); err != nil {
		t.Fatal(err)
	}
	return paths
}
