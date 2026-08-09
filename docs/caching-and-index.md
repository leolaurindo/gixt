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
- After a successful run, old revisions are pruned while retaining both the executed revision and any pinned revision.
- `--no-cache` (or `GIXT_NO_CACHE`) avoids the persistent content cache by using a temporary directory.
- `--offline` skips the network and runs the exact requested or pinned cached revision. It takes precedence over no-cache mode and prints a warning when both are requested.
- `--ref <sha>` runs a specific revision (downloaded on demand); a pinned gist runs its pin by default.
- Commands:
  - `gixt cache list`: show cached gists and their latest revision.
  - `gixt cache prune`: remove old revisions while retaining the latest and pinned revisions.
  - `gixt cache clear`: delete all cached contents, including offline copies of pinned revisions; pin metadata remains.

## Known gists (naming)

`known.json` stores the gists you've chosen to remember, so you can run them by name. An entry holds the gist id, owner, description, filenames, and an optional alias and pin. Name lookup matches the alias or the file basename/full name.

Commands:

- `gixt add <id|url|owner/gist> [--as <name>]` — remember a single gist (custom name with `--as`).
- `gixt add owner <login>` — refresh an owner's gists while preserving pins.
- `gixt remove <target>` — forget a gist.
- `gixt remove owner <login>` — forget all of an owner's gists.
- `gixt list` — pretty table of known gists.
- `gixt list refresh` — re-fetch metadata, dropping deleted unpinned gists and retaining pinned entries.
- `gixt list clear` — forget everything, including pins stored on those entries.
- `gixt pin <target> [<sha>]` — validate and pin a gist revision; `gixt run` uses it by default.
- `gixt pin list` / `gixt pin remove <target>` / `gixt pin clear` — manage pins.

Metadata updates never move a pin. Re-running `gixt pin`, or using `pin remove`/`pin clear`, changes it explicitly. Forgetting a known entry with `remove`, `remove owner`, or `list clear` also deletes its pin.

Running a gist does **not** auto-add it to `known.json`; use `gixt run <target> --add` (or `--as <name>`) to remember it, or `gixt add` explicitly.

## Quick recipes

- Run offline: `gixt run --offline <name>`.
- Force the latest revision: just run normally; a changed gist re-downloads and re-prompts.
- Keep a stable revision: `gixt pin <name>`, then run by name; re-pin to move.
- No persistent content cache: `gixt run --no-cache <name>`.
- Use your project's venv for a python gist: `gixt run --python .venv/bin/python app.py`.
- Remember a one-off gist: `gixt run 1234abc --add`, then `gixt 1234abc` by name next time.
- Custom name for a gist you use often: `gixt add owner/script.py --as myscript`.
