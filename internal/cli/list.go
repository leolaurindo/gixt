package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/gist"
	"github.com/leolaurindo/gixt/internal/known"
)

func newListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "show your known gists",
		Args:  cobra.NoArgs,
		RunE:  listGists,
	}
	c.AddCommand(
		&cobra.Command{
			Use:   "refresh",
			Short: "refresh metadata for every known gist",
			Args:  cobra.NoArgs,
			RunE:  listRefresh,
		},
		&cobra.Command{
			Use:   "clear",
			Short: "forget all known gists",
			Args:  cobra.NoArgs,
			RunE:  listClear,
		},
	)
	return c
}

func listGists(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	st, err := known.Load(paths.KnownFile)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "OWNER\tID\tNAME\tDESCRIPTION")
	for _, e := range st.Sorted() {
		name := e.Alias
		if name == "" && len(e.Filenames) > 0 {
			name = e.Filenames[0]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Owner, e.ID, name, e.Description)
	}
	return w.Flush()
}

func listRefresh(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	st, err := known.Load(paths.KnownFile)
	if err != nil {
		return err
	}
	client := gist.New(loadToken(paths.AuthFile))
	kept := make([]known.Entry, 0, len(st.Entries))
	for _, e := range st.Entries {
		g, err := client.Fetch(cmd.Context(), e.ID, "")
		if err != nil {
			if gist.IsNotFound(err) {
				if e.Pin != "" {
					logf("retained %s (pinned revision)", e.ID)
					kept = append(kept, e)
					continue
				}
				logf("removed %s (deleted)", e.ID)
				continue
			}
			return err
		}
		g2 := toKnownEntry(g, e.Alias)
		g2.Pin = e.Pin
		kept = append(kept, g2)
	}
	st.Entries = kept
	return known.Save(paths.KnownFile, st)
}

func listClear(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	if err := os.Remove(paths.KnownFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	logf("known gists cleared")
	return nil
}
