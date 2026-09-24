package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/cache"
	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/gist"
)

func TestCatRejectsEntryWithMultipleTargets(t *testing.T) {
	cmd := newCatCmd()
	if err := cmd.Flags().Set("entry", "x"); err != nil {
		t.Fatal(err)
	}
	if err := catTargets(cmd, []string{"first", "second"}); err == nil || err.Error() != "--entry requires exactly one target" {
		t.Fatalf("expected multiple-target entry error, got %v", err)
	}
}

func TestCatOneWritesSelectedBytesWithoutSeparator(t *testing.T) {
	paths := config.Paths{
		CacheDir:  filepath.Join(t.TempDir(), "cache"),
		KnownFile: filepath.Join(t.TempDir(), "known.json"),
	}
	id := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	sha := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	dir := cache.Dir(paths.CacheDir, id, sha)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cache.SaveMeta(cache.MetaPath(dir), cache.Meta{
		GistID: id,
		SHA:    sha,
		Files:  []string{"a.bin", "b.bin"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.bin"), []byte{0x00, 0x01}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.bin"), []byte{0x02, 0x03}, 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	got, err := captureStdout(t, func() error {
		return catOne(cmd, paths, gist.New(""), id, catOptions{offline: true})
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte{0x00, 0x01, 0x02, 0x03}) {
		t.Fatalf("unexpected cat output: %v", got)
	}
}

func TestCatOneExplicitEntry(t *testing.T) {
	paths := config.Paths{CacheDir: filepath.Join(t.TempDir(), "cache")}
	id := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	sha := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	dir := cache.Dir(paths.CacheDir, id, sha)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cache.SaveMeta(cache.MetaPath(dir), cache.Meta{GistID: id, SHA: sha, Files: []string{"a.txt", "b.txt"}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	got, err := captureStdout(t, func() error {
		return catOne(cmd, paths, gist.New(""), id, catOptions{offline: true, entry: "b.txt"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "b" {
		t.Fatalf("expected selected entry, got %q", got)
	}
}

func captureStdout(t *testing.T, fn func() error) ([]byte, error) {
	t.Helper()
	old := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write
	callErr := fn()
	_ = write.Close()
	os.Stdout = old
	output, readErr := io.ReadAll(read)
	_ = read.Close()
	if callErr != nil {
		return output, callErr
	}
	return output, readErr
}
