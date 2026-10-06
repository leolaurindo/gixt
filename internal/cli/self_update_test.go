package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/leolaurindo/gixt/internal/version"
)

func TestSelfUpdateHomebrew(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Homebrew supports macOS and Linux")
	}
	if os.Getenv("GIXT_BREW_TEST_CHILD") == "1" {
		http.DefaultTransport = selfUpdateTransport{}
		err := Execute(context.Background(), []string{"self", "update"})
		want := os.Getenv("GIXT_BREW_TEST_ERROR")
		if want == "" && err != nil {
			t.Fatal(err)
		}
		if want != "" && (err == nil || !strings.Contains(err.Error(), want)) {
			t.Fatalf("error = %v, want %q", err, want)
		}
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name      string
		mismatch  bool
		missing   bool
		wantError string
	}{
		{name: "verified formula"},
		{name: "different installation", mismatch: true, wantError: "does not match Homebrew"},
		{name: "brew unavailable", missing: true, wantError: "restore brew on PATH"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			prefix := filepath.Join(root, "Cellar", "gixt", "1.0.0")
			target := filepath.Join(prefix, "bin", "gixt")
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, binary, 0o755); err != nil {
				t.Fatal(err)
			}
			if tc.mismatch {
				prefix = filepath.Join(root, "other")
				if err := os.MkdirAll(filepath.Join(prefix, "bin"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(prefix, "bin", "gixt"), []byte("unrelated executable"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			bin := filepath.Join(root, "tools")
			if err := os.MkdirAll(bin, 0o755); err != nil {
				t.Fatal(err)
			}
			if !tc.missing {
				const script = `#!/bin/sh
if [ "$1" = "--prefix" ] && [ "$2" = "gixt" ]; then
    printf '%s\n' "$GIXT_BREW_TEST_PREFIX"
    exit 0
fi
printf 'unexpected command' > "$GIXT_BREW_TEST_MARKER"
exit 64
`
				if err := os.WriteFile(filepath.Join(bin, "brew"), []byte(script), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			marker := filepath.Join(root, "upgraded")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			launcher := filepath.Join(root, "gixt")
			if err := os.Symlink(target, launcher); err != nil {
				t.Fatal(err)
			}
			child := exec.CommandContext(ctx, launcher, "-test.run=^TestSelfUpdateHomebrew$")
			child.Env = append(os.Environ(), "PATH="+bin, "GIXT_BREW_TEST_CHILD=1",
				"GIXT_BREW_TEST_PREFIX="+prefix, "GIXT_BREW_TEST_MARKER="+marker,
				"GIXT_BREW_TEST_ERROR="+tc.wantError)
			out, err := child.CombinedOutput()
			if err != nil {
				t.Fatalf("self update failed: %v\n%s", err, out)
			}
			if tc.wantError == "" && !strings.Contains(string(out), "Update with: brew upgrade gixt") {
				t.Fatalf("missing Homebrew update guidance: %s", out)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("self update ran an unexpected brew command: %v", err)
			}
			got, err := os.ReadFile(target)
			if err != nil || string(got) != string(binary) {
				t.Fatalf("Homebrew executable was modified directly: %v", err)
			}
		})
	}
}

func TestSelfUpdateVersionPolicy(t *testing.T) {
	oldVersion, oldTransport := version.Version, http.DefaultTransport
	t.Cleanup(func() { version.Version, http.DefaultTransport = oldVersion, oldTransport })
	for _, tc := range []struct {
		name, current, latest, wantError string
	}{
		{name: "up to date", current: "v0.5.0", latest: "v0.5.0"},
		{name: "no downgrade", current: "v0.6.0", latest: "v0.5.0"},
		{name: "invalid release", current: "v0.4.2", latest: "v0.5.0/../other", wantError: "invalid stable release version"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			version.Version = tc.current
			http.DefaultTransport = selfUpdateTransport{latest: tc.latest}
			err := Execute(context.Background(), []string{"self", "update"})
			if tc.wantError == "" && err != nil {
				t.Fatal(err)
			}
			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Fatalf("error = %v, want %q", err, tc.wantError)
			}
		})
	}
}

type selfUpdateTransport struct{ latest string }

func (transport selfUpdateTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if transport.latest != "" && req.URL.String() == "https://api.github.com/repos/leolaurindo/gixt/releases/latest" {
		body := fmt.Sprintf(`{"tag_name":%q}`, transport.latest)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: req}, nil
	}
	return nil, fmt.Errorf("unexpected network request during self update: %s", req.URL)
}
