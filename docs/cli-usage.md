# CLI Usage Guide

How to run gists, resolve identifiers, control caching, and understand what gixt actually executes.

## Command form and argument forwarding

```text
gixt run <gist-id|url|alias|name|owner/name> [-- <args to gist>]
gixt <gist-id|url|alias|name|owner/name> [-- <args to gist>]   # shorthand
```

- Flags before the target configure gixt itself.
- After the literal `--`, every argument is forwarded to the gist unchanged.
- Any additional arguments after the target are also forwarded as gist arguments.
- Show the version with `gixt self version` or `gixt --version`.

## Identifier resolution

Accepted forms, in order:

1. Gist ID or URL (last path segment is extracted).
2. Known name or alias (from `known.json`).
3. `owner/gist` — from the known store, falling back to a live lookup of that owner's gists.

When multiple known entries match the same name, gixt prefers your platform's shell variant (`.bat/.cmd/.ps1` on Windows, `.sh/.bash/.zsh` elsewhere) when all candidates are shell scripts; otherwise the match is ambiguous and gixt reports the candidates. Disambiguate with `owner/name`, `name.ext`, or an alias (`--as`).

## What happens during a run

1. The target is resolved to a gist ID.
2. If the gist is pinned and no `--ref` was given, the pinned revision is used.
3. gixt contacts GitHub unless `--offline`. Normal runs reuse an unchanged ETag cache; `--no-cache` uses a temporary directory instead. If both are set, offline wins and gixt warns that no-cache mode was ignored.
4. Files are materialized with path sanitization into `cache/<id>/<sha>`, or into the temporary directory in no-cache mode.
5. `--view` prints the files and exits. Otherwise gixt resolves the command from `--python`, the chosen file's shebang, or its extension.
6. `--dry-run` prints that command and exits. Inspection does not prompt for or persist trust.
7. Before execution, gixt checks trust. An unapproved revision prompts unless `-y` was given.
8. The command runs in your current directory, or the gist work directory with `--isolate`. Stdio and the child exit status are propagated; `--timeout` cancels long runs.
9. After success, old cached revisions are pruned while retaining the executed and pinned revisions. `--add` or `--as` also remembers the gist without changing its pin.

Inspection shortcuts:

- `--view` prints the gist files and exits.
- `--dry-run` resolves everything, prints the command, and exits before execution.

## Run flags

- `-y`, `--yes`: skip the trust prompt for this run.
- `--offline`: run the exact requested or pinned cached revision without contacting GitHub. It takes precedence over no-cache mode.
- `--no-cache`: download to a temp dir and delete it after the run (also `GIXT_NO_CACHE`). This does not disable config or trust-store writes during an executing run.
- `--python <interpreter>`: override the shebang (e.g. `.venv/bin/python`).
- `--isolate`: run in the gist work dir instead of the current directory.
- `--ref <sha>`: run a specific gist revision (overrides a pin for this run).
- `--view`: print files without executing or changing trust.
- `--dry-run`: print the resolved command without executing or changing trust.
- `--add`: remember the gist after a successful run.
- `--as <name>`: remember it with a custom name (implies `--add`).
- `--timeout <duration>`: cancel execution after a duration like `30s` or `2m`.

## Authentication

gixt uses direct HTTP and does not require `gh`.

- `gixt auth login`: store a personal access token (scope: `gist`).
- Token resolution order: `GITHUB_TOKEN` env, stored token, `gh auth token` (only if `gh` happens to be installed).
- No token at all: public gists still work, with a lower rate limit.
- Mutations (`gist set-description`, `gist fork`, `trust mine`) require a token.

## Command tree

```text
gixt
  run <target> [-- <args>]
  add
    <id|url|owner/gist> [--as <name>]
    owner <login>
  remove
    <target>
    owner <login>
  list
    (bare)             # pretty table of known gists
    refresh
    clear
  trust
    mine
    list
    remove <target>
    clear
  pin
    <target> [<sha>]
    list
    remove <target>
    clear
  gist
    show <target>
    set-description <target> <text>
    clone <target> [--dir <path>]
    fork <target> [--public] [--description <text>]
  cache
    list
    prune
    clear
  auth
    login [--token <value>]
    status
    logout
  self
    version
    update-check
```

## Common errors

- `cannot determine how to run <file> (unknown extension)` -> add a shebang to the file.
- `name "x" matches multiple known gists` -> disambiguate via `owner/name`, `name.ext`, or `--as`.
- `could not resolve "x" as a gist id, URL, owner/gist, or known name` -> run `gixt add <target> --as <name>` to remember it.
- `no cached copy of <id>` -> the gist was never run online; drop `--offline`.
- `this action requires authentication` -> run `gixt auth login`.
