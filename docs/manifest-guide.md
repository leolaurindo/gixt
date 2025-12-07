# Manifest guide

How to define and use `gixt.json` (or a custom manifest name) when running gists with `gixt`.

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
- `details` (string, optional): docstring shown by `gixt describe`; defaults to `"No description provided"` when empty/missing.
- `version` (string, optional): surfaced by `gixt describe` when present.
- Default filename is `gixt.json`; override with `--manifest <name>` when running or when generating via `gixt manifest`.

## Workflows

### Local authoring (keeps a file on disk)
- Create: `gixt manifest --create --name gixt.json --run "python app.py" --details "Usage: ..." --version 0.1.0 --env FOO=BAR`
- Edit: `gixt manifest --edit --name gixt.json --run "./script.sh" --details "Updated"` (prompts before overwrite unless `--force`).
- Upload an existing local manifest: `gixt manifest --upload --gist <id|name> --name gixt.json`.

### In-memory authoring + upload (no local file written)
- Create + upload in one go: `gixt manifest --create --upload --gist <id|name> --run "python app.py" --details "Usage" --version 0.1.0`
- Edit + upload without touching disk: `gixt manifest --edit --upload --gist <id|name> --details "Updated" --version 0.2.0`
  - If no local manifest is present, `--edit --upload` fetches the existing manifest from the gist, applies overrides, and uploads the result.

### View a manifest from a gist (no write)
- `gixt manifest --view --gist <id|name> [--name <file>]` prints the manifest JSON from the gist (default `gixt.json`) without caching other files.

### Running with manifests
- Place the manifest at the gist root (or specify `--manifest <name>`).
- Command resolution order prefers manifest over shebang/extension maps.
- Manifests are cached only when gist files are cached; temp runs do not persist manifests.

## Tips

- Use `gixt describe <id|name>` to see manifest `details` and `version` when available.
- Use `gixt manifest --env KEY=VAL` repeatedly to set multiple env entries.
- Use `gixt manifest run "..." details "..." version "..."` positional pairs as a shortcut.***
