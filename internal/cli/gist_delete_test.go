package cli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/leolaurindo/gixt/internal/known"
)

func TestGistDelete(t *testing.T) {
	const id = "abcdef1234567890"
	for _, tc := range []struct {
		name      string
		target    string
		yes       bool
		status    int
		wantError string
	}{
		{name: "alias", target: "hello", yes: true, status: http.StatusNoContent},
		{name: "id", target: id, yes: true, status: http.StatusNoContent},
		{name: "url", target: "https://gist.github.com/me/" + id, yes: true, status: http.StatusNoContent},
		{name: "unregistered id", target: "bbbbbbbbbbbbbbbb", yes: true, status: http.StatusNoContent},
		{name: "confirmation required", target: "hello", wantError: "--yes"},
		{name: "forbidden", target: "hello", yes: true, status: http.StatusForbidden, wantError: "403"},
		{name: "missing remote", target: "hello", yes: true, status: http.StatusNotFound, wantError: "404"},
		{name: "unknown target", target: "unknown", yes: true, wantError: "could not resolve"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "APPDATA", "LOCALAPPDATA"} {
				t.Setenv(key, filepath.Join(home, key))
			}
			t.Setenv("GITHUB_TOKEN", "test-token")
			paths, err := ensurePaths()
			if err != nil {
				t.Fatal(err)
			}
			entries := []known.Entry{{ID: id, Alias: "hello", Pin: "old"}, {ID: "cccccccccccccccc", Alias: "keep", Pin: "pinned"}}
			if err := known.Save(paths.KnownFile, known.Store{Entries: entries}); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(paths.KnownFile)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(paths.TrustFile, []byte("keep trust"), 0o600); err != nil {
				t.Fatal(err)
			}
			stdin, err := os.CreateTemp(home, "stdin")
			if err != nil {
				t.Fatal(err)
			}
			oldStdin, oldTransport := os.Stdin, http.DefaultTransport
			os.Stdin = stdin
			t.Cleanup(func() { os.Stdin, http.DefaultTransport = oldStdin, oldTransport; stdin.Close() })
			calls := 0
			http.DefaultTransport = gistWriteTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				wantID := id
				if tc.name == "unregistered id" {
					wantID = tc.target
				}
				if r.Method != http.MethodDelete || r.URL.Path != "/gists/"+wantID || r.Header.Get("Authorization") != "Bearer test-token" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
				}
				return &http.Response{StatusCode: tc.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
			})
			root := newRootCmd()
			var out bytes.Buffer
			root.SetOut(&out)
			args := []string{"gist", "delete", tc.target}
			if tc.yes {
				args = append(args, "--yes")
			}
			root.SetArgs(args)
			err = root.ExecuteContext(context.Background())
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %q", err, tc.wantError)
				}
				after, readErr := os.ReadFile(paths.KnownFile)
				if readErr != nil || !bytes.Equal(before, after) || out.Len() != 0 {
					t.Fatalf("metadata/output changed on failure: %v, %q", readErr, out.String())
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				wantID := id
				wantEntries := entries[1:]
				if tc.name == "unregistered id" {
					wantID, wantEntries = tc.target, entries
				}
				if out.String() != "https://gist.github.com/"+wantID+"\n" {
					t.Fatalf("output = %q", out.String())
				}
				store, err := known.Load(paths.KnownFile)
				if err != nil || !reflect.DeepEqual(store.Entries, wantEntries) {
					t.Fatalf("entries = %+v, error = %v", store.Entries, err)
				}
			}
			wantCalls := 1
			if !tc.yes || tc.name == "unknown target" {
				wantCalls = 0
			}
			if calls != wantCalls {
				t.Fatalf("requests = %d, want %d", calls, wantCalls)
			}
			trust, err := os.ReadFile(paths.TrustFile)
			if err != nil || string(trust) != "keep trust" {
				t.Fatalf("trust changed: %q, %v", trust, err)
			}
		})
	}
}

func TestGistDeleteRequiresOneTarget(t *testing.T) {
	for _, args := range [][]string{{"gist", "delete", "--yes"}, {"gist", "delete", "one", "two", "--yes"}} {
		if err := Execute(context.Background(), args); err == nil {
			t.Errorf("%v: expected argument error", args)
		}
	}
}
