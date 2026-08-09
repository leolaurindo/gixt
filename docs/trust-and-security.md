# Trust Model

gixt executes code from gists. Trust is **trust-on-first-use (TOFU) keyed by commit**: an approved revision never re-prompts; a changed revision always does. There is no "trust my own gists always" special case.

Approvals are stored in `trust.json` under your gixt config directory.

## Rules

1. If the exact commit (`sha`) you're running was approved before, gixt runs it without prompting.
2. Otherwise gixt prompts (owner, description, commit, files). Answering `y` records that commit as trusted.
3. `-y` / `--yes` skips the prompt for a single run without persisting trust.
4. If the gist later has a new commit, you are prompted again — even for gists you own. A changed gist can never run silently.

## Approving your own gists

Running your own gists still prompts on first use (or after a change). To approve all of your gists at their current commits:

```sh
gixt auth login      # required once
gixt trust mine      # approve the current commit of every gist you own
```

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

- `gixt run --view <target>` prints the gist files without executing anything.
- `gixt run --dry-run <target>` shows the exact command gixt would run.
- `gixt gist show <target>` shows metadata and the file list.

## Clearing trust

- `gixt trust clear` revokes every approval.
- `gixt trust remove <target>` revokes one.
- Deleting `trust.json` from the config directory also resets everything.
