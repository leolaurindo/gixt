# Trust Model

gixt executes code from gists. Trust is **trust-on-first-use (TOFU) keyed by commit**: the currently approved revision does not re-prompt, while a changed revision does. There is no "trust my own gists always" special case.

Approvals are stored in `trust.json` under your gixt config directory.

## Rules

1. If the exact commit (`sha`) you're running was approved before, gixt runs it without prompting.
2. Otherwise gixt prompts (owner, description, commit, files). Answering `y` records that commit as trusted.
3. `-y` / `--yes` skips the prompt for a single run without persisting trust.
4. If the gist later has a new commit, you are prompted again — even for gists you own. A changed gist can never run silently.

## Approving your own gists

Running your own gists still prompts on first use or after a change. To take an explicit snapshot of all gists you currently own:

```sh
gixt auth login      # required once
gixt trust mine      # fetch and approve every exact current commit
```

The snapshot includes public and secret gists, and the trust store is saved only after every revision is fetched. A gist changed after the snapshot has a different commit and prompts again. Taking another snapshot replaces the previously approved commit for each gist.

Managing approvals:

- `gixt trust list` — show approved gists and commits (the "why am I being prompted again?" tool).
- `gixt trust remove <target>` — revoke one approval.
- `gixt trust clear` — revoke everything.

## Non-interactive runs

When stdin is not a terminal (pipes, scripts, CI), gixt refuses to prompt and exits with:

```
error: refusing to prompt on non-interactive input; pass -y to run untrusted code
```

## Inspecting before you run

- `gixt run --view <target>` prints the gist files without executing or changing trust.
- `gixt run --dry-run <target>` shows the command without executing or changing trust.
- `gixt gist show <target>` shows metadata and the file list.

## Clearing trust

- `gixt trust clear` revokes every approval.
- `gixt trust remove <target>` revokes one.
- Deleting `trust.json` from the config directory also resets everything.
