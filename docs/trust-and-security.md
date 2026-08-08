# Trust Model

gixt executes code from gists. Trust is managed with **trust-on-first-use (TOFU) keyed by commit**: an approved gist revision never re-prompts; a new revision does.

Trust state is stored in `trust.json` under your gixt config directory.

## Rules

1. If the exact commit (`sha`) of the gist you're running was approved before, gixt runs it without prompting.
2. Gists owned by your authenticated GitHub user are trusted automatically.
3. Otherwise gixt prompts (owner, description, commit, files). Answering `y` records that commit as trusted.
4. `-y` / `--yes` skips the prompt for a single run without persisting trust.
5. If the gist later has a new commit, you are prompted again — the old approval does not carry over.

## Non-interactive runs

When stdin is not a terminal (pipes, scripts, CI), gixt refuses to prompt and exits with:

```
error: refusing to prompt on non-interactive input; pass -y to run untrusted code
```

This keeps piped usage safe and predictable: `gixt <target> | sh` never stalls waiting for input.

## Inspecting before you run

- `gixt run --view <target>` prints the gist files without executing anything.
- `gixt run --dry-run <target>` shows the exact command gixt would run.
- `gixt gist show <target>` shows metadata and the file list.

## Clearing trust

There is no per-command trust editor; delete `trust.json` from the config directory to reset all approvals (or remove the entry for a single gist).
