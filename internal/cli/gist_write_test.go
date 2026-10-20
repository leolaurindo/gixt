package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/leolaurindo/gixt/internal/known"
)

type gistWriteTransport func(*http.Request) (*http.Response, error)

func (f gistWriteTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGistWrite(t *testing.T) {
	const id = "abcdef1234567890"
	const url = "https://gist.github.com/me/" + id
	for _, tc := range []struct {
		name   string
		args   []string
		public bool
	}{
		{name: "create secret", args: []string{"create"}},
		{name: "create public", args: []string{"create", "--public", "--description", "demo"}, public: true},
		{name: "update alias", args: []string{"update", "tool"}},
		{name: "update id", args: []string{"update", id}},
		{name: "update url", args: []string{"update", url}},
		{name: "update owner filename", args: []string{"update", "me/main.py"}},
		{name: "update filename", args: []string{"update", "main.py"}},
		{name: "update stem", args: []string{"update", "main"}},
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
			updating := tc.args[0] == "update"
			if updating {
				if err := known.Save(paths.KnownFile, known.Store{Entries: []known.Entry{{ID: id, Alias: "tool", Pin: "old-revision", Owner: "me", Filenames: []string{"main.py"}}}}); err != nil {
					t.Fatal(err)
				}
			}
			// Writes must not approve a new revision or modify prior trust.
			if err := os.WriteFile(paths.TrustFile, []byte("existing trust"), 0o600); err != nil {
				t.Fatal(err)
			}
			files := []string{filepath.Join(home, "main.py"), filepath.Join(home, "prompt.md")}
			for i, content := range []string{"print('hi')\n", "Review this.\n"} {
				if err := os.WriteFile(files[i], []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			requests := 0
			old := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = old })
			http.DefaultTransport = gistWriteTransport(func(r *http.Request) (*http.Response, error) {
				requests++
				method, path := http.MethodPost, "/gists"
				if updating {
					method, path = http.MethodPatch, "/gists/"+id
				}
				if r.Method != method || r.URL.Path != path || r.Header.Get("Authorization") != "Bearer test-token" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
				}
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				want := map[string]any{"files": map[string]any{
					"main.py":   map[string]any{"content": "print('hi')\n"},
					"prompt.md": map[string]any{"content": "Review this.\n"},
				}}
				if !updating {
					want["public"] = tc.public
					if tc.public {
						want["description"] = "demo"
					}
				}
				if !reflect.DeepEqual(payload, want) {
					t.Fatalf("payload = %#v, want %#v", payload, want)
				}
				body := `{"id":"` + id + `","html_url":"` + url + `","description":"fresh","owner":{"login":"me"},"files":{"main.py":{},"prompt.md":{},"kept.txt":{}},"history":[{"version":"new-revision"}]}`
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			root := newRootCmd()
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetArgs(append(append([]string{"gist"}, tc.args...), files...))
			if err := root.ExecuteContext(context.Background()); err != nil {
				t.Fatal(err)
			}
			if out.String() != url+"\n" || requests != 1 {
				t.Fatalf("output %q, requests %d", out.String(), requests)
			}
			store, err := known.Load(paths.KnownFile)
			if err != nil {
				t.Fatal(err)
			}
			if len(store.Entries) != 1 {
				t.Fatalf("entries: %+v", store.Entries)
			}
			entry := store.Entries[0]
			if entry.ID != id || entry.Description != "fresh" || !reflect.DeepEqual(entry.Filenames, []string{"kept.txt", "main.py", "prompt.md"}) {
				t.Fatalf("metadata: %+v", entry)
			}
			if updating && (entry.Alias != "tool" || entry.Pin != "old-revision") {
				t.Fatalf("lost alias or pin: %+v", entry)
			}
			if !updating && (entry.Alias != "" || entry.Pin != "") {
				t.Fatalf("unexpected alias or pin: %+v", entry)
			}
			trust, err := os.ReadFile(paths.TrustFile)
			if err != nil || string(trust) != "existing trust" {
				t.Fatalf("trust changed: %q, %v", trust, err)
			}
		})
	}
}

func TestReadUploadFilesRejectsInvalidInputs(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string][]byte{"empty.txt": {}, "binary.txt": {0xff}, "valid.txt": []byte("hello")} {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		paths []string
		want  string
	}{
		{[]string{"-"}, "stdin"},
		{[]string{dir}, "regular file"},
		{[]string{filepath.Join(dir, "missing")}, "read"},
		{[]string{filepath.Join(dir, "empty.txt")}, "non-empty UTF-8"},
		{[]string{filepath.Join(dir, "binary.txt")}, "non-empty UTF-8"},
		{[]string{filepath.Join(dir, "valid.txt"), filepath.Join(dir, "other", "valid.txt")}, "duplicate"},
	} {
		if _, err := readUploadFiles(tc.paths); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: error = %v, want %q", tc.paths, err, tc.want)
		}
	}
}

func TestGistWriteRequiresFiles(t *testing.T) {
	for _, args := range [][]string{{"gist", "create"}, {"gist", "update"}, {"gist", "update", "tool"}} {
		if err := Execute(context.Background(), args); err == nil {
			t.Errorf("%v: expected missing argument error", args)
		}
	}
}

func TestGistWriteFailurePreservesMetadata(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		t.Run(operation, func(t *testing.T) {
			home := t.TempDir()
			for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "APPDATA", "LOCALAPPDATA"} {
				t.Setenv(key, filepath.Join(home, key))
			}
			t.Setenv("GITHUB_TOKEN", "test-token")
			paths, err := ensurePaths()
			if err != nil {
				t.Fatal(err)
			}
			if err := known.Save(paths.KnownFile, known.Store{Entries: []known.Entry{{ID: "abcdef1234567890", Alias: "tool", Pin: "old"}}}); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(paths.KnownFile)
			if err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(home, "main.py")
			if err := os.WriteFile(file, []byte("print(1)"), 0o600); err != nil {
				t.Fatal(err)
			}
			old := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = old })
			http.DefaultTransport = gistWriteTransport(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusForbidden, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"message":"denied"}`))}, nil
			})
			args := []string{"gist", operation}
			if operation == "update" {
				args = append(args, "tool")
			}
			root := newRootCmd()
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetArgs(append(args, file))
			if err := root.ExecuteContext(context.Background()); err == nil || !strings.Contains(err.Error(), "403") {
				t.Fatalf("expected API failure, got %v", err)
			}
			after, err := os.ReadFile(paths.KnownFile)
			if err != nil || !bytes.Equal(before, after) || out.Len() != 0 {
				t.Fatalf("write failure changed metadata/output: %v, %q", err, out.String())
			}
			if _, err := os.Stat(paths.TrustFile); !os.IsNotExist(err) {
				t.Fatalf("unexpected trust file: %v", err)
			}
		})
	}
}
