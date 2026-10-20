package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/gist"
	"github.com/leolaurindo/gixt/internal/known"
)

func gistDelete(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	id, err := resolveTarget(cmd.Context(), args[0], paths, true)
	if err != nil {
		return err
	}
	url := "https://gist.github.com/" + id
	if !mustBool(cmd, "yes") {
		if !isTTY(os.Stdin) {
			return errors.New("remote deletion requires confirmation; use --yes in non-interactive mode")
		}
		fmt.Fprintf(os.Stderr, "Permanently delete gist %q (%s) from GitHub? [y/N] ", args[0], url)
		answer, err := readLine()
		if err != nil {
			return fmt.Errorf("read confirmation: %w", err)
		}
		if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
			return errors.New("deletion cancelled")
		}
	}
	client := gist.New(loadToken(paths.AuthFile))
	if err := client.Delete(cmd.Context(), id); err != nil {
		return err
	}
	if err := saveKnown(paths, func(s *known.Store) {
		kept := s.Entries[:0]
		for _, entry := range s.Entries {
			if entry.ID != id {
				kept = append(kept, entry)
			}
		}
		s.Entries = kept
	}); err != nil {
		return fmt.Errorf("gist %s deleted from GitHub, but local cleanup failed: %w", url, err)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), url)
	return err
}
