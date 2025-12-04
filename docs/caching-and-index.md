# Caching and index

## Directories and files
g
- Config dir (stores aliases, index, settings):
  - Windows: `%APPDATA%\gix` (e.g. `C:\Users\<you>\AppData\Roaming\gix`)
  - Linux: `~/.config/gix`
  - macOS: `~/Library/Application Support/gix`
  - Files: `aliases.json`, `index.json`, `settings.json`.
- Cache dir (stores downloaded gist files + `manifest.json` per gist/sha):
  - Windows: `%LOCALAPPDATA%\gix`
  - Linux: `~/.cache/gix`
  - macOS: `~/Library/Caches/gix`

`--cache-dir` overrides the cache root for `gix` runs, `gix register`, `gix clean-cache`, and `gix clear-index`.

## Cache behavior

- Default cache mode is `never`: runs use a temp dir inside the cache root and remove it after execution. Cached gists are untouched.
- Persist cache by running `gix config-cache --mode cache`.
- Per-run controls:
  - `--no-cache`: force temp/ephemeral even when mode=cache.
  - `--update`: redownload files even if a manifest and files already exist.
  - `--clear-cache`: wipe the cache dir before running.
- Commands:
  - `gix clean-cache [--cache-dir <path>]`: delete the entire cache dir.
  - `gix register <gist-id|url> [--ref <sha>] [--cache-dir <path>] [--update]`: download and cache a gist without running it (does not add to the index).

## Index behavior

- The index lives at `index.json` in the config dir and enables friendly-name lookups.
- Matching rules: filename basenames (case-insensitive, extension stripped); add `--desc-lookup` to also match exact descriptions.
- Commands:
  - `gix index-owner <owner>`: add all gists for an owner (up to 5 pages of 100) to the index.
  - `gix index-mine`: fetch or re-sync all gists for your authenticated user (adds new ones, drops deleted gists).
  - `gix update-index`: refresh existing index entries one-by-one via `gh`, skipping/pruning gists that return 404.
  - `gix clear-index [--cache-dir <path>]`: delete only the index file.

## Listing

`gix list [--cache|-c] [--mine]` shows cached + indexed gists in one table.

- `Source` column: `cache`, `index`, or `cache+index` depending on where the entry came from.
- `Aliases` column is derived from `aliases.json` (if present).
- `--cache` limits to cached entries; `--mine` filters to gists owned by your `gh` user.

## Quick recipes

- Temporary/uvx-style runs (default): do nothing; gix uses a temp workdir and deletes it after the run.
- Persistent cache: `gix config-cache --mode cache` then run normally.
- Force refresh a cached gist: `gix <id> --update` (or rerun `gix register ... --update`).
- Start friendly-name usage: `gix index-owner <your-gh-login>` to fill the index, then `gix update-index` to refresh later.
