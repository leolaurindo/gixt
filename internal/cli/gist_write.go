package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/gist"
	"github.com/leolaurindo/gixt/internal/known"
)

func gistCreate(cmd *cobra.Command, args []string) error {
	files, err := readUploadFiles(args)
	if err != nil {
		return err
	}
	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	client := gist.New(loadToken(paths.AuthFile))
	public, _ := cmd.Flags().GetBool("public")
	created, err := client.Create(cmd.Context(), files, mustString(cmd, "description"), public)
	if err != nil {
		return err
	}
	return recordWrittenGist(cmd, paths, created)
}

func gistUpdate(cmd *cobra.Command, args []string) error {
	files, err := readUploadFiles(args[1:])
	if err != nil {
		return err
	}
	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	id, err := resolveTarget(cmd.Context(), args[0], paths, true)
	if err != nil {
		return err
	}
	client := gist.New(loadToken(paths.AuthFile))
	updated, err := client.UpdateFiles(cmd.Context(), id, files)
	if err != nil {
		return err
	}
	return recordWrittenGist(cmd, paths, updated)
}

func readUploadFiles(paths []string) (map[string]string, error) {
	files := make(map[string]string, len(paths))
	for _, path := range paths {
		if path == "-" {
			return nil, fmt.Errorf("stdin is not supported; provide local files")
		}
		name := filepath.Base(path)
		if _, exists := files[name]; exists {
			return nil, fmt.Errorf("duplicate gist filename %q", name)
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a regular file", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		if len(data) == 0 || !utf8.Valid(data) {
			return nil, fmt.Errorf("%s must contain non-empty UTF-8 text", path)
		}
		files[name] = string(data)
	}
	return files, nil
}

func recordWrittenGist(cmd *cobra.Command, paths config.Paths, g gist.Gist) error {
	if err := saveKnown(paths, func(s *known.Store) {
		alias := ""
		for _, entry := range s.Entries {
			if entry.ID == g.ID {
				alias = entry.Alias
				break
			}
		}
		known.Upsert(s, toKnownEntry(g, alias))
	}); err != nil {
		return fmt.Errorf("gist saved at %s, but local metadata update failed: %w", g.HTMLURL, err)
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), g.HTMLURL)
	return err
}
