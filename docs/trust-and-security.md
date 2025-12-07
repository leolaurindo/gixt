# Trust Model

This CLI executes code from gists. Trust decisions are stored in `settings.json` under your gixt config directory (see `docs/caching-and-index.md`).

Check current trust config:
```sh
gixt config-trust --show
```

## Modes and stored trust

- Modes (`gixt config-trust --mode <never|mine|all>`):
  - `never` (default): always prompt unless another rule applies.
  - `mine`: trust gists owned by your current `gh` user; prompt for others.
  - `all`: trust everything without prompting.
- Persistent entries:
  - Trusted owners: `gixt config-trust --owner <username>` (repeatable).
    - Remove with `gixt config-trust --remove-owner <username>`.
  - Trusted gists: `--trust-always` on a run stores that gist ID; you can also manage them with `gixt config-trust --remove-gist <id>` or `--clear-gists`.
- Global trust flag: `--trust-all` on a run immediately sets mode=all and saves it before continuing. **WARNING**: this can be dangerous; use with caution.
- Non persistent skip: `--yes` or `-y` skips the prompt for that run only.


## Checking a gist content and gixt command before running

Use `--view` at the prompt to see all gist files before confirming execution.

The `--dry-run` flag shows what would run without executing it.


## Trust check order during a run

1. `--yes/-y` or `--trust-always` flag.
2. Mode `all` (including when set by `--trust-all`).
3. Gist ID stored in trusted gists (e.g., from previous `--trust-always`).
4. Owner stored in trusted owners.
5. Mode `mine` **and** owner matches your `gh` user.
6. Otherwise, gixt prompts before execution.

At the prompt, `v`/`view` shows all gist files; any non-yes answer aborts the run. If you ran with `--trust-always`, the gist ID is added to trusted gists after the run.

## Managing trust entries

- Show current config: `gixt config-trust --show`
- Add trusted owners: `gixt config-trust --owner <login>` (repeatable)
- Remove entries: `gixt config-trust --remove-owner <login>` and `--remove-gist <id>`
- Clear subsets: `gixt config-trust --clear-owners` or `--clear-gists`
- Reset everything: `gixt config-trust --reset` (sets mode=never and clears stored owners/gists)

Trust settings are unaffected by cache/index cleaning.
