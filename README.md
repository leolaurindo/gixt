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
- Pin a gist to a fixed revision with `gixt pin`; metadata updates and cache pruning preserve it.
- Trust-on-first-use per commit: changed revisions prompt again. `gixt trust mine` explicitly snapshots your current gists.
- Inspect with `--view` and `--dry-run` without changing trust approvals.
- Content cache on by default (ETag/304, retaining latest and pinned revisions); `--no-cache` opts out.
- No `gh` dependency; public gists work without any token.
- Command-line friendly: gist output goes to stdout, gixt chatter to stderr, exit codes and signals are propagated.

## Quick start

### Option 1: Install without Go

Linux and macOS:

```sh
curl -fsSL https://get.leolaurindo.com/gixt.sh | sh
```

Windows PowerShell:

```powershell
irm https://get.leolaurindo.com/gixt.ps1 | iex
```

The installer downloads the matching GitHub release, verifies its SHA-256 checksum, and installs it for the current user. Review the [shell](docs/gixt.sh) or [PowerShell](docs/gixt.ps1) script before running it if preferred.

### Option 2: Download a prebuilt binary

- Go to the [releases page](https://github.com/leolaurindo/gixt/releases) and download the appropriate binary for your OS.

### Option 3: Install via `go install`

Requires Go 1.21 or newer:

```sh
go install github.com/leolaurindo/gixt/cmd/gixt@latest
```

### Option 4: Build from source

```sh
go build -o bin/gixt ./cmd/gixt
```

For manual downloads and source builds, place `gixt` (macOS/Linux) or `gixt.exe` (Windows) somewhere on your `PATH`.

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
- Running the currently approved commit again does not re-prompt.
- If the gist has a new commit, you are prompted again — even for your own gists. That way a changed gist can't run silently.
- `gixt trust mine` explicitly snapshots and approves the exact current commit of every gist you own. Later changes still prompt.
- Inspect approvals with `gixt trust list`; revoke them with `gixt trust remove <target>` or `gixt trust clear`.
- `-y` skips the prompt for one run; non-interactive runs refuse to prompt and tell you to pass `-y`.

## Check the docs

- [CLI usage, command tree, and resolution details](docs/cli-usage.md)
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
