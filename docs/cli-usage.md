# CLI Usage Guide

`gixt` retrieves Gist artifacts with `cat` and executes code explicitly with `run`.

## Command forms

```sh
gixt cat <target> [<target> ...]
gixt print <target> [<target> ...]
gixt run <target> [--entry <filename>] [-- <args>]
gixt <target>                         # shorthand for gixt cat <target>
```

The `print` command is an alias for `cat`. Bare targets retrieve content; execution always requires `gixt run`.

## Targets and entries

A target identifies a Gist. An entry identifies one file in that Gist.

Targets resolve in this order:

1. Gist ID or URL;
2. known alias;
3. `owner/name` from the known store, then live owner lookup;
4. exact full filename among known Gists;
5. filename stem among known Gists.

An alias identifies a Gist, never a file. An exact filename target carries the requested filename into selection. A stem identifies the Gist but does not force a particular variant. Ambiguous targets remain errors and show disambiguators.

## `cat`

```sh
gixt cat review-prompt | agent
gixt cat review.md style.md | agent
gixt cat review.md --entry review.md
```

Each target is resolved independently and processed in command-line order. A Gist target emits all files in lexical order. An exact filename target emits that file. `--entry` is valid only with one target.

`cat` writes selected bytes directly to stdout. It adds no headers, separators, or newlines, and it never executes code or checks trust. Diagnostics and warnings go to stderr, so binary output and shell pipelines remain safe.

Flags:

- `--offline` — use only the cached revision.
- `--no-cache` — use a temporary directory and remove it afterwards.
- `--ref <sha>` — select a specific revision, overriding a pin.
- `--entry <filename>` / `-e` — select one exact file.

## `run`

```sh
gixt run cleanup-script -- --dry-run
gixt run --entry cleanup.sh cleanup-script
gixt run --python .venv/bin/python script.py
```

Entry selection order is:

1. explicit `--entry`;
2. exact filename inferred from the target;
3. `main.*`;
4. `index.*`;
5. a platform shell variant when candidates share a basename;
6. lexical fallback.

After selection, execution resolution is Python override, shebang, then extension mapping. Trust approval applies only to `run`, keyed by Gist ID and revision.

`run --view` remains temporarily available for compatibility but is deprecated. Use `gixt cat` for retrieval.

## Resolution examples

```sh
# Exact filename target: selects that file
gixt run review.sh

# Stem target: resolves the Gist, then uses normal fallback selection
gixt run review

# Alias identifies the Gist; --entry identifies the file
gixt run --entry review.sh review-prompt
```

Use `gixt add <target> --as <name>` to remember an alias. Aliases are globally unique; use `--entry` to select files within the aliased Gist.

Forget remembered entries with `gixt remove <target>`, or remove an owner's local entries with `gixt remove --owner <owner> --yes`.

List remembered entries with `gixt list`, or inspect an owner's remote Gists with `gixt list <owner> --limit 30`. Add `--verbose` to show full IDs, update dates, and descriptions wrapped to the available table width.

## Create and update gists

```sh
# Create a secret gist from local text or script files
gixt gist create prompt.md --description "Review prompt"
gixt gist create main.py helpers.py

# Public creation is explicit
gixt gist create main.py --public

# Add or replace files in an existing gist
gixt gist update <target> prompt.md
gixt gist update <target> main.py helpers.py
```

Both commands require authentication through `gixt auth login` or an existing token (see `gixt auth --help`). New gists are secret by default, not private: anyone with the URL can access them. Never upload credentials or secrets.

Local basenames become gist filenames. Inputs must be non-empty UTF-8 text files; directories and duplicate basenames are rejected. Text and runnable scripts use the same upload mechanism. Use descriptive `.md`/`.txt` names for text, and appropriate extensions or shebangs for scripts. There is no stdin, interactive editor, individual-file deletion, or renaming support.

`update` accepts normal gixt targets and adds or replaces only supplied files, preserving all other remote files and the gist's description and visibility. A filename target identifies the gist, not a destination filename: local basenames still determine which files are written. Updates always modify the live gist, even when its local alias is pinned.

Both commands print the resulting gist URL to stdout and refresh the known-gist metadata. Existing aliases and pins are preserved; no alias or trust approval is automatically granted. A pinned gist continues to retrieve/run its pinned revision, and running an updated unpinned gist requires approval of the new revision.

## Delete a remote gist

```sh
gixt gist delete <target>          # Confirm interactively
gixt gist delete <target> --yes    # Explicit non-interactive confirmation
```

Unlike `gixt remove`, which only forgets a local registration, `gist delete` permanently deletes the entire gist from GitHub. It requires authentication and accepts normal gixt targets. A filename target or pinned alias still deletes the entire live gist, not a single file or revision.

Without `--yes`, deletion requires interactive confirmation and refuses non-interactive input. After GitHub confirms deletion, gixt forgets the local registration (including aliases and pins) and prints the gist URL. API failures leave local registrations unchanged. Cached files and trust records are retained; use the cache and trust commands to clear them if needed.

## Self-management and updates

```sh
gixt self version
gixt self update-check
gixt self update
```

`update-check` compares your version with the latest stable GitHub release.
`update` downloads the matching platform archive, verifies its SHA-256 checksum
from that release, and replaces the executable you actually ran. Symlinks are
preserved; their resolved executable is updated. Existing versioned installations
are not downgraded, and config, cache, and trust data are untouched.

- **Script, manual download, Go-installed, and source-built binaries:** update
  directly to the official prebuilt release. Custom Go build settings are not
  preserved. To keep a Go-based build, use
  `go install github.com/leolaurindo/gixt/cmd/gixt@latest` instead.
- **Homebrew:** detect the resolved Cellar path and verify it against
  `brew --prefix gixt`, then print `brew upgrade gixt`. No upgrade is run and
  managed binaries are never overwritten. The tap's version can lag GitHub;
  `update-check` still reports the GitHub version.
- **Other package managers:** automatic detection is not supported yet. Use
  your package manager's update command rather than `self update`.

Updates require write access to the executable's directory; gixt does not elevate
privileges. Downloads are bounded and failures before replacement leave the
current executable intact. Unix replacement is atomic. Windows uses a backup
and rolls back if installation fails; a `.gixt.exe.old` backup can remain while
the old executable is running and is removed by the next update.

Concurrent updates are rejected. If an interrupted update leaves an
`.update-lock` file, remove the lock path reported by the error only after
confirming no update is running.

## Caching, pins, and trust

- Online retrieval uses the existing ETag/cache behavior.
- `--offline` never contacts GitHub and requires a cached revision.
- A pin is used when `--ref` is absent.
- `cat` does not prompt or modify trust.
- `run` prompts for an unapproved revision unless `--yes` is supplied.

Inspect metadata and files with:

```sh
gixt gist show <target>
```

Remember Gists with:

```sh
gixt add <target> --as <name>
gixt add owner <owner>
gixt add mine
```
