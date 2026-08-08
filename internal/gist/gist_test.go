package gist

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func gistJSON(id, rawURL string) string {
	return `{
		"id": "` + id + `",
		"description": "demo gist",
		"html_url": "https://gist.github.com/me/` + id + `",
		"owner": {"login": "me"},
		"files": {"main.py": {"filename": "main.py", "raw_url": "` + rawURL + `", "content": "print('hi')"}},
		"history": [{"version": "abc123def456"}]
	}`
}

func TestFetchAndLatestVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(gistJSON("1234567890abcdef", "https://example.com/raw")))
	}))
	defer srv.Close()

	c := newWithBase(srv.URL, "")
	g, err := c.Fetch(context.Background(), "1234567890abcdef", "")
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if g.ID != "1234567890abcdef" {
		t.Fatalf("unexpected id: %s", g.ID)
	}
	if g.LatestVersion() != "abc123def456" {
		t.Fatalf("unexpected version: %s", g.LatestVersion())
	}
	if GuessOwner(g) != "me" {
		t.Fatalf("unexpected owner: %s", GuessOwner(g))
	}
}

func TestFetchCachedNotModified(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == `"etag-1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Etag", `"etag-1"`)
		w.Write([]byte(gistJSON("1234567890abcdef", "https://example.com/raw")))
	}))
	defer srv.Close()

	c := newWithBase(srv.URL, "")

	g, _, notModified, err := c.FetchCached(context.Background(), "1234567890abcdef", "", "")
	if err != nil {
		t.Fatalf("FetchCached error: %v", err)
	}
	if notModified || g.ID == "" {
		t.Fatalf("expected full fetch on first call")
	}

	_, etag, notModified, err := c.FetchCached(context.Background(), "1234567890abcdef", "", `"etag-1"`)
	if err != nil {
		t.Fatalf("FetchCached error: %v", err)
	}
	if !notModified {
		t.Fatalf("expected 304 when etag matches")
	}
	if etag != "" {
		t.Fatalf("unexpected etag on 304: %s", etag)
	}
}

func TestAuthHeaderSent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sekret" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Write([]byte(`{"login":"me"}`))
	}))
	defer srv.Close()

	c := newWithBase(srv.URL, "sekret")
	user, err := c.CurrentUser(context.Background())
	if err != nil {
		t.Fatalf("CurrentUser error: %v", err)
	}
	if user != "me" {
		t.Fatalf("unexpected user: %s", user)
	}
}

func TestDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/raw/") {
			w.Write([]byte("script-body"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newWithBase(srv.URL, "")
	data, err := c.Download(context.Background(), srv.URL+"/raw/main.py")
	if err != nil {
		t.Fatalf("Download error: %v", err)
	}
	if string(data) != "script-body" {
		t.Fatalf("unexpected body: %s", data)
	}
}

func TestNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newWithBase(srv.URL, "")
	if _, err := c.Fetch(context.Background(), "deadbeefdeadbeef", ""); !IsNotFound(err) {
		t.Fatalf("expected IsNotFound, got %v", err)
	}
}

func TestExtractID(t *testing.T) {
	cases := map[string]string{
		"1234567890abcdef":                "1234567890abcdef",
		"https://gist.github.com/me/abc":  "abc",
		"https://gist.github.com/me/abc#": "abc",
		"  spaced  ":                      "spaced",
	}
	for in, want := range cases {
		if got := ExtractID(in); got != want {
			t.Fatalf("ExtractID(%q) = %q, want %q", in, got, want)
		}
	}
}

func mutationServer(t *testing.T, wantMethod, wantPath string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != wantMethod || r.URL.Path != wantPath {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sekret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte(gistJSON("new1234567890ab", "https://example.com/raw")))
	}))
}

func TestUpdateDescription(t *testing.T) {
	srv := mutationServer(t, http.MethodPatch, "/gists/abc1234567890")
	defer srv.Close()

	c := newWithBase(srv.URL, "sekret")
	g, err := c.UpdateDescription(context.Background(), "abc1234567890", "new desc")
	if err != nil {
		t.Fatalf("UpdateDescription error: %v", err)
	}
	if g.ID != "new1234567890ab" {
		t.Fatalf("unexpected id: %s", g.ID)
	}
}

func TestCreate(t *testing.T) {
	srv := mutationServer(t, http.MethodPost, "/gists")
	defer srv.Close()

	c := newWithBase(srv.URL, "sekret")
	g, err := c.Create(context.Background(), map[string]string{"a.py": "print(1)"}, "desc", false)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if g.ID != "new1234567890ab" {
		t.Fatalf("unexpected id: %s", g.ID)
	}
}

func TestMutationRequiresToken(t *testing.T) {
	c := newWithBase("http://unused", "")
	if _, err := c.Create(context.Background(), nil, "", false); err == nil {
		t.Fatal("expected error when not authenticated")
	}
	if _, err := c.UpdateDescription(context.Background(), "abc", "x"); err == nil {
		t.Fatal("expected error when not authenticated")
	}
}
