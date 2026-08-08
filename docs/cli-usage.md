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
2. gixt contacts GitHub (unless `--offline`) with the cached ETag: if the gist is unchanged the server replies `304` and the cached copy is reused — no files are downloaded.
3. Files are materialized into a content-addressed cache dir `cache/<id>/<sha>` with path sanitization (no `..`, no absolute or drive-prefixed paths).
4. Trust is checked (see below). On first use of a revision you are prompted; `-y` skips the prompt.
5. The command is resolved: `--python <interpreter>` override, else the shebang on the chosen file, else the extension map (`.py` -> `python`, `.sh` -> `sh`, `.js` -> `node`, `.ts` -> `npx ts-node`, `.go` -> `go run`, `.rb` -> `ruby`, `.pl` -> `perl`, `.php` -> `php`, `.ps1` -> `powershell`, `.bat/.cmd` -> `cmd /C` on Windows).
6. The command runs in your current directory (or the gist work dir with `--isolate`). Child stdin/stdout/stderr are wired straight through; the child's exit code is propagated. `--timeout` cancels long runs.
7. After a successful run, older cached revisions of that gist are pruned (only the latest is kept). With `--add` (or `--as <name>`, which implies `--add`), the gist is also remembered in `known.json`.

Inspection shortcuts:

- `--view` prints the gist files and exits.
- `--dry-run` resolves everything, prints the command, and exits before execution.

## Run flags

- `-y`, `--yes`: skip the trust prompt for this run.
- `--offline`: run the cached copy without contacting GitHub (error if not cached).
- `--python <interpreter>`: override the shebang (e.g. `.venv/bin/python`).
- `--isolate`: run in the gist work dir instead of the current directory.
- `--ref <sha>`: run a specific gist revision.
- `--view`: print files and exit.
- `--dry-run`: resolve and print the command, do not execute.
- `--add`: remember the gist after a successful run.
- `--as <name>`: remember it with a custom name (implies `--add`).
- `--timeout <duration>`: cancel execution after a duration like `30s` or `2m`.

## Authentication

gixt uses direct HTTP and does not require `gh`.

- `gixt auth login`: store a personal access token (scope: `gist`).
- Token resolution order: `GITHUB_TOKEN` env, stored token, `gh auth token` (only if `gh` happens to be installed).
- No token at all: public gists still work, with a lower rate limit.
- Mutations (`gist set-description`, `gist fork`, `index mine`) require a token.

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
  gist
    show <target>
    set-description <target> <text>
    clone <target> [--dir <path>]
    fork <target> [--public] [--description <text>]
  cache
    list
    prune
    clear
  config
    get [key]
    set <key> <value>
    unset <key>
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
