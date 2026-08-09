# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `gixt run --no-cache` (and `GIXT_NO_CACHE`) downloads to a temp dir and deletes it after the run.
- `gixt pin <target> [<sha>]` validates and pins a gist revision. Metadata updates preserve pins, and cache pruning retains pinned contents for offline use.

### Changed
- Replaced the `config` command and automatic owner trust with pure commit-scoped TOFU. `gixt trust mine` explicitly snapshots exact current revisions; later changes still prompt.
- `go install` builds now report the module version (via `debug.ReadBuildInfo`) instead of `dev`.

### Fixed
- `--offline` now takes precedence over no-cache mode without contacting GitHub.
- `--view` and `--dry-run` no longer change trust approvals.
- Pins survive add/refresh operations, explicit revisions are validated, and offline runs select the exact pinned cache entry.

## [0.2.1] - 2026-08-08

### Fixed
- Rebuilt release binaries with Go 1.24 (Go 1.21 produced macOS binaries dyld rejects).

## [0.2.0] - 2026-08-08

### Added

- **A hierarchical command tree** built on `cobra`, replacing the flat, flag-heavy
  `urfave/cli` surface. Commands are grouped by intent: `run`, `add`, `remove`,
  `list`, `gist`, `cache`, `config`, `auth`, `self`.
- **`gixt run <target>` / `gixt <target>`** resolves gists by ID, URL, `owner/gist`
  (with a live GitHub lookup when not remembered locally), or a friendly name.
- **`gixt add`** remembers gists so you can run them by name:
  - `gixt add <id|url|owner/gist>` — remember one gist under its file basename.
  - `gixt add <target> --as <name>` — remember it with a custom name.
  - `gixt add owner <login>` — remember all of an owner's gists.
- **`gixt remove`** forgets gists:
  - `gixt remove <target>` — forget a specific gist.
  - `gixt remove owner <login>` — forget all of an owner's gists.
- **`gixt list`** — a pretty, aligned table of remembered gists
  (`OWNER`, `ID`, `NAME`, `DESCRIPTION`), plus `gixt list refresh` and `gixt list clear`.
- **`gixt auth login|status|logout`** for GitHub authentication, storing a
  personal access token instead of depending on `gh`.
- **`gixt config get|set|unset`** for settings (currently `trust.mine`).
- **`gixt self version`** and **`gixt self update-check`**.
- **Direct GitHub HTTP transport** with optional ETag/`If-None-Match` caching:
  unchanged gists short-circuit with `304` and reuse the cache with zero downloads.
- **`--offline`** runs the cached copy without any network contact.
- **`--python <interpreter>`** overrides the shebang (e.g. `.venv/bin/python`),
  so gists can use a project's virtualenv.
- **`--ref <sha>`** pins a specific gist revision.
- **`--add` / `--as`** on `run` to remember a gist after a successful run.
- **`--timeout <duration>`** to bound execution.
- **Exit-code propagation**: the gist's exit status becomes gixt's.
- **Clean stdio separation**: gist output goes to stdout, gixt logs and prompts to
  stderr, so `gixt <target> | sh` and other pipelines work as expected.
- **Non-interactive safety**: gixt refuses to prompt when stdin is not a terminal
  and tells you to pass `-y`, instead of silently swallowing piped input.
- **Colors only on a TTY**, honoring `NO_COLOR`.

### Changed

- **Framework**: `urfave/cli/v2` → `cobra`.
- **`gh` is no longer required.** All GitHub access is direct HTTP; token source is
  `GITHUB_TOKEN`, the token stored by `gixt auth login`, or (as an optional
  convenience) `gh auth token`. Public gists work with no token at all.
- **Trust model** is now trust-on-first-use (TOFU) keyed by commit: the exact
  revision you approved runs silently, and a changed revision prompts again.
  The old `never`/`mine`/`all` modes, trusted-owner list, and trusted-gist list
  are gone. Gists you own are auto-trusted.
- **Caching is on by default** (previously opt-in), content-addressed by
  `cache/<gist-id>/<sha>`, and pruned to the latest revision per gist after each
  successful run.
- **Default execution directory** is now your current directory (matching
  `npx`/`uvx`), with `--isolate` opting into the sandboxed work dir. Previously
  the default was isolated with a first-run prompt.
- **Aliases and the index are merged** into a single `known.json` store: one entry
  per gist (id, owner, description, filenames, optional alias).
- **Name resolution order**: gist ID/URL → known name/alias → `owner/gist`
  (live lookup).

### Removed

- **The entire `manifest` family and `gixt.json`** support. Command resolution is
  now `--python` override → shebang → extension map.
- **The `index` and `alias` families**, replaced by the unified `known.json` store
  and the `add`/`remove`/`list` commands.
- **Top-level commands**: `register`, `clean-cache`, `clear-index`, `remove`,
  `update-index`, `index-mine`, `index-owner`, `config-trust`, `config-cache`,
  `config-exec`, `describe`, `set-description`, `clone`, `fork`, `check-updates`,
  and the old `list`.
- **Run flags**: `--manifest`, `--ignore-manifest`, `--no-cache`, `--update`,
  `--update-index`, `--clear-cache`, `--user-lookup/-u`, `--user-pages/-p`,
  `--desc-lookup`, `--cwd`/`--here`, `--print-cmd`, `--trust-always`,
  `--trust-all`, `--verbose`.
- **Dependencies**: `urfave/cli/v2` (replaced by `cobra`).

### Fixed

- gixt no longer pollutes stdout with its own logs and prompts; informational
  output moved to stderr.
- Non-interactive runs no longer hang waiting for input: the trust prompt is
  refused on non-terminals and tells you to pass `-y`.
- The child process's exit code is now propagated instead of always exiting 1.

### Security

- Code from untrusted sources is only ever run after a per-commit approval.
  Re-running an unchanged revision cannot re-approve new content; a changed
  revision always requires a fresh prompt.
- Gist file paths are sanitized before materialization (no `..` traversal,
  no absolute or drive-prefixed paths).
