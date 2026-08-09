package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leolaurindo/gixt/internal/cache"
	"github.com/leolaurindo/gixt/internal/gist"
)

// materializeFiles writes a gist's files into dir. When the work dir already
// holds the same files (cached), they are reused and no downloads happen.
func materializeFiles(ctx context.Context, client *gist.Client, g gist.Gist, dir string) ([]string, bool, error) {
	type gistFile struct {
		name string
		info gist.File
	}
	seen := map[string]bool{}
	var files []gistFile
	for name, info := range g.Files {
		sanitized, err := sanitizeGistPath(name)
		if err != nil {
			return nil, false, err
		}
		if seen[sanitized] {
			return nil, false, fmt.Errorf("duplicate file after sanitization: %s", sanitized)
		}
		seen[sanitized] = true
		files = append(files, gistFile{name: sanitized, info: info})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	var filenames []string
	for _, f := range files {
		filenames = append(filenames, f.name)
	}

	metaPath := cache.MetaPath(dir)
	if _, err := os.Stat(metaPath); err == nil {
		existing, err := cache.LoadMeta(metaPath)
		if err == nil && cache.PresentFiles(dir, existing.Files) {
			return existing.Files, true, nil
		}
	}

	for _, gf := range files {
		name := gf.name
		info := gf.info
		target := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, false, err
		}
		var data []byte
		if info.Content != "" && !info.Truncated {
			data = []byte(info.Content)
		} else {
			var err error
			data, err = client.Download(ctx, info.RawURL)
			if err != nil {
				return nil, false, fmt.Errorf("download %s: %w", name, err)
			}
		}
		if err := os.WriteFile(target, data, fileModeFor(name)); err != nil {
			return nil, false, fmt.Errorf("write file %s: %w", name, err)
		}
	}
	return filenames, false, nil
}

func sanitizeGistPath(name string) (string, error) {
	cleaned := filepath.Clean(name)
	if cleaned == "" || cleaned == "." {
		return "", fmt.Errorf("invalid file name %q", name)
	}
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("invalid file name %q: absolute paths are not allowed", name)
	}
	if strings.HasPrefix(cleaned, "..") || cleaned == ".." {
		return "", fmt.Errorf("invalid file name %q: parent traversal is not allowed", name)
	}
	if vol := filepath.VolumeName(cleaned); vol != "" {
		return "", fmt.Errorf("invalid file name %q: drive-prefixed paths are not allowed", name)
	}
	return cleaned, nil
}

func fileModeFor(name string) os.FileMode {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".sh", ".bash", ".zsh", ".py", ".rb", ".pl", ".php", ".js", ".ts", ".go":
		return 0o755
	default:
		return 0o644
	}
}
