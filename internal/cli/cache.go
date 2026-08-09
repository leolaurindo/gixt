package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/cache"
	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/known"
)

func newCacheCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "cache",
		Short: "manage cached gist contents",
		Args:  cobra.NoArgs,
	}
	c.AddCommand(
		&cobra.Command{Use: "list", Short: "list cached gists", Args: cobra.NoArgs, RunE: cacheList},
		&cobra.Command{Use: "prune", Short: "remove old revisions, keeping only the latest per gist", Args: cobra.NoArgs, RunE: cachePrune},
		&cobra.Command{Use: "clear", Short: "remove all cached gist contents", Args: cobra.NoArgs, RunE: cacheClear},
	)
	return c
}

func cacheList(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	root := paths.CacheDir
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if dir, m, ok := cache.Latest(root, e.Name()); ok {
			fmt.Printf("%s\t%s\t%s\n", e.Name(), cache.Shorten(m.SHA), dir)
		}
	}
	return nil
}

func cachePrune(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	root := paths.CacheDir
	st, err := known.Load(paths.KnownFile)
	if err != nil {
		return err
	}
	pins := make(map[string]string)
	for _, e := range st.Entries {
		if e.Pin != "" {
			pins[e.ID] = e.Pin
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, m, ok := cache.Latest(root, e.Name()); ok {
			if err := cache.Prune(root, e.Name(), m.SHA, pins[e.Name()]); err != nil {
				return err
			}
		}
	}
	logf("cache pruned")
	return nil
}

func cacheClear(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	if err := os.RemoveAll(paths.CacheDir); err != nil {
		return err
	}
	if err := config.EnsureDirs(paths); err != nil {
		return err
	}
	logf("cache cleared")
	return nil
}
