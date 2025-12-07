# Manifest guide

How to define and use `gix.json` (or a custom manifest name) when running gists with `gix`.

## Schema

```json
{
  "run": "python app.py --verbose",
  "details": "Describe what the gist does and its arguments",
  "version": "1.0.0",
  "env": {
    "API_BASE": "https://api.example.com",
    "DEBUG": "1"
  }
}
```

- `run` (string, required): executed via the shell (`sh -c` on Unix, `cmd /C` on Windows).
- `env` (object, optional): key/value pairs injected into the execution environment.
- `details` (string, optional): docstring shown by `gix describe`; defaults to `"No description provided"` when empty/missing.
- `version` (string, optional): surfaced by `gix describe` when present.
- Default filename is `gix.json`; override with `--manifest <name>` when running or when generating via `gix manifest`.

## Workflows

### Local authoring (keeps a file on disk)
- Create: `gix manifest --create --name gix.json --run "python app.py" --details "Usage: ..." --version 0.1.0 --env FOO=BAR`
- Edit: `gix manifest --edit --name gix.json --run "./script.sh" --details "Updated"` (prompts before overwrite unless `--force`).
- Upload an existing local manifest: `gix manifest --upload --gist <id|name> --name gix.json`.

### In-memory authoring + upload (no local file written)
- Create + upload in one go: `gix manifest --create --upload --gist <id|name> --run "python app.py" --details "Usage" --version 0.1.0`
- Edit + upload without touching disk: `gix manifest --edit --upload --gist <id|name> --details "Updated" --version 0.2.0`
  - If no local manifest is present, `--edit --upload` fetches the existing manifest from the gist, applies overrides, and uploads the result.

### View a manifest from a gist (no write)
- `gix manifest --view --gist <id|name> [--name <file>]` prints the manifest JSON from the gist (default `gix.json`) without caching other files.

### Running with manifests
- Place the manifest at the gist root (or specify `--manifest <name>`).
- Command resolution order prefers manifest over shebang/extension maps.
- Manifests are cached only when gist files are cached; temp runs do not persist manifests.

## Tips

- Use `gix describe <id|name>` to see manifest `details` and `version` when available.
- Use `gix manifest --env KEY=VAL` repeatedly to set multiple env entries.
- Use `gix manifest run "..." details "..." version "..."` positional pairs as a shortcut.***
