package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/leolaurindo/gix/internal/alias"
	"github.com/leolaurindo/gix/internal/cache"
	"github.com/leolaurindo/gix/internal/config"
	"github.com/leolaurindo/gix/internal/gist"
	"github.com/leolaurindo/gix/internal/runner"
)

type runOptions struct {
	ref          string
	noCache      bool
	update       bool
	updateIndex  bool
	cacheDir     string
	manifestFile string
	printCmd     bool
	dryRun       bool
	view         bool
	clearCache   bool
	verbose      bool
	userLookup   bool
	descLookup   bool
	userPages    int
	isolate      bool
	cwd          bool
	timeout      time.Duration
	yes          bool
	trustAlways  bool
	trustAll     bool
}

var errViewAborted = errors.New("aborted after view")

func handleRegister(ctx context.Context, id string, ref string, cacheOverride string, update bool) error {
	id = gist.ExtractID(id)
	if id == "" {
		return errors.New("usage: gix register <gist-id|url> [--ref <sha>]")
	}
	paths, err := ensurePaths(cacheOverride)
	if err != nil {
		return err
	}

	fmt.Printf("fetching gist %s via gh...\n", id)
	g, err := gist.Fetch(ctx, id, ref)
	if err != nil {
		return err
	}
	sha := g.LatestVersion()
	if sha == "" {
		sha = ref
	}
	if sha == "" {
		return errors.New("could not determine gist version")
	}
	owner := gist.GuessOwner(g)
	workDir := cache.Dir(paths.CacheDir, id, sha)
	if err := cache.EnsureDir(workDir); err != nil {
		return err
	}
	files, _, err := materializeFiles(g, workDir, update)
	if err != nil {
		return err
	}
	manifest := cache.Manifest{
		GistID:      id,
		SHA:         sha,
		Description: g.Description,
		Owner:       owner,
		Files:       files,
		Source:      g.HTMLURL,
		CreatedAt:   time.Now(),
	}
	if err := cache.SaveManifest(cache.ManifestPath(workDir), manifest); err != nil {
		return err
	}
	fmt.Printf("cached gist %s (%s) at %s\n", cache.Shorten(id), sha, workDir)
	return nil
}

func runWithOptions(ctx context.Context, opts runOptions, identifier string, forwarded []string) error {
	originalCWD, _ := os.Getwd()

	paths, settings, err := ensurePathsAndSettings(opts.cacheDir)
	if err != nil {
		return err
	}

	if opts.trustAll {
		settings.Mode = config.TrustAll
		if err := config.SaveSettings(paths.Settings, settings); err != nil {
			return err
		}
		fmt.Printf("%sall gists trusted (prompt disabled globally).%s\n", clrWarn, clrReset)
	}
	if opts.clearCache {
		fmt.Printf("%sclearing cache at %s...%s\n", clrWarn, paths.CacheDir, clrReset)
		if err := os.RemoveAll(paths.CacheDir); err != nil {
			return err
		}
		if err := config.EnsureDirs(paths); err != nil {
			return err
		}
	} else if settings.CacheMode == config.CacheModeDefault && opts.verbose {
		fmt.Printf("%scache mode 'never': using temp dir; cache untouched%s\n", clrInfo, clrReset)
	}

	aliases, err := alias.Load(paths.AliasFile)
	if err != nil {
		return err
	}

	if opts.updateIndex {
		if err := handleUpdateIndex(ctx); err != nil {
			return err
		}
	}

	resolvedID, owner, resolvedFromIndex, err := resolveIdentifier(ctx, identifier, aliases, paths, opts.userLookup, opts.descLookup, opts.userPages)
	if err != nil {
		return err
	}

	if settings.TrustedGists == nil {
		settings.TrustedGists = map[string]bool{}
	}
	if opts.verbose {
		fmt.Printf("fetching gist %s via gh...\n", resolvedID)
	}
	g, err := gist.Fetch(ctx, resolvedID, opts.ref)
	if err != nil {
		return err
	}
	sha := g.LatestVersion()
	if sha == "" {
		sha = opts.ref
	}
	if sha == "" {
		return errors.New("could not determine gist version")
	}
	if owner == "" {
		owner = gist.GuessOwner(g)
	}

	effectiveNoCache := opts.noCache || settings.CacheMode == config.CacheModeDefault
	workDir, cleanup, err := prepareWorkDir(paths.CacheDir, resolvedID, sha, effectiveNoCache, opts.verbose)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	shouldPromptExecMode := settings.ExecMode == "" && (!resolvedFromIndex || opts.userLookup)
	if shouldPromptExecMode {
		chosen := config.ExecModeIsolate
		if !opts.yes {
			mode, err := promptExecMode()
			if err != nil {
				return err
			}
			chosen = mode
		}
		settings.ExecMode = chosen
		if err := config.SaveSettings(paths.Settings, settings); err != nil {
			return err
		}
	}

	effectiveExecMode, err := decideExecMode(settings.ExecMode, opts.isolate, opts.cwd)
	if err != nil {
		return err
	}

	execDir := originalCWD
	if effectiveExecMode == config.ExecModeIsolate {
		execDir = workDir
	}
	if opts.verbose {
		fmt.Printf("%sexecuting in dir: %s (mode=%s)%s\n", clrInfo, execDir, effectiveExecMode, clrReset)
	}

	files, usedCache, err := materializeFiles(g, workDir, opts.update)
	if err != nil {
		return err
	}

	manifest := cache.Manifest{
		GistID:      resolvedID,
		SHA:         sha,
		Description: g.Description,
		Owner:       owner,
		Files:       files,
		Source:      g.HTMLURL,
		CreatedAt:   time.Now(),
	}
	if !opts.noCache {
		if err := cache.SaveManifest(cache.ManifestPath(workDir), manifest); err != nil {
			return err
		}
	}

	if opts.verbose {
		fmt.Printf("working dir: %s (cache: %v)\n", workDir, usedCache)
	}

	if opts.view {
		if err := viewFiles(manifest, workDir); err != nil && !errors.Is(err, errViewAborted) {
			return err
		}
		return nil
	}

	trusted := trustDecision(ctx, settings, owner, resolvedID, opts.yes || opts.trustAlways)
	if !trusted {
		if err := promptTrust(manifest, workDir); err != nil {
			return err
		}
	}
	if opts.trustAlways {
		settings.TrustedGists[resolvedID] = true
		if err := config.SaveSettings(paths.Settings, settings); err != nil {
			return err
		}
		fmt.Printf("trusted gist %s permanently.\n", resolvedID)
	}

	resolvedArgs := resolveUserArgs(forwarded, originalCWD)
	cmd, envAdd, reason, err := runner.BuildCommand(workDir, opts.manifestFile, files, resolvedArgs, execDir)
	if err != nil {
		return err
	}
	if opts.printCmd || opts.dryRun {
		fmt.Printf("command (%s): %s\n", reason, strings.Join(cmd, " "))
	}
	if opts.dryRun {
		return nil
	}

	runCtx := ctx
	var cancel context.CancelFunc
	if opts.timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, opts.timeout)
		defer cancel()
	}

	return execute(runCtx, execDir, cmd, envAdd)
}

func prepareWorkDir(cacheRoot, gistID, sha string, temp bool, verbose bool) (string, func(), error) {
	if temp {
		tmpDir, err := os.MkdirTemp(cacheRoot, "gix-")
		if err != nil {
			return "", nil, fmt.Errorf("create temp dir: %w", err)
		}
		if verbose {
			fmt.Printf("%srunning from temp dir (no cache persistence): %s%s\n", clrInfo, tmpDir, clrReset)
		}
		if err := cache.EnsureDir(tmpDir); err != nil {
			os.RemoveAll(tmpDir)
			return "", nil, fmt.Errorf("prepare work dir: %w", err)
		}
		return tmpDir, func() { _ = os.RemoveAll(tmpDir) }, nil
	}

	workDir := cache.Dir(cacheRoot, gistID, sha)
	if err := cache.EnsureDir(workDir); err != nil {
		return "", nil, fmt.Errorf("prepare work dir: %w", err)
	}
	return workDir, nil, nil
}

func resolveUserArgs(args []string, originalCWD string) []string {
	resolved := make([]string, 0, len(args))
	for _, a := range args {
		if filepath.IsAbs(a) {
			resolved = append(resolved, a)
			continue
		}
		candidate := filepath.Clean(filepath.Join(originalCWD, a))
		if _, err := os.Stat(candidate); err == nil {
			resolved = append(resolved, candidate)
			continue
		}
		resolved = append(resolved, a)
	}
	return resolved
}

func promptTrust(m cache.Manifest, dir string) error {
	fmt.Printf("%sAbout to run gist %s (owner: %s)%s\n", clrTitle, cache.Shorten(m.GistID), m.Owner, clrReset)
	fmt.Printf("Description: %s\n", strings.TrimSpace(m.Description))
	fmt.Printf("Commit: %s\n", cache.Shorten(m.SHA))
	fmt.Printf("Files: %s\n", strings.Join(m.Files, ", "))
	fmt.Printf("%sTip: manage trust defaults with `gix config-trust --mode mine|all --owner <name>`.%s\n", clrInfo, clrReset)
	fmt.Printf("%sProceed? [y/N/v]: %s", clrPrompt, clrReset)
	var resp string
	fmt.Scanln(&resp)
	resp = strings.ToLower(strings.TrimSpace(resp))
	if resp == "y" || resp == "yes" {
		return nil
	}
	if resp == "v" || resp == "view" {
		return viewFiles(m, dir)
	}
	return errors.New("aborted by user")
}

func viewFiles(m cache.Manifest, dir string) error {
	fmt.Println(colorize("Viewing files (cached):", clrTitle))
	for _, f := range m.Files {
		path := cache.JoinPath(dir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("  %s: %s[error reading: %v]%s\n", f, clrWarn, err, clrReset)
			continue
		}
		fmt.Printf("%s== %s ==%s\n%s\n\n", clrInfo, f, clrReset, colorize(string(data), clrDim))
	}
	return errViewAborted
}

func execute(ctx context.Context, dir string, cmd []string, envAdd map[string]string) error {
	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	c.Dir = dir
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = os.Environ()
	for k, v := range envAdd {
		c.Env = append(c.Env, fmt.Sprintf("%s=%s", k, v))
	}

	return c.Run()
}
