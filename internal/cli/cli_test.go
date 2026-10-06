package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leolaurindo/gixt/internal/cache"
	"github.com/leolaurindo/gixt/internal/known"
)

func TestExecuteBareTargetSuggestions(t *testing.T) {
	home := t.TempDir()
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(key, filepath.Join(home, key))
	}
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GIXT_NO_CACHE", "")
	paths, err := ensurePaths()
	if err != nil {
		t.Fatal(err)
	}
	id, sha := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if err := known.Save(paths.KnownFile, known.Store{Entries: []known.Entry{
		{ID: id, Filenames: []string{"cc.bat"}},
	}}); err != nil {
		t.Fatal(err)
	}
	dir := cache.Dir(paths.CacheDir, id, sha)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cache.SaveMeta(cache.MetaPath(dir), cache.Meta{GistID: id, SHA: sha, Files: []string{"cc.bat"}}); err != nil {
		t.Fatal(err)
	}
	const content = "@echo off\npowershell -command \"Set-Clipboard -Path '%~1'\"\n"
	if err := os.WriteFile(filepath.Join(dir, "cc.bat"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name    string
		args    []string
		wantOut string
		wantErr string
	}{
		{name: "known stem", args: []string{"cc"}, wantOut: content},
		{name: "later target miss", args: []string{"cc", "pypi.txt"}, wantOut: content, wantErr: `could not resolve "pypi.txt"`},
		{name: "later command-like target miss", args: []string{"cc", "cta"}, wantOut: content, wantErr: `could not resolve "cta"`},
		{name: "first target typo", args: []string{"cta"}, wantErr: `unknown command "cta", did you mean "cat"?`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := captureStdout(t, func() error {
				return Execute(context.Background(), append([]string{"--offline"}, tc.args...))
			})
			if string(out) != tc.wantOut {
				t.Fatalf("stdout = %q, want %q", out, tc.wantOut)
			}
			if tc.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want %q", err, tc.wantErr)
			}
		})
	}
}
