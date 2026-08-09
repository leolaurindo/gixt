<div align="center">

# ✨gixt

### Run GitHub Gists as real CLI commands

</div>

Turn GitHub gists into ephemeral command-line tools, invoking them by friendly names or aliases. `gixt` makes it easy and safe to run code snippets from GitHub Gists.

```sh
gixt run <gist-name> [-- <args>]
```

## Features and highlights

- Run any gist by name, ID, URL, or `owner/gist`. No setup needed for one-offs.
- Remember gists you use with `gixt add` (or `gixt run --add`) so they get a friendly name.
- Pin a gist to a fixed revision with `gixt pin`; `gixt run` honors it silently.
- Trust-on-first-use per commit: prompted once per revision, re-prompted when it changes. Approve all of your gists at once with `gixt trust mine`.
- Inspect what will run with `--view` and `--dry-run`.
- Content cache on by default (ETag/304, pruned to latest per gist); `--no-cache` opts out.
- No `gh` dependency; public gists work without any token.
- Command-line friendly: gist output goes to stdout, gixt chatter to stderr, exit codes and signals are propagated.

## Quick start

### Prerequisites

- Go 1.21+ to build from source.

### Option 1: Download prebuilt binary

- Go to the [releases page](https://github.com/leolaurindo/gixt/releases) and download the appropriate binary for your OS.

### Option 2: Build

```sh
go build -o bin/gixt ./cmd/gixt
```

For both options, place `gixt` (macOS/Linux) or `gixt.exe` (Windows) somewhere on your `PATH` (e.g., `~/.local/bin`, `/usr/local/bin`, or `%USERPROFILE%\bin`).

### Option 3: Install via `go install`

```sh
go install github.com/leolaurindo/gixt/cmd/gixt@latest
```

### First runs

```sh
# run any gist by ID or URL (no setup)
gixt 1234567890abcdef
gixt run https://gist.github.com/you/1234567890abcdef

# remember gists you use so they get a friendly name
gixt run 1234567890abcdef --add            # remembered as its file basename
gixt add hex23-git/ssh-helper.sh --as ssh  # remember with a custom name
gixt add owner <username>                  # remember all of a user's gists

# see your known gists
gixt list

# run by friendly name
gixt ssh

# optional: log in for higher rate limits / private gists / mutations
gixt auth login
```

**Other examples:**

```sh
# one-off, never remembered
gixt leolaurindo/hello-world -y

# run offline from the cache
gixt run --offline ssh

# run a python gist with your project's venv
gixt run --python .venv/bin/python app.py

# inspect before running
gixt run --view leolaurindo/hello-world
gixt run --dry-run ssh

# pin a gist to its current revision so runs are stable
gixt pin ssh
gixt pin list

# approve all of your gists at their current commits
gixt trust mine
gixt trust list
```

## Trust model

gixt uses **trust-on-first-use keyed by commit**:

- The first time you run a gist, you are prompted (owner, description, commit, files). On `y`, that exact commit is recorded as trusted.
- Running the same commit again never re-prompts.
- If the gist has a new commit, you are prompted again — even for your own gists. That way a changed gist can't run silently.
- Approve all of your gists at their current commits with `gixt trust mine`; inspect with `gixt trust list`, revoke with `gixt trust remove <target>`.
- `-y` skips the prompt for one run; non-interactive runs refuse to prompt and tell you to pass `-y`.

## Command tree

```text
gixt run <target> [-- <args>]      # primary
gixt <target> [-- <args>]          # root shortcut
gixt add <id|url|owner/gist> [--as <name>]
gixt add owner <login>
gixt remove <target> | owner <login>
gixt list | refresh | clear
gixt trust mine | list | remove <target> | clear
gixt pin <target> [<sha>] | list | remove <target> | clear
gixt gist show <target> | set-description <target> <text> | clone <target> | fork <target>
gixt cache list | prune | clear
gixt auth login | status | logout
gixt self version | update-check
```

## Check the docs

- [CLI usage and resolution details](docs/cli-usage.md)
- [Caching and index locations/modes](docs/caching-and-index.md)
- [Trust model and safety](docs/trust-and-security.md)

## Uninstall

- Delete the installed binary.
- Remove config/cache dirs to clear known gists, trust approvals, auth, and cached contents:
  - Windows: config `%APPDATA%\gixt`, cache `%LOCALAPPDATA%\gixt`
  - macOS: config `~/Library/Application Support/gixt`, cache `~/Library/Caches/gixt`
  - Linux: config `~/.config/gixt`, cache `~/.cache/gixt`

## Contributing

Contributions are welcome! I will eventually write contribution guidelines, but for now, feel free to open issues or pull requests.

## Related

Freely inspired by the user experience `uvx` and `npx` provide.
