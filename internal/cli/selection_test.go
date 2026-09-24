package cli

import (
	"runtime"
	"strings"
	"testing"
)

func TestSelectEntryExplicitAndDeterministic(t *testing.T) {
	got, err := SelectEntry([]string{"z.txt", "main.py", "a.txt"}, "main.py")
	if err != nil || got != "main.py" {
		t.Fatalf("expected explicit entry, got %q: %v", got, err)
	}

	got, err = SelectEntry([]string{"z.txt", "a.txt"}, "")
	if err != nil || got != "a.txt" {
		t.Fatalf("expected lexical fallback, got %q: %v", got, err)
	}
}

func TestSelectEntryFallbackOrder(t *testing.T) {
	got, err := SelectEntry([]string{"index.py", "main.py", "z.sh"}, "")
	if err != nil || got != "main.py" {
		t.Fatalf("expected main entry, got %q: %v", got, err)
	}

	got, err = SelectEntry([]string{"z.py", "index.py", "a.py"}, "")
	if err != nil || got != "index.py" {
		t.Fatalf("expected index entry, got %q: %v", got, err)
	}
}

func TestSelectEntryPrefersPlatformVariant(t *testing.T) {
	got, err := SelectEntry([]string{"test.sh", "test.bat"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" && got != "test.bat" {
		t.Fatalf("expected windows variant, got %s", got)
	}
	if runtime.GOOS != "windows" && got != "test.sh" {
		t.Fatalf("expected Unix variant, got %s", got)
	}
}

func TestSelectEntryMissingIncludesAvailableFiles(t *testing.T) {
	_, err := SelectEntry([]string{"a.txt", "b.txt"}, "missing.txt")
	if err == nil || !strings.Contains(err.Error(), "a.txt") || !strings.Contains(err.Error(), "b.txt") {
		t.Fatalf("expected available-file diagnostics, got %v", err)
	}
}
