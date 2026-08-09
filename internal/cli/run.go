package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/leolaurindo/gixt/internal/cache"
	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/gist"
	"github.com/leolaurindo/gixt/internal/known"
	"github.com/leolaurindo/gixt/internal/runner"
	"github.com/leolaurindo/gixt/internal/trust"
)

type runOptions struct {
	yes     bool
	offline bool
	noCache bool
	python  string
	isolate bool
	ref     string
	view    bool
	dryRun  bool
	add     bool
	as      string
	timeout time.Duration
}

func addRunFlags(fs *pflag.FlagSet) {
	fs.BoolP("yes", "y", false, "skip the trust prompt for this run")
	fs.Bool("offline", false, "run the cached copy without contacting GitHub")
	fs.Bool("no-cache", false, "download to a temp dir and delete it after the run")
	fs.String("python", "", "interpreter to use instead of the shebang (e.g. .venv/bin/python)")
	fs.Bool("isolate", false, "run in the gist work dir instead of the current directory")
	fs.String("ref", "", "run a specific gist revision")
	fs.Bool("view", false, "print the gist files and exit without running")
	fs.Bool("dry-run", false, "resolve and print the command without running it")
	fs.Bool("add", false, "remember the gist after a successful run")
	fs.String("as", "", "custom name to remember it by (implies --add)")
	fs.Duration("timeout", 0, "cancel execution after a duration (e.g. 30s)")
}

func newRunCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "run <target> [-- <args>]",
		Short: "run a gist by id, URL, owner/gist, or known name",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runTarget,
	}
	addRunFlags(c.Flags())
	return c
}

func runFlagsOf(cmd *cobra.Command) *runOptions {
	return &runOptions{
		yes:     mustBool(cmd, "yes"),
		offline: mustBool(cmd, "offline"),
		noCache: mustBool(cmd, "no-cache") || os.Getenv("GIXT_NO_CACHE") != "",
		python:  mustString(cmd, "python"),
		isolate: mustBool(cmd, "isolate"),
		ref:     mustString(cmd, "ref"),
		view:    mustBool(cmd, "view"),
		dryRun:  mustBool(cmd, "dry-run"),
		add:     mustBool(cmd, "add"),
		as:      mustString(cmd, "as"),
		timeout: mustDuration(cmd, "timeout"),
	}
}

func runTarget(cmd *cobra.Command, args []string) error {
	return runWithOptions(cmd.Context(), runFlagsOf(cmd), args[0], args[1:])
}

func runWithOptions(ctx context.Context, o *runOptions, target string, forwarded []string) error {
	originalCWD, _ := os.Getwd()
	if o.offline && o.noCache {
		logf("warning: no-cache mode ignored with --offline")
		o.noCache = false
	}

	paths, err := ensurePaths()
	if err != nil {
		return err
	}

	id, err := resolveTarget(ctx, target, paths)
	if err != nil {
		return err
	}

	pin, err := pinnedRef(paths, id)
	if err != nil {
		return err
	}
	ref := o.ref
	if ref == "" {
		ref = pin
	}

	client := gist.New(loadToken(paths.AuthFile))
	workDir, meta, fromCache, err := obtain(ctx, client, paths, id, ref, o.offline, o.noCache)
	if err != nil {
		return err
	}
	if o.noCache {
		defer os.RemoveAll(workDir)
	}

	if o.view {
		return viewFiles(meta, workDir)
	}

	cmd, reason, err := runner.BuildCommand(workDir, meta.Files, forwarded, o.python)
	if err != nil {
		return err
	}

	if o.dryRun {
		fmt.Printf("command (%s): %s\n", reason, strings.Join(cmd, " "))
		return nil
	}
	if !o.yes {
		store, err := trust.Load(paths.TrustFile)
		if err != nil {
			return err
		}
		if !store.Trusted(id, meta.SHA) {
			if err := promptTrust(meta); err != nil {
				return err
			}
			store.Trust(id, meta.SHA, meta.Owner)
			if err := trust.Save(paths.TrustFile, store); err != nil {
				return err
			}
		}
	}

	execDir := originalCWD
	if o.isolate {
		execDir = workDir
	}

	runCtx := ctx
	var cancel context.CancelFunc
	if o.timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, o.timeout)
		defer cancel()
	}

	err = runner.Execute(runCtx, execDir, cmd)
	if err == nil {
		if !fromCache && !o.noCache {
			if err := cache.Prune(paths.CacheDir, id, meta.SHA, pin); err != nil {
				return err
			}
		}
		if o.add || o.as != "" {
			if err := rememberGist(paths, id, meta, o.as); err != nil {
				return err
			}
		}
	}
	return err
}

// rememberGist records a gist in the known list after a successful run.
func rememberGist(paths config.Paths, id string, m cache.Meta, alias string) error {
	return saveKnown(paths, func(s *known.Store) {
		known.Upsert(s, known.Entry{
			ID:          id,
			Description: m.Description,
			Filenames:   m.Files,
			Alias:       alias,
			UpdatedAt:   time.Now(),
			Owner:       m.Owner,
		})
	})
}

// obtain makes the gist available on disk and returns its work dir + cache metadata.
// With --no-cache it uses a temp dir. With --offline it only touches the cache.
// Otherwise it performs a conditional request: if the server replies 304 the
// cached revision is reused and no files are downloaded.
func obtain(ctx context.Context, client *gist.Client, paths config.Paths, id, ref string, offline, noCache bool) (string, cache.Meta, bool, error) {
	if noCache {
		return obtainTemp(ctx, client, id, ref)
	}
	if offline {
		var dir string
		var m cache.Meta
		var ok bool
		if ref != "" {
			dir, m, ok = cache.Revision(paths.CacheDir, id, ref)
		} else {
			dir, m, ok = cache.Latest(paths.CacheDir, id)
		}
		if !ok {
			if ref != "" {
				return "", cache.Meta{}, false, fmt.Errorf("no cached revision of %s matching %s; run it online first", id, ref)
			}
			return "", cache.Meta{}, false, fmt.Errorf("no cached copy of %s; run once online first", id)
		}
		return dir, m, true, nil
	}

	etag := ""
	cachedDir := ""
	var cached cache.Meta
	if dir, m, ok := cache.Latest(paths.CacheDir, id); ok {
		etag = m.Etag
		cachedDir = dir
		cached = m
	}
	if ref != "" {
		etag = ""
	}

	g, newEtag, notModified, err := client.FetchCached(ctx, id, ref, etag)
	if err != nil {
		return "", cache.Meta{}, false, err
	}

	if notModified {
		if cachedDir == "" {
			return "", cache.Meta{}, false, errors.New("server reported no change but no cached copy exists")
		}
		return cachedDir, cached, true, nil
	}

	sha := g.LatestVersion()
	if sha == "" {
		sha = ref
	}
	if sha == "" {
		sha = cached.SHA
	}
	if sha == "" {
		return "", cache.Meta{}, false, errors.New("could not determine gist revision")
	}

	workDir := cache.Dir(paths.CacheDir, id, sha)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", cache.Meta{}, false, err
	}
	files, reused, err := materializeFiles(ctx, client, g, workDir)
	if err != nil {
		return "", cache.Meta{}, false, err
	}

	m := cache.Meta{
		GistID:      id,
		SHA:         sha,
		Description: g.Description,
		Owner:       g.Owner.Login,
		Files:       files,
		Etag:        newEtag,
		CreatedAt:   time.Now(),
	}
	if reused {
		if lm, err := cache.LoadMeta(cache.MetaPath(workDir)); err == nil {
			m = lm
		}
	} else {
		if err := cache.SaveMeta(cache.MetaPath(workDir), m); err != nil {
			return "", cache.Meta{}, false, err
		}
	}
	return workDir, m, false, nil
}

// obtainTemp downloads a gist into a fresh temp dir. The caller removes the
// dir after the run, so nothing persists.
func obtainTemp(ctx context.Context, client *gist.Client, id, ref string) (string, cache.Meta, bool, error) {
	g, _, _, err := client.FetchCached(ctx, id, ref, "")
	if err != nil {
		return "", cache.Meta{}, false, err
	}
	sha := g.LatestVersion()
	if sha == "" {
		sha = ref
	}
	if sha == "" {
		return "", cache.Meta{}, false, errors.New("could not determine gist revision")
	}
	dir, err := os.MkdirTemp("", "gixt-")
	if err != nil {
		return "", cache.Meta{}, false, err
	}
	files, _, err := materializeFiles(ctx, client, g, dir)
	if err != nil {
		os.RemoveAll(dir)
		return "", cache.Meta{}, false, err
	}
	return dir, cache.Meta{
		GistID:      id,
		SHA:         sha,
		Description: g.Description,
		Owner:       g.Owner.Login,
		Files:       files,
		CreatedAt:   time.Now(),
	}, false, nil
}

func promptTrust(m cache.Meta) error {
	if !isTTY(os.Stdin) {
		return errors.New("refusing to prompt on non-interactive input; pass -y to run untrusted code")
	}
	fmt.Fprintf(os.Stderr, "%sFirst run: gist %s (owner: %s)%s\n", clrTitle, cache.Shorten(m.GistID), m.Owner, clrReset)
	fmt.Fprintf(os.Stderr, "Description: %s\n", strings.TrimSpace(m.Description))
	fmt.Fprintf(os.Stderr, "Commit: %s\n", cache.Shorten(m.SHA))
	fmt.Fprintf(os.Stderr, "Files: %s\n", strings.Join(m.Files, ", "))
	fmt.Fprintf(os.Stderr, "Run it? [y/N]: ")
	line, err := readLine()
	if err != nil {
		return errors.New("aborted by user")
	}
	line = strings.ToLower(line)
	if line == "y" || line == "yes" {
		return nil
	}
	return errors.New("aborted by user")
}

func viewFiles(m cache.Meta, dir string) error {
	for _, f := range m.Files {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", data)
	}
	return nil
}
