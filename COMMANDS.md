# Command Inventory

This file documents the CLI surface implemented on this branch.

Framework: `cobra`.

## Top-Level Behavior

- `gixt cat <target> [<target> ...]` — write selected Gist files to stdout.
- `gixt print <target> [<target> ...]` — alias for `cat`.
- `gixt run <target> [-- <args>]` — explicitly execute one selected entry.
- `gixt <target>` — shorthand for `gixt cat <target>`.
- Exact Cobra commands win over target resolution.
- Command typo suggestions are only produced after typed target-not-found errors.
- `gixt --version` / `gixt -v` — print the build version.

## `gixt cat`

```text
gixt cat <target> [<target> ...] [flags]
gixt print <target> [<target> ...] [flags]
```

Each target is resolved independently and processed in command-line order. A Gist target emits all files in lexical order. An exact filename target emits that file. `--entry` selects one exact file and is valid only with one target.

Flags:

| Flag | Arg | Description |
| --- | --- | --- |
| `--entry`, `-e` | `<filename>` | Select one exact file; requires one target. |
| `--offline` | none | Read the cached revision without contacting GitHub. |
| `--no-cache` | none | Use a temporary directory and remove it afterwards. |
| `--ref` | `<sha>` | Read a specific revision, overriding a pin. |

Output is byte-exact: no headers, separators, filtering, or added newlines. Diagnostics go to stderr. `cat` never executes code or changes trust approvals.

## `gixt run`

```text
gixt run <target> [--entry <filename>] [-- <args>]
```

The target accepts a Gist ID, URL, `owner/name`, known alias, or known filename. The selected entry is resolved explicitly, by an exact filename inferred from the target, or by `main.*`, `index.*`, platform shell variant, and lexical fallback.

Flags:

| Flag | Arg | Description |
| --- | --- | --- |
| `--yes`, `-y` | none | Skip the trust prompt for this run. |
| `--entry`, `-e` | `<filename>` | Execute this exact file. |
| `--offline` | none | Run the cached copy without contacting GitHub. |
| `--no-cache` | none | Use a temporary directory and remove it afterwards. |
| `--python` | `<interpreter>` | Override the selected file's shebang. |
| `--isolate` | none | Run in the Gist work directory instead of the current directory. |
| `--ref` | `<sha>` | Run a specific revision, overriding a pin. |
| `--view` | none | Deprecated; print files and exit without running. |
| `--dry-run` | none | Resolve and print the command without running it. |
| `--add` | none | Remember the Gist after a successful run. |
| `--as` | `<name>` | Remember it under a custom alias. |
| `--timeout` | `<duration>` | Cancel execution after a duration. |

Execution resolution order is Python override, shebang, then extension mapping. Trust is checked only for `run`.

## `gixt add`

- `gixt add <id|url|owner/name> [--as <name>]` — remember one Gist.
- `gixt add owner <login>` — remember all of an owner's Gists.

## `gixt remove`

- `gixt remove <target>` — forget one known Gist.
- `gixt remove owner <login>` — forget all known Gists for an owner.

## `gixt list`

- `gixt list` — show locally remembered Gists.
- `gixt list refresh` — refresh remembered metadata.
- `gixt list clear` — remove all remembered entries.

## Advanced commands

- `gixt trust mine|list|remove|clear`
- `gixt pin <target> [<sha>]`, `gixt pin list|remove|clear`
- `gixt gist show|set-description|clone|fork`
- `gixt cache list|prune|clear`
- `gixt auth login|status|logout`
- `gixt self version|update-check`

## State files

- `known.json` — remembered Gists, aliases, filenames, and pins.
- `trust.json` — approved Gist ID/revision pairs.
- `cache/<gist-id>/<sha>` — materialized files and metadata.
