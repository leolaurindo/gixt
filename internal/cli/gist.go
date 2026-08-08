package cli

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/gist"
	"github.com/leolaurindo/gixt/internal/known"
)

func newGistCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "gist",
		Short: "inspect and modify gists",
		Args:  cobra.NoArgs,
	}

	clone := &cobra.Command{
		Use:   "clone <target>",
		Short: "clone a gist into a local git repo",
		Args:  cobra.ExactArgs(1),
		RunE:  gistClone,
	}
	clone.Flags().String("dir", "", "destination directory (default: gist id)")

	fork := &cobra.Command{
		Use:   "fork <target>",
		Short: "copy a gist to a new gist owned by you",
		Args:  cobra.ExactArgs(1),
		RunE:  gistFork,
	}
	fork.Flags().Bool("public", false, "create the fork as public (default: private)")
	fork.Flags().String("description", "", "description for the new gist")

	c.AddCommand(
		&cobra.Command{
			Use:   "show <target>",
			Short: "show a gist's metadata and files",
			Args:  cobra.ExactArgs(1),
			RunE:  gistShow,
		},
		&cobra.Command{
			Use:   "set-description <target> <text>",
			Short: "update the description of a gist you own",
			Args:  cobra.ExactArgs(2),
			RunE:  gistSetDescription,
		},
		clone,
		fork,
	)
	return c
}

func gistShow(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	id, _, err := resolveTarget(cmd.Context(), args[0], paths)
	if err != nil {
		return err
	}
	client := gist.New(loadToken(paths.AuthFile))
	g, err := client.Fetch(cmd.Context(), id, "")
	if err != nil {
		return err
	}
	fmt.Printf("ID:          %s\n", g.ID)
	fmt.Printf("Owner:       %s\n", gist.GuessOwner(g))
	fmt.Printf("Description: %s\n", g.Description)
	fmt.Printf("Updated:     %s\n", g.UpdatedAt.Format(time.RFC3339))
	fmt.Printf("Revision:    %s\n", g.LatestVersion())
	fmt.Printf("URL:         %s\n", g.HTMLURL)
	fmt.Println("Files:")
	for name := range g.Files {
		fmt.Printf("  %s\n", name)
	}
	return nil
}

func gistSetDescription(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	id, _, err := resolveTarget(cmd.Context(), args[0], paths)
	if err != nil {
		return err
	}
	client := gist.New(loadToken(paths.AuthFile))
	if _, err := client.UpdateDescription(cmd.Context(), id, args[1]); err != nil {
		return err
	}
	logf("updated description of gist %s", id)
	return nil
}

func gistClone(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	id, _, err := resolveTarget(cmd.Context(), args[0], paths)
	if err != nil {
		return err
	}
	dir := mustString(cmd, "dir")
	if dir == "" {
		dir = id
	}
	git, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git not found: %w", err)
	}
	clone := exec.CommandContext(cmd.Context(), git, "clone", "https://gist.github.com/"+id+".git", dir)
	clone.Stdin = os.Stdin
	clone.Stdout = os.Stdout
	clone.Stderr = os.Stderr
	return clone.Run()
}

func gistFork(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	id, _, err := resolveTarget(cmd.Context(), args[0], paths)
	if err != nil {
		return err
	}
	client := gist.New(loadToken(paths.AuthFile))
	g, err := client.Fetch(cmd.Context(), id, "")
	if err != nil {
		return err
	}
	files, err := gistFileContents(cmd, client, g)
	if err != nil {
		return err
	}
	description := mustString(cmd, "description")
	if description == "" {
		description = g.Description
	}
	public, _ := cmd.Flags().GetBool("public")
	created, err := client.Create(cmd.Context(), files, description, public)
	if err != nil {
		return err
	}
	if err := saveKnown(paths, func(s *known.Store) { known.Upsert(s, toKnownEntry(created, "")) }); err != nil {
		return err
	}
	logf("forked gist %s to %s", id, created.ID)
	return nil
}

func gistFileContents(cmd *cobra.Command, client *gist.Client, g gist.Gist) (map[string]string, error) {
	out := map[string]string{}
	for name, f := range g.Files {
		if f.Content != "" && !f.Truncated {
			out[name] = f.Content
			continue
		}
		data, err := client.Download(cmd.Context(), f.RawURL)
		if err != nil {
			return nil, err
		}
		out[name] = string(data)
	}
	return out, nil
}
