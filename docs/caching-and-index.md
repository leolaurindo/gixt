# Caching and index

## Directories and files

- Config dir (stores aliases, index, settings):
  - Windows: `%APPDATA%\gixt` (e.g. `C:\Users\<you>\AppData\Roaming\gixt`)
  - Linux: `~/.config/gixt`
  - macOS: `~/Library/Application Support/gixt`
  - Files: `aliases.json`, `index.json`, `settings.json`.
- Cache dir (stores downloaded gist files + `manifest.json` per gist/sha):
  - Windows: `%LOCALAPPDATA%\gixt`
  - Linux: `~/.cache/gixt`
  - macOS: `~/Library/Caches/gixt`

`--cache-dir` overrides the cache root for `gixt` runs, `gixt register`, `gixt clean-cache`, and `gixt clear-index`.

## Cache behavior

- Default cache mode is `never`: runs use a temp dir inside the cache root and remove it after execution. Cached gists are untouched.
- Persist cache by running `gixt config-cache --mode cache`.
- Per-run controls:
  - `--no-cache`: force temp/ephemeral even when mode=cache.
  - `--update`: redownload files even if a manifest and files already exist.
  - `--clear-cache`: wipe the cache dir before running.
- Commands:
  - `gixt clean-cache [--cache-dir <path>]`: delete the entire cache dir.
  - `gixt register <gist-id|url> [--ref <sha>] [--cache-dir <path>] [--update]`: download and cache a gist without running it (does not add to the index).

## Index behavior

- The index lives at `index.json` in the config dir and enables friendly-name lookups.
- Matching rules: filename basenames (case-insensitive, extension stripped); add `--desc-lookup` to also match exact descriptions.
- Commands:
  - `gixt index-owner <owner>`: add all gists for an owner (up to 5 pages of 100) to the index.
  - `gixt index-mine`: fetch or re-sync all gists for your authenticated user (adds new ones, drops deleted gists).
  - `gixt update-index`: refresh existing index entries one-by-one via `gh`, skipping/pruning gists that return 404.
  - `gixt clear-index [--cache-dir <path>]`: delete only the index file.

## Listing

`gixt list [--cache|-c] [--mine]` shows cached + indexed gists in one table.

- `Source` column: `cache`, `index`, or `cache+index` depending on where the entry came from.
- `Aliases` column is derived from `aliases.json` (if present).
- `--cache` limits to cached entries; `--mine` filters to gists owned by your `gh` user.

## Quick recipes

- Temporary/uvx-style runs (default): do nothing; gixt uses a temp workdir and deletes it after the run.
- Persistent cache: `gixt config-cache --mode cache` then run normally.
- Force refresh a cached gist: `gixt <id> --update` (or rerun `gixt register ... --update`).
- Start friendly-name usage: `gixt index-owner <your-gh-login>` to fill the index, then `gixt update-index` to refresh later.
