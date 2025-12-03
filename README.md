<div align="center">

# ✨ gix

### Run GitHub Gists as real CLI commands

</div>

Turn GitHub gists into ephemeral command-line tools, invoking them by friendly names or aliases, as well as by ID or URL. You can also fetch from other users. With caching, indexing, and a trust model, `gix` makes it easy and safe to run code snippets from GitHub Gists.

```sh
gix <gist-name> [gist-args...]
```


## Features and highlights

- Index gists so you can type `gix hello-world` instead of pasting long IDs.
- Manage aliases (`gix alias add/list/remove`) for frequently used gists.
- Choose between ephemeral runs or a persistent cache.
- Control where code executes: isolated work directory or your current directory.
- Configure a trust policy and prompts before executing untrusted code.
- Inspect what will run with `--view` and `--dry-run`.
- Implicit resolver for commands: [manifest](docs/manifest-example.md), shebang, or extension map.
- Anything after -- is passed verbatim to the gist (needed when gist args start with -/--).
- Relative paths are rebased to your original shell CWD so they still point to the same files if gix runs in an isolated workdir.


## Quick start

### Prerequisites

- Go 1.21+
- GitHub CLI [`gh`](https://github.com/cli/cli#installation) installed and authenticated (`gh auth status` passes).


### Option 1: Download prebuilt binary

- Go to the [releases page](https://github.com/leolaurindo/gix-cli/releases) and download the appropriate binary for your OS.


### Option 2: Build

```sh
go build -o bin/gix ./cmd/gix
```

For both options, place `gix` (macOS/Linux) or `gix.exe` (Windows) somewhere on your `PATH` (e.g., `~/.local/bin`, `/usr/local/bin`, or `%USERPROFILE%\bin`, or any other directory on your PATH).


### First runs

**Setting your gists to run easily**

```sh
# index your own gists so you can run them by name
gix index-mine

# see what gists are indexed by gix
gix list

# trust your own gists
gix config-trust --mode mine

# run your gists by file basename
gix hello-world
```

**Other examples:**


```sh
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
```


## Using names instead of IDs

To run gists by friendly names, `gix` uses an index stored in your config directory. Index is just a local mapping of names to gist IDs.

Populate it with `gix index-mine` (your gists) or `gix index-owner <owner>` (another user). Then run gists by name:

- Use the file basename as the identifier (`hello-world` for `hello-world.py`).
- Use `owner/name` to disambiguate when multiple owners have the same name.
- Enable description matching with `--desc-lookup` if you prefer using gist descriptions.

For one-off runs without indexing, `--user-lookup/-u` resolves `owner/name` live via the GitHub API.

Cache is turned off by default, so runs are ephemeral and any downloaded files are removed after execution. Use `config-cache --mode cache` to enable persistent caching or check [caching docs](docs/caching-and-index.md) for more options.


## Warning

Running code from untrusted sources can be dangerous. Use the [trust model](docs/trust-and-security.md) to manage which gists you trust to run without prompts. When in doubt, inspect the code first with `--view` or `--dry-run`.


## Check the docs

- [CLI usage and resolution details](docs/cli-usage.md)
- [Caching and index locations/modes](docs/caching-and-index.md)
- [Trust model and safety options](docs/trust-and-security.md)
- [Manifest example](docs/manifest-example.md)


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
