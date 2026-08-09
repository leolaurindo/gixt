package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/cache"
	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/gist"
	"github.com/leolaurindo/gixt/internal/known"
)

// pinnedRef returns the pinned revision for a gist, or "" if not pinned.
func pinnedRef(paths config.Paths, id string) (string, error) {
	st, err := known.Load(paths.KnownFile)
	if err != nil {
		return "", err
	}
	for _, e := range st.Entries {
		if e.ID == id && e.Pin != "" {
			return e.Pin, nil
		}
	}
	return "", nil
}

func newPinCmd() *cobra.Command {
	pin := &cobra.Command{
		Use:   "pin <target> [<sha>]",
		Short: "pin a gist to a revision",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  pinGist,
	}
	pin.AddCommand(
		&cobra.Command{Use: "list", Short: "show pinned gists", Args: cobra.NoArgs, RunE: pinList},
		&cobra.Command{Use: "remove <target>", Short: "unpin a gist", Args: cobra.ExactArgs(1), RunE: pinRemove},
		&cobra.Command{Use: "clear", Short: "unpin every gist", Args: cobra.NoArgs, RunE: pinClear},
	)
	return pin
}

func pinGist(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	client := gist.New(loadToken(paths.AuthFile))
	id, _, err := resolveTarget(cmd.Context(), args[0], paths)
	if err != nil {
		return err
	}
	ref := ""
	if len(args) == 2 {
		ref = args[1]
	}
	g, err := client.Fetch(cmd.Context(), id, ref)
	if err != nil {
		return err
	}
	sha := g.LatestVersion()
	if sha == "" {
		return fmt.Errorf("could not determine the requested revision of %s", id)
	}

	st, err := known.Load(paths.KnownFile)
	if err != nil {
		return err
	}
	var entry *known.Entry
	for i := range st.Entries {
		if st.Entries[i].ID == id {
			entry = &st.Entries[i]
			break
		}
	}
	if entry == nil {
		st.Entries = append(st.Entries, toKnownEntry(g, ""))
		entry = &st.Entries[len(st.Entries)-1]
	}
	entry.Pin = sha
	st.GeneratedAt = time.Now()
	if err := known.Save(paths.KnownFile, st); err != nil {
		return err
	}
	logf("pinned gist %s to %s", id, cache.Shorten(sha))
	return nil
}

func pinList(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	st, err := known.Load(paths.KnownFile)
	if err != nil {
		return err
	}
	for _, e := range st.Sorted() {
		if e.Pin == "" {
			continue
		}
		name := e.Alias
		if name == "" && len(e.Filenames) > 0 {
			name = e.Filenames[0]
		}
		fmt.Printf("%s\t%s\t%s\t%s\n", e.Owner, name, e.ID, cache.Shorten(e.Pin))
	}
	return nil
}

func pinRemove(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	id, _, err := resolveTarget(cmd.Context(), args[0], paths)
	if err != nil {
		return err
	}
	st, err := known.Load(paths.KnownFile)
	if err != nil {
		return err
	}
	for i := range st.Entries {
		if st.Entries[i].ID == id {
			if st.Entries[i].Pin == "" {
				return fmt.Errorf("gist %s is not pinned", id)
			}
			st.Entries[i].Pin = ""
			return known.Save(paths.KnownFile, st)
		}
	}
	return fmt.Errorf("gist %s is not in the known list", id)
}

func pinClear(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	st, err := known.Load(paths.KnownFile)
	if err != nil {
		return err
	}
	for i := range st.Entries {
		st.Entries[i].Pin = ""
	}
	return known.Save(paths.KnownFile, st)
}
