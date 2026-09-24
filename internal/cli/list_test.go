package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/leolaurindo/gixt/internal/known"
)

func TestListHasNoMutatingSubcommands(t *testing.T) {
	cmd := newListCmd()
	for _, sub := range cmd.Commands() {
		if sub.Name() == "clear" || sub.Name() == "refresh" {
			t.Fatalf("list still has mutating subcommand %q", sub.Name())
		}
	}
}

func TestRenderListTable(t *testing.T) {
	entries := []known.Entry{
		{ID: "1234567890abcdef", Alias: "review-prompt", Owner: "leo", Description: "Review PRs for correctness"},
		{ID: "abcdef0123456789", Filenames: []string{"tools.sh"}, Owner: "sam"},
	}
	var out bytes.Buffer
	if err := renderList(&out, entries, false, false, 100); err != nil {
		t.Fatal(err)
	}
	want := "┌───────────────┬───────┬──────────────────┐\n│ NAME          │ OWNER │ ID               │\n├───────────────┼───────┼──────────────────┤\n│ review-prompt │ @leo  │ 1234567890abcdef │\n│ tools.sh      │ @sam  │ abcdef0123456789 │\n└───────────────┴───────┴──────────────────┘\n"
	if out.String() != want {
		t.Fatalf("renderList() = %q, want %q", out.String(), want)
	}
	if strings.Contains(out.String(), "\033[") {
		t.Fatalf("plain output contains ANSI escapes: %q", out.String())
	}
}

func TestRenderListColor(t *testing.T) {
	var out bytes.Buffer
	if err := renderList(&out, []known.Entry{{ID: "12345678", Alias: "prompt"}}, true, false, 100); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "\033[36mprompt\033[0m") {
		t.Fatalf("colored output missing styled name: %q", out.String())
	}
}

func TestRenderListVerboseShowsDescription(t *testing.T) {
	entries := []known.Entry{{ID: "12345678", Alias: "prompt", Description: "Review pull requests\ncarefully"}}
	var out bytes.Buffer
	if err := renderList(&out, entries, false, true, 100); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "DESCRIPTION") || !strings.Contains(out.String(), "Review pull requests carefully") {
		t.Fatalf("verbose output missing description: %q", out.String())
	}
}

func TestRenderListWrapsVerboseDescription(t *testing.T) {
	entry := known.Entry{
		ID:          "1234567890abcdef1234567890abcdef",
		Alias:       "review-prompt",
		Owner:       "leolaurindo",
		Description: "A deliberately long description that should wrap cleanly across the wide column",
	}
	var out bytes.Buffer
	if err := renderList(&out, []known.Entry{entry}, false, true, 90); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "A deliberately long") || !strings.Contains(out.String(), "description that") {
		t.Fatalf("description was not wrapped at word boundaries: %q", out.String())
	}
	for _, line := range strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n") {
		if width := len([]rune(line)); width > 90 {
			t.Fatalf("table line is %d columns wide, want at most 90: %q", width, line)
		}
	}
}

func TestListHasVerboseFlag(t *testing.T) {
	if newListCmd().Flags().Lookup("verbose") == nil {
		t.Fatal("list command has no --verbose flag")
	}
}
