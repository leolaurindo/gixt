package cli

import (
	"fmt"
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
	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	client := gist.New(loadToken(paths.AuthFile))
	id, err := resolveTarget(cmd.Context(), args[0], paths, true)
	if err != nil {
		return err
	}
	alias := mustString(cmd, "as")
	if err := ensureAliasAvailable(paths, id, alias); err != nil {
		return err
	}
	g, err := client.Fetch(cmd.Context(), id, "")
	if err != nil {
		return err
	}
	return saveKnown(paths, func(s *known.Store) {
		known.Upsert(s, toKnownEntry(g, alias))
	})
}

func addOwner(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	return registerOwner(cmd, paths, args[0])
}

func registerOwner(cmd *cobra.Command, paths config.Paths, owner string) error {
	client := gist.New(loadToken(paths.AuthFile))
	items, err := client.ListForOwnerWithProgress(cmd.Context(), owner, 100, 0, func(page, total int) {
		logf("Loading Gists: page %d (%d found)", page, total)
	})
	if err != nil {
		return err
	}
	entries := make([]known.Entry, 0, len(items))
	for _, it := range items {
		entries = append(entries, toKnownEntryFromList(it))
	}
	return saveKnown(paths, func(s *known.Store) {
		s.Entries = replaceOwner(s.Entries, owner, entries)
	})
}

// saveKnown loads the store, applies mutate, and saves it.
func ensureAliasAvailable(paths config.Paths, id, alias string) error {
	if strings.TrimSpace(alias) == "" {
		return nil
	}
	st, err := known.Load(paths.KnownFile)
	if err != nil {
		return err
	}
	for _, entry := range st.Entries {
		if entry.ID != id && strings.EqualFold(entry.Alias, alias) {
			return fmt.Errorf("alias %q is already used by gist %s", alias, entry.ID)
		}
	}
	return nil
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
	aliases := make(map[string]string)
	freshIDs := make(map[string]bool, len(fresh))
	for _, e := range entries {
		if strings.EqualFold(e.Owner, owner) {
			if e.Pin != "" {
				pins[e.ID] = e.Pin
			}
			if e.Alias != "" {
				aliases[e.ID] = e.Alias
			}
		}
	}
	for i := range fresh {
		fresh[i].Pin = pins[fresh[i].ID]
		fresh[i].Alias = aliases[fresh[i].ID]
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
		Owner:       g.Owner.Login,
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
