# CLI Usage Guide

How to run gists, resolve identifiers, control caching/indexing, and understand what gixt actually executes.

## Prerequisites

- Built `gixt` binary (`bin/gixt` or `bin/gixt.exe`).
- `gh` installed and authenticated (`gh auth status` succeeds).

## Command form and argument forwarding

```text
gixt [run flags] <gist-id|url|alias|name|owner/name> [-- <args to gist>]
```

- Flags before the target configure gixt itself (resolution, caching, manifests, execution, trust, timeouts).
- After the literal `--`, every argument is forwarded to the gist unchanged.
  - If the argument does not look like a flag and comes after the target, gixt treats it as a gist argument automatically (no need for `--`).
- Relative forwarded args are resolved against your original shell CWD so they still point at the same files when you run in an isolated workdir.
- Show the build version with `gixt --version`.

## Identifier resolution

Accepted forms:

- Alias (`gixt alias add cool <id>`)
- Gist ID or URL (last path segment is extracted)
 - Friendly filename from the index (basename or full filename with extension)
- `owner/name`

Resolution order:

1. Alias map.
2. Looks like a gist ID/URL.
3. Index lookups:
   - `owner/name` -> match owner + filename basename or full filename (extension allowed). Add `--desc-lookup` to also match exact descriptions.
   - bare `name` -> match filename basename or full filename (extension allowed) (or exact description when `--desc-lookup`).
4. Live `owner/name` lookup with `--user-lookup/-u` (uses `gh api /users/<owner>/gists`, 100 per page, `--user-pages/-p` pages, default 2). Matches filename basenames or full filenames; add `--desc-lookup` for exact descriptions.
5. Ambiguities produce an error with counts; otherwise gixt says it could not resolve the identifier and suggests indexing or `-u`.

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
   - Stored mode from `gixt config-exec`, or a one-time prompt if no mode is stored and you are running an unindexed or live `-u` gist. Prompt is skipped when `--yes` is set (defaults to `isolate`).
   - Per-run overrides: `--isolate` or `--cwd/--here`.
   - Workdir always holds the gist files; exec dir controls where the command runs.
7. Files are materialized with path sanitization (no `..`, no absolute or drive-prefixed paths). Cached manifest+files are reused unless `--update` is set. A manifest is saved unless `--no-cache` is in effect.
8. Inspection shortcuts:
   - `--view` prints all gist files (from cache/workdir) and exits.
   - `--print-cmd` shows the command gixt will run.
   - `--dry-run` resolves everything and exits before execution (prints the command too).
9. Trust decision:
   - Skipped when `--yes` or `--trust-always` is set, when mode is `all`, when the gist ID is already trusted, when the owner is trusted, or when mode=`mine` and the owner matches your `gh` user.
   - Otherwise, you are prompted; entering `v` shows files before deciding. `--trust-always` also stores the gist as trusted after the run.
10. Command resolution (in order): manifest (`gixt.json` or `--manifest <name>`) with `run` (string, executed via shell) + optional `env`; shebang on the chosen file; extension map (.sh -> sh, .ps1 -> powershell, .bat/.cmd -> cmd /C on Windows, .py -> python, .js -> node, .ts -> npx ts-node, .go -> go run, .rb -> ruby, .pl -> perl, .php -> php). Entrypoint preference: `main.*` then `index.*` then the first file (sorted).
11. Execution: runs the resolved command in the exec dir with any extra env from the manifest. `--timeout` cancels long runs.

## Execution directory modes

- Default is **isolate**: the gist files live in a temp/cache dir and execution also happens there (safer for untrusted code).
- Per-run override: `--cwd` (alias `--here`) executes in your current shell directory; `--isolate` forces the temp/cache dir even if you changed the default.
- Change the default: `gixt config-exec --mode cwd` to prefer running in your current directory (use `--isolate` when you want safety for a specific run).

## Run flags (high level)

- Resolution: `--ref <sha>`, `--user-lookup/-u`, `--user-pages/-p <n>`, `--desc-lookup`
- Caching: `--no-cache`, `--update`, `--cache-dir <path>`, `--clear-cache`, `--update-index` (refresh existing index entries before running)
- Manifests/inspection: `--manifest <file>`, `--print-cmd`, `--dry-run`, `--view`, `--verbose`
- Safety: `--ignore-manifest` to skip a manifest and fall back to shebang/extension resolution
- Execution: `--isolate`, `--cwd/--here`, `--timeout <duration>`
- Trust: `--yes/-y`, `--trust-always`, `--trust-all`

## Subcommands

- `gixt alias add <name> <gist-id>` | `list` | `remove <name>`: manage aliases.
- `gixt list [--cache|-c] [--mine]`: show cached + indexed gists (columns: ID, Source, Owner, Files, Aliases, Description). `--cache` limits to cached; `--mine` filters to gists owned by your `gh` user.
- `gixt index-mine`: fetch or re-sync all gists for your authenticated user (adds new ones, drops deleted gists).
- `gixt update-index`: refresh existing index entries individually via `gh`, skipping missing gists (404) and pruning them from the index.
- `gixt index-owner <owner>`: add all gists for an owner (up to 5 pages of 100) to the index.
- `gixt clear-index [--cache-dir <path>]`: delete the index file only.
- `gixt clean-cache [--cache-dir <path>]`: delete the cache directory.
- `gixt register <gist-id|url> [--ref <sha>] [--cache-dir <path>] [--update]`: download and cache a gist without running it (does not add to the index).
- `gixt config-trust [flags]`: manage trust mode, trusted owners, and stored gist trust.
- `gixt config-cache --mode cache|never [--show]`: set or display cache mode.
- `gixt config-exec --mode isolate|cwd [--show]`: set or display execution directory mode.
- `gixt describe <gist-id|url|alias|name|owner/name>`: show description (prefers index/cache, otherwise fetches).
- `gixt manifest --create|--edit [--name <file>] [--run ... --env KEY=VAL --details ... --version ...] [--force]`: scaffold or update a manifest locally (defaults to `gixt.json`).
- `gixt manifest --create|--edit --upload --gist <id|name>`: build the manifest in-memory and upload directly to a user-owned gist (no local write). `--edit --upload` will fetch the existing manifest from the gist when there is no local file. Indexed name or owner/name is allowed; cache/index refresh after upload.
- `gixt manifest --upload --gist <id|name>`: upload an existing local manifest file (no create/edit), refreshing cache/index on success.
- `gixt manifest --view --gist <id|name> [--name <file>]`: fetch and print the manifest JSON from a gist without writing locally (defaults to `gixt.json`).
- `gixt clone <id|name> [--dir <path>]`: clone a gist into a local directory (wraps `gh gist clone`).
- `gixt fork <id|name> [--public] [--description <desc>]`: copy a gist into a new user-owned gist (private by default), reusing files and optional description override.
- `gixt set-description --description "<text>" --gist <id|name|owner/name>`: update the description of a user-owned gist without running it.
- `gixt check-updates [--json]`: compare the current binary against the latest GitHub release and print copy/paste download/replace commands for your platform (does not self update, but includes platform-specific instructions for easy copy/paste).

## Manifest example

See `docs/manifest-guide.md` for manifest schema, workflows, and examples.

## Common errors

- `cannot determine how to run <file> (unknown extension)` -> add a manifest or shebang.
- `friendly name matches multiple gists` or `owner/name matches multiple gists` -> disambiguate via ID/URL or index-owner.
- `gh <...> failed` -> check `gh auth status` and your network access.
