package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/leolaurindo/gixt/internal/alias"
	"github.com/leolaurindo/gixt/internal/cache"
	"github.com/leolaurindo/gixt/internal/gist"
	"github.com/leolaurindo/gixt/internal/index"
)

func handleClone(ctx context.Context, target string, dir string) error {
	if strings.TrimSpace(target) == "" {
		return errors.New("usage: gixt clone <gist-id|url|alias|name|owner/name> [--dir <path>]")
	}
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	aliases, _ := alias.Load(paths.AliasFile)
	id, _, _, err := resolveIdentifier(ctx, target, aliases, paths, false, true, normalizeUserPages(0))
	if err != nil {
		return err
	}
	dest := dir
	if strings.TrimSpace(dest) == "" {
		dest = id
	}
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("target path %s already exists", dest)
	}
	cmd := exec.CommandContext(ctx, "gh", "gist", "clone", id, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh gist clone failed: %w", err)
	}
	fmt.Printf("cloned gist %s into %s\n", cache.Shorten(id), dest)
	return nil
}

func handleFork(ctx context.Context, target string, public bool, desc string) error {
	if strings.TrimSpace(target) == "" {
		return errors.New("usage: gixt fork <gist-id|url|alias|name|owner/name> [--public] [--description <desc>]")
	}
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	aliases, _ := alias.Load(paths.AliasFile)
	id, _, _, err := resolveIdentifier(ctx, target, aliases, paths, false, true, normalizeUserPages(0))
	if err != nil {
		return err
	}

	g, err := gist.Fetch(ctx, id, "")
	if err != nil {
		return err
	}
	files, err := extractFiles(ctx, g)
	if err != nil {
		return err
	}
	description := strings.TrimSpace(desc)
	if description == "" {
		description = g.Description
	}

	newGist, err := gist.Create(ctx, files, description, public)
	if err != nil {
		return err
	}
	fmt.Printf("forked gist %s -> %s (%s)\n", cache.Shorten(id), cache.Shorten(newGist.ID), newGist.HTMLURL)

	// Update index with new gist
	idx, _ := index.Load(paths.IndexFile)
	idx.Entries = append(idx.Entries, toIndexEntryFromGist(newGist))
	sortIndexEntries(idx.Entries)
	idx.GeneratedAt = newGist.UpdatedAt
	_ = index.Save(paths.IndexFile, idx)

	return nil
}

func extractFiles(ctx context.Context, g gist.Gist) (map[string]string, error) {
	out := map[string]string{}
	client := http.Client{}
	for name, f := range g.Files {
		if f.Content != "" && !f.Truncated {
			out[name] = f.Content
			continue
		}
		if f.RawURL == "" {
			return nil, fmt.Errorf("gist file %s has no content or raw_url", name)
		}
		req, err := http.NewRequestWithContext(ctx, "GET", f.RawURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("download %s: %w", name, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("download %s: http %d", name, resp.StatusCode)
		}
		out[name] = string(body)
	}
	return out, nil
}
