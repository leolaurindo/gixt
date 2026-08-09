# Caching and known gists

## Directories and files

- Config dir (stores known gists, trust, auth):
  - Windows: `%APPDATA%\gixt`
  - Linux: `~/.config/gixt`
  - macOS: `~/Library/Application Support/gixt`
  - Files: `known.json`, `trust.json`, `auth.json`.
- Cache dir (stores downloaded gist files + a `meta.json` per gist/revision):
  - Windows: `%LOCALAPPDATA%\gixt`
  - Linux: `~/.cache/gixt`
  - macOS: `~/Library/Caches/gixt`

## Cache behavior

- Caching is **on by default**. Each run stores the gist's files under `cache/<gist-id>/<sha>`.
- On every run gixt performs a conditional request: if the gist is unchanged, the server replies `304` and the cached copy is reused with zero file downloads. The ETag is stored in the cache `meta.json`.
- After a successful run, older revisions of that gist are pruned automatically, so the cache stays at roughly one directory per gist.
- `--no-cache` (or `GIXT_NO_CACHE`) downloads to a temp dir and deletes it after the run.
- `--offline` skips the network entirely and runs the cached copy (errors if it isn't cached).
- `--ref <sha>` runs a specific revision (downloaded on demand); a pinned gist runs its pin by default.
- Commands:
  - `gixt cache list`: show cached gists and their latest revision.
  - `gixt cache prune`: manually remove old revisions, keeping the latest per gist.
  - `gixt cache clear`: delete all cached contents.

## Known gists (naming)

`known.json` stores the gists you've chosen to remember, so you can run them by name. An entry holds the gist id, owner, description, filenames, and an optional alias and pin. Name lookup matches the alias or the file basename/full name.

Commands:

- `gixt add <id|url|owner/gist> [--as <name>]` — remember a single gist (custom name with `--as`).
- `gixt add owner <login>` — remember all of an owner's gists (replaces that owner's entries).
- `gixt remove <target>` — forget a gist.
- `gixt remove owner <login>` — forget all of an owner's gists.
- `gixt list` — pretty table of known gists.
- `gixt list refresh` — re-fetch metadata for each entry, dropping deleted gists.
- `gixt list clear` — forget everything.
- `gixt pin <target> [<sha>]` — pin a gist to a revision; `gixt run` honors it silently. Pins live on the known entry, so cache pruning never drops them.
- `gixt pin list` / `gixt pin remove <target>` / `gixt pin clear` — manage pins.

Running a gist does **not** auto-add it to `known.json`; use `gixt run <target> --add` (or `--as <name>`) to remember it, or `gixt add` explicitly.

## Quick recipes

- Run offline: `gixt run --offline <name>`.
- Force the latest revision: just run normally; a changed gist re-downloads and re-prompts.
- Keep a stable revision: `gixt pin <name>`, then run by name; re-pin to move.
- No disk writes: `gixt run --no-cache <name>`.
- Use your project's venv for a python gist: `gixt run --python .venv/bin/python app.py`.
- Remember a one-off gist: `gixt run 1234abc --add`, then `gixt 1234abc` by name next time.
- Custom name for a gist you use often: `gixt add owner/script.py --as myscript`.
