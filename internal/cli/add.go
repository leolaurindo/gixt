package cli

import (
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/gist"
	"github.com/leolaurindo/gixt/internal/known"
)

func newAddCmd() *cobra.Command {
	add := &cobra.Command{
		Use:   "add <id|url|owner/gist>",
		Short: "remember a gist so you can run it by name",
		Args:  cobra.MinimumNArgs(1),
		RunE:  addGist,
	}
	add.Flags().String("as", "", "custom name to run it by")

	owner := &cobra.Command{
		Use:   "owner <login>",
		Short: "remember all of an owner's gists",
		Args:  cobra.ExactArgs(1),
		RunE:  addOwner,
	}
	add.AddCommand(owner)
	return add
}

func addGist(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	client := gist.New(loadToken(paths.AuthFile))
	id, _, err := resolveTarget(cmd.Context(), args[0], paths)
	if err != nil {
		return err
	}
	g, err := client.Fetch(cmd.Context(), id, "")
	if err != nil {
		return err
	}
	return saveKnown(paths, func(s *known.Store) {
		known.Upsert(s, toKnownEntry(g, mustString(cmd, "as")))
	})
}

func addOwner(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	client := gist.New(loadToken(paths.AuthFile))
	items, err := client.ListForOwner(cmd.Context(), args[0], 100, 5)
	if err != nil {
		return err
	}
	entries := make([]known.Entry, 0, len(items))
	for _, it := range items {
		entries = append(entries, toKnownEntryFromList(it))
	}
	return saveKnown(paths, func(s *known.Store) {
		s.Entries = replaceOwner(s.Entries, args[0], entries)
	})
}

// saveKnown loads the store, applies mutate, and saves it.
func saveKnown(paths config.Paths, mutate func(*known.Store)) error {
	st, err := known.Load(paths.KnownFile)
	if err != nil {
		return err
	}
	st.GeneratedAt = time.Now()
	mutate(&st)
	return known.Save(paths.KnownFile, st)
}

func replaceOwner(entries []known.Entry, owner string, fresh []known.Entry) []known.Entry {
	pins := make(map[string]string)
	freshIDs := make(map[string]bool, len(fresh))
	for _, e := range entries {
		if strings.EqualFold(e.Owner, owner) && e.Pin != "" {
			pins[e.ID] = e.Pin
		}
	}
	for i := range fresh {
		fresh[i].Pin = pins[fresh[i].ID]
		freshIDs[fresh[i].ID] = true
	}
	var kept []known.Entry
	for _, e := range entries {
		if !strings.EqualFold(e.Owner, owner) || (e.Pin != "" && !freshIDs[e.ID]) {
			kept = append(kept, e)
		}
	}
	return append(kept, fresh...)
}

func toKnownEntry(g gist.Gist, alias string) known.Entry {
	return known.Entry{
		ID:          g.ID,
		Description: g.Description,
		Filenames:   mapFileNames(g.Files),
		Alias:       alias,
		UpdatedAt:   g.UpdatedAt,
		Owner:       gist.GuessOwner(g),
	}
}

func toKnownEntryFromList(it gist.ListItem) known.Entry {
	return known.Entry{
		ID:          it.ID,
		Description: it.Description,
		Filenames:   mapFileNames(it.Files),
		UpdatedAt:   it.UpdatedAt,
		Owner:       it.Owner.Login,
	}
}

func mapFileNames(m map[string]gist.File) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
