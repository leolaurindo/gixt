package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/leolaurindo/gixt/internal/cache"
	"github.com/leolaurindo/gixt/internal/gist"
)

func materializeFiles(g gist.Gist, dir string, forceUpdate bool) ([]string, bool, error) {
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

	manifestPath := cache.ManifestPath(dir)
	if !forceUpdate {
		if cache.PathExists(manifestPath) {
			existing, err := cache.LoadManifest(manifestPath)
			if err == nil {
				valid := true
				for _, f := range existing.Files {
					if _, err := sanitizeGistPath(f); err != nil {
						valid = false
						break
					}
				}
				if valid && cache.PresentFiles(dir, existing.Files) {
					return existing.Files, true, nil
				}
			}
		}
	}

	client := http.Client{Timeout: 30 * time.Second}
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
			resp, err := client.Get(info.RawURL)
			if err != nil {
				return nil, false, fmt.Errorf("download %s: %w", name, err)
			}
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, false, fmt.Errorf("read %s: %w", name, err)
			}
			if resp.StatusCode >= 300 {
				return nil, false, fmt.Errorf("download %s: http %d", name, resp.StatusCode)
			}
			data = body
		}
		mode := fileModeFor(name)
		if err := os.WriteFile(target, data, mode); err != nil {
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
		if runtime.GOOS == "windows" {
			return 0o644
		}
		return 0o644
	}
}
