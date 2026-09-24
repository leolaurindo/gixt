package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/known"
)

func newRemoveCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "remove <target>",
		Short: "forget a gist or an owner's gists",
		Args: func(cmd *cobra.Command, args []string) error {
			owner, _ := cmd.Flags().GetString("owner")
			if cmd.Flags().Changed("owner") {
				if strings.TrimSpace(owner) == "" {
					return errors.New("--owner requires an owner")
				}
				if len(args) != 0 {
					return errors.New("--owner and a target are mutually exclusive")
				}
				return nil
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: removeCommand,
	}
	c.Flags().String("owner", "", "forget all locally remembered gists for this owner")
	c.Flags().Bool("yes", false, "confirm owner-wide removal without prompting")
	return c
}

func removeCommand(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("owner") {
		owner, _ := cmd.Flags().GetString("owner")
		return removeOwner(cmd, owner)
	}
	return removeGist(cmd, args[0])
}

func removeGist(cmd *cobra.Command, target string) error {
	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	id, err := resolveTarget(cmd.Context(), target, paths, true)
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

func removeOwner(cmd *cobra.Command, owner string) error {
	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	st, err := known.Load(paths.KnownFile)
	if err != nil {
		return err
	}
	kept := st.Entries[:0]
	for _, e := range st.Entries {
		if !strings.EqualFold(e.Owner, owner) {
			kept = append(kept, e)
		}
	}
	if len(kept) == len(st.Entries) {
		return fmt.Errorf("no known gists owned by %s", owner)
	}

	if !mustBool(cmd, "yes") {
		if !isTTY(os.Stdin) {
			return errors.New("owner removal requires confirmation; use --yes in non-interactive mode")
		}
		fmt.Fprintf(os.Stderr, "Remove all locally remembered gists, aliases, and pins for owner %q? [y/N] ", owner)
		answer, err := readLine()
		if err != nil {
			return fmt.Errorf("read confirmation: %w", err)
		}
		if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
			return errors.New("removal cancelled")
		}
	}

	st.Entries = kept
	return known.Save(paths.KnownFile, st)
}
