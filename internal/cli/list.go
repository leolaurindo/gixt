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
		Use:   "list [owner]",
		Short: "list remembered gists or an owner's gists",
		Args:  cobra.MaximumNArgs(1),
		RunE:  listGists,
	}
	c.Flags().Int("limit", 30, "maximum number of remote gists to list")
	c.Flags().Bool("all", false, "list all remote gists")
	return c
}

func listGists(cmd *cobra.Command, args []string) error {
	all := mustBool(cmd, "all")
	limit, _ := cmd.Flags().GetInt("limit")
	if len(args) == 0 {
		if cmd.Flags().Changed("limit") || all {
			return fmt.Errorf("--limit and --all require an owner")
		}
		paths, err := ensurePaths()
		if err != nil {
			return err
		}
		st, err := known.Load(paths.KnownFile)
		if err != nil {
			return err
		}
		return writeList(st.Sorted())
	}
	if cmd.Flags().Changed("limit") && all {
		return fmt.Errorf("--limit and --all are mutually exclusive")
	}
	if !all && limit <= 0 {
		return fmt.Errorf("--limit must be positive")
	}

	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	client := gist.New(loadToken(paths.AuthFile))
	items, err := listOwner(cmd, client, args[0], limit, all)
	if err != nil {
		return err
	}
	entries := make([]known.Entry, 0, len(items))
	for _, item := range items {
		entries = append(entries, toKnownEntryFromList(item))
	}
	return writeList(known.Store{Entries: entries}.Sorted())
}

func listOwner(cmd *cobra.Command, client *gist.Client, owner string, limit int, all bool) ([]gist.ListItem, error) {
	if all {
		return client.ListForOwner(cmd.Context(), owner, 100, 0)
	}
	perPage, maxPages := limit, 1
	if perPage > 100 {
		perPage = 100
		maxPages = (limit + perPage - 1) / perPage
	}
	items, err := client.ListForOwner(cmd.Context(), owner, perPage, maxPages)
	if err != nil {
		return nil, err
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func writeList(entries []known.Entry) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "OWNER\tID\tNAME\tDESCRIPTION")
	for _, e := range entries {
		name := e.Alias
		if name == "" && len(e.Filenames) > 0 {
			name = e.Filenames[0]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Owner, e.ID, name, e.Description)
	}
	return w.Flush()
}
