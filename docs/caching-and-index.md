# Caching and known gists

## Directories and files

- Config dir (stores known gists, settings, trust, auth):
  - Windows: `%APPDATA%\gixt`
  - Linux: `~/.config/gixt`
  - macOS: `~/Library/Application Support/gixt`
  - Files: `known.json`, `settings.json`, `trust.json`, `auth.json`.
- Cache dir (stores downloaded gist files + a `meta.json` per gist/revision):
  - Windows: `%LOCALAPPDATA%\gixt`
  - Linux: `~/.cache/gixt`
  - macOS: `~/Library/Caches/gixt`

## Cache behavior

- Caching is **on by default**. Each run stores the gist's files under `cache/<gist-id>/<sha>`.
- On every run gixt performs a conditional request: if the gist is unchanged, the server replies `304` and the cached copy is reused with zero file downloads. The ETag is stored in the cache `meta.json`.
- After a successful run, older revisions of that gist are pruned automatically, so the cache stays at roughly one directory per gist.
- `--offline` skips the network entirely and runs the cached copy (errors if it isn't cached).
- `--ref <sha>` runs a specific revision (downloaded on demand).
- Commands:
  - `gixt cache list`: show cached gists and their latest revision.
  - `gixt cache prune`: manually remove old revisions, keeping the latest per gist.
  - `gixt cache clear`: delete all cached contents.

## Known gists (naming)

`known.json` stores the gists you've chosen to remember, so you can run them by name. An entry holds the gist id, owner, description, filenames, and an optional alias. Name lookup matches the alias or the file basename/full name.

Commands:

- `gixt add <id|url|owner/gist> [--as <name>]` — remember a single gist (custom name with `--as`).
- `gixt add owner <login>` — remember all of an owner's gists (replaces that owner's entries).
- `gixt remove <target>` — forget a gist.
- `gixt remove owner <login>` — forget all of an owner's gists.
- `gixt list` — pretty table of known gists.
- `gixt list refresh` — re-fetch metadata for each entry, dropping deleted gists.
- `gixt list clear` — forget everything.

Running a gist does **not** auto-add it to `known.json`; use `gixt run <target> --add` (or `--as <name>`) to remember it, or `gixt add` explicitly.

## Quick recipes

- Run offline: `gixt run --offline <name>`.
- Force the latest revision: just run normally; a changed gist re-downloads and re-prompts.
- Use your project's venv for a python gist: `gixt run --python .venv/bin/python app.py`.
- Remember a one-off gist: `gixt run 1234abc --add`, then `gixt 1234abc` by name next time.
- Custom name for a gist you use often: `gixt add owner/script.py --as myscript`.
