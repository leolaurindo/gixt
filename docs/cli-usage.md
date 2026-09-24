# CLI Usage Guide

`gixt` retrieves Gist artifacts with `cat` and executes code explicitly with `run`.

## Command forms

```sh
gixt cat <target> [<target> ...]
gixt print <target> [<target> ...]
gixt run <target> [--entry <filename>] [-- <args>]
gixt <target>                         # shorthand for gixt cat <target>
```

The `print` command is an alias for `cat`. Bare targets retrieve content; execution always requires `gixt run`.

## Targets and entries

A target identifies a Gist. An entry identifies one file in that Gist.

Targets resolve in this order:

1. Gist ID or URL;
2. known alias;
3. `owner/name` from the known store, then live owner lookup;
4. exact full filename among known Gists;
5. filename stem among known Gists.

An alias identifies a Gist, never a file. An exact filename target carries the requested filename into selection. A stem identifies the Gist but does not force a particular variant. Ambiguous targets remain errors and show disambiguators.

## `cat`

```sh
gixt cat review-prompt | agent
gixt cat review.md style.md | agent
gixt cat review.md --entry review.md
```

Each target is resolved independently and processed in command-line order. A Gist target emits all files in lexical order. An exact filename target emits that file. `--entry` is valid only with one target.

`cat` writes selected bytes directly to stdout. It adds no headers, separators, or newlines, and it never executes code or checks trust. Diagnostics and warnings go to stderr, so binary output and shell pipelines remain safe.

Flags:

- `--offline` — use only the cached revision.
- `--no-cache` — use a temporary directory and remove it afterwards.
- `--ref <sha>` — select a specific revision, overriding a pin.
- `--entry <filename>` / `-e` — select one exact file.

## `run`

```sh
gixt run cleanup-script -- --dry-run
gixt run --entry cleanup.sh cleanup-script
gixt run --python .venv/bin/python script.py
```

Entry selection order is:

1. explicit `--entry`;
2. exact filename inferred from the target;
3. `main.*`;
4. `index.*`;
5. a platform shell variant when candidates share a basename;
6. lexical fallback.

After selection, execution resolution is Python override, shebang, then extension mapping. Trust approval applies only to `run`, keyed by Gist ID and revision.

`run --view` remains temporarily available for compatibility but is deprecated. Use `gixt cat` for retrieval.

## Resolution examples

```sh
# Exact filename target: selects that file
gixt run review.sh

# Stem target: resolves the Gist, then uses normal fallback selection
gixt run review

# Alias identifies the Gist; --entry identifies the file
gixt run --entry review.sh review-prompt
```

Use `gixt add <target> --as <name>` to remember an alias. Aliases are globally unique; use `--entry` to select files within the aliased Gist.

## Caching, pins, and trust

- Online retrieval uses the existing ETag/cache behavior.
- `--offline` never contacts GitHub and requires a cached revision.
- A pin is used when `--ref` is absent.
- `cat` does not prompt or modify trust.
- `run` prompts for an unapproved revision unless `--yes` is supplied.

Inspect metadata and files with:

```sh
gixt gist show <target>
```

Remember Gists with:

```sh
gixt add <target> --as <name>
gixt add owner <owner>
```
