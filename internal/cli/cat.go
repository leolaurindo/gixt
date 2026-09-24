package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/gist"
)

type catOptions struct {
	offline bool
	noCache bool
	ref     string
	entry   string
}

func addCatFlags(fs *pflag.FlagSet) {
	fs.Bool("offline", false, "read the cached copy without contacting GitHub")
	fs.Bool("no-cache", false, "download to a temporary directory and remove it afterwards")
	fs.String("ref", "", "read a specific gist revision")
	fs.StringP("entry", "e", "", "read this exact gist file")
}

func newCatCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "cat <target> [<target> ...]",
		Aliases: []string{"print"},
		Short:   "write gist files to stdout",
		Args:    cobra.MinimumNArgs(1),
		RunE:    catTargets,
	}
	addCatFlags(c.Flags())
	return c
}

func catFlagsOf(cmd *cobra.Command) catOptions {
	return catOptions{
		offline: mustBool(cmd, "offline"),
		noCache: mustBool(cmd, "no-cache") || os.Getenv("GIXT_NO_CACHE") != "",
		ref:     mustString(cmd, "ref"),
		entry:   mustString(cmd, "entry"),
	}
}

func catTargets(cmd *cobra.Command, args []string) error {
	o := catFlagsOf(cmd)
	if o.entry != "" && len(args) != 1 {
		return fmt.Errorf("--entry requires exactly one target")
	}
	if o.offline && o.noCache {
		logf("warning: no-cache mode ignored with --offline")
		o.noCache = false
	}

	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	client := gist.New(loadToken(paths.AuthFile))
	for _, target := range args {
		if err := catOne(cmd, paths, client, target, o); err != nil {
			return err
		}
	}
	return nil
}

func catOne(cmd *cobra.Command, paths config.Paths, client *gist.Client, target string, o catOptions) error {
	artifact, err := acquireArtifact(cmd.Context(), paths, client, target, o.offline, o.noCache, o.ref)
	if err != nil {
		return err
	}
	defer artifact.close()

	requested := o.entry
	if requested == "" {
		requested = artifact.resolved.RequestedFile
	}
	if requested != "" {
		entry, err := SelectEntry(artifact.meta.Files, requested)
		if err != nil {
			return err
		}
		return writeArtifactFile(artifact.workDir, entry)
	}

	files := append([]string(nil), artifact.meta.Files...)
	sort.Strings(files)
	for _, filename := range files {
		if err := writeArtifactFile(artifact.workDir, filename); err != nil {
			return err
		}
	}
	return nil
}

func writeArtifactFile(dir, filename string) error {
	file, err := os.Open(filepath.Join(dir, filename))
	if err != nil {
		return fmt.Errorf("read entry %q: %w", filename, err)
	}
	defer file.Close()
	if _, err := io.Copy(os.Stdout, file); err != nil {
		return fmt.Errorf("write entry %q: %w", filename, err)
	}
	return nil
}
