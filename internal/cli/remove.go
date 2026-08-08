package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/known"
)

func newRemoveCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "remove <target>",
		Short: "forget a gist",
		Args:  cobra.MinimumNArgs(1),
		RunE:  removeGist,
	}
	c.AddCommand(&cobra.Command{
		Use:   "owner <login>",
		Short: "forget all of an owner's gists",
		Args:  cobra.ExactArgs(1),
		RunE:  removeOwner,
	})
	return c
}

func removeGist(cmd *cobra.Command, args []string) error {
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
	kept := st.Entries[:0]
	for _, e := range st.Entries {
		if e.ID != id {
			kept = append(kept, e)
		}
	}
	if len(kept) == len(st.Entries) {
		return fmt.Errorf("gist %s is not in the known list", id)
	}
	st.Entries = kept
	return known.Save(paths.KnownFile, st)
}

func removeOwner(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	st, err := known.Load(paths.KnownFile)
	if err != nil {
		return err
	}
	kept := st.Entries[:0]
	for _, e := range st.Entries {
		if !strings.EqualFold(e.Owner, args[0]) {
			kept = append(kept, e)
		}
	}
	if len(kept) == len(st.Entries) {
		return fmt.Errorf("no known gists owned by %s", args[0])
	}
	st.Entries = kept
	return known.Save(paths.KnownFile, st)
}
