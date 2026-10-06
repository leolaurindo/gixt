package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestApply(t *testing.T) {
	for _, tc := range []struct {
		name      string
		zip       bool
		entry     string
		symlink   bool
		checksum  string
		status    int
		corrupt   bool
		canceled  bool
		large     bool
		link      bool
		wantError string
	}{
		{name: "tar release", entry: "gixt"},
		{name: "zip release", zip: true, entry: "gixt.exe"},
		{name: "symlinked installation", entry: "gixt", link: true},
		{name: "invalid archive", entry: "gixt", corrupt: true, wantError: "gzip: invalid header"},
		{name: "canceled update", entry: "gixt", canceled: true, wantError: "context canceled"},
		{name: "oversized checksum response", entry: "gixt", large: true, wantError: "payload exceeds"},
		{name: "wrong checksum", entry: "gixt", checksum: "wrong", wantError: "checksum verification failed"},
		{name: "missing checksum", entry: "gixt", checksum: "missing", wantError: "checksum not found"},
		{name: "failed download", entry: "gixt", status: http.StatusNotFound, wantError: "HTTP 404"},
		{name: "tar traversal", entry: "../gixt", wantError: "does not contain gixt"},
		{name: "zip traversal", zip: true, entry: "../gixt.exe", wantError: "does not contain gixt.exe"},
		{name: "tar symlink", entry: "gixt", symlink: true, wantError: "not a regular file"},
		{name: "zip symlink", zip: true, entry: "gixt.exe", symlink: true, wantError: "not a regular file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.link && runtime.GOOS == "windows" {
				t.Skip("creating symlinks requires Windows developer mode or elevated privileges")
			}
			archive := releaseArchive(t, tc.zip, tc.entry, tc.symlink)
			if tc.corrupt {
				archive = []byte("not a release archive")
			}
			asset := "release.tar.gz"
			if tc.zip {
				asset = "release.zip"
			}
			checksum := fmt.Sprintf("%x", sha256.Sum256(archive))
			if tc.checksum == "wrong" {
				checksum = strings.Repeat("0", 64)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/checksums.txt" {
					if tc.large {
						_, _ = w.Write(bytes.Repeat([]byte("x"), (1<<20)+1))
						return
					}
					if tc.checksum != "missing" {
						fmt.Fprintf(w, "%s  %s\n", checksum, asset)
					}
					return
				}
				if tc.status != 0 {
					w.WriteHeader(tc.status)
					return
				}
				_, _ = w.Write(archive)
			}))
			defer server.Close()

			dir := t.TempDir()
			target := filepath.Join(dir, "installed-gixt")
			if err := os.WriteFile(target, []byte("original executable"), 0o750); err != nil {
				t.Fatal(err)
			}
			invoked := target
			if tc.link {
				invoked = filepath.Join(dir, "launcher")
				if err := os.Symlink(target, invoked); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.canceled {
				cancel()
			}
			err := Apply(ctx, server.Client(), server.URL, asset, invoked)
			want := "updated executable"
			if tc.wantError != "" {
				want = "original executable"
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %q", err, tc.wantError)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(target)
			if err != nil || string(got) != want {
				t.Fatalf("installed bytes = %q, error = %v, want %q", got, err, want)
			}
			if runtime.GOOS != "windows" {
				info, err := os.Stat(target)
				if err != nil || info.Mode().Perm() != 0o750 {
					t.Fatalf("executable permissions were not preserved: %v, %v", info, err)
				}
			}
			wantEntries := 1
			if tc.link {
				wantEntries++
				if destination, err := os.Readlink(invoked); err != nil || destination != target {
					t.Fatalf("launcher symlink was modified: %q, %v", destination, err)
				}
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != wantEntries {
				t.Fatalf("update left staging files: %v, %v", entries, err)
			}
		})
	}
}

func TestApplyExcludesConcurrentUpdates(t *testing.T) {
	archive := releaseArchive(t, false, "gixt", false)
	started, release := make(chan struct{}, 1), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/checksums.txt" {
			select {
			case started <- struct{}{}:
			default:
			}
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
			fmt.Fprintf(w, "%x  release.tar.gz\n", sha256.Sum256(archive))
			return
		}
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	target := filepath.Join(t.TempDir(), "gixt")
	if err := os.WriteFile(target, []byte("original executable"), 0o755); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- Apply(ctx, server.Client(), server.URL, "release.tar.gz", target) }()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if err := Apply(ctx, server.Client(), server.URL, "release.tar.gz", target); err == nil || !strings.Contains(err.Error(), "cannot lock update") {
		t.Fatalf("concurrent update was not rejected: %v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestReplaceExecutableFailurePreservesCurrent(t *testing.T) {
	target := filepath.Join(t.TempDir(), "gixt")
	if err := os.WriteFile(target, []byte("original executable"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := replaceExecutable(filepath.Join(t.TempDir(), "missing"), target)
	if err == nil {
		t.Fatal("expected an install failure")
	}
	if runtime.GOOS == "windows" && !strings.Contains(err.Error(), "previous executable restored") {
		t.Fatalf("expected Windows rollback, got %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "original executable" {
		t.Fatalf("original executable not restored: %q, %v", got, err)
	}
}

// This exercises the operating system's running-executable replacement rules,
// not merely replacement of an ordinary file. CI runs it on Windows too.
func TestApplyRunningExecutable(t *testing.T) {
	if url := os.Getenv("GIXT_UPDATE_TEST_URL"); url != "" {
		target, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		asset := "release.tar.gz"
		if runtime.GOOS == "windows" {
			asset = "release.zip"
		}
		if err := Apply(context.Background(), &http.Client{Timeout: 15 * time.Second}, url, asset, target); err != nil {
			t.Fatal(err)
		}
		return
	}
	isZip := runtime.GOOS == "windows"
	name, asset := "gixt", "release.tar.gz"
	if isZip {
		name, asset = "gixt.exe", "release.zip"
	}
	archive := releaseArchive(t, isZip, name, false)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/checksums.txt" {
			fmt.Fprintf(w, "%x  %s\n", sha256.Sum256(archive), asset)
			return
		}
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(target, binary, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, target, "-test.run=^TestApplyRunningExecutable$")
	child.Env = append(os.Environ(), "GIXT_UPDATE_TEST_URL="+server.URL)
	if out, err := child.CombinedOutput(); err != nil {
		t.Fatalf("running-executable update failed: %v\n%s", err, out)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "updated executable" {
		t.Fatalf("running executable was not replaced: %q, %v", got, err)
	}
}

func releaseArchive(t *testing.T, isZip bool, name string, symlink bool) []byte {
	t.Helper()
	var buffer bytes.Buffer
	const content = "updated executable"
	if isZip {
		archive := zip.NewWriter(&buffer)
		header := &zip.FileHeader{Name: name}
		header.SetMode(0o755)
		if symlink {
			header.SetMode(os.ModeSymlink | 0o755)
		}
		file, err := archive.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := archive.Close(); err != nil {
			t.Fatal(err)
		}
	} else {
		compressed := gzip.NewWriter(&buffer)
		archive := tar.NewWriter(compressed)
		header := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}
		if symlink {
			header.Typeflag, header.Size, header.Linkname = tar.TypeSymlink, 0, "../outside"
		}
		if err := archive.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if !symlink {
			if _, err := archive.Write([]byte(content)); err != nil {
				t.Fatal(err)
			}
		}
		if err := archive.Close(); err != nil {
			t.Fatal(err)
		}
		if err := compressed.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return buffer.Bytes()
}
