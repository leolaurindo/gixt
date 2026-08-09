package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/cache"
	"github.com/leolaurindo/gixt/internal/gist"
	"github.com/leolaurindo/gixt/internal/trust"
)

func newTrustCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "trust",
		Short: "manage trust approvals",
		Args:  cobra.NoArgs,
	}
	c.AddCommand(
		&cobra.Command{
			Use:   "mine",
			Short: "approve the current commit of every gist you own",
			Args:  cobra.NoArgs,
			RunE:  trustMine,
		},
		&cobra.Command{
			Use:   "list",
			Short: "show approved gists and commits",
			Args:  cobra.NoArgs,
			RunE:  trustList,
		},
		&cobra.Command{
			Use:   "remove <target>",
			Short: "revoke an approval",
			Args:  cobra.ExactArgs(1),
			RunE:  trustRemove,
		},
		&cobra.Command{
			Use:   "clear",
			Short: "revoke every approval",
			Args:  cobra.NoArgs,
			RunE:  trustClear,
		},
	)
	return c
}

func trustMine(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	client := gist.New(loadToken(paths.AuthFile))
	me, err := client.CurrentUser(cmd.Context())
	if err != nil {
		return fmt.Errorf("`trust mine` requires authentication: %w (run `gixt auth login`)", err)
	}
	items, err := client.ListForOwner(cmd.Context(), me, 100, 5)
	if err != nil {
		return err
	}
	store, err := trust.Load(paths.TrustFile)
	if err != nil {
		return err
	}
	approved := 0
	for _, it := range items {
		if len(it.History) == 0 || it.History[0].Version == "" {
			continue
		}
		store.Trust(it.ID, it.History[0].Version, it.Owner.Login)
		approved++
	}
	if err := trust.Save(paths.TrustFile, store); err != nil {
		return err
	}
	logf("approved %d gists at their current commits", approved)
	return nil
}

func trustList(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	store, err := trust.Load(paths.TrustFile)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(store.Entries))
	for id := range store.Entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		e := store.Entries[id]
		fmt.Printf("%s\t%s\t%s\n", id, cache.Shorten(e.SHA), e.Owner)
	}
	return nil
}

func trustRemove(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	id, _, err := resolveTarget(cmd.Context(), args[0], paths)
	if err != nil {
		return err
	}
	store, err := trust.Load(paths.TrustFile)
	if err != nil {
		return err
	}
	if _, ok := store.Entries[id]; !ok {
		return fmt.Errorf("gist %s has no trust approval", id)
	}
	delete(store.Entries, id)
	return trust.Save(paths.TrustFile, store)
}

func trustClear(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	store, err := trust.Load(paths.TrustFile)
	if err != nil {
		return err
	}
	store.Entries = map[string]trust.Entry{}
	return trust.Save(paths.TrustFile, store)
}
