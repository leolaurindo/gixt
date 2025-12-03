# gix: Run GitHub Gists as Real CLI Commands

Turn GitHub gists into ephemeral command-line tools, invoking them by friendly names or aliases, as well as by ID or URL. You can also fetch from other users. With caching, indexing, and a trust model, `gix` makes it easy and safe to run code snippets from GitHub Gists.

```sh
gix <gist-name> [gist-args...]
```

## Highlights

- Index gists so you can type `gix hello-world` instead of pasting long IDs.
- Manage aliases (`gix alias add/list/remove`) for frequently used gists.
- Choose between ephemeral runs or a persistent cache.
- Control where code executes: isolated work directory or your current directory.
- Configure a trust policy and prompts before executing untrusted code.
- Inspect what will run with `--view` and `--dry-run`.

## Quick start

### Prerequisites

- Go 1.21+
- GitHub CLI `gh` installed and authenticated (`gh auth status` passes).

### Option 1: Download prebuilt binary

- Go to the [releases page](https://github.com/leolaurindo/gix-cli/releases) and download the appropriate binary for your OS.

### Option 2: Build

```sh
go build -o bin/gix ./cmd/gix
```

### First runs

```sh
# index your own gists and run by name
gix index-mine
# trust your own gists
gix config-trust --mode mine
# matches filename basenames
gix hello-world

# run by gist ID or URL
gix 1234567890abcdef
gix https://gist.github.com/you/1234567890abcdef

# add an alias and use it
gix alias add hello 1234567890abcdef
gix hello

 # index another user's gists
gix index-owner <username>
# cache without running
gix register <gist-id>


# search for gists! this can be dangerous, but trust
# configs help manage safety
gix owner/gist --user-lookup

# see what gists are indexed by gix
gix list
```

## Using names instead of IDs

To run gists by friendly names, `gix` uses an index stored in your config directory. Populate it with `gix index-mine` (your gists) or `gix index-owner <owner>` (another user). Then run gists by name:

- Use the file basename as the identifier (`hello-world` for `hello-world.py`).
- Use `owner/name` to disambiguate when multiple owners have the same name.
- Enable description matching with `--desc-lookup` if you prefer using gist descriptions.

For one-off runs without indexing, `--user-lookup/-u` resolves `owner/name` live via the GitHub API.

## Warning

Running code from untrusted sources can be dangerous. Use the [trust model](docs/trust-and-security.md) to manage which gists you trust to run without prompts. When in doubt, inspect the code first with `--view` or `--dry-run`.

## Check the docs

- [CLI usage and resolution details](docs/cli-usage.md)
- [Caching and index locations/modes](docs/caching-and-index.md)
- [Trust model and safety options](docs/trust-and-security.md)


## Uninstall

- Delete the installed binary (e.g., `bin/gix` or `bin/gix.exe`, or wherever you placed it on PATH).
- Remove config/cache dirs to clear aliases, index, settings, and cached gists:
  - Windows: config `%APPDATA%\gix`, cache `%LOCALAPPDATA%\gix`
  - macOS: config `~/Library/Application Support/gix`, cache `~/Library/Caches/gix`
  - Linux: config `~/.config/gix`, cache `~/.cache/gix`

## Contributing

Contributions are welcome! I will eventually write contribution guidelines, but for now, feel free to open issues or pull requests.

## Motivation

This is a small project for educational purposes and personal use. I wanted a simple way to run gists as commands without installing them globally or copying code around. Although possible, it involves some boilerplate. `gix` is much more ergonomic. 

Freely inspired by the user experience `uvx` provides. AI tools were used for boilerplate, documentation, testing, syntax (I am learning golang) and some refactoring, but the design and implementation are mainly my own.
