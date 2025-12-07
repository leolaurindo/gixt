package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	ucli "github.com/urfave/cli/v2"

	"github.com/leolaurindo/gix/internal/config"
	"github.com/leolaurindo/gix/internal/version"
)

const commandName = "gix"

const (
	clrTitle  = "\033[1;36m"
	clrInfo   = "\033[32m"
	clrWarn   = "\033[33m"
	clrError  = "\033[31m"
	clrPrompt = "\033[35m"
	clrDim    = "\033[2m"
	clrReset  = "\033[0m"
)

func Execute(ctx context.Context, args []string) error {
	app := newApp()
	return app.RunContext(ctx, append([]string{commandName}, args...))
}

func newApp() *ucli.App {
	runFlags := runFlags()

	return &ucli.App{
		Name:      commandName,
		Version:   version.Version,
		Usage:     "run code directly from GitHub gists",
		UsageText: "gix [run flags] <gist-id|url|alias|name> [-- <args to gist>]",
		Flags:     runFlags,
		Action: func(c *ucli.Context) error {
			return runAction(c, c.Args().Slice())
		},
		CommandNotFound: func(c *ucli.Context, name string) {
			args := append([]string{name}, c.Args().Slice()...)
			if err := runAction(c, args); err != nil {
				PrintError(err)
				os.Exit(1)
			}
		},
		Commands: []*ucli.Command{
			{
				Name:      "run",
				Usage:     "run a gist (default)",
				ArgsUsage: "<gist-id|url|alias|name> [-- <args to gist>]",
				Flags:     runFlags,
				Action: func(c *ucli.Context) error {
					return runAction(c, c.Args().Slice())
				},
			},
			{
				Name:      "alias",
				Usage:     "manage aliases",
				ArgsUsage: "add <name> <gist-id> | list | remove <name>",
				Action: func(c *ucli.Context) error {
					return handleAlias(c.Args().Slice())
				},
			},
			{
				Name:  "update-index",
				Usage: "refresh friendly-name index",
				Action: func(c *ucli.Context) error {
					return handleUpdateIndex(c.Context)
				},
			},
			{
				Name:  "index-mine",
				Usage: "refresh friendly-name index for your user",
				Action: func(c *ucli.Context) error {
					return handleIndexMine(c.Context)
				},
			},
			{
				Name:  "clean-cache",
				Usage: "remove the cache directory",
				Flags: []ucli.Flag{
					&ucli.StringFlag{Name: "cache-dir", Usage: "override cache dir"},
				},
				Action: func(c *ucli.Context) error {
					return handleCleanCache(c.String("cache-dir"))
				},
			},
			{
				Name:  "list",
				Usage: "list indexed and cached gists",
				Flags: []ucli.Flag{
					&ucli.BoolFlag{Name: "cache", Aliases: []string{"c"}, Usage: "show cached gists only"},
					&ucli.BoolFlag{Name: "mine", Usage: "filter to gists owned by the authenticated user"},
				},
				Action: func(c *ucli.Context) error {
					return handleList(c.Context, c.Bool("cache"), c.Bool("mine"))
				},
			},
			{
				Name:      "describe",
				Usage:     "show description for a gist",
				ArgsUsage: "<gist-id|url|alias|name|owner/name>",
				Action: func(c *ucli.Context) error {
					if c.Args().Len() == 0 {
						return errors.New("usage: gix describe <gist-id|url|alias|name|owner/name>")
					}
					return handleDescribe(c.Context, c.Args().First())
				},
			},
			{
				Name:  "config-trust",
				Usage: "configure trust policy",
				Flags: []ucli.Flag{
					&ucli.StringFlag{Name: "mode", Usage: "trust mode: never|mine|all"},
					&ucli.StringSliceFlag{Name: "owner", Usage: "trust this owner (repeatable)"},
					&ucli.StringSliceFlag{Name: "trust-owner", Usage: "alias for --owner"},
					&ucli.StringSliceFlag{Name: "remove-owner", Usage: "remove this owner from trusted list (repeatable)"},
					&ucli.StringSliceFlag{Name: "remove-gist", Usage: "remove this gist ID from trusted list (repeatable)"},
					&ucli.BoolFlag{Name: "clear-owners", Usage: "clear trusted owners"},
					&ucli.BoolFlag{Name: "clear-gists", Usage: "clear per-gist trust"},
					&ucli.BoolFlag{Name: "reset", Usage: "clear all trust and return to mode=never"},
					&ucli.BoolFlag{Name: "show", Usage: "show current trust config"},
				},
				Action: func(c *ucli.Context) error {
					owners := append([]string{}, c.StringSlice("owner")...)
					owners = append(owners, c.StringSlice("trust-owner")...)
					return handleConfigTrust(
						c.Context,
						c.String("mode"),
						owners,
						c.StringSlice("remove-owner"),
						c.StringSlice("remove-gist"),
						c.Bool("clear-owners"),
						c.Bool("clear-gists"),
						c.Bool("reset"),
						c.Bool("show"),
					)
				},
			},
			{
				Name:  "clear-index",
				Usage: "remove the index file (cache untouched)",
				Flags: []ucli.Flag{
					&ucli.StringFlag{Name: "cache-dir", Usage: "override cache dir"},
				},
				Action: func(c *ucli.Context) error {
					return handleClearIndex(c.String("cache-dir"))
				},
			},
			{
				Name:      "index-owner",
				Usage:     "index gists for a specific owner",
				ArgsUsage: "--owner <login>",
				Flags: []ucli.Flag{
					&ucli.StringFlag{Name: "owner", Usage: "owner login whose gists to index"},
				},
				Action: func(c *ucli.Context) error {
					owner := c.String("owner")
					if owner == "" && c.Args().Len() > 0 {
						owner = c.Args().First()
					}
					return handleIndexOwner(c.Context, owner)
				},
			},
			{
				Name:  "register",
				Usage: "cache a gist without running it",
				Flags: []ucli.Flag{
					&ucli.StringFlag{Name: "ref", Usage: "pin to specific ref when caching"},
					&ucli.StringFlag{Name: "cache-dir", Usage: "override cache dir"},
					&ucli.BoolFlag{Name: "update", Usage: "force re-download even if cached"},
				},
				Action: func(c *ucli.Context) error {
					if c.Args().Len() == 0 {
						return errors.New("usage: gix register <gist-id|url> [--ref <sha>]")
					}
					return handleRegister(c.Context, c.Args().First(), c.String("ref"), c.String("cache-dir"), c.Bool("update"))
				},
			},
			{
				Name:  "config-cache",
				Usage: "configure cache mode",
				Flags: []ucli.Flag{
					&ucli.StringFlag{Name: "mode", Usage: "cache|never"},
					&ucli.BoolFlag{Name: "show", Usage: "show current cache mode"},
				},
				Action: func(c *ucli.Context) error {
					return handleConfigCache(c.String("mode"), c.Bool("show"))
				},
			},
			{
				Name:  "config-exec",
				Usage: "configure execution directory mode",
				Flags: []ucli.Flag{
					&ucli.StringFlag{Name: "mode", Usage: "isolate|cwd"},
					&ucli.BoolFlag{Name: "show", Usage: "show current execution mode"},
				},
				Action: func(c *ucli.Context) error {
					return handleConfigExec(c.String("mode"), c.Bool("show"))
				},
			},
			{
				Name:  "check-updates",
				Usage: "check if a newer gix release is available",
				Flags: []ucli.Flag{
					&ucli.BoolFlag{Name: "json", Usage: "output machine-readable update info"},
				},
				Action: func(c *ucli.Context) error {
					return handleCheckUpdates(c.Context, c.Bool("json"))
				},
			},
			{
				Name:      "manifest",
				Usage:     "create, edit, or upload gix manifest files",
				ArgsUsage: "",
				Flags: []ucli.Flag{
					&ucli.StringFlag{Name: "name", Usage: "manifest filename", Value: "gix.json"},
					&ucli.BoolFlag{Name: "create", Usage: "create a new manifest locally"},
					&ucli.BoolFlag{Name: "edit", Usage: "edit/overwrite an existing manifest"},
					&ucli.BoolFlag{Name: "upload", Usage: "upload the manifest to a user-owned gist"},
					&ucli.BoolFlag{Name: "view", Usage: "fetch and print a manifest from a gist (no write)"},
					&ucli.StringFlag{Name: "gist", Usage: "gist id or indexed name to upload to"},
					&ucli.StringFlag{Name: "run", Usage: "run command"},
					&ucli.StringSliceFlag{Name: "env", Usage: "env entries (KEY=VAL)", Value: ucli.NewStringSlice()},
					&ucli.StringFlag{Name: "details", Usage: "manifest details/docstring"},
					&ucli.StringFlag{Name: "version", Usage: "manifest version"},
					&ucli.BoolFlag{Name: "force", Usage: "skip overwrite confirmation"},
				},
				Action: func(c *ucli.Context) error {
					opts := manifestOpts{
						name:    c.String("name"),
						create:  c.Bool("create"),
						edit:    c.Bool("edit"),
						upload:  c.Bool("upload"),
						view:    c.Bool("view"),
						gist:    c.String("gist"),
						run:     c.String("run"),
						env:     c.StringSlice("env"),
						details: c.String("details"),
						version: c.String("version"),
						force:   c.Bool("force"),
					}
					if err := applyManifestArgs(c.Args().Slice(), &opts); err != nil {
						return err
					}
					return handleManifest(c.Context, opts)
				},
			},
			{
				Name:      "index-description",
				Usage:     "manage local description overrides for indexed gists",
				ArgsUsage: "list | add <id|name> <desc> | remove <id|name>",
				Action: func(c *ucli.Context) error {
					return handleDescOverride(c.Context, c.Args().Slice())
				},
			},
		},
	}
}

func runAction(c *ucli.Context, args []string) error {
	if len(args) == 0 {
		_ = ucli.ShowAppHelp(c)
		return errors.New("missing gist identifier")
	}

	opts := runOptions{
		ref:          c.String("ref"),
		noCache:      c.Bool("no-cache"),
		update:       c.Bool("update"),
		updateIndex:  c.Bool("update-index"),
		cacheDir:     c.String("cache-dir"),
		manifestFile: c.String("manifest"),
		printCmd:     c.Bool("print-cmd"),
		dryRun:       c.Bool("dry-run"),
		view:         c.Bool("view"),
		clearCache:   c.Bool("clear-cache"),
		verbose:      c.Bool("verbose"),
		userLookup:   c.Bool("user-lookup"),
		descLookup:   c.Bool("desc-lookup"),
		userPages:    c.Int("user-pages"),
		isolate:      c.Bool("isolate"),
		cwd:          c.Bool("cwd"),
		timeout:      c.Duration("timeout"),
		yes:          c.Bool("yes"),
		trustAlways:  c.Bool("trust-always"),
		trustAll:     c.Bool("trust-all"),
	}

	opts.userPages = normalizeUserPages(opts.userPages)

	identifier := args[0]
	forwarded := args[1:]
	return runWithOptions(c.Context, opts, identifier, forwarded)
}

func runFlags() []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{Name: "ref", Usage: "pin to a specific gist ref"},
		&ucli.BoolFlag{Name: "no-cache", Usage: "use temp dir and delete after run"},
		&ucli.BoolFlag{Name: "update", Usage: "force re-download even if cached"},
		&ucli.BoolFlag{Name: "update-index", Usage: "refresh friendly-name index before running"},
		&ucli.StringFlag{Name: "cache-dir", Usage: "override cache directory"},
		&ucli.StringFlag{Name: "manifest", Value: "gix.json", Usage: "run manifest filename"},
		&ucli.BoolFlag{Name: "print-cmd", Usage: "print resolved command"},
		&ucli.BoolFlag{Name: "dry-run", Usage: "resolve but do not execute"},
		&ucli.BoolFlag{Name: "view", Usage: "print gist text content and exit without running"},
		&ucli.BoolFlag{Name: "clear-cache", Usage: "clear cache directory before running"},
		&ucli.BoolFlag{Name: "verbose", Usage: "verbose logging for run command"},
		&ucli.BoolFlag{Name: "user-lookup", Aliases: []string{"u"}, Usage: "enable live user/name lookup without index"},
		&ucli.IntFlag{Name: "user-pages", Aliases: []string{"p"}, Value: 2, Usage: "pages (100 per page) to scan for user lookup"},
		&ucli.BoolFlag{Name: "desc-lookup", Usage: "allow matching gist descriptions when resolving names"},
		&ucli.BoolFlag{Name: "isolate", Usage: "run in an isolated work dir instead of current directory"},
		&ucli.BoolFlag{Name: "cwd", Aliases: []string{"here"}, Usage: "run in current working directory (overrides execution mode)"},
		&ucli.DurationFlag{Name: "timeout", Usage: "timeout for gist execution (e.g. 30s, 2m)"},
		&ucli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "skip trust prompt"},
		&ucli.BoolFlag{Name: "trust-always", Usage: "trust this gist permanently"},
		&ucli.BoolFlag{Name: "trust-all", Usage: "trust all gists permanently"},
	}
}

func colorize(s, code string) string {
	if code == "" {
		return s
	}
	return code + s + clrReset
}

func PrintError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "%serror: %v%s\n", clrError, err, clrReset)
}

func discoverPaths(cacheOverride string) (config.Paths, error) {
	return config.Discover(cacheOverride)
}

func ensurePaths(cacheOverride string) (config.Paths, error) {
	paths, err := discoverPaths(cacheOverride)
	if err != nil {
		return config.Paths{}, err
	}
	if err := config.EnsureDirs(paths); err != nil {
		return config.Paths{}, err
	}
	return paths, nil
}

func ensurePathsAndSettings(cacheOverride string) (config.Paths, config.Settings, error) {
	paths, err := ensurePaths(cacheOverride)
	if err != nil {
		return config.Paths{}, config.Settings{}, err
	}
	settings, err := config.LoadSettings(paths.Settings)
	if err != nil {
		return config.Paths{}, config.Settings{}, err
	}
	return paths, settings, nil
}
