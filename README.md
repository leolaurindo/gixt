<div align="center">

# ✨ gixt

### Run GitHub Gists as real CLI commands

</div>

Turn GitHub gists into ephemeral command-line tools, invoking them by friendly names or aliases, as well as by ID or URL. You can also fetch from other users. With caching, indexing, and a trust model, `gixt` makes it easy and safe to run code snippets from GitHub Gists.

```sh
gixt <gist-name> [GIST-ARGS...]
```


## Features and highlights

- Index gists so you can type `gixt hello-world` instead of pasting long IDs.
- Manage aliases (`gixt alias add/list/remove`) for frequently used gists.
- Choose between ephemeral runs or a persistent cache.
- Control where code executes: isolated work directory or your current directory.
- Configure a trust policy and prompts before executing untrusted code.
- Inspect what will run with `--view` and `--dry-run`.
- Implicit resolver for commands: [manifest](docs/manifest-guide.md), shebang, or extension map.
- `gixt manifest` scaffolds/edits/uploads/views `gixt.json` (with details/version/docstring) and keeps cache/index in sync.
- `gixt clone` and `gixt fork` help bring gists locally or copy them to your own account.
- Update your own gist descriptions with `gixt set-description --description "new description" --gist <id|name|owner/name>`.
- Anything after -- is passed verbatim to the gist (needed when gist args start with -/--).
- Relative paths are rebased to your original shell CWD so they still point to the same files if gixt runs in an isolated workdir.


## Quick start

### Prerequisites

- Go 1.21+
- GitHub CLI [`gh`](https://github.com/cli/cli#installation) installed and authenticated (`gh auth status` passes).


### Option 1: Download prebuilt binary

- Go to the [releases page](https://github.com/leolaurindo/gixt-cli/releases) and download the appropriate binary for your OS.


### Option 2: Build

```sh
go build -o bin/gixt ./cmd/gixt
```

For both options, place `gixt` (macOS/Linux) or `gixt.exe` (Windows) somewhere on your `PATH` (e.g., `~/.local/bin`, `/usr/local/bin`, or `%USERPROFILE%\bin`, or any other directory on your PATH).

### Option 3: Install via `go install`

If you have Go in your environment, you can install `gixt` as a go tool with:

```sh
go install github.com/leolaurindo/gixt/cmd/gixt@latest
```

This installs to your Go `GOBIN`/`GOPATH/bin`; ensure that directory is on your `PATH`.

### Updating

- If installed with `go install`, update by re-running `go install github.com/leolaurindo/gixt/cmd/gixt@latest`.
- If using a prebuilt binary, download the latest release and replace the existing `gixt` on your `PATH`.
- ✨ To check whether a newer release exists (and get copy/paste commands to download/replace), run `gixt check-updates`.


### First runs

**Setting your gists to run easily**

```sh
# index your own gists so you can run them by name
gixt index-mine

# see what gists are indexed by gixt
gixt list

# trust your own gists
gixt config-trust --mode mine

# run your gists by file basename
gixt hello-world
```

**Other examples:**


```sh
# run by gist ID or URL
gixt 1234567890abcdef
gixt https://gist.github.com/you/1234567890abcdef

# add an alias and use it
gixt alias add hello 1234567890abcdef
gixt hello

 # index another user's gists
gixt index-owner <username>

# cache without running
gixt register <gist-id>


# search for gists! this can be dangerous, but trust
# configs help manage safety
gixt owner/gist --user-lookup
```

### Run in your current directory

By default gixt executes gists in an isolated temp/cache directory for safety. To run a gist in the directory you invoked gixt from, add `--here` (alias `--cwd`):

```sh
gixt --here <gist> [ARGS...]
```

If you prefer this behavior by default, set `gixt config-exec --mode cwd` and use `--isolate` on runs where you want the safer temp directory.


## Using names instead of IDs

To run gists by friendly names, `gixt` uses an index stored in your config directory. Index is just a local mapping of names to gist IDs.

Populate it with `gixt index-mine` (syncs all your gists, adding new ones and removing deleted) or `gixt index-owner <owner>` (another user). Then run gists by name:

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
- [Manifest guide](docs/manifest-guide.md)


## Uninstall

- Delete the installed binary (e.g., `bin/gixt` or `bin/gixt.exe`, or wherever you placed it on PATH).
- Remove config/cache dirs to clear aliases, index, settings, and cached gists:
  - Windows: config `%APPDATA%\gixt`, cache `%LOCALAPPDATA%\gixt`
  - macOS: config `~/Library/Application Support/gixt`, cache `~/Library/Caches/gixt`
  - Linux: config `~/.config/gixt`, cache `~/.cache/gixt`


## Contributing

Contributions are welcome! I will eventually write contribution guidelines, but for now, feel free to open issues or pull requests.


## Motivation

This is a small project for educational purposes and personal use. I wanted a simple way to run gists as commands without installing them globally or copying code around. Although possible, it involves some boilerplate. `gixt` is much more ergonomic. 

Freely inspired by the user experience `uvx` provides. AI tools were used for boilerplate, documentation, testing, syntax (I am learning golang) and some refactoring, but the design and implementation are mainly my own.
