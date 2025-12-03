# CLI Usage Guide

How to run gists, resolve identifiers, control caching/indexing, and understand what gix actually executes.

## Prerequisites

- Built `gix` binary (`bin/gix` or `bin/gix.exe`).
- `gh` installed and authenticated (`gh auth status` succeeds).

## Command form and argument forwarding

```text
gix [run flags] <gist-id|url|alias|name|owner/name> [-- <args to gist>]
```

- Flags before the target configure gix itself (resolution, caching, manifests, execution, trust, timeouts).
- After the literal `--`, every argument is forwarded to the gist unchanged.
  - If the argument does not look like a flag and comes after the target, gix treats it as a gist argument automatically (no need for `--`).
- Relative forwarded args are resolved against your original shell CWD so they still point at the same files when you run in an isolated workdir.

## Identifier resolution

Accepted forms:

- Alias (`gix alias add cool <id>`)
- Gist ID or URL (last path segment is extracted)
- Friendly filename from the index (basename only, extension stripped)
- `owner/name`

Resolution order:

1. Alias map.
2. Looks like a gist ID/URL.
3. Index lookups:
   - `owner/name` -> match owner + filename basename. Add `--desc-lookup` to also match exact descriptions.
   - bare `name` -> match filename basename (or exact description when `--desc-lookup`).
4. Live `owner/name` lookup with `--user-lookup/-u` (uses `gh api /users/<owner>/gists`, 100 per page, `--user-pages/-p` pages, default 2). Matches filename basenames; add `--desc-lookup` for exact descriptions.
5. Ambiguities produce an error with counts; otherwise gix says it could not resolve the identifier and suggests indexing or `-u`.

Descriptions are never used unless `--desc-lookup` is set, and description matching is exact (case-insensitive trim).

## What happens during a run

1. Settings + paths are loaded from your user config/cache directories (or `--cache-dir`).
2. `--trust-all` immediately sets mode=all and saves it.
3. `--clear-cache` wipes the cache dir before continuing.
4. Gist is fetched via `gh api /gists/<id>` (or `/gists/<id>/<ref>` when `--ref` is set); the latest SHA is recorded.
5. Workdir is chosen:
   - Cache mode `never` (default) or `--no-cache` -> temp dir inside the cache root, removed after the run.
   - Cache mode `cache` -> persistent dir per gist+SHA. `--update` redownloads even if files already exist.
6. Execution directory is decided:
   - Stored mode from `gix config-exec`, or a one-time prompt if no mode is stored and you are running an unindexed or live `-u` gist. Prompt is skipped when `--yes` is set (defaults to `isolate`).
   - Per-run overrides: `--isolate` or `--cwd/--here`.
   - Workdir always holds the gist files; exec dir controls where the command runs.
7. Files are materialized with path sanitization (no `..`, no absolute or drive-prefixed paths). Cached manifest+files are reused unless `--update` is set. A manifest is saved unless `--no-cache` is in effect.
8. Inspection shortcuts:
   - `--view/-v` prints all gist files (from cache/workdir) and exits.
   - `--print-cmd` shows the command gix will run.
   - `--dry-run` resolves everything and exits before execution (prints the command too).
9. Trust decision:
   - Skipped when `--yes` or `--trust-always` is set, when mode is `all`, when the gist ID is already trusted, when the owner is trusted, or when mode=`mine` and the owner matches your `gh` user.
   - Otherwise, you are prompted; entering `v` shows files before deciding. `--trust-always` also stores the gist as trusted after the run.
10. Command resolution (in order): manifest (`gix.json` or `--manifest <name>`) with `run` + optional `env`; shebang on the chosen file; extension map (.sh -> sh, .ps1 -> powershell, .bat/.cmd -> cmd /C on Windows, .py -> python, .js -> node, .ts -> npx ts-node, .go -> go run, .rb -> ruby, .pl -> perl, .php -> php). Entrypoint preference: `main.*` then `index.*` then the first file (sorted).
11. Execution: runs the resolved command in the exec dir with any extra env from the manifest. `--timeout` cancels long runs.

## Run flags (high level)

- Resolution: `--ref <sha>`, `--user-lookup/-u`, `--user-pages/-p <n>`, `--desc-lookup`
- Caching: `--no-cache`, `--update`, `--cache-dir <path>`, `--clear-cache`, `--update-index` (refresh existing index entries before running)
- Manifests/inspection: `--manifest <file>`, `--print-cmd`, `--dry-run`, `--view/-v`, `--verbose`
- Execution: `--isolate`, `--cwd/--here`, `--timeout <duration>`
- Trust: `--yes/-y`, `--trust-always`, `--trust-all`

## Subcommands

- `gix alias add <name> <gist-id>` | `list` | `remove <name>`: manage aliases.
- `gix list [--cache|-c] [--mine]`: show cached + indexed gists (columns: ID, Source, Owner, Files, Aliases, Description). `--cache` limits to cached; `--mine` filters to gists owned by your `gh` user.
- `gix update-index` / `gix index-mine`: refresh existing index entries individually. If the index is empty, nothing is fetched; use `index-owner` to add entries first.
- `gix index-owner <owner>`: add all gists for an owner (up to 5 pages of 100) to the index.
- `gix clear-index [--cache-dir <path>]`: delete the index file only.
- `gix clean-cache [--cache-dir <path>]`: delete the cache directory.
- `gix register <gist-id|url> [--ref <sha>] [--cache-dir <path>] [--update]`: download and cache a gist without running it (does not add to the index).
- `gix config-trust [flags]`: manage trust mode, trusted owners, and stored gist trust.
- `gix config-cache --mode cache|never [--show]`: set or display cache mode.
- `gix config-exec --mode isolate|cwd [--show]`: set or display execution directory mode.
- `gix describe <gist-id|url|alias|name|owner/name>`: show description (prefers index/cache, otherwise fetches).

## Common errors

- `cannot determine how to run <file> (unknown extension)` -> add a manifest or shebang.
- `friendly name matches multiple gists` or `owner/name matches multiple gists` -> disambiguate via ID/URL or index-owner.
- `gh <...> failed` -> check `gh auth status` and your network access.
